// Package rpc contains the yukino_rpc service implementations (servers) and the
// matching client wrappers used across the gateway, activity and trade services.
package rpc

import (
	"context"
	"log"

	"github.com/hangtiancheng/be261028/internal/activity"
	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/types"
)

// ActivityServer exposes the activity bounded context over yukino_rpc. It mirrors
// the legacy AdminController + MarketIndexController orchestration plus the
// domain services, so the gateway can stay a thin HTTP layer.
type ActivityServer struct {
	manage *activity.ActivityManageService
	index  *activity.IndexGroupBuyMarketService
	tag    *activity.TagService
}

// NewActivityServer wires the server.
func NewActivityServer(manage *activity.ActivityManageService, index *activity.IndexGroupBuyMarketService, tag *activity.TagService) *ActivityServer {
	return &ActivityServer{manage: manage, index: index, tag: tag}
}

// ---- admin (mirrors AdminController: SUCCESS/ERROR envelope) ----

// CreateActivity mirrors AdminController.createActivity.
func (s *ActivityServer) CreateActivity(ctx context.Context, req *api.ActivityManageRequestDTO) (reply *api.ActivityManageReply, err error) {
	defer func() { recordRPC("ActivityService", "CreateActivity", reply.RPCResultRef(), err) }()
	entity := requestDTOToEntity(req)
	saved, err := s.manage.CreateActivity(ctx, entity)
	if err != nil {
		log.Printf("[activity] create activity failed: %v", err)
		return activityErrorReply("create activity failed", err), nil
	}
	dto := entityToResponseDTO(saved)
	dto.Success = true
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity created"},
		Data:      dto,
	}, nil
}

// UpdateActivity mirrors AdminController.updateActivity.
func (s *ActivityServer) UpdateActivity(ctx context.Context, req *api.ActivityManageRequestDTO) (reply *api.ActivityManageReply, err error) {
	defer func() { recordRPC("ActivityService", "UpdateActivity", reply.RPCResultRef(), err) }()
	entity := requestDTOToEntity(req)
	saved, err := s.manage.UpdateActivity(ctx, entity)
	if err != nil {
		log.Printf("[activity] update activity failed: %v", err)
		return activityErrorReply("update activity failed", err), nil
	}
	dto := entityToResponseDTO(saved)
	dto.Success = true
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity updated"},
		Data:      dto,
	}, nil
}

// DeleteActivity mirrors AdminController.deleteActivity.
func (s *ActivityServer) DeleteActivity(ctx context.Context, req *api.ActivityIdRequest) (reply *api.ActivityManageReply, err error) {
	defer func() { recordRPC("ActivityService", "DeleteActivity", reply.RPCResultRef(), err) }()
	result, err := s.manage.DeleteActivity(ctx, req.ActivityId)
	if err != nil {
		log.Printf("[activity] delete activity failed: activityId=%d err=%v", req.ActivityId, err)
		reply := activityErrorReply("delete activity failed", err)
		reply.Data.ActivityId = req.ActivityId
		return reply, nil
	}
	dto := &api.ActivityManageResponseDTO{ActivityId: req.ActivityId, Success: result}
	if result {
		return &api.ActivityManageReply{
			RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity deleted"},
			Data:      dto,
		}, nil
	}
	dto.ErrorMessage = "delete activity failed"
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "ERROR", Info: "delete activity failed"},
		Data:      dto,
	}, nil
}

// GetActivityDetail mirrors AdminController.getActivityDetail.
func (s *ActivityServer) GetActivityDetail(ctx context.Context, req *api.ActivityIdRequest) (reply *api.ActivityManageReply, err error) {
	defer func() { recordRPC("ActivityService", "GetActivityDetail", reply.RPCResultRef(), err) }()
	entity, err := s.manage.GetActivityDetail(ctx, req.ActivityId)
	if err != nil {
		log.Printf("[activity] get activity detail failed: activityId=%d err=%v", req.ActivityId, err)
		reply := activityErrorReply("get activity detail failed", err)
		reply.Data.ActivityId = req.ActivityId
		return reply, nil
	}
	dto := entityToResponseDTO(entity)
	dto.Success = true
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity detail retrieved"},
		Data:      dto,
	}, nil
}

