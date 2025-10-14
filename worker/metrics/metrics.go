package metrics

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	coremetrics "github.com/snow-ghost/agent/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	mode      string
	namespace string

	// Prometheus instruments
	taskReceivedProm         *prometheus.CounterVec
	taskCompletedProm        *prometheus.CounterVec
	taskDurationProm         *prometheus.HistogramVec
	stageDurationProm        *prometheus.HistogramVec
	stageDurationLabeledProm *prometheus.HistogramVec
	llmDesignFailProm        *prometheus.CounterVec
	dslParseFailProm         *prometheus.CounterVec
	validationFailProm       *prometheus.CounterVec
	kbHitsProm               prometheus.Counter
	kbMissesProm             prometheus.Counter
	ragHitsProm              prometheus.Counter

	// OTel instruments
	meter                    metric.Meter
	taskReceivedOTel         metric.Int64Counter
	taskCompletedOTel        metric.Int64Counter
	taskDurationOTel         metric.Float64Histogram
	stageDurationOTel        metric.Float64Histogram
	stageDurationLabeledOTel metric.Float64Histogram
	llmDesignFailOTel        metric.Int64Counter
	dslParseFailOTel         metric.Int64Counter
	validationFailOTel       metric.Int64Counter
	kbHitsOTel               metric.Int64Counter
	kbMissesOTel             metric.Int64Counter
	ragHitsOTel              metric.Int64Counter

	// Sandbox / policy metrics
	sandboxExecTotalProm   *prometheus.CounterVec
	sandboxExecSecondsProm prometheus.Observer
	policyDeniedProm       *prometheus.CounterVec

	sandboxExecTotalOTel   metric.Int64Counter
	sandboxExecSecondsOTel metric.Float64Histogram
	policyDeniedOTel       metric.Int64Counter

	// Evolution / tests metrics
	mutationsTotalProm *prometheus.CounterVec
	testsRunTotalProm  *prometheus.CounterVec
	testsDurationProm  prometheus.Observer

	mutationsTotalOTel metric.Int64Counter
	testsRunTotalOTel  metric.Int64Counter
	testsDurationOTel  metric.Float64Histogram

	// KB and RAG metrics
	kbArtifactsLoadedProm   prometheus.Gauge
	kbSaveArtifactTotalProm prometheus.Counter
	ragSearchTotalProm      *prometheus.CounterVec
	ragSearchDurationProm   prometheus.Observer
	ragCandidatesFoundProm  prometheus.Observer

	kbArtifactsLoadedOTel   metric.Int64Gauge
	kbSaveArtifactTotalOTel metric.Int64Counter
	ragSearchTotalOTel      metric.Int64Counter
	ragSearchDurationOTel   metric.Float64Histogram
	ragCandidatesFoundOTel  metric.Int64Histogram
)

