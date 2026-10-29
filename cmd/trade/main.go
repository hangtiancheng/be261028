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
		log.Fatalf("[trade] failed to load config: %v", err)
	}
	log.Printf("[trade] starting service name:%s rpc:%s metrics:%s", cfg.Service.Name, cfg.Service.RPCAddr, cfg.Service.MetricsAddr)

	ctx := bootstrap.SignalContext()

	// MySQL
	gdb, err := db.New(cfg.MySQL)
	if err != nil {
		log.Fatalf("[trade] failed to connect to MySQL: %v", err)
	}
	defer db.Close(gdb) //nolint:errcheck

	// Redis
	redis, err := redisx.New(cfg.Redis)
	if err != nil {
		log.Fatalf("[trade] failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	// Kafka topics (the broker has auto-create disabled, so create them explicitly at startup).
	if err := kafkax.EnsureTopics(cfg.Kafka.Brokers, cfg.Kafka.ClientID,
		[]string{cfg.Kafka.TopicTeamSuccess, cfg.Kafka.TopicTeamRefund}); err != nil {
		log.Fatalf("[trade] failed to create Kafka topics: %v", err)
	}

	// Kafka producer
	producer, err := kafkax.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.ClientID)
	if err != nil {
		log.Fatalf("[trade] failed to create the Kafka producer: %v", err)
	}
	defer producer.Close()

	// DCC (subscribe to dynamic config).
	dccService := dcc.New(cfg.DCC.Topic, cfg.DCC.Defaults)
	go dccService.Subscribe(context.Background(), redis)

	// DAO + repository.
	d := dao.New()
	tradeRepo := trade.NewTradeRepository(gdb, d, redis, dccService, cfg.Kafka.TopicTeamSuccess, cfg.Kafka.TopicTeamRefund)

	// activity RPC client (lock-order trial callback).
	activityConn, err := yukinorpc.Dial(cfg.RPC.ActivityTarget, yukinorpc.WithTimeout(cfg.RPC.Timeout))
	if err != nil {
		log.Fatalf("[trade] failed to connect to the activity service: %v", err)
	}
	defer activityConn.Close()
	activityCli := rpc.NewActivityClient(activityConn)

	// Trade port + task service.
	port := trade.NewTradePort(redis, producer, cfg.Notify.CallbackTimeout, cfg.Notify.HTTPLockWait, cfg.Notify.HTTPLockLease)
	taskService := trade.NewTradeTaskService(tradeRepo, port, cfg.Notify.MaxRetry)

	// Rule chains + strategies.
	lockChain := trade.NewLockRuleChain(tradeRepo)
	settlementChain := trade.NewSettlementRuleChain(tradeRepo)
	strategies := trade.NewRefundStrategies(tradeRepo, taskService)
	refundChain := trade.NewRefundRuleChain(tradeRepo, strategies)

	// Domain services.
	lockService := trade.NewTradeLockOrderService(tradeRepo, lockChain)
	settlementService := trade.NewTradeSettlementOrderService(tradeRepo, settlementChain, taskService)
	refundService := trade.NewTradeRefundOrderService(refundChain, strategies)

	// RPC service.
	tradeServer := rpc.NewTradeServer(lockService, settlementService, refundService, activityCli)
	rpcServer, err := bootstrap.ServeRPC(cfg.Service.RPCAddr, api.TradeServiceName, tradeServer)
	if err != nil {
		log.Fatalf("[trade] failed to start the RPC service: %v", err)
	}

	// Kafka consumer: team_refund -> restore locked team stock (defect fix); team_success -> log.
	consumer, err := kafkax.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, cfg.Kafka.ClientID,
		[]string{cfg.Kafka.TopicTeamRefund, cfg.Kafka.TopicTeamSuccess})
	if err != nil {
		log.Fatalf("[trade] failed to create the Kafka consumer: %v", err)
	}
	consumer.SetOnError(func(topic string, value []byte, err error) {
		log.Printf("[trade] Kafka consume failed topic:%s value:%s err:%v", topic, string(value), err)
	})
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	go func() {
		if err := consumer.Run(consumerCtx, messageHandler(refundService, cfg)); err != nil {
			log.Printf("[trade] Kafka consumer exited: %v", err)
		}
	}()

	// Scheduled callback notify job.
	jobCtx, cancelJob := context.WithCancel(context.Background())
	go runNotifyJob(jobCtx, cfg, redis, taskService)

	// Metrics / health.
	metricsSrv, err := metrics.Server(cfg.Service.MetricsAddr)
	if err != nil {
		log.Fatalf("[trade] failed to start the metrics service: %v", err)
	}

	// Wait for the shutdown signal.
	<-ctx.Done()
	log.Printf("[trade] shutdown signal received, starting graceful stop")
	cancelJob()
	cancelConsumer()
	consumer.Close()
	rpcServer.GracefulStop()
	metrics.Shutdown(metricsSrv)
	log.Printf("[trade] stopped")
}

// messageHandler dispatches consumed Kafka messages by topic.
func messageHandler(refundService *trade.TradeRefundOrderService, cfg *config.Config) kafkax.Handler {
	return func(ctx context.Context, topic string, value []byte) error {
		switch topic {
		case cfg.Kafka.TopicTeamRefund:
			var msg trade.TeamRefundSuccess
			if err := json.Unmarshal(value, &msg); err != nil {
				log.Printf("[trade] failed to parse the refund message value:%s err:%v", string(value), err)
				return nil // skip malformed messages to avoid a poison-message loop
			}
			metrics.KafkaMessagesTotal.WithLabelValues(topic, "consume").Inc()
			// Restore locked team stock (retries transient errors; the operation itself is idempotent).
			return restoreWithRetry(ctx, refundService, &msg)
		case cfg.Kafka.TopicTeamSuccess:
			log.Printf("[trade] received team-success message:%s", string(value))
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
			log.Printf("[trade] failed to restore locked team stock (attempt %d) teamId:%s orderId:%s err:%v", attempt+1, msg.TeamId, msg.OrderId, err)
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
				log.Printf("[trade] scheduled job failed to acquire the lock: %v", err)
				cancel()
				continue
			}
			if !acquired {
				cancel()
				continue
			}
			result, err := taskService.ExecNotifyJob(lockCtx)
			if err != nil {
				log.Printf("[trade] scheduled job: notify group-buy task failed: %v", err)
			} else {
				log.Printf("[trade] scheduled job: notify complete result:%v", result)
			}
			if err := lock.Unlock(lockCtx); err != nil {
				log.Printf("[trade] scheduled job failed to release the lock: %v", err)
			}
			cancel()
		}
	}
}
