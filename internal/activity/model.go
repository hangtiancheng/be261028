// Package activity implements the marketing/activity bounded context: activity
// administration, the trial-calculation strategy tree, discount plans, crowd tags
// and the read-side queries that back the market index. It mirrors the legacy
// easy-group-buy-domain activity package.
package activity

import (
	"time"

	"github.com/shopspring/decimal"
)

// SCSkuActivityVO mirrors com.buy.group.easy.domain.activity.model.valobj.SCSkuActivityVO.
type SCSkuActivityVO struct {
	Source     string
	Channel    string
	ActivityId int64
	GoodsId    string
}

// CrowdTagsJobEntity mirrors com.buy.group.easy.domain.tag.model.entity.CrowdTagsJobEntity.
type CrowdTagsJobEntity struct {
	TagType       int
	TagRule       string
	StatStartTime time.Time
	StatEndTime   time.Time
}

// ActivityManageEntity mirrors com.buy.group.easy.domain.activity.model.entity.ActivityManageEntity.
// DiscountInfo/SkuInfo reuse the api shapes since they are pure data holders.
type ActivityManageEntity struct {
	ActivityId     int64
	ActivityName   string
	DiscountId     string
	GroupType      int
	TakeLimitCount int
	Target         int
	ValidTime      int
	Status         int
	StartTime      time.Time
	EndTime        time.Time
	DiscountInfo   *DiscountInfo
	SkuInfo        *SkuInfo
	CreateTime     time.Time
	UpdateTime     time.Time
}

// DiscountInfo mirrors ActivityManageEntity.DiscountInfo.
type DiscountInfo struct {
	DiscountId   string
	DiscountName string
	DiscountDesc string
	DiscountType int
	MarketPlan   string
	MarketExpr   string
	TagId        string
}

// SkuInfo mirrors ActivityManageEntity.SkuInfo.
type SkuInfo struct {
	GoodsId       string
	GoodsName     string
	Source        string
	Channel       string
	OriginalPrice decimal.Decimal
}
