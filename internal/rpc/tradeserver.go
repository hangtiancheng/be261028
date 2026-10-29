package rpc

import (
	"context"
	"log"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/hangtiancheng/be261028/internal/trade"
	"github.com/hangtiancheng/be261028/internal/types"
)

// TradeServer exposes the trade bounded context over yukino_rpc. LockMarketPayOrder
// orchestrates the full legacy controller flow, calling back into the activity
// service for the trial calculation.
type TradeServer struct {
	lock        *trade.TradeLockOrderService
	settlement  *trade.TradeSettlementOrderService
	refund      *trade.TradeRefundOrderService
	activityCli *ActivityClient
}

// NewTradeServer wires the server.
func NewTradeServer(lock *trade.TradeLockOrderService, settlement *trade.TradeSettlementOrderService, refund *trade.TradeRefundOrderService, activityCli *ActivityClient) *TradeServer {
	return &TradeServer{lock: lock, settlement: settlement, refund: refund, activityCli: activityCli}
}

// LockMarketPayOrder mirrors MarketTradeController.lockMarketPayOrder.
func (s *TradeServer) LockMarketPayOrder(ctx context.Context, req *api.LockMarketPayOrderRequestDTO) (reply *api.LockOrderReply, err error) {
	defer func() {
		recordRPC("TradeService", "LockMarketPayOrder", &reply.RPCResult, err)
		metrics.TradeOpsTotal.WithLabelValues("lock", replyCode(&reply.RPCResult)).Inc()
	}()
	userId := req.UserId
	source := req.Source
	channel := req.Channel
	goodsId := req.GoodsId
	activityId := req.ActivityId
	outTradeNo := req.OutTradeNo
	teamId := req.TeamId
	notifyConfig := req.NotifyConfigVO

	log.Printf("营销交易锁单:%s LockMarketPayOrderRequestDTO:%+v", userId, req)

	if types.IsBlank(userId) || types.IsBlank(source) || types.IsBlank(channel) ||
		types.IsBlank(goodsId) || activityId == 0 || notifyConfig == nil ||
		(notifyConfig.NotifyType == string(types.NotifyTypeHTTP) && types.IsBlank(notifyConfig.NotifyUrl)) {
		return &api.LockOrderReply{RPCResult: resultFromCode(types.ResponseIllegalParam)}, nil
	}

	// 查询 outTradeNo 是否已经存在交易记录（幂等）
	existing, err := s.lock.QueryNoPayMarketPayOrderByOutTradeNo(ctx, userId, outTradeNo)
	if err != nil {
		return &api.LockOrderReply{RPCResult: resultFromError(err)}, nil
	}
	if existing != nil {
		log.Printf("交易锁单记录(存在):%s marketPayOrderEntity:%+v", userId, existing)
		return &api.LockOrderReply{
			RPCResult: resultFromCode(types.ResponseSuccess),
			Data: &api.LockMarketPayOrderResponseDTO{
				OrderId:          existing.OrderId,
				OriginalPrice:    types.NewMoney(existing.OriginalPrice),
				DeductionPrice:   types.NewMoney(existing.DeductionPrice),
				PayPrice:         types.NewMoney(existing.PayPrice),
				TradeOrderStatus: int(existing.TradeOrderStatus),
			},
		}, nil
	}

	// 判断拼团锁单是否完成了目标
	if !types.IsBlank(teamId) {
		progress, err := s.lock.QueryGroupBuyProgress(ctx, teamId)
		if err != nil {
			return &api.LockOrderReply{RPCResult: resultFromError(err)}, nil
		}
		if progress != nil && progress.TargetCount == progress.LockCount {
			log.Printf("交易锁单拦截-拼单目标已达成:%s %s", userId, teamId)
			return &api.LockOrderReply{RPCResult: resultFromCode(types.ResponseE0006)}, nil
		}
	}

	// 营销优惠试算（回调 activity 服务）
	trialReply, err := s.activityCli.IndexMarketTrial(ctx, &api.MarketProductEntity{
		ActivityId: &activityId,
		UserId:     userId,
		Source:     source,
		Channel:    channel,
		GoodsId:    goodsId,
	})
	if err != nil {
		return &api.LockOrderReply{RPCResult: resultFromError(err)}, nil
	}
	if !trialReply.IsSuccess() {
		return &api.LockOrderReply{RPCResult: trialReply.RPCResult}, nil
	}
	trial := trialReply.Data

	// 人群限定
	if !trial.IsVisible || !trial.IsEnable {
		return &api.LockOrderReply{RPCResult: resultFromCode(types.ResponseE0007)}, nil
	}

	discountVO := trial.GroupBuyActivityDiscountVO
	// 锁单
	order, err := s.lock.LockMarketPayOrder(ctx,
		&trade.UserEntity{UserId: userId},
		&trade.PayActivityEntity{
			TeamId:       teamId,
			ActivityId:   activityId,
			ActivityName: discountVO.ActivityName,
			StartTime:    discountVO.StartTime.ToTime(),
			EndTime:      discountVO.EndTime.ToTime(),
			ValidTime:    discountVO.ValidTime,
			TargetCount:  discountVO.Target,
		},
		&trade.PayDiscountEntity{
			Source:         source,
			Channel:        channel,
			GoodsId:        goodsId,
			GoodsName:      trial.GoodsName,
			OriginalPrice:  trial.OriginalPrice.Decimal,
			DeductionPrice: trial.DeductionPrice.Decimal,
			PayPrice:       trial.PayPrice.Decimal,
			OutTradeNo:     outTradeNo,
			NotifyConfigVO: &trade.NotifyConfigVO{
				NotifyType: types.NotifyTypeOf(notifyConfig.NotifyType),
				NotifyMQ:   notifyConfig.NotifyMQ,
				NotifyUrl:  notifyConfig.NotifyUrl,
			},
		})
	if err != nil {
		return &api.LockOrderReply{RPCResult: resultFromError(err)}, nil
	}

	log.Printf("交易锁单记录(新):%s marketPayOrderEntity:%+v", userId, order)
	return &api.LockOrderReply{
		RPCResult: resultFromCode(types.ResponseSuccess),
		Data: &api.LockMarketPayOrderResponseDTO{
			OrderId:          order.OrderId,
			OriginalPrice:    types.NewMoney(order.OriginalPrice),
			DeductionPrice:   types.NewMoney(order.DeductionPrice),
			PayPrice:         types.NewMoney(order.PayPrice),
			TradeOrderStatus: int(order.TradeOrderStatus),
		},
	}, nil
}

