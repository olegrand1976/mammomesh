package auth

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter compte les tentatives par clé sur une fenêtre fixe.
// Les compteurs vivent dans Redis ; si Redis manque, un compteur en mémoire prend le relais.
type Limiter struct {
	rdb    *redis.Client
	max    int64
	window time.Duration

	mu    sync.Mutex
	local map[string]localCount
}

type localCount struct {
	n     int64
	until time.Time
}

// NewLimiter crée un limiteur ; rdb peut être nil.
func NewLimiter(rdb *redis.Client, max int64, window time.Duration) *Limiter {
	return &Limiter{rdb: rdb, max: max, window: window, local: map[string]localCount{}}
}

// Allow incrémente le compteur de la clé et dit si la limite est respectée.
func (l *Limiter) Allow(ctx context.Context, key string) bool {
	if l.rdb != nil {
		k := "mammomesh:limite:" + key
		n, err := l.rdb.Incr(ctx, k).Result()
		if err == nil {
			if n == 1 {
				l.rdb.Expire(ctx, k, l.window)
			}
			return n <= l.max
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
	return c.n <= l.max
}

// Reset efface le compteur (connexion réussie).
func (l *Limiter) Reset(ctx context.Context, key string) {
	if l.rdb != nil {
		l.rdb.Del(ctx, "mammomesh:limite:"+key)
	}
	l.mu.Lock()
	delete(l.local, key)
	l.mu.Unlock()
}
