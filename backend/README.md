# backend

`backend` 是商城项目的 Go 后端服务，基于 `Kratos` 组织 HTTP / gRPC / SSE / MCP 接口、数据库访问、文件上传、静态资源托管、定时任务、推荐同步和 OpenAPI 文档。

## 目录职责

```text
backend
├── api
│   ├── proto           # base、common、system、shop 四个业务域的 proto 契约
│   └── gen/go          # proto 生成的 Go 代码
├── configs             # 服务运行配置
├── data                # 本地 OSS、日志、前端构建产物
├── internal/cmd/server # 实际启动入口、Wire 入口、内嵌 OpenAPI
├── pkg                 # 系统配置、公共能力、生成模型、队列、任务、推荐、微信支付
├── server              # base、system、shop 的 HTTP / gRPC / MCP 装配
└── service             # base、system、shop 业务服务与领域用例
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
| `configs/oauth.yaml`、`configs/oauth_local.yaml` | 三方登录与微信小程序 OAuth Provider 默认示例及本地应用密钥、回调地址与授权范围配置。 |
| `configs/configs.yaml` | 通用业务配置。 |
| `configs/configs_local.yaml` | 本地微信支付、推荐等业务配置。 |
| `configs/ai.yaml`、`configs/ai_local.yaml` | 大模型默认配置与本地覆盖配置。 |
| `configs/logger.yaml`、`trace.yaml`、`pprof.yaml`、`registry.yaml` | 日志、链路追踪、性能分析、注册中心相关配置。 |

默认数据库连接在 `configs/data.yaml`：

```yaml
source: root:112233@tcp(127.0.0.1:3306)/shop_test?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms
enable_migrate: true
```

首次启动会按当前模型自动建表。生产或共享环境应按实际情况调整账号、密码、库名和 `enable_migrate`。

三方登录与微信小程序登录的默认示例配置在 `configs/oauth.yaml`，本地应用密钥通过 `configs/oauth_local.yaml` 覆盖。启用跳转型 Provider 时，需要填写对应平台分配的 `clientId`、`clientSecret`、`redirectUri` 和 `scopes`；`redirectUri` 应配置为后端 OAuth callback 地址，前端创建授权地址时只传 `redirect_url` 作为登录或绑定完成后的前端接收地址。登录页会按已配置且后端组件支持的 Provider 动态展示三方登录入口。微信小程序登录使用 `wechatmini` Provider 的 `clientId` 和 `clientSecret`，由小程序端 `wx.login()` 获取 code 后调用 `base.v1.OauthService/CreateOauthSession` 创建会话，手机号授权也复用该 Provider 配置。

## 本地启动

先创建数据库：

```sql
CREATE DATABASE shop_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

启动服务：

```bash
cd backend
go run ./internal/cmd/server --conf ./configs
```

项目标识定义在 `internal/cmd/server/main.go` 的 `AppInfo.Project`。`pkg/config.GetAppInfo(ctx)` 会从 Bootstrap Context 读取该值，并通过依赖注入提供给应用注册名、自定义配置包装键、文件上传目录和静态资源前缀；不会改变 Go import、Proto package 或已发布的业务接口路径。

默认地址：

- HTTP：`http://localhost:7001`
- gRPC：`localhost:6001`
- SSE：`http://localhost:7001/events/1`
- OpenAPI：`http://localhost:7001/api/docs/openapi`（需携带已登录管理后台的 Bearer Token）

也可以使用 `make run` 生成接口产物并启动服务。

## 初始化数据

后端完成自动建表后，先停止服务，再在仓库根目录导入基础数据：

```bash
mysql -uroot -p shop_test < sql/default-data.sql
mysql -uroot -p shop_test < sql/base_area.sql
```

如需演示商品、分类、轮播和商城服务数据，再导入：

```bash
mysql -uroot -p shop_test < sql/shop.sql
```

