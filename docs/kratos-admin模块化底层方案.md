# kratos-admin 模块化底层方案

## 1. 文档定位

本文记录将当前项目中的通用后台能力下沉为 `kratos-admin` 底层模块的目标结构、公开契约和迁移顺序。本文是拆分前的设计基线，不代表拆分已经完成。

本方案采用“底层模块 + 业务项目组合”模式：

- `kratos-admin` 提供可版本化的 Go 后端基础能力；
- 独立的 admin 前端仓库发布 `@liujitcn/kratos-admin`；
- `shop` 通过 Go module 和 npm 包引用基础能力，在同一进程、同一数据库中注册商城业务模块；
- `shop` 保留自己的 module、商城业务域和商城端应用，不再复制或维护一套基础实现。

## 2. 已确认决策

| 主题 | 决策 |
| --- | --- |
| 后端仓库 | `kratos-admin` 只承载 Go 后端基础能力及随 module 发布的迁移资源，不放前端应用 |
| Go module | `github.com/liujitcn/kratos-admin`，`go.mod` 位于仓库根目录 |
| 业务项目 | `shop` 保留 `module github.com/liujitcn/shop` |
| 后端运行形态 | `shop` 引用 `kratos-admin`，在同一进程、同一数据库中组合业务模块 |
| 前端仓库 | 独立前端仓库发布 `@liujitcn/kratos-admin` |
| 前端消费方式 | 支持固定 npm 版本和 Git 源码依赖；包本身可以直接运行 |
| 版本策略 | Go module 与 npm 包使用相同版本号，消费者锁定精确版本 |
| 本地开发 | Go 使用 `go.work` 或未提交的 `replace`；前端使用 `pnpm link`、`file:` 或 Git 依赖 |
| 数据库 | `kratos-admin` 维护基础表和基础 seed，业务模块维护业务迁移；最终使用一个数据库 |
| 迁移资源 | 迁移 SQL 随 Go module 发布，并通过 `go:embed` 供运行时执行；具体迁移工具暂缓决定 |
| 多租户 | 共享数据库、共享表，通过 `tenant_id` 做行级隔离 |
| 租户删除 | 基础层定义资源检查/清理 hook，业务模块实现；检查和清理在同一事务中执行 |
| 代码生成 | 基础层提供通用生成引擎和模板契约，业务模块提供自己的协议、模板和目录元数据 |
| 历史迁移 | `kratos-admin` 初始仓库从当前最新代码生成干净快照，不要求保留 `shop` 历史 |

“统一为 `kratos`”只适用于基础项目身份和 module；商城业务目录、Proto package、接口路径和前端业务目录继续使用 `shop`，不得全局替换。

## 3. 仓库与依赖关系

```text
kratos-admin
├── go.mod                         # module github.com/liujitcn/kratos-admin
├── api/                           # base/common/system 协议与 Go 生成代码
├── configs/                       # 基础服务配置示例
├── internal/                      # 基础服务启动实现与组合根
├── migrations/                    # 随 Go module 发布的版本化 SQL 资源
├── pkg/                           # 基础设施、仓储、认证、事件、任务等稳定接口
├── server/                        # 基础 HTTP / gRPC / SSE / MCP 装配能力
└── service/                       # base/system 基础服务与代码生成引擎

admin-web（独立仓库）
└── @liujitcn/kratos-admin         # 可直接运行的后台壳和扩展 API

shop
├── backend/go.mod                 # module github.com/liujitcn/shop
├── service/shop                   # 商城业务实现
├── api/proto/shop                 # 商城协议
├── frontend/admin                 # 商城后台入口与业务模块
└── frontend/app                   # 商城端及其基础移动端调用
```

`shop` 的正式依赖固定为：

```go
module github.com/liujitcn/shop

require github.com/liujitcn/kratos-admin v0.1.0
```

本地开发时可在工作区文件中将该依赖替换为相邻的 `kratos-admin` 源码；生产构建和 CI 不提交本地路径，始终使用精确 tag。

