package trade

import (
	"context"
	"log"

	"github.com/hangtiancheng/be261028/internal/types"
)

// lockDynamicContext mirrors TradeLockRuleFilterFactory.DynamicContext.
type lockDynamicContext struct {
	groupBuyActivity   *GroupBuyActivityEntity
	userTakeOrderCount int
}

// lockFilter is one link of the lock rule chain. Returning a non-nil back entity
// terminates the chain; returning (nil, nil) continues to the next filter.
type lockFilter func(ctx context.Context, cmd *TradeLockRuleCommandEntity, dc *lockDynamicContext) (*TradeLockRuleFilterBackEntity, error)

// LockRuleChain mirrors the legacy tradeRuleFilter business linked list:
// ActivityUsability -> UserTakeLimit -> TeamStockOccupy.
type LockRuleChain struct {
	repo *TradeRepository
}

// NewLockRuleChain wires the chain.
func NewLockRuleChain(repo *TradeRepository) *LockRuleChain {
	return &LockRuleChain{repo: repo}
}

// Apply runs the chain.
func (c *LockRuleChain) Apply(ctx context.Context, cmd *TradeLockRuleCommandEntity) (*TradeLockRuleFilterBackEntity, error) {
	dc := &lockDynamicContext{}
	for _, f := range []lockFilter{c.activityUsability, c.userTakeLimit, c.teamStockOccupy} {
		back, err := f(ctx, cmd, dc)
		if err != nil {
			return nil, err
		}
		if back != nil {
			return back, nil
		}
	}
	// The terminal filter always returns a back entity; this is unreachable.
	return &TradeLockRuleFilterBackEntity{UserTakeOrderCount: dc.userTakeOrderCount}, nil
}

// activityUsability mirrors ActivityUsabilityRuleFilter (status + time window).
func (c *LockRuleChain) activityUsability(ctx context.Context, cmd *TradeLockRuleCommandEntity, dc *lockDynamicContext) (*TradeLockRuleFilterBackEntity, error) {
	log.Printf("[trade] lock rule filter - activity usability check userId:%s activityId:%d", cmd.UserId, cmd.ActivityId)
	activity, err := c.repo.QueryGroupBuyActivityEntityByActivityId(ctx, cmd.ActivityId)
	if err != nil {
		return nil, err
	}
	if activity.Status != types.ActivityStatusEffective {
		log.Printf("[trade] activity usability check failed, not in effective status activityId:%d", cmd.ActivityId)
		return nil, types.NewAppError(types.ResponseE0101)
	}
	now := types.Now()
	if now.Before(activity.StartTime) || now.After(activity.EndTime) {
		log.Printf("[trade] activity usability check failed, outside the join time window activityId:%d", cmd.ActivityId)
		return nil, types.NewAppError(types.ResponseE0102)
	}
	dc.groupBuyActivity = activity
	return nil, nil
}

// userTakeLimit mirrors UserTakeLimitRuleFilter.
func (c *LockRuleChain) userTakeLimit(ctx context.Context, cmd *TradeLockRuleCommandEntity, dc *lockDynamicContext) (*TradeLockRuleFilterBackEntity, error) {
	log.Printf("[trade] lock rule filter - user take-limit check userId:%s activityId:%d", cmd.UserId, cmd.ActivityId)
	activity := dc.groupBuyActivity
	count, err := c.repo.QueryOrderCountByActivityId(ctx, cmd.ActivityId, cmd.UserId)
	if err != nil {
		return nil, err
	}
	if count >= activity.TakeLimitCount {
		log.Printf("[trade] user take-limit check failed, participation limit reached activityId:%d", cmd.ActivityId)
		return nil, types.NewAppError(types.ResponseE0103)
	}
	dc.userTakeOrderCount = count
	return nil, nil
}

// teamStockOccupy mirrors TeamStockOccupyRuleFilter (Redis stock pre-emption).
func (c *LockRuleChain) teamStockOccupy(ctx context.Context, cmd *TradeLockRuleCommandEntity, dc *lockDynamicContext) (*TradeLockRuleFilterBackEntity, error) {
	log.Printf("[trade] lock rule filter - team stock check userId:%s activityId:%d", cmd.UserId, cmd.ActivityId)
	// An empty teamId means opening a new team, so no stock limit applies.
	if types.IsBlank(cmd.TeamId) {
		return &TradeLockRuleFilterBackEntity{UserTakeOrderCount: dc.userTakeOrderCount}, nil
	}
	activity := dc.groupBuyActivity
	teamStockKey := GenerateTeamStockKey(activity.ActivityId, cmd.TeamId)
	recoveryTeamStockKey := GenerateRecoveryTeamStockKey(activity.ActivityId, cmd.TeamId)
	status, err := c.repo.OccupyTeamStock(ctx, teamStockKey, recoveryTeamStockKey, activity.Target, activity.ValidTime)
	if err != nil {
		return nil, err
	}
	if !status {
		log.Printf("[trade] lock rule filter - team stock check failed userId:%s activityId:%d key:%s", cmd.UserId, cmd.ActivityId, teamStockKey)
		return nil, types.NewAppError(types.ResponseE0008)
	}
	return &TradeLockRuleFilterBackEntity{
		UserTakeOrderCount:   dc.userTakeOrderCount,
		RecoveryTeamStockKey: recoveryTeamStockKey,
	}, nil
}
