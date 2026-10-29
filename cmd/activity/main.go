// Command activity runs the be261028 activity microservice: a yukino_rpc server
// providing activity administration, the market-index trial tree, discount plans
// and crowd-tag operations, backed by MySQL, Redis and the dynamic config center.
package main

import (
	"context"
	"log"

	"github.com/hangtiancheng/be261028/internal/activity"
	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/bootstrap"
	"github.com/hangtiancheng/be261028/internal/dao"
	"github.com/hangtiancheng/be261028/internal/infra/db"
	"github.com/hangtiancheng/be261028/internal/infra/dcc"
	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/rpc"
)

func main() {
	cfg, err := bootstrap.LoadConfig(bootstrap.ConfigPath("configs/activity.yaml"))
	if err != nil {
		log.Fatalf("[activity] 加载配置失败: %v", err)
	}
	log.Printf("[activity] 启动服务 name:%s rpc:%s metrics:%s", cfg.Service.Name, cfg.Service.RPCAddr, cfg.Service.MetricsAddr)

	ctx := bootstrap.SignalContext()

	// MySQL
	gdb, err := db.New(cfg.MySQL)
	if err != nil {
		log.Fatalf("[activity] 连接 MySQL 失败: %v", err)
	}
	defer db.Close(gdb) //nolint:errcheck

	// Redis
	redis, err := redisx.New(cfg.Redis)
	if err != nil {
		log.Fatalf("[activity] 连接 Redis 失败: %v", err)
	}
	defer redis.Close()

	// DCC（订阅动态配置）
	dccService := dcc.New(cfg.DCC.Topic, cfg.DCC.Defaults)
	go dccService.Subscribe(context.Background(), redis)

	// DAO + 仓储
	d := dao.New()
	activityRepo := activity.NewActivityRepository(gdb, d, redis, dccService)
	manageRepo := activity.NewActivityManageRepository(gdb, d, redis)
	tagRepo := activity.NewTagRepository(gdb, d, redis)

	// 领域服务
	discounts := activity.NewDiscountRegistry(activityRepo)
	trial := activity.NewTrialService(activityRepo, discounts)
	indexService := activity.NewIndexGroupBuyMarketService(activityRepo, trial)
	manageService := activity.NewActivityManageService(manageRepo)
	tagService := activity.NewTagService(tagRepo)

	// RPC 服务
	activityServer := rpc.NewActivityServer(manageService, indexService, tagService)
	rpcServer, err := bootstrap.ServeRPC(cfg.Service.RPCAddr, api.ActivityServiceName, activityServer)
	if err != nil {
		log.Fatalf("[activity] 启动 RPC 服务失败: %v", err)
	}

	// 指标/健康
	metricsSrv, err := metrics.Server(cfg.Service.MetricsAddr)
	if err != nil {
		log.Fatalf("[activity] 启动 metrics 服务失败: %v", err)
	}

	// 等待退出信号
	<-ctx.Done()
	log.Printf("[activity] 收到退出信号，开始优雅停机")
	rpcServer.GracefulStop()
	metrics.Shutdown(metricsSrv)
	log.Printf("[activity] 已停机")
}