// GetActivityList mirrors AdminController.getActivityList.
func (s *ActivityServer) GetActivityList(ctx context.Context, req *api.EmptyRequest) (reply *api.ActivityListReply, err error) {
	defer func() { recordRPC("ActivityService", "GetActivityList", reply.RPCResultRef(), err) }()
	entities, err := s.manage.GetActivityList(ctx)
	if err != nil {
		log.Printf("[activity] get activity list failed: %v", err)
		return &api.ActivityListReply{
			RPCResult: api.RPCResult{Code: "ERROR", Info: "get activity list failed: " + err.Error()},
		}, nil
	}
	list := make([]*api.ActivityManageResponseDTO, 0, len(entities))
	for _, e := range entities {
		dto := entityToResponseDTO(e)
		dto.Success = true
		list = append(list, dto)
	}
	return &api.ActivityListReply{
		RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity list retrieved"},
		Data:      list,
	}, nil
}

// EnableActivity mirrors AdminController.enableActivity.
func (s *ActivityServer) EnableActivity(ctx context.Context, req *api.ActivityIdRequest) (reply *api.ActivityManageReply, err error) {
	defer func() { recordRPC("ActivityService", "EnableActivity", reply.RPCResultRef(), err) }()
	entity, err := s.manage.EnableActivity(ctx, req.ActivityId)
	if err != nil {
		log.Printf("[activity] enable activity failed: activityId=%d err=%v", req.ActivityId, err)
		reply := activityErrorReply("enable activity failed", err)
		reply.Data.ActivityId = req.ActivityId
		return reply, nil
	}
	dto := entityToResponseDTO(entity)
	dto.Success = true
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity enabled"},
		Data:      dto,
	}, nil
}

// DisableActivity mirrors AdminController.disableActivity.
func (s *ActivityServer) DisableActivity(ctx context.Context, req *api.ActivityIdRequest) (reply *api.ActivityManageReply, err error) {
	defer func() { recordRPC("ActivityService", "DisableActivity", reply.RPCResultRef(), err) }()
	entity, err := s.manage.DisableActivity(ctx, req.ActivityId)
	if err != nil {
		log.Printf("[activity] disable activity failed: activityId=%d err=%v", req.ActivityId, err)
		reply := activityErrorReply("disable activity failed", err)
		reply.Data.ActivityId = req.ActivityId
		return reply, nil
	}
	dto := entityToResponseDTO(entity)
	dto.Success = true
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "SUCCESS", Info: "activity disabled"},
		Data:      dto,
	}, nil
}

// ---- market index (mirrors MarketIndexController) ----

