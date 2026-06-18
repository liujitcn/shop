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
| `configs/configs_local.yaml` | 本地微信、支付、推荐等业务配置。 |
| `configs/ai.yaml`、`configs/local.yaml` | 大模型默认配置与本地覆盖配置。 |
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

也可以使用 `make run` 生成接口产物并启动服务。

## 初始化数据

后端完成自动建表后，先停止服务，再在仓库根目录依次导入基础数据和角色接口权限策略：

```bash
mysql -uroot -p shop_test < sql/default-data.sql
mysql -uroot -p shop_test < sql/casbin_rule.sql
mysql -uroot -p shop_test < sql/base_area.sql
```

如需演示商品、分类、轮播和商城服务数据，再导入：

```bash
mysql -uroot -p shop_test < sql/shop.sql
```

导入完成后重新启动后端，使 Casbin 加载最新权限策略。默认后台账号：

- `super / 112233`
- `admin / 112233`

说明：`sql/default-data.sql` 维护接口、菜单、角色和用户等基础数据，`sql/casbin_rule.sql` 维护 `admin`、`user`、`guest` 的角色接口权限策略。

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

当前 `base` 公共接口内已包含 AI 助手接口，路径前缀为 `/api/v1/base/ai/assistant`。会话与消息会持久化到 `ai_assistant_session`、`ai_assistant_message` 两张表；对话主链已经切到 `github.com/cloudwego/eino` 的消息与模型接口，并明确使用以下能力：

- `context`：每轮调用会把当前终端、用户名称、会话标题、摘要和历史消息组装为 Eino 消息列表。
- `tools`：AI 助手启动时会注册 `api/gen/go` 下已生成的 admin / app Agent 工具，并从当前终端可用工具中挑选相关内部 function tool，在需要时执行工具调用后回填结果；消息完成后会保存工具名称、状态、原始入参与原始出参，便于后台排查。
- `web search`：AI 助手仍保留 Responses Provider 默认启用的 `web_search` 工具，用于补充公开实时信息。
- `prompts`：AI 助手标准提示词内置在代码中，并结合当前会话上下文渲染为系统消息。
- `shortcuts`：`/api/v1/base/ai/assistant/shortcut` 按 `terminal` 和当前实际启用的 Agent 工具一次性返回快捷入口，前端只做本地切换展示，不再维护固定快捷问题数组。
- `direct stream`：管理端 AI 助手通过 `/api/v1/base/ai/assistant/session/{sessionId}/message` 直连 SSE 推送增量文本，发送接口会在完成事件中返回本轮消息，避免占用工作台共用 `/events` 流。
- `message status`：每轮消息使用 `GENERATING / SUCCESS / FAILED` 表达生成中、成功和失败状态，删除统一通过 `deleted_at` 逻辑删除。失败消息可通过 `/retry` 基于同一轮输入重新发送；助手输出可通过 `/regeneration` 基于同一轮输入重新生成；单轮消息删除会持久化到后端，回复完成后会同步刷新会话 `updated_at`。
- `flow action state`：商城端闭环流程生成的按钮、表单和卡片动作会写入 `source_message_id`、`action_id` 与 `flow_version`，后端只接受来自当前会话最新成功消息且仍存在于 `blocks_json` 的动作，防止历史消息里的上一步操作被重复触发。
- `branch session`：`/api/v1/base/ai/assistant/session/{sourceSessionId}/branch` 会复制锚点之前的成功消息，创建新的持久化分支会话。

其中 `ai_assistant_session.terminal` 已统一为终端枚举整型字段：`1` 表示商城端，`2` 表示管理端；对应的 proto 字段使用 `common.v1.Terminal`。

消息结构按一轮一条记录返回：`input_content` 保存输入类型与正文，`output_content` 保存输出类型、正文、回复来源、模型名、是否降级和降级原因，`attachments` 保存附件列表，`tools` 保存本轮实际使用的工具列表及工具原始请求/响应，`token` 保存模型真实输入、输出、缓存和总 Token 统计，`first_token_ms` 与 `duration_ms` 分别保存首 Token 耗时和总耗时。管理端附件会先走 `/api/v1/base/file/multi` 上传到 OSS，再由 AI 助手在服务端读取图片附件字节作为多模态视觉输入，文本、JSON、XML、CSV 类附件内容会直接拼入当前用户消息供模型参考。

AI 助手默认通过 `github.com/liujitcn/kratos-kit/ai/eino` 创建 OpenAI Responses AgenticModel，并启用 Responses 内置 `web_search` 服务端工具；评论审核与摘要通过同一组件创建 OpenAI Chat Completions AgenticModel。该能力要求配置的 `ai.model.cloud.baseUrl` 支持 OpenAI 兼容接口；AI 助手使用 `/responses`，普通只兼容 Chat Completions 的代理可能不支持。

## MCP 工具暴露

