package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Embed the IANA timezone database so time.LoadLocation works in the
	// scratch/minimal deploy container (the AI copilot resolves the user's zone).
	_ "time/tzdata"

	"kinnect-v3/internal/api"
	"kinnect-v3/internal/auth"
	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/config"
	cfglimits "kinnect-v3/internal/config"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/monitoring"
	"kinnect-v3/internal/ws"
)

func main() {
	startedAt := time.Now()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	fmt.Fprintln(os.Stderr, "STARTUP: running InitDB")
	// Run InitDB in a goroutine so we can enforce a hard wall-clock timeout
	// independent of context/driver cancellation. Aiven may run PgBouncer which
	// drops cancel requests, so ExecContext alone cannot guarantee we exit.
	// Wall-clock select guarantees exit ≤ 45s — well inside Render's deploy window.
	initDBResult := make(chan error, 1)
	go func() {
		initDBCtx, initDBCancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer initDBCancel()
		initDBResult <- db.InitDB(initDBCtx, pool.DB)
	}()
	select {
	case initDBErr := <-initDBResult:
		if initDBErr != nil {
			fmt.Fprintf(os.Stderr, "STARTUP FATAL: InitDB failed: %v\n", initDBErr)
			slog.Error("Failed to initialize database schema", "error", initDBErr)
			os.Exit(1)
		}
	case <-time.After(45 * time.Second):
		fmt.Fprintf(os.Stderr, "STARTUP FATAL: InitDB timed out after 45s — DDL lock or DB unavailable\n")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "STARTUP: InitDB OK")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wait for the database to be ready before loading data.
	// On Render free tier, the DB and app cold-start together; the DB may take 5–10s
	// before it accepts connections. Retry up to 5 times with 2s backoff.
	{
		const maxPingAttempts = 5
		const pingBackoff = 2 * time.Second
		for i := 1; i <= maxPingAttempts; i++ {
			pingCtx, pingCancel := context.WithTimeout(ctx, 3*time.Second)
			err := pool.DB.PingContext(pingCtx)
			pingCancel()
			if err == nil {
				slog.Info("Database ready", "attempt", i)
				break
			}
			if i == maxPingAttempts {
				fmt.Fprintf(os.Stderr, "STARTUP FATAL: DB not ready after %d retries: %v\n", maxPingAttempts, err)
				slog.Error("Database not ready after retries", "attempts", maxPingAttempts, "error", err)
				os.Exit(1)
			}
			slog.Warn("Database not yet ready, retrying...", "attempt", i, "error", err)
			time.Sleep(pingBackoff)
		}
	}

	// Load all persistent data (users, rooms, contacts, guardianships, live tokens)
	// into the in-memory cache at startup. This is required so that family members,
	// contacts, and guardianships are visible immediately after deployment/restart.
	// The prior "lazy load" approach was a stub that loaded nothing, causing all
	// relationship data to appear missing until the process was restarted with activity.
	fmt.Fprintln(os.Stderr, "STARTUP: running LoadAll")
	startupCtx, startupCancel := context.WithTimeout(ctx, 30*time.Second)
	result, err := db.LoadAll(startupCtx, pool.DB)
	startupCancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "STARTUP FATAL: LoadAll failed: %v\n", err)
		slog.Error("Failed to load data from database", "error", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "STARTUP: LoadAll OK")

	c := cache.New()
	c.Init(result)

	// Lazy loader for on-demand user lookups (share codes, email/mobile search)
	lazyLoader := cache.NewLazyLoader(pool.DB)
	c.SetLazyLoader(lazyLoader)

	// Initialize monitoring
	metrics := monitoring.NewMetrics()
	monitoringPort := os.Getenv("MONITORING_PORT")
	if monitoringPort == "" {
		monitoringPort = "9090"
	}

	hub := ws.NewHub(c, pool, cfg)
	hub.SetMetrics(metrics)
	go hub.Run(ctx)
	go hub.StartPositionPurger(ctx)
	hub.StartCleanupRoutines(ctx)

	// Start auth IP-limiter cleaner with a shutdown path.
	api.StartAuthCleaner(ctx)

	// Start memory monitoring for free-tier optimization
	go hub.MemoryMonitor.Start(ctx, hub)

	slog.Info("Kinnect initialized",
		"mode", cfg.NodeEnv,
		"max_connections", cfglimits.MaxWebSocketConnections,
		"max_db_connections", cfglimits.MaxDatabaseConnections)

	store := auth.NewSessionStore(pool.DB)
	monServer := monitoring.NewMonitoringServer(monitoringPort, metrics, c, pool.DB)
	var redisCache *cache.RedisCache
	if cfg.RedisURL != "" {
		rc, rcErr := cache.NewRedisCache(ctx, cache.RedisConfig{URL: cfg.RedisURL, Prefix: "kinnect:"})
		if rcErr != nil {
			slog.Warn("Redis/Valkey unavailable, sessions will use PostgreSQL", "error", rcErr)
		} else {
			defer rc.Close()
			redisCache = rc
			store.SetRedis(rc)
			monServer.SetRedis(rc)
			slog.Info("Redis/Valkey wired for session storage")
		}
	}
	// Only pass the Redis health checker when Redis actually connected: a nil
	// *RedisCache wrapped in the interface defeats the handler's nil check and
	// panics /health on every request (typed-nil footgun).
	var handler http.Handler
	if redisCache != nil {
		handler = api.NewRouter(cfg, pool, c, store, hub, redisCache)
	} else {
		handler = api.NewRouter(cfg, pool, c, store, hub)
	}
	go func() {
		if err := monServer.Start(); err != nil && err != http.ErrServerClosed {
			slog.Error("Monitoring server error", "error", err)
		}
	}()

	slog.Info("Kinnect backend-v3 started",
		"port", cfg.Port,
		"monitoring_port", monitoringPort,
		"env", cfg.NodeEnv,
		"users", len(result.UsersCache),
		"rooms", len(result.Rooms))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second, // WebSocket connections are hijacked and unaffected by this
		Handler:      handler,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	metrics.StartupDuration.Set(float64(time.Since(startedAt).Milliseconds()))
	// Sample uptime and DB pool stats every 15s.
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics.ProcessUptime.Set(time.Since(startedAt).Seconds())
				stats := pool.DB.Stats()
				metrics.DBConnectionsActive.Set(float64(stats.InUse))
				metrics.DBConnectionsIdle.Set(float64(stats.Idle))
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	hub.Shutdown(shutdownCtx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}
	if err := monServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("Monitoring server shutdown error", "error", err)
	}
	slog.Info("Goodbye")
}
