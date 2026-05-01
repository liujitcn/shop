# backend

`backend` 是商城项目的 Go 后端服务，基于 `Kratos` 组织 HTTP / gRPC / SSE / MCP 接口、数据库访问、文件上传、静态资源托管、定时任务、推荐同步和 OpenAPI 文档。

## 目录职责

```text
backend
├── api
│   ├── protos          # admin / app / base / common / conf proto 契约
│   └── gen/go          # proto 生成的 Go 代码
├── configs             # 服务运行配置
├── data                # 本地 OSS、日志、前端构建产物
├── internal/cmd/server # 实际启动入口、Wire 入口、内嵌 OpenAPI
├── pkg                 # 公共能力、生成模型、队列、任务、推荐、微信支付
├── server              # HTTP / gRPC Server 装配
└── service             # admin / app / base 业务服务
```

## 环境要求

- `Go 1.26+`
- `MySQL 8.x`
- `make`
- `buf`、`wire`、`goimports` 等生成工具可通过 `make init` 安装
- 前端依赖已安装时，`make ts` 可以生成前端 RPC TypeScript 代码

## 配置文件

| 文件 | 作用 |
| --- | --- |
| `configs/server.yaml` | HTTP / gRPC / SSE 端口、超时、CORS、Swagger、pprof、HTTP 中间件。 |
| `configs/data.yaml` | MySQL 连接、自动迁移、连接池、Redis 预留配置。 |
| `configs/oss.yaml` | 本地文件存储根目录，默认 `./data`。 |
| `configs/auth.yaml` | 登录认证、JWT、权限等基础配置。 |
| `configs/configs.yaml` | 通用业务配置。 |
| `configs/configs_local.yaml` | 本地微信、支付、推荐、大模型等业务配置。 |
| `configs/logger.yaml`、`trace.yaml`、`pprof.yaml`、`registry.yaml` | 日志、链路追踪、性能分析、注册中心相关配置。 |

默认数据库连接在 `configs/data.yaml`：

```yaml
source: root:112233@tcp(127.0.0.1:3306)/shop_test?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms
enable_migrate: true
```

首次启动会按当前模型自动建表。生产或共享环境应按实际情况调整账号、密码、库名和 `enable_migrate`。

## 本地启动

先创建数据库：

```sql
CREATE DATABASE shop_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

启动服务：

```bash
cd backend
go run ./internal/cmd/server -conf ./configs
```

默认地址：

- HTTP：`http://localhost:7001`
- gRPC：`localhost:6001`
- SSE：`http://localhost:7001/events`
- Swagger UI：`http://localhost:7001/docs/`
- OpenAPI：`http://localhost:7001/docs/openapi.yaml`

当前 `backend/Makefile` 的 `make run` 仍指向旧入口 `./cmd/server`，本地调试请优先使用上面的 `go run ./internal/cmd/server -conf ./configs`。

## 初始化数据

后端完成自动建表后，在仓库根目录导入基础数据：

```bash
mysql -uroot -p shop_test < sql/default-data.sql
mysql -uroot -p shop_test < sql/base_area.sql
```

如需演示商品、分类、轮播和商城服务数据，再导入：

```bash
mysql -uroot -p shop_test < sql/shop.sql
```

默认后台账号：

- `super / 112233`
- `admin / 112233`

说明：`sql/casbin_rule.sql` 当前为空文件，权限、菜单、接口、角色和用户初始化主要维护在 `sql/default-data.sql`。

## 接口与生成

常用命令：

```bash
cd backend
make init
make fmt
make api
make openapi
make ts
make gorm-gen
make wire
make gen
```

命令说明：

- `make init`：安装 Go / proto / buf / wire / goimports 等开发工具。
- `make fmt`：使用 `goimports` 格式化 Go 文件。
- `make api`：根据 `api/protos` 生成 Go 接口代码到 `api/gen/go`。
- `make openapi`：生成 OpenAPI 文档到 `internal/cmd/server/assets/openapi.yaml`。
- `make ts`：生成管理后台和商城端 RPC TypeScript 代码，需先在前端模块执行 `pnpm install`。
- `make gorm-gen`：按当前数据库生成 `pkg/gen` 下的数据模型、查询对象和仓储代码。
- `make wire`：在 `internal/cmd/server` 下生成依赖注入代码。
- `make gen`：生成 Go / OpenAPI / TypeScript 产物并格式化 Go 代码。

