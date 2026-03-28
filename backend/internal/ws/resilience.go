package ws

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"kinnect-v3/internal/cache"
)

// HealthCheckService performs regular health checks
type HealthCheckService struct {
	db              *sql.DB
	cache           *cache.Cache
	lastCheck       time.Time
	checkInterval   time.Duration
	unhealthyCount  int
	maxUnhealthy    int
}

// NewHealthCheckService creates a new health check service
func NewHealthCheckService(database *sql.DB, c *cache.Cache) *HealthCheckService {
	return &HealthCheckService{
		db:            database,
		cache:         c,
		checkInterval: 30 * time.Second,
		maxUnhealthy:  3, // Allow 3 consecutive unhealthy checks before alerting
	}
}

// StartHealthChecks begins periodic health checks
func (hcs *HealthCheckService) StartHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(hcs.checkInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hcs.PerformCheck()
			}
		}
	}()
}

// PerformCheck performs a single health check
func (hcs *HealthCheckService) PerformCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := &HealthStatus{
		Timestamp:  time.Now().UnixMilli(),
		IsHealthy:  true,
		Checks:     make(map[string]interface{}),
	}

	// Check database connectivity
	dbOk := hcs.checkDatabase(ctx)
	status.Checks["database"] = dbOk
	if !dbOk {
		status.IsHealthy = false
	}

	// Check cache
	cacheOk := hcs.checkCache()
	status.Checks["cache"] = cacheOk
	if !cacheOk {
		status.IsHealthy = false
	}

	// Check memory
	memOk := hcs.checkMemory()
	status.Checks["memory"] = memOk
	if !memOk {
		status.IsHealthy = false
	}

	// Update health status
	if !status.IsHealthy {
		hcs.unhealthyCount++
		if hcs.unhealthyCount >= hcs.maxUnhealthy {
			slog.Error("Service unhealthy", "checks", status.Checks, "count", hcs.unhealthyCount)
		}
	} else {
		hcs.unhealthyCount = 0
	}

	hcs.lastCheck = time.Now()
}

// HealthStatus represents health check result
type HealthStatus struct {
	Timestamp int64                  `json:"timestamp"`
	IsHealthy bool                   `json:"healthy"`
	Checks    map[string]interface{} `json:"checks"`
}

// checkDatabase checks if database is responding
func (hcs *HealthCheckService) checkDatabase(ctx context.Context) bool {
	err := hcs.db.PingContext(ctx)
	return err == nil
}

// checkCache checks if cache has data
func (hcs *HealthCheckService) checkCache() bool {
	// Simple check: cache should have some data or be empty (ok)
	return hcs.cache != nil
}

// checkMemory checks if memory is within acceptable bounds
func (hcs *HealthCheckService) checkMemory() bool {
	stats := GetMemoryStats()
	heapMB := stats["heap_alloc_mb"].(float64)
	return heapMB < 400 // Critical threshold
}

// ClientKeepAliveService manages client keep-alive mechanism
type ClientKeepAliveService struct {
	interval       time.Duration
	clients        map[string]time.Time
	mu             sync.RWMutex
	lastActivity   sync.Map // clientID -> last activity time
}

// NewClientKeepAliveService creates a new keep-alive service
func NewClientKeepAliveService() *ClientKeepAliveService {
	return &ClientKeepAliveService{
		interval: 14 * time.Minute, // Render spins down after 15 min inactivity
		clients:  make(map[string]time.Time),
	}
}

// StartKeepAlive begins keep-alive monitoring
func (cka *ClientKeepAliveService) StartKeepAlive(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cka.pruneInactive()
			}
		}
	}()
}

// RecordActivity records client activity
func (cka *ClientKeepAliveService) RecordActivity(clientID string) {
	cka.lastActivity.Store(clientID, time.Now())
}

// IsActive checks if a client is still active
func (cka *ClientKeepAliveService) IsActive(clientID string) bool {
	val, ok := cka.lastActivity.Load(clientID)
	if !ok {
		return false
	}

	lastTime := val.(time.Time)
	return time.Since(lastTime) < 30*time.Minute // 30 min max inactivity
}

// pruneInactive removes inactive clients from tracking
func (cka *ClientKeepAliveService) pruneInactive() {
	cutoff := time.Now().Add(-30 * time.Minute)

	cka.lastActivity.Range(func(key, value interface{}) bool {
		lastTime := value.(time.Time)
		if lastTime.Before(cutoff) {
			cka.lastActivity.Delete(key)
		}
		return true
	})
}

// GracefulShutdown handles clean shutdown
type GracefulShutdown struct {
	timeout time.Duration
	once    sync.Once
	done    chan struct{}
}

// NewGracefulShutdown creates a new graceful shutdown handler
func NewGracefulShutdown(timeout time.Duration) *GracefulShutdown {
	return &GracefulShutdown{
		timeout: timeout,
		done:    make(chan struct{}),
	}
}

// Shutdown initiates graceful shutdown
func (gs *GracefulShutdown) Shutdown(ctx context.Context, hub *Hub) error {
	var err error

	gs.once.Do(func() {
		slog.Info("Graceful shutdown initiated", "timeout_sec", gs.timeout.Seconds())

		// Stop accepting new connections
		hub.IsShuttingDown = true

		// Close all existing connections
		hub.mu.RLock()
		clientCount := len(hub.clients)
		hub.mu.RUnlock()

		slog.Info("Closing client connections", "count", clientCount)

		// Wait for clients to disconnect gracefully
		shutdownCtx, cancel := context.WithTimeout(ctx, gs.timeout)
		defer cancel()

		// Drain dispatch queue
	drain:
		for {
			select {
			case <-shutdownCtx.Done():
				slog.Warn("Shutdown timeout reached, force closing remaining connections")
				return
			default:
				hub.mu.RLock()
				empty := len(hub.clients) == 0
				hub.mu.RUnlock()
				if empty {
					break drain
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		slog.Info("Graceful shutdown complete")
		close(gs.done)
	})

	return err
}

