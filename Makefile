# be261028 group-buy marketing system - build and run
# Migrated from easy-group-buy (Java/Spring Boot) to Go + yukino_http + yukino_rpc + Kafka
#
# How to run: this machine runs an enterprise security agent (EDR) + macOS AMFI that
# performs code-signing verification on the ad-hoc signed binaries under bin/ and
# intermittently SIGKILLs them. `go run` (binaries in a temp directory) is not
# affected, so prefer the run-* targets to start services. If the bin/ binaries are
# needed, the build target re-signs them with codesign as a best-effort workaround.

GO ?= go
BIN_DIR := bin

.PHONY: all build sign run-gateway run-activity run-trade stop \
        tidy vet test fmt clean init-db client-install client-build client-dev

all: build

build: sign

# Build and re-sign ad-hoc (--options runtime helps pass AMFI verification)
sign:
	$(GO) build -o $(BIN_DIR)/activity ./cmd/activity
	$(GO) build -o $(BIN_DIR)/trade ./cmd/trade
	$(GO) build -o $(BIN_DIR)/gateway ./cmd/gateway
	@for b in activity trade gateway; do \
		codesign --force -s - --options runtime $(BIN_DIR)/$$b 2>/dev/null || true; \
	done
	@echo "built + signed: $(BIN_DIR)/{activity,trade,gateway}"

# Start in dependency order: activity -> trade -> gateway (recommended; avoids EDR kills of bin/ binaries)
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

# Initialize the database (requires the local docker mysql running, root/pass)
init-db:
	docker exec -i hangtiancheng-mysql-1 mysql -uroot -ppass < deployments/sql/be261028.sql

# client (standalone project): pnpm build produces dist, served statically by the gateway webDir
client-install:
	pnpm install

client-build:
	pnpm install --frozen-lockfile && pnpm build

client-dev:
	pnpm dev
