// Commande mammomesh : un seul binaire pour l'API, l'orchestrateur, les migrations et l'administration.
//
//	mammomesh api            API REST, flux SSE, serveur MCP (/mcp)
//	mammomesh orchestrator   campagnes en cours (un seul actif à la fois)
//	mammomesh migrate        migrations, puis premier administrateur si aucun compte n'existe
//	mammomesh user create    création d'un compte (mot de passe lu sur l'entrée standard)
//	mammomesh mcp-stdio      serveur MCP en stdio pour le développement (jeton dans MAMMOMESH_MCP_TOKEN)
//	mammomesh healthcheck    test de santé du conteneur
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/redis/go-redis/v9"

	"github.com/olegrand1976/mammomesh/backend/internal/api"
	"github.com/olegrand1976/mammomesh/backend/internal/auth"
	"github.com/olegrand1976/mammomesh/backend/internal/config"
	"github.com/olegrand1976/mammomesh/backend/internal/db"
	"github.com/olegrand1976/mammomesh/backend/internal/mcpserver"
	"github.com/olegrand1976/mammomesh/backend/internal/orchestrator"
	"github.com/olegrand1976/mammomesh/backend/internal/realtime"
	"github.com/olegrand1976/mammomesh/backend/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage : mammomesh api | orchestrator | migrate | user create | mcp-stdio | healthcheck")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	// En stdio, la sortie standard porte le protocole MCP : les journaux vont sur stderr.
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	switch os.Args[1] {
	case "healthcheck":
		healthcheck(cfg)
	case "migrate":
		err = runMigrate(ctx, cfg, log)
	case "api":
		err = runAPI(ctx, cfg, log)
	case "orchestrator":
		err = runOrchestrator(ctx, cfg, log)
	case "user":
		err = runUser(ctx, cfg, os.Args[2:])
	case "mcp-stdio":
		err = runMCPStdio(ctx, cfg)
	default:
		err = fmt.Errorf("commande inconnue %q", os.Args[1])
	}
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erreur :", err)
	os.Exit(1)
}

func openStore(ctx context.Context, cfg config.Config) (*store.Store, error) {
	if cfg.MCPTokenPepper == "" {
		return nil, errors.New("MCP_TOKEN_PEPPER manquant (secret mcp_token_pepper)")
	}
	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	return store.New(pool, cfg.MCPTokenPepper), nil
}

func runMigrate(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Pool.Close()
	applied, err := db.Migrate(ctx, st.Pool)
	if err != nil {
		return err
	}
	log.Info("migrations appliquées", "nouvelles", applied)

	n, err := st.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n == 0 && cfg.AdminUsername != "" && cfg.AdminPassword != "" {
		if _, err := st.CreateUser(ctx, cfg.AdminUsername, "", cfg.AdminPassword, auth.RoleAdmin); err != nil {
			return fmt.Errorf("premier administrateur : %w", err)
		}
		log.Info("premier administrateur créé", "identifiant", cfg.AdminUsername)
	}
	return nil
}

func runAPI(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Pool.Close()

	var rdb *redis.Client
	if opt, err := redis.ParseURL(cfg.RedisURL); err == nil {
		// Réseau local : une réponse arrive en quelques ms ; au-delà, on bascule vite en mémoire.
		opt.DialTimeout, opt.ReadTimeout, opt.WriteTimeout = 300*time.Millisecond, 300*time.Millisecond, 300*time.Millisecond
		opt.MaxRetries, opt.DialerRetries = -1, 1 // pas de nouvel essai : le limiteur a son propre repli
		rdb = redis.NewClient(opt)
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Warn("Redis injoignable : limitation des tentatives en mémoire", "err", err)
		}
		defer rdb.Close()
	}

	hub := realtime.NewHub(st, log, cfg.CoalesceInterval)
	go hub.Run(ctx)

	srv := api.New(cfg, st, hub, log, api.Limits{
		LoginPerUser: auth.NewLimiter(rdb, 10, 15*time.Minute), // 10 tentatives par identifiant et par 15 minutes
		LoginPerIP:   auth.NewLimiter(rdb, 50, 15*time.Minute), // 50 par adresse : un poste partagé reste utilisable
		MCPPerToken:  auth.NewLimiter(rdb, 120, time.Minute),   // 120 appels MCP par minute et par jeton
	})
	httpSrv := &http.Server{Addr: cfg.ListenAddr, Handler: srv.Router(), ReadHeaderTimeout: 10 * time.Second}
	metricsSrv := &http.Server{Addr: cfg.MetricsAddr, Handler: api.MetricsHandler(hub), ReadHeaderTimeout: 5 * time.Second}

	errc := make(chan error, 2)
	go func() { errc <- httpSrv.ListenAndServe() }()
	go func() { errc <- metricsSrv.ListenAndServe() }()
	log.Info("API à l'écoute", "adresse", cfg.ListenAddr, "metriques", cfg.MetricsAddr)

	select {
	case <-ctx.Done():
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = metricsSrv.Shutdown(shutdown)
	return httpSrv.Shutdown(shutdown)
}

func runOrchestrator(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Pool.Close()
	return orchestrator.New(st, log).Run(ctx)
}

func runUser(ctx context.Context, cfg config.Config, args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return errors.New("usage : mammomesh user create --username nom --role rôle [--name « Nom affiché »] < mot_de_passe")
	}
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	username := fs.String("username", "", "identifiant")
	role := fs.String("role", auth.RoleReader, "rôle : "+strings.Join(auth.Roles, ", "))
	name := fs.String("name", "", "nom affiché")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if *username == "" {
		return errors.New("--username requis")
	}
	fmt.Fprint(os.Stderr, "Mot de passe (12 caractères au moins) : ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return errors.New("mot de passe attendu sur l'entrée standard")
	}
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Pool.Close()
	id, err := st.CreateUser(ctx, *username, *name, strings.TrimRight(line, "\r\n"), *role)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "\nCompte %d créé : %s (%s)\n", id, *username, *role)
	return nil
}

func runMCPStdio(ctx context.Context, cfg config.Config) error {
	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Pool.Close()
	p, err := st.ResolveMCPToken(ctx, os.Getenv("MAMMOMESH_MCP_TOKEN"))
	if err != nil {
		return errors.New("MAMMOMESH_MCP_TOKEN absent ou invalide")
	}
	return mcpserver.NewServer(st, p).Run(ctx, &mcp.StdioTransport{})
}

func healthcheck(cfg config.Config) {
	addr := cfg.ListenAddr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://" + addr + "/api/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}
