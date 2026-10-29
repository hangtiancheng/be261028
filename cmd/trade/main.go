// Command trade runs the be261028 trade microservice: a yukino_rpc server for
// order locking, settlement and refunds, a Kafka producer/consumer pair replacing
// the legacy RabbitMQ topics, and the scheduled notify-task job.
//
// Defect fix: the legacy project published team-refund messages but never consumed
// them, so restoreTeamLockStock was dead code and refunded team stock was never
// recovered. This service consumes the team-refund topic and closes that loop.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/bootstrap"
	"github.com/hangtiancheng/be261028/internal/config"
	"github.com/hangtiancheng/be261028/internal/dao"
	"github.com/hangtiancheng/be261028/internal/infra/db"
	"github.com/hangtiancheng/be261028/internal/infra/dcc"
	"github.com/hangtiancheng/be261028/internal/infra/kafkax"
	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/rpc"
	"github.com/hangtiancheng/be261028/internal/trade"
	yukinorpc "github.com/hangtiancheng/yukino.go/libs/yukino_rpc/pkg/rpc"
)

func main() {
	cfg, err := bootstrap.LoadConfig(bootstrap.ConfigPath("configs/trade.yaml"))
	if err != nil {
		log.Fatalf("[trade] 加载配置失败: %v", err)
	}
	log.Printf("[trade] 启动服务 name:%s rpc:%s metrics:%s", cfg.Service.Name, cfg.Service.RPCAddr, cfg.Service.MetricsAddr)

	ctx := bootstrap.SignalContext()

	// MySQL
	gdb, err := db.New(cfg.MySQL)
	if err != nil {
		log.Fatalf("[trade] 连接 MySQL 失败: %v", err)
	}
	defer db.Close(gdb) //nolint:errcheck

	// Redis
	redis, err := redisx.New(cfg.Redis)
	if err != nil {
		log.Fatalf("[trade] 连接 Redis 失败: %v", err)
	}
	defer redis.Close()

	// Kafka 主题（broker 未开启自动创建，启动时显式创建）
	if err := kafkax.EnsureTopics(cfg.Kafka.Brokers, cfg.Kafka.ClientID,
		[]string{cfg.Kafka.TopicTeamSuccess, cfg.Kafka.TopicTeamRefund}); err != nil {
		log.Fatalf("[trade] 创建 Kafka 主题失败: %v", err)
	}

	// Kafka producer
	producer, err := kafkax.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.ClientID)
	if err != nil {
		log.Fatalf("[trade] 创建 Kafka 生产者失败: %v", err)
	}
	defer producer.Close()

	// DCC（订阅动态配置）
	dccService := dcc.New(cfg.DCC.Topic, cfg.DCC.Defaults)
	go dccService.Subscribe(context.Background(), redis)

	// DAO + 仓储
	d := dao.New()
	tradeRepo := trade.NewTradeRepository(gdb, d, redis, dccService, cfg.Kafka.TopicTeamSuccess, cfg.Kafka.TopicTeamRefund)

	// activity RPC 客户端（锁单试算回调）
	activityConn, err := yukinorpc.Dial(cfg.RPC.ActivityTarget, yukinorpc.WithTimeout(cfg.RPC.Timeout))
	if err != nil {
		log.Fatalf("[trade] 连接 activity 服务失败: %v", err)
	}
	defer activityConn.Close()
	activityCli := rpc.NewActivityClient(activityConn)

	// 交易端口 + 任务服务
	port := trade.NewTradePort(redis, producer, cfg.Notify.CallbackTimeout, cfg.Notify.HTTPLockWait, cfg.Notify.HTTPLockLease)
	taskService := trade.NewTradeTaskService(tradeRepo, port, cfg.Notify.MaxRetry)

	// 规则链 + 策略
	lockChain := trade.NewLockRuleChain(tradeRepo)
	settlementChain := trade.NewSettlementRuleChain(tradeRepo)
	strategies := trade.NewRefundStrategies(tradeRepo, taskService)
	refundChain := trade.NewRefundRuleChain(tradeRepo, strategies)

	// 领域服务
	lockService := trade.NewTradeLockOrderService(tradeRepo, lockChain)
	settlementService := trade.NewTradeSettlementOrderService(tradeRepo, settlementChain, taskService)
	refundService := trade.NewTradeRefundOrderService(refundChain, strategies)

	// RPC 服务
	tradeServer := rpc.NewTradeServer(lockService, settlementService, refundService, activityCli)
	rpcServer, err := bootstrap.ServeRPC(cfg.Service.RPCAddr, api.TradeServiceName, tradeServer)
	if err != nil {
		log.Fatalf("[trade] 启动 RPC 服务失败: %v", err)
	}

	// Kafka 消费者：team_refund -> 恢复锁单库存（缺陷修复）；team_success -> 记录
	consumer, err := kafkax.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, cfg.Kafka.ClientID,
		[]string{cfg.Kafka.TopicTeamRefund, cfg.Kafka.TopicTeamSuccess})
	if err != nil {
		log.Fatalf("[trade] 创建 Kafka 消费者失败: %v", err)
	}
	consumer.SetOnError(func(topic string, value []byte, err error) {
		log.Printf("[trade] Kafka 消费失败 topic:%s value:%s err:%v", topic, string(value), err)
	})
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	go func() {
		if err := consumer.Run(consumerCtx, messageHandler(refundService, cfg)); err != nil {
			log.Printf("[trade] Kafka 消费者退出: %v", err)
		}
	}()

	// 定时回调通知任务
	jobCtx, cancelJob := context.WithCancel(context.Background())
	go runNotifyJob(jobCtx, cfg, redis, taskService)

	// 指标/健康
	metricsSrv, err := metrics.Server(cfg.Service.MetricsAddr)
	if err != nil {
		log.Fatalf("[trade] 启动 metrics 服务失败: %v", err)
	}

	// 等待退出信号
	<-ctx.Done()
	log.Printf("[trade] 收到退出信号，开始优雅停机")
	cancelJob()
	cancelConsumer()
	consumer.Close()
	rpcServer.GracefulStop()
	metrics.Shutdown(metricsSrv)
	log.Printf("[trade] 已停机")
}