// Init sets up worker metrics according to METRICS_MODE.
func Init() {
	if mode != "" {
		return
	}

	// Ensure global metrics core is initialized
	_ = coremetrics.Init()

	mode = strings.ToLower(strings.TrimSpace(os.Getenv("METRICS_MODE")))
	if mode == "" {
		mode = "prom"
	}
	namespace = os.Getenv("METRICS_NAMESPACE")
	if namespace == "" {
		namespace = "agent"
	}

	if mode == "otel" {
		meter = coremetrics.Meter()
		// If nil, the global provider returns a no-op meter
		if meter == nil {
			meter = otel.Meter("agent-worker")
		}

		taskReceivedOTel, _ = meter.Int64Counter(namespace + ".worker.task_received_total")
		taskCompletedOTel, _ = meter.Int64Counter(namespace + ".worker.task_completed_total")
		taskDurationOTel, _ = meter.Float64Histogram(namespace + ".worker.task_duration_seconds")
		stageDurationOTel, _ = meter.Float64Histogram(namespace + ".worker.solve_stage_seconds")
		stageDurationLabeledOTel, _ = meter.Float64Histogram(namespace + ".worker.solve_stage_seconds_labeled")
		llmDesignFailOTel, _ = meter.Int64Counter(namespace + ".worker.llm_design_fail_total")
		dslParseFailOTel, _ = meter.Int64Counter(namespace + ".worker.dsl_parse_fail_total")
		validationFailOTel, _ = meter.Int64Counter(namespace + ".worker.validation_fail_total")
		kbHitsOTel, _ = meter.Int64Counter(namespace + ".worker.kb_hits_total")
		kbMissesOTel, _ = meter.Int64Counter(namespace + ".worker.kb_misses_total")
		ragHitsOTel, _ = meter.Int64Counter(namespace + ".worker.rag_hits_total")

		// Sandbox / policy
		sandboxExecTotalOTel, _ = meter.Int64Counter(namespace + ".worker.sandbox_exec_total")
		sandboxExecSecondsOTel, _ = meter.Float64Histogram(namespace + ".worker.sandbox_exec_seconds")
		policyDeniedOTel, _ = meter.Int64Counter(namespace + ".worker.policy_denied_total")

		// Evolution / tests
		mutationsTotalOTel, _ = meter.Int64Counter(namespace + ".worker.mutations_total")
		testsRunTotalOTel, _ = meter.Int64Counter(namespace + ".worker.tests_run_total")
		testsDurationOTel, _ = meter.Float64Histogram(namespace + ".worker.tests_duration_seconds")

		// KB and RAG
		kbArtifactsLoadedOTel, _ = meter.Int64Gauge(namespace + ".worker.kb_artifacts_loaded")
		kbSaveArtifactTotalOTel, _ = meter.Int64Counter(namespace + ".worker.kb_save_artifact_total")
		ragSearchTotalOTel, _ = meter.Int64Counter(namespace + ".worker.rag_search_total")
		ragSearchDurationOTel, _ = meter.Float64Histogram(namespace + ".worker.rag_search_duration_seconds")
		ragCandidatesFoundOTel, _ = meter.Int64Histogram(namespace + ".worker.rag_candidates_found")
		return
	}

	// prom
	reg := coremetrics.Registry()
	if reg == nil {
		return
	}

	taskReceivedProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_task_received_total", Help: "Tasks received"},
		[]string{"worker_type", "domain"},
	)
	taskCompletedProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_task_completed_total", Help: "Tasks completed"},
		[]string{"worker_type", "domain", "status"},
	)
	taskDurationProm = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_task_duration_seconds", Help: "E2E task duration", Buckets: prometheus.DefBuckets},
		[]string{"worker_type", "domain"},
	)
	stageDurationProm = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_solve_stage_seconds", Help: "Solve stage duration", Buckets: prometheus.DefBuckets},
		[]string{"stage"},
	)
	stageDurationLabeledProm = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_solve_stage_seconds_labeled", Help: "Solve stage duration (labeled)", Buckets: prometheus.DefBuckets},
		[]string{"stage", "worker_type", "domain"},
	)
	llmDesignFailProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_llm_design_fail_total", Help: "LLM design failures"},
		[]string{"worker_type", "domain"},
	)
	dslParseFailProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_dsl_parse_fail_total", Help: "DSL parse failures"},
		[]string{"worker_type", "domain"},
	)
	validationFailProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_validation_fail_total", Help: "Validation failures"},
		[]string{"worker_type", "domain"},
	)
	kbHitsProm = prometheus.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "worker_kb_hits_total", Help: "KB hits"})
	kbMissesProm = prometheus.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "worker_kb_misses_total", Help: "KB misses"})
	ragHitsProm = prometheus.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "worker_rag_hits_total", Help: "RAG hits"})

	// Sandbox / policy
	sandboxExecTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_sandbox_exec_total", Help: "Sandbox executions"},
		[]string{"result"},
	)
	sandboxExecSeconds := prometheus.NewHistogram(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_sandbox_exec_seconds", Help: "Sandbox execution duration", Buckets: prometheus.DefBuckets},
	)
	sandboxExecSecondsProm = sandboxExecSeconds
	policyDeniedProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_policy_denied_total", Help: "Policy denials"},
		[]string{"reason"},
	)

	// Evolution / tests
	mutationsTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_mutations_total", Help: "Mutations applied"},
		[]string{"kind"},
	)
	testsRunTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_tests_run_total", Help: "Tests executed"},
		[]string{"result"},
	)
	testsDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_tests_duration_seconds", Help: "Per-test duration", Buckets: prometheus.DefBuckets},
	)
	testsDurationProm = testsDuration

	// KB and RAG
	kbArtifactsLoadedProm = prometheus.NewGauge(prometheus.GaugeOpts{Namespace: namespace, Name: "worker_kb_artifacts_loaded", Help: "Number of artifacts loaded in KB"})
	kbSaveArtifactTotalProm = prometheus.NewCounter(prometheus.CounterOpts{Namespace: namespace, Name: "worker_kb_save_artifact_total", Help: "Total artifacts saved to KB"})
	ragSearchTotalProm = prometheus.NewCounterVec(
		prometheus.CounterOpts{Namespace: namespace, Name: "worker_rag_search_total", Help: "RAG searches performed"},
		[]string{"backend"},
	)
	ragSearchDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_rag_search_duration_seconds", Help: "RAG search duration", Buckets: prometheus.DefBuckets},
	)
	ragSearchDurationProm = ragSearchDuration
	ragCandidatesFound := prometheus.NewHistogram(
		prometheus.HistogramOpts{Namespace: namespace, Name: "worker_rag_candidates_found", Help: "Number of candidates found in RAG search", Buckets: prometheus.DefBuckets},
	)
	ragCandidatesFoundProm = ragCandidatesFound

	reg.MustRegister(taskReceivedProm, taskCompletedProm, taskDurationProm, stageDurationProm, stageDurationLabeledProm, llmDesignFailProm, dslParseFailProm, validationFailProm, kbHitsProm, kbMissesProm, ragHitsProm, sandboxExecTotalProm, sandboxExecSeconds, policyDeniedProm, mutationsTotalProm, testsRunTotalProm, testsDuration, kbArtifactsLoadedProm, kbSaveArtifactTotalProm, ragSearchTotalProm, ragSearchDuration, ragCandidatesFound)
}

