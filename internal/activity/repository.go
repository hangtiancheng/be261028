package activity

import (
	"context"
	"errors"
	"log"
	"math/rand"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/dao"
	"github.com/hangtiancheng/be261028/internal/infra/dcc"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/types"
	"gorm.io/gorm"
)

// ActivityRepository mirrors the legacy ActivityRepository (read side) and
// TagRepository, backed by MySQL with optional Redis caching gated by the DCC
// cache switch.
type ActivityRepository struct {
	db    *gorm.DB
	dao   *dao.DAO
	redis *redisx.RedisService
	dcc   *dcc.DCC
}

// NewActivityRepository wires an ActivityRepository.
func NewActivityRepository(db *gorm.DB, d *dao.DAO, redis *redisx.RedisService, dc *dcc.DCC) *ActivityRepository {
	return &ActivityRepository{db: db, dao: d, redis: redis, dcc: dc}
}

// getFromCacheOrDb mirrors AbstractRepository.getFromCacheOrDb: when the DCC cache
// switch is open it reads through Redis and populates it on a miss; otherwise it
// queries the database directly. A nil result means "not found".
func getFromCacheOrDb[T any](ctx context.Context, r *ActivityRepository, cacheKey string, dbFallback func() (*T, error)) (*T, error) {
	cacheOpen := r.dcc.IsCacheOpenSwitch()
	if cacheOpen {
		var cached T
		found, err := r.redis.GetValue(ctx, cacheKey, &cached)
		if err != nil {
			log.Printf("[activity] 缓存读取失败，回源数据库 key:%s err:%v", cacheKey, err)
		} else if found {
			return &cached, nil
		}
	} else {
		log.Printf("[activity] 缓存降级 %s", cacheKey)
	}

	dbResult, err := dbFallback()
	if err != nil {
		return nil, err
	}
	if dbResult == nil {
		return nil, nil
	}
	if cacheOpen {
		if err := r.redis.SetValue(ctx, cacheKey, dbResult); err != nil {
			log.Printf("[activity] 缓存写入失败 key:%s err:%v", cacheKey, err)
		}
	}
	return dbResult, nil
}

// QueryGroupBuyActivityDiscountVO mirrors ActivityRepository.queryGroupBuyActivityDiscountVO.
func (r *ActivityRepository) QueryGroupBuyActivityDiscountVO(ctx context.Context, activityId int64) (*api.GroupBuyActivityDiscountVO, error) {
	activityPO, err := getFromCacheOrDb(ctx, r, (&dao.GroupBuyActivity{}).CacheRedisKey(activityId), func() (*dao.GroupBuyActivity, error) {
		po, err := r.dao.QueryValidGroupBuyActivityId(ctx, r.db, activityId)
		if errors.Is(err, dao.ErrNotFound) {
			return nil, nil
		}
		return po, err
	})
	if err != nil {
		return nil, err
	}
	if activityPO == nil {
		return nil, nil
	}

	discountId := activityPO.DiscountId
	discountPO, err := getFromCacheOrDb(ctx, r, (&dao.GroupBuyDiscount{}).CacheRedisKey(discountId), func() (*dao.GroupBuyDiscount, error) {
		po, err := r.dao.QueryGroupBuyDiscountByDiscountId(ctx, r.db, discountId)
		if errors.Is(err, dao.ErrNotFound) {
			return nil, nil
		}
		return po, err
	})
	if err != nil {
		return nil, err
	}
	if discountPO == nil {
		return nil, nil
	}

	return &api.GroupBuyActivityDiscountVO{
		ActivityId:   activityPO.ActivityId,
		ActivityName: activityPO.ActivityName,
		GroupBuyDiscount: &api.GroupBuyDiscount{
			DiscountName: discountPO.DiscountName,
			DiscountDesc: discountPO.DiscountDesc,
			DiscountType: types.DiscountTypeOf(discountPO.DiscountType),
			MarketPlan:   discountPO.MarketPlan,
			MarketExpr:   discountPO.MarketExpr,
			TagId:        discountPO.TagId.String,
		},
		GroupType:      activityPO.GroupType,
		TakeLimitCount: activityPO.TakeLimitCount,
		Target:         activityPO.Target,
		ValidTime:      activityPO.ValidTime,
		Status:         activityPO.Status,
		StartTime:      types.NewJSONTime(activityPO.StartTime),
		EndTime:        types.NewJSONTime(activityPO.EndTime),
		TagId:          activityPO.TagId.String,
		TagScope:       activityPO.TagScope.String,
	}, nil
}

