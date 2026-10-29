package types

import "fmt"

// ---- ActivityStatusEnumVO ----

// ActivityStatus mirrors com.buy.group.easy.types.enums.ActivityStatusEnumVO.
type ActivityStatus int

const (
	ActivityStatusCreate    ActivityStatus = 0 // created
	ActivityStatusEffective ActivityStatus = 1 // effective
	ActivityStatusOverdue   ActivityStatus = 2 // overdue
	ActivityStatusAbandoned ActivityStatus = 3 // abandoned
)

func (s ActivityStatus) Info() string {
	switch s {
	case ActivityStatusCreate:
		return "Created"
	case ActivityStatusEffective:
		return "Effective"
	case ActivityStatusOverdue:
		return "Overdue"
	case ActivityStatusAbandoned:
		return "Abandoned"
	}
	return "Unknown"
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
	GroupBuyOrderProgress GroupBuyOrderStatus = 0 // in progress
	GroupBuyOrderComplete GroupBuyOrderStatus = 1 // complete
	GroupBuyOrderFail     GroupBuyOrderStatus = 2 // failed
	// GroupBuyOrderCompleteWithRefund is the db-only state (status = 3) written by
	// paidTeam2Refund meaning "completed with refund". The legacy
	// GroupBuyOrderEnumVO.valueOf threw on code 3, which crashed any later read of
	// such a team. We map it to Complete so the team is still treated as formed for
	// subsequent refunds.
	GroupBuyOrderCompleteWithRefund GroupBuyOrderStatus = 3
)

func (s GroupBuyOrderStatus) Info() string {
	switch s {
	case GroupBuyOrderProgress:
		return "In progress"
	case GroupBuyOrderComplete, GroupBuyOrderCompleteWithRefund:
		return "Completed"
	case GroupBuyOrderFail:
		return "Failed"
	}
	return "Unknown"
}

// GroupBuyOrderStatusOf maps a raw db code to a GroupBuyOrderStatus. Code 3
// (completed with refund) is normalised to Complete; unknown codes fall back to
// Progress.
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
	TradeOrderCreate   TradeOrderStatus = 0 // initial create / locked
	TradeOrderComplete TradeOrderStatus = 1 // consumption complete
	TradeOrderClose    TradeOrderStatus = 2 // closed by timeout or refund
)

func (s TradeOrderStatus) Info() string {
	switch s {
	case TradeOrderCreate:
		return "Created"
	case TradeOrderComplete:
		return "Completed"
	case TradeOrderClose:
		return "Closed"
	}
	return "Created"
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
		return "HTTP callback"
	case NotifyTypeMQ:
		return "MQ message notification"
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
	DiscountTypeBase DiscountType = 0 // base discount
	DiscountTypeTag  DiscountType = 1 // crowd tag
)

func (d DiscountType) Info() string {
	switch d {
	case DiscountTypeBase:
		return "Base discount"
	case DiscountTypeTag:
		return "Crowd tag"
	}
	return "Unknown"
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
		return "Trade settlement"
	case TaskNotifyTradeUnpaid2Refund:
		return "Trade refund - unpaid, team not formed"
	case TaskNotifyTradePaid2Refund:
		return "Trade refund - paid, team not formed"
	case TaskNotifyTradePaidTeam2Refund:
		return "Trade refund - paid, team formed"
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
		return "Success"
	case RefundBehaviorRepeat:
		return "Repeat"
	case RefundBehaviorFail:
		return "Fail"
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
		return "Unpaid, team not formed"
	case RefundTypePaidUnformed:
		return "Paid, team not formed"
	case RefundTypePaidFormed:
		return "Paid, team formed"
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
		// Both "complete" and "complete with refund" are handled this way.
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
	return "", fmt.Errorf("unsupported refund status combination: groupBuyOrderStatus=%d, tradeOrderStatus=%d", team, order)
}

// GetRefundTypeByCode mirrors RefundTypeEnumVO.getRefundTypeEnumVOByCode.
func GetRefundTypeByCode(code string) (RefundType, error) {
	switch RefundType(code) {
	case RefundTypeUnpaidUnlock, RefundTypePaidUnformed, RefundTypePaidFormed:
		return RefundType(code), nil
	}
	return "", fmt.Errorf("unknown refund type enum value: %s", code)
}