导入完成后重新启动后端。启动阶段会先同步默认租户的 `tenant` 角色菜单，再根据全部角色关联的菜单和接口重新生成 `casbin_rule`，最后加载 Casbin 内存策略。默认后台账号：

- `super / 112233`
- `admin / 112233`

管理后台登录页不默认填充租户编码，需要用户手动输入；默认租户编码为 `0000`。`sql/default-data.sql` 维护默认租户、菜单、固定角色、用户和统计表结构升级段；后端启动时使用 GORM 执行 `TRUNCATE` 清空 `base_api`、`casbin_rule` 并重置自增 ID，根据当前 OpenAPI 重新生成接口元数据，再将默认租户的 `tenant` 角色菜单同步到所有普通租户，并按所有角色的菜单和接口关系重建租户化 Casbin 策略，策略动作使用真实 HTTP Method。默认角色固定为 `super(1)`、`tenant(2)`、`admin(3)`、`authuser(4)`、`user(5)`；商城用户注册时使用 `user`，认证通过后切换为 `authuser`；`admin`、`authuser`、`user` 角色不允许启用、禁用或删除。角色列表展示 `super` 和各租户的 `tenant` 内置角色；`super`、普通租户自己的 `tenant` 及其他租户的 `tenant` 禁止操作，默认租户自己的 `tenant` 权限模板允许编辑、删除、分配权限和启用/禁用。模板软删除后，默认租户可重新创建 `code=tenant` 的角色以恢复原记录，并将菜单重新同步到普通租户副本。默认租户为普通租户的自定义角色分配权限时，以目标租户的 `tenant` 角色作为权限上限。用户账号创建后禁止通过用户管理修改；绑定 `super` 或 `tenant` 内置角色的管理员账号禁止通过用户管理执行详情读取、编辑、删除、状态切换和重置密码，只能登录后通过个人中心维护自身资料、手机号和密码。用户与角色分页响应统一通过末位字段 `is_protected = 300` 提供管理保护标记，前端不再自行推断；角色和用户批量删除均要求请求中的全部 ID 可见且允许操作。受保护资源操作统一返回 `409 CONFLICT`。存量库升级统计租户字段后，需要按历史统计日期重跑 `GoodsStatDay` 和 `OrderStatDay` 任务。

订单链路使用 `order_trade` 聚合整笔支付和取消，`order_info` 按门店独立履约和退款；单门店与多门店走同一创建流程。`OrderStatDay` 分别按支付成功、交易创建、取消创建和退款成功时间汇总事实，`OrderRefundRetry` 定时补查结果不确定的微信退款。

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
- `make api`：根据 `api/proto/{base,common,shop,system}` 生成 Go 接口代码到 `api/gen/go`，并整理生成文件的导入。
- `make openapi`：生成 OpenAPI 文档到 `internal/cmd/server/assets/openapi.yaml`。
- `make ts`：生成管理后台和商城端 RPC TypeScript 代码，需先在前端模块执行 `pnpm install`。
- `make gorm-gen`：按当前数据库生成 `pkg/gen` 下的数据模型、查询对象和仓储代码。
- `make wire`：在 `internal/cmd/server` 下生成依赖注入代码。
- `make check-boundary`（仓库根目录执行）：检查基础模块与通用协议未反向依赖业务模块（shop 等，按 `service/` 目录自动发现）。
- `make gen`：生成 Go / OpenAPI / TypeScript 产物并格式化 Go 代码。

生成产物不要手工修改，优先通过对应命令更新：

- `api/gen/go`
- `pkg/gen`
- `internal/cmd/server/assets/openapi.yaml`
- `../frontend/admin/src/rpc`
- `../frontend/app/src/rpc`

当前 `base` 公共接口内已包含 AI 助手接口，路径前缀为 `/api/v1/base/ai`。`AiToolService` 提供快捷入口，`AiSessionService` 提供会话与消息列表，`AiMessageService` 提供消息发送及消息操作；会话与消息会持久化到 `ai_session`、`ai_message` 两张表。对话主链已经切到 `github.com/cloudwego/eino` 的消息与模型接口，并明确使用以下能力：

