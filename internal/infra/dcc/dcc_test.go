package dcc

import "testing"

// TestIsCutRangeBoundaries pins the defect fix: the legacy <= comparison let
// cutRange=0 admit bucket-0 users and made every setting off by one percent.
func TestIsCutRangeBoundaries(t *testing.T) {
	users := []string{"u1", "xiaofuge", "liergou", "xfg01", "user-42", "test", "xifeiya"}

	// cutRange=0 必须拦截所有用户。
	d := New("", map[string]string{KeyCutRange: "0"})
	for _, u := range users {
		if d.IsCutRange(u) {
			t.Errorf("cutRange=0 admitted %s, want reject", u)
		}
	}

	// cutRange=100 必须放行所有用户。
	d = New("", map[string]string{KeyCutRange: "100"})
	for _, u := range users {
		if !d.IsCutRange(u) {
			t.Errorf("cutRange=100 rejected %s, want admit", u)
		}
	}

	// 非数字配置回落到默认 100（与旧版一致）。
	d = New("", map[string]string{KeyCutRange: "abc"})
	for _, u := range users {
		if !d.IsCutRange(u) {
			t.Errorf("cutRange=abc rejected %s, want admit", u)
		}
	}
}

// TestIsCutRangePartial verifies an intermediate percentage admits a strict
// subset (never everyone, never no-one across a spread of user ids).
func TestIsCutRangePartial(t *testing.T) {
	d := New("", map[string]string{KeyCutRange: "50"})
	admitted, total := 0, 200
	for i := 0; i < total; i++ {
		if d.IsCutRange(userIdOf(i)) {
			admitted++
		}
	}
	if admitted == 0 || admitted == total {
		t.Fatalf("cutRange=50 admitted %d/%d, want a strict subset", admitted, total)
	}
}

func TestIsSCBlackIntercept(t *testing.T) {
	d := New("", nil) // 默认黑名单 s02c02
	if !d.IsSCBlackIntercept("s02", "c02") {
		t.Error("s02c02 should be intercepted by default")
	}
	if d.IsSCBlackIntercept("s01", "c01") {
		t.Error("s01c01 should not be intercepted")
	}
	d.Update(KeySCBlacklist, "s01c01,s03c03")
	if d.IsSCBlackIntercept("s02", "c02") {
		t.Error("s02c02 should not be intercepted after update")
	}
	if !d.IsSCBlackIntercept("s03", "c03") {
		t.Error("s03c03 should be intercepted after update")
	}
}

func TestDefaultsAndOverrides(t *testing.T) {
	d := New("", map[string]string{KeyDowngradeSwitch: "1"})
	if !d.IsDowngradeSwitch() {
		t.Error("downgradeSwitch=1 should be on")
	}
	// cacheSwitch 默认 "0"（开启缓存，与旧版语义一致）。
	if !d.IsCacheOpenSwitch() {
		t.Error("default cacheSwitch should be open")
	}
	d.Update(KeyCacheSwitch, "1")
	if d.IsCacheOpenSwitch() {
		t.Error("cacheSwitch=1 should close the cache")
	}
}

func userIdOf(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "user-0"
	}
	var buf []byte
	for n := i; n > 0; n /= 10 {
		buf = append([]byte{digits[n%10]}, buf...)
	}
	return "user-" + string(buf)
}