// WithStage measures a named stage duration and records it.
func WithStage(ctx context.Context, stage string, f func(context.Context)) {
	Init()
	start := time.Now()
	f(ctx)
	dur := time.Since(start).Seconds()
	if mode == "otel" {
		stageDurationOTel.Record(ctx, dur, metric.WithAttributes(attribute.String("stage", stage)))
		return
	}
	if stageDurationProm != nil {
		stageDurationProm.WithLabelValues(stage).Observe(dur)
	}
}

// WithLabeledStage measures a named stage with worker_type and domain labels
func WithLabeledStage(ctx context.Context, stage, workerType, domain string, f func(context.Context)) {
	Init()
	start := time.Now()
	f(ctx)
	dur := time.Since(start).Seconds()
	if mode == "otel" {
		stageDurationLabeledOTel.Record(ctx, dur, metric.WithAttributes(
			attribute.String("stage", stage),
			attribute.String("worker_type", workerType),
			attribute.String("domain", domain),
		))
		return
	}
	if stageDurationLabeledProm != nil {
		stageDurationLabeledProm.WithLabelValues(stage, workerType, domain).Observe(dur)
	}
}

// IncDesignFail increments design failure counters
func IncDesignFail(ctx context.Context, workerType, domain, kind string) {
	Init()
	switch kind {
	case "llm":
		if mode == "otel" {
			llmDesignFailOTel.Add(ctx, 1, metric.WithAttributes(
				attribute.String("worker_type", workerType),
				attribute.String("domain", domain),
			))
			return
		}
		if llmDesignFailProm != nil {
			llmDesignFailProm.WithLabelValues(workerType, domain).Inc()
		}
	case "dsl":
		if mode == "otel" {
			dslParseFailOTel.Add(ctx, 1, metric.WithAttributes(
				attribute.String("worker_type", workerType),
				attribute.String("domain", domain),
			))
			return
		}
		if dslParseFailProm != nil {
			dslParseFailProm.WithLabelValues(workerType, domain).Inc()
		}
	case "validation":
		if mode == "otel" {
			validationFailOTel.Add(ctx, 1, metric.WithAttributes(
				attribute.String("worker_type", workerType),
				attribute.String("domain", domain),
			))
			return
		}
		if validationFailProm != nil {
			validationFailProm.WithLabelValues(workerType, domain).Inc()
		}
	}
}

// ObserveTask increments task counters and observes end-to-end duration.
func ObserveTask(ctx context.Context, start time.Time, status, workerType, domain string) {
	Init()
	duration := time.Since(start).Seconds()
	if mode == "otel" {
		attrs := []attribute.KeyValue{
			attribute.String("worker_type", workerType),
			attribute.String("domain", domain),
		}
		taskCompletedOTel.Add(ctx, 1, metric.WithAttributes(append(attrs, attribute.String("status", status))...))
		taskDurationOTel.Record(ctx, duration, metric.WithAttributes(attrs...))
		return
	}
	if taskCompletedProm != nil && taskDurationProm != nil {
		taskCompletedProm.WithLabelValues(workerType, domain, status).Inc()
		taskDurationProm.WithLabelValues(workerType, domain).Observe(duration)
	}
}

