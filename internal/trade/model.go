// Package trade implements the trade bounded context: order locking, settlement,
// refunds, the notify-task callback machinery and the Redis-backed team stock.
// It mirrors the legacy easy-group-buy-domain trade package.
package trade

import (
	"time"

	"github.com/hangtiancheng/be261028/internal/types"
	"github.com/shopspring/decimal"
)

// NotifyConfigVO mirrors com.buy.group.easy.domain.trade.model.valobj.NotifyConfigVO.
type NotifyConfigVO struct {
	NotifyType types.NotifyType
	NotifyMQ   string
	NotifyUrl  string
}

// UserEntity mirrors com.buy.group.easy.domain.trade.model.entity.UserEntity.
type UserEntity struct {
	UserId string
}

// MarketPayOrderEntity mirrors com.buy.group.easy.domain.trade.model.entity.MarketPayOrderEntity.
// ActivityId is carried alongside so idempotent settlement replies can be built
// without an extra team lookup.
type MarketPayOrderEntity struct {
	TeamId           string
	OrderId          string
	ActivityId       int64
	OriginalPrice    decimal.Decimal
	DeductionPrice   decimal.Decimal
	PayPrice         decimal.Decimal
	TradeOrderStatus types.TradeOrderStatus
}

// GroupBuyActivityEntity mirrors com.buy.group.easy.domain.trade.model.entity.GroupBuyActivityEntity.
type GroupBuyActivityEntity struct {
	ActivityId     int64
	ActivityName   string
	DiscountId     string
	GroupType      int
	TakeLimitCount int
	Target         int
	ValidTime      int
	Status         types.ActivityStatus
	StartTime      time.Time
	EndTime        time.Time
	TagId          string
	TagScope       string
}

// GroupBuyTeamEntity mirrors com.buy.group.easy.domain.trade.model.entity.GroupBuyTeamEntity.
type GroupBuyTeamEntity struct {
	TeamId         string
	ActivityId     int64
	TargetCount    int
	CompleteCount  int
	LockCount      int
	Status         types.GroupBuyOrderStatus
	ValidStartTime time.Time
	ValidEndTime   time.Time
	NotifyUrl      string
	NotifyConfigVO *NotifyConfigVO
}

// NotifyTaskEntity mirrors com.buy.group.easy.domain.trade.model.entity.NotifyTaskEntity.
// Id is added so status updates target a single task (see the notify_task defect
// fix in the dao package).
type NotifyTaskEntity struct {
	Id            int64
	TeamId        string
	NotifyType    types.NotifyType
	NotifyMQ      string
	NotifyUrl     string
	NotifyCount   int
	ParameterJson string
}

// LockKey mirrors NotifyTaskEntity.lockKey.
func (n *NotifyTaskEntity) LockKey() string {
	return "be261028_notify_job_lock_key_" + n.TeamId
}

// PayActivityEntity mirrors com.buy.group.easy.domain.trade.model.entity.PayActivityEntity.
type PayActivityEntity struct {
	TeamId       string
	ActivityId   int64
	ActivityName string
	StartTime    time.Time
	EndTime      time.Time
	ValidTime    int
	TargetCount  int
}

// PayDiscountEntity mirrors com.buy.group.easy.domain.trade.model.entity.PayDiscountEntity.
type PayDiscountEntity struct {
	Source         string
	Channel        string
	GoodsId        string
	GoodsName      string
	OriginalPrice  decimal.Decimal
	DeductionPrice decimal.Decimal
	PayPrice       decimal.Decimal
	OutTradeNo     string
	NotifyConfigVO *NotifyConfigVO
}

// GroupBuyOrderAggregate mirrors com.buy.group.easy.domain.trade.model.aggregate.GroupBuyOrderAggregate.
type GroupBuyOrderAggregate struct {
	UserEntity         *UserEntity
	PayActivityEntity  *PayActivityEntity
	PayDiscountEntity  *PayDiscountEntity
	UserTakeOrderCount int
}

// GroupBuyProgressVO mirrors com.buy.group.easy.domain.trade.model.valobj.GroupBuyProgressVO.
type GroupBuyProgressVO struct {
	TargetCount   int
	CompleteCount int
	LockCount     int
}

// TradePaySuccessEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradePaySuccessEntity.
type TradePaySuccessEntity struct {
	Source       string
	Channel      string
	UserId       string
	OutTradeNo   string
	OutTradeTime time.Time
}

// TradePaySettlementEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradePaySettlementEntity.
type TradePaySettlementEntity struct {
	Source     string
	Channel    string
	UserId     string
	TeamId     string
	ActivityId int64
	OutTradeNo string
}

