package trade

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hangtiancheng/be261028/internal/dao"
	"github.com/hangtiancheng/be261028/internal/infra/dcc"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/types"
	"gorm.io/gorm"
)

// TradeRepository mirrors the legacy TradeRepository: all trade persistence plus
// the Redis team-stock counters. Settlement and refund operations run in a
// transaction; notify tasks are addressed by id (see the dao defect fix).
type TradeRepository struct {
	db               *gorm.DB
	dao              *dao.DAO
	redis            *redisx.RedisService
	dcc              *dcc.DCC
	topicTeamSuccess string
	topicTeamRefund  string
}

// NewTradeRepository wires the repository. topicTeamSuccess/topicTeamRefund are
// the Kafka topics recorded on notify tasks (replacing the legacy routing keys).
func NewTradeRepository(db *gorm.DB, d *dao.DAO, redis *redisx.RedisService, dc *dcc.DCC, topicTeamSuccess, topicTeamRefund string) *TradeRepository {
	return &TradeRepository{db: db, dao: d, redis: redis, dcc: dc, topicTeamSuccess: topicTeamSuccess, topicTeamRefund: topicTeamRefund}
}

// QueryMarketPayOrderEntityByOutTradeNo mirrors the legacy query.
func (r *TradeRepository) QueryMarketPayOrderEntityByOutTradeNo(ctx context.Context, userId, outTradeNo string) (*MarketPayOrderEntity, error) {
	po, err := r.dao.QueryGroupBuyOrderRecordByOutTradeNo(ctx, r.db, userId, outTradeNo)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &MarketPayOrderEntity{
		TeamId:           po.TeamId,
		OrderId:          po.OrderId,
		ActivityId:       po.ActivityId,
		OriginalPrice:    po.OriginalPrice,
		DeductionPrice:   po.DeductionPrice,
		PayPrice:         po.PayPrice,
		TradeOrderStatus: types.TradeOrderStatusOf(po.Status),
	}, nil
}

