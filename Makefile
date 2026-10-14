GO ?= go
SERVICES := user chat connect gateway

.PHONY: init-local build test vet generate up down logs e2e
init-local:
	$(GO) run ./tools/init-local
build:
	mkdir -p bin
	$(GO) build -trimpath -o bin/user ./app/user/service/cmd/server
	$(GO) build -trimpath -o bin/chat ./app/chat/service/cmd/server
	$(GO) build -trimpath -o bin/connect ./app/connect/service/cmd/server
	$(GO) build -trimpath -o bin/gateway ./app/gateway/cmd/server
test:
	$(GO) test ./...
vet:
	$(GO) vet ./...
generate:
	$(GO) run ./tools/generate
up:
	docker compose up -d --build
down:
	docker compose down
logs:
	docker compose logs --tail=100
e2e:
	IMCORE_E2E=1 $(GO) test ./test/e2e -count=1 -v

# 插件版本固定在 Kratos v2，避免生成 v3 代码。
.PHONY: tools
tools:
	mkdir -p .tools
	GOBIN=$(CURDIR)/.tools $(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	GOBIN=$(CURDIR)/.tools $(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
	GOBIN=$(CURDIR)/.tools $(GO) install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.0.0-20260404020628-f149714c1d54
	GOBIN=$(CURDIR)/.tools $(GO) install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0
	GOBIN=$(CURDIR)/.tools $(GO) install github.com/google/wire/cmd/wire@v0.7.0

.PHONY: audit
audit:
	$(GO) run ./tools/audit