后端通过 `protoc-gen-go-mcp-tool` 生成 MCP 工具注册代码，服务启动时按本地服务实例注册工具。`base_api` 表保存接口元数据、`tool_name`、`tool_prompts`、`mcp_enabled` 和 `agent_enabled` 开关；`tool_prompts` 是 JSON 数组，默认由接口 `service_desc` 与 `desc` 组合生成多条工具提示词，会作为 MCP 工具和 Agent 工具的运行时描述与命中依据。管理后台的“系统管理 / API 管理”页面可查看、搜索工具名与工具提示词，可在列表中分别切换接口是否暴露为 MCP 工具或 Agent 工具，也可以通过编辑弹窗统一维护 MCP 启用状态、Agent 启用状态和工具提示词。

MCP 工具调用时会按工具名查询 `base_api.tool_name`，再检查 `mcp_enabled` 和当前终端归属；未启用或不属于当前终端时不会执行。MCP 工具列表会用 `base_api.tool_prompts` 合并后的文本覆盖生成描述。Agent 工具仍会完整注册加载，候选工具筛选和实际执行前按 `agent_enabled` 过滤，并用 `base_api.tool_prompts` 覆盖候选工具和工具目录里的生成描述；工具目录查询会展示当前终端完整注册工具名，并明确区别于本轮候选工具。商城端 AI 助手启动时会加载大部分 App 端生成工具，排除登录授权、支付回调等不适合由助手主动调用的敏感接口，并将每轮模型候选工具上限提升到 6 个；快捷入口覆盖推荐商品、待付款、评价、物流、购物车、收藏、收货地址、个人资料、门店入驻、商品分类、热门专区和商城服务说明等流程。工具调用链路直接走当前进程内服务实例，不再转发 HTTP，也不再依赖 `input_schema`、`arg_mapping`、`output_schema`。

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

`Dockerfile` 会复制 `bin/server` 与 `configs`，创建可挂载的 `certs` 目录，并暴露 `6001`、`7001`。容器内默认工作目录为 `/app`，`data`、`configs`、`certs` 都声明为卷。

## 推荐与 AI 配置

Gorse 推荐客户端配置在 `configs/configs_local.yaml`：

```yaml
shop:
  recommend:
    entryPoint: http://127.0.0.1:8088
    apiKey: ...
```

`entryPoint` 需要指向 Gorse HTTP API 端口。Gorse 本地服务说明见 [../gorse/README.md](../gorse/README.md)。

大模型默认配置在 `configs/ai.yaml` 的顶层 `ai.model` 下，本地覆盖配置放在 `configs/local.yaml`。评价审核、摘要和 AI 助手的标准提示词内置在代码中，不再通过商城配置覆盖。默认未配置有效密钥和模型时不会启用相关能力。评价图片审核会将本地 `/shop/*` 图片读取为多模态图片字节传给模型，避免把相对路径直接作为远端 `image_url` 使用；AI 助手会读取已上传附件中的图片字节作为视觉输入，文本类内容会拼入用户消息供模型参考。AI 助手的实时公开问题会由 OpenAI Responses API 的内置联网搜索工具补充上下文，评价审核和摘要不会启用联网搜索。评价审核和摘要使用 OpenAI 兼容 Chat Completions API，AI 助手使用 Responses API。`modelName`、`maxTokens`、`temperature`、`timeoutSeconds`、`maxRetries` 会传给对应模型接口；评论 Chat Completions 会省略 `temperature`，避免触发部分模型固定采样参数限制。模型判定不通过时必须返回具体违规类别、命中文本片段或图片序号和判定依据，缺少具体原因时会记录为审核异常等待人工复核。

`pkg/agent/provider` 只负责读取 `ai.model` 并调用 `github.com/liujitcn/kratos-kit/ai/eino` 装配评论 Chat 模型和 AI 助手 Responses 模型。

## 设计文档

| 文档 | 说明 |
| --- | --- |
| [后端服务设计](../docs/后端服务设计.md) | 后端分层、接口生成、业务域、任务和静态资源托管。 |
| [数据库与初始化数据设计](../docs/数据库与初始化数据设计.md) | 主业务库、推荐库、SQL 初始化、菜单和接口权限数据。 |
| [订单数据流转设计](../docs/订单数据流转设计.md) | 下单、支付、取消、退款、发货、收货、评价和删除状态流转。 |
| [推荐系统设计](../docs/推荐系统设计.md) | 推荐场景、Gorse 集成、本地兜底和后台管理能力。 |
| [推荐数据流转设计](../docs/推荐数据流转设计.md) | 匿名 ID、推荐请求、推荐事件、业务事实回写和同步任务。 |
| [统计数据流转设计](../docs/统计数据流转设计.md) | 订单日统计、商品日统计、交易账单和后台分析口径。 |
| [评价与审核数据流转设计](../docs/评价与审核数据流转设计.md) | 评价、讨论、评价摘要、审核和互动数据流转。 |

## 校验

后端默认检查命令：

```bash
cd backend
go test ./...
```

涉及生成代码、接口契约、数据库模型或依赖注入时，应同时执行匹配的 `make api`、`make openapi`、`make ts`、`make gorm-gen`、`make wire` 或 `make gen`。
