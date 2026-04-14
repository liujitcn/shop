# Makefile 常用目标：初始化工具、生成协议、格式化代码、执行测试
.PHONY: help init plugin cli fmt proto gen test

# 初始化开发环境
init: plugin cli

# 安装 protoc 插件
plugin:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 安装命令行工具
cli:
	@go install github.com/bufbuild/buf/cmd/buf@latest

# 使用 gofmt 统一格式化 Go 代码
fmt:
	@files="$$(find . -name '*.go' -not -path './api/gen/*')"; \
	if [ -n "$$files" ]; then gofmt -w $$files; fi

# 生成 protobuf 的 Go 消息代码
proto:
	@cd api && \
	buf generate

# 一键生成并格式化
gen: proto fmt

# 执行模块测试
test:
	@go test ./...

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
			printf "\033[36m%-18s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
