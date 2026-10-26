package mcpserver

import (
	"testing"
	"time"

	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

// Un même principal réutilise son serveur ; un changement de portée ou de rôle en reconstruit un.
func TestServerCache(t *testing.T) {
	c := &serverCache{st: &store.Store{}, ttl: time.Minute, m: map[principalKey]cachedServer{}}
	p := &store.MCPPrincipal{TokenID: 1, Scope: "lecture", User: auth.User{ID: 1, Role: auth.RoleAdmin}}
	a := c.get(p)
	if c.get(&store.MCPPrincipal{TokenID: 1, Scope: "lecture", User: p.User}) != a {
		t.Fatal("serveur non réutilisé")
	}
	if c.get(&store.MCPPrincipal{TokenID: 1, Scope: "proposition", User: p.User}) == a {
		t.Fatal("portée ignorée par le cache")
	}
	c.ttl = -time.Second
	c.m = map[principalKey]cachedServer{}
	b := c.get(p)
	if c.get(p) == b {
		t.Fatal("entrée expirée réutilisée")
	}
}

func BenchmarkHandlerServer(b *testing.B) {
	c := &serverCache{st: &store.Store{}, ttl: time.Minute, m: map[principalKey]cachedServer{}}
	p := &store.MCPPrincipal{TokenID: 1, Scope: "lecture"}
	for b.Loop() {
		c.get(p)
	}
}
