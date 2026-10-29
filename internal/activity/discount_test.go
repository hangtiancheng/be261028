package activity

import (
	"context"
	"testing"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/types"
	"github.com/shopspring/decimal"
)

// baseDiscount returns a GroupBuyDiscount of BASE type (no crowd-tag filtering),
// so the calculators can be exercised without a repository.
func baseDiscount(plan, expr string) *api.GroupBuyDiscount {
	return &api.GroupBuyDiscount{
		DiscountType: types.DiscountTypeBase,
		MarketPlan:   plan,
		MarketExpr:   expr,
	}
}

func TestDiscountRegistryPlans(t *testing.T) {
	reg := NewDiscountRegistry(nil)
	for _, plan := range []string{"ZJ", "MJ", "N", "ZK"} {
		if _, ok := reg.Get(plan); !ok {
			t.Errorf("expected calculator for plan %s", plan)
		}
	}
	if _, ok := reg.Get("UNKNOWN"); ok {
		t.Errorf("unexpected calculator for UNKNOWN plan")
	}
}

func TestZJDirectReduction(t *testing.T) {
	reg := NewDiscountRegistry(nil)
	calc, _ := reg.Get("ZJ")
	ctx := context.Background()

	// 100 - 20 = 80
	got, err := calc.Calculate(ctx, "u1", decimal.NewFromInt(100), baseDiscount("ZJ", "20"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(decimal.NewFromInt(80)) {
		t.Errorf("ZJ 100-20 = %s, want 80", got)
	}

	// Reduction exceeding the price floors at 0.01.
	got, err = calc.Calculate(ctx, "u1", decimal.NewFromInt(10), baseDiscount("ZJ", "20"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(types.MinPayPrice) {
		t.Errorf("ZJ 10-20 = %s, want 0.01", got)
	}
}

func TestMJThresholdReduction(t *testing.T) {
	reg := NewDiscountRegistry(nil)
	calc, _ := reg.Get("MJ")
	ctx := context.Background()

	// 200 >= 100 threshold -> 200 - 20 = 180
	got, err := calc.Calculate(ctx, "u1", decimal.NewFromInt(200), baseDiscount("MJ", "100,20"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(decimal.NewFromInt(180)) {
		t.Errorf("MJ 200 (100,20) = %s, want 180", got)
	}

	// 50 < 100 threshold -> original price unchanged
	got, err = calc.Calculate(ctx, "u1", decimal.NewFromInt(50), baseDiscount("MJ", "100,20"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(decimal.NewFromInt(50)) {
		t.Errorf("MJ 50 (100,20) = %s, want 50", got)
	}
}

func TestNFixedPrice(t *testing.T) {
	reg := NewDiscountRegistry(nil)
	calc, _ := reg.Get("N")
	ctx := context.Background()

	got, err := calc.Calculate(ctx, "u1", decimal.NewFromInt(200), baseDiscount("N", "80"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(decimal.NewFromInt(80)) {
		t.Errorf("N 200 (80) = %s, want 80", got)
	}
}

func TestZKPercentage(t *testing.T) {
	reg := NewDiscountRegistry(nil)
	calc, _ := reg.Get("ZK")
	ctx := context.Background()

	// 200 * 0.8 = 160
	got, err := calc.Calculate(ctx, "u1", decimal.NewFromInt(200), baseDiscount("ZK", "0.8"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(decimal.NewFromInt(160)) {
		t.Errorf("ZK 200*0.8 = %s, want 160", got)
	}
}

func TestGroupBuyActivityDiscountVisibility(t *testing.T) {
	// tag_scope 语义：包含 "1" -> 可见限制；包含 "2" -> 参与限制。
	// 缺陷修复后按"包含"判断（旧版只看固定下标，"2" 单独出现或 "2,1" 顺序颠倒都会误判）。
	cases := []struct {
		tagScope string
		visible  bool
		enable   bool
	}{
		{"", true, true},
		{"1", false, true},
		{"2", true, false},
		{"1,2", false, false},
		{"2,1", false, false},
		{"0,2", true, false},
	}
	for _, c := range cases {
		vo := &api.GroupBuyActivityDiscountVO{TagScope: c.tagScope}
		if got := vo.IsVisible(); got != c.visible {
			t.Errorf("IsVisible(tagScope=%q) = %v, want %v", c.tagScope, got, c.visible)
		}
		if got := vo.IsEnable(); got != c.enable {
			t.Errorf("IsEnable(tagScope=%q) = %v, want %v", c.tagScope, got, c.enable)
		}
	}
}
