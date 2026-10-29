# be261028 — 拼团营销系统（Go 微服务）

由旧项目 **easy-group-buy**（Java / Spring Boot 单体，DDD 分层）对等迁移而来的
**Go 微服务**实现。技术栈迁移为 **yukino_http**（HTTP 网关）、**yukino_rpc**
（TCP RPC 微服务通信）、**Kafka**（替代 RabbitMQ），并复用本机 Docker 中的
MySQL / Redis / Prometheus / Grafana。

> 项目名：`easy-group-buy` → `be261028`；数据库名同步改为 `be261028`。

---

## 1. 架构

旧系统是单体应用，迁移后按 DDD 限界上下文拆为 3 个微服务，通过 `yukino_rpc`
通信，`yukino_http` 作为对外 HTTP 网关：

```
                         ┌──────────────────────────────┐
   HTTP (yukino_http)    │           gateway             │  :8091
   /api/v1/...  ───────▶ │  REST→RPC 编排 / 限流 / CORS   │  /metrics /health
                         └──────────┬───────────────────┘
                    yukino_rpc      │        yukino_rpc
                 ┌──────────────────┴───────────────────┐
                 ▼                                        ▼
      ┌───────────────────────┐              ┌────────────────────────┐
      │        activity        │◀─────────────│          trade          │
      │ 活动管理/试算树/折扣/标签 │  锁单试算回调  │ 锁单/结算/退款/通知任务      │
      │        :19001          │              │        :19002           │
      └───────┬───────────────┘              └───────┬────────────────┘
              │                                       │
        MySQL + Redis                          MySQL + Redis + Kafka
```

- **gateway**：`yukino_http` HTTP 服务，复刻旧 `trigger` 层全部 REST 端点，
  作为 RPC 客户端编排 activity/trade；含限流、TraceId、CORS、Prometheus 指标。
- **activity**：`yukino_rpc` 服务端，复刻 `domain/activity` + `domain/tag`：
  活动管理（CRUD）、营销试算策略树、折扣计算（ZJ/MJ/N/ZK）、人群标签。
- **trade**：`yukino_rpc` 服务端，复刻 `domain/trade`：锁单/结算/退款责任链、
  退款策略、通知任务；作为 RPC 客户端回调 activity 做锁单试算；Kafka 生产者/
  消费者（成团/退款主题）与定时回调任务。

### 技术栈映射

| 旧 (Java/Spring Boot)          | 新 (Go)                                   |
| ------------------------------ | ----------------------------------------- |
| Spring MVC Controller          | yukino_http（gateway）                    |
| 内部进程调用（单体）            | yukino_rpc（gateway↔activity↔trade）      |
| RabbitMQ (topic exchange)      | Kafka（team_success / team_refund 主题）   |
| MyBatis + MySQL                | GORM（gorm.io/gorm + driver/mysql）        |
| Redisson                       | go-redis/v9（含位图、分布式锁、计数器）     |
| BigDecimal                     | shopspring/decimal + `types.Money`        |
| SnakeYAML（Spring 配置）        | goccy/go-yaml                             |
| 原生 HTML/JS/CSS 静态页          | Vite + Lit 3 + @yukino.js/lit-jsx + Tailwind v4 + motion（`frontend/`） |
| DCC 动态配置（Redis pub/sub）   | `infra/dcc`（Redis pub/sub）              |
| Actuator + Micrometer          | prometheus/client_golang（/metrics）      |
| CompletableFuture 并行试算      | goroutine + sync.WaitGroup                |

---

## 2. 目录结构

