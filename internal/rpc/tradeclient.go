package rpc

import (
	"context"

	"github.com/hangtiancheng/be261028/internal/api"
	yukinorpc "github.com/hangtiancheng/yukino.go/libs/yukino_rpc/pkg/rpc"
)

// TradeClient is the yukino_rpc client for the trade service, used by the gateway.
type TradeClient struct {
	conn *yukinorpc.ClientConn
}

// NewTradeClient wraps an established client connection.
func NewTradeClient(conn *yukinorpc.ClientConn) *TradeClient {
	return &TradeClient{conn: conn}
}

func (c *TradeClient) invoke(ctx context.Context, method string, args, reply any) error {
	return c.conn.Invoke(ctx, api.TradeServiceName, method, args, reply)
}

// LockMarketPayOrder calls the remote lock flow.
func (c *TradeClient) LockMarketPayOrder(ctx context.Context, req *api.LockMarketPayOrderRequestDTO) (*api.LockOrderReply, error) {
	var reply api.LockOrderReply
	if err := c.invoke(ctx, api.MethodLockMarketPayOrder, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// SettlementMarketPayOrder calls the remote settlement flow.
func (c *TradeClient) SettlementMarketPayOrder(ctx context.Context, req *api.SettlementMarketPayOrderRequestDTO) (*api.SettlementReply, error) {
	var reply api.SettlementReply
	if err := c.invoke(ctx, api.MethodSettlementMarketPayOrder, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// RefundMarketPayOrder calls the remote refund flow.
func (c *TradeClient) RefundMarketPayOrder(ctx context.Context, req *api.RefundMarketPayOrderRequestDTO) (*api.RefundReply, error) {
	var reply api.RefundReply
	if err := c.invoke(ctx, api.MethodRefundMarketPayOrder, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}
