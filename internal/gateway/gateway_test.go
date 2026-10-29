package gateway

import (
	"fmt"
	"testing"
	"time"
)

func TestRateLimiterBlocksWithinInterval(t *testing.T) {
	rl := NewRateLimiter(time.Second)
	if !rl.Allow("u1") {
		t.Fatal("first call should pass")
	}
	if rl.Allow("u1") {
		t.Fatal("second call within the interval should be blocked")
	}
	if !rl.Allow("u2") {
		t.Fatal("different key should pass")
	}
	if !rl.Allow("") {
		t.Fatal("empty key should always pass")
	}
}

// TestRateLimiterEvictsExpiredEntries pins the defect fix: the legacy limiter
// kept an ever-growing map. Expired entries must be swept so memory is bounded.
func TestRateLimiterEvictsExpiredEntries(t *testing.T) {
	rl := NewRateLimiter(10 * time.Millisecond)
	for i := 0; i < 100; i++ {
		if !rl.Allow(fmt.Sprintf("user-%d", i)) {
			t.Fatalf("first call for user-%d should be allowed", i)
		}
	}
	rl.mu.Lock()
	size := len(rl.last)
	rl.mu.Unlock()
	if size != 100 {
		t.Fatalf("map size = %d, want 100", size)
	}

	// 等条目过期并强制触发清扫（lastCleanup 置零）。
	time.Sleep(15 * time.Millisecond)
	rl.mu.Lock()
	rl.lastCleanup = time.Time{}
	rl.mu.Unlock()

	if !rl.Allow("user-0") {
		t.Fatal("expired key should be allowed again")
	}
	rl.mu.Lock()
	size = len(rl.last)
	rl.mu.Unlock()
	if size != 1 {
		t.Errorf("map size after sweep = %d, want 1 (only the refreshed key)", size)
	}
}

// TestRateLimiterEvictsAtCap verifies the size-cap path also triggers a sweep.
func TestRateLimiterEvictsAtCap(t *testing.T) {
	rl := NewRateLimiter(time.Millisecond)
	// 超过容量上限触发清扫；过期条目应被移除，容量有界。
	for round := 0; round < 3; round++ {
		for i := 0; i < limiterMaxEntries+100; i++ {
			rl.Allow(fmt.Sprintf("round-%d-user-%d", round, i))
		}
		time.Sleep(2 * time.Millisecond)
	}
	rl.mu.Lock()
	size := len(rl.last)
	rl.mu.Unlock()
	if size > limiterMaxEntries+100 {
		t.Errorf("map size = %d, expected bounded by cap + last round", size)
	}
}
