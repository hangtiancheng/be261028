package activity

import (
	"context"
	"fmt"
	"strings"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/types"
	"github.com/shopspring/decimal"
)

// DiscountCalculator computes the payable price for a discount plan, mirroring
// IDiscountCalculateService.
type DiscountCalculator interface {
	Calculate(ctx context.Context, userId string, originalPrice decimal.Decimal, discount *api.GroupBuyDiscount) (decimal.Decimal, error)
}

// DiscountRegistry maps market plan codes (ZJ/MJ/N/ZK) to calculators, mirroring
// the Spring bean-name keyed Map<String, IDiscountCalculateService>.
type DiscountRegistry struct {
	calculators map[string]DiscountCalculator
}

// NewDiscountRegistry builds the registry with the four legacy plans.
func NewDiscountRegistry(repo *ActivityRepository) *DiscountRegistry {
	base := baseCalculator{repo: repo}
	return &DiscountRegistry{calculators: map[string]DiscountCalculator{
		"ZJ": &zjCalculator{base},
		"MJ": &mjCalculator{base},
		"N":  &nCalculator{base},
		"ZK": &zkCalculator{base},
	}}
}

// Get returns the calculator for a plan and whether it exists.
func (r *DiscountRegistry) Get(plan string) (DiscountCalculator, bool) {
	c, ok := r.calculators[plan]
	return c, ok
}

// Plans lists the supported plan codes (for error messages).
func (r *DiscountRegistry) Plans() []string {
	out := make([]string, 0, len(r.calculators))
	for k := range r.calculators {
		out = append(out, k)
	}
	return out
}

// baseCalculator provides the shared crowd-tag filter from
// AbstractDiscountCalculateService.calculate.
type baseCalculator struct {
	repo *ActivityRepository
}

func (b baseCalculator) calculate(ctx context.Context, userId string, originalPrice decimal.Decimal, discount *api.GroupBuyDiscount, doCalculate func(decimal.Decimal, *api.GroupBuyDiscount) (decimal.Decimal, error)) (decimal.Decimal, error) {
	// 1. Crowd-tag filter: the discount is restricted to a tagged crowd.
	if discount.DiscountType == types.DiscountTypeTag {
		inRange, err := b.repo.IsTagCrowdRange(ctx, discount.TagId, userId)
		if err != nil {
			return decimal.Zero, err
		}
		if !inRange {
			return originalPrice, nil
		}
	}
	// 2. Discount calculation.
	return doCalculate(originalPrice, discount)
}

// zjCalculator mirrors ZJCalculateService (direct reduction).
type zjCalculator struct{ baseCalculator }

func (c *zjCalculator) Calculate(ctx context.Context, userId string, originalPrice decimal.Decimal, discount *api.GroupBuyDiscount) (decimal.Decimal, error) {
	return c.calculate(ctx, userId, originalPrice, discount, func(op decimal.Decimal, d *api.GroupBuyDiscount) (decimal.Decimal, error) {
		expr, err := types.ParseMoney(strings.TrimSpace(d.MarketExpr))
		if err != nil {
			return decimal.Zero, fmt.Errorf("ZJ expression parse failed %q: %w", d.MarketExpr, err)
		}
		pay := op.Sub(expr)
		if pay.LessThanOrEqual(decimal.Zero) {
			return types.MinPayPrice, nil
		}
		return pay, nil
	})
}

// mjCalculator mirrors MJCalculateService (threshold-based reduction).
type mjCalculator struct{ baseCalculator }

func (c *mjCalculator) Calculate(ctx context.Context, userId string, originalPrice decimal.Decimal, discount *api.GroupBuyDiscount) (decimal.Decimal, error) {
	return c.calculate(ctx, userId, originalPrice, discount, func(op decimal.Decimal, d *api.GroupBuyDiscount) (decimal.Decimal, error) {
		parts := types.JavaSplit(d.MarketExpr, types.Split)
		if len(parts) < 2 {
			return decimal.Zero, fmt.Errorf("MJ expression format invalid %q", d.MarketExpr)
		}
		threshold, err := types.ParseMoney(strings.TrimSpace(parts[0]))
		if err != nil {
			return decimal.Zero, fmt.Errorf("MJ threshold parse failed %q: %w", parts[0], err)
		}
		reduction, err := types.ParseMoney(strings.TrimSpace(parts[1]))
		if err != nil {
			return decimal.Zero, fmt.Errorf("MJ reduction parse failed %q: %w", parts[1], err)
		}
		// The minimum-threshold condition is not met; charge the original price.
		if op.LessThan(threshold) {
			return op, nil
		}
		pay := op.Sub(reduction)
		if pay.LessThanOrEqual(decimal.Zero) {
			return types.MinPayPrice, nil
		}
		return pay, nil
	})
}

// nCalculator mirrors NCalculateService (fixed-price purchase): the expression is
// the final price.
type nCalculator struct{ baseCalculator }

func (c *nCalculator) Calculate(ctx context.Context, userId string, originalPrice decimal.Decimal, discount *api.GroupBuyDiscount) (decimal.Decimal, error) {
	return c.calculate(ctx, userId, originalPrice, discount, func(op decimal.Decimal, d *api.GroupBuyDiscount) (decimal.Decimal, error) {
		pay, err := types.ParseMoney(strings.TrimSpace(d.MarketExpr))
		if err != nil {
			return decimal.Zero, fmt.Errorf("N expression parse failed %q: %w", d.MarketExpr, err)
		}
		return pay, nil
	})
}

// zkCalculator mirrors ZKCalculateService (percentage discount).
type zkCalculator struct{ baseCalculator }

func (c *zkCalculator) Calculate(ctx context.Context, userId string, originalPrice decimal.Decimal, discount *api.GroupBuyDiscount) (decimal.Decimal, error) {
	return c.calculate(ctx, userId, originalPrice, discount, func(op decimal.Decimal, d *api.GroupBuyDiscount) (decimal.Decimal, error) {
		rate, err := types.ParseMoney(strings.TrimSpace(d.MarketExpr))
		if err != nil {
			return decimal.Zero, fmt.Errorf("ZK expression parse failed %q: %w", d.MarketExpr, err)
		}
		pay := op.Mul(rate)
		if pay.LessThanOrEqual(decimal.Zero) {
			return types.MinPayPrice, nil
		}
		return pay, nil
	})
}
