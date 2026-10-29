package trade

import (
	"strings"
	"testing"
)

func TestGenerateTeamStockKey(t *testing.T) {
	got := GenerateTeamStockKey(200001, "31449070")
	want := "be261028_team_stock_key_200001_31449070"
	if got != want {
		t.Errorf("GenerateTeamStockKey = %s, want %s", got, want)
	}
	rec := GenerateRecoveryTeamStockKey(200001, "31449070")
	if rec != want+"_recovery" {
		t.Errorf("GenerateRecoveryTeamStockKey = %s, want %s_recovery", rec, want)
	}
}

func TestRandomNumeric(t *testing.T) {
	for _, n := range []int{8, 12} {
		s := randomNumeric(n)
		if len(s) != n {
			t.Fatalf("randomNumeric(%d) length = %d", n, len(s))
		}
		for _, c := range s {
			if c < '0' || c > '9' {
				t.Fatalf("randomNumeric(%d) contains non-digit: %s", n, s)
			}
		}
	}
	if randomNumeric(0) != "" {
		t.Errorf("randomNumeric(0) should be empty")
	}
	// Two calls should (almost certainly) differ.
	if randomNumeric(12) == randomNumeric(12) {
		t.Errorf("randomNumeric produced identical values twice")
	}
}

func TestNotifyTaskLockKey(t *testing.T) {
	n := &NotifyTaskEntity{TeamId: "31449070"}
	if !strings.HasPrefix(n.LockKey(), "be261028_notify_job_lock_key_") {
		t.Errorf("unexpected lock key: %s", n.LockKey())
	}
}
