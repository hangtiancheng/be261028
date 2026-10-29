package api

import (
	"testing"
	"time"
)

func TestDifferenceDateTime2Str(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// 1 hour 2 minutes 3 seconds remaining.
	got := DifferenceDateTime2Str(base, base.Add(time.Hour+2*time.Minute+3*time.Second))
	if got != "01:02:03" {
		t.Errorf("countdown = %s, want 01:02:03", got)
	}

	// Ended (end before start).
	if got := DifferenceDateTime2Str(base, base.Add(-time.Second)); got != "已结束" {
		t.Errorf("ended countdown = %s, want 已结束", got)
	}

	// Missing value.
	if got := DifferenceDateTime2Str(time.Time{}, base); got != "无效的时间" {
		t.Errorf("invalid countdown = %s, want 无效的时间", got)
	}

	// 缺陷修复回归：超过 24 小时的倒计时必须显示天数（旧版 %24 会丢掉天）。
	if got := DifferenceDateTime2Str(base, base.Add(25*time.Hour+2*time.Minute+3*time.Second)); got != "1天01:02:03" {
		t.Errorf("countdown = %s, want 1天01:02:03", got)
	}
	if got := DifferenceDateTime2Str(base, base.Add(72*time.Hour)); got != "3天00:00:00" {
		t.Errorf("countdown = %s, want 3天00:00:00", got)
	}
	// 恰好 24 小时 = 1 天（旧版会误显示为 00:00:00，形似已结束）。
	if got := DifferenceDateTime2Str(base, base.Add(24*time.Hour)); got != "1天00:00:00" {
		t.Errorf("countdown = %s, want 1天00:00:00", got)
	}
}
