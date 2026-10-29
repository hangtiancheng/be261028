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
	if got := DifferenceDateTime2Str(base, base.Add(-time.Second)); got != "Ended" {
		t.Errorf("ended countdown = %s, want Ended", got)
	}

	// Missing value.
	if got := DifferenceDateTime2Str(time.Time{}, base); got != "Invalid time" {
		t.Errorf("invalid countdown = %s, want Invalid time", got)
	}

	// Regression guard for the defect fix: countdowns longer than 24h must show
	// the day component (the legacy %24 dropped whole days).
	if got := DifferenceDateTime2Str(base, base.Add(25*time.Hour+2*time.Minute+3*time.Second)); got != "1d 01:02:03" {
		t.Errorf("countdown = %s, want 1d 01:02:03", got)
	}
	if got := DifferenceDateTime2Str(base, base.Add(72*time.Hour)); got != "3d 00:00:00" {
		t.Errorf("countdown = %s, want 3d 00:00:00", got)
	}
	// Exactly 24 hours = 1 day (the legacy version mis-rendered it as 00:00:00, looking like "ended").
	if got := DifferenceDateTime2Str(base, base.Add(24*time.Hour)); got != "1d 00:00:00" {
		t.Errorf("countdown = %s, want 1d 00:00:00", got)
	}
}
