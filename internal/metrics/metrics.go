// Package metrics Prometheus 指标导出
//
// 商业标准可观测性：HTTP 请求、引擎队列、LLM 调用、Guard 拦截
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP 请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lexscale_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	
	// 引擎指标
	EngineQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lexscale_engine_queue_depth",
			Help: "Current depth of the intent queue",
		},
	)
	
	EngineIntentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_engine_intents_total",
			Help: "Total number of intents processed",
		},
		[]string{"type", "status"},
	)
	
	// Worker 指标
	WorkerExecutionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_worker_executions_total",
			Help: "Total number of worker executions",
		},
		[]string{"worker", "status"},
	)
	
	WorkerExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lexscale_worker_execution_duration_seconds",
			Help:    "Worker execution duration",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"worker"},
	)
	
	// LLM 指标
	LLMRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_llm_requests_total",
			Help: "Total number of LLM API requests",
		},
		[]string{"provider", "model", "status"},
	)
	
	LLMRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "lexscale_llm_request_duration_seconds",
			Help:    "LLM API request duration",
			Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60, 120},
		},
		[]string{"provider", "model"},
	)
	
	LLMTokensUsed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_llm_tokens_used_total",
			Help: "Total tokens used (prompt + completion)",
		},
		[]string{"provider", "model", "type"},
	)
	
	LLMProviderUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lexscale_llm_provider_up",
			Help: "LLM provider availability (1=up, 0=down)",
		},
		[]string{"provider"},
	)
	
	// Guard 指标
	GuardDecisionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_guard_decisions_total",
			Help: "Total guard decisions",
		},
		[]string{"decision", "rule_id"},
	)
	
	PendingApprovals = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lexscale_pending_approvals",
			Help: "Number of pending approvals",
		},
	)
	
	// 风险发现指标
	FindingsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "lexscale_findings_total",
			Help: "Total risk findings by level",
		},
		[]string{"level", "category"},
	)
	
	FindingsByTier = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lexscale_findings_by_tier",
			Help: "Current findings by evidence tier",
		},
		[]string{"tier"},
	)
	
	// 图规模指标
	GraphEntities = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lexscale_graph_entities",
			Help: "Number of entities in the graph",
		},
		[]string{"matter_id", "type"},
	)
	
	GraphClaims = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lexscale_graph_claims",
			Help: "Number of claims in the graph",
		},
	)
	
	GraphFindings = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lexscale_graph_findings",
			Help: "Number of findings in the graph",
		},
	)
)

// MetricsHandler 返回 Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// InstrumentHTTP HTTP 指标中间件
func InstrumentHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		wrapped := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.status)
		
		HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// RecordWorkerExecution 记录 Worker 执行指标
func RecordWorkerExecution(workerID, status string, duration time.Duration) {
	WorkerExecutionsTotal.WithLabelValues(workerID, status).Inc()
	WorkerExecutionDuration.WithLabelValues(workerID).Observe(duration.Seconds())
}

// RecordLLMRequest 记录 LLM 调用指标
func RecordLLMRequest(provider, model, status string, duration time.Duration, promptTokens, completionTokens int) {
	LLMRequestsTotal.WithLabelValues(provider, model, status).Inc()
	LLMRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
	LLMTokensUsed.WithLabelValues(provider, model, "prompt").Add(float64(promptTokens))
	LLMTokensUsed.WithLabelValues(provider, model, "completion").Add(float64(completionTokens))
}

// RecordGuardDecision 记录 Guard 决策
func RecordGuardDecision(decision, ruleID string) {
	GuardDecisionsTotal.WithLabelValues(decision, ruleID).Inc()
}

// RecordFinding 记录风险发现
func RecordFinding(level, category string) {
	FindingsTotal.WithLabelValues(level, category).Inc()
}
