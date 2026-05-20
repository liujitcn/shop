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
- SSE：`http://localhost:7001/events/1`
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

当前 `base` 公共接口内已包含 AI 助手接口，路径前缀为 `/api/v1/base/ai/assistant`。会话与消息会持久化到 `ai_assistant_session`、`ai_assistant_message` 两张表；对话主链已经切到 `github.com/go-kratos/blades` 的 `Agent + Runner` 机制，并明确使用以下能力：

- `session / state`：每个后台会话在服务端映射为独立的 Blades Session，当前终端、场景、用户名称、会话标题、摘要等状态会注入到 session state。
- `chat-only`：AI 助手默认按通用纯聊天模式工作，不注册 MCP、业务工具或 Blades Memory 工具，任何主题都可以直接用模型能力回答。
- `prompts`：AI 助手提示词来自商城配置 `prompt.ai_assistant`，并结合 session state 以模板形式渲染。
- `runstream`：管理端 AI 助手通过 `/events/{stream}` 上的 SSE 流推送增量文本，按后台用户隔离专属 stream，避免不同管理员之间互串回复内容。

其中 `ai_assistant_session.terminal` 已统一为终端枚举整型字段：`1` 表示商城端，`2` 表示管理端；对应的 proto 字段使用 `common.v1.Terminal`。

当前阶段助手主流程先聚焦“通用纯聊天”，消息结构以普通文本回复为主；业务工具执行、确认卡动作、MCP 调用等能力已从默认主链移出，后续在主流程稳定后再按场景追加。消息结构仍会返回回复来源、模型名、是否降级和降级原因；未配置模型或模型调用失败时会明确回退为本地兜底回复。管理端附件会先走 `/api/v1/base/file/multi` 上传到 OSS，再由 AI 助手在服务端读取图片附件字节作为多模态视觉输入，文本类附件内容会直接拼入当前用户消息供模型参考。

AI 助手默认使用 `pkg/agent/provider` 内的 OpenAI Responses Provider，并启用 OpenAI 内置 `web_search` 工具；这类模式适合回答新闻、天气、金价、行情等强实时问题。该能力要求配置的 `baseUrl` 支持 OpenAI Responses API，普通 OpenAI-compatible Chat Completions 代理可能不支持 `/responses`。

AI 图片生成已改为异步队列模式，资源接口位于 `/api/v1/base/ai/image`：列表使用 `GET /api/v1/base/ai/image`，详情使用 `GET /api/v1/base/ai/image/{id}`，创建使用 `POST /api/v1/base/ai/image`，失败或超时后可通过 `POST /api/v1/base/ai/image/{id}/retry` 重新投递队列。创建记录会先写入 `ai_image`，再投递 `ai_image_generate_queue` 后台生成；调用模型的参数快照会保存到 `params_json`，不保存密钥等敏感配置。生成状态使用 `base.v1.AiImageStatus` 枚举：`PENDING` 待处理、`RUNNING` 生成中、`SUCCESS` 成功、`FAILED` 失败、`TIMEOUT` 超时。

图片生成仍通过 `pkg/agent/provider.ImageClient` 复用 `github.com/go-kratos/blades/contrib/openai` 的图片生成 Provider。生成成功后会回写 `request_id` 批次编号与 `image_urls_json` 图片结果；保存到 OSS 时目录为 `/shop/ai/images/{yyyy/mm/dd}/{request_id}`，图片结果会返回 `storage_path` 便于追溯素材来源。提示词润色接口仍位于 `/api/v1/base/ai/image/prompt/polish`，复用 `client.llm` 对话模型把用户输入整理成更适合文生图的中文提示词。

## MCP 工具暴露

后端通过 `protoc-gen-go-mcp-tool` 生成 MCP 工具注册代码，服务启动时按本地服务实例注册工具。`base_api` 表只保存接口元数据、`mcp_tool_name` 和 `mcp_enabled` 开关；管理后台的“系统管理 / API 管理”页面可查看工具名，并切换接口是否暴露为 MCP 工具。

MCP 工具调用时会按工具名查询 `base_api.mcp_tool_name`，再检查 `mcp_enabled` 和当前终端归属；未启用或不属于当前终端时不会执行。工具调用链路直接走当前进程内服务实例，不再转发 HTTP，也不再依赖 `input_schema`、`arg_mapping`、`output_schema`。

当前后端会按 `server.mcp.transport: TRANSPORT_IN_PROCESS` 把 Streamable HTTP MCP 处理器挂载到现有 HTTP 服务，并通过 `/mcp/{terminal}` 按服务关键字过滤工具。例如 `server.http.addr = :7001` 时，管理端 MCP 地址为 `http://127.0.0.1:7001/mcp/admin`。

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

大模型连接配置在 `configs/client_local.yaml` 的 `client.llm` 下；评价审核和摘要提示词在 `configs/configs_local.yaml` 的 `shop.prompt` 下。默认未配置有效密钥和模型时不会启用相关能力。评价图片审核会将本地 `/shop/*` 图片读取为多模态图片字节传给模型，避免把相对路径直接作为远端 `image_url` 使用；AI 助手当前按纯聊天模式读取已上传附件中的图片字节作为视觉输入，文本类内容会拼入用户消息；AI 图片生成复用同一组 `baseUrl/apiKey` 并默认使用 `gpt-image-2`。实时问题会由 OpenAI Responses API 的内置联网搜索工具补充上下文。`client.llm.reasoningEffort` 默认设为 `xhigh`，AI 助手通过 Responses 原生 `reasoning.effort` 传递；`maxOutputTokens`、`temperature`、`topP` 也会传给 Responses，`seed`、`frequencyPenalty`、`presencePenalty`、`stopSequences` 仅在 Chat Completions 模型链路完整生效，若走 sub2api/Codex 中转需以中转实际支持为准，可通过 `extraFields` 显式透传兼容字段。模型判定不通过时必须返回具体违规类别、命中文本片段或图片序号和判定依据，缺少具体原因时会记录为审核异常等待人工复核。

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
