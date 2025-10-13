package metrics

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	rt "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

var (
	mode   string
	reg    *prometheus.Registry
	exp    *otelprom.Exporter
	mp     *sdkmetric.MeterProvider
	meter  metric.Meter
	inited bool
)

// Init initializes the metrics subsystem based on METRICS_MODE.
// Modes:
//   - prom (default): creates a dedicated Prometheus registry
//   - otel: configures OTel Prometheus exporter and MeterProvider
func Init() error {
	if inited {
		return nil
	}

	mode = strings.ToLower(strings.TrimSpace(os.Getenv("METRICS_MODE")))
	if mode == "" {
		mode = "prom"
	}

	// For both modes, we may collect runtime metrics depending on env
	collectRuntime := strings.EqualFold(os.Getenv("METRICS_COLLECT_RUNTIME"), "true")

	if mode == "otel" {
		// Configure Prometheus exporter for OTel metrics
		var err error
		exp, err = otelprom.New()
		if err != nil {
			return err
		}

		mp = sdkmetric.NewMeterProvider(sdkmetric.WithReader(exp))
		otel.SetMeterProvider(mp)

		serviceName := os.Getenv("SERVICE_NAME")
		if serviceName == "" {
			serviceName = "agent"
		}
		meter = otel.Meter(serviceName)

		if collectRuntime {
			_ = rt.Start(rt.WithMeterProvider(mp))
		}

		// Prepare a Prometheus registry for optional Go/process collectors
		reg = prometheus.NewRegistry()

		inited = true
		return nil
	}

	// prom mode: create a new registry
	reg = prometheus.NewRegistry()

	if collectRuntime {
		reg.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	inited = true
	return nil
}

// Handler returns the HTTP handler that serves metrics according to the selected mode.
func Handler() http.Handler {
	if mode == "otel" {
		return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

// StartServer starts an HTTP server that serves metrics at the given addr and path.
// addr example: "0.0.0.0:9001"; path example: "/metrics".
// It returns a shutdown function for graceful termination.
func StartServer(ctx context.Context, addr, path string) (func(context.Context) error, error) {
	if !inited {
		if err := Init(); err != nil {
			return nil, err
		}
	}

	if path == "" {
		path = "/metrics"
	}

	mux := http.NewServeMux()
	mux.Handle(path, Handler())

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		// Start server; ignore ErrServerClosed on shutdown
		_ = srv.ListenAndServe()
	}()

	shutdown := func(c context.Context) error {
		// Provide a default timeout if caller didn't
		if _, ok := c.Deadline(); !ok {
			var cancel context.CancelFunc
			c, cancel = context.WithTimeout(c, 5*time.Second)
			defer cancel()
		}
		return srv.Shutdown(c)
	}
	return shutdown, nil
}

// Meter returns the current OTel meter (nil if prom mode).
func Meter() metric.Meter { return meter }

// Registry returns the current Prometheus registry (nil if otel mode).
func Registry() *prometheus.Registry { return reg }
