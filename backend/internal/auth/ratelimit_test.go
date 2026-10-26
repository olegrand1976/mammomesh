package auth

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis injoignable : le repli en mémoire doit compter juste et rester rapide.
func TestLimiterRedisDown(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond, MaxRetries: -1, DialerRetries: 1})
	defer rdb.Close()
	l := NewLimiter(rdb, 3, time.Minute)
	ctx := context.Background()
	start := time.Now()
	for i := 1; i <= 5; i++ {
		if got, want := l.Allow(ctx, "k"), i <= 3; got != want {
			t.Fatalf("essai %d : %v, attendu %v", i, got, want)
		}
	}
	l.Reset(ctx, "k")
	if !l.Allow(ctx, "k") {
		t.Fatal("compteur non remis à zéro")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("repli trop lent : %v", d)
	}
}
