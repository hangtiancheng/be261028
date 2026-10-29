package trade

import (
	"context"
	"fmt"
	"log"

	"github.com/hangtiancheng/be261028/internal/types"
)

// TradeLockOrderService mirrors the legacy TradeLockOrderService.
type TradeLockOrderService struct {
	repo      *TradeRepository
	lockChain *LockRuleChain
}

// NewTradeLockOrderService wires the service.
func NewTradeLockOrderService(repo *TradeRepository, lockChain *LockRuleChain) *TradeLockOrderService {
	return &TradeLockOrderService{repo: repo, lockChain: lockChain}
}

// QueryNoPayMarketPayOrderByOutTradeNo mirrors the legacy idempotency lookup.
func (s *TradeLockOrderService) QueryNoPayMarketPayOrderByOutTradeNo(ctx context.Context, userId, outTradeNo string) (*MarketPayOrderEntity, error) {
	log.Printf("拼团交易-查询未支付营销订单:%s outTradeNo:%s", userId, outTradeNo)
	return s.repo.QueryMarketPayOrderEntityByOutTradeNo(ctx, userId, outTradeNo)
}

// QueryGroupBuyProgress mirrors the legacy progress lookup.
func (s *TradeLockOrderService) QueryGroupBuyProgress(ctx context.Context, teamId string) (*GroupBuyProgressVO, error) {
	log.Printf("拼团交易-查询拼单进度:%s", teamId)
	return s.repo.QueryGroupBuyProgress(ctx, teamId)
}

// LockMarketPayOrder mirrors the legacy lockMarketPayOrder: run the rule chain,
// build the aggregate, persist, and recover Redis stock on failure.
func (s *TradeLockOrderService) LockMarketPayOrder(ctx context.Context, user *UserEntity, payActivity *PayActivityEntity, payDiscount *PayDiscountEntity) (*MarketPayOrderEntity, error) {
	log.Printf("拼团交易-锁定营销优惠支付订单:%s activityId:%d goodsId:%s", user.UserId, payActivity.ActivityId, payDiscount.GoodsId)
	back, err := s.lockChain.Apply(ctx, &TradeLockRuleCommandEntity{
		UserId:     user.UserId,
		ActivityId: payActivity.ActivityId,
		TeamId:     payActivity.TeamId,
	})
	if err != nil {
		return nil, err
	}
	agg := &GroupBuyOrderAggregate{
		UserEntity:         user,
		PayActivityEntity:  payActivity,
		PayDiscountEntity:  payDiscount,
		UserTakeOrderCount: back.UserTakeOrderCount,
	}
	entity, err := s.repo.LockMarketPayOrder(ctx, agg)
	if err != nil {
		// 记录失败恢复量
		if recErr := s.repo.RecoveryTeamStock(ctx, back.RecoveryTeamStockKey, payActivity.ValidTime); recErr != nil {
			log.Printf("拼团交易-恢复组队库存失败 key:%s err:%v", back.RecoveryTeamStockKey, recErr)
		}
		return nil, err
	}
	return entity, nil
}

// TradeSettlementOrderService mirrors the legacy TradeSettlementOrderService.
type TradeSettlementOrderService struct {
	repo            *TradeRepository
	settlementChain *SettlementRuleChain
	task            *TradeTaskService
}

// NewTradeSettlementOrderService wires the service.
func NewTradeSettlementOrderService(repo *TradeRepository, chain *SettlementRuleChain, task *TradeTaskService) *TradeSettlementOrderService {
	return &TradeSettlementOrderService{repo: repo, settlementChain: chain, task: task}
}

