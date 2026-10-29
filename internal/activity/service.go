package activity

import (
	"context"
	"errors"
	"fmt"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/types"
)

// ActivityManageService mirrors the legacy ActivityManageServiceImpl: validation
// plus delegation to the manage repository.
type ActivityManageService struct {
	repo *ActivityManageRepository
}

// NewActivityManageService wires the service.
func NewActivityManageService(repo *ActivityManageRepository) *ActivityManageService {
	return &ActivityManageService{repo: repo}
}

// CreateActivity validates and persists a new activity.
func (s *ActivityManageService) CreateActivity(ctx context.Context, entity *ActivityManageEntity) (*ActivityManageEntity, error) {
	if err := validateActivity(entity); err != nil {
		return nil, err
	}
	now := types.Now()
	entity.CreateTime = now
	entity.UpdateTime = now
	return s.repo.SaveActivity(ctx, entity)
}

// UpdateActivity validates and updates an existing activity.
func (s *ActivityManageService) UpdateActivity(ctx context.Context, entity *ActivityManageEntity) (*ActivityManageEntity, error) {
	existing, err := s.repo.GetActivityById(ctx, entity.ActivityId)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("activity not found: %d", entity.ActivityId)
	}
	if err := validateActivity(entity); err != nil {
		return nil, err
	}
	entity.UpdateTime = types.Now()
	return s.repo.UpdateActivity(ctx, entity)
}

// DeleteActivity removes an activity after confirming it exists.
func (s *ActivityManageService) DeleteActivity(ctx context.Context, activityId int64) (bool, error) {
	existing, err := s.repo.GetActivityById(ctx, activityId)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, fmt.Errorf("activity not found: %d", activityId)
	}
	return s.repo.DeleteActivity(ctx, activityId)
}

// GetActivityDetail returns one activity or an error when absent.
func (s *ActivityManageService) GetActivityDetail(ctx context.Context, activityId int64) (*ActivityManageEntity, error) {
	entity, err := s.repo.GetActivityById(ctx, activityId)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("activity not found: %d", activityId)
	}
	return entity, nil
}

// GetActivityList returns every activity.
func (s *ActivityManageService) GetActivityList(ctx context.Context) ([]*ActivityManageEntity, error) {
	return s.repo.GetActivityList(ctx)
}

// EnableActivity sets the activity status to Effective (1).
func (s *ActivityManageService) EnableActivity(ctx context.Context, activityId int64) (*ActivityManageEntity, error) {
	entity, err := s.repo.GetActivityById(ctx, activityId)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("activity not found: %d", activityId)
	}
	entity.Status = int(types.ActivityStatusEffective)
	entity.UpdateTime = types.Now()
	return s.repo.UpdateActivity(ctx, entity)
}

// DisableActivity sets the activity status to Abandoned (3).
func (s *ActivityManageService) DisableActivity(ctx context.Context, activityId int64) (*ActivityManageEntity, error) {
	entity, err := s.repo.GetActivityById(ctx, activityId)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, fmt.Errorf("activity not found: %d", activityId)
	}
	entity.Status = int(types.ActivityStatusAbandoned)
	entity.UpdateTime = types.Now()
	return s.repo.UpdateActivity(ctx, entity)
}

// validateActivity mirrors ActivityManageServiceImpl.validateActivity.
func validateActivity(e *ActivityManageEntity) error {
	if e.ActivityId == 0 {
		return errors.New("activity id must not be empty")
	}
	if types.IsBlank(e.ActivityName) {
		return errors.New("activity name must not be empty")
	}
	if types.IsBlank(e.DiscountId) {
		return errors.New("discount id must not be empty")
	}
	if e.StartTime.IsZero() {
		return errors.New("activity start time must not be empty")
	}
	if e.EndTime.IsZero() {
		return errors.New("activity end time must not be empty")
	}
	if e.StartTime.After(e.EndTime) {
		return errors.New("activity start time must not be later than end time")
	}
	if e.DiscountInfo != nil {
		if err := validateDiscountInfo(e.DiscountInfo); err != nil {
			return err
		}
	}
	if e.SkuInfo != nil {
		if err := validateSkuInfo(e.SkuInfo); err != nil {
			return err
		}
	}
	return nil
}

