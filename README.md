# be261028 — Group-Buy Marketing System (Go microservices)

A parity migration of the legacy **easy-group-buy** project (Java / Spring Boot
monolith, DDD layering) into **Go microservices**. The tech stack is migrated to
**yukino_http** (HTTP gateway), **yukino_rpc** (TCP RPC for inter-service calls),
and **Kafka** (replacing RabbitMQ), reusing the local Docker MySQL / Redis /
Prometheus / Grafana instances.

> Project rename: `easy-group-buy` → `be261028`; the database was renamed to
> `be261028` accordingly.

---

## 1. Architecture

The legacy system is a monolith. It is split into 3 microservices along the DDD
bounded contexts, communicating over `yukino_rpc`, with `yukino_http` serving as
the external HTTP gateway:

```
                         ┌──────────────────────────────┐
   HTTP (yukino_http)    │           gateway             │  :8091
   /api/v1/...  ───────▶ │  REST→RPC orchestration /     │  /metrics /health
                         │  rate limit / CORS            │
                         └──────────┬───────────────────┘
                    yukino_rpc      │        yukino_rpc
                 ┌──────────────────┴───────────────────┐
                 ▼                                        ▼
      ┌───────────────────────┐              ┌────────────────────────┐
      │        activity        │◀─────────────│          trade          │
      │ admin / trial tree /   │ lock-order   │ lock / settlement /     │
      │ discounts / crowd tags │ trial calls  │ refund / notify tasks   │
      │        :19001          │              │        :19002           │
      └───────┬───────────────┘              └───────┬────────────────┘
              │                                       │
        MySQL + Redis                          MySQL + Redis + Kafka
```

- **gateway**: `yukino_http` service. Replicates every REST endpoint of the
  legacy `trigger` layer and orchestrates activity/trade as an RPC client.
  Includes rate limiting, TraceId propagation, CORS, and Prometheus metrics.
- **activity**: `yukino_rpc` server. Replicates `domain/activity` +
  `domain/tag`: activity administration (CRUD), the marketing trial strategy
  tree, discount calculators (ZJ/MJ/N/ZK), and crowd tags.
- **trade**: `yukino_rpc` server. Replicates `domain/trade`: the
  lock/settlement/refund rule chains, refund strategies, and notify tasks.
  Calls back into activity via RPC for lock-time trial pricing. Kafka
  producer/consumer (team-success / team-refund topics) plus the scheduled
  callback notify job.

### Technology mapping

| Legacy (Java/Spring Boot)          | New (Go)                                                             |
| ---------------------------------- | -------------------------------------------------------------------- |
| Spring MVC controllers             | yukino_http (gateway)                                                |
| In-process calls (monolith)        | yukino_rpc (gateway ↔ activity ↔ trade)                              |
| RabbitMQ (topic exchange)          | Kafka (`team_success` / `team_refund` topics)                        |
| MyBatis + MySQL                    | GORM (`gorm.io/gorm` + `gorm.io/driver/mysql`)                       |
| Redisson                           | go-redis/v9 (bitmaps, distributed locks, counters)                   |
| BigDecimal                         | shopspring/decimal + `types.Money`                                   |
| SnakeYAML (Spring configuration)   | goccy/go-yaml                                                        |
| Static HTML/JS/CSS pages           | Vite + Lit 3 + @yukino.js/lit-jsx + Tailwind v4 + motion (repo root) |
| DCC dynamic config (Redis pub/sub) | `infra/dcc` (Redis pub/sub)                                          |
| Actuator + Micrometer              | prometheus/client_golang (`/metrics`)                                |
| CompletableFuture parallel trials  | goroutines + sync.WaitGroup                                          |

---

## 2. Directory layout

```
be261028/
├── cmd/{gateway,activity,trade}/main.go   # service entry points
├── internal/
│   ├── api/          # REST/RPC contract DTOs + cross-service value objects (incl. Money)
│   ├── types/        # enums / response codes / errors / Java-compat helpers (hashCode/split) / JSONTime / Money
│   ├── config/       # YAML configuration (goccy/go-yaml)
│   ├── bootstrap/    # startup helpers (signals, RPC lifecycle)
│   ├── infra/
│   │   ├── db/       # GORM connection (parseTime+loc+clientFoundRows+TranslateError)
│   │   ├── redisx/   # Redis wrapper (JSON cache/counters/SETNX/bitmaps/distributed locks)
│   │   ├── kafkax/   # Kafka producer/consumer + EnsureTopics
│   │   ├── dcc/      # dynamic config center (Redis pub/sub)
│   │   └── metrics/  # Prometheus /metrics + /health
│   ├── dao/          # GORM POs (TableName bindings) + queries (semantics aligned with MyBatis mappers)
│   ├── activity/     # activity domain: repository / trial tree / discounts / admin / crowd tags
│   ├── trade/        # trade domain: repository / lock / settlement / refund / notify / strategies
│   ├── rpc/          # yukino_rpc server implementations + client wrappers
│   └── gateway/      # yukino_http handlers + middleware
├── src/{lib,components,pages}         # frontend (Vite + Lit + lit-jsx + Tailwind + motion): api/auth/router/toast/art + login/home/admin pages
├── index.html                         # frontend entry page (Vite)
├── dist/                              # pnpm build output, served statically by the gateway webDir
├── configs/{gateway,activity,trade}.yaml
├── deployments/sql/be261028.sql           # schema + seed data
└── deployments/prometheus/scrape-example.yml
```

