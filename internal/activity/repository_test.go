package activity

import (
	"testing"

	"github.com/hangtiancheng/be261028/internal/dao"
)

// TestDedupeBriefsByTeam covers the random team-list defect fix: several orders of
// the same team must collapse to one candidate slot, otherwise duplicates consume
// the sample and the homepage can show no teams while joinable teams exist.
func TestDedupeBriefsByTeam(t *testing.T) {
	briefs := []*dao.OrderListBrief{
		{UserId: "a", TeamId: "t1"},
		{UserId: "b", TeamId: "t2"},
		{UserId: "c", TeamId: "t1"},
		{UserId: "d", TeamId: "t3"},
		{UserId: "e", TeamId: "t2"},
	}
	got := dedupeBriefsByTeam(briefs)
	if len(got) != 3 {
		t.Fatalf("deduped len = %d, want 3", len(got))
	}
	want := []string{"t1", "t2", "t3"}
	for i, b := range got {
		if b.TeamId != want[i] {
			t.Fatalf("deduped[%d] team = %s, want %s (order preserved)", i, b.TeamId, want[i])
		}
	}
	if got[0].UserId != "a" {
		t.Fatalf("kept brief user = %s, want first occurrence a", got[0].UserId)
	}

	if out := dedupeBriefsByTeam(nil); len(out) != 0 {
		t.Fatalf("nil input returned %d briefs", len(out))
	}
	one := []*dao.OrderListBrief{{UserId: "a", TeamId: "t1"}}
	if out := dedupeBriefsByTeam(one); len(out) != 1 || out[0] != one[0] {
		t.Fatal("single brief should pass through unchanged")
	}
}