// QueryGroupBuyMarketConfig mirrors MarketIndexController.queryGroupBuyMarketConfig:
// trial calculation, in-progress team list and team statistics assembled into one
// response.
func (s *ActivityServer) QueryGroupBuyMarketConfig(ctx context.Context, req *api.GoodsMarketRequestDTO) (reply *api.MarketConfigReply, err error) {
	defer func() { recordRPC("ActivityService", "QueryGroupBuyMarketConfig", reply.RPCResultRef(), err) }()
	log.Printf("[activity] query group-buy market config start userId:%s goodsId:%s", req.UserId, req.GoodsId)
	if types.IsBlank(req.UserId) || types.IsBlank(req.Source) || types.IsBlank(req.Channel) || types.IsBlank(req.GoodsId) {
		return &api.MarketConfigReply{RPCResult: resultFromCode(types.ResponseIllegalParam)}, nil
	}

	trial, err := s.index.IndexMarketTrial(ctx, &api.MarketProductEntity{
		UserId:  req.UserId,
		Source:  req.Source,
		Channel: req.Channel,
		GoodsId: req.GoodsId,
	})
	if err != nil {
		return &api.MarketConfigReply{RPCResult: resultFromError(err)}, nil
	}
	if trial.GroupBuyActivityDiscountVO == nil {
		return &api.MarketConfigReply{RPCResult: resultFromCode(types.ResponseE0002)}, nil
	}
	activityId := trial.GroupBuyActivityDiscountVO.ActivityId

	details, err := s.index.QueryInProgressUserGroupBuyOrderDetailList(ctx, activityId, req.UserId, 1, 2)
	if err != nil {
		return &api.MarketConfigReply{RPCResult: resultFromError(err)}, nil
	}
	statistic, err := s.index.QueryTeamStatisticByActivityId(ctx, activityId)
	if err != nil {
		return &api.MarketConfigReply{RPCResult: resultFromError(err)}, nil
	}

	goods := &api.Goods{
		GoodsId:        trial.GoodsId,
		ActivityId:     formatInt64(activityId),
		OriginalPrice:  trial.OriginalPrice,
		DeductionPrice: trial.DeductionPrice,
		PayPrice:       trial.PayPrice,
	}
	teams := make([]*api.Team, 0, len(details))
	now := types.Now()
	for _, d := range details {
		teams = append(teams, &api.Team{
			UserId:             d.UserId,
			TeamId:             d.TeamId,
			ActivityId:         d.ActivityId,
			TargetCount:        d.TargetCount,
			CompleteCount:      d.CompleteCount,
			LockCount:          d.LockCount,
			ValidStartTime:     d.ValidStartTime,
			ValidEndTime:       d.ValidEndTime,
			ValidTimeCountdown: api.DifferenceDateTime2Str(now, d.ValidEndTime.ToTime()),
			OutTradeNo:         d.OutTradeNo,
		})
	}
	resp := &api.GoodsMarketResponseDTO{
		Goods:    goods,
		TeamList: teams,
		TeamStatistic: &api.TeamStatistic{
			AllTeamCount:         statistic.AllTeamCount,
			AllTeamCompleteCount: statistic.AllTeamCompleteCount,
			AllTeamUserCount:     statistic.AllTeamUserCount,
		},
	}
	log.Printf("[activity] query group-buy market config done userId:%s goodsId:%s", req.UserId, req.GoodsId)
	return &api.MarketConfigReply{RPCResult: resultFromCode(types.ResponseSuccess), Data: resp}, nil
}

// IndexMarketTrial exposes the trial tree to the trade service (used during lock).
func (s *ActivityServer) IndexMarketTrial(ctx context.Context, req *api.MarketProductEntity) (reply *api.MarketTrialReply, err error) {
	defer func() { recordRPC("ActivityService", "IndexMarketTrial", reply.RPCResultRef(), err) }()
	trial, err := s.index.IndexMarketTrial(ctx, req)
	if err != nil {
		return &api.MarketTrialReply{RPCResult: resultFromError(err)}, nil
	}
	return &api.MarketTrialReply{RPCResult: resultFromCode(types.ResponseSuccess), Data: trial}, nil
}

// ExecTagBatchJob exposes the crowd-tag batch job.
func (s *ActivityServer) ExecTagBatchJob(ctx context.Context, req *api.TagBatchJobRequest) (reply *api.RPCResult, err error) {
	defer func() { recordRPC("ActivityService", "ExecTagBatchJob", reply, err) }()
	if jobErr := s.tag.ExecTagBatchJob(ctx, req.TagId, req.BatchId); jobErr != nil {
		res := resultFromError(jobErr)
		return &res, nil
	}
	res := resultFromCode(types.ResponseSuccess)
	return &res, nil
}

// ---- helpers ----

func activityErrorReply(prefix string, err error) *api.ActivityManageReply {
	return &api.ActivityManageReply{
		RPCResult: api.RPCResult{Code: "ERROR", Info: prefix + ": " + err.Error()},
		Data: &api.ActivityManageResponseDTO{
			Success:      false,
			ErrorMessage: err.Error(),
		},
	}
}

func resultFromCode(rc types.ResponseCode) api.RPCResult {
	return api.RPCResult{Code: rc.Code, Info: rc.Info}
}

func resultFromError(err error) api.RPCResult {
	if ae := types.AsAppError(err); ae != nil {
		return api.RPCResult{Code: ae.Code, Info: ae.Info}
	}
	return api.RPCResult{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info}
}

func formatInt64(n int64) string {
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
