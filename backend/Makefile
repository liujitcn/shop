# Makefile 常用目标：初始化工具、生成代码、格式化代码、启动服务
.PHONY: help init plugin cli fmt gorm-gen wire api openapi ts run build docker-build

IMAGE ?= backend
TAG ?= latest
DOCKERFILE ?= Dockerfile
BINARY ?= bin/server

# 初始化开发环境
init: plugin cli

# 安装 protoc 插件
plugin:
	# Go 相关插件
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
	@go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	@go install github.com/envoyproxy/protoc-gen-validate@latest
	@go install github.com/liujitcn/kratos-kit/cmd/protoc-gen-go-service@latest
	@go install github.com/go-kratos/protoc-gen-typescript-http@latest

# 安装命令行工具
cli:
	@go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	@go install github.com/google/gnostic@latest
	@go install github.com/bufbuild/buf/cmd/buf@latest
	@go install entgo.io/ent/cmd/ent@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest

# 使用 goimports 统一整理 Go 代码的 import 与格式
fmt:
	@goimports -w $$(rg --files -g '*.go')

# 生成 GORM 代码
gorm-gen:
	gen -base_path=pkg/gen -source='root:112233@tcp(127.0.0.1:3306)/shop?charset=utf8&parseTime=True&loc=Local&timeout=1000ms'

# 生成 wire 依赖注入代码
wire: fmt
	@cd internal/cmd/server && \
	wire

# 生成 protobuf 的 Go 代码
api:
	@cd api && \
	buf generate

# 生成 OpenAPI v3 文档
openapi:
	@cd api && \
	buf generate --template buf.openapi.gen.yaml --exclude-path protos/conf

# 生成 TypeScript 代码
ts:
	@cd api && \
	buf generate --template buf.admin.typescript.gen.yaml --exclude-path protos/conf --exclude-path protos/app && \
	buf generate --template buf.app.typescript.gen.yaml --exclude-path protos/conf --exclude-path protos/admin

# 一键生成全部接口产物（Go 代码 + OpenAPI 文档 + TypeScript 代码）
gen: api openapi ts

# 启动应用
run: api openapi
	@go run ./cmd/server -conf ./configs

# 构建 Linux 可执行文件，供 Docker 镜像直接打包
build:
	@mkdir -p $(dir $(BINARY))
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BINARY) ./internal/cmd/server

# 构建后端服务 Docker 镜像
docker-build: build
	@docker build -f $(DOCKERFILE) -t $(IMAGE):$(TAG) .

# 显示帮助
help:
	@echo ""
	@echo "Usage:"
	@echo " make [target]"
	@echo ""
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
