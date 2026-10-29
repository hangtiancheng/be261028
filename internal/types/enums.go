package types

import "fmt"

// ---- ActivityStatusEnumVO ----

// ActivityStatus mirrors com.buy.group.easy.types.enums.ActivityStatusEnumVO.
type ActivityStatus int

const (
	ActivityStatusCreate    ActivityStatus = 0 // 创建
	ActivityStatusEffective ActivityStatus = 1 // 生效
	ActivityStatusOverdue   ActivityStatus = 2 // 过期
	ActivityStatusAbandoned ActivityStatus = 3 // 废弃
)

func (s ActivityStatus) Info() string {
	switch s {
	case ActivityStatusCreate:
		return "创建"
	case ActivityStatusEffective:
		return "生效"
	case ActivityStatusOverdue:
		return "过期"
	case ActivityStatusAbandoned:
		return "废弃"
	}
	return "未知"
}

// ActivityStatusOf maps a raw db code to an ActivityStatus. Unknown codes fall
// back to Create rather than panicking (the Java valueOf threw a RuntimeException).
func ActivityStatusOf(code int) ActivityStatus {
	switch ActivityStatus(code) {
	case ActivityStatusCreate, ActivityStatusEffective, ActivityStatusOverdue, ActivityStatusAbandoned:
		return ActivityStatus(code)
	}
	return ActivityStatusCreate
}

// ---- GroupBuyOrderEnumVO ----

// GroupBuyOrderStatus mirrors com.buy.group.easy.types.enums.GroupBuyOrderEnumVO.
type GroupBuyOrderStatus int

const (
	GroupBuyOrderProgress GroupBuyOrderStatus = 0 // 拼单中
	GroupBuyOrderComplete GroupBuyOrderStatus = 1 // 完成
	GroupBuyOrderFail     GroupBuyOrderStatus = 2 // 失败
	// GroupBuyOrderCompleteWithRefund is the db-only state (status = 3) written by
	// paidTeam2Refund meaning "完成含退单". The legacy GroupBuyOrderEnumVO.valueOf
	// threw on code 3, which crashed any later read of such a team. We map it to
	// Complete so the team is still treated as formed for subsequent refunds.
	GroupBuyOrderCompleteWithRefund GroupBuyOrderStatus = 3
)

func (s GroupBuyOrderStatus) Info() string {
	switch s {
	case GroupBuyOrderProgress:
		return "拼单中"
	case GroupBuyOrderComplete, GroupBuyOrderCompleteWithRefund:
		return "完成"
	case GroupBuyOrderFail:
		return "失败"
	}
	return "未知"
}

// GroupBuyOrderStatusOf maps a raw db code to a GroupBuyOrderStatus. Code 3
// (完成含退单) is normalised to Complete; unknown codes fall back to Progress.
func GroupBuyOrderStatusOf(code int) GroupBuyOrderStatus {
	switch GroupBuyOrderStatus(code) {
	case GroupBuyOrderProgress:
		return GroupBuyOrderProgress
	case GroupBuyOrderComplete, GroupBuyOrderCompleteWithRefund:
		return GroupBuyOrderComplete
	case GroupBuyOrderFail:
		return GroupBuyOrderFail
	}
	return GroupBuyOrderProgress
}

// ---- TradeOrderStatusEnumVO ----

// TradeOrderStatus mirrors com.buy.group.easy.domain.trade.model.valobj.TradeOrderStatusEnumVO.
type TradeOrderStatus int

const (
	TradeOrderCreate   TradeOrderStatus = 0 // 初始创建/锁定
	TradeOrderComplete TradeOrderStatus = 1 // 消费完成
	TradeOrderClose    TradeOrderStatus = 2 // 超时关单/退单
)

func (s TradeOrderStatus) Info() string {
	switch s {
	case TradeOrderCreate:
		return "初始创建"
	case TradeOrderComplete:
		return "消费完成"
	case TradeOrderClose:
		return "超时关单"
	}
	return "初始创建"
}

// TradeOrderStatusOf mirrors TradeOrderStatusEnumVO.valueOf: unknown codes fall
// back to Create.
func TradeOrderStatusOf(code int) TradeOrderStatus {
	switch TradeOrderStatus(code) {
	case TradeOrderCreate, TradeOrderComplete, TradeOrderClose:
		return TradeOrderStatus(code)
	}
	return TradeOrderCreate
}

// ---- NotifyTypeEnumVO ----

// NotifyType mirrors com.buy.group.easy.types.enums.NotifyTypeEnumVO.
type NotifyType string

const (
	NotifyTypeHTTP NotifyType = "HTTP"
	NotifyTypeMQ   NotifyType = "MQ"
)

func (t NotifyType) Info() string {
	switch t {
	case NotifyTypeHTTP:
		return "HTTP 回调"
	case NotifyTypeMQ:
		return "MQ 消息通知"
	}
	return ""
}

// NotifyTypeOf maps a raw string to a NotifyType, defaulting to HTTP for unknown
// values so reads never fail (the Java enum valueOf would throw).
func NotifyTypeOf(code string) NotifyType {
	switch NotifyType(code) {
	case NotifyTypeHTTP:
		return NotifyTypeHTTP
	case NotifyTypeMQ:
		return NotifyTypeMQ
	}
	return NotifyTypeHTTP
}

// ---- NotifyTaskHTTPEnumVO ----

// NotifyTaskResult mirrors com.buy.group.easy.types.enums.NotifyTaskHTTPEnumVO.
// The NULL variant (java code == null) is represented by the empty string.
type NotifyTaskResult string

