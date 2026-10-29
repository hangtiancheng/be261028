package dao

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ---- group_buy_order ----

// InsertGroupBuyOrder mirrors group_buy_order_mapper insert (status hardcoded 0,
// create/update via the database clock).
func (d *DAO) InsertGroupBuyOrder(ctx context.Context, tx *gorm.DB, o *GroupBuyOrder) error {
	o.Status = 0
	return checkAffected(tx.WithContext(ctx).Create(o), 1)
}

// UpdateAddLockCount mirrors updateAddLockCount: lock_count+1 while below target.
func (d *DAO) UpdateAddLockCount(ctx context.Context, tx *gorm.DB, teamId string) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND lock_count < target_count", teamId).
		Updates(map[string]any{
			"lock_count":  gorm.Expr("lock_count + 1"),
			"update_time": gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// UpdateAddCompleteCount mirrors updateAddCompleteCount.
func (d *DAO) UpdateAddCompleteCount(ctx context.Context, tx *gorm.DB, teamId string) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND complete_count < target_count", teamId).
		Updates(map[string]any{
			"complete_count": gorm.Expr("complete_count + 1"),
			"update_time":    gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// UpdateOrderStatus2COMPLETE mirrors updateOrderStatus2COMPLETE (team -> Completed).
func (d *DAO) UpdateOrderStatus2COMPLETE(ctx context.Context, tx *gorm.DB, teamId string) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND status = 0", teamId).
		Updates(map[string]any{
			"status":      1,
			"update_time": gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// QueryGroupBuyProgress mirrors queryGroupBuyProgress.
func (d *DAO) QueryGroupBuyProgress(ctx context.Context, tx *gorm.DB, teamId string) (*GroupBuyOrder, error) {
	var o GroupBuyOrder
	err := tx.WithContext(ctx).
		Select("target_count, complete_count, lock_count").
		Where("team_id = ?", teamId).
		Take(&o).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &o, nil
}

// QueryGroupBuyTeamByTeamId mirrors queryGroupBuyTeamByTeamId.
func (d *DAO) QueryGroupBuyTeamByTeamId(ctx context.Context, tx *gorm.DB, teamId string) (*GroupBuyOrder, error) {
	var o GroupBuyOrder
	err := tx.WithContext(ctx).
		Select("team_id, activity_id, target_count, complete_count, lock_count, status, valid_start_time, valid_end_time, notify_type, notify_url").
		Where("team_id = ?", teamId).
		Take(&o).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &o, nil
}

// QueryGroupBuyProgressByTeamIds mirrors queryGroupBuyProgressByTeamIds: only
// in-progress, unfilled, unexpired teams.
func (d *DAO) QueryGroupBuyProgressByTeamIds(ctx context.Context, tx *gorm.DB, teamIds []string) ([]*GroupBuyOrder, error) {
	if len(teamIds) == 0 {
		return nil, nil
	}
	var list []*GroupBuyOrder
	err := tx.WithContext(ctx).
		Select("team_id, activity_id, target_count, complete_count, lock_count, status, valid_start_time, valid_end_time, notify_type, notify_url").
		Where("status = 0 AND target_count > lock_count AND valid_end_time > now()").
		Where("team_id IN ?", teamIds).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// QueryGroupBuyOrdersByActivityId mirrors queryInProgressUserGroupBuyOrderDetailListByActivityId
// (select * from group_buy_order where activity_id = ?), used for team statistics.
func (d *DAO) QueryGroupBuyOrdersByActivityId(ctx context.Context, tx *gorm.DB, activityId int64) ([]*GroupBuyOrder, error) {
	var list []*GroupBuyOrder
	if err := tx.WithContext(ctx).Where("activity_id = ?", activityId).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateOrderUnpaid2Refund mirrors group_buy_order unpaid2Refund: lock_count += delta
// (delta is negative) while the team is in progress.
func (d *DAO) UpdateOrderUnpaid2Refund(ctx context.Context, tx *gorm.DB, teamId string, lockCountDelta int) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND status = 0", teamId).
		Updates(map[string]any{
			"lock_count":  gorm.Expr("lock_count + ?", lockCountDelta),
			"update_time": gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// UpdateOrderPaid2Refund mirrors group_buy_order paid2Refund.
func (d *DAO) UpdateOrderPaid2Refund(ctx context.Context, tx *gorm.DB, teamId string, lockCountDelta, completeCountDelta int) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND status = 0", teamId).
		Updates(map[string]any{
			"lock_count":     gorm.Expr("lock_count + ?", lockCountDelta),
			"complete_count": gorm.Expr("complete_count + ?", completeCountDelta),
			"update_time":    gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// UpdateOrderPaidTeam2Refund mirrors group_buy_order paidTeam2Refund: status -> 3
// (Completed with refunds) when more than one member remains.
func (d *DAO) UpdateOrderPaidTeam2Refund(ctx context.Context, tx *gorm.DB, teamId string, lockCountDelta, completeCountDelta int) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND (status = 1 OR status = 3) AND complete_count > 1", teamId).
		Updates(map[string]any{
			"lock_count":     gorm.Expr("lock_count + ?", lockCountDelta),
			"complete_count": gorm.Expr("complete_count + ?", completeCountDelta),
			"status":         3,
			"update_time":    gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// UpdateOrderPaidTeam2RefundFail mirrors group_buy_order paidTeam2RefundFail:
// status -> 2 (Failed) when the last member refunds.
func (d *DAO) UpdateOrderPaidTeam2RefundFail(ctx context.Context, tx *gorm.DB, teamId string, lockCountDelta, completeCountDelta int) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrder{}).
		Where("team_id = ? AND (status = 1 OR status = 3) AND complete_count = 1", teamId).
		Updates(map[string]any{
			"lock_count":     gorm.Expr("lock_count + ?", lockCountDelta),
			"complete_count": gorm.Expr("complete_count + ?", completeCountDelta),
			"status":         2,
			"update_time":    gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// ---- group_buy_order_list ----

// InsertGroupBuyOrderList mirrors group_buy_order_list_mapper insert.
func (d *DAO) InsertGroupBuyOrderList(ctx context.Context, tx *gorm.DB, o *GroupBuyOrderList) error {
	return checkAffected(tx.WithContext(ctx).Create(o), 1)
}

// QueryGroupBuyOrderRecordByOutTradeNo mirrors queryGroupBuyOrderRecordByOutTradeNo.
func (d *DAO) QueryGroupBuyOrderRecordByOutTradeNo(ctx context.Context, tx *gorm.DB, userId, outTradeNo string) (*GroupBuyOrderList, error) {
	var o GroupBuyOrderList
	err := tx.WithContext(ctx).
		Select("user_id, team_id, order_id, activity_id, start_time, end_time, goods_id, source, channel, original_price, deduction_price, pay_price, status").
		Where("out_trade_no = ? AND user_id = ?", outTradeNo, userId).
		Take(&o).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &o, nil
}

// QueryOrderCountByActivityId mirrors queryOrderCountByActivityId.
func (d *DAO) QueryOrderCountByActivityId(ctx context.Context, tx *gorm.DB, userId string, activityId int64) (int, error) {
	var count int64
	err := tx.WithContext(ctx).Model(&GroupBuyOrderList{}).
		Where("user_id = ? AND activity_id = ?", userId, activityId).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// UpdateOrderListStatus2COMPLETE mirrors group_buy_order_list updateOrderStatus2COMPLETE.
//
// Defect fix: the legacy SQL had no status guard, so a duplicated settlement
// callback re-matched the row and (combined with client-found-rows semantics)
// incremented the team's complete_count a second time. Only locked (status=0)
// orders may transition to complete; the settlement chain answers repeat
// callbacks idempotently before reaching this update.
func (d *DAO) UpdateOrderListStatus2COMPLETE(ctx context.Context, tx *gorm.DB, userId, outTradeNo string, outTradeTime time.Time) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrderList{}).
		Where("out_trade_no = ? AND user_id = ? AND status = 0", outTradeNo, userId).
		Updates(map[string]any{
			"status":         1,
			"out_trade_time": outTradeTime,
			"update_time":    gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// QueryGroupBuyCompleteOrderOutTradeNoListByTeamId mirrors the same mapper query.
func (d *DAO) QueryGroupBuyCompleteOrderOutTradeNoListByTeamId(ctx context.Context, tx *gorm.DB, teamId string) ([]string, error) {
	var list []string
	err := tx.WithContext(ctx).Model(&GroupBuyOrderList{}).
		Where("team_id = ? AND status = 1", teamId).
		Pluck("out_trade_no", &list).Error
	return list, err
}

// OrderListBrief is the projection returned by the in-progress detail queries.
type OrderListBrief struct {
	UserId     string `gorm:"column:user_id"`
	TeamId     string `gorm:"column:team_id"`
	OutTradeNo string `gorm:"column:out_trade_no"`
}

// QueryInProgressOrderDetailListByUserId mirrors queryInProgressUserGroupBuyOrderDetailListByUserId.
func (d *DAO) QueryInProgressOrderDetailListByUserId(ctx context.Context, tx *gorm.DB, activityId int64, userId string, count int) ([]*OrderListBrief, error) {
	var list []*OrderListBrief
	err := tx.WithContext(ctx).Model(&GroupBuyOrderList{}).
		Select("user_id, team_id, out_trade_no").
		Where("activity_id = ? AND user_id = ? AND status IN ? AND end_time > now()", activityId, userId, []int{0, 1}).
		Order("id DESC").
		Limit(count).
		Find(&list).Error
	return list, err
}

// QueryInProgressOrderDetailListByRandom mirrors queryInProgressUserGroupBuyOrderDetailListByRandom.
func (d *DAO) QueryInProgressOrderDetailListByRandom(ctx context.Context, tx *gorm.DB, activityId int64, userId string, count int) ([]*OrderListBrief, error) {
	var list []*OrderListBrief
	err := tx.WithContext(ctx).Model(&GroupBuyOrderList{}).
		Select("user_id, team_id, out_trade_no").
		Where("activity_id = ?", activityId).
		Where("team_id IN (SELECT team_id FROM group_buy_order WHERE activity_id = ? AND status = 0)", activityId).
		Where("user_id != ? AND status IN ? AND end_time > now()", userId, []int{0, 1}).
		Order("id DESC").
		Limit(count).
		Find(&list).Error
	return list, err
}

// UpdateOrderListUnpaid2Refund mirrors group_buy_order_list unpaid2Refund (0 -> 2).
func (d *DAO) UpdateOrderListUnpaid2Refund(ctx context.Context, tx *gorm.DB, userId, orderId string) (int64, error) {
	return d.updateOrderListStatus2Refund(ctx, tx, userId, orderId, 0)
}

// UpdateOrderListPaid2Refund mirrors group_buy_order_list paid2Refund (1 -> 2).
func (d *DAO) UpdateOrderListPaid2Refund(ctx context.Context, tx *gorm.DB, userId, orderId string) (int64, error) {
	return d.updateOrderListStatus2Refund(ctx, tx, userId, orderId, 1)
}

// UpdateOrderListPaidTeam2Refund mirrors group_buy_order_list paidTeam2Refund (1 -> 2).
func (d *DAO) UpdateOrderListPaidTeam2Refund(ctx context.Context, tx *gorm.DB, userId, orderId string) (int64, error) {
	return d.updateOrderListStatus2Refund(ctx, tx, userId, orderId, 1)
}

func (d *DAO) updateOrderListStatus2Refund(ctx context.Context, tx *gorm.DB, userId, orderId string, fromStatus int) (int64, error) {
	res := tx.WithContext(ctx).Model(&GroupBuyOrderList{}).
		Where("user_id = ? AND order_id = ? AND status = ?", userId, orderId, fromStatus).
		Updates(map[string]any{
			"status":      2,
			"update_time": gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}

// ---- notify_task ----
//
// Defect fix: the legacy schema put a UNIQUE key on notify_task.team_id, but a
// team legitimately produces several notify tasks over its lifetime (settlement
// success plus one or more refunds). The second insert violated the constraint
// and rolled back the whole refund transaction. The be261028 schema drops that
// unique key and tasks are addressed by their auto-increment id, so status
// updates target exactly one task.

// InsertNotifyTask mirrors notify_task_mapper insert and returns the new id.
func (d *DAO) InsertNotifyTask(ctx context.Context, tx *gorm.DB, t *NotifyTask) (int64, error) {
	if err := tx.WithContext(ctx).Create(t).Error; err != nil {
		return 0, err
	}
	return t.Id, nil
}

// QueryUnExecutedNotifyTaskList mirrors queryUnExecutedNotifyTaskList (status 0 or 2).
func (d *DAO) QueryUnExecutedNotifyTaskList(ctx context.Context, tx *gorm.DB, limit int) ([]*NotifyTask, error) {
	if limit <= 0 {
		limit = 50
	}
	var list []*NotifyTask
	err := tx.WithContext(ctx).
		Where("notify_status IN ?", []int{0, 2}).
		Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// QueryUnExecutedNotifyTaskByTeamId mirrors queryUnExecutedNotifyTaskByTeamId,
// returning the most recent pending task for a team (kept for parity although the
// legacy team-scoped execNotifyJob had no production caller).
func (d *DAO) QueryUnExecutedNotifyTaskByTeamId(ctx context.Context, tx *gorm.DB, teamId string) (*NotifyTask, error) {
	var t NotifyTask
	err := tx.WithContext(ctx).
		Where("team_id = ? AND notify_status IN ?", teamId, []int{0, 2}).
		Order("id DESC").
		Take(&t).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &t, nil
}

// UpdateNotifyTaskStatusSuccess marks a task complete (status 1) by id.
func (d *DAO) UpdateNotifyTaskStatusSuccess(ctx context.Context, tx *gorm.DB, id int64) (int64, error) {
	return d.updateNotifyTaskStatus(ctx, tx, id, 1)
}

// UpdateNotifyTaskStatusError marks a task failed (status 3) by id.
func (d *DAO) UpdateNotifyTaskStatusError(ctx context.Context, tx *gorm.DB, id int64) (int64, error) {
	return d.updateNotifyTaskStatus(ctx, tx, id, 3)
}

// UpdateNotifyTaskStatusRetry marks a task for retry (status 2) by id.
func (d *DAO) UpdateNotifyTaskStatusRetry(ctx context.Context, tx *gorm.DB, id int64) (int64, error) {
	return d.updateNotifyTaskStatus(ctx, tx, id, 2)
}

func (d *DAO) updateNotifyTaskStatus(ctx context.Context, tx *gorm.DB, id int64, status int) (int64, error) {
	res := tx.WithContext(ctx).Model(&NotifyTask{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"notify_count":  gorm.Expr("notify_count + 1"),
			"notify_status": status,
			"update_time":   gorm.Expr("now()"),
		})
	return res.RowsAffected, res.Error
}