生成产物不要手工修改，优先通过对应命令更新：

- `api/gen/go`
- `pkg/gen`
- `internal/cmd/server/assets/openapi.yaml`
- `../frontend/admin/src/rpc`
- `../frontend/app/src/rpc`

## MCP 工具暴露

后端会从 `base_api` 表读取接口元数据，并根据 OpenAPI 元数据生成 MCP 工具定义。管理后台的“系统管理 / API 管理”页面可查看接口入参、参数映射、出参 Schema，并切换接口是否暴露为 MCP 工具。

MCP 工具列表在客户端执行 `tools/list` 时会按当前 `base_api.mcp_enabled` 动态过滤；工具调用前也会再次检查开关，禁用后不会继续转发到后端 HTTP 接口。工具调用内部复用本服务 HTTP 接口，默认转发到 `server.http.addr` 解析出的本地地址，并复用 HTTP 服务超时时间。

当前后端会把 Streamable HTTP MCP 处理器挂载到现有 HTTP 服务的 `/mcp` 路径，不再额外监听独立端口。例如 `server.http.addr = :7001` 时，MCP 地址为 `http://127.0.0.1:7001/mcp`。

## 静态资源

本地 OSS 根目录默认来自 `configs/oss.yaml` 的 `rootDirectory: ./data`。

- `/shop/*` 映射到 `backend/data/shop/*`，用于上传图片和本地文件访问。
- 后端会扫描 `backend/data` 下包含 `index.html` 的一级子目录，并按目录名注册 SPA 路由。
- 管理后台构建到 `backend/data/admin` 后，可通过 `http://localhost:7001/admin` 访问。
- 商城 H5 构建到 `backend/data/app` 后，可通过 `http://localhost:7001/app` 访问。

## 构建与镜像

构建 Linux amd64 可执行文件：

```bash
cd backend
make build
```

输出：`backend/bin/server`。

构建 Docker 镜像：

```bash
cd backend
make docker-build
```

`Dockerfile` 会复制 `bin/server`、`configs`、`certs`，并暴露 `6001`、`7001`。容器内默认工作目录为 `/app`，`data`、`configs`、`certs` 都声明为卷。

## 推荐与 AI 配置

Gorse 推荐客户端配置在 `configs/configs_local.yaml`：

```yaml
shop:
  recommend:
    entryPoint: http://127.0.0.1:8088
    apiKey: ...
```

`entryPoint` 需要指向 Gorse HTTP API 端口。Gorse 本地服务说明见 [../gorse/README.md](../gorse/README.md)。

大模型连接配置在 `configs/client_local.yaml` 的 `client.llm` 下；评价审核和摘要提示词在 `configs/configs_local.yaml` 的 `shop.prompt` 下。默认未配置有效密钥和模型时不会启用相关能力。

## 设计文档

| 文档 | 说明 |
| --- | --- |
| [后端服务设计](../docs/后端服务设计.md) | 后端分层、接口生成、业务域、任务和静态资源托管。 |
| [数据库与初始化数据设计](../docs/数据库与初始化数据设计.md) | 主业务库、推荐库、SQL 初始化、菜单和接口权限数据。 |
| [订单数据流转设计](../docs/订单数据流转设计.md) | 下单、支付、取消、退款、发货、收货、评价和删除状态流转。 |
| [推荐系统设计](../docs/推荐系统设计.md) | 推荐场景、Gorse 集成、本地兜底和后台管理能力。 |
| [推荐数据流转设计](../docs/推荐数据流转设计.md) | 匿名主体、推荐请求、推荐事件、业务事实回写和同步任务。 |
| [统计数据流转设计](../docs/统计数据流转设计.md) | 订单日统计、商品日统计、交易账单和后台分析口径。 |
| [评价与审核数据流转设计](../docs/评价与审核数据流转设计.md) | 评价、讨论、AI 摘要、审核和互动数据流转。 |

## 校验

后端默认检查命令：

```bash
cd backend
go test ./...
```

涉及生成代码、接口契约、数据库模型或依赖注入时，应同时执行匹配的 `make api`、`make openapi`、`make ts`、`make gorm-gen`、`make wire` 或 `make gen`。
