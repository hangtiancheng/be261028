package trade

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hangtiancheng/be261028/internal/infra/kafkax"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/types"
)

// TradePort mirrors the legacy TradePort: it dispatches a notify task either as an
// HTTP callback or a Kafka message, guarded by a per-team distributed lock so the
// same task is not executed twice across instances.
type TradePort struct {
	redis      *redisx.RedisService
	producer   *kafkax.Producer
	httpClient *http.Client
	lockWait   time.Duration
	lockLease  time.Duration
}

// NewTradePort wires the port.
func NewTradePort(redis *redisx.RedisService, producer *kafkax.Producer, callbackTimeout, lockWait, lockLease time.Duration) *TradePort {
	return &TradePort{
		redis:      redis,
		producer:   producer,
		httpClient: &http.Client{Timeout: callbackTimeout},
		lockWait:   lockWait,
		lockLease:  lockLease,
	}
}

// GroupBuyNotify mirrors TradePort.groupBuyNotify. It returns the callback result:
// "success"/"error" (or the raw HTTP body) on completion, or the empty NULL result
// when the lock could not be acquired or the callback failed.
func (p *TradePort) GroupBuyNotify(ctx context.Context, task *NotifyTaskEntity) (types.NotifyTaskResult, error) {
	lock := p.redis.NewLock(task.LockKey())
	acquired, err := lock.TryLock(ctx, p.lockWait, p.lockLease)
	if err != nil {
		log.Printf("[trade] notify task failed to acquire the lock lockKey:%s err:%v", task.LockKey(), err)
		return types.NotifyTaskNull, nil
	}
	if !acquired {
		return types.NotifyTaskNull, nil
	}
	defer func() {
		if err := lock.Unlock(ctx); err != nil {
			log.Printf("[trade] notify task failed to release the lock lockKey:%s err:%v", task.LockKey(), err)
		}
	}()

	switch task.NotifyType {
	case types.NotifyTypeHTTP:
		// An invalid notifyUrl is treated as success with no callback.
		if types.IsBlank(task.NotifyUrl) || task.NotifyUrl == notifyUrlNone {
			return types.NotifyTaskSuccess, nil
		}
		body, err := p.httpNotify(ctx, task.NotifyUrl, task.ParameterJson)
		if err != nil {
			log.Printf("[trade] group-buy HTTP callback failed url:%s err:%v", task.NotifyUrl, err)
			return types.NotifyTaskNull, nil
		}
		return types.NotifyTaskResult(strings.TrimSpace(body)), nil
	case types.NotifyTypeMQ:
		if err := p.producer.Publish(ctx, task.NotifyMQ, []byte(task.ParameterJson)); err != nil {
			log.Printf("[trade] failed to publish MQ message topic:%s message:%s err:%v", task.NotifyMQ, task.ParameterJson, err)
			return types.NotifyTaskNull, nil
		}
		return types.NotifyTaskSuccess, nil
	}
	return types.NotifyTaskNull, nil
}

// notifyUrlNone is the placeholder value the legacy system stored in notify_url to
// mean "no callback configured". It is compared against potentially-persisted data,
// so the literal is preserved verbatim for parity with legacy rows and clients;
// translating it would cause such values to be dialed as real URLs.
const notifyUrlNone = "暂无"

// httpNotify POSTs the parameter JSON to the callback URL and returns the body.
func (p *TradePort) httpNotify(ctx context.Context, url, body string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