// LockMarketPayOrder mirrors the legacy lockMarketPayOrder. It is wrapped in a
// transaction so a failed order-list insert cannot leave an orphaned team row
// (the legacy version was non-transactional).
func (r *TradeRepository) LockMarketPayOrder(ctx context.Context, agg *GroupBuyOrderAggregate) (*MarketPayOrderEntity, error) {
	user := agg.UserEntity
	payActivity := agg.PayActivityEntity
	payDiscount := agg.PayDiscountEntity
	notifyConfig := payDiscount.NotifyConfigVO
	userTakeOrderCount := agg.UserTakeOrderCount

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() //nolint:errcheck

	teamId := payActivity.TeamId
	if types.IsBlank(teamId) {
		// First-time team creation: generate a teamId and insert the group-buy order.
		teamId = randomNumeric(8)
		currentDate := types.Now()
		validEndTime := currentDate.Add(time.Duration(payActivity.ValidTime) * time.Minute)
		order := &dao.GroupBuyOrder{
			TeamId:         teamId,
			ActivityId:     payActivity.ActivityId,
			Source:         payDiscount.Source,
			Channel:        payDiscount.Channel,
			OriginalPrice:  payDiscount.OriginalPrice,
			DeductionPrice: payDiscount.DeductionPrice,
			PayPrice:       payDiscount.PayPrice,
			TargetCount:    payActivity.TargetCount,
			CompleteCount:  0,
			LockCount:      1,
			ValidStartTime: currentDate,
			ValidEndTime:   validEndTime,
			NotifyType:     string(notifyConfig.NotifyType),
			NotifyUrl:      nullString(notifyConfig.NotifyUrl),
		}
		err := r.dao.InsertGroupBuyOrder(ctx, tx, order)
		if err != nil && dao.IsDuplicateKey(err) {
			// Defect fix: on a random-teamId unique-key collision, regenerate and retry
			// once instead of surfacing the error to the user.
			log.Printf("[trade] team creation teamId collision, regenerating and retrying oldTeamId:%s", teamId)
			teamId = randomNumeric(8)
			order.TeamId = teamId
			err = r.dao.InsertGroupBuyOrder(ctx, tx, order)
		}
		if err != nil {
			return nil, err
		}
	} else {
		// Joining an existing team: bump the lock count by 1; only allowed while not full.
		updated, err := r.dao.UpdateAddLockCount(ctx, tx, teamId)
		if err != nil {
			return nil, err
		}
		if updated != 1 {
			return nil, types.NewAppError(types.ResponseE0005)
		}
	}

	orderId := randomNumeric(12)
	bizId := strconv.FormatInt(payActivity.ActivityId, 10) + types.Underline + user.UserId +
		types.Underline + strconv.Itoa(userTakeOrderCount+1)
	orderList := &dao.GroupBuyOrderList{
		UserId:         user.UserId,
		TeamId:         teamId,
		OrderId:        orderId,
		ActivityId:     payActivity.ActivityId,
		StartTime:      payActivity.StartTime,
		EndTime:        payActivity.EndTime,
		GoodsId:        payDiscount.GoodsId,
		Source:         payDiscount.Source,
		Channel:        payDiscount.Channel,
		OriginalPrice:  payDiscount.OriginalPrice,
		DeductionPrice: payDiscount.DeductionPrice,
		PayPrice:       payDiscount.PayPrice,
		Status:         int(types.TradeOrderCreate),
		OutTradeNo:     payDiscount.OutTradeNo,
		BizId:          bizId,
	}
	if err := r.dao.InsertGroupBuyOrderList(ctx, tx, orderList); err != nil {
		if dao.IsDuplicateKey(err) {
			return nil, types.NewAppError(types.ResponseIndexException)
		}
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &MarketPayOrderEntity{
		TeamId:           teamId,
		OrderId:          orderId,
		ActivityId:       payActivity.ActivityId,
		OriginalPrice:    payDiscount.OriginalPrice,
		DeductionPrice:   payDiscount.DeductionPrice,
		PayPrice:         payDiscount.PayPrice,
		TradeOrderStatus: types.TradeOrderCreate,
	}, nil
}

// QueryGroupBuyProgress mirrors the legacy query.
func (r *TradeRepository) QueryGroupBuyProgress(ctx context.Context, teamId string) (*GroupBuyProgressVO, error) {
	po, err := r.dao.QueryGroupBuyProgress(ctx, r.db, teamId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &GroupBuyProgressVO{
		TargetCount:   po.TargetCount,
		CompleteCount: po.CompleteCount,
		LockCount:     po.LockCount,
	}, nil
}

// QueryGroupBuyActivityEntityByActivityId mirrors the legacy query. A missing
// activity yields a plain error (the legacy code dereferenced null, surfacing as
// an unknown failure).
func (r *TradeRepository) QueryGroupBuyActivityEntityByActivityId(ctx context.Context, activityId int64) (*GroupBuyActivityEntity, error) {
	po, err := r.dao.QueryGroupBuyActivityByActivityId(ctx, r.db, activityId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, fmt.Errorf("group-buy activity not found: %d", activityId)
	}
	if err != nil {
		return nil, err
	}
	return &GroupBuyActivityEntity{
		ActivityId:     po.ActivityId,
		ActivityName:   po.ActivityName,
		DiscountId:     po.DiscountId,
		GroupType:      po.GroupType,
		TakeLimitCount: po.TakeLimitCount,
		Target:         po.Target,
		ValidTime:      po.ValidTime,
		Status:         types.ActivityStatusOf(po.Status),
		StartTime:      po.StartTime,
		EndTime:        po.EndTime,
		TagId:          po.TagId.String,
		TagScope:       po.TagScope.String,
	}, nil
}

// QueryOrderCountByActivityId mirrors the legacy query.
func (r *TradeRepository) QueryOrderCountByActivityId(ctx context.Context, activityId int64, userId string) (int, error) {
	return r.dao.QueryOrderCountByActivityId(ctx, r.db, userId, activityId)
}

// QueryGroupBuyTeamByTeamId mirrors the legacy query, including the fixed
// notifyMQ (team-success topic) on the notify config.
func (r *TradeRepository) QueryGroupBuyTeamByTeamId(ctx context.Context, teamId string) (*GroupBuyTeamEntity, error) {
	po, err := r.dao.QueryGroupBuyTeamByTeamId(ctx, r.db, teamId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, fmt.Errorf("group-buy team not found: %s", teamId)
	}
	if err != nil {
		return nil, err
	}
	return &GroupBuyTeamEntity{
		TeamId:         po.TeamId,
		ActivityId:     po.ActivityId,
		TargetCount:    po.TargetCount,
		CompleteCount:  po.CompleteCount,
		LockCount:      po.LockCount,
		Status:         types.GroupBuyOrderStatusOf(po.Status),
		ValidStartTime: po.ValidStartTime,
		ValidEndTime:   po.ValidEndTime,
		NotifyUrl:      po.NotifyUrl.String,
		NotifyConfigVO: &NotifyConfigVO{
			NotifyType: types.NotifyTypeOf(po.NotifyType),
			NotifyUrl:  po.NotifyUrl.String,
			NotifyMQ:   r.topicTeamSuccess, // MQ callbacks always use the team-success topic
		},
	}, nil
}

// SettlementMarketPayOrder mirrors the legacy transactional settlement. It returns
// a notify task only when this settlement completes the team.
func (r *TradeRepository) SettlementMarketPayOrder(ctx context.Context, agg *GroupBuyTeamSettlementAggregate) (*NotifyTaskEntity, error) {
	user := agg.UserEntity
	team := agg.GroupBuyTeamEntity
	notifyConfig := team.NotifyConfigVO
	paySuccess := agg.TradePaySuccessEntity

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() //nolint:errcheck

	// 1. Mark the order-detail row complete.
	cnt, err := r.dao.UpdateOrderListStatus2COMPLETE(ctx, tx, user.UserId, paySuccess.OutTradeNo, paySuccess.OutTradeTime)
	if err != nil {
		return nil, err
	}
	if cnt != 1 {
		return nil, types.NewAppError(types.ResponseUpdateZero)
	}

	// 2. Increment the team's complete count.
	cnt, err = r.dao.UpdateAddCompleteCount(ctx, tx, team.TeamId)
	if err != nil {
		return nil, err
	}
	if cnt != 1 {
		return nil, types.NewAppError(types.ResponseUpdateZero)
	}

	// 3. If the target is met, form the team and record the callback task.
	//
	// Defect fix: the legacy version decided team completion from a pre-transaction
	// snapshot (targetCount-completeCount==1). When the last two members settled
	// concurrently, both read the stale completeCount, neither triggered completion,
	// and the team stayed at status=0. Here we re-read the latest counts inside the
	// transaction (holding the row lock) before deciding.
	current, err := r.dao.QueryGroupBuyTeamByTeamId(ctx, tx, team.TeamId)
	if err != nil {
		return nil, err
	}
	var notifyTaskEntity *NotifyTaskEntity
	if current.Status == int(types.GroupBuyOrderProgress) && current.CompleteCount >= current.TargetCount {
		cnt, err = r.dao.UpdateOrderStatus2COMPLETE(ctx, tx, team.TeamId)
		if err != nil {
			return nil, err
		}
		if cnt != 1 {
			return nil, types.NewAppError(types.ResponseUpdateZero)
		}
		outTradeNoList, err := r.dao.QueryGroupBuyCompleteOrderOutTradeNoListByTeamId(ctx, tx, team.TeamId)
		if err != nil {
			return nil, err
		}
		parameterJson, err := json.Marshal(TeamSuccessMessage{
			Result:         "success",
			TeamId:         team.TeamId,
			OutTradeNoList: outTradeNoList,
		})
		if err != nil {
			return nil, err
		}
		notifyTask := &dao.NotifyTask{
			ActivityId:    team.ActivityId,
			TeamId:        team.TeamId,
			NotifyType:    string(notifyConfig.NotifyType),
			NotifyMQ:      nullStringIfMQ(notifyConfig.NotifyType, notifyConfig.NotifyMQ),
			NotifyUrl:     nullStringIfHTTP(notifyConfig.NotifyType, notifyConfig.NotifyUrl),
			NotifyCount:   0,
			NotifyStatus:  0,
			ParameterJson: string(parameterJson),
		}
		id, err := r.dao.InsertNotifyTask(ctx, tx, notifyTask)
		if err != nil {
			return nil, err
		}
		notifyTaskEntity = &NotifyTaskEntity{
			Id:            id,
			TeamId:        notifyTask.TeamId,
			NotifyType:    types.NotifyTypeOf(notifyTask.NotifyType),
			NotifyMQ:      notifyTask.NotifyMQ.String,
			NotifyUrl:     notifyTask.NotifyUrl.String,
			NotifyCount:   notifyTask.NotifyCount,
			ParameterJson: notifyTask.ParameterJson,
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return notifyTaskEntity, nil
}

// IsSCBlackIntercept mirrors the DCC-backed blacklist check.
func (r *TradeRepository) IsSCBlackIntercept(source, channel string) bool {
	return r.dcc.IsSCBlackIntercept(source, channel)
}

// QueryUnExecutedNotifyTaskList mirrors the legacy query (status 0 or 2, limit 50).
func (r *TradeRepository) QueryUnExecutedNotifyTaskList(ctx context.Context) ([]*NotifyTaskEntity, error) {
	tasks, err := r.dao.QueryUnExecutedNotifyTaskList(ctx, r.db, 50)
	if err != nil {
		return nil, err
	}
	return toNotifyTaskEntities(tasks), nil
}

// QueryUnExecutedNotifyTaskListByTeamId mirrors the legacy team-scoped query.
func (r *TradeRepository) QueryUnExecutedNotifyTaskListByTeamId(ctx context.Context, teamId string) ([]*NotifyTaskEntity, error) {
	task, err := r.dao.QueryUnExecutedNotifyTaskByTeamId(ctx, r.db, teamId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return toNotifyTaskEntities([]*dao.NotifyTask{task}), nil
}

func toNotifyTaskEntities(tasks []*dao.NotifyTask) []*NotifyTaskEntity {
	out := make([]*NotifyTaskEntity, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, &NotifyTaskEntity{
			Id:            t.Id,
			TeamId:        t.TeamId,
			NotifyType:    types.NotifyTypeOf(t.NotifyType),
			NotifyMQ:      t.NotifyMQ.String,
			NotifyUrl:     t.NotifyUrl.String,
			NotifyCount:   t.NotifyCount,
			ParameterJson: t.ParameterJson,
		})
	}
	return out
}

// UpdateNotifyTaskStatusSuccess marks a task complete by id.
func (r *TradeRepository) UpdateNotifyTaskStatusSuccess(ctx context.Context, id int64) (int, error) {
	n, err := r.dao.UpdateNotifyTaskStatusSuccess(ctx, r.db, id)
	return int(n), err
}

// UpdateNotifyTaskStatusError marks a task failed by id.
func (r *TradeRepository) UpdateNotifyTaskStatusError(ctx context.Context, id int64) (int, error) {
	n, err := r.dao.UpdateNotifyTaskStatusError(ctx, r.db, id)
	return int(n), err
}

// UpdateNotifyTaskStatusRetry marks a task for retry by id.
func (r *TradeRepository) UpdateNotifyTaskStatusRetry(ctx context.Context, id int64) (int, error) {
	n, err := r.dao.UpdateNotifyTaskStatusRetry(ctx, r.db, id)
	return int(n), err
}

// OccupyTeamStock mirrors the legacy Redis stock occupation, with two defect fixes.
//
// Invariant: occupy = incr + 1. The +1 accounts for the team leader's seat, which
// is recorded via lock_count=1 in MySQL and never counted in the Redis teamStockKey.
// Every subsequent joiner runs through this counter, so incr is the number of
// joiners and occupy is the total team size including the leader. This is NOT an
// off-by-one; do not "fix" it by dropping the +1.
//
// Defect fix 1 (legacy: rejected attempts never rolled back their INCR, permanently
// inflating the counter so a later legitimate join could be refused): when the
// capacity check fails we now DECR the counter to undo our speculative increment,
// keeping it accurate.
//
// Defect fix 2 (legacy: teamStockKey/recoveryTeamStockKey had no TTL and grew
// without bound): we now set a TTL of validTime+60 minutes, refreshed on each call,
// so the counters expire shortly after the team's join window closes. Expiry is
// safe because joins past the window are already rejected by the activity time
// check (E0102), so a reset counter cannot cause overselling.
func (r *TradeRepository) OccupyTeamStock(ctx context.Context, teamStockKey, recoveryTeamStockKey string, target, validTime int) (bool, error) {
	recoveryCount, err := r.redis.GetAtomicLong(ctx, recoveryTeamStockKey)
	if err != nil {
		return false, err
	}
	ttl := time.Duration(validTime+60) * time.Minute
	// Bound the recovery counter's lifetime too (best effort; ignore result).
	if _, err := r.redis.Expire(ctx, recoveryTeamStockKey, ttl); err != nil {
		log.Printf("failed to set TTL on recovery stock key %s: %v", recoveryTeamStockKey, err)
	}
	incr, err := r.redis.Incr(ctx, teamStockKey)
	if err != nil {
		return false, err
	}
	// Keep the counter alive for the team's join window (best effort).
	if _, err := r.redis.Expire(ctx, teamStockKey, ttl); err != nil {
		log.Printf("failed to set TTL on team stock key %s: %v", teamStockKey, err)
	}
	occupy := incr + 1
	if occupy > int64(target)+recoveryCount {
		// Capacity exceeded: roll back our speculative increment so the counter
		// does not stay inflated for subsequent joiners.
		if _, decErr := r.redis.Decr(ctx, teamStockKey); decErr != nil {
			log.Printf("failed to roll back team stock key %s: %v", teamStockKey, decErr)
		}
		return false, nil
	}
	lockKey := teamStockKey + types.Underline + strconv.FormatInt(occupy, 10)
	lock, err := r.redis.SetNx(ctx, lockKey, ttl)
	if err != nil {
		return false, err
	}
	if !lock {
		log.Printf("team stock lock acquisition failed %s", lockKey)
	}
	return lock, nil
}

// RecoveryTeamStock mirrors the legacy failure-recovery counter increment.
func (r *TradeRepository) RecoveryTeamStock(ctx context.Context, recoveryTeamStockKey string, validTime int) error {
	if types.IsBlank(recoveryTeamStockKey) {
		return nil
	}
	_, err := r.redis.Incr(ctx, recoveryTeamStockKey)
	return err
}

// Refund2AddRecovery mirrors the legacy idempotent stock recovery, guarded by a
// per-order lock so a redelivered message cannot double-count.
func (r *TradeRepository) Refund2AddRecovery(ctx context.Context, recoveryTeamStockKey, orderId string) error {
	if types.IsBlank(recoveryTeamStockKey) || types.IsBlank(orderId) {
		return nil
	}
	lockKey := refundLockKey(orderId)
	acquired, err := r.redis.SetNx(ctx, lockKey, 30*24*time.Hour)
	if err != nil {
		return err
	}
	if !acquired {
		log.Printf("[trade] stock recovery for order %s is already in progress, skipping duplicate", orderId)
		return nil
	}
	if _, err := r.redis.Incr(ctx, recoveryTeamStockKey); err != nil {
		// Release the lock so the message can be redelivered and recover the stock.
		if rmErr := r.redis.Remove(ctx, lockKey); rmErr != nil {
			log.Printf("[trade] failed to release the lock after a stock-recovery error for order %s: %v", orderId, rmErr)
		}
		return err
	}
	log.Printf("[trade] stock recovered for order %s, recovery key: %s", orderId, recoveryTeamStockKey)
	return nil
}

// ---- refund operations ----

// Unpaid2Refund mirrors the legacy transactional unpaid refund.
func (r *TradeRepository) Unpaid2Refund(ctx context.Context, agg *GroupBuyRefundAggregate) (*NotifyTaskEntity, error) {
	order := agg.TradeRefundOrderEntity
	progress := agg.GroupBuyProgress
	return r.runRefund(ctx, order, types.RefundTypeUnpaidUnlock, types.TaskNotifyTradeUnpaid2Refund, func(tx *gorm.DB) error {
		cnt, err := r.dao.UpdateOrderListUnpaid2Refund(ctx, tx, order.UserId, order.OrderId)
		if err != nil {
			return err
		}
		if cnt != 1 {
			log.Printf("[trade] reverse flow - unpaid2Refund failed to update order status (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
			return types.NewAppError(types.ResponseUpdateZero)
		}
		cnt, err = r.dao.UpdateOrderUnpaid2Refund(ctx, tx, order.TeamId, progress.LockCount)
		if err != nil {
			return err
		}
		if cnt != 1 {
			log.Printf("[trade] reverse flow - unpaid2Refund failed to update team record (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
			return types.NewAppError(types.ResponseUpdateZero)
		}
		return nil
	})
}

// Paid2Refund mirrors the legacy transactional paid (unformed team) refund.
func (r *TradeRepository) Paid2Refund(ctx context.Context, agg *GroupBuyRefundAggregate) (*NotifyTaskEntity, error) {
	order := agg.TradeRefundOrderEntity
	progress := agg.GroupBuyProgress
	return r.runRefund(ctx, order, types.RefundTypePaidUnformed, types.TaskNotifyTradePaid2Refund, func(tx *gorm.DB) error {
		cnt, err := r.dao.UpdateOrderListPaid2Refund(ctx, tx, order.UserId, order.OrderId)
		if err != nil {
			return err
		}
		if cnt != 1 {
			log.Printf("[trade] reverse flow - paid2Refund failed to update order status (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
			return types.NewAppError(types.ResponseUpdateZero)
		}
		cnt, err = r.dao.UpdateOrderPaid2Refund(ctx, tx, order.TeamId, progress.LockCount, progress.CompleteCount)
		if err != nil {
			return err
		}
		if cnt != 1 {
			log.Printf("[trade] reverse flow - paid2Refund failed to update team record (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
			return types.NewAppError(types.ResponseUpdateZero)
		}
		return nil
	})
}

// PaidTeam2Refund mirrors the legacy transactional paid (formed team) refund. The
// team status transition (complete-with-refund vs failed) depends on whether the
// last member refunded.
func (r *TradeRepository) PaidTeam2Refund(ctx context.Context, agg *GroupBuyRefundAggregate) (*NotifyTaskEntity, error) {
	order := agg.TradeRefundOrderEntity
	progress := agg.GroupBuyProgress
	teamStatus := agg.GroupBuyOrderStatus
	return r.runRefund(ctx, order, types.RefundTypePaidFormed, types.TaskNotifyTradePaidTeam2Refund, func(tx *gorm.DB) error {
		cnt, err := r.dao.UpdateOrderListPaidTeam2Refund(ctx, tx, order.UserId, order.OrderId)
		if err != nil {
			return err
		}
		if cnt != 1 {
			log.Printf("[trade] reverse flow - paidTeam2Refund failed to update order status (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
			return types.NewAppError(types.ResponseUpdateZero)
		}
		switch teamStatus {
		case types.GroupBuyOrderComplete:
			cnt, err = r.dao.UpdateOrderPaidTeam2Refund(ctx, tx, order.TeamId, progress.LockCount, progress.CompleteCount)
			if err != nil {
				return err
			}
			if cnt != 1 {
				log.Printf("[trade] reverse flow - paidTeam2Refund failed to update team record (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
				return types.NewAppError(types.ResponseUpdateZero)
			}
		case types.GroupBuyOrderFail:
			cnt, err = r.dao.UpdateOrderPaidTeam2RefundFail(ctx, tx, order.TeamId, progress.LockCount, progress.CompleteCount)
			if err != nil {
				return err
			}
			if cnt != 1 {
				log.Printf("[trade] reverse flow - updateTeamPaid2RefundFail failed to update team record (refund) userId:%s orderId:%s", order.UserId, order.OrderId)
				return types.NewAppError(types.ResponseUpdateZero)
			}
		}
		return nil
	})
}

// runRefund executes the refund DB mutations in a transaction, then writes the
// team-refund notify task and returns it.
func (r *TradeRepository) runRefund(ctx context.Context, order *TradeRefundOrderEntity, refundType types.RefundType, category types.TaskNotifyCategory, mutate func(tx *gorm.DB) error) (*NotifyTaskEntity, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() //nolint:errcheck

	if err := mutate(tx); err != nil {
		return nil, err
	}

	parameterJson, err := json.Marshal(TeamRefundSuccess{
		Type:       string(refundType),
		UserId:     order.UserId,
		TeamId:     order.TeamId,
		ActivityId: order.ActivityId,
		OrderId:    order.OrderId,
	})
	if err != nil {
		return nil, err
	}
	notifyTask := &dao.NotifyTask{
		ActivityId:     order.ActivityId,
		TeamId:         order.TeamId,
		NotifyCategory: sql.NullString{String: string(category), Valid: true},
		NotifyType:     string(types.NotifyTypeMQ),
		NotifyMQ:       sql.NullString{String: r.topicTeamRefund, Valid: true},
		NotifyCount:    0,
		NotifyStatus:   0,
		ParameterJson:  string(parameterJson),
	}
	id, err := r.dao.InsertNotifyTask(ctx, tx, notifyTask)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &NotifyTaskEntity{
		Id:            id,
		TeamId:        notifyTask.TeamId,
		NotifyType:    types.NotifyTypeMQ,
		NotifyMQ:      notifyTask.NotifyMQ.String,
		NotifyCount:   notifyTask.NotifyCount,
		ParameterJson: notifyTask.ParameterJson,
	}, nil
}

// ---- helpers ----

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullStringIfMQ(t types.NotifyType, mq string) sql.NullString {
	if t == types.NotifyTypeMQ {
		return nullString(mq)
	}
	return sql.NullString{}
}

func nullStringIfHTTP(t types.NotifyType, url string) sql.NullString {
	if t == types.NotifyTypeHTTP {
		return nullString(url)
	}
	return sql.NullString{}
}
