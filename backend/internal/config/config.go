// Package config lit la configuration depuis l'environnement et les secrets Docker.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config regroupe les réglages des services Go.
type Config struct {
	DatabaseURL      string
	RedisURL         string
	ListenAddr       string
	MetricsAddr      string
	CookieSecure     bool
	SessionTTL       time.Duration
	AllowedOrigins   []string
	MCPTokenPepper   string
	CoalesceInterval time.Duration
	AdminUsername    string
	AdminPassword    string
	LogLevel         string
}

// Load construit la configuration. Un secret se lit dans le fichier désigné par NOM_FILE
// (secrets Docker) ou, à défaut, dans la variable NOM.
func Load() (Config, error) {
	c := Config{
		RedisURL:         redisURL(),
		ListenAddr:       env("LISTEN_ADDR", ":8080"),
		MetricsAddr:      env("METRICS_ADDR", ":9090"),
		CookieSecure:     envBool("COOKIE_SECURE", true),
		SessionTTL:       envDuration("SESSION_TTL", 12*time.Hour),
		CoalesceInterval: envDuration("COALESCE_INTERVAL", 2*time.Second),
		AdminUsername:    env("ADMIN_USERNAME", "admin"),
		LogLevel:         env("LOG_LEVEL", "info"),
	}
	if o := env("ALLOWED_ORIGINS", ""); o != "" {
		for _, s := range strings.Split(o, ",") {
			if s = strings.TrimSpace(s); s != "" {
				c.AllowedOrigins = append(c.AllowedOrigins, s)
			}
		}
	}

	var err error
	if c.MCPTokenPepper, err = secret("MCP_TOKEN_PEPPER"); err != nil {
		return c, err
	}
	if c.AdminPassword, err = secret("ADMIN_PASSWORD"); err != nil {
		return c, err
	}

	c.DatabaseURL = env("DATABASE_URL", "")
	if c.DatabaseURL == "" {
		pw, err := secret("POSTGRES_PASSWORD")
		if err != nil {
			return c, err
		}
		u := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(env("POSTGRES_USER", "mammomesh"), pw),
			Host:     env("POSTGRES_HOST", "postgres") + ":" + env("POSTGRES_PORT", "5432"),
			Path:     "/" + env("POSTGRES_DB", "mammomesh"),
			RawQuery: "sslmode=" + env("POSTGRES_SSLMODE", "disable"),
		}
		c.DatabaseURL = u.String()
	}
	return c, nil
}

// redisURL : absent, l'adresse de la pile Docker ; défini vide, aucun Redis (compteurs en mémoire).
func redisURL() string {
	if v, ok := os.LookupEnv("REDIS_URL"); ok {
		return strings.TrimSpace(v)
	}
	return "redis://redis:6379/0"
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, err := strconv.ParseBool(env(key, strconv.FormatBool(def)))
	if err != nil {
		return def
	}
	return v
}

func envDuration(key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(env(key, def.String()))
	if err != nil {
		return def
	}
	return d
}

func secret(key string) (string, error) {
	if path := os.Getenv(key + "_FILE"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("secret %s illisible : %w", key, err)
		}
		return strings.TrimSpace(string(b)), nil
	}
	return os.Getenv(key), nil
}
