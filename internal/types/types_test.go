package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestJavaStringHashCode(t *testing.T) {
	// Reference values computed from java.lang.String.hashCode().
	cases := map[string]int32{
		"":            0,
		"a":           97,
		"abc":         96354,
		"Aa":          2112, // classic Java hash collision with "BB"
		"BB":          2112,
		"xfg01":       113961722,
		"hello world": 1794106052,
		// Overflow case: exercises 32-bit wraparound to Integer.MIN_VALUE.
		"polygenelubricants": -2147483648,
	}
	for s, want := range cases {
		if got := JavaStringHashCode(s); got != want {
			t.Errorf("JavaStringHashCode(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestJavaAbs(t *testing.T) {
	if got := JavaAbs(-5); got != 5 {
		t.Errorf("JavaAbs(-5) = %d, want 5", got)
	}
	if got := JavaAbs(5); got != 5 {
		t.Errorf("JavaAbs(5) = %d, want 5", got)
	}
}

func TestIndexFromUserId(t *testing.T) {
	idx := IndexFromUserId("xfg01")
	if idx < 0 || idx > 2147483646 {
		t.Errorf("IndexFromUserId out of range: %d", idx)
	}
	// Deterministic.
	if idx != IndexFromUserId("xfg01") {
		t.Errorf("IndexFromUserId not deterministic")
	}
}

func TestJavaSplit(t *testing.T) {
	cases := []struct {
		in   string
		sep  string
		want int
	}{
		{"1,2", ",", 2},
		{"100,20", ",", 2},
		{"1,", ",", 1}, // trailing empty removed, matching Java split
		{"1", ",", 1},
	}
	for _, c := range cases {
		if got := len(JavaSplit(c.in, c.sep)); got != c.want {
			t.Errorf("JavaSplit(%q) len = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestIsBlank(t *testing.T) {
	if !IsBlank("   ") || !IsBlank("") {
		t.Errorf("IsBlank should be true for empty/whitespace")
	}
	if IsBlank("a") {
		t.Errorf("IsBlank should be false for non-empty")
	}
}

func TestGetRefundStrategy(t *testing.T) {
	cases := []struct {
		team  GroupBuyOrderStatus
		order TradeOrderStatus
		want  RefundType
	}{
		{GroupBuyOrderProgress, TradeOrderCreate, RefundTypeUnpaidUnlock},
		{GroupBuyOrderProgress, TradeOrderComplete, RefundTypePaidUnformed},
		{GroupBuyOrderComplete, TradeOrderComplete, RefundTypePaidFormed},
	}
	for _, c := range cases {
		got, err := GetRefundStrategy(c.team, c.order)
		if err != nil {
			t.Fatalf("GetRefundStrategy(%d,%d) unexpected error: %v", c.team, c.order, err)
		}
		if got != c.want {
			t.Errorf("GetRefundStrategy(%d,%d) = %s, want %s", c.team, c.order, got, c.want)
		}
	}
	// Unsupported combination.
	if _, err := GetRefundStrategy(GroupBuyOrderFail, TradeOrderCreate); err == nil {
		t.Errorf("expected error for unsupported refund combination")
	}
}

func TestGroupBuyOrderStatusOfNormalizesRefundState(t *testing.T) {
	// db status 3 (完成含退单) normalises to Complete; the legacy valueOf threw.
	if got := GroupBuyOrderStatusOf(3); got != GroupBuyOrderComplete {
		t.Errorf("GroupBuyOrderStatusOf(3) = %d, want Complete", got)
	}
	if got := GroupBuyOrderStatusOf(99); got != GroupBuyOrderProgress {
		t.Errorf("GroupBuyOrderStatusOf(99) should fall back to Progress")
	}
}

func TestJSONTimeRoundTrip(t *testing.T) {
	cases := []string{
		`"2026-01-01T00:00:00"`,
		`"2026-01-01 00:00:00"`,
		`"2026-01-01T00:00:00Z"`,
		`1767225600000`,
	}
	for _, in := range cases {
		var jt JSONTime
		if err := json.Unmarshal([]byte(in), &jt); err != nil {
			t.Fatalf("unmarshal %s: %v", in, err)
		}
		if jt.IsZero() {
			t.Errorf("unmarshal %s produced zero time", in)
		}
		out, err := json.Marshal(jt)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		// Re-parse the output; it must round-trip to the same instant.
		var jt2 JSONTime
		if err := json.Unmarshal(out, &jt2); err != nil {
			t.Fatalf("re-unmarshal %s: %v", out, err)
		}
		if !jt.ToTime().Equal(jt2.ToTime()) {
			t.Errorf("round trip mismatch: %s -> %s", in, out)
		}
	}
	// null and empty are zero.
	var jt JSONTime
	if err := json.Unmarshal([]byte(`null`), &jt); err != nil || !jt.IsZero() {
		t.Errorf("null should unmarshal to zero time")
	}
}

func TestJSONTimeOutputFormat(t *testing.T) {
	jt := NewJSONTime(time.Date(2026, 1, 2, 3, 4, 5, 0, AppLoc))
	b, _ := json.Marshal(jt)
	if string(b) != `"2026-01-02T03:04:05"` {
		t.Errorf("JSONTime marshal = %s, want naive date-time", string(b))
	}
}

func TestMoneyMarshalTwoDecimals(t *testing.T) {
	m := NewMoney(MustMoney("100.00"))
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "100.00" {
		t.Errorf("Money marshal = %s, want 100.00", string(b))
	}
	// Arithmetic result wrapped back into Money keeps two decimals.
	sum := NewMoney(m.Sub(MustMoney("20")))
	b2, _ := json.Marshal(sum)
	if string(b2) != "80.00" {
		t.Errorf("Money(100-20) marshal = %s, want 80.00", string(b2))
	}
	// Unmarshal accepts a bare number.
	var m2 Money
	if err := json.Unmarshal([]byte("170.00"), &m2); err != nil {
		t.Fatal(err)
	}
	if m2.StringFixed(2) != "170.00" {
		t.Errorf("Money unmarshal = %s, want 170.00", m2.StringFixed(2))
	}
}

func TestAppError(t *testing.T) {
	err := NewAppError(ResponseE0002)
	ae := AsAppError(err)
	if ae == nil || ae.Code != "E0002" {
		t.Errorf("AsAppError did not recover code: %v", err)
	}
	if AsAppError(nil) != nil {
		t.Errorf("AsAppError(nil) should be nil")
	}

	// 缺陷修复回归：被 fmt.Errorf("%w") 包裹的 AppError 也必须能解包出业务码。
	wrapped := fmt.Errorf("settlement failed: %w", NewAppError(ResponseE0002))
	if ae := AsAppError(wrapped); ae == nil || ae.Code != "E0002" {
		t.Errorf("AsAppError did not unwrap wrapped error: %v", wrapped)
	}
	if AsAppError(errors.New("plain error")) != nil {
		t.Errorf("plain error should not match AppError")
	}
}
