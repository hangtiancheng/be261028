package api

// RPC service and method names shared by the gateway, activity and trade
// services. The activity service also acts as a provider to the trade service
// (IndexMarketTrial), so both sides import these constants.
const (
	ActivityServiceName = "ActivityService"
	TradeServiceName    = "TradeService"
)

// ActivityService method names.
const (
	MethodCreateActivity            = "CreateActivity"
	MethodUpdateActivity            = "UpdateActivity"
	MethodDeleteActivity            = "DeleteActivity"
	MethodGetActivityDetail         = "GetActivityDetail"
	MethodGetActivityList           = "GetActivityList"
	MethodEnableActivity            = "EnableActivity"
	MethodDisableActivity           = "DisableActivity"
	MethodQueryGroupBuyMarketConfig = "QueryGroupBuyMarketConfig"
	MethodIndexMarketTrial          = "IndexMarketTrial"
	MethodExecTagBatchJob           = "ExecTagBatchJob"
)

// TradeService method names.
const (
	MethodLockMarketPayOrder       = "LockMarketPayOrder"
	MethodSettlementMarketPayOrder = "SettlementMarketPayOrder"
	MethodRefundMarketPayOrder     = "RefundMarketPayOrder"
)

// ActivityIdRequest carries a single activityId for detail/delete/enable/disable.
type ActivityIdRequest struct {
	ActivityId int64 `json:"activityId"`
}

// EmptyRequest is used by parameter-less RPC methods such as GetActivityList.
type EmptyRequest struct{}

// TagBatchJobRequest triggers a crowd-tag batch job.
type TagBatchJobRequest struct {
	TagId   string `json:"tagId"`
	BatchId string `json:"batchId"`
}

// Each reply embeds RPCResult so business outcomes (including business errors
// with response codes) survive the RPC wire; Data carries the method payload.

// ActivityManageReply is the reply for single-activity admin operations.
type ActivityManageReply struct {
	RPCResult
	Data *ActivityManageResponseDTO `json:"data,omitempty"`
}

// ActivityListReply is the reply for GetActivityList.
type ActivityListReply struct {
	RPCResult
	Data []*ActivityManageResponseDTO `json:"data,omitempty"`
}

// MarketConfigReply is the reply for QueryGroupBuyMarketConfig.
type MarketConfigReply struct {
	RPCResult
	Data *GoodsMarketResponseDTO `json:"data,omitempty"`
}

// MarketTrialReply is the reply for IndexMarketTrial (trade -> activity).
type MarketTrialReply struct {
	RPCResult
	Data *TrialBalanceEntity `json:"data,omitempty"`
}

// LockOrderReply is the reply for LockMarketPayOrder.
type LockOrderReply struct {
	RPCResult
	Data *LockMarketPayOrderResponseDTO `json:"data,omitempty"`
}

// SettlementReply is the reply for SettlementMarketPayOrder.
type SettlementReply struct {
	RPCResult
	Data *SettlementMarketPayOrderResponseDTO `json:"data,omitempty"`
}

// RefundReply is the reply for RefundMarketPayOrder.
type RefundReply struct {
	RPCResult
	Data *RefundMarketPayOrderResponseDTO `json:"data,omitempty"`
}
