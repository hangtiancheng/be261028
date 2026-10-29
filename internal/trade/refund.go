package trade

import (
	"context"
	"fmt"
	"log"

	"github.com/hangtiancheng/be261028/internal/types"
)

// RefundOrderStrategy mirrors IRefundOrderStrategy: one strategy per refund type.
type RefundOrderStrategy interface {
	RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error
	ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error
}

// baseRefundStrategy mirrors AbstractRefundOrderStrategy: shared repository access
// and the async notify + stock-recovery helpers.
type baseRefundStrategy struct {
	repo *TradeRepository
	task *TradeTaskService
}

// sendRefundNotifyMessage mirrors AbstractRefundOrderStrategy.sendRefundNotifyMessage:
// the notify task is executed asynchronously so the refund response is not blocked.
func (b *baseRefundStrategy) sendRefundNotifyMessage(notifyTask *NotifyTaskEntity, refundType string) {
	if notifyTask == nil {
		return
	}
	asyncExec(func() {
		result, err := b.task.ExecNotifyJobTask(context.Background(), notifyTask)
		if err != nil {
			log.Printf("回调通知交易退单失败(%s) result:%v err:%v", refundType, result, err)
			return
		}
		log.Printf("回调通知交易退单(%s) result:%v", refundType, result)
	})
}

// doReverseStock mirrors AbstractRefundOrderStrategy.doReverseStock.
func (b *baseRefundStrategy) doReverseStock(ctx context.Context, msg *TeamRefundSuccess, refundType string) error {
	log.Printf("退单；恢复锁单量 - %s %s %d %s", refundType, msg.UserId, msg.ActivityId, msg.TeamId)
	recoveryTeamStockKey := GenerateRecoveryTeamStockKey(msg.ActivityId, msg.TeamId)
	return b.repo.Refund2AddRecovery(ctx, recoveryTeamStockKey, msg.OrderId)
}

// Unpaid2RefundStrategy mirrors unpaid2RefundStrategy (未支付，未成团).
type Unpaid2RefundStrategy struct{ baseRefundStrategy }

// RefundOrder decrements the lock count and records the refund notify task.
func (s *Unpaid2RefundStrategy) RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error {
	log.Printf("退单；未支付，未成团 userId:%s teamId:%s orderId:%s", order.UserId, order.TeamId, order.OrderId)
	notifyTask, err := s.repo.Unpaid2Refund(ctx, BuildUnpaid2RefundAggregate(order, -1))
	if err != nil {
		return err
	}
	s.sendRefundNotifyMessage(notifyTask, "未支付，未成团")
	return nil
}

// ReverseStock restores the lock stock (there was a lock record to recover).
func (s *Unpaid2RefundStrategy) ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error {
	return s.doReverseStock(ctx, msg, "未支付，未成团，但有锁单记录，要恢复锁单库存")
}

// Paid2RefundStrategy mirrors paid2RefundStrategy (已支付，未成团).
type Paid2RefundStrategy struct{ baseRefundStrategy }

// RefundOrder decrements lock and complete counts and records the refund notify task.
func (s *Paid2RefundStrategy) RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error {
	log.Printf("退单；已支付，未成团 userId:%s teamId:%s orderId:%s", order.UserId, order.TeamId, order.OrderId)
	notifyTask, err := s.repo.Paid2Refund(ctx, BuildPaid2RefundAggregate(order, -1, -1))
	if err != nil {
		return err
	}
	s.sendRefundNotifyMessage(notifyTask, "已支付，未成团")
	return nil
}

// ReverseStock restores the lock stock.
func (s *Paid2RefundStrategy) ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error {
	return s.doReverseStock(ctx, msg, "已支付，未成团，但有锁单记录，要恢复锁单库存")
}

// PaidTeam2RefundStrategy mirrors paidTeam2RefundStrategy (已支付，已成团).
type PaidTeam2RefundStrategy struct{ baseRefundStrategy }

// RefundOrder refunds a formed-team member; the last member failing the team marks
// the team failed, otherwise it stays complete-with-refund.
func (s *PaidTeam2RefundStrategy) RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error {
	log.Printf("退单；已支付，已成团 userId:%s teamId:%s orderId:%s", order.UserId, order.TeamId, order.OrderId)
	team, err := s.repo.QueryGroupBuyTeamByTeamId(ctx, order.TeamId)
	if err != nil {
		return err
	}
	// 最后一笔也退单，则更新拼团订单为失败
	teamStatus := types.GroupBuyOrderComplete
	if team.CompleteCount == 1 {
		teamStatus = types.GroupBuyOrderFail
	}
	notifyTask, err := s.repo.PaidTeam2Refund(ctx, BuildPaidTeam2RefundAggregate(order, -1, -1, teamStatus))
	if err != nil {
		return err
	}
	s.sendRefundNotifyMessage(notifyTask, "已支付，已成团")
	return nil
}

// ReverseStock is a no-op for formed teams (the team window has ended).
func (s *PaidTeam2RefundStrategy) ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error {
	log.Printf("退单；已支付、已成团，队伍组队结束，不需要恢复锁单量 %s %d %s", msg.UserId, msg.ActivityId, msg.TeamId)
	return nil
}