// SettlementMarketPayOrder mirrors the legacy settlementMarketPayOrder.
func (s *TradeSettlementOrderService) SettlementMarketPayOrder(ctx context.Context, paySuccess *TradePaySuccessEntity) (*TradePaySettlementEntity, error) {
	log.Printf("拼团交易-支付订单结算:%s outTradeNo:%s", paySuccess.UserId, paySuccess.OutTradeNo)
	// 1. 结算规则过滤
	back, err := s.settlementChain.Apply(ctx, &TradeSettlementRuleCommandEntity{
		Source:       paySuccess.Source,
		Channel:      paySuccess.Channel,
		UserId:       paySuccess.UserId,
		OutTradeNo:   paySuccess.OutTradeNo,
		OutTradeTime: paySuccess.OutTradeTime,
	})
	if err != nil {
		return nil, err
	}
	// 缺陷修复：重复结算回调幂等返回成功（旧版会重复累加成团数量）
	if back.AlreadySettled {
		return &TradePaySettlementEntity{
			Source:     paySuccess.Source,
			Channel:    paySuccess.Channel,
			UserId:     paySuccess.UserId,
			TeamId:     back.TeamId,
			ActivityId: back.ActivityId,
			OutTradeNo: paySuccess.OutTradeNo,
		}, nil
	}
	// 2. 查询组团信息
	team := &GroupBuyTeamEntity{
		TeamId:         back.TeamId,
		ActivityId:     back.ActivityId,
		TargetCount:    back.TargetCount,
		CompleteCount:  back.CompleteCount,
		LockCount:      back.LockCount,
		Status:         back.Status,
		ValidStartTime: back.ValidStartTime,
		ValidEndTime:   back.ValidEndTime,
		NotifyConfigVO: back.NotifyConfigVO,
	}
	// 3. 构建聚合对象
	agg := &GroupBuyTeamSettlementAggregate{
		UserEntity:            &UserEntity{UserId: paySuccess.UserId},
		GroupBuyTeamEntity:    team,
		TradePaySuccessEntity: paySuccess,
	}
	// 4. 拼团交易结算
	notifyTask, err := s.repo.SettlementMarketPayOrder(ctx, agg)
	if err != nil {
		return nil, err
	}
	// 5. 组队回调处理（异步；失败有定时任务补偿）
	if notifyTask != nil {
		task := s.task
		asyncExec(func() {
			result, err := task.ExecNotifyJobTask(context.Background(), notifyTask)
			if err != nil {
				log.Printf("回调通知拼团完结失败 result:%v err:%v", result, err)
				return
			}
			log.Printf("回调通知拼团完结 result:%v", result)
		})
	}
	// 6. 返回结算信息
	return &TradePaySettlementEntity{
		Source:     paySuccess.Source,
		Channel:    paySuccess.Channel,
		UserId:     paySuccess.UserId,
		TeamId:     team.TeamId,
		ActivityId: team.ActivityId,
		OutTradeNo: paySuccess.OutTradeNo,
	}, nil
}

// TradeRefundOrderService mirrors the legacy TradeRefundOrderService.
type TradeRefundOrderService struct {
	refundChain *RefundRuleChain
	strategies  map[string]RefundOrderStrategy
}

// NewTradeRefundOrderService wires the service.
func NewTradeRefundOrderService(refundChain *RefundRuleChain, strategies map[string]RefundOrderStrategy) *TradeRefundOrderService {
	return &TradeRefundOrderService{refundChain: refundChain, strategies: strategies}
}

// RefundOrder mirrors the legacy refundOrder (forward refund flow).
func (s *TradeRefundOrderService) RefundOrder(ctx context.Context, cmd *TradeRefundCommandEntity) (*TradeRefundBehaviorEntity, error) {
	log.Printf("逆向流程，退单操作 userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	return s.refundChain.Apply(ctx, cmd)
}

// RestoreTeamLockStock mirrors the legacy restoreTeamLockStock. In the legacy
// project this had no caller (the team-refund consumer was never wired), so the
// refund stock-recovery loop was broken; the be261028 trade service consumes the
// team-refund Kafka topic and invokes this to close the loop.
func (s *TradeRefundOrderService) RestoreTeamLockStock(ctx context.Context, msg *TeamRefundSuccess) error {
	log.Printf("逆向流程，恢复锁单量 userId:%s activityId:%d teamId:%s", msg.UserId, msg.ActivityId, msg.TeamId)
	refundType, err := types.GetRefundTypeByCode(msg.Type)
	if err != nil {
		return err
	}
	strategy := s.strategies[refundType.Strategy()]
	if strategy == nil {
		return fmt.Errorf("不存在退单策略: %s", refundType.Strategy())
	}
	return strategy.ReverseStock(ctx, msg)
}