```
be261028/
├── cmd/{gateway,activity,trade}/main.go   # 三个服务入口
├── internal/
│   ├── api/          # REST/RPC 契约 DTO + 跨服务值对象（含 Money）
│   ├── types/        # 枚举/响应码/异常/Java 兼容函数(hashCode/split)/JSONTime/Money
│   ├── config/       # YAML 配置（goccy/go-yaml）
│   ├── bootstrap/    # 启动辅助（信号、RPC 生命周期）
│   ├── infra/
│   │   ├── db/       # GORM 连接（parseTime+loc+clientFoundRows+TranslateError）
│   │   ├── redisx/   # Redis 封装（JSON缓存/计数器/SETNX/位图/分布式锁）
│   │   ├── kafkax/   # Kafka 生产者/消费者 + EnsureTopics
│   │   ├── dcc/      # 动态配置中心（Redis pub/sub）
│   │   └── metrics/  # Prometheus /metrics + /health
│   ├── dao/          # GORM PO（TableName 绑定）+ 查询方法（语义对齐 MyBatis mapper）
│   ├── activity/     # activity 域：仓储/试算树/折扣/管理/标签
│   ├── trade/        # trade 域：仓储/锁单/结算/退款/通知/策略
│   ├── rpc/          # yukino_rpc 服务端实现 + 客户端封装
│   └── gateway/      # yukino_http 处理器 + 中间件
├── frontend/         # 独立前端工程（Vite+Lit+lit-jsx+Tailwind+motion）
│   ├── src/{lib,components,pages}     # api/auth/router/toast + 登录/首页/管理后台
│   ├── public/images/                 # 商品轮播图与支付二维码
│   └── dist/                          # pnpm build 产物，由网关 webDir 静态服务
├── configs/{gateway,activity,trade}.yaml
├── deployments/sql/be261028.sql           # 建库建表 + 种子数据
└── deployments/prometheus/scrape-example.yml
```

---

## 3. 运行

前置：本机 Docker 已启动 `mysql / redis / kafka / prometheus / grafana`
（见 `~/github/hangtiancheng/docker-compose.yml`）。

```bash
# 1) 初始化数据库（创建 be261028 库 + 表 + 种子数据）
make init-db

# 2) 依次启动三个服务（推荐 go run 方式，见下方说明）
make run-activity    # :19001
make run-trade       # :19002
make run-gateway     # :8091

# 或构建后运行（已含 codesign 重签名）
make build
./bin/activity -config configs/activity.yaml
./bin/trade    -config configs/trade.yaml
./bin/gateway  -config configs/gateway.yaml

# 停止
make stop
```

> **注意（本机环境）**：本机装有企业安全代理(EDR)与 macOS AMFI，会对项目
> `bin/` 下的 ad-hoc 签名二进制做签名校验并**间歇性 SIGKILL**。`go run`
> （临时目录二进制）不受影响，故推荐 `make run-*`。`make build` 已附带
> `codesign --force -s - --options runtime` 重签名以尽量规避。

端口一览：gateway `8091`（含 /metrics、/health）、activity RPC `19001` /
metrics `19101`、trade RPC `19002` / metrics `19102`。

### 前端（独立工程 `frontend/`）

旧版 `web/`（原生 HTML/JS/CSS 副本）已删除，前端重建为独立 Vite 工程：
Vite 7 + TypeScript + Lit 3 + `@yukino.js/lit-jsx`（JSX 写组件）+ TailwindCSS v4
（零手写 CSS）+ motion 动画。

```bash
cd frontend
pnpm install
pnpm dev      # http://127.0.0.1:5173 ，/api 代理到网关 8091
pnpm build    # 产出 frontend/dist（tsc --noEmit + vite build）
```

生产模式下由网关直接服务 `frontend/dist`（`configs/gateway.yaml` 的
`service.webDir: frontend/dist`），前端与 API 同源，刷新/直达路由不需额外配置。

页面：`#/login` 登录（cookie `username`，纯前端，与旧版一致）、`#/` 商品详情 +
拼团队伍（实时倒计时、开团/参团 → 锁单 → 扫码支付 → 结算、退单、待支付恢复）、
`#/admin` 活动 CRUD/启停 + DCC 动态配置面板。首页支持 `?goodsId=` 指定商品
（默认 `9890001`；`9890002` 为种子数据里的双人团演示活动，可看到队伍列表）。

### 验证

```bash
make test            # 单元测试（折扣/试算/责任链/hashCode/Money/JSONTime 等）
make vet
curl http://127.0.0.1:8091/health
```

---

## 4. REST API（与旧项目一致）

