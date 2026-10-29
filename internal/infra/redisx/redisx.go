package redisx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hangtiancheng/be261028/internal/config"
	"github.com/hangtiancheng/be261028/internal/types"
	"github.com/redis/go-redis/v9"
)

// RedisService wraps a go-redis client with the operations the legacy
// RedissonService provided: JSON value caching, atomic counters, SETNX locks,
// bitmaps and a reentrant-free distributed lock.
type RedisService struct {
	client *redis.Client
}

// New builds a RedisService and verifies connectivity.
func New(cfg config.RedisConfig) (*RedisService, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return &RedisService{client: client}, nil
}

// Client exposes the underlying client for pub/sub and advanced use.
func (r *RedisService) Client() *redis.Client { return r.client }

// Close releases the client.
func (r *RedisService) Close() error { return r.client.Close() }

// SetValue stores value as JSON with no expiry.
func (r *RedisService) SetValue(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, 0).Err()
}

// SetValueWithTTL stores value as JSON with an expiry.
func (r *RedisService) SetValueWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, ttl).Err()
}

// GetValue reads a JSON value into out. It reports found=false when the key does
// not exist. A present-but-corrupt value is surfaced as an error.
func (r *RedisService) GetValue(ctx context.Context, key string, out any) (bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return false, err
		}
	}
	return true, nil
}

// Incr atomically increments key by one and returns the new value.
func (r *RedisService) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// IncrBy atomically adds delta to key and returns the new value.
func (r *RedisService) IncrBy(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, key, delta).Result()
}

// Decr atomically decrements key by one and returns the new value. It is used to
// roll back a speculative Incr when a subsequent check rejects the operation.
func (r *RedisService) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

// Expire sets a TTL on an existing key, returning whether the key existed. A
// non-positive ttl is ignored. It is used to bound the lifetime of counters that
// the legacy implementation left to accumulate forever.
func (r *RedisService) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		return false, nil
	}
	return r.client.Expire(ctx, key, ttl).Result()
}

// GetAtomicLong reads a counter value, returning 0 when the key is absent.
func (r *RedisService) GetAtomicLong(ctx context.Context, key string) (int64, error) {
	v, err := r.client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return v, err
}

// SetNx sets key to a sentinel with a TTL only if it does not exist, returning
// whether the value was set. It mirrors RedissonService.setNx(key, expired, unit).
func (r *RedisService) SetNx(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, key, "lock", ttl).Result()
}

// Remove deletes key.
func (r *RedisService) Remove(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// IsExists reports whether key exists.
func (r *RedisService) IsExists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// BitSetExists reports whether the bitmap key exists.
func (r *RedisService) BitSetExists(ctx context.Context, key string) (bool, error) {
	return r.IsExists(ctx, key)
}

// BitSetGet reads a single bit.
func (r *RedisService) BitSetGet(ctx context.Context, key string, offset int64) (bool, error) {
	v, err := r.client.GetBit(ctx, key, offset).Result()
	return v == 1, err
}

// BitSetSet writes a single bit.
func (r *RedisService) BitSetSet(ctx context.Context, key string, offset int64, value bool) error {
	bit := 0
	if value {
		bit = 1
	}
	return r.client.SetBit(ctx, key, offset, bit).Err()
}

// IndexFromUserId returns the bitmap offset for a user id (MD5 mod MaxInt32).
func (r *RedisService) IndexFromUserId(userID string) int64 {
	return types.IndexFromUserId(userID)
}

// DistLock is a single-owner distributed lock backed by SET NX PX plus a Lua
// compare-and-delete release. It replaces the Redisson RLock usage for the notify
// job and per-task callback locks.
type DistLock struct {
	r     *RedisService
	key   string
	token string
}

// NewLock creates a lock handle for key. The token is generated lazily per handle.
func (r *RedisService) NewLock(key string) *DistLock {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// Extremely unlikely; fall back to a time-based token.
		return &DistLock{r: r, key: key, token: fmt.Sprintf("%d", time.Now().UnixNano())}
	}
	return &DistLock{r: r, key: key, token: hex.EncodeToString(buf)}
}

// TryLock attempts to acquire the lock, retrying until wait elapses. A non-positive
// lease defaults to 30s. It reports whether the lock was acquired.
func (l *DistLock) TryLock(ctx context.Context, wait, lease time.Duration) (bool, error) {
	if lease <= 0 {
		lease = 30 * time.Second
	}
	deadline := time.Now().Add(wait)
	for {
		ok, err := l.r.client.SetNX(ctx, l.key, l.token, lease).Result()
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// unlockScript deletes the key only when the stored token matches.
var unlockScript = redis.NewScript(
	`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`,
)

// Unlock releases the lock if it is still held by this handle.
func (l *DistLock) Unlock(ctx context.Context) error {
	_, err := unlockScript.Run(ctx, l.r.client, []string{l.key}, l.token).Result()
	if err == redis.Nil {
		return nil
	}
	return err
}
