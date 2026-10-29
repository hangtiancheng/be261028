package api

import (
	"github.com/hangtiancheng/be261028/internal/types"
)

// NotifyConfigVO mirrors LockMarketPayOrderRequestDTO.NotifyConfigVO.
type NotifyConfigVO struct {
	NotifyType string `json:"notifyType"`
	NotifyMQ   string `json:"notifyMQ,omitempty"`
	NotifyUrl  string `json:"notifyUrl,omitempty"`
}

// LockMarketPayOrderRequestDTO mirrors com.buy.group.easy.api.dto.LockMarketPayOrderRequestDTO.
type LockMarketPayOrderRequestDTO struct {
	UserId         string          `json:"userId"`
	TeamId         string          `json:"teamId"`
	ActivityId     int64           `json:"activityId"`
	GoodsId        string          `json:"goodsId"`
	Source         string          `json:"source"`
	Channel        string          `json:"channel"`
	OutTradeNo     string          `json:"outTradeNo"`
	NotifyConfigVO *NotifyConfigVO `json:"notifyConfigVO"`
}

// LockMarketPayOrderResponseDTO mirrors com.buy.group.easy.api.dto.LockMarketPayOrderResponseDTO.
type LockMarketPayOrderResponseDTO struct {
	OrderId          string      `json:"orderId"`
	OriginalPrice    types.Money `json:"originalPrice"`
	DeductionPrice   types.Money `json:"deductionPrice"`
	PayPrice         types.Money `json:"payPrice"`
	TradeOrderStatus int         `json:"tradeOrderStatus"`
}

// SettlementMarketPayOrderRequestDTO mirrors com.buy.group.easy.api.dto.SettlementMarketPayOrderRequestDTO.
type SettlementMarketPayOrderRequestDTO struct {
	Source       string         `json:"source"`
	Channel      string         `json:"channel"`
	UserId       string         `json:"userId"`
	OutTradeNo   string         `json:"outTradeNo"`
	OutTradeTime types.JSONTime `json:"outTradeTime"`
}

// SettlementMarketPayOrderResponseDTO mirrors com.buy.group.easy.api.dto.SettlementMarketPayOrderResponseDTO.
type SettlementMarketPayOrderResponseDTO struct {
	UserId     string `json:"userId"`
	TeamId     string `json:"teamId"`
	ActivityId int64  `json:"activityId"`
	OutTradeNo string `json:"outTradeNo"`
}

// RefundMarketPayOrderRequestDTO mirrors com.buy.group.easy.api.dto.RefundMarketPayOrderRequestDTO.
type RefundMarketPayOrderRequestDTO struct {
	UserId     string `json:"userId"`
	OutTradeNo string `json:"outTradeNo"`
	Source     string `json:"source"`
	Channel    string `json:"channel"`
}

// RefundMarketPayOrderResponseDTO mirrors com.buy.group.easy.api.dto.RefundMarketPayOrderResponseDTO.
type RefundMarketPayOrderResponseDTO struct {
	UserId  string `json:"userId"`
	OrderId string `json:"orderId"`
	TeamId  string `json:"teamId"`
	Code    string `json:"code"`
	Info    string `json:"info"`
}

// NotifyRequestDTO mirrors com.buy.group.easy.api.dto.NotifyRequestDTO, the payload
// posted to third-party callback URLs and to the test endpoint.
type NotifyRequestDTO struct {
	TeamId         string   `json:"teamId"`
	OutTradeNoList []string `json:"outTradeNoList"`
}
