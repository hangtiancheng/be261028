package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Business counters shared across services. They are registered on the default
// Prometheus registry via promauto so /metrics exposes them automatically.
var (
	// RPCCallsTotal counts inbound/outbound RPC invocations by service, method
	// and result ("success" or "error").
	RPCCallsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "be261028",
		Name:      "rpc_calls_total",
		Help:      "Total RPC calls by service, method and result.",
	}, []string{"service", "method", "result"})

	// HTTPRequestsTotal counts gateway HTTP responses by path and business code.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "be261028",
		Name:      "http_requests_total",
		Help:      "Total HTTP requests by path and response code.",
	}, []string{"path", "code"})

	// TradeOpsTotal counts core trade operations by op and result code.
	TradeOpsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "be261028",
		Name:      "trade_ops_total",
		Help:      "Total trade operations (lock/settlement/refund) by result code.",
	}, []string{"op", "code"})

	// NotifyTasksTotal counts notify-task executions by result.
	NotifyTasksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "be261028",
		Name:      "notify_tasks_total",
		Help:      "Total notify tasks by result (success/error/retry/null).",
	}, []string{"result"})

	// KafkaMessagesTotal counts Kafka messages produced/consumed by topic and side.
	KafkaMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "be261028",
		Name:      "kafka_messages_total",
		Help:      "Total Kafka messages by topic and side (produce/consume).",
	}, []string{"topic", "side"})
)

// Handler returns an http.Handler exposing /metrics and /health, suitable for
// mounting on an existing server (e.g. the gateway) or running standalone.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"UP"}`))
	})
	return mux
}

// Server runs a standalone metrics/health HTTP server. RPC-only services use it
// to stay scrapeable by Prometheus. The returned *http.Server is already
// listening; call Shutdown to stop it.
func Server(addr string) (*http.Server, error) {
	srv := &http.Server{
		Addr:              addr,
		Handler:           Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// The caller monitors liveness via /health; log-free best effort.
			return
		}
	}()
	return srv, nil
}

// Shutdown gracefully stops a metrics server.
func Shutdown(srv *http.Server) {
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
