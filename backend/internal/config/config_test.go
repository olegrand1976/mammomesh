package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var keys = []string{"REDIS_URL", "DATABASE_URL", "POSTGRES_PASSWORD", "POSTGRES_PASSWORD_FILE", "POSTGRES_HOST",
	"MCP_TOKEN_PEPPER", "MCP_TOKEN_PEPPER_FILE", "ADMIN_USERNAME", "ADMIN_PASSWORD", "ADMIN_PASSWORD_FILE",
	"ALLOWED_ORIGINS", "SESSION_TTL", "COOKIE_SECURE", "COALESCE_INTERVAL"}

// clean efface les variables lues par Load ; t.Setenv les rétablit en fin de test.
func clean(t *testing.T) {
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestDefaults(t *testing.T) {
	clean(t)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.RedisURL != "redis://redis:6379/0" || c.AdminUsername != "admin" || !c.CookieSecure ||
		c.SessionTTL != 12*time.Hour || c.CoalesceInterval != 2*time.Second || c.ListenAddr != ":8080" {
		t.Fatalf("valeurs par défaut : %+v", c)
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || u.Host != "postgres:5432" || u.Path != "/mammomesh" || u.Query().Get("sslmode") != "disable" {
		t.Fatalf("DATABASE_URL composée : %q", c.DatabaseURL)
	}
}

// REDIS_URL défini mais vide : aucun Redis (compteurs en mémoire).
func TestRedisDisabled(t *testing.T) {
	clean(t)
	t.Setenv("REDIS_URL", "  ")
	if c, _ := Load(); c.RedisURL != "" {
		t.Fatalf("REDIS_URL vide donne %q", c.RedisURL)
	}
}

func TestSecretsFromFiles(t *testing.T) {
	clean(t)
	dir := t.TempDir()
	write := func(name, v string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(v), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	t.Setenv("POSTGRES_PASSWORD_FILE", write("pg", "p@ss/word\n"))
	t.Setenv("MCP_TOKEN_PEPPER_FILE", write("pepper", "poivre\n"))
	t.Setenv("ADMIN_PASSWORD_FILE", write("admin", "admin$1"))
	t.Setenv("POSTGRES_HOST", "db.lan")
	t.Setenv("ALLOWED_ORIGINS", " https://a.lan , ,https://b.lan")
	t.Setenv("SESSION_TTL", "30m")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COALESCE_INTERVAL", "n'importe quoi")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.MCPTokenPepper != "poivre" || c.AdminPassword != "admin$1" {
		t.Fatalf("secrets : %q %q", c.MCPTokenPepper, c.AdminPassword)
	}
	u, _ := url.Parse(c.DatabaseURL)
	if pw, _ := u.User.Password(); pw != "p@ss/word" || u.Hostname() != "db.lan" {
		t.Fatalf("mot de passe ou hôte de la base mal encodé : %q", c.DatabaseURL)
	}
	if len(c.AllowedOrigins) != 2 || c.AllowedOrigins[1] != "https://b.lan" {
		t.Fatalf("origines : %v", c.AllowedOrigins)
	}
	if c.SessionTTL != 30*time.Minute || c.CookieSecure || c.CoalesceInterval != 2*time.Second {
		t.Fatalf("durées ou booléen : %+v", c)
	}

	t.Setenv("ADMIN_PASSWORD_FILE", filepath.Join(dir, "absent"))
	if _, err := Load(); err == nil {
		t.Fatal("secret illisible accepté")
	}
}
