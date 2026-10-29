package api

import (
	"github.com/hangtiancheng/be261028/internal/types"
	"github.com/shopspring/decimal"
)

// ActivityManageRequestDTO mirrors com.buy.group.easy.api.dto.ActivityManageRequestDTO.
type ActivityManageRequestDTO struct {
	ActivityId     int64          `json:"activityId"`
	ActivityName   string         `json:"activityName"`
	DiscountId     string         `json:"discountId"`
	GroupType      int            `json:"groupType"`
	TakeLimitCount int            `json:"takeLimitCount"`
	Target         int            `json:"target"`
	ValidTime      int            `json:"validTime"`
	Status         int            `json:"status"`
	StartTime      types.JSONTime `json:"startTime"`
	EndTime        types.JSONTime `json:"endTime"`
	DiscountInfo   *DiscountInfo  `json:"discountInfo,omitempty"`
	SkuInfo        *SkuInfo       `json:"skuInfo,omitempty"`
}

// DiscountInfo mirrors ActivityManageRequestDTO.DiscountInfo.
type DiscountInfo struct {
	DiscountId   string `json:"discountId"`
	DiscountName string `json:"discountName"`
	DiscountDesc string `json:"discountDesc"`
	DiscountType int    `json:"discountType"`
	MarketPlan   string `json:"marketPlan"`
	MarketExpr   string `json:"marketExpr"`
	TagId        string `json:"tagId"`
}

// SkuInfo mirrors ActivityManageRequestDTO.SkuInfo.
type SkuInfo struct {
	GoodsId       string          `json:"goodsId"`
	GoodsName     string          `json:"goodsName"`
	Source        string          `json:"source"`
	Channel       string          `json:"channel"`
	OriginalPrice decimal.Decimal `json:"originalPrice"`
}

// ActivityManageResponseDTO mirrors com.buy.group.easy.api.dto.ActivityManageResponseDTO.
// The legacy DTO intentionally exposes only scalar activity fields (plus discountId);
// the nested discount/sku objects queried by the repository are not part of the
// REST contract, so they are omitted here for parity.
type ActivityManageResponseDTO struct {
	ActivityId     int64          `json:"activityId"`
	ActivityName   string         `json:"activityName"`
	DiscountId     string         `json:"discountId"`
	GroupType      int            `json:"groupType"`
	TakeLimitCount int            `json:"takeLimitCount"`
	Target         int            `json:"target"`
	ValidTime      int            `json:"validTime"`
	Status         int            `json:"status"`
	StartTime      types.JSONTime `json:"startTime"`
	EndTime        types.JSONTime `json:"endTime"`
	CreateTime     types.JSONTime `json:"createTime"`
	UpdateTime     types.JSONTime `json:"updateTime"`
	Success        bool           `json:"success"`
	ErrorMessage   string         `json:"errorMessage,omitempty"`
}
