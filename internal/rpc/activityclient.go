package rpc

import (
	"context"

	"github.com/hangtiancheng/be261028/internal/api"
	yukinorpc "github.com/hangtiancheng/yukino.go/libs/yukino_rpc/pkg/rpc"
)

// ActivityClient is the yukino_rpc client for the activity service. The gateway
// uses it for admin and index endpoints; the trade service uses it for the trial
// calculation during order locking.
type ActivityClient struct {
	conn *yukinorpc.ClientConn
}

// NewActivityClient wraps an established client connection.
func NewActivityClient(conn *yukinorpc.ClientConn) *ActivityClient {
	return &ActivityClient{conn: conn}
}

func (c *ActivityClient) invoke(ctx context.Context, method string, args, reply any) error {
	return c.conn.Invoke(ctx, api.ActivityServiceName, method, args, reply)
}

// CreateActivity calls the remote admin create.
func (c *ActivityClient) CreateActivity(ctx context.Context, req *api.ActivityManageRequestDTO) (*api.ActivityManageReply, error) {
	var reply api.ActivityManageReply
	if err := c.invoke(ctx, api.MethodCreateActivity, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// UpdateActivity calls the remote admin update.
func (c *ActivityClient) UpdateActivity(ctx context.Context, req *api.ActivityManageRequestDTO) (*api.ActivityManageReply, error) {
	var reply api.ActivityManageReply
	if err := c.invoke(ctx, api.MethodUpdateActivity, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// DeleteActivity calls the remote admin delete.
func (c *ActivityClient) DeleteActivity(ctx context.Context, req *api.ActivityIdRequest) (*api.ActivityManageReply, error) {
	var reply api.ActivityManageReply
	if err := c.invoke(ctx, api.MethodDeleteActivity, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// GetActivityDetail calls the remote admin detail query.
func (c *ActivityClient) GetActivityDetail(ctx context.Context, req *api.ActivityIdRequest) (*api.ActivityManageReply, error) {
	var reply api.ActivityManageReply
	if err := c.invoke(ctx, api.MethodGetActivityDetail, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// GetActivityList calls the remote admin list query.
func (c *ActivityClient) GetActivityList(ctx context.Context) (*api.ActivityListReply, error) {
	var reply api.ActivityListReply
	if err := c.invoke(ctx, api.MethodGetActivityList, &api.EmptyRequest{}, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// EnableActivity calls the remote admin enable.
func (c *ActivityClient) EnableActivity(ctx context.Context, req *api.ActivityIdRequest) (*api.ActivityManageReply, error) {
	var reply api.ActivityManageReply
	if err := c.invoke(ctx, api.MethodEnableActivity, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// DisableActivity calls the remote admin disable.
func (c *ActivityClient) DisableActivity(ctx context.Context, req *api.ActivityIdRequest) (*api.ActivityManageReply, error) {
	var reply api.ActivityManageReply
	if err := c.invoke(ctx, api.MethodDisableActivity, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// QueryGroupBuyMarketConfig calls the remote market index orchestration.
func (c *ActivityClient) QueryGroupBuyMarketConfig(ctx context.Context, req *api.GoodsMarketRequestDTO) (*api.MarketConfigReply, error) {
	var reply api.MarketConfigReply
	if err := c.invoke(ctx, api.MethodQueryGroupBuyMarketConfig, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// IndexMarketTrial calls the remote trial tree.
func (c *ActivityClient) IndexMarketTrial(ctx context.Context, req *api.MarketProductEntity) (*api.MarketTrialReply, error) {
	var reply api.MarketTrialReply
	if err := c.invoke(ctx, api.MethodIndexMarketTrial, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}

// ExecTagBatchJob calls the remote crowd-tag batch job.
func (c *ActivityClient) ExecTagBatchJob(ctx context.Context, req *api.TagBatchJobRequest) (*api.RPCResult, error) {
	var reply api.RPCResult
	if err := c.invoke(ctx, api.MethodExecTagBatchJob, req, &reply); err != nil {
		return nil, err
	}
	return &reply, nil
}
