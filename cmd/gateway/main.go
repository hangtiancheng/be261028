// Command gateway runs the be261028 HTTP edge service. It exposes the legacy REST
// API on yukino_http and forwards to the activity and trade services over
// yukino_rpc. It also publishes dynamic-config updates to Redis (mirroring the
// legacy DCCController) and exposes Prometheus metrics and health on the same port.
package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/hangtiancheng/be261028/internal/bootstrap"
	"github.com/hangtiancheng/be261028/internal/gateway"
	"github.com/hangtiancheng/be261028/internal/infra/dcc"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/rpc"
	yukinohttp "github.com/hangtiancheng/yukino.go/libs/yukino_http"
	yukinorpc "github.com/hangtiancheng/yukino.go/libs/yukino_rpc/pkg/rpc"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := bootstrap.LoadConfig(bootstrap.ConfigPath("configs/gateway.yaml"))
	if err != nil {
		log.Fatalf("[gateway] failed to load config: %v", err)
	}
	log.Printf("[gateway] starting service name:%s http:%s", cfg.Service.Name, cfg.Service.HTTPAddr)

	ctx := bootstrap.SignalContext()

	// Redis (used to publish DCC dynamic-config updates).
	redis, err := redisx.New(cfg.Redis)
	if err != nil {
		log.Fatalf("[gateway] failed to connect to Redis: %v", err)
	}
	defer redis.Close()
	dccService := dcc.New(cfg.DCC.Topic, cfg.DCC.Defaults)

	// RPC clients.
	activityConn, err := yukinorpc.Dial(cfg.RPC.ActivityTarget, yukinorpc.WithTimeout(cfg.RPC.Timeout))
	if err != nil {
		log.Fatalf("[gateway] failed to connect to the activity service: %v", err)
	}
	defer activityConn.Close()
	tradeConn, err := yukinorpc.Dial(cfg.RPC.TradeTarget, yukinorpc.WithTimeout(cfg.RPC.Timeout))
	if err != nil {
		log.Fatalf("[gateway] failed to connect to the trade service: %v", err)
	}
	defer tradeConn.Close()

	activityCli := rpc.NewActivityClient(activityConn)
	tradeCli := rpc.NewTradeClient(tradeConn)

	// HTTP application.
	app := yukinohttp.Default()
	gw := gateway.New(activityCli, tradeCli, dccService, redis)
	gw.Register(app)
	mountOps(app)
	mountWeb(app, cfg.Service.WebDir)

	srv := &http.Server{
		Addr:              cfg.Service.HTTPAddr,
		Handler:           app,
		ReadHeaderTimeout: 15 * time.Second,
	}
	go func() {
		log.Printf("[gateway] HTTP listening on %s", cfg.Service.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[gateway] HTTP server error: %v", err)
		}
	}()

	// Wait for the shutdown signal.
	<-ctx.Done()
	log.Printf("[gateway] shutdown signal received, starting graceful stop")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[gateway] HTTP shutdown error: %v", err)
	}
	log.Printf("[gateway] stopped")
}

// mountOps exposes Prometheus metrics and a health probe on the main HTTP port,
// mirroring the legacy actuator endpoints.
func mountOps(app *yukinohttp.Application) {
	promHandler := promhttp.Handler()
	app.Get("/metrics", func(ctx *yukinohttp.Context, next func()) {
		// TakeOver disables deferred rendering (which would otherwise promote the
		// implicit 404 to 200). Record the real status so the access logger and
		// metrics middleware report 200 instead of a misleading 404.
		ctx.SetStatus(http.StatusOK)
		ctx.TakeOver()
		promHandler.ServeHTTP(ctx.Writer, ctx.Request)
	})
	app.Get("/health", func(ctx *yukinohttp.Context, next func()) {
		ctx.JSON(yukinohttp.H{"status": "UP"})
	})
}

// mountWeb serves the migrated demo frontend (static files) from webDir. The
// legacy frontend called the backend at http://127.0.0.1:8091, which is exactly
// this gateway, so it works unchanged. The static wildcard requires a path segment,
// so the bare "/" gets its own handler that serves index.html directly (a redirect
// to /index.html would loop, because http.FileServer canonicalises /index.html
// back to /).
func mountWeb(app *yukinohttp.Application, webDir string) {
	if webDir == "" {
		return
	}
	app.Static("/", webDir)
	app.Get("/", func(ctx *yukinohttp.Context, next func()) {
		ctx.SetStatus(http.StatusOK)
		ctx.TakeOver()
		http.ServeFile(ctx.Writer, ctx.Request, filepath.Join(webDir, "index.html"))
	})
	log.Printf("[gateway] static frontend dir: %s", webDir)
}
