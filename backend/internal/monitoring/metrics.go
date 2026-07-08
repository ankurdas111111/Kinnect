package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application.
// Every metric here is incremented somewhere — decorative metrics that were
// defined but never wired have been removed rather than kept as dead weight.
type Metrics struct {
	// WebSocket metrics
	WSConnectionsActive prometheus.Gauge
	WSConnectionsTotal  prometheus.Counter
	WSMessagesRecv      prometheus.Counter
	WSMessagesSent      prometheus.Counter

	// HubDispatchDuration times each event handler on the hub's single dispatch
	// goroutine — the head-of-line-blocking metric. Anything slow here stalls
	// every connected client.
	HubDispatchDuration *prometheus.HistogramVec

	// DBOffloadDuration times bounded-worker DB tasks (Hub.offloadDB).
	DBOffloadDuration prometheus.Histogram

	// Broadcast drop counter (H-4)
	BroadcastDropped prometheus.Counter

	// Position metrics
	PositionUpdatesTotal prometheus.Counter

	// Database pool gauges (from sql.DBStats, sampled periodically)
	DBConnectionsActive prometheus.Gauge
	DBConnectionsIdle   prometheus.Gauge

	// Application metrics
	StartupDuration prometheus.Gauge
	ProcessUptime   prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		WSConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "ws_connections_active",
			Help: "Number of active WebSocket connections",
		}),
		WSConnectionsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ws_connections_total",
			Help: "Total number of WebSocket connections accepted",
		}),
		WSMessagesRecv: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ws_messages_received_total",
			Help: "Total number of WebSocket messages dispatched to handlers",
		}),
		WSMessagesSent: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ws_messages_sent_total",
			Help: "Total number of WebSocket frames written to clients",
		}),

		HubDispatchDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "hub_dispatch_duration_ms",
			Help:    "Time spent in a WS event handler on the hub dispatch goroutine (ms)",
			Buckets: []float64{0.05, 0.1, 0.5, 1, 5, 10, 50, 100, 500},
		}, []string{"event"}),

		DBOffloadDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "db_offload_duration_ms",
			Help:    "Duration of bounded-worker DB tasks spawned off the hub loop (ms)",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 5000},
		}),

		BroadcastDropped: promauto.NewCounter(prometheus.CounterOpts{
			Name: "kinnect_broadcast_dropped_total",
			Help: "WebSocket broadcast messages dropped due to full channel",
		}),

		PositionUpdatesTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "position_updates_total",
			Help: "Total number of position updates",
		}),

		DBConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of in-use database connections",
		}),
		DBConnectionsIdle: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections",
		}),

		StartupDuration: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "startup_duration_ms",
			Help: "Application startup duration in milliseconds (config load to server listen)",
		}),
		ProcessUptime: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "process_uptime_seconds",
			Help: "Process uptime in seconds",
		}),
	}
}
