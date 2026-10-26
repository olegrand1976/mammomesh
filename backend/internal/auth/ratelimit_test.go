package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func quickClient(opt *redis.Options) *redis.Client {
	opt.DialTimeout, opt.ReadTimeout, opt.WriteTimeout = 200*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond
	opt.MaxRetries, opt.DialerRetries = -1, 1
	return redis.NewClient(opt)
}

func expect(t *testing.T, l *Limiter, key string, want ...bool) {
	t.Helper()
	for i, w := range want {
		if got := l.Allow(context.Background(), key); got != w {
			t.Fatalf("%s, essai %d : %v, attendu %v", key, i+1, got, w)
		}
	}
}

// Redis injoignable : le repli en mémoire compte juste, reste rapide et l'état le dit.
func TestLimiterRedisDown(t *testing.T) {
	c := quickClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer c.Close()
	r := NewRedis(c, nil)
	l := NewLimiter(r, 3, time.Minute)
	start := time.Now()
	expect(t, l, "k", true, true, true, false, false)
	l.Reset(context.Background(), "k")
	expect(t, l, "k", true)
	if d := time.Since(start); d > time.Second {
		t.Fatalf("repli trop lent : %v", d)
	}
	if s := r.State(); s != "repli_memoire" {
		t.Fatalf("état %q", s)
	}
	if s := NewRedis(nil, nil).State(); s != "desactive" {
		t.Fatalf("état sans Redis %q", s)
	}
}

// Avec un vrai Redis (REDIS_URL_TEST) : compteur partagé, durée de vie posée, repli puis retour.
func TestLimiterRedis(t *testing.T) {
	url := os.Getenv("REDIS_URL_TEST")
	if url == "" {
		t.Skip("REDIS_URL_TEST absent")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	c := quickClient(opt)
	defer c.Close()
	ctx := context.Background()
	key := "essai-" + time.Now().Format("150405.000000")
	defer c.Del(ctx, "mammomesh:limite:"+key)

	r := NewRedis(c, nil)
	r.Backoff = 200 * time.Millisecond
	r.Probe(ctx)
	if r.State() != "ok" {
		t.Fatalf("état initial %q", r.State())
	}
	a, b := NewLimiter(r, 3, time.Minute), NewLimiter(r, 3, time.Minute)
	expect(t, a, key, true, true)
	expect(t, b, key, true, false) // second limiteur : même compteur, comme une seconde instance de l'API
	if ttl := c.TTL(ctx, "mammomesh:limite:"+key).Val(); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("durée de vie %v", ttl)
	}

	// Redis gelé 400 ms : repli immédiat, puis retour automatique après Backoff.
	other := quickClient(&redis.Options{Addr: opt.Addr, Password: opt.Password, DB: opt.DB})
	defer other.Close()
	if err := other.Do(ctx, "CLIENT", "PAUSE", 400, "ALL").Err(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	expect(t, a, key+"-repli", true)
	if r.State() != "repli_memoire" || time.Since(start) > 300*time.Millisecond {
		t.Fatalf("repli : état %q en %v", r.State(), time.Since(start))
	}
	time.Sleep(700 * time.Millisecond)
	expect(t, a, key+"-retour", true)
	if r.State() != "ok" {
		t.Fatalf("Redis non repris : %q", r.State())
	}
	c.Del(ctx, "mammomesh:limite:"+key+"-retour")
}
