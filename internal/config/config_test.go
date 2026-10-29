package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadTradeConfig(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "configs", "trade.yaml"))
	if err != nil {
		t.Fatalf("load trade config: %v", err)
	}
	if cfg.Service.Name != "trade" {
		t.Fatalf("service name = %q, want trade", cfg.Service.Name)
	}
	if cfg.Service.RPCAddr != "127.0.0.1:19002" {
		t.Fatalf("rpcAddr = %q", cfg.Service.RPCAddr)
	}
	if cfg.RPC.Timeout != 5*time.Second {
		t.Fatalf("rpc timeout = %v, want 5s", cfg.RPC.Timeout)
	}
	if cfg.Notify.JobInterval != 20*time.Minute {
		t.Fatalf("job interval = %v, want 20m", cfg.Notify.JobInterval)
	}
	if cfg.Notify.HTTPLockLease != 30*time.Second {
		t.Fatalf("http lock lease = %v, want 30s", cfg.Notify.HTTPLockLease)
	}
	if cfg.MySQL.LocName != "Asia/Shanghai" || cfg.MySQL.MaxOpenConns != 15 {
		t.Fatalf("mysql config = %+v", cfg.MySQL)
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "127.0.0.1:9092" {
		t.Fatalf("kafka brokers = %v", cfg.Kafka.Brokers)
	}
	if cfg.DCC.Defaults["scBlacklist"] != "s02c02" {
		t.Fatalf("dcc defaults = %v", cfg.DCC.Defaults)
	}
}

func TestLoadGatewayConfigAppliesDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "configs", "gateway.yaml"))
	if err != nil {
		t.Fatalf("load gateway config: %v", err)
	}
	if cfg.Service.WebDir != "frontend/dist" {
		t.Fatalf("webDir = %q", cfg.Service.WebDir)
	}
	if cfg.Kafka.TopicTeamSuccess != "be261028_team_success" {
		t.Fatalf("default topic = %q", cfg.Kafka.TopicTeamSuccess)
	}
	if cfg.Notify.MaxRetry != 4 {
		t.Fatalf("default maxRetry = %d", cfg.Notify.MaxRetry)
	}
}
