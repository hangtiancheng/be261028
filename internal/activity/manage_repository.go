package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/hangtiancheng/be261028/internal/dao"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/types"
	"gorm.io/gorm"
)

// ActivityManageRepository mirrors the legacy ActivityManageRepository (write side
// of activity administration). Multi-table writes are wrapped in a transaction so
// a partial failure cannot leave orphaned rows (the legacy version was
// non-transactional, a latent defect).
type ActivityManageRepository struct {
	db    *gorm.DB
	dao   *dao.DAO
	redis *redisx.RedisService
}

// NewActivityManageRepository wires the repository.
func NewActivityManageRepository(db *gorm.DB, d *dao.DAO, redis *redisx.RedisService) *ActivityManageRepository {
	return &ActivityManageRepository{db: db, dao: d, redis: redis}
}

// invalidateCache drops the cached activity/discount configs so an admin change
// takes effect immediately. Defect fix: the legacy system cached these forever and
// never invalidated on update/enable/disable/delete, so e.g. disabling an activity
// had no effect on the trial path until the process restarted.
func (r *ActivityManageRepository) invalidateCache(ctx context.Context, activityId int64, discountId string) {
	if r.redis == nil {
		return
	}
	if err := r.redis.Remove(ctx, (&dao.GroupBuyActivity{}).CacheRedisKey(activityId)); err != nil {
		log.Printf("[activity] failed to invalidate activity cache activityId:%d err:%v", activityId, err)
	}
	if discountId != "" {
		if err := r.redis.Remove(ctx, (&dao.GroupBuyDiscount{}).CacheRedisKey(discountId)); err != nil {
			log.Printf("[activity] failed to invalidate discount cache discountId:%s err:%v", discountId, err)
		}
	}
}

