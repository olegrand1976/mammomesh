package auth

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter compte les tentatives par clé sur une fenêtre fixe.
// Les compteurs vivent dans Redis ; si Redis manque, un compteur en mémoire prend le relais
// et Redis n'est plus sollicité pendant redisBackoff : une panne ne ralentit pas chaque requête.
type Limiter struct {
	rdb    *redis.Client
	max    int64
	window time.Duration

	mu        sync.Mutex
	local     map[string]localCount
	skipUntil time.Time
}

const redisBackoff = 30 * time.Second

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
	if l.redisUp() {
		k := "mammomesh:limite:" + key
		n, err := l.rdb.Incr(ctx, k).Result()
		if err == nil {
			if n == 1 {
				l.rdb.Expire(ctx, k, l.window)
			}
			return n <= l.max
		}
		l.redisDown()
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
	if l.redisUp() {
		if err := l.rdb.Del(ctx, "mammomesh:limite:"+key).Err(); err != nil {
			l.redisDown()
		}
	}
	l.mu.Lock()
	delete(l.local, key)
	l.mu.Unlock()
}

func (l *Limiter) redisUp() bool {
	if l.rdb == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return time.Now().After(l.skipUntil)
}

func (l *Limiter) redisDown() {
	l.mu.Lock()
	l.skipUntil = time.Now().Add(redisBackoff)
	l.mu.Unlock()
}