// IncTaskReceived increments the task received counter.
func IncTaskReceived(ctx context.Context, workerType, domain string) {
	Init()
	if mode == "otel" {
		taskReceivedOTel.Add(ctx, 1, metric.WithAttributes(
			attribute.String("worker_type", workerType),
			attribute.String("domain", domain),
		))
		return
	}
	if taskReceivedProm != nil {
		taskReceivedProm.WithLabelValues(workerType, domain).Inc()
	}
}

// IncKBHit increments KB hit counter.
func IncKBHit(ctx context.Context) {
	Init()
	if mode == "otel" {
		kbHitsOTel.Add(ctx, 1)
		return
	}
	kbHitsProm.Inc()
}

// IncKBMiss increments KB miss counter.
func IncKBMiss(ctx context.Context) {
	Init()
	if mode == "otel" {
		kbMissesOTel.Add(ctx, 1)
		return
	}
	kbMissesProm.Inc()
}

// IncRAGHit increments RAG hit counter.
func IncRAGHit(ctx context.Context) {
	Init()
	if mode == "otel" {
		ragHitsOTel.Add(ctx, 1)
		return
	}
	ragHitsProm.Inc()
}

// ObserveSandbox records sandbox execution result and duration.
func ObserveSandbox(ctx context.Context, result string, durSeconds float64) {
	Init()
	if mode == "otel" {
		sandboxExecTotalOTel.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
		sandboxExecSecondsOTel.Record(ctx, durSeconds)
		return
	}
	if sandboxExecTotalProm != nil {
		sandboxExecTotalProm.WithLabelValues(result).Inc()
	}
	if sandboxExecSecondsProm != nil {
		sandboxExecSecondsProm.Observe(durSeconds)
	}
}

// IncPolicyDenied increments a policy denial with a reason label.
func IncPolicyDenied(ctx context.Context, reason string) {
	Init()
	if mode == "otel" {
		policyDeniedOTel.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))
		return
	}
	if policyDeniedProm != nil {
		policyDeniedProm.WithLabelValues(reason).Inc()
	}
}

// IncMutation increments mutation counter by kind (e.g., point|crossover|toggle).
func IncMutation(ctx context.Context, kind string) {
	Init()
	if mode == "otel" {
		mutationsTotalOTel.Add(ctx, 1, metric.WithAttributes(attribute.String("kind", kind)))
		return
	}
	if mutationsTotalProm != nil {
		mutationsTotalProm.WithLabelValues(kind).Inc()
	}
}

// ObserveTest records a single test run result and its duration.
func ObserveTest(ctx context.Context, result string, durationSeconds float64) {
	Init()
	if mode == "otel" {
		testsRunTotalOTel.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
		testsDurationOTel.Record(ctx, durationSeconds)
		return
	}
	if testsRunTotalProm != nil {
		testsRunTotalProm.WithLabelValues(result).Inc()
	}
	if testsDurationProm != nil {
		testsDurationProm.Observe(durationSeconds)
	}
}

// SetKBArtifactsLoaded sets the gauge for loaded artifacts count.
func SetKBArtifactsLoaded(ctx context.Context, count int64) {
	Init()
	if mode == "otel" {
		kbArtifactsLoadedOTel.Record(ctx, count)
		return
	}
	if kbArtifactsLoadedProm != nil {
		kbArtifactsLoadedProm.Set(float64(count))
	}
}

// IncKBSaveArtifact increments the artifact save counter.
func IncKBSaveArtifact(ctx context.Context) {
	Init()
	if mode == "otel" {
		kbSaveArtifactTotalOTel.Add(ctx, 1)
		return
	}
	if kbSaveArtifactTotalProm != nil {
		kbSaveArtifactTotalProm.Inc()
	}
}

// ObserveRAGSearch records a RAG search with backend, duration, and candidate count.
func ObserveRAGSearch(ctx context.Context, backend string, durationSeconds float64, candidatesFound int64) {
	Init()
	if mode == "otel" {
		ragSearchTotalOTel.Add(ctx, 1, metric.WithAttributes(attribute.String("backend", backend)))
		ragSearchDurationOTel.Record(ctx, durationSeconds)
		ragCandidatesFoundOTel.Record(ctx, candidatesFound)
		return
	}
	if ragSearchTotalProm != nil {
		ragSearchTotalProm.WithLabelValues(backend).Inc()
	}
	if ragSearchDurationProm != nil {
		ragSearchDurationProm.Observe(durationSeconds)
	}
	if ragCandidatesFoundProm != nil {
		ragCandidatesFoundProm.Observe(float64(candidatesFound))
	}
}
