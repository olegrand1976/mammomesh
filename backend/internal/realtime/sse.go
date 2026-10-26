package realtime

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ReplayLimit : au-delà, le navigateur recharge l'état complet au lieu de rejouer le journal.
const ReplayLimit = 1000

// Heartbeat garde la connexion ouverte à travers les proxys.
var Heartbeat = 15 * time.Second

// WriteMessage écrit un message au format SSE.
func WriteMessage(w *bytes.Buffer, m Message) {
	if m.ID > 0 {
		fmt.Fprintf(w, "id: %d\n", m.ID)
	}
	fmt.Fprintf(w, "event: %s\n", m.Type)
	for _, line := range bytes.Split(m.Data, []byte("\n")) {
		w.WriteString("data: ")
		w.Write(line)
		w.WriteByte('\n')
	}
	w.WriteByte('\n')
}

// LastEventID lit l'en-tête Last-Event-ID (reconnexion automatique du navigateur) ou le paramètre lastEventId.
func LastEventID(r *http.Request) int64 {
	v := r.Header.Get("Last-Event-ID")
	if v == "" {
		v = r.URL.Query().Get("lastEventId")
	}
	id, _ := strconv.ParseInt(v, 10, 64)
	if id < 0 {
		return 0
	}
	return id
}

// Serve tient un flux SSE pour une campagne. stillAllowed est vérifiée à chaque battement :
// une session révoquée ou expirée ferme le flux.
func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, campaignID int64, stillAllowed func(context.Context) bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "flux non pris en charge", http.StatusInternalServerError)
		return
	}
	ctx := r.Context()

	// S'abonner avant de rejouer : aucun événement ne tombe entre les deux.
	c := h.Subscribe(campaignID)
	defer h.Unsubscribe(c)

	hdr := w.Header()
	hdr.Set("Content-Type", "text/event-stream")
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("Connection", "keep-alive")
	hdr.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	var buf bytes.Buffer
	buf.WriteString("retry: 3000\n\n")

	last := LastEventID(r)
	if last > 0 {
		events, err := h.st.ListEvents(ctx, campaignID, last, ReplayLimit+1, false)
		if err != nil || len(events) > ReplayLimit {
			WriteMessage(&buf, Message{Type: "resync", Data: []byte(`{}`)})
		} else {
			for _, e := range events {
				WriteMessage(&buf, EventMessage(e))
				last = e.ID
			}
		}
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return
	}
	flusher.Flush()

	beat := time.NewTicker(Heartbeat)
	defer beat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-beat.C:
			if !stillAllowed(ctx) {
				return
			}
			if _, err := w.Write([]byte(": battement\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case m, open := <-c.ch:
			if !open {
				return // trop lent : le navigateur se reconnecte et rejoue depuis Last-Event-ID
			}
			if m.ID > 0 && m.ID <= last {
				continue // déjà envoyé pendant la reprise
			}
			buf.Reset()
			WriteMessage(&buf, m)
			if _, err := w.Write(buf.Bytes()); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
