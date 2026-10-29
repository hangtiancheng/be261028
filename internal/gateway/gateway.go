// Package gateway implements the be261028 HTTP edge: it exposes the legacy REST
// endpoints on top of yukino_http and forwards to the activity and trade services
// over yukino_rpc. It mirrors the legacy trigger-layer controllers.
package gateway

import (
	"crypto/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/hangtiancheng/be261028/internal/api"
	"github.com/hangtiancheng/be261028/internal/infra/dcc"
	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/hangtiancheng/be261028/internal/infra/redisx"
	"github.com/hangtiancheng/be261028/internal/rpc"
	"github.com/hangtiancheng/be261028/internal/types"
	yukinohttp "github.com/hangtiancheng/yukino.go/libs/yukino_http"
)

// Gateway wires the RPC clients and cross-cutting concerns for the HTTP handlers.
type Gateway struct {
	activity     *rpc.ActivityClient
	trade        *rpc.TradeClient
	dcc          *dcc.DCC
	redis        *redisx.RedisService
	indexLimiter *RateLimiter
}

// New builds a Gateway.
func New(activity *rpc.ActivityClient, trade *rpc.TradeClient, dc *dcc.DCC, redis *redisx.RedisService) *Gateway {
	return &Gateway{
		activity:     activity,
		trade:        trade,
		dcc:          dc,
		redis:        redis,
		indexLimiter: NewRateLimiter(time.Second), // 1 req/s per userId, mirroring permitsPerSecond=1.0
	}
}

// Register wires all legacy routes onto the application.
func (g *Gateway) Register(app *yukinohttp.Application) {
	// Cross-cutting middleware: CORS + trace id + request body bound.
	app.Use(g.cors(), g.traceId(), g.bodyLimit())

	// Activity administration (AdminController: /api/v1/admin).
	admin := app.Router("/api/v1/admin")
	admin.Post("/activity/create", g.handle(g.createActivity))
	admin.Post("/activity/update", g.handle(g.updateActivity))
	admin.Post("/activity/delete/:activityId", g.handle(g.deleteActivity))
	admin.Get("/activity/detail/:activityId", g.handle(g.getActivityDetail))
	admin.Get("/activity/list", g.handle(g.getActivityList))
	admin.Post("/activity/enable/:activityId", g.handle(g.enableActivity))
	admin.Post("/activity/disable/:activityId", g.handle(g.disableActivity))

	// Dynamic config (DCCController: /api/v1/gbm/dcc).
	dccGroup := app.Router("/api/v1/gbm/dcc")
	dccGroup.Get("/update_config", g.handle(g.updateConfig))

	// Market index (MarketIndexController: /api/v1/gbm/index).
	indexGroup := app.Router("/api/v1/gbm/index")
	indexGroup.Post("/query_group_buy_market_config", g.handle(g.queryGroupBuyMarketConfig))

	// Market trade (MarketTradeController: /api/v1/gbm/trade).
	tradeGroup := app.Router("/api/v1/gbm/trade")
	tradeGroup.Post("/lock_market_pay_order", g.handle(g.lockMarketPayOrder))
	tradeGroup.Post("/settlement_market_pay_order", g.handle(g.settlementMarketPayOrder))
	tradeGroup.Post("/refund_market_pay_order", g.handle(g.refundMarketPayOrder))

	// Test callback (TestApiClientController: /api/v1/test).
	testGroup := app.Router("/api/v1/test")
	testGroup.Post("/group_buy_notify", g.handle(g.groupBuyNotify))
}

// handle wraps a handler to record HTTP metrics.
func (g *Gateway) handle(h func(ctx *yukinohttp.Context)) yukinohttp.Middleware {
	return func(ctx *yukinohttp.Context, next func()) {
		h(ctx)
		metrics.HTTPRequestsTotal.WithLabelValues(ctx.Path, strconv.Itoa(ctx.Status)).Inc()
		if next != nil {
			next()
		}
	}
}

// ---- admin handlers ----

