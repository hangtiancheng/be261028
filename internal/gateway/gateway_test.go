package gateway

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hangtiancheng/be261028/internal/api"
	yukinohttp "github.com/hangtiancheng/yukino.go/libs/yukino_http"
)

// TestBodyLimitBoundsRequestSize pins the hardening fix: yukino_http's BindJSON
// has no built-in size limit, so the gateway wraps bodies in an
// http.MaxBytesReader. Oversized bodies must fail at bind time; normal bodies
// must be unaffected.
func TestBodyLimitBoundsRequestSize(t *testing.T) {
	app := yukinohttp.New()
	g := &Gateway{}
	app.Use(g.bodyLimit())
	app.Post("/echo", func(ctx *yukinohttp.Context, next func()) {
		var payload struct {
			Data string `json:"data"`
		}
		if err := ctx.BindJSON(&payload); err != nil {
			ctx.JSON(&api.Response{Code: "ERROR", Info: "body rejected"})
			return
		}
		ctx.JSON(&api.Response{Code: "0000", Info: "ok"})
	})

	// An oversized body must be rejected at bind time.
	large := strings.Repeat("a", maxBodyBytes+1024)
	req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte(`{"data":"`+large+`"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "body rejected") {
		t.Fatalf("oversized body was not rejected: %s", rec.Body.String())
	}

	// A normal body must still bind successfully.
	req = httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte(`{"data":"hello"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `"0000"`) {
		t.Fatalf("normal body was rejected: %s", rec.Body.String())
	}
}

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

	// Let entries expire and force a sweep by zeroing lastCleanup.
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
	// Exceeding the entry cap triggers a sweep; expired entries must be removed
	// so the map size stays bounded.
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