// GroupBuyTeamSettlementAggregate mirrors com.buy.group.easy.domain.trade.model.aggregate.GroupBuyTeamSettlementAggregate.
type GroupBuyTeamSettlementAggregate struct {
	UserEntity            *UserEntity
	GroupBuyTeamEntity    *GroupBuyTeamEntity
	TradePaySuccessEntity *TradePaySuccessEntity
}

// TradeLockRuleCommandEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeLockRuleCommandEntity.
type TradeLockRuleCommandEntity struct {
	UserId     string
	ActivityId int64
	TeamId     string
}

// TradeLockRuleFilterBackEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeLockRuleFilterBackEntity.
type TradeLockRuleFilterBackEntity struct {
	UserTakeOrderCount   int
	RecoveryTeamStockKey string
}

// TradeSettlementRuleCommandEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeSettlementRuleCommandEntity.
type TradeSettlementRuleCommandEntity struct {
	Source       string
	Channel      string
	UserId       string
	OutTradeNo   string
	OutTradeTime time.Time
}

// TradeSettlementRuleFilterBackEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeSettlementRuleFilterBackEntity.
// AlreadySettled marks a repeat settlement of an already-completed order; the
// service answers it as an idempotent success instead of re-running the
// transaction (which the legacy code did, double-counting the team).
type TradeSettlementRuleFilterBackEntity struct {
	TeamId         string
	ActivityId     int64
	TargetCount    int
	CompleteCount  int
	LockCount      int
	Status         types.GroupBuyOrderStatus
	ValidStartTime time.Time
	ValidEndTime   time.Time
	NotifyConfigVO *NotifyConfigVO
	AlreadySettled bool
}

// TradeRefundCommandEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeRefundCommandEntity.
type TradeRefundCommandEntity struct {
	UserId     string
	OutTradeNo string
	Source     string
	Channel    string
}

// TradeRefundOrderEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeRefundOrderEntity.
type TradeRefundOrderEntity struct {
	UserId     string
	TeamId     string
	ActivityId int64
	OrderId    string
}

// TradeRefundBehaviorEntity mirrors com.buy.group.easy.domain.trade.model.entity.TradeRefundBehaviorEntity.
type TradeRefundBehaviorEntity struct {
	UserId   string
	OrderId  string
	TeamId   string
	Behavior types.RefundBehavior
}

// GroupBuyRefundAggregate mirrors com.buy.group.easy.domain.trade.model.aggregate.GroupBuyRefundAggregate.
type GroupBuyRefundAggregate struct {
	TradeRefundOrderEntity *TradeRefundOrderEntity
	GroupBuyProgress       *GroupBuyProgressVO
	GroupBuyOrderStatus    types.GroupBuyOrderStatus
}

// BuildUnpaid2RefundAggregate mirrors GroupBuyRefundAggregate.buildUnpaid2RefundAggregate.
func BuildUnpaid2RefundAggregate(order *TradeRefundOrderEntity, lockCount int) *GroupBuyRefundAggregate {
	return &GroupBuyRefundAggregate{
		TradeRefundOrderEntity: order,
		GroupBuyProgress:       &GroupBuyProgressVO{LockCount: lockCount},
	}
}

// BuildPaid2RefundAggregate mirrors GroupBuyRefundAggregate.buildPaid2RefundAggregate.
func BuildPaid2RefundAggregate(order *TradeRefundOrderEntity, lockCount, completeCount int) *GroupBuyRefundAggregate {
	return &GroupBuyRefundAggregate{
		TradeRefundOrderEntity: order,
		GroupBuyProgress:       &GroupBuyProgressVO{LockCount: lockCount, CompleteCount: completeCount},
	}
}

// BuildPaidTeam2RefundAggregate mirrors GroupBuyRefundAggregate.buildPaidTeam2RefundAggregate.
func BuildPaidTeam2RefundAggregate(order *TradeRefundOrderEntity, lockCount, completeCount int, status types.GroupBuyOrderStatus) *GroupBuyRefundAggregate {
	return &GroupBuyRefundAggregate{
		TradeRefundOrderEntity: order,
		GroupBuyProgress:       &GroupBuyProgressVO{LockCount: lockCount, CompleteCount: completeCount},
		GroupBuyOrderStatus:    status,
	}
}

// TeamRefundSuccess mirrors com.buy.group.easy.domain.trade.model.valobj.TeamRefundSuccess,
// the payload published on the team-refund topic and consumed to restore stock.
type TeamRefundSuccess struct {
	Type       string `json:"type"`
	UserId     string `json:"userId"`
	TeamId     string `json:"teamId"`
	ActivityId int64  `json:"activityId"`
	OrderId    string `json:"orderId"`
}

// TeamSuccessMessage mirrors the settlement notify payload published on the
// team-success topic.
type TeamSuccessMessage struct {
	Result         string   `json:"result"`
	TeamId         string   `json:"teamId"`
	OutTradeNoList []string `json:"outTradeNoList"`
}
