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

// RPCResultRef returns a pointer to the embedded RPCResult, or nil when the
// receiver itself is nil. Server handlers call it from a deferred metrics hook so
// that a handler panicking before it assigns the named reply (leaving it nil) does
// not trigger a secondary nil dereference of &reply.RPCResult. Pointer-receiver
// methods are safe to invoke on a nil value, which is what makes this guard work.
func (r *ActivityManageReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}

// RPCResultRef implements the nil-safe embedded-result accessor for ActivityListReply.
func (r *ActivityListReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}

// RPCResultRef implements the nil-safe embedded-result accessor for MarketConfigReply.
func (r *MarketConfigReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}

// RPCResultRef implements the nil-safe embedded-result accessor for MarketTrialReply.
func (r *MarketTrialReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}

// RPCResultRef implements the nil-safe embedded-result accessor for LockOrderReply.
func (r *LockOrderReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}

// RPCResultRef implements the nil-safe embedded-result accessor for SettlementReply.
func (r *SettlementReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}

// RPCResultRef implements the nil-safe embedded-result accessor for RefundReply.
func (r *RefundReply) RPCResultRef() *RPCResult {
	if r == nil {
		return nil
	}
	return &r.RPCResult
}
