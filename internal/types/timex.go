package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // embed the tz database so Asia/Shanghai resolves anywhere
)

// appTZName is the timezone the legacy system used (JDBC serverTimezone and the
// MySQL container TZ are both Asia/Shanghai). Keeping the whole pipeline on this
// zone preserves the original time comparisons.
const appTZName = "Asia/Shanghai"

// AppLoc is the application-wide time location. It falls back to UTC only if the
// embedded tz database cannot provide Asia/Shanghai (should never happen).
var AppLoc = func() *time.Location {
	loc, err := time.LoadLocation(appTZName)
	if err != nil {
		return time.UTC
	}
	return loc
}()

// Now returns the current time in the application timezone.
func Now() time.Time { return time.Now().In(AppLoc) }

// dateTimeLayout is the naive layout used for JSON output, matching the legacy
// REST examples (e.g. "2026-01-01T00:00:00").
const dateTimeLayout = "2006-01-02T15:04:05"

// JSONTime is a time.Time that serialises to the legacy naive date-time format
// and parses a tolerant set of inputs (RFC3339, naive date-time, naive date,
// space-separated date-time, and epoch millis). Naive inputs are interpreted in
// AppLoc so that request times, DB times and "now" share one reference frame.
type JSONTime struct {
	time.Time
}

// NewJSONTime wraps t, converting it into AppLoc.
func NewJSONTime(t time.Time) JSONTime {
	if t.IsZero() {
		return JSONTime{}
	}
	return JSONTime{Time: t.In(AppLoc)}
}

// JSONTimeFrom builds a JSONTime from a *time.Time, tolerating nil.
func JSONTimeFrom(t *time.Time) JSONTime {
	if t == nil {
		return JSONTime{}
	}
	return NewJSONTime(*t)
}

// IsZero reports whether the time is unset.
func (t JSONTime) IsZero() bool { return t.Time.IsZero() }

// MarshalJSON writes the time as a naive AppLoc string, or null when unset.
func (t JSONTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(t.Time.In(AppLoc).Format(dateTimeLayout))), nil
}

// UnmarshalJSON accepts null, empty string, epoch millis (number or numeric
// string), RFC3339, and several naive layouts interpreted in AppLoc.
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" || s == `""` {
		t.Time = time.Time{}
		return nil
	}
	// Strip surrounding quotes if present.
	raw := s
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		raw = s[1 : len(s)-1]
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		t.Time = time.Time{}
		return nil
	}

	// Epoch millis (numeric).
	if millis, err := strconv.ParseInt(raw, 10, 64); err == nil {
		t.Time = time.UnixMilli(millis).In(AppLoc)
		return nil
	}

	// RFC3339 (carries an offset).
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		t.Time = parsed.In(AppLoc)
		return nil
	}

	// Naive layouts interpreted in AppLoc.
	layouts := []string{
		dateTimeLayout,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000",
		"2006-01-02 15:04:05.000",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, raw, AppLoc); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("unparseable time format: %s", raw)
}

// String returns the naive AppLoc representation (or "" when unset).
func (t JSONTime) String() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.In(AppLoc).Format(dateTimeLayout)
}

// MarshalText implements encoding.TextMarshaler.
func (t JSONTime) MarshalText() ([]byte, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return []byte(t.Time.In(AppLoc).Format(dateTimeLayout)), nil
}

// ToTime returns the wrapped time.Time (in AppLoc), or the zero time.
func (t JSONTime) ToTime() time.Time {
	if t.Time.IsZero() {
		return time.Time{}
	}
	return t.Time.In(AppLoc)
}

// Ensure JSONTime satisfies json interfaces.
var (
	_ json.Marshaler   = JSONTime{}
	_ json.Unmarshaler = (*JSONTime)(nil)
)