// SaveActivity persists a new activity plus its discount, sku and channel mapping.
func (r *ActivityManageRepository) SaveActivity(ctx context.Context, entity *ActivityManageEntity) (*ActivityManageEntity, error) {
	now := types.Now()
	entity.CreateTime = now
	entity.UpdateTime = now

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	activityPO := entityToActivityPO(entity)
	if err := r.dao.InsertGroupBuyActivity(ctx, tx, activityPO); err != nil {
		return nil, fmt.Errorf("failed to save activity: %w", err)
	}
	if entity.DiscountInfo != nil {
		discountPO := discountInfoToPO(entity.DiscountInfo)
		discountPO.CreateTime = now
		discountPO.UpdateTime = now
		if err := r.dao.InsertGroupBuyDiscount(ctx, tx, discountPO); err != nil {
			return nil, fmt.Errorf("failed to save discount: %w", err)
		}
	}
	if entity.SkuInfo != nil {
		skuPO := skuInfoToPO(entity.SkuInfo)
		skuPO.CreateTime = now
		skuPO.UpdateTime = now
		if err := r.dao.InsertSku(ctx, tx, skuPO); err != nil {
			return nil, fmt.Errorf("failed to save sku: %w", err)
		}
		scPO := &dao.SCSkuActivity{
			ActivityId: entity.ActivityId,
			GoodsId:    entity.SkuInfo.GoodsId,
			Source:     entity.SkuInfo.Source,
			Channel:    entity.SkuInfo.Channel,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := r.dao.InsertSCSkuActivity(ctx, tx, scPO); err != nil {
			return nil, fmt.Errorf("failed to save sku-activity mapping: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return entity, nil
}

// UpdateActivity updates an existing activity and its related rows. The existing
// tag_id/tag_scope are preserved because the admin contract does not carry them
// (the legacy update blanked them out, a defect).
func (r *ActivityManageRepository) UpdateActivity(ctx context.Context, entity *ActivityManageEntity) (*ActivityManageEntity, error) {
	existing, err := r.dao.QueryGroupBuyActivityByActivityId(ctx, r.db, entity.ActivityId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, fmt.Errorf("activity not found: %d", entity.ActivityId)
	}
	if err != nil {
		return nil, err
	}
	entity.UpdateTime = types.Now()

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback() //nolint:errcheck

	activityPO := entityToActivityPO(entity)
	// Preserve crowd-tag columns not exposed by the admin API.
	activityPO.TagId = existing.TagId
	activityPO.TagScope = existing.TagScope
	if err := r.dao.UpdateGroupBuyActivity(ctx, tx, activityPO); err != nil {
		return nil, fmt.Errorf("failed to update activity: %w", err)
	}
	if entity.DiscountInfo != nil {
		discountPO := discountInfoToPO(entity.DiscountInfo)
		discountPO.UpdateTime = entity.UpdateTime
		if err := r.dao.UpdateGroupBuyDiscount(ctx, tx, discountPO); err != nil {
			return nil, fmt.Errorf("failed to update discount: %w", err)
		}
	}
	if entity.SkuInfo != nil {
		skuPO := skuInfoToPO(entity.SkuInfo)
		skuPO.UpdateTime = entity.UpdateTime
		if err := r.dao.UpdateSku(ctx, tx, skuPO); err != nil {
			return nil, fmt.Errorf("failed to update sku: %w", err)
		}
		scPO := &dao.SCSkuActivity{
			ActivityId: entity.ActivityId,
			GoodsId:    entity.SkuInfo.GoodsId,
			Source:     entity.SkuInfo.Source,
			Channel:    entity.SkuInfo.Channel,
			UpdateTime: entity.UpdateTime,
		}
		if err := r.dao.UpdateSCSkuActivity(ctx, tx, scPO); err != nil {
			return nil, fmt.Errorf("failed to update sku-activity mapping: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	r.invalidateCache(ctx, entity.ActivityId, entity.DiscountId)
	return entity, nil
}

// DeleteActivity removes the channel mapping and the activity. Discount and sku
// rows are intentionally kept (they may be shared), matching the legacy behaviour.
func (r *ActivityManageRepository) DeleteActivity(ctx context.Context, activityId int64) (bool, error) {
	if _, err := r.dao.QueryGroupBuyActivityByActivityId(ctx, r.db, activityId); errors.Is(err, dao.ErrNotFound) {
		return false, fmt.Errorf("activity not found: %d", activityId)
	} else if err != nil {
		return false, err
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return false, tx.Error
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := r.dao.DeleteSCSkuActivityByActivityId(ctx, tx, activityId); err != nil {
		return false, fmt.Errorf("failed to delete sku-activity mapping: %w", err)
	}
	deleted, err := r.dao.DeleteGroupBuyActivity(ctx, tx, activityId)
	if err != nil {
		return false, fmt.Errorf("failed to delete activity: %w", err)
	}
	if deleted <= 0 {
		return false, errors.New("failed to delete activity: no rows affected")
	}
	if err := tx.Commit().Error; err != nil {
		return false, err
	}
	r.invalidateCache(ctx, activityId, "")
	return true, nil
}

// GetActivityById loads an activity with its discount and sku (the sku relation is
// resolved by activity id, fixing the legacy query that filtered by a null goods id).
func (r *ActivityManageRepository) GetActivityById(ctx context.Context, activityId int64) (*ActivityManageEntity, error) {
	activityPO, err := r.dao.QueryGroupBuyActivityByActivityId(ctx, r.db, activityId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entity := activityPOToEntity(activityPO)
	if err := r.attachDiscountAndSku(ctx, r.db, entity, activityPO.DiscountId, activityId); err != nil {
		return nil, err
	}
	return entity, nil
}

// GetActivityList loads every activity with its discount and sku.
func (r *ActivityManageRepository) GetActivityList(ctx context.Context) ([]*ActivityManageEntity, error) {
	activityPOs, err := r.dao.QueryGroupBuyActivityList(ctx, r.db)
	if err != nil {
		return nil, err
	}
	list := make([]*ActivityManageEntity, 0, len(activityPOs))
	for _, po := range activityPOs {
		entity := activityPOToEntity(po)
		if err := r.attachDiscountAndSku(ctx, r.db, entity, po.DiscountId, po.ActivityId); err != nil {
			return nil, err
		}
		list = append(list, entity)
	}
	return list, nil
}

func (r *ActivityManageRepository) attachDiscountAndSku(ctx context.Context, e *gorm.DB, entity *ActivityManageEntity, discountId string, activityId int64) error {
	if discountId != "" {
		discountPO, err := r.dao.QueryGroupBuyDiscountByDiscountId(ctx, e, discountId)
		if err != nil && !errors.Is(err, dao.ErrNotFound) {
			return err
		}
		if discountPO != nil {
			entity.DiscountInfo = discountPOToInfo(discountPO)
		}
	}
	sc, err := r.dao.QuerySCSkuActivityByActivityId(ctx, e, activityId)
	if err != nil && !errors.Is(err, dao.ErrNotFound) {
		return err
	}
	if sc != nil && sc.GoodsId != "" {
		skuPO, err := r.dao.QuerySkuByGoodsId(ctx, e, sc.GoodsId)
		if err != nil && !errors.Is(err, dao.ErrNotFound) {
			return err
		}
		if skuPO != nil {
			entity.SkuInfo = skuPOToInfo(skuPO)
		}
	}
	return nil
}

// ---- conversions ----

func entityToActivityPO(e *ActivityManageEntity) *dao.GroupBuyActivity {
	return &dao.GroupBuyActivity{
		ActivityId:     e.ActivityId,
		ActivityName:   e.ActivityName,
		DiscountId:     e.DiscountId,
		GroupType:      e.GroupType,
		TakeLimitCount: e.TakeLimitCount,
		Target:         e.Target,
		ValidTime:      e.ValidTime,
		Status:         e.Status,
		StartTime:      e.StartTime,
		EndTime:        e.EndTime,
		// TagId/TagScope are not carried by the admin entity; callers preserve them.
		CreateTime: e.CreateTime,
		UpdateTime: e.UpdateTime,
	}
}

func activityPOToEntity(po *dao.GroupBuyActivity) *ActivityManageEntity {
	return &ActivityManageEntity{
		ActivityId:     po.ActivityId,
		ActivityName:   po.ActivityName,
		DiscountId:     po.DiscountId,
		GroupType:      po.GroupType,
		TakeLimitCount: po.TakeLimitCount,
		Target:         po.Target,
		ValidTime:      po.ValidTime,
		Status:         po.Status,
		StartTime:      po.StartTime,
		EndTime:        po.EndTime,
		CreateTime:     po.CreateTime,
		UpdateTime:     po.UpdateTime,
	}
}

func discountInfoToPO(info *DiscountInfo) *dao.GroupBuyDiscount {
	return &dao.GroupBuyDiscount{
		DiscountId:   info.DiscountId,
		DiscountName: info.DiscountName,
		DiscountDesc: info.DiscountDesc,
		DiscountType: info.DiscountType,
		MarketPlan:   info.MarketPlan,
		MarketExpr:   info.MarketExpr,
		TagId:        nullString(info.TagId),
	}
}

func discountPOToInfo(po *dao.GroupBuyDiscount) *DiscountInfo {
	return &DiscountInfo{
		DiscountId:   po.DiscountId,
		DiscountName: po.DiscountName,
		DiscountDesc: po.DiscountDesc,
		DiscountType: po.DiscountType,
		MarketPlan:   po.MarketPlan,
		MarketExpr:   po.MarketExpr,
		TagId:        po.TagId.String,
	}
}

func skuInfoToPO(info *SkuInfo) *dao.Sku {
	return &dao.Sku{
		Source:        info.Source,
		Channel:       info.Channel,
		GoodsId:       info.GoodsId,
		GoodsName:     info.GoodsName,
		OriginalPrice: info.OriginalPrice,
	}
}

func skuPOToInfo(po *dao.Sku) *SkuInfo {
	return &SkuInfo{
		GoodsId:       po.GoodsId,
		GoodsName:     po.GoodsName,
		Source:        po.Source,
		Channel:       po.Channel,
		OriginalPrice: po.OriginalPrice,
	}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
