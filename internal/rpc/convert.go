package rpc

import (
	"github.com/hangtiancheng/be261028/internal/activity"
	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/hangtiancheng/be261028/internal/types"
)

// recordRPC counts one server-side RPC invocation on the shared Prometheus
// counter. Admin replies use the "SUCCESS"/"ERROR" envelope while market replies
// carry numeric business codes, so both success markers are recognized.
func recordRPC(service, method string, result *api.RPCResult, err error) {
	outcome := "error"
	if err == nil && result != nil &&
		(result.Code == types.ResponseSuccess.Code || result.Code == "SUCCESS") {
		outcome = "success"
	}
	metrics.RPCCallsTotal.WithLabelValues(service, method, outcome).Inc()
}

// replyCode extracts the business code from a reply result (empty when nil).
func replyCode(result *api.RPCResult) string {
	if result == nil {
		return ""
	}
	return result.Code
}

// requestDTOToEntity converts the admin request DTO into the activity domain entity.
func requestDTOToEntity(req *api.ActivityManageRequestDTO) *activity.ActivityManageEntity {
	entity := &activity.ActivityManageEntity{
		ActivityId:     req.ActivityId,
		ActivityName:   req.ActivityName,
		DiscountId:     req.DiscountId,
		GroupType:      req.GroupType,
		TakeLimitCount: req.TakeLimitCount,
		Target:         req.Target,
		ValidTime:      req.ValidTime,
		Status:         req.Status,
		StartTime:      req.StartTime.ToTime(),
		EndTime:        req.EndTime.ToTime(),
	}
	if req.DiscountInfo != nil {
		entity.DiscountInfo = &activity.DiscountInfo{
			DiscountId:   req.DiscountInfo.DiscountId,
			DiscountName: req.DiscountInfo.DiscountName,
			DiscountDesc: req.DiscountInfo.DiscountDesc,
			DiscountType: req.DiscountInfo.DiscountType,
			MarketPlan:   req.DiscountInfo.MarketPlan,
			MarketExpr:   req.DiscountInfo.MarketExpr,
			TagId:        req.DiscountInfo.TagId,
		}
	}
	if req.SkuInfo != nil {
		entity.SkuInfo = &activity.SkuInfo{
			GoodsId:       req.SkuInfo.GoodsId,
			GoodsName:     req.SkuInfo.GoodsName,
			Source:        req.SkuInfo.Source,
			Channel:       req.SkuInfo.Channel,
			OriginalPrice: req.SkuInfo.OriginalPrice,
		}
	}
	return entity
}

// entityToResponseDTO converts the activity domain entity into the admin response
// DTO. The legacy DTO exposes only scalar fields (plus discountId), so nested
// discount/sku objects are intentionally not carried here.
func entityToResponseDTO(e *activity.ActivityManageEntity) *api.ActivityManageResponseDTO {
	return &api.ActivityManageResponseDTO{
		ActivityId:     e.ActivityId,
		ActivityName:   e.ActivityName,
		DiscountId:     e.DiscountId,
		GroupType:      e.GroupType,
		TakeLimitCount: e.TakeLimitCount,
		Target:         e.Target,
		ValidTime:      e.ValidTime,
		Status:         e.Status,
		StartTime:      types.NewJSONTime(e.StartTime),
		EndTime:        types.NewJSONTime(e.EndTime),
		CreateTime:     types.NewJSONTime(e.CreateTime),
		UpdateTime:     types.NewJSONTime(e.UpdateTime),
	}
}
