package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

// Config is the unified configuration for every be261028 service. Each service
// loads its own YAML file and consumes the sections relevant to it.
type Config struct {
	Service ServiceConfig `yaml:"service"`
	RPC     RPCConfig     `yaml:"rpc"`
	MySQL   MySQLConfig   `yaml:"mysql"`
	Redis   RedisConfig   `yaml:"redis"`
	Kafka   KafkaConfig   `yaml:"kafka"`
	DCC     DCCConfig     `yaml:"dcc"`
	Notify  NotifyConfig  `yaml:"notify"`
	Log     LogConfig     `yaml:"log"`
}

// ServiceConfig identifies the running service and its listen addresses.
type ServiceConfig struct {
	Name        string `yaml:"name"`
	HTTPAddr    string `yaml:"httpAddr"`
	RPCAddr     string `yaml:"rpcAddr"`
	MetricsAddr string `yaml:"metricsAddr"`
	// WebDir is the static directory served by the gateway (relative to
	// the working directory). Empty disables static serving.
	WebDir string `yaml:"webDir"`
}

// RPCConfig holds the dial targets for cross-service RPC (static mode; the etcd
// registry is not published to the host in the local docker-compose).
type RPCConfig struct {
	ActivityTarget string        `yaml:"activityTarget"`
	TradeTarget    string        `yaml:"tradeTarget"`
	Timeout        time.Duration `yaml:"timeout"`
}

// MySQLConfig describes the database connection.
type MySQLConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	MaxOpenConns int    `yaml:"maxOpenConns"`
	MaxIdleConns int    `yaml:"maxIdleConns"`
	// LocName is the IANA timezone used to interpret DATETIME columns. It must
	// match the MySQL server timezone (the docker container runs Asia/Shanghai).
	LocName string `yaml:"locName"`
}

// RedisConfig describes the Redis connection.
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"poolSize"`
}

// KafkaConfig describes the Kafka brokers and topic names (replacing RabbitMQ).
type KafkaConfig struct {
	Brokers          []string `yaml:"brokers"`
	TopicTeamSuccess string   `yaml:"topicTeamSuccess"`
	TopicTeamRefund  string   `yaml:"topicTeamRefund"`
	GroupID          string   `yaml:"groupId"`
	// ClientID prefixes generated client ids.
	ClientID string `yaml:"clientId"`
}

// DCCConfig holds the dynamic-config-center pub/sub topic and default values.
type DCCConfig struct {
	Topic    string            `yaml:"topic"`
	Defaults map[string]string `yaml:"defaults"`
}

// NotifyConfig holds the scheduled notify-job parameters.
type NotifyConfig struct {
	// JobInterval is how often the notify job runs (replaces the legacy cron
	// "0 1/20 * * * *"). Default 20m.
	JobInterval time.Duration `yaml:"jobInterval"`
	// JobLockKey is the distributed lock guarding the job across instances.
	JobLockKey string `yaml:"jobLockKey"`
	// HTTPLockWait and HTTPLockLease bound the per-task notify lock.
	HTTPLockWait  time.Duration `yaml:"httpLockWait"`
	HTTPLockLease time.Duration `yaml:"httpLockLease"`
	// CallbackTimeout bounds the outbound HTTP callback.
	CallbackTimeout time.Duration `yaml:"callbackTimeout"`
	// MaxRetry is the notify_count threshold after which a task is marked failed
	// instead of retried (legacy used > 4).
	MaxRetry int `yaml:"maxRetry"`
}

// LogConfig controls log verbosity.
type LogConfig struct {
	Level string `yaml:"level"`
}

// Load reads a YAML config file and applies defaults for any unset fields.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.RPC.Timeout <= 0 {
		c.RPC.Timeout = 5 * time.Second
	}
	if c.MySQL.MaxOpenConns <= 0 {
		c.MySQL.MaxOpenConns = 15
	}
	if c.MySQL.MaxIdleConns <= 0 {
		c.MySQL.MaxIdleConns = 5
	}
	if c.MySQL.LocName == "" {
		c.MySQL.LocName = "Asia/Shanghai"
	}
	if c.Redis.PoolSize <= 0 {
		c.Redis.PoolSize = 10
	}
	if c.Kafka.GroupID == "" {
		c.Kafka.GroupID = "be261028"
	}
	if c.Kafka.ClientID == "" {
		c.Kafka.ClientID = "be261028"
	}
	if c.Kafka.TopicTeamSuccess == "" {
		c.Kafka.TopicTeamSuccess = "be261028_team_success"
	}
	if c.Kafka.TopicTeamRefund == "" {
		c.Kafka.TopicTeamRefund = "be261028_team_refund"
	}
	if c.DCC.Topic == "" {
		c.DCC.Topic = "be261028_dynamic_config_center"
	}
	if c.Notify.JobInterval <= 0 {
		c.Notify.JobInterval = 20 * time.Minute
	}
	if c.Notify.JobLockKey == "" {
		c.Notify.JobLockKey = "be261028_notify_job_exec"
	}
	if c.Notify.HTTPLockWait <= 0 {
		c.Notify.HTTPLockWait = 3 * time.Second
	}
	if c.Notify.HTTPLockLease <= 0 {
		c.Notify.HTTPLockLease = 30 * time.Second
	}
	if c.Notify.CallbackTimeout <= 0 {
		c.Notify.CallbackTimeout = 10 * time.Second
	}
	if c.Notify.MaxRetry <= 0 {
		c.Notify.MaxRetry = 4
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
}
