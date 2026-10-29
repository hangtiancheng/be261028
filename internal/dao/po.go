// Package dao holds the persistence objects (PO) mirroring the MySQL schema and
// the GORM queries that operate on them. It is shared by the activity and trade
// domains, reproducing the legacy MyBatis mapper behaviour.
package dao

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// GroupBuyActivity mirrors the group_buy_activity table.
type GroupBuyActivity struct {
	Id             int64          `gorm:"column:id;primaryKey;autoIncrement"`
	ActivityId     int64          `gorm:"column:activity_id"`
	ActivityName   string         `gorm:"column:activity_name"`
	DiscountId     string         `gorm:"column:discount_id"`
	GroupType      int            `gorm:"column:group_type"`
	TakeLimitCount int            `gorm:"column:take_limit_count"`
	Target         int            `gorm:"column:target"`
	ValidTime      int            `gorm:"column:valid_time"`
	Status         int            `gorm:"column:status"`
	StartTime      time.Time      `gorm:"column:start_time"`
	EndTime        time.Time      `gorm:"column:end_time"`
	TagId          sql.NullString `gorm:"column:tag_id"`
	TagScope       sql.NullString `gorm:"column:tag_scope"`
	CreateTime     time.Time      `gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time      `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (GroupBuyActivity) TableName() string { return "group_buy_activity" }

// CacheRedisKey returns the Redis cache key for an activity, renamed to the
// be261028 project prefix.
func (GroupBuyActivity) CacheRedisKey(activityId int64) string {
	return "be261028_GroupBuyActivity_" + itoa(activityId)
}

// GroupBuyDiscount mirrors the group_buy_discount table.
type GroupBuyDiscount struct {
	Id           int64          `gorm:"column:id;primaryKey;autoIncrement"`
	DiscountId   string         `gorm:"column:discount_id"`
	DiscountName string         `gorm:"column:discount_name"`
	DiscountDesc string         `gorm:"column:discount_desc"`
	DiscountType int            `gorm:"column:discount_type"`
	MarketPlan   string         `gorm:"column:market_plan"`
	MarketExpr   string         `gorm:"column:market_expr"`
	TagId        sql.NullString `gorm:"column:tag_id"`
	CreateTime   time.Time      `gorm:"column:create_time;autoCreateTime"`
	UpdateTime   time.Time      `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (GroupBuyDiscount) TableName() string { return "group_buy_discount" }

// CacheRedisKey returns the Redis cache key for a discount.
func (GroupBuyDiscount) CacheRedisKey(discountId string) string {
	return "be261028_GroupBuyDiscount_" + discountId
}

// Sku mirrors the sku table.
type Sku struct {
	Id            int64           `gorm:"column:id;primaryKey;autoIncrement"`
	Source        string          `gorm:"column:source"`
	Channel       string          `gorm:"column:channel"`
	GoodsId       string          `gorm:"column:goods_id"`
	GoodsName     string          `gorm:"column:goods_name"`
	OriginalPrice decimal.Decimal `gorm:"column:original_price"`
	CreateTime    time.Time       `gorm:"column:create_time;autoCreateTime"`
	UpdateTime    time.Time       `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (Sku) TableName() string { return "sku" }

// SCSkuActivity mirrors the sc_sku_activity table.
type SCSkuActivity struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Source     string    `gorm:"column:source"`
	Channel    string    `gorm:"column:channel"`
	ActivityId int64     `gorm:"column:activity_id"`
	GoodsId    string    `gorm:"column:goods_id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (SCSkuActivity) TableName() string { return "sc_sku_activity" }

// GroupBuyOrder mirrors the group_buy_order table.
type GroupBuyOrder struct {
	Id             int64           `gorm:"column:id;primaryKey;autoIncrement"`
	TeamId         string          `gorm:"column:team_id"`
	ActivityId     int64           `gorm:"column:activity_id"`
	Source         string          `gorm:"column:source"`
	Channel        string          `gorm:"column:channel"`
	OriginalPrice  decimal.Decimal `gorm:"column:original_price"`
	DeductionPrice decimal.Decimal `gorm:"column:deduction_price"`
	PayPrice       decimal.Decimal `gorm:"column:pay_price"`
	TargetCount    int             `gorm:"column:target_count"`
	CompleteCount  int             `gorm:"column:complete_count"`
	LockCount      int             `gorm:"column:lock_count"`
	Status         int             `gorm:"column:status"`
	ValidStartTime time.Time       `gorm:"column:valid_start_time"`
	ValidEndTime   time.Time       `gorm:"column:valid_end_time"`
	NotifyType     string          `gorm:"column:notify_type"`
	NotifyUrl      sql.NullString  `gorm:"column:notify_url"`
	CreateTime     time.Time       `gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time       `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (GroupBuyOrder) TableName() string { return "group_buy_order" }

// GroupBuyOrderList mirrors the group_buy_order_list table.
type GroupBuyOrderList struct {
	Id             int64           `gorm:"column:id;primaryKey;autoIncrement"`
	UserId         string          `gorm:"column:user_id"`
	TeamId         string          `gorm:"column:team_id"`
	OrderId        string          `gorm:"column:order_id"`
	ActivityId     int64           `gorm:"column:activity_id"`
	StartTime      time.Time       `gorm:"column:start_time"`
	EndTime        time.Time       `gorm:"column:end_time"`
	GoodsId        string          `gorm:"column:goods_id"`
	Source         string          `gorm:"column:source"`
	Channel        string          `gorm:"column:channel"`
	OriginalPrice  decimal.Decimal `gorm:"column:original_price"`
	DeductionPrice decimal.Decimal `gorm:"column:deduction_price"`
	PayPrice       decimal.Decimal `gorm:"column:pay_price"`
	Status         int             `gorm:"column:status"`
	OutTradeNo     string          `gorm:"column:out_trade_no"`
	OutTradeTime   sql.NullTime    `gorm:"column:out_trade_time"`
	BizId          string          `gorm:"column:biz_id"`
	CreateTime     time.Time       `gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time       `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (GroupBuyOrderList) TableName() string { return "group_buy_order_list" }

// NotifyTask mirrors the notify_task table.
type NotifyTask struct {
	Id             int64          `gorm:"column:id;primaryKey;autoIncrement"`
	ActivityId     int64          `gorm:"column:activity_id"`
	TeamId         string         `gorm:"column:team_id"`
	NotifyCategory sql.NullString `gorm:"column:notify_category"`
	NotifyType     string         `gorm:"column:notify_type"`
	NotifyMQ       sql.NullString `gorm:"column:notify_mq"`
	NotifyUrl      sql.NullString `gorm:"column:notify_url"`
	NotifyCount    int            `gorm:"column:notify_count"`
	NotifyStatus   int            `gorm:"column:notify_status"`
	ParameterJson  string         `gorm:"column:parameter_json"`
	CreateTime     time.Time      `gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time      `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (NotifyTask) TableName() string { return "notify_task" }

// CrowdTags mirrors the crowd_tags table.
type CrowdTags struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TagId      string    `gorm:"column:tag_id"`
	TagName    string    `gorm:"column:tag_name"`
	TagDesc    string    `gorm:"column:tag_desc"`
	Statistics int       `gorm:"column:statistics"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (CrowdTags) TableName() string { return "crowd_tags" }

// CrowdTagsDetail mirrors the crowd_tags_detail table.
type CrowdTagsDetail struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TagId      string    `gorm:"column:tag_id"`
	UserId     string    `gorm:"column:user_id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (CrowdTagsDetail) TableName() string { return "crowd_tags_detail" }

// CrowdTagsJob mirrors the crowd_tags_job table.
type CrowdTagsJob struct {
	Id            int64     `gorm:"column:id;primaryKey;autoIncrement"`
	TagId         string    `gorm:"column:tag_id"`
	BatchId       string    `gorm:"column:batch_id"`
	TagType       int       `gorm:"column:tag_type"`
	TagRule       string    `gorm:"column:tag_rule"`
	StatStartTime time.Time `gorm:"column:stat_start_time"`
	StatEndTime   time.Time `gorm:"column:stat_end_time"`
	Status        int       `gorm:"column:status"`
	CreateTime    time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime    time.Time `gorm:"column:update_time;autoUpdateTime"`
}

// TableName pins the GORM table for the PO.
func (CrowdTagsJob) TableName() string { return "crowd_tags_job" }

// itoa avoids importing strconv in every cache-key helper.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
