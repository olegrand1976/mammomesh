package auth

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis garde le client partagé par les limiteurs et sait s'il répond.
// Après une erreur, Redis n'est plus sollicité pendant Backoff : une panne coûte
// une seule tentative, puis les compteurs passent en mémoire sans ralentir les requêtes.
// Une seule ligne de journal marque la panne, une autre le retour.
type Redis struct {
	Client  *redis.Client
	Backoff time.Duration
	log     *slog.Logger

	mu        sync.Mutex
	skipUntil time.Time
	down      bool
}

// NewRedis enveloppe un client ; client nil signifie compteurs en mémoire seulement.
func NewRedis(client *redis.Client, log *slog.Logger) *Redis {
	if log == nil {
		log = slog.Default()
	}
	return &Redis{Client: client, Backoff: 30 * time.Second, log: log}
}

// State vaut « ok », « repli_memoire » (Redis en panne) ou « desactive » (aucun Redis configuré).
func (r *Redis) State() string {
	if r == nil || r.Client == nil {
		return "desactive"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.down {
		return "repli_memoire"
	}
	return "ok"
}

func (r *Redis) usable() bool {
	if r == nil || r.Client == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return time.Now().After(r.skipUntil)
}

func (r *Redis) result(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case err != nil:
		r.skipUntil = time.Now().Add(r.Backoff)
		if !r.down {
			r.down = true
			r.log.Warn("Redis injoignable : compteurs en mémoire, nouvel essai périodique", "err", err, "intervalle", r.Backoff)
		}
	case r.down:
		r.down = false
		r.log.Info("Redis rétabli : compteurs de nouveau partagés")
	}
}

// Probe vérifie Redis au démarrage et fixe l'état initial.
func (r *Redis) Probe(ctx context.Context) {
	if r == nil || r.Client == nil {
		return
	}
	r.result(r.Client.Ping(ctx).Err())
}

// Limiter compte les tentatives par clé sur une fenêtre fixe, dans Redis s'il répond, sinon en mémoire.
type Limiter struct {
	rdb    *Redis
	max    int64
	window time.Duration

	mu    sync.Mutex
	local map[string]localCount
}

type localCount struct {
	n     int64
	until time.Time
}

// NewLimiter crée un limiteur ; rdb peut être nil (compteurs en mémoire).
func NewLimiter(rdb *Redis, max int64, window time.Duration) *Limiter {
	return &Limiter{rdb: rdb, max: max, window: window, local: map[string]localCount{}}
}

// Allow incrémente le compteur de la clé et dit si la limite est respectée.
func (l *Limiter) Allow(ctx context.Context, key string) bool {
	if l.rdb.usable() {
		k := "mammomesh:limite:" + key
		pipe := l.rdb.Client.TxPipeline() // INCR et EXPIRE NX en un aller-retour, sans clé immortelle
		incr := pipe.Incr(ctx, k)
		pipe.ExpireNX(ctx, k, l.window)
		_, err := pipe.Exec(ctx)
		l.rdb.result(err)
		if err == nil {
			return incr.Val() <= l.max
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	c := l.local[key]
	if now.After(c.until) {
		c = localCount{until: now.Add(l.window)}
	}
	c.n++
	l.local[key] = c
	if len(l.local) > 10000 { // purge : une rafale de clés ne fait pas grossir la mémoire sans fin
		for k2, c2 := range l.local {
			if now.After(c2.until) {
				delete(l.local, k2)
			}
		}
	}
	return c.n <= l.max
}

// Reset efface le compteur (connexion réussie).
func (l *Limiter) Reset(ctx context.Context, key string) {
	if l.rdb.usable() {
		l.rdb.result(l.rdb.Client.Del(ctx, "mammomesh:limite:"+key).Err())
	}
	l.mu.Lock()
	delete(l.local, key)
	l.mu.Unlock()
}