- `context`：每轮调用会把当前终端、用户名称、会话标题、摘要和历史消息组装为 Eino 消息列表。
- `tools`：AI 助手启动时会注册 `api/gen/go` 下已生成的 `system` / `shop` Admin、App Agent 工具，并从当前终端可用工具中挑选相关内部 function tool，在需要时执行工具调用后回填结果；消息完成后会保存工具名称、状态、原始入参与原始出参，便于后台排查。
- `web search`：AI 助手仍保留 Responses Provider 默认启用的 `web_search` 工具，用于补充公开实时信息。
- `prompts`：AI 助手标准提示词内置在代码中，并结合当前会话上下文渲染为系统消息。
- `shortcuts`：`/api/v1/base/ai/shortcut` 按 `terminal` 和当前实际启用的 Agent 工具一次性返回快捷入口，前端只做本地切换展示，不再维护固定快捷问题数组。
- `direct stream`：管理端 AI 助手通过 `/api/v1/base/ai/session/{sessionId}/message` 直连 SSE 推送增量文本，发送接口会在完成事件中返回本轮消息，避免占用工作台共用 `/events` 流。
- `message status`：每轮消息使用 `GENERATING / SUCCESS / FAILED` 表达生成中、成功和失败状态，删除统一通过 `deleted_at` 逻辑删除。失败消息可通过 `/retry` 基于同一轮输入重新发送；助手输出可通过 `/regeneration` 基于同一轮输入重新生成；单轮消息删除会持久化到后端，回复完成后会同步刷新会话 `updated_at`。
- `flow action state`：商城端闭环流程生成的按钮、表单和卡片动作会写入 `source_message_id`、`action_id` 与 `flow_version`，后端只接受来自当前会话最新成功消息且仍存在于 `blocks_json` 的动作，防止历史消息里的上一步操作被重复触发。
- `branch session`：`/api/v1/base/ai/session/{sourceSessionId}/branch` 会复制锚点之前的成功消息，创建新的持久化分支会话。

其中 `ai_session.terminal` 已统一为终端枚举整型字段：`1` 表示商城端，`2` 表示管理端；对应的 proto 字段使用 `common.v1.Terminal`。

消息结构按一轮一条记录返回：`input_content` 保存输入类型与正文，`output_content` 保存输出类型、正文、回复来源、模型名、是否降级和降级原因，`attachments` 保存附件列表，`tools` 使用 `AiToolCall` 保存本轮实际工具调用及其原始请求/响应，`token` 保存模型真实输入、输出、缓存和总 Token 统计，`first_token_ms` 与 `duration_ms` 分别保存首 Token 耗时和总耗时。管理端附件会先走 `/api/v1/base/file/multi` 上传到 OSS，再由 AI 助手在服务端读取图片附件字节作为多模态视觉输入，文本、JSON、XML、CSV 类附件内容会直接拼入当前用户消息供模型参考。

AI 助手默认通过 `github.com/liujitcn/kratos-kit/ai/eino` 创建 OpenAI Responses AgenticModel，并启用 Responses 内置 `web_search` 服务端工具；评论审核与摘要通过同一组件创建 OpenAI Chat Completions AgenticModel。该能力要求配置的 `ai.model.cloud.baseUrl` 支持 OpenAI 兼容接口；AI 助手使用 `/responses`，普通只兼容 Chat Completions 的代理可能不支持。

## MCP 工具暴露