const (
	NotifyTaskSuccess NotifyTaskResult = "success"
	NotifyTaskError   NotifyTaskResult = "error"
	NotifyTaskNull    NotifyTaskResult = ""
)

// ---- DiscountTypeEnum ----

// DiscountType mirrors com.buy.group.easy.domain.activity.model.valobj.DiscountTypeEnum.
type DiscountType int

const (
	DiscountTypeBase DiscountType = 0 // 基础优惠
	DiscountTypeTag  DiscountType = 1 // 人群标签
)

func (d DiscountType) Info() string {
	switch d {
	case DiscountTypeBase:
		return "基础优惠"
	case DiscountTypeTag:
		return "人群标签"
	}
	return "未知"
}

// DiscountTypeOf maps a raw code, defaulting to Base for unknown values.
func DiscountTypeOf(code int) DiscountType {
	switch DiscountType(code) {
	case DiscountTypeBase, DiscountTypeTag:
		return DiscountType(code)
	}
	return DiscountTypeBase
}

// ---- TaskNotifyCategoryEnumVO ----

// TaskNotifyCategory mirrors com.buy.group.easy.domain.trade.model.valobj.TaskNotifyCategoryEnumVO.
type TaskNotifyCategory string

const (
	TaskNotifyTradeSettlement      TaskNotifyCategory = "trade_settlement"
	TaskNotifyTradeUnpaid2Refund   TaskNotifyCategory = "trade_unpaid2refund"
	TaskNotifyTradePaid2Refund     TaskNotifyCategory = "trade_paid2refund"
	TaskNotifyTradePaidTeam2Refund TaskNotifyCategory = "trade_paid_team2refund"
)

func (c TaskNotifyCategory) Info() string {
	switch c {
	case TaskNotifyTradeSettlement:
		return "交易结算"
	case TaskNotifyTradeUnpaid2Refund:
		return "交易退单-未支付&未成团"
	case TaskNotifyTradePaid2Refund:
		return "交易退单-已支付&未成团"
	case TaskNotifyTradePaidTeam2Refund:
		return "交易退单-已支付&已成团"
	}
	return ""
}

// ---- TradeRefundBehaviorEnum ----

// RefundBehavior mirrors TradeRefundBehaviorEntity.TradeRefundBehaviorEnum.
type RefundBehavior string

const (
	RefundBehaviorSuccess RefundBehavior = "success"
	RefundBehaviorRepeat  RefundBehavior = "repeat"
	RefundBehaviorFail    RefundBehavior = "fail"
)

func (b RefundBehavior) Info() string {
	switch b {
	case RefundBehaviorSuccess:
		return "成功"
	case RefundBehaviorRepeat:
		return "重复"
	case RefundBehaviorFail:
		return "失败"
	}
	return ""
}

// ---- RefundTypeEnumVO ----

// RefundType mirrors com.buy.group.easy.domain.trade.model.valobj.RefundTypeEnumVO.
type RefundType string

const (
	RefundTypeUnpaidUnlock RefundType = "unpaid_unlock"
	RefundTypePaidUnformed RefundType = "paid_unformed"
	RefundTypePaidFormed   RefundType = "paid_formed"
)

// Strategy returns the strategy bean name for the refund type, matching the Java
// enum's strategy field.
func (r RefundType) Strategy() string {
	switch r {
	case RefundTypeUnpaidUnlock:
		return "unpaid2RefundStrategy"
	case RefundTypePaidUnformed:
		return "paid2RefundStrategy"
	case RefundTypePaidFormed:
		return "paidTeam2RefundStrategy"
	}
	return ""
}

func (r RefundType) Info() string {
	switch r {
	case RefundTypeUnpaidUnlock:
		return "未支付，未成团"
	case RefundTypePaidUnformed:
		return "已支付，未成团"
	case RefundTypePaidFormed:
		return "已支付，已成团"
	}
	return ""
}

// Matches mirrors each enum constant's matches(groupBuyOrderStatus, tradeOrderStatus).
func (r RefundType) Matches(team GroupBuyOrderStatus, order TradeOrderStatus) bool {
	switch r {
	case RefundTypeUnpaidUnlock:
		return team == GroupBuyOrderProgress && order == TradeOrderCreate
	case RefundTypePaidUnformed:
		return team == GroupBuyOrderProgress && order == TradeOrderComplete
	case RefundTypePaidFormed:
		// 完成、完成含退单，都做此处理
		return team == GroupBuyOrderComplete && order == TradeOrderComplete
	}
	return false
}

// refundStrategyOrder is the iteration order used by getRefundStrategy (the Java
// enum values() order).
var refundStrategyOrder = []RefundType{RefundTypeUnpaidUnlock, RefundTypePaidUnformed, RefundTypePaidFormed}

// GetRefundStrategy mirrors RefundTypeEnumVO.getRefundStrategy. It returns an
// error for unsupported combinations instead of throwing a RuntimeException.
func GetRefundStrategy(team GroupBuyOrderStatus, order TradeOrderStatus) (RefundType, error) {
	for _, rt := range refundStrategyOrder {
		if rt.Matches(team, order) {
			return rt, nil
		}
	}
	return "", fmt.Errorf("不支持的退款状态组合: groupBuyOrderStatus=%d, tradeOrderStatus=%d", team, order)
}

// GetRefundTypeByCode mirrors RefundTypeEnumVO.getRefundTypeEnumVOByCode.
func GetRefundTypeByCode(code string) (RefundType, error) {
	switch RefundType(code) {
	case RefundTypeUnpaidUnlock, RefundTypePaidUnformed, RefundTypePaidFormed:
		return RefundType(code), nil
	}
	return "", fmt.Errorf("退单类型枚举值不存在: %s", code)
}