func validateDiscountInfo(d *DiscountInfo) error {
	if types.IsBlank(d.DiscountName) {
		return errors.New("discount name must not be empty")
	}
	if types.IsBlank(d.MarketPlan) {
		return errors.New("market plan must not be empty")
	}
	if types.IsBlank(d.MarketExpr) {
		return errors.New("market expression must not be empty")
	}
	return nil
}

func validateSkuInfo(k *SkuInfo) error {
	if types.IsBlank(k.GoodsId) {
		return errors.New("goods id must not be empty")
	}
	if types.IsBlank(k.GoodsName) {
		return errors.New("goods name must not be empty")
	}
	// Defect fix: the legacy validation paired each field with the other field's
	// label (a blank source reported "channel must not be empty" and a blank
	// channel reported "source must not be empty"), a copy-paste confusion carried
	// over from the legacy DTO comments, which labelled the source field "channel"
	// and the channel field "source". Messages now match the field actually being
	// validated; the rejection behavior is unchanged.
	if types.IsBlank(k.Source) {
		return errors.New("source must not be empty")
	}
	if types.IsBlank(k.Channel) {
		return errors.New("channel must not be empty")
	}
	if k.OriginalPrice.IsZero() || k.OriginalPrice.IsNegative() {
		return errors.New("goods price must be greater than zero")
	}
	return nil
}

// IndexGroupBuyMarketService mirrors the legacy IndexGroupBuyMarketServiceImpl.
type IndexGroupBuyMarketService struct {
	repo  *ActivityRepository
	trial *TrialService
}

// NewIndexGroupBuyMarketService wires the service.
func NewIndexGroupBuyMarketService(repo *ActivityRepository, trial *TrialService) *IndexGroupBuyMarketService {
	return &IndexGroupBuyMarketService{repo: repo, trial: trial}
}

// IndexMarketTrial delegates to the trial strategy tree.
func (s *IndexGroupBuyMarketService) IndexMarketTrial(ctx context.Context, req *api.MarketProductEntity) (*api.TrialBalanceEntity, error) {
	return s.trial.IndexMarketTrial(ctx, req)
}

// QueryInProgressUserGroupBuyOrderDetailList unions the owner and random
// projections, mirroring the legacy method.
func (s *IndexGroupBuyMarketService) QueryInProgressUserGroupBuyOrderDetailList(ctx context.Context, activityId int64, userId string, ownerCount, randomCount int) ([]*api.UserGroupBuyOrderDetailEntity, error) {
	var union []*api.UserGroupBuyOrderDetailEntity
	if ownerCount != 0 {
		ownerList, err := s.repo.QueryInProgressUserGroupBuyOrderDetailListByOwner(ctx, activityId, userId, ownerCount)
		if err != nil {
			return nil, err
		}
		union = append(union, ownerList...)
	}
	if randomCount != 0 {
		randomList, err := s.repo.QueryInProgressUserGroupBuyOrderDetailListByRandom(ctx, activityId, userId, randomCount)
		if err != nil {
			return nil, err
		}
		union = append(union, randomList...)
	}
	return union, nil
}

// QueryTeamStatisticByActivityId delegates to the repository.
func (s *IndexGroupBuyMarketService) QueryTeamStatisticByActivityId(ctx context.Context, activityId int64) (*api.TeamStatisticVO, error) {
	return s.repo.QueryTeamStatisticByActivityId(ctx, activityId)
}

// TagService mirrors the legacy TagService.
type TagService struct {
	repo *TagRepository
}

// NewTagService wires the service.
func NewTagService(repo *TagRepository) *TagService {
	return &TagService{repo: repo}
}

// ExecTagBatchJob mirrors execTagBatchJob. The legacy implementation queries the
// batch job, then writes a fixed demo user list into the tag bitmap and updates
// the statistics counter; real user collection is a documented TODO upstream.
func (s *TagService) ExecTagBatchJob(ctx context.Context, tagId, batchId string) error {
	if _, err := s.repo.QueryCrowdTagsJobEntity(ctx, tagId, batchId); err != nil {
		return err
	}
	userIds := []string{"xifeiya", "gaofei", "xifeiya01", "xifeiya02", "xifeiya03"}
	for _, uid := range userIds {
		if err := s.repo.AddCrowdTagsUserId(ctx, tagId, uid); err != nil {
			return err
		}
	}
	return s.repo.UpdateCrowdTagsStatistics(ctx, tagId, len(userIds))
}
