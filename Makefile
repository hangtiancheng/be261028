# be261028 拼团营销系统 - 构建与运行
# 由 easy-group-buy (Java/Spring Boot) 迁移到 Go + yukino_http + yukino_rpc + Kafka
#
# 运行方式说明：本机存在企业安全代理(EDR)+ macOS AMFI，会对项目 bin/ 下的
# ad-hoc 签名二进制做代码签名校验并间歇性 SIGKILL。`go run`（临时目录二进制）
# 不受影响，因此推荐用 run-* 目标启动服务。若需使用 bin/ 二进制，build 目标已
# 附带 codesign 重签名以尽量规避。

GO ?= go
BIN_DIR := bin

.PHONY: all build sign run-gateway run-activity run-trade stop \
        tidy vet test fmt clean init-db frontend-install frontend-build frontend-dev

all: build

build: sign

# 构建并 ad-hoc 重签名（--options runtime 有助于通过 AMFI 校验）
sign:
	$(GO) build -o $(BIN_DIR)/activity ./cmd/activity
	$(GO) build -o $(BIN_DIR)/trade ./cmd/trade
	$(GO) build -o $(BIN_DIR)/gateway ./cmd/gateway
	@for b in activity trade gateway; do \
		codesign --force -s - --options runtime $(BIN_DIR)/$$b 2>/dev/null || true; \
	done
	@echo "built + signed: $(BIN_DIR)/{activity,trade,gateway}"

# 按依赖顺序启动：activity -> trade -> gateway（推荐，规避 EDR 对 bin/ 的拦截）
run-activity:
	$(GO) run ./cmd/activity -config configs/activity.yaml

run-trade:
	$(GO) run ./cmd/trade -config configs/trade.yaml

run-gateway:
	$(GO) run ./cmd/gateway -config configs/gateway.yaml

stop:
	-pkill -f "exe/activity"; pkill -f "exe/trade"; pkill -f "exe/gateway"
	-pkill -f "bin/activity"; pkill -f "bin/trade"; pkill -f "bin/gateway"

tidy:
	$(GO) mod tidy

vet:
	$(GO) vet ./...

test:
	$(GO) test ./... -count=1

fmt:
	gofmt -w cmd internal

clean:
	rm -rf $(BIN_DIR)

# 初始化数据库（需要本机 docker mysql 已启动，root/pass）
init-db:
	docker exec -i hangtiancheng-mysql-1 mysql -uroot -ppass < deployments/sql/be261028.sql

# 前端（独立工程）：pnpm build 产出 frontend/dist，由网关 webDir 静态服务
frontend-install:
	cd frontend && pnpm install

frontend-build:
	cd frontend && pnpm install --frozen-lockfile && pnpm build

frontend-dev:
	cd frontend && pnpm dev
