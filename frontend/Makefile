# Makefile 常用目标：安装依赖、启动前端、构建前端、显示帮助
.PHONY: help init pnpm-admin pnpm-app run run-admin run-app build build-admin build-app

# 安装管理后台与商城端依赖
init: pnpm-admin pnpm-app

# 安装管理后台依赖
pnpm-admin:
	@cd admin && \
		CI=1 pnpm install

# 安装商城端依赖
pnpm-app:
	@cd app && \
		CI=1 pnpm install

# 一键启动管理后台与商城端开发环境，日志按多行输出
run: run-admin run-app

# 启动管理后台开发服务
run-admin:
	@cd admin && \
		CI=1 pnpm run dev

# 启动商城端微信小程序开发服务
run-app:
	@cd app && \
		CI=1 pnpm run dev:mp-weixin

# 一键构建管理后台与商城端产物，日志按多行输出
build: build-admin build-app

# 构建管理后台静态资源
build-admin:
	@cd admin && \
		CI=1 pnpm run build

# 构建商城端微信小程序产物
build-app:
	@cd app && \
		CI=1 pnpm run build:mp-weixin

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