基础层不得 import `service/shop`、`server/shop` 或 `api/gen/go/shop`。商城模块可以依赖基础层公开的稳定接口，但不能依赖基础层的内部实现。

## 4. 后端模块契约

### 4.1 组合根

`kratos-admin` 提供基础应用构造函数，例如 `NewApp(opts ...Option)`。`shop` 保留自己的启动入口，在组合根中创建基础应用并注册商城模块。基础层不拥有商城启动逻辑，也不通过运行时扫描、反射、`init()` 或 build tag 隐藏模块启用状态。

### 4.2 Module 接口职责

公开的 `Module` 契约需要覆盖以下可变能力；只为实际需要的能力提供实现：

- HTTP、gRPC、SSE、MCP 服务和路由注册；
- Wire/provider 集合和模块依赖；
- GORM 模型与迁移资源；
- 菜单、API 权限、字典、配置和默认数据；
- 租户资源检查与清理 hook；
- 定时任务、用户事件订阅和其他异步接缝；
- 代码生成模板、协议目标和前端目录元数据；
- AI 固定流程或工具注册。

模块注册应明确名称、依赖和执行顺序。重复的模块名、任务名、流标识、AI 流程标识或代码生成目标必须导致启动/生成失败，不能静默覆盖。

### 4.3 基础能力范围

除商城业务外，以下能力进入 `kratos-admin` 首版核心：

- 登录认证、JWT、OAuth provider、文件与对象存储适配；
- 租户、用户、角色、部门、岗位、菜单、API 权限、字典、配置、日志和任务；
- 移动端通用认证、地区和字典接口；
- SSE、队列、Swagger/OpenAPI、MCP 和 AI 通用能力；
- 通用代码生成引擎及其基础模板契约；
- 模块迁移、依赖注入和服务注册基础设施。

以下能力只属于 `shop`：商品、分类、SKU、订单、支付、推荐、评论、门店和商城工作台等业务。

## 5. 数据库、租户与迁移

### 5.1 单库模型

`shop` 初始化自己的数据库时，同时执行 `kratos-admin` 的基础迁移和商城模块迁移。基础表的唯一事实来源是 `kratos-admin`；商城不得在本地复制一份基础表定义后独立修改。

基础层负责租户、用户、角色、菜单、部门、岗位、权限规则等基础表和 seed。业务模块负责自己的表、菜单、API 权限、字典、配置和演示数据。

### 5.2 迁移执行

迁移 SQL 随 module 发布，并通过 Go 的 `embed` 资源提供给运行时。`NewApp` 统一执行基础迁移和已注册模块迁移，使用同一版本记录和数据库连接。具体采用 `goose`、`golang-migrate`、Atlas 或其他工具，另行决策；本方案先冻结迁移注册接口和资源所有权。

### 5.3 租户资源 hook

基础租户删除流程只处理基础表，并调用已注册业务模块的资源检查/清理 hook：

1. 开启一个数据库事务；
2. 调用所有模块的 `Check`，任一模块发现业务数据则拒绝删除；
3. 清理基础数据和各模块数据；
4. 任一错误都回滚整个事务；
5. 提交成功后再发布已删除事实事件。

基础层不能直接检查商城表。`shop` 通过模块实现门店、商品、订单、评论等业务数据的检查与清理。

## 6. Admin npm 包契约

### 6.1 包形态

`@liujitcn/kratos-admin` 是可直接运行的完整后台壳，包含登录、布局、权限路由、请求封装、主题、通用组件和基础系统页面。它同时导出组合入口，例如：

```ts
createAdminApp(options)
defineAdminModule(options)
```

`shop` 通过模块选项注入商城页面、API、菜单、主题和业务行为，最终构建自己的后台站点。基础包不能依赖商城页面，也不能假设消费者仓库存在某个 `views/shop` 文件。

### 6.2 RPC 边界

