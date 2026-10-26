// Package realtime relaie les avis PostgreSQL (LISTEN/NOTIFY) vers les navigateurs en SSE.
//
// Chaîne : l'orchestrateur écrit un événement → le déclencheur SQL envoie un avis au commit →
// le relais lit l'événement et le diffuse aux navigateurs abonnés à la campagne.
// Les avis de progression, très fréquents, sont fusionnés : un message par campagne et par intervalle.
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// Canaux PostgreSQL écoutés.
const (
	EventsChannel   = "mammomesh_events"
	ProgressChannel = "mammomesh_progress"
)

// Message est ce qu'un navigateur reçoit.
type Message struct {
	ID   int64  // identifiant SSE ; 0 pour un message non journalisé (progression, resync)
	Type string // type d'événement SSE
	Data []byte // JSON compact
}

type client struct {
	campaignID int64
	ch         chan Message
	closed     atomic.Bool
}

// Stats sont exposées dans /metrics.
type Stats struct {
	Clients        atomic.Int64
	Sent           atomic.Int64
	Coalesced      atomic.Int64
	SlowDisconnect atomic.Int64
	NotifyLagMs    atomic.Int64 // dernier délai entre l'écriture de l'événement et sa diffusion
}

// Hub diffuse les messages aux abonnés.
type Hub struct {
	pool     *pgxpool.Pool
	st       *store.Store
	log      *slog.Logger
	interval time.Duration
	Stats    Stats

	mu      sync.Mutex
	clients map[int64]map[*client]struct{}
	pending map[int64][]byte // dernière progression par campagne, en attente de fusion
}

// NewHub crée le relais ; interval est la fenêtre de fusion des progressions.
func NewHub(st *store.Store, log *slog.Logger, interval time.Duration) *Hub {
	return &Hub{pool: st.Pool, st: st, log: log, interval: interval,
		clients: map[int64]map[*client]struct{}{}, pending: map[int64][]byte{}}
}

// Subscribe abonne un navigateur à une campagne.
func (h *Hub) Subscribe(campaignID int64) *client {
	c := &client{campaignID: campaignID, ch: make(chan Message, 256)}
	h.mu.Lock()
	if h.clients[campaignID] == nil {
		h.clients[campaignID] = map[*client]struct{}{}
	}
	h.clients[campaignID][c] = struct{}{}
	h.mu.Unlock()
	h.Stats.Clients.Add(1)
	return c
}

// Unsubscribe retire un abonné.
func (h *Hub) Unsubscribe(c *client) {
	h.mu.Lock()
	if set := h.clients[c.campaignID]; set != nil {
		if _, ok := set[c]; ok {
			delete(set, c)
			h.Stats.Clients.Add(-1)
		}
		if len(set) == 0 {
			delete(h.clients, c.campaignID)
		}
	}
	h.mu.Unlock()
	if c.closed.CompareAndSwap(false, true) {
		close(c.ch)
	}
}

// broadcast envoie sans bloquer ; un abonné dont le tampon est plein est déconnecté et reprendra.
func (h *Hub) broadcast(campaignID int64, m Message) {
	h.mu.Lock()
	var slow []*client
	for c := range h.clients[campaignID] {
		select {
		case c.ch <- m:
			h.Stats.Sent.Add(1)
		default:
			slow = append(slow, c)
		}
	}
	h.mu.Unlock()
	for _, c := range slow {
		h.Stats.SlowDisconnect.Add(1)
		h.Unsubscribe(c)
	}
}

func (h *Hub) broadcastAll(m Message) {
	h.mu.Lock()
	ids := make([]int64, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	h.mu.Unlock()
	for _, id := range ids {
		h.broadcast(id, m)
	}
}

// Run écoute PostgreSQL jusqu'à l'arrêt du contexte, en se reconnectant après une coupure.
func (h *Hub) Run(ctx context.Context) {
	go h.flushLoop(ctx)
	backoff := time.Second
	first := true
	for ctx.Err() == nil {
		err := h.listen(ctx, func() {
			backoff = time.Second
			if !first {
				// Des avis ont pu se perdre pendant la coupure : les navigateurs rechargent l'état.
				h.broadcastAll(Message{Type: "resync", Data: []byte(`{}`)})
			}
			first = false
		})
		if ctx.Err() != nil {
			return
		}
		h.log.Warn("écoute PostgreSQL interrompue ; nouvelle tentative", "err", err, "délai", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (h *Hub) listen(ctx context.Context, onReady func()) error {
	conn, err := h.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	for _, ch := range []string{EventsChannel, ProgressChannel} {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			return err
		}
	}
	onReady()
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			conn.Conn().Close(context.Background()) //nolint:errcheck // connexion jetée après une erreur
			return err
		}
		switch n.Channel {
		case EventsChannel:
			h.onEvent(ctx, n.Payload)
		case ProgressChannel:
			h.onProgress(n.Payload)
		}
	}
}

func (h *Hub) onEvent(ctx context.Context, payload string) {
	var head struct {
		ID         int64  `json:"id"`
		CampaignID *int64 `json:"campaign_id"`
	}
	if err := json.Unmarshal([]byte(payload), &head); err != nil || head.CampaignID == nil {
		return
	}
	h.mu.Lock()
	_, watched := h.clients[*head.CampaignID]
	h.mu.Unlock()
	if !watched {
		return
	}
	e, err := h.st.GetEvent(ctx, head.ID)
	if err != nil {
		h.log.Warn("lecture d'un événement", "id", head.ID, "err", err)
		return
	}
	h.Stats.NotifyLagMs.Store(time.Since(e.CreatedAt).Milliseconds())
	h.broadcast(*head.CampaignID, EventMessage(e))
}

// EventMessage met un événement du journal au format SSE.
func EventMessage(e store.Event) Message {
	b, _ := json.Marshal(e)
	return Message{ID: e.ID, Type: e.Type, Data: b}
}

func (h *Hub) onProgress(payload string) {
	var head struct {
		CampaignID int64 `json:"campaign_id"`
	}
	if err := json.Unmarshal([]byte(payload), &head); err != nil {
		return
	}
	h.mu.Lock()
	if _, replaced := h.pending[head.CampaignID]; replaced {
		h.Stats.Coalesced.Add(1)
	}
	h.pending[head.CampaignID] = []byte(payload)
	h.mu.Unlock()
}

func (h *Hub) flushLoop(ctx context.Context) {
	t := time.NewTicker(h.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.mu.Lock()
			batch := h.pending
			h.pending = map[int64][]byte{}
			h.mu.Unlock()
			for id, data := range batch {
				h.broadcast(id, Message{Type: "progression", Data: data})
			}
		}
	}
}
