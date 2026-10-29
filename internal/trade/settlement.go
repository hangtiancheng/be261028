package trade

import (
	"context"
	"log"

	"github.com/hangtiancheng/be261028/internal/types"
)

// settlementDynamicContext mirrors TradeSettlementRuleFilterFactory.DynamicContext.
type settlementDynamicContext struct {
	marketPayOrderEntity *MarketPayOrderEntity
	groupBuyTeamEntity   *GroupBuyTeamEntity
}

// settlementFilter is one link of the settlement rule chain.
type settlementFilter func(ctx context.Context, cmd *TradeSettlementRuleCommandEntity, dc *settlementDynamicContext) (*TradeSettlementRuleFilterBackEntity, error)

// SettlementRuleChain mirrors the legacy tradeSettlementRuleFilter business linked
// list: SC -> OutTradeNo -> Settable -> End.
type SettlementRuleChain struct {
	repo *TradeRepository
}

// NewSettlementRuleChain wires the chain.
func NewSettlementRuleChain(repo *TradeRepository) *SettlementRuleChain {
	return &SettlementRuleChain{repo: repo}
}

// Apply runs the chain.
func (c *SettlementRuleChain) Apply(ctx context.Context, cmd *TradeSettlementRuleCommandEntity) (*TradeSettlementRuleFilterBackEntity, error) {
	dc := &settlementDynamicContext{}
	for _, f := range []settlementFilter{c.scRule, c.outTradeNoRule, c.settableRule, c.endRule} {
		back, err := f(ctx, cmd, dc)
		if err != nil {
			return nil, err
		}
		if back != nil {
			return back, nil
		}
	}
	return nil, nil
}

// scRule mirrors SCRuleFilter (channel blacklist).
func (c *SettlementRuleChain) scRule(ctx context.Context, cmd *TradeSettlementRuleCommandEntity, dc *settlementDynamicContext) (*TradeSettlementRuleFilterBackEntity, error) {
	log.Printf("结算规则过滤-渠道黑名单校验%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	if c.repo.IsSCBlackIntercept(cmd.Source, cmd.Channel) {
		log.Printf("%s%s 渠道黑名单拦截", cmd.Source, cmd.Channel)
		return nil, types.NewAppError(types.ResponseE0105)
	}
	return nil, nil
}

// outTradeNoRule mirrors OutTradeNoRuleFilter (order exists and not refunded).
//
// Defect fix: an order that is already COMPLETE short-circuits the chain as an
// idempotent success. The legacy filter let repeat settlement callbacks through
// and the unguarded status update double-incremented the team's complete_count,
// completing teams with fewer real payments than the target.
func (c *SettlementRuleChain) outTradeNoRule(ctx context.Context, cmd *TradeSettlementRuleCommandEntity, dc *settlementDynamicContext) (*TradeSettlementRuleFilterBackEntity, error) {
	log.Printf("结算规则过滤-外部单号校验%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order, err := c.repo.QueryMarketPayOrderEntityByOutTradeNo(ctx, cmd.UserId, cmd.OutTradeNo)
	if err != nil {
		return nil, err
	}
	if order == nil || order.TradeOrderStatus == types.TradeOrderClose {
		log.Printf("不存在的外部交易单号或用户已退单，不需要做支付订单结算:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
		return nil, types.NewAppError(types.ResponseE0104)
	}
	if order.TradeOrderStatus == types.TradeOrderComplete {
		log.Printf("外部交易单号已结算，幂等返回:%s outTradeNo:%s teamId:%s", cmd.UserId, cmd.OutTradeNo, order.TeamId)
		return &TradeSettlementRuleFilterBackEntity{
			TeamId:         order.TeamId,
			ActivityId:     order.ActivityId,
			AlreadySettled: true,
		}, nil
	}
	dc.marketPayOrderEntity = order
	return nil, nil
}

// settableRule mirrors SettableRuleFilter (payment time within the team window).
func (c *SettlementRuleChain) settableRule(ctx context.Context, cmd *TradeSettlementRuleCommandEntity, dc *settlementDynamicContext) (*TradeSettlementRuleFilterBackEntity, error) {
	log.Printf("结算规则过滤-有效时间校验%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	team, err := c.repo.QueryGroupBuyTeamByTeamId(ctx, dc.marketPayOrderEntity.TeamId)
	if err != nil {
		return nil, err
	}
	// 外部交易时间（用户支付完成时间）必须在拼团有效时间范围内
	if !cmd.OutTradeTime.Before(team.ValidEndTime) {
		log.Printf("订单交易时间不在拼团有效时间范围内")
		return nil, types.NewAppError(types.ResponseE0106)
	}
	dc.groupBuyTeamEntity = team
	return nil, nil
}

// endRule mirrors EndRuleFilter (assemble the back entity).
func (c *SettlementRuleChain) endRule(ctx context.Context, cmd *TradeSettlementRuleCommandEntity, dc *settlementDynamicContext) (*TradeSettlementRuleFilterBackEntity, error) {
	log.Printf("结算规则过滤-结束节点%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	team := dc.groupBuyTeamEntity
	return &TradeSettlementRuleFilterBackEntity{
		TeamId:         team.TeamId,
		ActivityId:     team.ActivityId,
		TargetCount:    team.TargetCount,
		CompleteCount:  team.CompleteCount,
		LockCount:      team.LockCount,
		Status:         team.Status,
		ValidStartTime: team.ValidStartTime,
		ValidEndTime:   team.ValidEndTime,
		NotifyConfigVO: team.NotifyConfigVO,
	}, nil
}