后端通过 `protoc-gen-go-mcp-tool` 生成 MCP 工具注册代码，服务启动时按本地服务实例注册工具。`base_api` 表保存接口元数据、`tool_name`、`tool_prompts`、`mcp_status` 和 `agent_status` 状态；`tool_prompts` 是 JSON 数组，默认由接口 `service_desc` 与 `desc` 组合生成多条工具提示词，会作为 MCP 工具和 Agent 工具的运行时描述与命中依据。管理后台的“系统管理 / API 管理”页面可查看、搜索工具名与工具提示词，可在列表中分别切换 MCP 工具或 Agent 工具状态，也可以通过编辑弹窗统一维护两个工具状态和工具提示词。

MCP 工具调用时会按工具名查询 `base_api.tool_name`，再检查 `mcp_status` 是否为启用状态以及当前终端归属；未启用或不属于当前终端时不会执行。MCP 工具列表会用 `base_api.tool_prompts` 合并后的文本覆盖生成描述。Agent 工具仍会完整注册加载，候选工具筛选和实际执行前按 `agent_status` 是否为启用状态过滤，并用 `base_api.tool_prompts` 覆盖候选工具和工具目录里的生成描述；工具目录查询会展示当前终端完整注册工具名，并明确区别于本轮候选工具。商城端 AI 助手启动时会加载大部分 App 端生成工具，排除登录授权、支付回调等不适合由助手主动调用的敏感接口，并将每轮模型候选工具上限提升到 6 个；快捷入口覆盖推荐商品、待付款、评价、物流、购物车、收藏、收货地址、个人资料、门店入驻、商品分类、热门专区和商城服务说明等流程。工具调用链路直接走当前进程内服务实例，不再转发 HTTP，也不再依赖 `input_schema`、`arg_mapping`、`output_schema`。

当前后端会按 `server.mcp.transport: TRANSPORT_IN_PROCESS` 把 Streamable HTTP MCP 处理器挂载到现有 HTTP 服务，并通过 `/mcp/{terminal}` 按服务关键字过滤工具。例如 `server.http.addr = :7001` 时，管理端 MCP 地址为 `http://127.0.0.1:7001/mcp/admin`。

## 代码生成配置

代码生成配置当前包含数据库表配置、字段配置和 Proto 接口配置。接口按职责拆分为：

- `CodeGenTableService`：数据库表选项、代码生成表配置分页和 CRUD。
- `CodeGenColumnService`：数据库字段元数据查询、字段配置查询和保存。
- `CodeGenProtoService`：按表与字段配置检查所需 Proto 接口，并保存缺失接口的生成选择。

数据库表与字段选项固定读取当前连接库的 `information_schema`，不接受客户端传入 SQL 片段。字段配置持久化在 `code_gen_column`：查询、列表、表单分别维护自己的选项配置，并使用 `sort` 字段作为三类页面元素的共用顺序；字段名为 `status`、`state`、后缀为 `_status`、`_state` 或数据库类型为 `tinyint` 时，默认按状态字段处理，查询使用 `status` 字典下拉，列表和表单使用开关，并在各自选项中保存 `status` 字典、开启值 `1` 和关闭值 `2`；后端根据已启用的列表开关自动推导设置状态 Proto 接口，方法名直接使用实际字段名，例如 `status` 生成 `SetBaseDeptStatus`，`mcp_status` 生成 `SetBaseApiMcpStatus`。表单树形选择默认单选；仅 JSON 字段可配置多选，生成的表单契约使用 `repeated int64` 并在业务层与 JSON 数组双向转换。字段配置查询接口不返回数据库主键和 `deleted_at`，内部 Proto 推导仍使用完整字段集合。树形页面单独配置父节点字段和树显示字段，生成的表格隐藏父节点列并把树显示字段放在首个数据列，保持与部门页面相同的缩进效果；树形 Option 默认使用 `parent_id`、`name`、`id` 作为父节点、显示和值字段，接口和字段选项均可勾选懒加载，懒加载请求携带父节点 ID 和 `lazy` 标记并返回 `has_children`。这些配置在 Proto 中定义为结构化消息，并通过 mapper 与数据库 JSON 列双向转换。Proto 接口配置持久化在 `code_gen_proto`，不同接口类型的可变字段统一保存在 `config` JSON 列：`option` 使用显示字段和值字段，`tree` 额外使用父节点字段和懒加载开关，`status` 使用状态字段，`crud` 与 `list` 不需要类型配置；只有勾选缺失时生成的接口才校验配置。检查结果会读取仓库内目标 Proto 文件判断 RPC 是否存在。

