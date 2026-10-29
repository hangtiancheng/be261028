package api

import (
	"github.com/hangtiancheng/be261028/internal/types"
)

// The types below are the shared value objects that cross service boundaries
// (gateway -> activity, trade -> activity). They mirror the legacy domain VOs so
// the RPC payload is a faithful representation of the in-process objects.

// MarketProductEntity mirrors com.buy.group.easy.domain.activity.model.entity.MarketProductEntity.
// ActivityId is nil when the caller does not pin a specific activity (the index
// flow resolves it from source/channel/goodsId).
type MarketProductEntity struct {
	ActivityId *int64 `json:"activityId,omitempty"`
	UserId     string `json:"userId"`
	GoodsId    string `json:"goodsId"`
	Source     string `json:"source"`
	Channel    string `json:"channel"`
}

// SkuVO mirrors com.buy.group.easy.domain.activity.model.valobj.SkuVO.
type SkuVO struct {
	GoodsId       string      `json:"goodsId"`
	GoodsName     string      `json:"goodsName"`
	OriginalPrice types.Money `json:"originalPrice"`
}

// GroupBuyDiscount mirrors GroupBuyActivityDiscountVO.GroupBuyDiscount.
type GroupBuyDiscount struct {
	DiscountName string             `json:"discountName"`
	DiscountDesc string             `json:"discountDesc"`
	DiscountType types.DiscountType `json:"discountType"`
	MarketPlan   string             `json:"marketPlan"`
	MarketExpr   string             `json:"marketExpr"`
	TagId        string             `json:"tagId"`
}

// GroupBuyActivityDiscountVO mirrors com.buy.group.easy.domain.activity.model.valobj.GroupBuyActivityDiscountVO.
type GroupBuyActivityDiscountVO struct {
	ActivityId       int64             `json:"activityId"`
	ActivityName     string            `json:"activityName"`
	Source           string            `json:"source"`
	Channel          string            `json:"channel"`
	GoodsId          string            `json:"goodsId"`
	GroupBuyDiscount *GroupBuyDiscount `json:"groupBuyDiscount"`
	GroupType        int               `json:"groupType"`
	TakeLimitCount   int               `json:"takeLimitCount"`
	Target           int               `json:"target"`
	ValidTime        int               `json:"validTime"`
	Status           int               `json:"status"`
	StartTime        types.JSONTime    `json:"startTime"`
	EndTime          types.JSONTime    `json:"endTime"`
	TagId            string            `json:"tagId"`
	TagScope         string            `json:"tagScope"`
}

// IsVisible mirrors GroupBuyActivityDiscountVO.isVisible: a blank tagScope allows
// visibility; otherwise scope "1" (visibility restriction) refuses it.
//
// Defect fix: the legacy parser only inspected split[0] for "1", so a scope of
// "2,1" was treated as visible. Membership is now checked across all segments.
func (v *GroupBuyActivityDiscountVO) IsVisible() bool {
	if types.IsBlank(v.TagScope) {
		return true
	}
	return !hasTagScopeSegment(v.TagScope, "1")
}

// IsEnable mirrors GroupBuyActivityDiscountVO.isEnable: a blank tagScope allows
// participation; otherwise scope "2" (participation restriction) refuses it.
//
// Defect fix: the legacy parser required exactly two segments with "2" in the
// second position, so "2" alone or "2,1" silently allowed participation.
// Membership is now checked across all segments.
func (v *GroupBuyActivityDiscountVO) IsEnable() bool {
	if types.IsBlank(v.TagScope) {
		return true
	}
	return !hasTagScopeSegment(v.TagScope, "2")
}

// hasTagScopeSegment reports whether the comma-separated tagScope contains the
// given segment.
func hasTagScopeSegment(tagScope, segment string) bool {
	for _, s := range types.JavaSplit(tagScope, types.Split) {
		if s == segment {
			return true
		}
	}
	return false
}

// TrialBalanceEntity mirrors com.buy.group.easy.domain.activity.model.entity.TrialBalanceEntity.
type TrialBalanceEntity struct {
	GoodsId                    string                      `json:"goodsId"`
	GoodsName                  string                      `json:"goodsName"`
	OriginalPrice              types.Money                 `json:"originalPrice"`
	DeductionPrice             types.Money                 `json:"deductionPrice"`
	PayPrice                   types.Money                 `json:"payPrice"`
	TargetCount                int                         `json:"targetCount"`
	StartTime                  types.JSONTime              `json:"startTime"`
	EndTime                    types.JSONTime              `json:"endTime"`
	IsVisible                  bool                        `json:"isVisible"`
	IsEnable                   bool                        `json:"isEnable"`
	GroupBuyActivityDiscountVO *GroupBuyActivityDiscountVO `json:"groupBuyActivityDiscountVO"`
}

// TeamStatisticVO mirrors com.buy.group.easy.domain.activity.model.valobj.TeamStatisticVO.
type TeamStatisticVO struct {
	AllTeamCount         int `json:"allTeamCount"`
	AllTeamCompleteCount int `json:"allTeamCompleteCount"`
	AllTeamUserCount     int `json:"allTeamUserCount"`
}

// UserGroupBuyOrderDetailEntity mirrors com.buy.group.easy.domain.activity.model.entity.UserGroupBuyOrderDetailEntity.
type UserGroupBuyOrderDetailEntity struct {
	UserId         string         `json:"userId"`
	TeamId         string         `json:"teamId"`
	ActivityId     int64          `json:"activityId"`
	TargetCount    int            `json:"targetCount"`
	CompleteCount  int            `json:"completeCount"`
	LockCount      int            `json:"lockCount"`
	ValidStartTime types.JSONTime `json:"validStartTime"`
	ValidEndTime   types.JSONTime `json:"validEndTime"`
	OutTradeNo     string         `json:"outTradeNo"`
}
