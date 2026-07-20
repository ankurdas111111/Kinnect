package ai

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics self-register into the default registry, which the monitoring
// server already exposes via promhttp on MONITORING_PORT.
var (
	metricRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kinnect_ai_requests_total",
		Help: "Ask-the-Map requests by terminal status.",
	}, []string{"status"}) // ok | error | rate_limited

	metricToolCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kinnect_ai_tool_calls_total",
		Help: "Agent tool executions by tool and status.",
	}, []string{"tool", "status"})

	metricLLMCalls = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kinnect_ai_llm_calls_total",
		Help: "LLM API calls by phase (step|synthesis) and status.",
	}, []string{"phase", "status"})

	metricTokens = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kinnect_ai_tokens_total",
		Help: "LLM tokens consumed by kind.",
	}, []string{"kind"}) // prompt | completion

	metricTTFT = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kinnect_ai_time_to_first_token_seconds",
		Help:    "Synthesis-call time to first streamed token.",
		Buckets: []float64{0.25, 0.5, 1, 2, 4, 8, 15, 30},
	})

	metricLLMLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kinnect_ai_llm_step_seconds",
		Help:    "Latency of non-streaming agent-loop LLM calls.",
		Buckets: []float64{0.5, 1, 2, 4, 8, 15, 30},
	})

	metricRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "kinnect_ai_request_seconds",
		Help:    "End-to-end Ask-the-Map request duration.",
		Buckets: []float64{1, 2, 4, 8, 15, 30, 60},
	})

	metricDirectivesDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "kinnect_ai_directives_dropped_total",
		Help: "Map directives dropped by grounding validation.",
	})
)

// RecordRequest finalizes metrics for a request that reached the agent.
func RecordRequest(status string, seconds float64, usage Usage) {
	metricRequests.WithLabelValues(status).Inc()
	metricRequestDuration.Observe(seconds)
	metricTokens.WithLabelValues("prompt").Add(float64(usage.PromptTokens))
	metricTokens.WithLabelValues("completion").Add(float64(usage.CompletionTokens))
}

// RecordRejected counts a request rejected before the agent ran (cooldown,
// in-flight, busy). No duration is observed so the latency histogram stays
// representative of real answers.
func RecordRejected(status string) {
	metricRequests.WithLabelValues(status).Inc()
}
