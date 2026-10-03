package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"finguard.ai/internal/aigateway"
	"finguard.ai/internal/api"
	"finguard.ai/internal/auth"
	"finguard.ai/internal/casesvc"
	"finguard.ai/internal/config"
	"finguard.ai/internal/domain"
	"finguard.ai/internal/graph"
	"finguard.ai/internal/ingestion"
	"finguard.ai/internal/jobs"
	"finguard.ai/internal/knockout"
	"finguard.ai/internal/metrics"
	"finguard.ai/internal/report"
	"finguard.ai/internal/rules"
	"finguard.ai/internal/seed"
	"finguard.ai/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	var err error
	switch os.Args[1] {
	case "api":
		err = runAPI()
	case "worker":
		err = runWorker()
	case "migrate":
		err = runMigrate(false)
	case "rollback":
		err = runMigrate(true)
	case "seed":
		err = runSeed(os.Args[2:])
	case "import":
		err = runImport(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		slog.Error("command failed", "error", err.Error())
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: finguard api|worker|migrate|rollback|seed|import\n")
}

func runAPI() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if cfg.Env != "production" {
		if err := migrateAdmin(ctx, cfg); err != nil {
			return err
		}
	}
	app, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer app.Close()
	srv, worker, err := assemble(cfg, app)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if cfg.WorkerEmbedded {
		go worker.Run(ctx)
	}
	go func() {
		<-ctx.Done()
		shctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shctx)
	}()
	slog.Info("api listening", "addr", cfg.HTTPAddr, "ai_provider", srv.Reports.ProviderName, "live", cfg.LiveModel())
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func runWorker() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	app, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer app.Close()
	_, worker, err := assemble(cfg, app)
	if err != nil {
		return err
	}
	slog.Info("worker running", "ai_provider", cfg.AIProvider)
	worker.Run(ctx)
	return nil
}

func runMigrate(down bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	admin, err := store.Open(ctx, cfg.MigrateDatabaseURL)
	if err != nil {
		return err
	}
	defer admin.Close()
	if down {
		return store.RollbackLatest(ctx, admin.Pool)
	}
	return store.Migrate(ctx, admin.Pool)
}

func runSeed(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	reset := fs.Bool("reset", false, "truncate synthetic business data first")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := migrateAdmin(ctx, cfg); err != nil {
		return err
	}
	app, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer app.Close()
	var adminPool = app.Pool
	if *reset {
		admin, err := store.Open(ctx, cfg.MigrateDatabaseURL)
		if err != nil {
			return err
		}
		defer admin.Close()
		adminPool = admin.Pool
	}
	return seed.Run(ctx, app, adminPool, cfg.DemoPassword, *reset)
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	email := fs.String("email", "investigator@northwind.demo", "synthetic user that owns the import")
	file := fs.String("file", "", "JSON file of events")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return errors.New("--file is required")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var events []ingestion.Event
	if err := json.Unmarshal(raw, &events); err != nil {
		var wrapped struct {
			Events []ingestion.Event `json:"events"`
		}
		if err2 := json.Unmarshal(raw, &wrapped); err2 != nil {
			return err
		}
		events = wrapped.Events
	}
	ctx := context.Background()
	app, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer app.Close()
	user, err := app.UserByEmail(ctx, *email)
	if err != nil {
		return err
	}
	svc := &ingestion.Service{Store: app, Rules: rules.DefaultConfig()}
	p := domain.Principal{UserID: user.ID, TenantID: user.TenantID, Email: user.Email, Role: user.Role}
	for _, event := range events {
		res, err := svc.Ingest(ctx, p, event)
		if err != nil && res.Status == "" {
			return err
		}
		fmt.Printf("%s %s\n", res.IdempotencyKey, res.Status)
	}
	return nil
}

func migrateAdmin(ctx context.Context, cfg config.Config) error {
	if err := store.EnsureAppRole(ctx, cfg.MigrateDatabaseURL); err != nil {
		return err
	}
	admin, err := store.Open(ctx, cfg.MigrateDatabaseURL)
	if err != nil {
		return err
	}
	defer admin.Close()
	return store.Migrate(ctx, admin.Pool)
}

func assemble(cfg config.Config, app *store.Store) (*api.Server, *jobs.Worker, error) {
	provider, name, err := buildProvider(cfg)
	if err != nil {
		return nil, nil, err
	}
	rep := &report.Service{
		Store: app, Provider: provider, ProviderName: name, Timeout: cfg.AITimeout,
		MaxChars: cfg.AIMaxInputChars, MaxTokens: cfg.AIMaxOutputTokens, Retention: cfg.AssistantRetention,
	}
	rep.ConfigureBudget(cfg.AIDailyCallCap)
	worker := &jobs.Worker{Store: app, Report: rep}
	srv := &api.Server{
		Config:    cfg,
		Store:     app,
		Auth:      &auth.Service{Store: app, TTL: cfg.SessionTTL},
		Ingest:    &ingestion.Service{Store: app, Rules: rules.DefaultConfig()},
		Cases:     &casesvc.Service{Store: app},
		Reports:   rep,
		Graph:     &graph.Service{Store: app},
		Knockouts: &knockout.Service{Store: app, Rules: rules.DefaultConfig()},
		Metrics:   &metrics.Metrics{},
	}
	return srv, worker, nil
}

func buildProvider(cfg config.Config) (aigateway.Generator, string, error) {
	stub := aigateway.Retry{Next: aigateway.Stub{}, MaxRetries: 0}
	useGemini := cfg.AIProvider == "gemini" || (cfg.AIProvider == "auto" && cfg.GeminiAPIKey != "")
	if cfg.AIProvider == "gemini" && cfg.GeminiAPIKey == "" {
		return nil, "", errors.New("AI_PROVIDER=gemini requires GEMINI_API_KEY")
	}
	if !useGemini {
		return stub, "local-stub", nil
	}
	client := &http.Client{Timeout: cfg.AITimeout}
	live := &aigateway.Gemini{APIKey: cfg.GeminiAPIKey, Model: cfg.GeminiModel, HTTP: client}
	return aigateway.Retry{Next: live, MaxRetries: 2}, "gemini", nil
}
