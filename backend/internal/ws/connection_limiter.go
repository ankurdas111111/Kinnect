package ws

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"kinnect-v3/internal/config"
)

// clientIP mirrors api.clientIP (unexported there): prefer the proxy-set
// X-Real-IP / X-Forwarded-For over RemoteAddr, or every connection behind a
// proxy shares one bucket.
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := strings.IndexByte(fwd, ','); idx >= 0 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i >= 0 {
		return ip[:i]
	}
	return ip
}

// ConnectionLimiter enforces hard limits on concurrent WebSocket connections
type ConnectionLimiter struct {
	maxConnections int32
	activeConns    int32
	mu             sync.RWMutex
	rejectedCount  int64
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(maxConnections int) *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConnections: int32(maxConnections),
	}
}

// AcquireConnection attempts to add a new connection.
// Returns true if allowed, false if at limit.
// Uses a CAS loop instead of recursion to avoid stack growth under thundering herd.
func (cl *ConnectionLimiter) AcquireConnection() bool {
	for {
		current := atomic.LoadInt32(&cl.activeConns)
		if current >= cl.maxConnections {
			atomic.AddInt64(&cl.rejectedCount, 1)
			return false
		}
		if atomic.CompareAndSwapInt32(&cl.activeConns, current, current+1) {
			return true
		}
		// CAS lost the race — another goroutine updated first, retry immediately.
	}
}

// ReleaseConnection removes a connection from the limit
func (cl *ConnectionLimiter) ReleaseConnection() {
	atomic.AddInt32(&cl.activeConns, -1)
}

// GetStats returns current statistics
func (cl *ConnectionLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"active_connections": atomic.LoadInt32(&cl.activeConns),
		"max_connections":    cl.maxConnections,
		"rejected_total":     atomic.LoadInt64(&cl.rejectedCount),
		"utilization":        float64(atomic.LoadInt32(&cl.activeConns)) / float64(cl.maxConnections),
	}
}

// InitConnectionLimiter adds a connection limiter to the hub
// This is called once during hub initialization
func (h *Hub) InitConnectionLimiter() {
	h.ConnLimiter = NewConnectionLimiter(config.MaxWebSocketConnections)
	h.ViewerLimiter = NewViewerLimiter(maxViewerConnsPerIP)
	slog.Info("Connection limiter initialized", "max_connections", config.MaxWebSocketConnections)
}

// maxViewerConnsPerIP caps concurrent anonymous share-link sockets from one IP.
// Anonymous viewers have no account to throttle against, so without this a
// single host could drain the global connection pool. A household opening the
// same link on several devices shares one NAT address, hence the headroom.
const maxViewerConnsPerIP = 12

// ViewerLimiter caps concurrent anonymous (sessionless) connections per IP.
type ViewerLimiter struct {
	mu      sync.Mutex
	perIP   map[string]int
	maxPerIP int
}

func NewViewerLimiter(maxPerIP int) *ViewerLimiter {
	return &ViewerLimiter{perIP: make(map[string]int), maxPerIP: maxPerIP}
}

// Acquire reserves a slot for ip, or returns false when the IP is at its cap.
func (vl *ViewerLimiter) Acquire(ip string) bool {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	if vl.perIP[ip] >= vl.maxPerIP {
		return false
	}
	vl.perIP[ip]++
	return true
}

// Release frees a slot; the map entry is deleted at zero so it cannot grow
// without bound across the lifetime of the process.
func (vl *ViewerLimiter) Release(ip string) {
	vl.mu.Lock()
	defer vl.mu.Unlock()
	if vl.perIP[ip] <= 1 {
		delete(vl.perIP, ip)
		return
	}
	vl.perIP[ip]--
}