// QuerySkuByGoodsId mirrors ActivityRepository.querySkuByGoodsId.
func (r *ActivityRepository) QuerySkuByGoodsId(ctx context.Context, goodsId string) (*api.SkuVO, error) {
	sku, err := r.dao.QuerySkuByGoodsId(ctx, r.db, goodsId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &api.SkuVO{
		GoodsId:       sku.GoodsId,
		GoodsName:     sku.GoodsName,
		OriginalPrice: types.NewMoney(sku.OriginalPrice),
	}, nil
}

// QuerySCSkuActivityBySCGoodsId mirrors ActivityRepository.querySCSkuActivityBySCGoodsId.
// Defect fix: source/channel are now part of the lookup (see the dao comment);
// the legacy query matched goods_id alone and could resolve a foreign channel.
func (r *ActivityRepository) QuerySCSkuActivityBySCGoodsId(ctx context.Context, source, channel, goodsId string) (*SCSkuActivityVO, error) {
	sc, err := r.dao.QuerySCSkuActivityBySCGoodsId(ctx, r.db, source, channel, goodsId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &SCSkuActivityVO{
		Source:     sc.Source,
		Channel:    sc.Channel,
		ActivityId: sc.ActivityId,
		GoodsId:    sc.GoodsId,
	}, nil
}

// IsTagCrowdRange mirrors ActivityRepository.isTagCrowdRange: a missing bitmap
// means no restriction (true); otherwise membership is the user's bit.
//
// 设计说明（保留旧版 fail-open 语义）：位图缺失通常表示人群任务尚未跑批。若改为
// fail-closed，种子活动 100123（tag_id='1' 无对应位图）将对所有用户不可见，直接
// 破坏默认演示链路，属于能力回归；因此这里与旧版一致按"无限制"处理。
func (r *ActivityRepository) IsTagCrowdRange(ctx context.Context, tagId, userId string) (bool, error) {
	exists, err := r.redis.BitSetExists(ctx, tagId)
	if err != nil {
		return false, err
	}
	if !exists {
		return true, nil
	}
	return r.redis.BitSetGet(ctx, tagId, types.IndexFromUserId(userId))
}

// DowngradeSwitch mirrors ActivityRepository.downgradeSwitch.
func (r *ActivityRepository) DowngradeSwitch() bool { return r.dcc.IsDowngradeSwitch() }

// CutRange mirrors ActivityRepository.cutRange.
func (r *ActivityRepository) CutRange(userId string) bool { return r.dcc.IsCutRange(userId) }

// IsSCBlackIntercept mirrors the DCC-backed blacklist check used by trade.
func (r *ActivityRepository) IsSCBlackIntercept(source, channel string) bool {
	return r.dcc.IsSCBlackIntercept(source, channel)
}

// QueryInProgressUserGroupBuyOrderDetailListByOwner mirrors the owner-scoped query.
func (r *ActivityRepository) QueryInProgressUserGroupBuyOrderDetailListByOwner(ctx context.Context, activityId int64, userId string, ownerCount int) ([]*api.UserGroupBuyOrderDetailEntity, error) {
	briefs, err := r.dao.QueryInProgressOrderDetailListByUserId(ctx, r.db, activityId, userId, ownerCount)
	if err != nil {
		return nil, err
	}
	if len(briefs) == 0 {
		return nil, nil
	}
	teamIds := uniqueTeamIds(briefs)
	orders, err := r.dao.QueryGroupBuyProgressByTeamIds(ctx, r.db, teamIds)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, nil
	}
	orderMap := indexOrdersByTeamId(orders)

	var result []*api.UserGroupBuyOrderDetailEntity
	for _, brief := range briefs {
		order, ok := orderMap[brief.TeamId]
		if !ok {
			continue
		}
		result = append(result, &api.UserGroupBuyOrderDetailEntity{
			UserId:         brief.UserId,
			TeamId:         order.TeamId,
			ActivityId:     order.ActivityId,
			TargetCount:    order.TargetCount,
			CompleteCount:  order.CompleteCount,
			LockCount:      order.LockCount,
			ValidStartTime: types.NewJSONTime(order.ValidStartTime),
			ValidEndTime:   types.NewJSONTime(order.ValidEndTime),
			OutTradeNo:     brief.OutTradeNo, // owner query carries outTradeNo
		})
	}
	return result, nil
}

// QueryInProgressUserGroupBuyOrderDetailListByRandom mirrors the random query:
// it fetches 2x the requested count, shuffles, then keeps randomCount. The legacy
// random projection omits outTradeNo, which is reproduced here.
//
// Defect fix: the legacy sample was truncated before teams were de-duplicated, so
// two orders of the same team could consume both slots and the filtered result came
// back empty even though joinable teams existed. Briefs are now collapsed to one
// row per team before sampling, so randomCount really means "randomCount teams".
func (r *ActivityRepository) QueryInProgressUserGroupBuyOrderDetailListByRandom(ctx context.Context, activityId int64, userId string, randomCount int) ([]*api.UserGroupBuyOrderDetailEntity, error) {
	briefs, err := r.dao.QueryInProgressOrderDetailListByRandom(ctx, r.db, activityId, userId, randomCount*2)
	if err != nil {
		return nil, err
	}
	briefs = dedupeBriefsByTeam(briefs)
	if len(briefs) == 0 {
		return nil, nil
	}
	if len(briefs) > randomCount {
		rand.Shuffle(len(briefs), func(i, j int) { briefs[i], briefs[j] = briefs[j], briefs[i] })
		briefs = briefs[:randomCount]
	}
	teamIds := uniqueTeamIds(briefs)
	orders, err := r.dao.QueryGroupBuyProgressByTeamIds(ctx, r.db, teamIds)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, nil
	}
	orderMap := indexOrdersByTeamId(orders)

	var result []*api.UserGroupBuyOrderDetailEntity
	for _, brief := range briefs {
		order, ok := orderMap[brief.TeamId]
		if !ok {
			continue
		}
		result = append(result, &api.UserGroupBuyOrderDetailEntity{
			UserId:         brief.UserId,
			TeamId:         order.TeamId,
			ActivityId:     order.ActivityId,
			TargetCount:    order.TargetCount,
			CompleteCount:  order.CompleteCount,
			LockCount:      order.LockCount,
			ValidStartTime: types.NewJSONTime(order.ValidStartTime),
			ValidEndTime:   types.NewJSONTime(order.ValidEndTime),
			// outTradeNo intentionally omitted to match the legacy random query
		})
	}
	return result, nil
}