func (g *Gateway) createActivity(ctx *yukinohttp.Context) {
	var req api.ActivityManageRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(adminFail("create activity failed", err))
		return
	}
	reply, err := g.activity.CreateActivity(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(adminFail("create activity failed", err))
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) updateActivity(ctx *yukinohttp.Context) {
	var req api.ActivityManageRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(adminFail("update activity failed", err))
		return
	}
	reply, err := g.activity.UpdateActivity(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(adminFail("update activity failed", err))
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) deleteActivity(ctx *yukinohttp.Context) {
	activityId, ok := parseActivityId(ctx)
	if !ok {
		ctx.JSON(adminFail("delete activity failed", errInvalidActivityId))
		return
	}
	reply, err := g.activity.DeleteActivity(ctx.Request.Context(), &api.ActivityIdRequest{ActivityId: activityId})
	if err != nil {
		ctx.JSON(adminFail("delete activity failed", err))
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) getActivityDetail(ctx *yukinohttp.Context) {
	activityId, ok := parseActivityId(ctx)
	if !ok {
		ctx.JSON(adminFail("get activity detail failed", errInvalidActivityId))
		return
	}
	reply, err := g.activity.GetActivityDetail(ctx.Request.Context(), &api.ActivityIdRequest{ActivityId: activityId})
	if err != nil {
		ctx.JSON(adminFail("get activity detail failed", err))
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) getActivityList(ctx *yukinohttp.Context) {
	reply, err := g.activity.GetActivityList(ctx.Request.Context())
	if err != nil {
		ctx.JSON(&api.Response{Code: "ERROR", Info: "get activity list failed: " + err.Error()})
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) enableActivity(ctx *yukinohttp.Context) {
	activityId, ok := parseActivityId(ctx)
	if !ok {
		ctx.JSON(adminFail("enable activity failed", errInvalidActivityId))
		return
	}
	reply, err := g.activity.EnableActivity(ctx.Request.Context(), &api.ActivityIdRequest{ActivityId: activityId})
	if err != nil {
		ctx.JSON(adminFail("enable activity failed", err))
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) disableActivity(ctx *yukinohttp.Context) {
	activityId, ok := parseActivityId(ctx)
	if !ok {
		ctx.JSON(adminFail("disable activity failed", errInvalidActivityId))
		return
	}
	reply, err := g.activity.DisableActivity(ctx.Request.Context(), &api.ActivityIdRequest{ActivityId: activityId})
	if err != nil {
		ctx.JSON(adminFail("disable activity failed", err))
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

// ---- DCC handler ----

func (g *Gateway) updateConfig(ctx *yukinohttp.Context) {
	key := ctx.Query("key")
	value := ctx.Query("value")
	if key == "" {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info, Data: false})
		return
	}
	if err := g.dcc.Publish(ctx.Request.Context(), g.redis, key, value); err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info, Data: false})
		return
	}
	ctx.JSON(&api.Response{Code: types.ResponseSuccess.Code, Info: types.ResponseSuccess.Info, Data: true})
}

// ---- index handler ----

func (g *Gateway) queryGroupBuyMarketConfig(ctx *yukinohttp.Context) {
	var req api.GoodsMarketRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	// Rate limit: one request per second per userId; exceeding it returns
	// RATE_LIMITER (mirrors the legacy fallbackMethod).
	if !g.indexLimiter.Allow(req.UserId) {
		ctx.JSON(&api.Response{Code: types.ResponseRateLimiter.Code, Info: types.ResponseRateLimiter.Info})
		return
	}
	reply, err := g.activity.QueryGroupBuyMarketConfig(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

// ---- trade handlers ----

func (g *Gateway) lockMarketPayOrder(ctx *yukinohttp.Context) {
	var req api.LockMarketPayOrderRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	reply, err := g.trade.LockMarketPayOrder(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) settlementMarketPayOrder(ctx *yukinohttp.Context) {
	var req api.SettlementMarketPayOrderRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	reply, err := g.trade.SettlementMarketPayOrder(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

func (g *Gateway) refundMarketPayOrder(ctx *yukinohttp.Context) {
	var req api.RefundMarketPayOrderRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	reply, err := g.trade.RefundMarketPayOrder(ctx.Request.Context(), &req)
	if err != nil {
		ctx.JSON(&api.Response{Code: types.ResponseUnError.Code, Info: types.ResponseUnError.Info})
		return
	}
	ctx.JSON(toResponse(&reply.RPCResult, reply.Data))
}

// ---- test handler ----

// groupBuyNotify mirrors TestApiClientController.groupBuyNotify: a fake third-party
// callback receiver that always answers "success".
func (g *Gateway) groupBuyNotify(ctx *yukinohttp.Context) {
	var req api.NotifyRequestDTO
	if err := ctx.BindJSON(&req); err != nil {
		ctx.String("error")
		return
	}
	ctx.String("success")
}

// ---- middleware ----

// maxBodyBytes bounds inbound request bodies. yukino_http's BindJSON has no
// built-in size limit, while the legacy Spring stack rejected oversized posts
// (2MB default); 1MB is far above any legitimate DTO here and keeps a hostile
// or buggy client from forcing unbounded memory use. Defect fix (hardening).
const maxBodyBytes = 1 << 20

// bodyLimit wraps the request body in an http.MaxBytesReader so oversized
// bodies fail at bind time with an error response instead of being buffered.
func (g *Gateway) bodyLimit() yukinohttp.Middleware {
	return func(ctx *yukinohttp.Context, next func()) {
		if ctx.Request.Body != nil {
			ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBodyBytes)
		}
		next()
	}
}

// cors mirrors @CrossOrigin("*").
func (g *Gateway) cors() yukinohttp.Middleware {
	return func(ctx *yukinohttp.Context, next func()) {
		ctx.Set("Access-Control-Allow-Origin", "*")
		ctx.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		ctx.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if ctx.Method == http.MethodOptions {
			ctx.SetStatus(http.StatusNoContent)
			return
		}
		next()
	}
}

// traceId mirrors TraceIdFilter: attach a request-scoped trace id.
func (g *Gateway) traceId() yukinohttp.Middleware {
	return func(ctx *yukinohttp.Context, next func()) {
		traceId := ctx.Get("trace-id")
		if traceId == "" {
			traceId = newTraceId()
		}
		ctx.Set("trace-id", traceId)
		ctx.State["trace-id"] = traceId
		next()
	}
}

// ---- helpers ----

func toResponse(result *api.RPCResult, data any) *api.Response {
	return &api.Response{Code: result.Code, Info: result.Info, Data: data}
}

func adminFail(prefix string, err error) *api.Response {
	msg := err.Error()
	return &api.Response{
		Code: "ERROR",
		Info: prefix + ": " + msg,
		Data: &api.ActivityManageResponseDTO{Success: false, ErrorMessage: msg},
	}
}

func parseActivityId(ctx *yukinohttp.Context) (int64, bool) {
	id, err := strconv.ParseInt(ctx.Param("activityId"), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

var errInvalidActivityId = &simpleError{"invalid activity id"}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

// newTraceId produces a random trace id, mirroring TraceIdFilter's UUID.
func newTraceId() string {
	const hexChars = "0123456789abcdef"
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	out := make([]byte, 36)
	p := 0
	for i, b := range buf {
		out[p] = hexChars[b>>4]
		out[p+1] = hexChars[b&0x0f]
		p += 2
		if i == 3 || i == 5 || i == 7 || i == 9 {
			out[p] = '-'
			p++
		}
	}
	return string(out)
}

// RateLimiter is a per-key limiter allowing one request per interval, mirroring
// the legacy RateLimiterAccessInterceptor(permitsPerSecond=1.0).
//
// Defect fix: the legacy interceptor kept an ever-growing map. Entries older than
// the interval can no longer influence a decision, so they are swept periodically
// (and whenever the map grows past a cap) to bound memory.
type RateLimiter struct {
	mu          sync.Mutex
	last        map[string]time.Time
	interval    time.Duration
	lastCleanup time.Time
}

const (
	limiterCleanupInterval = time.Minute
	limiterMaxEntries      = 10000
)

// NewRateLimiter builds a limiter with the given per-request interval.
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{last: make(map[string]time.Time), interval: interval, lastCleanup: time.Now()}
}

// Allow reports whether the key may proceed and records the access time.
func (r *RateLimiter) Allow(key string) bool {
	if key == "" {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if now.Sub(r.lastCleanup) >= limiterCleanupInterval || len(r.last) >= limiterMaxEntries {
		for k, t := range r.last {
			if now.Sub(t) >= r.interval {
				delete(r.last, k)
			}
		}
		r.lastCleanup = now
	}
	if last, ok := r.last[key]; ok && now.Sub(last) < r.interval {
		return false
	}
	r.last[key] = now
	return true
}