// NewRefundStrategies builds the strategy map keyed by the legacy bean names.
func NewRefundStrategies(repo *TradeRepository, task *TradeTaskService) map[string]RefundOrderStrategy {
	base := baseRefundStrategy{repo: repo, task: task}
	return map[string]RefundOrderStrategy{
		"unpaid2RefundStrategy":   &Unpaid2RefundStrategy{base},
		"paid2RefundStrategy":     &Paid2RefundStrategy{base},
		"paidTeam2RefundStrategy": &PaidTeam2RefundStrategy{base},
	}
}

// ---- refund rule chain ----

// refundDynamicContext mirrors TradeRefundRuleFilterFactory.DynamicContext.
type refundDynamicContext struct {
	marketPayOrderEntity *MarketPayOrderEntity
	groupBuyTeamEntity   *GroupBuyTeamEntity
}

// refundFilter is one link of the refund rule chain. A non-nil behavior entity
// terminates the chain.
type refundFilter func(ctx context.Context, cmd *TradeRefundCommandEntity, dc *refundDynamicContext) (*TradeRefundBehaviorEntity, error)

// RefundRuleChain mirrors the legacy tradeRefundRuleFilter business linked list:
// Data -> UniqueRefund -> RefundOrder.
type RefundRuleChain struct {
	repo       *TradeRepository
	strategies map[string]RefundOrderStrategy
}

// NewRefundRuleChain wires the chain.
func NewRefundRuleChain(repo *TradeRepository, strategies map[string]RefundOrderStrategy) *RefundRuleChain {
	return &RefundRuleChain{repo: repo, strategies: strategies}
}

// Apply runs the chain.
func (c *RefundRuleChain) Apply(ctx context.Context, cmd *TradeRefundCommandEntity) (*TradeRefundBehaviorEntity, error) {
	dc := &refundDynamicContext{}
	for _, f := range []refundFilter{c.dataNode, c.uniqueRefundNode, c.refundOrderNode} {
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

// dataNode mirrors DataNodeFilter. Defect fix: the legacy version also rejected
// already-refunded (CLOSE) orders here with E0107, which made the idempotent
// REPEAT branch in uniqueRefundNode unreachable. Only the missing-order check is
// kept here; duplicate refunds are handled idempotently by uniqueRefundNode.
func (c *RefundRuleChain) dataNode(ctx context.Context, cmd *TradeRefundCommandEntity, dc *refundDynamicContext) (*TradeRefundBehaviorEntity, error) {
	log.Printf("逆向流程-退单操作，数据加载节点 userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order, err := c.repo.QueryMarketPayOrderEntityByOutTradeNo(ctx, cmd.UserId, cmd.OutTradeNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		log.Printf("不存在的外部交易单号或用户已退单，无法进行退款业务:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
		return nil, types.NewAppError(types.ResponseE0104)
	}
	team, err := c.repo.QueryGroupBuyTeamByTeamId(ctx, order.TeamId)
	if err != nil {
		return nil, err
	}
	dc.marketPayOrderEntity = order
	dc.groupBuyTeamEntity = team
	return nil, nil
}

// uniqueRefundNode mirrors UniqueRefundNodeFilter: an already-refunded order is an
// idempotent repeat, not an error.
func (c *RefundRuleChain) uniqueRefundNode(ctx context.Context, cmd *TradeRefundCommandEntity, dc *refundDynamicContext) (*TradeRefundBehaviorEntity, error) {
	log.Printf("逆向流程-退单操作，重复退单检查 userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order := dc.marketPayOrderEntity
	if order.TradeOrderStatus == types.TradeOrderClose {
		log.Printf("逆向流程，退单操作(幂等-重复退单) userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
		return &TradeRefundBehaviorEntity{
			UserId:   cmd.UserId,
			OrderId:  order.OrderId,
			TeamId:   order.TeamId,
			Behavior: types.RefundBehaviorRepeat,
		}, nil
	}
	return nil, nil
}

// refundOrderNode mirrors RefundOrderNodeFilter: pick the strategy by the
// (team status, order status) pair and execute the refund.
func (c *RefundRuleChain) refundOrderNode(ctx context.Context, cmd *TradeRefundCommandEntity, dc *refundDynamicContext) (*TradeRefundBehaviorEntity, error) {
	log.Printf("逆向流程-退单操作，退单策略处理 userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order := dc.marketPayOrderEntity
	team := dc.groupBuyTeamEntity
	refundType, err := types.GetRefundStrategy(team.Status, order.TradeOrderStatus)
	if err != nil {
		return nil, err
	}
	strategy := c.strategies[refundType.Strategy()]
	if strategy == nil {
		return nil, fmt.Errorf("不存在退单策略: %s", refundType.Strategy())
	}
	if err := strategy.RefundOrder(ctx, &TradeRefundOrderEntity{
		UserId:     cmd.UserId,
		OrderId:    order.OrderId,
		TeamId:     order.TeamId,
		ActivityId: team.ActivityId,
	}); err != nil {
		return nil, err
	}
	return &TradeRefundBehaviorEntity{
		UserId:   cmd.UserId,
		OrderId:  order.OrderId,
		TeamId:   order.TeamId,
		Behavior: types.RefundBehaviorSuccess,
	}, nil
}

// asyncExec runs fn in a goroutine with panic recovery, replacing the legacy
// thread-pool fire-and-forget notification dispatch.
func asyncExec(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[trade] 异步任务 panic: %v", r)
			}
		}()
		fn()
	}()
}
