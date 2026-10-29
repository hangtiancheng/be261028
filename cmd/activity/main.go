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
		log.Fatalf("[activity] failed to load config: %v", err)
	}
	log.Printf("[activity] starting service name:%s rpc:%s metrics:%s", cfg.Service.Name, cfg.Service.RPCAddr, cfg.Service.MetricsAddr)

	ctx := bootstrap.SignalContext()

	// MySQL
	gdb, err := db.New(cfg.MySQL)
	if err != nil {
		log.Fatalf("[activity] failed to connect to MySQL: %v", err)
	}
	defer db.Close(gdb) //nolint:errcheck

	// Redis
	redis, err := redisx.New(cfg.Redis)
	if err != nil {
		log.Fatalf("[activity] failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// DCC (subscribe to dynamic config).
	dccService := dcc.New(cfg.DCC.Topic, cfg.DCC.Defaults)
	go dccService.Subscribe(context.Background(), redis)

	// DAO + repositories.
	d := dao.New()
	activityRepo := activity.NewActivityRepository(gdb, d, redis, dccService)
	manageRepo := activity.NewActivityManageRepository(gdb, d, redis)
	tagRepo := activity.NewTagRepository(gdb, d, redis)

	// Domain services.
	discounts := activity.NewDiscountRegistry(activityRepo)
	trial := activity.NewTrialService(activityRepo, discounts)
	indexService := activity.NewIndexGroupBuyMarketService(activityRepo, trial)
	manageService := activity.NewActivityManageService(manageRepo)
	tagService := activity.NewTagService(tagRepo)

	// RPC service.
	activityServer := rpc.NewActivityServer(manageService, indexService, tagService)
	rpcServer, err := bootstrap.ServeRPC(cfg.Service.RPCAddr, api.ActivityServiceName, activityServer)
	if err != nil {
		log.Fatalf("[activity] failed to start the RPC service: %v", err)
	}

	// Metrics / health.
	metricsSrv, err := metrics.Server(cfg.Service.MetricsAddr)
	if err != nil {
		log.Fatalf("[activity] failed to start the metrics service: %v", err)
	}

	// Wait for the shutdown signal.
	<-ctx.Done()
	log.Printf("[activity] shutdown signal received, starting graceful stop")
	rpcServer.GracefulStop()
	metrics.Shutdown(metricsSrv)
	log.Printf("[activity] stopped")
}