npm 包只发布 `base`、`common`、`system` 等基础 RPC 类型和客户端。`shop` 的商城协议由自己的后端 Proto 生成到商城前端源码，再通过业务模块接入后台；不在基础包中携带商品、订单等业务 RPC。

### 6.3 版本与消费

Go module 与 npm 包使用成对版本，例如：

```text
github.com/liujitcn/kratos-admin v0.1.0
@liujitcn/kratos-admin@0.1.0
```

`shop` 锁定精确版本，升级必须同时验证协议、权限、迁移、后端构建和前端构建。npm 包既支持 registry 安装，也支持 Git URL；Git 依赖用于源码联调，不替代正式版本锁定。

## 7. 拆分实施顺序

### 阶段 0：冻结基线

将当前 `shop` 工作树中已有的通用能力和业务改动整理后统一提交，以最新提交作为抽取输入。不回退用户改动，不要求保留抽取前的 Git 历史。

### 阶段 1：建立 `kratos-admin` Go module

1. 从当前最新代码抽取 `base`、`system` 和公共基础设施，商城代码不进入基础仓库。
2. 将 Go module 根设为 `github.com/liujitcn/kratos-admin`，重新整理目录和生成配置。
3. 消除基础层对商城表、协议、任务、SSE payload 和推荐类型的反向依赖。
4. 引入 `NewApp`、`Module`、迁移注册和租户资源 hook 契约。
5. 通过项目命令重新生成 Proto、OpenAPI、GORM、Wire 等产物。

### 阶段 2：拆分代码生成器

将代码生成器拆为基础引擎和业务扩展：基础仓库只保留通用渲染、管线和模板契约；`shop` 注册商城协议、模块路径、菜单和前端目录模板。

### 阶段 3：建立独立 Admin npm 包

1. 抽取登录、布局、权限、请求、主题、通用组件和系统页面。
2. 将基础 RPC 生成物作为包的一部分发布。
3. 实现 `createAdminApp` 与 `defineAdminModule`，让包既能独立启动，也能被 `shop` 组合。
4. 建立 npm registry、Git URL、本地 `pnpm link/file:` 三种消费路径的构建验证。

### 阶段 4：改造 `shop` 为消费者

1. 保留 `module github.com/liujitcn/shop`。
2. 通过 Go module 注册商城 `Module`，移除重复的基础服务、仓储、协议和启动装配。
3. 通过 npm 包组合商城后台；商城端保留在 `shop`，只调用基础移动端 API 和商城 API。
4. 商城数据库迁移、菜单、API 权限和代码生成模板由 `shop` 模块提供。

### 阶段 5：发布与升级

Go tag 与 npm package 使用相同版本号并成对发布。正式构建锁定精确版本；本地源码联调用 `go.work`/`replace` 和 `pnpm link/file:`。升级基础模块时，先在 `shop` 验证迁移、模块注册、权限、协议生成和双端构建，再更新锁定版本。

## 8. 验收标准

- `kratos-admin` 在没有商城代码时可以独立启动、初始化基础数据库并登录后台。
- `@liujitcn/kratos-admin` 不注入业务模块时可以直接运行。
- `shop` 通过精确 Go tag 和 npm 版本完成构建并启动。
- `shop` 只通过公开 Module/前端模块契约注册商城能力，不修改基础启动实现。
- 基础代码依赖图不包含 `shop` 反向依赖。
- 基础迁移与商城迁移在同一数据库中按注册顺序执行，租户删除 hook 能在同一事务中回滚。
- 基础 RPC 与商城 RPC 分离，所有生成产物由项目命令生成。
- 代码生成器可以生成通用模块，商城模板不进入基础仓库。

## 9. 待后续决策

- 迁移执行工具和迁移版本表的具体实现；
- 独立前端仓库的正式仓库名、构建发布流水线和 npm registry；
- Go module 与 npm 包的自动化成对发布流程；
- 基础模块的公开 API 文档和兼容性策略。
