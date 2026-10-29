package activity

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/types"
)

// marketFanoutTimeout bounds the concurrent activity/SKU load in the Market node.
//
// Defect fix (legacy defect 9): the Java version awaited the CompletableFuture
// fan-out with join(), which had no overall deadline, so a hung downstream could
// pin the request thread indefinitely. Both branches honour ctx, so capping the
// context here guarantees wg.Wait() returns within this bound.
const marketFanoutTimeout = 3 * time.Second

// TrialService implements the legacy trial strategy tree
// (Root -> Switch -> Market -> Tag -> End, with Error as the fallback branch).
// The Java version routes through AbstractMultiThreadStrategyRouter nodes; here the
// same decisions are expressed as sequential steps, with the Market node loading
// the activity config and SKU concurrently (mirroring the CompletableFuture fan-out).
type TrialService struct {
	repo      *ActivityRepository
	discounts *DiscountRegistry
}

// NewTrialService wires a TrialService.
func NewTrialService(repo *ActivityRepository, discounts *DiscountRegistry) *TrialService {
	return &TrialService{repo: repo, discounts: discounts}
}

// IndexMarketTrial mirrors IIndexGroupBuyMarketService.indexMarketTrial.
func (s *TrialService) IndexMarketTrial(ctx context.Context, req *api.MarketProductEntity) (*api.TrialBalanceEntity, error) {
	// RootNode: parameter validation.
	if types.IsBlank(req.UserId) || types.IsBlank(req.GoodsId) ||
		types.IsBlank(req.Source) || types.IsBlank(req.Channel) {
		return nil, types.NewAppError(types.ResponseIllegalParam)
	}

	// SwitchNode: downgrade + traffic cut.
	if s.repo.DowngradeSwitch() {
		log.Printf("[activity] group-buy downgraded, intercepting request userId:%s", req.UserId)
		return nil, types.NewAppError(types.ResponseE0003)
	}
	if !s.repo.CutRange(req.UserId) {
		log.Printf("[activity] group-buy traffic cut, intercepting request userId:%s", req.UserId)
		return nil, types.NewAppError(types.ResponseE0004)
	}

	// MarketNode.multiThread: load activity config and SKU concurrently, bounded by
	// marketFanoutTimeout so a hung downstream cannot pin the request (legacy defect 9).
	ctx, cancel := context.WithTimeout(ctx, marketFanoutTimeout)
	defer cancel()

	var (
		wg         sync.WaitGroup
		activityVO *api.GroupBuyActivityDiscountVO
		skuVO      *api.SkuVO
		skuErr     error
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		vo, err := s.loadActivityConfig(ctx, req)
		if err != nil {
			// The legacy activity branch swallowed errors into null (-> ErrorNode E0002).
			log.Printf("[activity] async activity config query failed: %v", err)
			vo = nil
		}
		activityVO = vo
	}()
	go func() {
		defer wg.Done()
		skuVO, skuErr = s.repo.QuerySkuByGoodsId(ctx, req.GoodsId)
	}()
	wg.Wait()
	if skuErr != nil {
		return nil, skuErr
	}

	// MarketNode.doApply + routing: missing config/SKU routes to ErrorNode.
	if activityVO == nil || activityVO.GroupBuyDiscount == nil || skuVO == nil {
		return s.errorNode(req, activityVO, skuVO)
	}

	discount := activityVO.GroupBuyDiscount
	calculator, ok := s.discounts.Get(discount.MarketPlan)
	if !ok {
		log.Printf("[activity] no discount calculator for market plan %s; supported plans: %v", discount.MarketPlan, s.discounts.Plans())
		return nil, types.NewAppError(types.ResponseE0001)
	}

	payPrice, err := calculator.Calculate(ctx, req.UserId, skuVO.OriginalPrice.Decimal, discount)
	if err != nil {
		return nil, err
	}
	deductionPrice := skuVO.OriginalPrice.Sub(payPrice)

	// TagNode: visibility / participation.
	visible, enable := s.tagNode(ctx, req, activityVO)

	// EndNode: assemble the trial result.
	return &api.TrialBalanceEntity{
		GoodsId:                    skuVO.GoodsId,
		GoodsName:                  skuVO.GoodsName,
		OriginalPrice:              skuVO.OriginalPrice,
		DeductionPrice:             types.NewMoney(deductionPrice),
		PayPrice:                   types.NewMoney(payPrice),
		TargetCount:                activityVO.Target,
		StartTime:                  activityVO.StartTime,
		EndTime:                    activityVO.EndTime,
		IsVisible:                  visible,
		IsEnable:                   enable,
		GroupBuyActivityDiscountVO: activityVO,
	}, nil
}

// loadActivityConfig resolves the activity (via SC mapping when no activityId is
// pinned) and loads its discount configuration.
func (s *TrialService) loadActivityConfig(ctx context.Context, req *api.MarketProductEntity) (*api.GroupBuyActivityDiscountVO, error) {
	activityId := req.ActivityId
	if activityId == nil {
		sc, err := s.repo.QuerySCSkuActivityBySCGoodsId(ctx, req.Source, req.Channel, req.GoodsId)
		if err != nil {
			return nil, err
		}
		if sc == nil {
			return nil, nil
		}
		activityId = &sc.ActivityId
	}
	return s.repo.QueryGroupBuyActivityDiscountVO(ctx, *activityId)
}

// errorNode mirrors ErrorNode: without config or SKU the goods have no group-buy
// marketing (E0002).
func (s *TrialService) errorNode(req *api.MarketProductEntity, activityVO *api.GroupBuyActivityDiscountVO, skuVO *api.SkuVO) (*api.TrialBalanceEntity, error) {
	if activityVO == nil || skuVO == nil {
		log.Printf("[activity] goods have no group-buy marketing config goodsId:%s", req.GoodsId)
		return nil, types.NewAppError(types.ResponseE0002)
	}
	// Unreachable in practice (a non-nil activityVO always carries a discount), but
	// preserved for parity with the legacy ErrorNode's empty-result branch.
	return &api.TrialBalanceEntity{}, nil
}

// tagNode mirrors TagNode: derives visible/enable from the tag scope and crowd
// membership.
func (s *TrialService) tagNode(ctx context.Context, req *api.MarketProductEntity, activityVO *api.GroupBuyActivityDiscountVO) (bool, bool) {
	tagId := activityVO.TagId
	visible := activityVO.IsVisible()
	enable := activityVO.IsEnable()

	if types.IsBlank(tagId) {
		return true, true
	}
	isWithin, err := s.repo.IsTagCrowdRange(ctx, tagId, req.UserId)
	if err != nil {
		log.Printf("[activity] crowd tag lookup failed tagId:%s userId:%s err:%v", tagId, req.UserId, err)
		isWithin = false
	}
	return visible || isWithin, enable || isWithin
}