// messageHandler dispatches consumed Kafka messages by topic.
func messageHandler(refundService *trade.TradeRefundOrderService, cfg *config.Config) kafkax.Handler {
	return func(ctx context.Context, topic string, value []byte) error {
		switch topic {
		case cfg.Kafka.TopicTeamRefund:
			var msg trade.TeamRefundSuccess
			if err := json.Unmarshal(value, &msg); err != nil {
				log.Printf("[trade] 退单消息解析失败 value:%s err:%v", string(value), err)
				return nil // 跳过畸形消息，避免毒消息循环
			}
			metrics.KafkaMessagesTotal.WithLabelValues(topic, "consume").Inc()
			// 恢复锁单库存（含瞬时错误重试，处理本身幂等）
			return restoreWithRetry(ctx, refundService, &msg)
		case cfg.Kafka.TopicTeamSuccess:
			log.Printf("[trade] 接收成团消息:%s", string(value))
			metrics.KafkaMessagesTotal.WithLabelValues(topic, "consume").Inc()
		}
		return nil
	}
}

// restoreWithRetry retries the idempotent stock restoration a few times to ride
// out transient Redis/DB errors before giving up.
func restoreWithRetry(ctx context.Context, refundService *trade.TradeRefundOrderService, msg *trade.TeamRefundSuccess) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := refundService.RestoreTeamLockStock(ctx, msg); err != nil {
			lastErr = err
			log.Printf("[trade] 恢复锁单库存失败(第%d次) teamId:%s orderId:%s err:%v", attempt+1, msg.TeamId, msg.OrderId, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 200 * time.Millisecond):
			}
			continue
		}
		return nil
	}
	return lastErr
}

// runNotifyJob periodically executes pending notify tasks under a distributed lock
// so only one instance runs per tick (mirrors GroupBuyNotifyJob).
func runNotifyJob(ctx context.Context, cfg *config.Config, redis *redisx.RedisService, taskService *trade.TradeTaskService) {
	ticker := time.NewTicker(cfg.Notify.JobInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lock := redis.NewLock(trade.NotifyJobExecLockKey())
			lockCtx, cancel := context.WithTimeout(ctx, cfg.Notify.CallbackTimeout+30*time.Second)
			acquired, err := lock.TryLock(lockCtx, 3*time.Second, 2*time.Minute)
			if err != nil {
				log.Printf("[trade] 定时任务抢锁失败: %v", err)
				cancel()
				continue
			}
			if !acquired {
				cancel()
				continue
			}
			result, err := taskService.ExecNotifyJob(lockCtx)
			if err != nil {
				log.Printf("[trade] 定时任务，回调通知拼团任务失败: %v", err)
			} else {
				log.Printf("[trade] 定时任务，回调通知完成 result:%v", result)
			}
			if err := lock.Unlock(lockCtx); err != nil {
				log.Printf("[trade] 定时任务释放锁失败: %v", err)
			}
			cancel()
		}
	}
}
