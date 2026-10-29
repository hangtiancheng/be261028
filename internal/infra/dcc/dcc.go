package dcc

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/types"
)

// Config keys managed by the dynamic config center. They mirror the legacy
// @DCCValue fields in DCCService.
const (
	KeyDowngradeSwitch = "downgradeSwitch"
	KeyCutRange        = "cutRange"
	KeySCBlacklist     = "scBlacklist"
	KeyCacheSwitch     = "cacheSwitch"
)

// DefaultValues mirrors the legacy @DCCValue defaults.
var DefaultValues = map[string]string{
	KeyDowngradeSwitch: "0",
	KeyCutRange:        "100",
	KeySCBlacklist:     "s02c02",
	KeyCacheSwitch:     "0",
}

// AttributeVO mirrors cn.bugstack.wrench.dynamic.config.center AttributeVO, the
// message published on the config topic.
type AttributeVO struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DCC is the dynamic config center. It holds the current config values in memory,
// subscribes to a Redis pub/sub topic for live updates, and exposes the same
// derived predicates the legacy DCCService provided.
type DCC struct {
	mu     sync.RWMutex
	values map[string]string
	topic  string
}

// New builds a DCC with the given topic, seeding defaults then applying overrides.
func New(topic string, overrides map[string]string) *DCC {
	values := make(map[string]string, len(DefaultValues))
	for k, v := range DefaultValues {
		values[k] = v
	}
	for k, v := range overrides {
		values[k] = v
	}
	if topic == "" {
		topic = "be261028_dynamic_config_center"
	}
	return &DCC{values: values, topic: topic}
}

// Topic returns the pub/sub topic name.
func (d *DCC) Topic() string { return d.topic }

// Get returns the current value for key (empty when unset).
func (d *DCC) Get(key string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.values[key]
}

// Update sets a config value in memory.
func (d *DCC) Update(key, value string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.values[key] = value
}

// IsDowngradeSwitch mirrors DCCService.isDowngradeSwitch ("1" == downgradeSwitch).
func (d *DCC) IsDowngradeSwitch() bool { return d.Get(KeyDowngradeSwitch) == "1" }

// IsCacheOpenSwitch mirrors DCCService.isCacheOpenSwitch ("0" == cacheSwitch).
func (d *DCC) IsCacheOpenSwitch() bool { return d.Get(KeyCacheSwitch) == "0" }

// IsCutRange mirrors DCCService.isCutRange: bucket the user by the absolute Java
// string hash mod 100 and compare against the configured percentage.
//
// Defect fix: the legacy comparison used <=, so cutRange=N actually admitted
// N+1 percent of users and cutRange=0 could not cut all traffic (bucket 0 still
// passed). With < the config is an exact percentage: 0 admits nobody, 100 admits
// everyone.
func (d *DCC) IsCutRange(userID string) bool {
	cutRange := d.Get(KeyCutRange)
	n, err := strconv.Atoi(strings.TrimSpace(cutRange))
	if err != nil {
		// Fail open like the default (100) when the value is not a number.
		n = 100
	}
	hash := types.JavaAbs(types.JavaStringHashCode(userID))
	lastTwo := int(hash % 100)
	return lastTwo < n
}

// IsSCBlackIntercept mirrors DCCService.isSCBlackIntercept: true when source+channel
// appears in the comma-separated blacklist.
func (d *DCC) IsSCBlackIntercept(source, channel string) bool {
	blacklist := d.Get(KeySCBlacklist)
	if blacklist == "" {
		return false
	}
	target := source + channel
	for _, item := range strings.Split(blacklist, types.Split) {
		if item == target {
			return true
		}
	}
	return false
}

// Publish sends a config change to the topic so every subscribed instance updates.
func (d *DCC) Publish(ctx context.Context, r *redisx.RedisService, key, value string) error {
	payload, err := json.Marshal(AttributeVO{Key: key, Value: value})
	if err != nil {
		return err
	}
	return r.Client().Publish(ctx, d.topic, payload).Err()
}

// Subscribe listens for config changes and applies them. It blocks until ctx is
// cancelled and is intended to run in its own goroutine.
func (d *DCC) Subscribe(ctx context.Context, r *redisx.RedisService) {
	pubsub := r.Client().Subscribe(ctx, d.topic)
	defer pubsub.Close()
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var attr AttributeVO
			if err := json.Unmarshal([]byte(msg.Payload), &attr); err != nil {
				log.Printf("[dcc] 无法解析配置消息: %v", err)
				continue
			}
			d.Update(attr.Key, attr.Value)
			log.Printf("[dcc] 动态配置更新 key:%s value:%s", attr.Key, attr.Value)
		}
	}
}