// QueryTeamStatisticByActivityId mirrors ActivityRepository.queryTeamStatisticByActivityId.
func (r *ActivityRepository) QueryTeamStatisticByActivityId(ctx context.Context, activityId int64) (*api.TeamStatisticVO, error) {
	orders, err := r.dao.QueryGroupBuyOrdersByActivityId(ctx, r.db, activityId)
	if err != nil {
		return nil, err
	}
	allTeamCount := len(orders)
	allTeamCompleteCount := 0
	allTeamUserCount := 0
	for _, o := range orders {
		if o.Status == int(types.GroupBuyOrderComplete) {
			allTeamCompleteCount++
		}
		allTeamUserCount += o.CompleteCount
	}
	return &api.TeamStatisticVO{
		AllTeamCount:         allTeamCount,
		AllTeamCompleteCount: allTeamCompleteCount,
		AllTeamUserCount:     allTeamUserCount,
	}, nil
}

// ---- TagRepository ----

// TagRepository mirrors the legacy TagRepository.
type TagRepository struct {
	db    *gorm.DB
	dao   *dao.DAO
	redis *redisx.RedisService
}

// NewTagRepository wires a TagRepository.
func NewTagRepository(db *gorm.DB, d *dao.DAO, redis *redisx.RedisService) *TagRepository {
	return &TagRepository{db: db, dao: d, redis: redis}
}

// QueryCrowdTagsJobEntity mirrors TagRepository.queryCrowdTagsJobEntity.
func (r *TagRepository) QueryCrowdTagsJobEntity(ctx context.Context, tagId, batchId string) (*CrowdTagsJobEntity, error) {
	job, err := r.dao.QueryCrowdTagsJob(ctx, r.db, tagId, batchId)
	if errors.Is(err, dao.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &CrowdTagsJobEntity{
		TagType:       job.TagType,
		TagRule:       job.TagRule,
		StatStartTime: job.StatStartTime,
		StatEndTime:   job.StatEndTime,
	}, nil
}

// AddCrowdTagsUserId mirrors TagRepository.addCrowdTagsUserId: insert the detail
// row (ignoring duplicate-key) and set the user's bit in the tag bitmap.
func (r *TagRepository) AddCrowdTagsUserId(ctx context.Context, tagId, userId string) error {
	err := r.dao.AddCrowdTagsUserId(ctx, r.db, tagId, userId)
	if err != nil && !dao.IsDuplicateKey(err) {
		return err
	}
	// Duplicate key is ignored (matching the legacy DuplicateKeyException catch);
	// the bitmap is still updated so membership is idempotent.
	return r.redis.BitSetSet(ctx, tagId, types.IndexFromUserId(userId), true)
}

// UpdateCrowdTagsStatistics mirrors TagRepository.updateCrowdTagsStatistics.
func (r *TagRepository) UpdateCrowdTagsStatistics(ctx context.Context, tagId string, count int) error {
	return r.dao.UpdateCrowdTagsStatistics(ctx, r.db, tagId, count)
}

// ---- helpers ----

func uniqueTeamIds(briefs []*dao.OrderListBrief) []string {
	seen := make(map[string]struct{}, len(briefs))
	var ids []string
	for _, b := range briefs {
		if b.TeamId == "" {
			continue
		}
		if _, ok := seen[b.TeamId]; ok {
			continue
		}
		seen[b.TeamId] = struct{}{}
		ids = append(ids, b.TeamId)
	}
	return ids
}

func indexOrdersByTeamId(orders []*dao.GroupBuyOrder) map[string]*dao.GroupBuyOrder {
	m := make(map[string]*dao.GroupBuyOrder, len(orders))
	for _, o := range orders {
		m[o.TeamId] = o
	}
	return m
}

// dedupeBriefsByTeam keeps the first brief of each team, preserving input order so
// the subsequent shuffle still provides the random ordering.
func dedupeBriefsByTeam(briefs []*dao.OrderListBrief) []*dao.OrderListBrief {
	if len(briefs) < 2 {
		return briefs
	}
	seen := make(map[string]struct{}, len(briefs))
	out := briefs[:0]
	for _, b := range briefs {
		if _, ok := seen[b.TeamId]; ok {
			continue
		}
		seen[b.TeamId] = struct{}{}
		out = append(out, b)
	}
	return out
}