---

## 3. Running

Prerequisites: the local Docker stack (`mysql / redis / kafka / prometheus /
grafana`) is up — see `~/github/hangtiancheng/docker-compose.yml`.

```bash
# 1) Initialize the database (creates the be261028 schema + seed data)
make init-db

# 2) Start the three services in dependency order (go run recommended, see note below)
make run-activity    # :19001
make run-trade       # :19002
make run-gateway     # :8091

# Or build first (includes codesign re-signing)
make build
./bin/activity -config configs/activity.yaml
./bin/trade    -config configs/trade.yaml
./bin/gateway  -config configs/gateway.yaml

# Stop all services
make stop
```

> **Environment note**: this machine runs an enterprise security agent (EDR)
> plus macOS AMFI, which verify the ad-hoc signatures of binaries under `bin/`
> and **intermittently SIGKILL** them. `go run` (temp-directory binaries) is
> unaffected, so `make run-*` is recommended. `make build` applies
> `codesign --force -s - --options runtime` as a best-effort workaround.

Ports: gateway `8091` (also /metrics, /health); activity RPC `19001` / metrics
`19101`; trade RPC `19002` / metrics `19102`.

### Frontend (repo root)

The frontend is rebuilt as a Vite project at the repository root: Vite + TypeScript +
Lit 3 + `@yukino.js/lit-jsx` (JSX-style components) + TailwindCSS v4 (no
hand-written CSS) + motion animations. It uses no image resources at all — all
visuals (goods art, avatars) are generated programmatically in `src/lib/art.ts`.

```bash
pnpm install
pnpm dev      # http://127.0.0.1:5173, /api proxied to the gateway on 8091
pnpm build    # outputs dist (tsc --noEmit + vite build)
```

In production mode the gateway serves `dist` directly
(`service.webDir: dist` in `configs/gateway.yaml`), so the frontend
and API are same-origin and no extra routing configuration is needed.

Pages: `#/login` login (cookie `username`, frontend-only, parity with the
legacy page), `#/` goods detail + group-buy teams (live countdowns,
start/join team → lock order → pay → settlement, refunds, and unpaid-order
resume), `#/admin` activity CRUD / enable / disable + DCC dynamic-config panel.
The home page accepts `?goodsId=` to pick the goods (default `9890001`;
`9890002` is the seeded two-person team demo activity with a visible team
list).

### Verification

```bash
make test            # unit tests (discounts / trial tree / rule chains / hashCode / Money / JSONTime, ...)
make vet
curl http://127.0.0.1:8091/health
```

---

## 4. REST API (parity with the legacy project)

| Method | Path                                              | Description                               |
| ------ | ------------------------------------------------- | ----------------------------------------- |
| POST   | `/api/v1/admin/activity/create`                   | Create activity                           |
| POST   | `/api/v1/admin/activity/update`                   | Update activity                           |
| POST   | `/api/v1/admin/activity/delete/:activityId`       | Delete activity                           |
| GET    | `/api/v1/admin/activity/detail/:activityId`       | Activity detail                           |
| GET    | `/api/v1/admin/activity/list`                     | Activity list                             |
| POST   | `/api/v1/admin/activity/enable/:activityId`       | Enable activity                           |
| POST   | `/api/v1/admin/activity/disable/:activityId`      | Disable activity                          |
| GET    | `/api/v1/gbm/dcc/update_config?key=&value=`       | Update dynamic config                     |
| POST   | `/api/v1/gbm/index/query_group_buy_market_config` | Home-page marketing config (rate limited) |
| POST   | `/api/v1/gbm/trade/lock_market_pay_order`         | Lock a marketing pay order                |
| POST   | `/api/v1/gbm/trade/settlement_market_pay_order`   | Settle a marketing pay order              |
| POST   | `/api/v1/gbm/trade/refund_market_pay_order`       | Refund a marketing pay order              |
| POST   | `/api/v1/test/group_buy_notify`                   | Simulated third-party callback receiver   |

> The legacy `AdminController` actually maps to `/api/v1/admin/` (the legacy
> README claimed `/api/v1/gbm/admin/`, which never matched the code); this
> implementation follows the code.

Money is always serialized with two decimal places (e.g. `100.00`), matching
the legacy Jackson + BigDecimal behavior.

