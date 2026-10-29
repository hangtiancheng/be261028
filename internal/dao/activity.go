package dao

import (
	"context"

	"gorm.io/gorm"
)

// ---- group_buy_activity ----

// QueryGroupBuyActivityList mirrors queryGroupBuyActivityList (select *).
func (d *DAO) QueryGroupBuyActivityList(ctx context.Context, tx *gorm.DB) ([]*GroupBuyActivity, error) {
	var list []*GroupBuyActivity
	if err := tx.WithContext(ctx).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// QueryValidGroupBuyActivityId mirrors queryValidGroupBuyActivityId: only
// status = 1 (Effective) activities are returned.
func (d *DAO) QueryValidGroupBuyActivityId(ctx context.Context, tx *gorm.DB, activityId int64) (*GroupBuyActivity, error) {
	var a GroupBuyActivity
	err := tx.WithContext(ctx).
		Where("activity_id = ? AND status = 1", activityId).
		Take(&a).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &a, nil
}

// QueryGroupBuyActivityByActivityId mirrors queryGroupBuyActivityByActivityId.
func (d *DAO) QueryGroupBuyActivityByActivityId(ctx context.Context, tx *gorm.DB, activityId int64) (*GroupBuyActivity, error) {
	var a GroupBuyActivity
	err := tx.WithContext(ctx).
		Where("activity_id = ?", activityId).
		Take(&a).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &a, nil
}

// InsertGroupBuyActivity mirrors insertGroupBuyActivity (explicit timestamps).
func (d *DAO) InsertGroupBuyActivity(ctx context.Context, tx *gorm.DB, a *GroupBuyActivity) error {
	return checkAffected(tx.WithContext(ctx).Create(a), 1)
}

// UpdateGroupBuyActivity mirrors updateGroupBuyActivity.
func (d *DAO) UpdateGroupBuyActivity(ctx context.Context, tx *gorm.DB, a *GroupBuyActivity) error {
	res := tx.WithContext(ctx).Model(&GroupBuyActivity{}).
		Where("activity_id = ?", a.ActivityId).
		Updates(map[string]any{
			"activity_name":    a.ActivityName,
			"discount_id":      a.DiscountId,
			"group_type":       a.GroupType,
			"take_limit_count": a.TakeLimitCount,
			"target":           a.Target,
			"valid_time":       a.ValidTime,
			"status":           a.Status,
			"start_time":       a.StartTime,
			"end_time":         a.EndTime,
			"tag_id":           a.TagId,
			"tag_scope":        a.TagScope,
			"update_time":      gorm.Expr("now()"),
		})
	return checkAffected(res, 1)
}

// DeleteGroupBuyActivity mirrors deleteGroupBuyActivity.
func (d *DAO) DeleteGroupBuyActivity(ctx context.Context, tx *gorm.DB, activityId int64) (int64, error) {
	res := tx.WithContext(ctx).Where("activity_id = ?", activityId).Delete(&GroupBuyActivity{})
	return res.RowsAffected, res.Error
}

// ---- group_buy_discount ----

// QueryGroupBuyDiscountByDiscountId mirrors queryGroupBuyActivityDiscountByDiscountId.
func (d *DAO) QueryGroupBuyDiscountByDiscountId(ctx context.Context, tx *gorm.DB, discountId string) (*GroupBuyDiscount, error) {
	var g GroupBuyDiscount
	err := tx.WithContext(ctx).
		Where("discount_id = ?", discountId).
		Take(&g).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &g, nil
}

// InsertGroupBuyDiscount mirrors insertGroupBuyDiscount.
func (d *DAO) InsertGroupBuyDiscount(ctx context.Context, tx *gorm.DB, g *GroupBuyDiscount) error {
	return checkAffected(tx.WithContext(ctx).Create(g), 1)
}

// UpdateGroupBuyDiscount mirrors updateGroupBuyDiscount.
func (d *DAO) UpdateGroupBuyDiscount(ctx context.Context, tx *gorm.DB, g *GroupBuyDiscount) error {
	res := tx.WithContext(ctx).Model(&GroupBuyDiscount{}).
		Where("discount_id = ?", g.DiscountId).
		Updates(map[string]any{
			"discount_name": g.DiscountName,
			"discount_desc": g.DiscountDesc,
			"discount_type": g.DiscountType,
			"market_plan":   g.MarketPlan,
			"market_expr":   g.MarketExpr,
			"tag_id":        g.TagId,
			"update_time":   gorm.Expr("now()"),
		})
	return checkAffected(res, 1)
}

// ---- sku ----

// QuerySkuByGoodsId mirrors querySkuByGoodsId.
func (d *DAO) QuerySkuByGoodsId(ctx context.Context, tx *gorm.DB, goodsId string) (*Sku, error) {
	var s Sku
	err := tx.WithContext(ctx).
		Where("goods_id = ?", goodsId).
		Take(&s).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &s, nil
}

// InsertSku mirrors insertSku.
func (d *DAO) InsertSku(ctx context.Context, tx *gorm.DB, s *Sku) error {
	return checkAffected(tx.WithContext(ctx).Create(s), 1)
}

// UpdateSku mirrors updateSku.
func (d *DAO) UpdateSku(ctx context.Context, tx *gorm.DB, s *Sku) error {
	res := tx.WithContext(ctx).Model(&Sku{}).
		Where("goods_id = ?", s.GoodsId).
		Updates(map[string]any{
			"goods_name":     s.GoodsName,
			"original_price": s.OriginalPrice,
			"update_time":    gorm.Expr("now()"),
		})
	return checkAffected(res, 1)
}

// ---- sc_sku_activity ----

// QuerySCSkuActivityBySCGoodsId mirrors querySCSkuActivityBySCGoodsId.
//
// Defect fix: the legacy SQL filtered only by goods_id even though the method
// name (and the table's unique key source+channel+goods_id) implies all three
// columns. Goods that exist in several channels could therefore resolve to the
// wrong channel's activity; the query now matches the full unique key.
func (d *DAO) QuerySCSkuActivityBySCGoodsId(ctx context.Context, tx *gorm.DB, source, channel, goodsId string) (*SCSkuActivity, error) {
	var s SCSkuActivity
	err := tx.WithContext(ctx).
		Where("source = ? AND channel = ? AND goods_id = ?", source, channel, goodsId).
		Take(&s).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &s, nil
}

// QuerySCSkuActivityByActivityId looks up the channel/goods relation by activity.
// The legacy code reused the goods_id query with an activity id (which never
// matched); this correct query fixes that latent defect.
func (d *DAO) QuerySCSkuActivityByActivityId(ctx context.Context, tx *gorm.DB, activityId int64) (*SCSkuActivity, error) {
	var s SCSkuActivity
	err := tx.WithContext(ctx).
		Where("activity_id = ?", activityId).
		Take(&s).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &s, nil
}

// InsertSCSkuActivity mirrors insertSCSkuActivity.
func (d *DAO) InsertSCSkuActivity(ctx context.Context, tx *gorm.DB, s *SCSkuActivity) error {
	return checkAffected(tx.WithContext(ctx).Create(s), 1)
}

// UpdateSCSkuActivity mirrors updateSCSkuActivity (keyed by activity_id).
func (d *DAO) UpdateSCSkuActivity(ctx context.Context, tx *gorm.DB, s *SCSkuActivity) error {
	res := tx.WithContext(ctx).Model(&SCSkuActivity{}).
		Where("activity_id = ?", s.ActivityId).
		Updates(map[string]any{
			"source":      s.Source,
			"channel":     s.Channel,
			"goods_id":    s.GoodsId,
			"update_time": gorm.Expr("now()"),
		})
	return checkAffected(res, 1)
}

// DeleteSCSkuActivityByActivityId mirrors deleteSCSkuActivityByActivityId.
func (d *DAO) DeleteSCSkuActivityByActivityId(ctx context.Context, tx *gorm.DB, activityId int64) (int64, error) {
	res := tx.WithContext(ctx).Where("activity_id = ?", activityId).Delete(&SCSkuActivity{})
	return res.RowsAffected, res.Error
}
