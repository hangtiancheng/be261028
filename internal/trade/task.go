package trade

import (
	"context"
	"log"

	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/hangtiancheng/be261028/internal/types"
)

// TradeTaskService mirrors the legacy TradeTaskService: it runs notify tasks
// (HTTP callbacks or Kafka messages) and updates their status based on the result.
type TradeTaskService struct {
	repo     *TradeRepository
	port     *TradePort
	maxRetry int
}

// NewTradeTaskService wires the service. maxRetry is the notify_count threshold
// above which a failing task is marked errored instead of retried (legacy used 4).
func NewTradeTaskService(repo *TradeRepository, port *TradePort, maxRetry int) *TradeTaskService {
	if maxRetry <= 0 {
		maxRetry = 4
	}
	return &TradeTaskService{repo: repo, port: port, maxRetry: maxRetry}
}

// ExecNotifyJob runs all unexecuted notify tasks (the scheduled job entry point).
func (s *TradeTaskService) ExecNotifyJob(ctx context.Context) (map[string]int, error) {
	log.Printf("拼团交易-执行回调通知任务")
	tasks, err := s.repo.QueryUnExecutedNotifyTaskList(ctx)
	if err != nil {
		return nil, err
	}
	return s.execNotifyJobList(ctx, tasks)
}

// ExecNotifyJobByTeamId runs the unexecuted notify task for a specific team.
func (s *TradeTaskService) ExecNotifyJobByTeamId(ctx context.Context, teamId string) (map[string]int, error) {
	log.Printf("拼团交易-执行回调通知回调，指定 teamId:%s", teamId)
	tasks, err := s.repo.QueryUnExecutedNotifyTaskListByTeamId(ctx, teamId)
	if err != nil {
		return nil, err
	}
	return s.execNotifyJobList(ctx, tasks)
}

// ExecNotifyJobTask runs a single notify task (used by the async settlement/refund
// callback path).
func (s *TradeTaskService) ExecNotifyJobTask(ctx context.Context, task *NotifyTaskEntity) (map[string]int, error) {
	log.Printf("拼团交易-执行回调通知回调，指定 teamId:%s notifyTaskId:%d", task.TeamId, task.Id)
	return s.execNotifyJobList(ctx, []*NotifyTaskEntity{task})
}

func (s *TradeTaskService) execNotifyJobList(ctx context.Context, tasks []*NotifyTaskEntity) (map[string]int, error) {
	successCount, errorCount, retryCount := 0, 0, 0
	for _, task := range tasks {
		response, err := s.port.GroupBuyNotify(ctx, task)
		if err != nil {
			log.Printf("回调通知执行异常 teamId:%s err:%v", task.TeamId, err)
			metrics.NotifyTasksTotal.WithLabelValues("error").Inc()
			continue
		}
		switch response {
		case types.NotifyTaskSuccess:
			n, err := s.repo.UpdateNotifyTaskStatusSuccess(ctx, task.Id)
			if err != nil {
				log.Printf("更新回调任务成功状态失败 id:%d err:%v", task.Id, err)
			} else if n == 1 {
				successCount++
				metrics.NotifyTasksTotal.WithLabelValues("success").Inc()
			}
		case types.NotifyTaskError:
			if task.NotifyCount > s.maxRetry {
				n, err := s.repo.UpdateNotifyTaskStatusError(ctx, task.Id)
				if err != nil {
					log.Printf("更新回调任务失败状态失败 id:%d err:%v", task.Id, err)
				} else if n == 1 {
					errorCount++
					metrics.NotifyTasksTotal.WithLabelValues("error").Inc()
				}
			} else {
				n, err := s.repo.UpdateNotifyTaskStatusRetry(ctx, task.Id)
				if err != nil {
					log.Printf("更新回调任务重试状态失败 id:%d err:%v", task.Id, err)
				} else if n == 1 {
					retryCount++
					metrics.NotifyTasksTotal.WithLabelValues("retry").Inc()
				}
			}
		default:
			// NotifyTaskNull: lock not acquired or callback failed; leave pending.
			metrics.NotifyTasksTotal.WithLabelValues("null").Inc()
		}
	}
	return map[string]int{
		"waitCount":    len(tasks),
		"successCount": successCount,
		"errorCount":   errorCount,
		"retryCount":   retryCount,
	}, nil
}