`CodeGenService` 与 `CodeGenCase` 在只读加载上述现有配置后提供代码预览、单项或批量生成和任务进度查询。预览请求可临时覆盖本次单表预览的仓库内输出路径，但不会回写配置；生成任务只接收表配置 ID，按各表默认路径串行写入，并固定执行接口产物、Wire 和格式化命令。外部选项目标已有的 Biz、Service 和前端 API 方法保持不变，只补齐缺失方法；已有前端页面按模板节点、配置项、顶层声明和函数的稳定功能键增量合并，同名生成项按最新配置更新，生成器未知的扩展功能保持原顺序追加到生成项之后。页面结构无法安全解析时，预览继续标记跳过，实际生成会在任何文件写入前取消整个批次，避免只覆盖接口与后端。实际生成会先在数据库事务中同步菜单权限，菜单脚本失败时不会写入页面或其他生成文件；写入前会快照全部目标文件，文件写入或事务提交失败时恢复原有内容并删除本次新建文件。生成任务保存在进程内并按用户隔离，通过任务 ID 的 SSE 通道推送实时进度，同时支持查询接口轮询恢复。生成流程不会修改代码生成表的状态或备注；启用菜单同步且前端页面所需 RPC 完整时，只同步目标业务页面及实际存在的按钮接口权限。

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

推荐查询会优先使用已启用的 Gorse 责任链；Gorse 调用失败或未命中商品时，自动回退到本地推荐策略，避免推荐位因远端冷启动或暂时不可用而返回空结果。推荐与评价异步消费者统一设置 60 秒超时，防止外部服务长期无响应时占用消费协程。

大模型默认配置在 `configs/ai.yaml` 的顶层 `ai.model` 下，本地覆盖配置放在 `configs/ai_local.yaml`。评价审核、摘要和 AI 助手的标准提示词内置在代码中，不再通过商城配置覆盖。默认未配置有效密钥和模型时不会启用相关能力。评价图片审核会将本地 `/shop/*` 图片读取为多模态图片字节传给模型，避免把相对路径直接作为远端 `image_url` 使用；AI 助手会读取已上传附件中的图片字节作为视觉输入，文本类内容会拼入用户消息供模型参考。AI 助手的实时公开问题会由 OpenAI Responses API 的内置联网搜索工具补充上下文，评价审核和摘要不会启用联网搜索。评价审核和摘要使用 OpenAI 兼容 Chat Completions API，AI 助手使用 Responses API。`modelName`、`maxTokens`、`temperature`、`timeoutSeconds`、`maxRetries` 会传给对应模型接口；评论 Chat Completions 会省略 `temperature`，避免触发部分模型固定采样参数限制。模型判定不通过时必须返回具体违规类别、命中文本片段或图片序号和判定依据，缺少具体原因时会记录为审核异常等待人工复核。

`pkg/agent/eino/model` 负责读取 `ai.model` 并调用 `github.com/liujitcn/kratos-kit/ai/eino` 装配评论 Chat 模型和 AI 助手 Responses 模型；业务运行时分别位于 `service/base/agent/ai` 和评价审核相关服务。

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
| [AI 助手设计](../docs/AI助手设计.md) | 会话、消息、工具、SSE 与商城固定流程。 |

## 校验

后端默认检查命令：

```bash
cd backend
go test ./...
```

涉及生成代码、接口契约、数据库模型或依赖注入时，应同时执行匹配的 `make api`、`make openapi`、`make ts`、`make gorm-gen`、`make wire` 或 `make gen`。