// SettlementMarketPayOrder mirrors MarketTradeController.settlementMarketPayOrder.
func (s *TradeServer) SettlementMarketPayOrder(ctx context.Context, req *api.SettlementMarketPayOrderRequestDTO) (reply *api.SettlementReply, err error) {
	defer func() {
		recordRPC("TradeService", "SettlementMarketPayOrder", &reply.RPCResult, err)
		metrics.TradeOpsTotal.WithLabelValues("settlement", replyCode(&reply.RPCResult)).Inc()
	}()
	log.Printf("营销交易组队结算开始:%s outTradeNo:%s", req.UserId, req.OutTradeNo)
	if types.IsBlank(req.UserId) || types.IsBlank(req.Source) || types.IsBlank(req.Channel) ||
		types.IsBlank(req.OutTradeNo) || req.OutTradeTime.IsZero() {
		return &api.SettlementReply{RPCResult: resultFromCode(types.ResponseIllegalParam)}, nil
	}
	settlement, err := s.settlement.SettlementMarketPayOrder(ctx, &trade.TradePaySuccessEntity{
		Source:       req.Source,
		Channel:      req.Channel,
		UserId:       req.UserId,
		OutTradeNo:   req.OutTradeNo,
		OutTradeTime: req.OutTradeTime.ToTime(),
	})
	if err != nil {
		return &api.SettlementReply{RPCResult: resultFromError(err)}, nil
	}
	log.Printf("营销交易组队结算完成:%s outTradeNo:%s", req.UserId, req.OutTradeNo)
	return &api.SettlementReply{
		RPCResult: resultFromCode(types.ResponseSuccess),
		Data: &api.SettlementMarketPayOrderResponseDTO{
			UserId:     settlement.UserId,
			TeamId:     settlement.TeamId,
			ActivityId: settlement.ActivityId,
			OutTradeNo: settlement.OutTradeNo,
		},
	}, nil
}

// RefundMarketPayOrder mirrors MarketTradeController.refundMarketPayOrder.
func (s *TradeServer) RefundMarketPayOrder(ctx context.Context, req *api.RefundMarketPayOrderRequestDTO) (reply *api.RefundReply, err error) {
	defer func() {
		recordRPC("TradeService", "RefundMarketPayOrder", &reply.RPCResult, err)
		metrics.TradeOpsTotal.WithLabelValues("refund", replyCode(&reply.RPCResult)).Inc()
	}()
	log.Printf("营销拼团退单开始:%s outTradeNo:%s", req.UserId, req.OutTradeNo)
	if types.IsBlank(req.UserId) || types.IsBlank(req.OutTradeNo) ||
		types.IsBlank(req.Source) || types.IsBlank(req.Channel) {
		return &api.RefundReply{RPCResult: resultFromCode(types.ResponseIllegalParam)}, nil
	}
	behavior, err := s.refund.RefundOrder(ctx, &trade.TradeRefundCommandEntity{
		UserId:     req.UserId,
		OutTradeNo: req.OutTradeNo,
		Source:     req.Source,
		Channel:    req.Channel,
	})
	if err != nil {
		return &api.RefundReply{RPCResult: resultFromError(err)}, nil
	}
	log.Printf("营销拼团退单完成:%s outTradeNo:%s", req.UserId, req.OutTradeNo)
	return &api.RefundReply{
		RPCResult: resultFromCode(types.ResponseSuccess),
		Data: &api.RefundMarketPayOrderResponseDTO{
			UserId:  behavior.UserId,
			OrderId: behavior.OrderId,
			TeamId:  behavior.TeamId,
			Code:    string(behavior.Behavior),
			Info:    behavior.Behavior.Info(),
		},
	}, nil
}