| 方法 | 路径 | 说明 |
| ---- | ---- | ---- |
| POST | `/api/v1/admin/activity/create` | 创建活动 |
| POST | `/api/v1/admin/activity/update` | 更新活动 |
| POST | `/api/v1/admin/activity/delete/:activityId` | 删除活动 |
| GET  | `/api/v1/admin/activity/detail/:activityId` | 活动详情 |
| GET  | `/api/v1/admin/activity/list` | 活动列表 |
| POST | `/api/v1/admin/activity/enable/:activityId` | 启用活动 |
| POST | `/api/v1/admin/activity/disable/:activityId` | 禁用活动 |
| GET  | `/api/v1/gbm/dcc/update_config?key=&value=` | 动态配置 |
| POST | `/api/v1/gbm/index/query_group_buy_market_config` | 首页营销配置（限流） |
| POST | `/api/v1/gbm/trade/lock_market_pay_order` | 营销锁单 |
| POST | `/api/v1/gbm/trade/settlement_market_pay_order` | 营销结算 |
| POST | `/api/v1/gbm/trade/refund_market_pay_order` | 营销退款 |
| POST | `/api/v1/test/group_buy_notify` | 模拟第三方回调 |

> 旧代码 `AdminController` 实际映射为 `/api/v1/admin/`（README 里写的
> `/api/v1/gbm/admin/` 与代码不符），本实现以代码为准。

金额统一以两位小数输出（如 `100.00`），对齐旧 Jackson+BigDecimal 行为。

### Kafka 主题（替代 RabbitMQ）

| 主题 | 说明 |
| ---- | ---- |
| `be261028_team_success` | 成团通知（结算完成时发布） |
| `be261028_team_refund`  | 退款通知（退款时发布，消费者恢复锁单库存） |

---

## 5. 迁移中修复的旧项目缺陷

1. **退款库存恢复链路断裂（核心）**：旧项目发布了退款 MQ 消息，但**没有任何
   消费者**，`restoreTeamLockStock` 是死代码，退款后锁单库存永不恢复。
   新实现接入 `team_refund` Kafka 消费者并调用恢复逻辑（`refund2AddRecovery`）。
2. **notify_task 唯一约束错误**：旧表对 `team_id` 加唯一约束，但一个队伍会产生
   多条回调任务（成团 + 各次退款），导致后续任务插入冲突、事务回滚。
   新实现去除该唯一约束，改为按自增 `id` 精确更新状态。
3. **重复退款非幂等**：旧 `DataNodeFilter` 先于幂等检查抛 `E0107`，使
   `UniqueRefundNodeFilter` 的 `REPEAT` 分支不可达。新实现让重复退款返回
   幂等 `REPEAT`。
4. **Redis 缓存永不失效**：旧系统活动/折扣配置缓存无 TTL 且管理端变更不清缓存，
   导致禁用/更新活动不生效。新实现在 update/enable/disable/delete 后主动失效缓存。
5. **活动管理多表写入非事务**：旧 `saveActivity`/`updateActivity` 无事务，
   可能留下孤儿行。新实现包裹事务。
6. **SC 关联查询缺陷**：旧 `getActivityById` 用“按 goods_id 过滤”的查询却传
   activity_id，导致 skuInfo 永远查不到。新实现提供按 activity_id 的正确查询。
7. **MySQL 行数语义差异**：MySQL 驱动默认返回“实际改变行数”，与旧 JDBC
   默认“匹配行数”不同，导致同秒更新误报 0 行失败。新实现通过 GORM DSN 设置
   `clientFoundRows=true` 对齐 JDBC。
8. **金额精度**：引入 `types.Money`（shopspring/decimal），JSON 恒输出两位小数，
   对齐旧 BigDecimal。

此外保留并复刻了旧项目的降级(downgradeSwitch)、切量(cutRange)、渠道黑名单
(scBlacklist)、缓存开关(cacheSwitch) 等动态配置能力（DCC）。

---

## 6. 监控

- 三个服务均暴露 Prometheus `/metrics` 与 `/health`。
- 抓取配置示例见 `deployments/prometheus/scrape-example.yml`（追加到本机
  Prometheus 的 `scrape_configs`）。Grafana 已配置 Prometheus 数据源。
- 业务指标：`be261028_http_requests_total`、`be261028_rpc_calls_total`、
  `be261028_trade_ops_total`、`be261028_notify_tasks_total`、
  `be261028_kafka_messages_total`。
