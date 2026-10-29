// Package bootstrap holds shared startup helpers for the be261028 services:
// config loading, signal handling and RPC server lifecycle.
package bootstrap

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/hangtiancheng/be261028/internal/config"
	yukinorpc "github.com/hangtiancheng/yukino.go/libs/yukino_rpc/pkg/rpc"
)

// ConfigPath parses the -config flag with the supplied default.
func ConfigPath(defaultPath string) string {
	var path string
	flag.StringVar(&path, "config", defaultPath, "path to the YAML config file")
	flag.Parse()
	if path == "" {
		path = defaultPath
	}
	return path
}

// LoadConfig reads and validates the config file.
func LoadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	if cfg.Service.Name == "" {
		return nil, fmt.Errorf("config: service.name is required")
	}
	return cfg, nil
}

// SignalContext returns a context cancelled on SIGINT/SIGTERM.
func SignalContext() context.Context {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	_ = stop
	return ctx
}

// ServeRPC starts a yukino_rpc server on addr with the given service registered
// under name. It returns the server (for graceful stop) once the listener is up.
func ServeRPC(addr, name string, service any) (*yukinorpc.Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen rpc %s: %w", addr, err)
	}
	server := yukinorpc.NewServer()
	server.Register(name, service)
	go func() {
		if err := server.Serve(lis); err != nil {
			log.Printf("[rpc] %s serve stopped: %v", name, err)
		}
	}()
	log.Printf("[rpc] %s listening on %s", name, addr)
	return server, nil
}