### Kafka topics (replacing RabbitMQ)

| Topic                   | Description                                                                 |
| ----------------------- | --------------------------------------------------------------------------- |
| `be261028_team_success` | Team-formed notification (published on settlement completion)               |
| `be261028_team_refund`  | Refund notification (published on refund; the consumer restores lock stock) |

---

## 5. Legacy defects fixed during the migration

1. **Broken refund stock-recovery chain (critical)**: the legacy project
   published refund MQ messages but had **no consumer** — `restoreTeamLockStock`
   was dead code, so lock stock was never restored after refunds. The new
   implementation wires a `team_refund` Kafka consumer to the recovery logic
   (`refund2AddRecovery`).
2. **notify_task unique constraint error**: the legacy table had a unique
   constraint on `team_id`, but one team produces several callback tasks
   (settlement + one per refund), causing insert conflicts and transaction
   rollbacks. The constraint is removed; tasks are updated by auto-increment
   `id` instead.
3. **Non-idempotent repeat refunds**: the legacy `DataNodeFilter` raised
   `E0107` before the idempotency check, making the `REPEAT` branch of
   `UniqueRefundNodeFilter` unreachable. Repeat refunds now return the
   idempotent `REPEAT` result.
4. **Redis cache never invalidated**: legacy activity/discount cache entries had
   no TTL and admin changes never cleared them, so disabling/updating an
   activity had no effect. The new implementation invalidates eagerly on
   update/enable/disable/delete and also applies a 30-minute TTL as defence in
   depth.
5. **Non-transactional multi-table writes**: legacy `saveActivity` /
   `updateActivity` wrote several tables without a transaction and could leave
   orphan rows. Writes are now wrapped in transactions.
6. **SC mapping query defect**: legacy `getActivityById` used a query that
   filters by `goods_id` while passing an `activity_id`, so `skuInfo` could
   never be loaded. The correct activity-id-based query is provided.
7. **MySQL row-count semantics**: Go MySQL drivers default to "rows changed"
   while legacy JDBC reports "rows matched", spurious zero-row failures on
   same-second updates. The GORM DSN sets `clientFoundRows=true` to align with
   JDBC.
8. **Money precision**: `types.Money` (shopspring/decimal) serializes with a
   fixed two-decimal format, matching legacy BigDecimal.
9. **Team stock occupancy leak**: a failed lock order left the Redis team-stock
   counter occupied forever (no TTL, no recovery). Occupied stock is now
   restored on failure and bounded by a TTL aligned with the team validity
   window.
10. **Duplicate settlement callbacks double-counted**: a repeated settlement
    callback re-ran the status update and double-incremented the team's
    `complete_count`, completing teams with fewer real payments than the
    target. Repeat settlements now return success idempotently.
11. **SC lookup ignored source/channel**: the legacy goods→activity lookup
    matched `goods_id` alone and could resolve a foreign source/channel. The
    lookup now includes source + channel.
12. **Random team sampling could return empty**: the legacy sample was
    truncated before teams were de-duplicated, so two orders of the same team
    could consume both slots and hide joinable teams. Briefs are collapsed to
    one row per team before sampling.
13. **cutRange off-by-one**: the legacy `<=` comparison let `cutRange=N` admit
    N+1 percent of users and made `cutRange=0` unable to cut all traffic. The
    comparison is now `<`, an exact percentage.
14. **tag_scope misclassification**: the legacy check only looked at fixed
    string positions; a lone `"2"` or reversed `"2,1"` was misclassified.
    Membership is now checked with substring containment.
15. **Unbounded trial fan-out**: the home-page trial waited on parallel
    discount calculations with no timeout. A bounded context (3s) now caps the
    fan-out.
16. **Validation message/field mismatch**: legacy SKU validation reported the
    opposite field's label (blank source → "channel must not be empty").
    Messages now match the field being validated.
17. **notify_task payload overflow**: `parameter_json` widened from
    varchar(256) to varchar(1024) (the team message grows with member count and
    large teams rolled back the whole settlement transaction); `notify_url`
    aligned to varchar(512) with `group_buy_order`.

The legacy dynamic-config capabilities are all preserved and replicated:
downgrade switch (`downgradeSwitch`), traffic cut (`cutRange`), channel
blacklist (`scBlacklist`), and cache switch (`cacheSwitch`) — see `infra/dcc`.

---

## 6. Monitoring

- All three services expose Prometheus `/metrics` and `/health`.
- See `deployments/prometheus/scrape-example.yml` for the scrape configuration
  (append it to the local Prometheus `scrape_configs`). Grafana already has the
  Prometheus data source configured.
- Business metrics: `be261028_http_requests_total`,
  `be261028_rpc_calls_total`, `be261028_trade_ops_total`,
  `be261028_notify_tasks_total`, `be261028_kafka_messages_total`.
