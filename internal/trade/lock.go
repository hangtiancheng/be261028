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
	log.Printf("交易规则过滤-活动的可用性校验%s activityId:%d", cmd.UserId, cmd.ActivityId)
	activity, err := c.repo.QueryGroupBuyActivityEntityByActivityId(ctx, cmd.ActivityId)
	if err != nil {
		return nil, err
	}
	if activity.Status != types.ActivityStatusEffective {
		log.Printf("活动的可用性校验，非生效状态 activityId:%d", cmd.ActivityId)
		return nil, types.NewAppError(types.ResponseE0101)
	}
	now := types.Now()
	if now.Before(activity.StartTime) || now.After(activity.EndTime) {
		log.Printf("活动的可用性校验，非可参与时间范围 activityId:%d", cmd.ActivityId)
		return nil, types.NewAppError(types.ResponseE0102)
	}
	dc.groupBuyActivity = activity
	return nil, nil
}

// userTakeLimit mirrors UserTakeLimitRuleFilter.
func (c *LockRuleChain) userTakeLimit(ctx context.Context, cmd *TradeLockRuleCommandEntity, dc *lockDynamicContext) (*TradeLockRuleFilterBackEntity, error) {
	log.Printf("交易规则过滤-用户参与次数校验%s activityId:%d", cmd.UserId, cmd.ActivityId)
	activity := dc.groupBuyActivity
	count, err := c.repo.QueryOrderCountByActivityId(ctx, cmd.ActivityId, cmd.UserId)
	if err != nil {
		return nil, err
	}
	if count >= activity.TakeLimitCount {
		log.Printf("用户参与次数校验，已达可参与上限 activityId:%d", cmd.ActivityId)
		return nil, types.NewAppError(types.ResponseE0103)
	}
	dc.userTakeOrderCount = count
	return nil, nil
}

// teamStockOccupy mirrors TeamStockOccupyRuleFilter (Redis stock pre-emption).
func (c *LockRuleChain) teamStockOccupy(ctx context.Context, cmd *TradeLockRuleCommandEntity, dc *lockDynamicContext) (*TradeLockRuleFilterBackEntity, error) {
	log.Printf("交易规则过滤-组队库存校验%s activityId:%d", cmd.UserId, cmd.ActivityId)
	// teamId 为空则为首次开团，不做库存限制
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
		log.Printf("交易规则过滤-组队库存校验%s activityId:%d 抢占失败:%s", cmd.UserId, cmd.ActivityId, teamStockKey)
		return nil, types.NewAppError(types.ResponseE0008)
	}
	return &TradeLockRuleFilterBackEntity{
		UserTakeOrderCount:   dc.userTakeOrderCount,
		RecoveryTeamStockKey: recoveryTeamStockKey,
	}, nil
}
