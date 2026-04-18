package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"kinnect-v3/internal/cache"
	"kinnect-v3/internal/db"
	"kinnect-v3/internal/shared"
)

var startTime = time.Now()

// redisHealthChecker is satisfied by *cache.RedisCache.
type redisHealthChecker interface {
	HealthCheck(ctx context.Context) bool
}

// wsClientCounter is satisfied by *ws.Hub.
type wsClientCounter interface {
	ClientCount() int
}

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	db      *sql.DB
	cache   *cache.Cache
	redis   redisHealthChecker
	hub     wsClientCounter
	env     string
}

// formatUptime returns a human-readable uptime string like "2h 14m 33s".
func formatUptime(d time.Duration) string {
	d = d.Truncate(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// Health handles GET /health and GET /api/health.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	dbStatus := "ok"
	if err := h.db.PingContext(ctx); err != nil {
		dbStatus = "error"
	}

	redisStatus := "not_configured"
	sessionBackend := "postgres"
	if h.redis != nil {
		sessionBackend = "redis"
		if h.redis.HealthCheck(ctx) {
			redisStatus = "ok"
		} else {
			redisStatus = "error"
		}
	}

	stats := h.db.Stats()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	wsClients := 0
	if h.hub != nil {
		wsClients = h.hub.ClientCount()
	}

	// Only include perf if something has been recorded
	var perfData interface{}
	snap := shared.PerfMetrics.Snapshot()
	counters, _ := snap["counters"].(map[string]int64)
	histograms, _ := snap["histograms"].(map[string]interface{})
	if len(counters) > 0 || len(histograms) > 0 {
		perfData = snap
	}

	response := map[string]interface{}{
		"status":           "ok",
		"env":              h.env,
		"uptime":           formatUptime(time.Since(startTime)),
		"db":               dbStatus,
		"db_pool":          map[string]interface{}{"open": stats.OpenConnections, "in_use": stats.InUse, "idle": stats.Idle},
		"redis":            redisStatus,
		"session_backend":  sessionBackend,
		"ws_clients":       wsClients,
		"rooms":            h.cache.RoomCount(),
		"memory_mb": map[string]interface{}{
			"heap":       mem.HeapAlloc / (1024 * 1024),
			"sys":        mem.Sys / (1024 * 1024),
			"gc_runs":    mem.NumGC,
		},
	}
	if perfData != nil {
		response["perf"] = perfData
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// HealthDb handles GET /health/db and GET /api/diagnostics
func (h *HealthHandler) HealthDb(w http.ResponseWriter, r *http.Request) {
	// For /api/diagnostics, return detailed diagnostics
	if r.URL.Path == "/api/diagnostics" {
		h.Diagnostics(w, r)
		return
	}

	// For /health/db, return table info
	ctx := context.Background()
	tables, totalSize, err := db.GetTableSizes(ctx, h.db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"ok": false, "error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"tables": tables,
		"total": totalSize,
	})
}

// Diagnostics returns detailed system diagnostics
func (h *HealthHandler) Diagnostics(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	if err := h.db.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
		return
	}

	stats := h.db.Stats()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	response := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"runtime": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory_mb": map[string]interface{}{
				"alloc":       int64(mem.Alloc) / (1024 * 1024),
				"total_alloc": int64(mem.TotalAlloc) / (1024 * 1024),
				"sys":         int64(mem.Sys) / (1024 * 1024),
				"heap_alloc":  int64(mem.HeapAlloc) / (1024 * 1024),
				"heap_sys":    int64(mem.HeapSys) / (1024 * 1024),
			},
			"gc": map[string]interface{}{
				"count":     mem.NumGC,
				"pause_ns":  mem.PauseNs[(mem.NumGC+255)%256],
			},
		},
		"database": map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
			"wait_duration_ns": stats.WaitDuration,
		},
		"cache": map[string]interface{}{
			"size_bytes": h.cache.CacheSize(),
		},
	}

	writeJSON(w, http.StatusOK, response)
}
