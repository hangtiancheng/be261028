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
			log.Printf("[trade] refund callback notify failed (%s) result:%v err:%v", refundType, result, err)
			return
		}
		log.Printf("[trade] refund callback notify (%s) result:%v", refundType, result)
	})
}

// doReverseStock mirrors AbstractRefundOrderStrategy.doReverseStock.
func (b *baseRefundStrategy) doReverseStock(ctx context.Context, msg *TeamRefundSuccess, refundType string) error {
	log.Printf("[trade] refund; restoring lock stock - %s userId:%s activityId:%d teamId:%s", refundType, msg.UserId, msg.ActivityId, msg.TeamId)
	recoveryTeamStockKey := GenerateRecoveryTeamStockKey(msg.ActivityId, msg.TeamId)
	return b.repo.Refund2AddRecovery(ctx, recoveryTeamStockKey, msg.OrderId)
}

// Unpaid2RefundStrategy mirrors unpaid2RefundStrategy (unpaid, team not formed).
type Unpaid2RefundStrategy struct{ baseRefundStrategy }

// RefundOrder decrements the lock count and records the refund notify task.
func (s *Unpaid2RefundStrategy) RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error {
	log.Printf("[trade] refund; unpaid, team not formed userId:%s teamId:%s orderId:%s", order.UserId, order.TeamId, order.OrderId)
	notifyTask, err := s.repo.Unpaid2Refund(ctx, BuildUnpaid2RefundAggregate(order, -1))
	if err != nil {
		return err
	}
	s.sendRefundNotifyMessage(notifyTask, "unpaid, team not formed")
	return nil
}

// ReverseStock restores the lock stock (there was a lock record to recover).
func (s *Unpaid2RefundStrategy) ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error {
	return s.doReverseStock(ctx, msg, "unpaid, team not formed, but a lock record exists; restoring lock stock")
}

// Paid2RefundStrategy mirrors paid2RefundStrategy (paid, team not formed).
type Paid2RefundStrategy struct{ baseRefundStrategy }

// RefundOrder decrements lock and complete counts and records the refund notify task.
func (s *Paid2RefundStrategy) RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error {
	log.Printf("[trade] refund; paid, team not formed userId:%s teamId:%s orderId:%s", order.UserId, order.TeamId, order.OrderId)
	notifyTask, err := s.repo.Paid2Refund(ctx, BuildPaid2RefundAggregate(order, -1, -1))
	if err != nil {
		return err
	}
	s.sendRefundNotifyMessage(notifyTask, "paid, team not formed")
	return nil
}

// ReverseStock restores the lock stock.
func (s *Paid2RefundStrategy) ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error {
	return s.doReverseStock(ctx, msg, "paid, team not formed, but a lock record exists; restoring lock stock")
}

// PaidTeam2RefundStrategy mirrors paidTeam2RefundStrategy (paid, team formed).
type PaidTeam2RefundStrategy struct{ baseRefundStrategy }

// RefundOrder refunds a formed-team member; the last member failing the team marks
// the team failed, otherwise it stays complete-with-refund.
func (s *PaidTeam2RefundStrategy) RefundOrder(ctx context.Context, order *TradeRefundOrderEntity) error {
	log.Printf("[trade] refund; paid, team formed userId:%s teamId:%s orderId:%s", order.UserId, order.TeamId, order.OrderId)
	team, err := s.repo.QueryGroupBuyTeamByTeamId(ctx, order.TeamId)
	if err != nil {
		return err
	}
	// If the last member also refunds, mark the team failed.
	teamStatus := types.GroupBuyOrderComplete
	if team.CompleteCount == 1 {
		teamStatus = types.GroupBuyOrderFail
	}
	notifyTask, err := s.repo.PaidTeam2Refund(ctx, BuildPaidTeam2RefundAggregate(order, -1, -1, teamStatus))
	if err != nil {
		return err
	}
	s.sendRefundNotifyMessage(notifyTask, "paid, team formed")
	return nil
}

// ReverseStock is a no-op for formed teams (the team window has ended).
func (s *PaidTeam2RefundStrategy) ReverseStock(ctx context.Context, msg *TeamRefundSuccess) error {
	log.Printf("[trade] refund; paid, team formed; team window ended, no lock stock to restore userId:%s activityId:%d teamId:%s", msg.UserId, msg.ActivityId, msg.TeamId)
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
	log.Printf("[trade] reverse flow - refund, data load node userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order, err := c.repo.QueryMarketPayOrderEntityByOutTradeNo(ctx, cmd.UserId, cmd.OutTradeNo)
	if err != nil {
		return nil, err
	}
	if order == nil {
		log.Printf("[trade] unknown outTradeNo or already refunded, cannot refund userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
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
	log.Printf("[trade] reverse flow - refund, duplicate refund check userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order := dc.marketPayOrderEntity
	if order.TradeOrderStatus == types.TradeOrderClose {
		log.Printf("[trade] reverse flow - refund (idempotent, duplicate) userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
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
	log.Printf("[trade] reverse flow - refund, strategy dispatch userId:%s outTradeNo:%s", cmd.UserId, cmd.OutTradeNo)
	order := dc.marketPayOrderEntity
	team := dc.groupBuyTeamEntity
	refundType, err := types.GetRefundStrategy(team.Status, order.TradeOrderStatus)
	if err != nil {
		return nil, err
	}
	strategy := c.strategies[refundType.Strategy()]
	if strategy == nil {
		return nil, fmt.Errorf("no refund strategy registered: %s", refundType.Strategy())
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
				log.Printf("[trade] async task panic: %v", r)
			}
		}()
		fn()
	}()
}
