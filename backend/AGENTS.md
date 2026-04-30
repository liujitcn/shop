# Codex 规则

## 基本原则
- 通用工作方式、Go 规范、Proto 规范均以 `.codex` 为主。
- 本文件仅保留后端模块的生成链路、业务流程、查询方式、错误体系和数据库约定。

## 提交与发布约定
- 提交流程固定为以下顺序：
  1. 先执行生成与测试，必须保证 `go test ./...` 通过；若失败涉及依赖冲突，先修复 `go.mod/go.sum` 后重试。
  2. 再检查并更新 `README.md`，确保文档与代码行为一致。
  3. 最后执行提交与推送，并将 `README.md` 改动与本次代码改动一起提交。
- 用户要求“提交”时，默认执行完整发布动作：`git commit` + `git push`。
- 用户要求“提交”时，默认先执行 `git add -A`，将未暂存与未跟踪文件一并加入本次提交（遵循 `.gitignore` 与用户明确排除的文件）。
- 未明确指定分支时，推送当前分支到同名远程分支。
- `git commit -m` 信息默认使用中文，简洁描述本次变更。
- 若用户未指定提交信息，按变更内容自动生成中文提交信息。

## Tag 规则
- 当用户要求“打 tag”时，默认执行 `make tag` 命令。

## 新增业务开发流程
- 所有新增业务开发，默认开发顺序固定为：
  1. 先判断是否需要新增数据表。
  2. 若不需要新增表，则直接从接口契约定义开始，按 `proto -> api/gen -> service/biz -> 前端` 顺序开发。
  3. 若需要新增表，则必须先从 `configs` 目录读取当前数据库配置，确认目标库连接信息后，直接将表结构写入数据库，再继续后续开发。
- 后续凡是涉及 `proto`、`api/gen`、`pkg/gen`、依赖注入、RPC 桩代码、数据库模型等生成内容时，必须使用仓库既有命令生成，禁止手写或复制生成结果。
- 常见生成动作必须优先通过项目命令执行，例如 `make gen`、`make gorm-gen`、`make wire`；若某类生成存在等价脚本或 make 目标，必须先使用仓库已约定命令，不得绕过生成流程直接修改产物。
- 生成代码前，先确认目标命令与当前改动范围匹配；生成代码后，若产物发生变化，必须一并纳入本次检查，禁止只改源文件不更新生成结果。
- 读取数据库配置时，优先检查 `configs/data.yaml` 中的 `data.database` 配置；若存在本地环境覆盖配置，则一并核对实际生效配置，避免将表建错库。
- 需要新增表时，禁止仅停留在本地 SQL 草稿或仅修改 `sql/shop.sql`；必须先把表结构真正建到当前开发数据库中，再继续生成 `pkg/gen`、编写 `proto`、实现服务和前端联调。
- 新增业务时，除业务代码外，必须同步补齐对应的功能脚本；不要只改业务代码而遗漏数据库和权限初始化动作。
- 功能脚本中需要覆盖新增业务涉及的数据库变更，包括但不限于：
  1. `ALTER TABLE` 修改已有表结构。
  2. 初始化菜单权限数据，对应 `base_menu` 表。
- 上述新增业务功能脚本统一写入 `sql/default-data.sql`，不要分散到其他临时 SQL 文件中。
- 若新增业务同时涉及菜单、接口权限、表结构变更，以上脚本内容必须在同一次开发中一并完成，禁止拆成“代码已支持，但库表/API/菜单初始化后补”。
- 新表落库后，再继续执行：
  1. 更新或生成 `pkg/gen/query` 等数据库访问代码。
  2. 定义或更新 `api/protos` 下的接口契约。
  3. 生成 `api/gen` 与前端 `src/rpc` 相关类型。
  4. 实现 `service/biz` 逻辑与前端页面。
- 若当前需求不引入新表，则从上述流程的“接口契约定义”步骤开始，不额外执行建表动作。

## 注释补充约定
- 后续新增或修改代码时，人工编写注释统一使用中文；生成代码头注释、Go 指令、被注释掉的 import 路径、第三方协议或标准库固定文本不受此限制。
- 后续新增或修改代码时，必须为每个新增或修改的方法补充中文方法注释。
- 后续新增或修改 `proto` 字段时，必须同时补齐字段后的中文尾注释，格式与现有契约保持一致，例如 `];  // 数量`。
- `proto` 字段注释只保留一种表达方式；若字段尾部已有中文注释，则不要在同一字段上方再重复书写语义相同的 `//` 注释。
- 后续新增或修改 `proto` 字段时，必须同时补齐 `(gnostic.openapi.v3.property).description`，且字段尾注释与 `description` 语义保持一致。
- 后续新增或修改 `proto` 中的 `message` 时，必须为每个 `message` 补充中文注释；注释应直接表达业务语义，例如“订单分页查询条件”“商品详情响应”“收货地址”。

## 代码修改约束
- DTO、VO、查询结果承载结构、聚合分组键等数据承载类型，禁止定义在 Case 文件中；统一放到对应模块的 `dto` 目录下维护。
- 后端代码必须优先复用 `kratos-kit`、`go-utils`、`gorm-kit` 中已有的方法、工具函数、通用封装与约定，禁止在项目内重复实现等价能力。
- `kratos-kit` 优先复用范围包括：`api`、`bootstrap`、`config` 及其配置中心子模块、`logger` 及其日志实现子模块、`registry` 及其注册中心子模块、`tracer`、`tracing`、`auth` 及其认证鉴权子模块、`cache`、`queue`、`locker`、`oss`、`database/gorm`、`broker`、`transport`、`rpc`、`swagger-ui`、`pprof`、`captcha`、`sdk`、`runtime`、`utils`。
- `go-utils` 优先复用范围包括：根包 `utils`、`byte`、`crypto`、`geoip` 及其子模块、`http`、`id`、`io`、`ip`、`jwt`、`map`、`mapper`、`slice`、`string`、`stringcase`、`time`、`tls`、`trans`。
- `gorm-kit` 优先复用范围包括：`repo` 通用仓储能力、分页、批量写入策略、函数式查询选项，以及 `gen` 代码生成入口。
- 判断是否已有现成实现时，不能只检查顶层目录或当前仓库内已经直接引用的包名，必须同时检查上述基础库及其子模块。
- 若基础库或其子模块中已提供可满足需求的方法，默认直接复用；只有在现有能力明显不适用、无法扩展或会引入不合理副作用时，才允许新增本地实现。
- 需要新增本地实现时，必须先确认基础库及其相关子模块中不存在合适方案，并保持新增实现与这些基础库的使用方式、返回结构和错误处理风格一致。
- 后续新增或修改涉及数据库查询条件的代码时，查询条件必须先通过 `Query(ctx)` 取得对应查询对象，再通过 `[]repo.QueryOption` 收敛条件，禁止直接在 `List`、`Page`、`Find` 等调用里内联堆砌大量 `repo.Where(...)`。
- 推荐写法固定为先定义查询对象和条件切片，再按需 `append` 条件，例如：
  - `query := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo`
  - `opts := make([]repo.QueryOption, 0, 2)`
- 后续查询调用应统一使用 `opts...` 传入，例如 `List(ctx, opts...)`、`Page(ctx, pageNum, pageSize, opts...)`、`Find(ctx, opts...)`。
- 如果当前方法内只有一个 query 对象，直接使用 `query`、`opts` 变量名，不用追加表相关信息，比如 `goodsQuery`、`goodsOpts`。
- 后续新增或修改后端数据库查询代码时，禁止手动拼接运行时 SQL，默认优先使用 `gorm/gen` 字段表达式、`repo.QueryOption`、`query.Field.Eq/In/Gte/Lt/Sum().IfNull(0)`、`query.Field.Count()`、`query.Field.Distinct()` 等类型化能力。
- 禁止新增 `.Raw(...)`、`db.Exec(...)`、`UnderlyingDB().Where("...")`、`Select("...")`、`Joins("...")`、`Order("...")`、`Table("...")`、`gorm.Expr(...)`、`field.NewUnsafeFieldRaw(...)` 等手写 SQL 或原生表达式；确有数据库特性无法用 gorm/gen 表达时，必须先说明原因、影响范围和参数安全策略，并优先封装到极小范围内。
- 聚合统计优先使用 gorm/gen 原生聚合函数，例如 `query.Amount.Sum().IfNull(0).As("amount")`、`query.ID.Count().As("count")`，禁止为了 `SUM`、`COUNT`、`COALESCE`、`DATE_FORMAT`、`MONTH`、`DAY` 等常见聚合或时间分组手写 SQL 字符串。
- 复杂查询不要强行写成一个 SQL；优先拆成“先查主表编号或明细，再用 `IN` 参数查询关联数据，最后在 Go 侧去重、合并、分组、排序或计算”的流程。
- JSON 字段筛选默认先读取候选记录并在 Go 侧解析过滤；只有在数据量、索引或性能明确要求数据库侧 JSON 函数时，才允许评估使用原生 JSON SQL，并需要在代码附近说明原因。
- 修改后端查询代码后，必须复扫运行时代码中是否仍存在手写 SQL 入口，至少检查 `.Raw(`、`NewUnsafeFieldRaw`、`gorm.Expr`、`JSON_TABLE`、`JSON_OVERLAPS`、`UNION ALL`、`sql :=`、`sql +=`、以及 GORM 字符串式 `Where/Select/Joins/Order/Table`。
- 设计 DTO、models 相互转换时，优先使用 `mapper` 包中的工具方法，禁止直接在业务代码中写冗余的类型转换。
- 后续新增或修改 Case 结构体依赖时，禁止将“当前 Case 主职责以外的 Repo”写成匿名嵌入字段；这类 Repo 必须使用具名字段保存，例如 `baseUserRepo *data.BaseUserRepo`、`goodsInfoRepo *data.GoodsInfoRepo`。
- 当前 Case 自身直接对应的主 Repo 可以按项目现有风格继续匿名嵌入；`*biz.BaseCase` 也允许匿名嵌入；除此之外的 Repo 一律不允许写成 `*data.XxxRepo` 这种匿名字段。
- `api/gen`、`pkg/gen` 生成的代码禁止直接修改，也禁止手工添加文件。
- 不要写 `_test.go`；如果自己临时需要，测试结束后删除。

## 错误处理规则
- 后续新增或修改后端错误处理时，错误体系只解决 3 件事：让调用方稳定判断少量必要场景、让用户看到合适提示文案、让日志保留足够排障信息；禁止恢复“按文案拆枚举”“按资源拆枚举”“按场景拆枚举”的旧做法。
- `reason` 只负责稳定的一级分类；`message` 只负责面向用户的提示文案；`metadata` 只负责调用方确有需要时的二级结构化信息；`cause`、日志与链路信息只负责保留底层技术细节。禁止把每一句中文提示、每一种资源类型或每一个细分场景都扩展成新的顶层 `reason`。
- 顶层 `reason` 只允许使用：`INVALID_ARGUMENT`、`UNAUTHENTICATED`、`PERMISSION_DENIED`、`RESOURCE_NOT_FOUND`、`CONFLICT`、`INTERNAL_ERROR`。
- `error.proto` 中上述 6 个顶层 `reason` 的集合视为冻结集合；其语义分别对应“请求有误 / 身份不成立 / 身份成立但不允许 / 资源不存在 / 当前状态冲突 / 其余内部异常”，并分别对应 HTTP `400/401/403/404/409/500` 语义；禁止把枚举值当作具体业务句子的清单继续扩充。
- 未经明确确认，禁止新增新的顶层 `reason`、禁止在 `error.proto` 中继续追加 `TOKEN_EXPIRED`、`USER_NOT_FOUND`、`STATE_CONFLICT`、`UNIQUE_VIOLATION` 这类旧风格或派生风格枚举。
- 只有在“现有 6 类无法准确表达一级语义、调用方存在明确稳定的跨页面或跨端分支需求、且该需求不能通过 `reason + metadata` 表达、并且该分类具备长期稳定性”这 4 个条件同时满足时，才允许评估是否新增顶层 `reason`；缺少任一条件都不得新增。
- 后续新增或修改业务错误时，禁止直接把 `errors.New(...)`、`fmt.Errorf(...)` 作为对外业务错误返回；对外返回必须优先使用 `shop/pkg/errorsx` 中的统一构造方法。
- `repo` 层、数据库访问层、第三方 SDK 层默认返回原始错误，不负责对外错误分类，也不要在这些层直接拼面向前端的业务提示。
- `biz` 层负责错误分类、对外 `message`、必要的 `metadata` 与 `cause`；`service` 层不做业务分类判断，只负责打印原始日志和统一兜底包装。
- `service` 层捕获错误后，默认统一使用 `errorsx.WrapInternal(err, "xxx失败")` 返回；若 `biz` 层已经返回结构化错误，`WrapInternal` 会直接透传，禁止在 `service` 层二次改写已有 `reason`。
- `service` 层打印错误时，必须直接在当前方法内使用 `log.Errorf("方法名 %v", err)` 这一类格式，禁止再封装新的日志 helper，禁止使用 `log.Error("xx err:", err.Error())` 这类会产生 `err:error:` 冗余前缀的写法。
- 默认优先使用 `reason + message`；只有当调用方确实需要在同一个 `reason` 下继续做稳定分支时，才允许补充 `metadata`。禁止把整句中文提示复制到 `metadata`，也禁止无限扩展 `metadata` 键，把它变成第二套枚举系统。
- `INVALID_ARGUMENT` 只用于“请求本身有问题”的场景，例如参数为空、格式错误、取值非法、验证码错误、地址错误、订单商品为空；能通过修改请求立即解决的问题，优先归入这一类。
- `UNAUTHENTICATED` 只用于“身份不成立”的场景，例如用户名或密码错误、未登录、token 无效、token 过期、刷新令牌无效；登录场景中的“用户不存在”也统一收敛到这一类，避免账号探测。
- `PERMISSION_DENIED` 只用于“身份成立但不允许操作”的场景，例如账号禁用、角色禁用、无权限、不能操作超级管理员、不能操作超级管理员角色。
- `RESOURCE_NOT_FOUND` 只用于“目标资源不存在”的场景，例如用户不存在、角色不存在、部门不存在、订单不存在、定时任务不存在、调用目标不存在；登录失败场景不要归到这一类。
- `CONFLICT` 只用于“资源存在，但当前状态不允许继续操作”的场景，例如唯一约束冲突、存在子节点不能删除、状态不允许变更、资源受保护、重复支付、重复退款、已支付不可取消。
- 唯一约束冲突必须优先使用 `errorsx.UniqueConflict(...)`；存在子资源不能删除必须优先使用 `errorsx.HasChildrenConflict(...)`；状态不匹配必须优先使用 `errorsx.StateConflict(...)`；受保护资源必须优先使用 `errorsx.ProtectedResourceConflict(...)`。
- 冲突类错误需要补充结构化信息时，必须优先复用 `errorsx` 中已经定义的元数据键：`conflict_type`、`resource`、`field`、`constraint`、`child_resource`、`current_state`、`expected_state`；不要在同类错误上随意新增命名分散的 metadata key。
- `INTERNAL_ERROR` 作为其余异常的唯一兜底分类，适用于数据库错误、Redis 错误、RPC 错误、第三方服务错误、配置错误、数据不一致、文件系统错误，以及“按当前请求内容无法直接修复”的内部异常。
- 需要把底层错误转换为结构化业务错误时，必须保留原始错误链，优先使用 `.WithCause(err)`；禁止只返回新的中文文案而把原始错误完全丢掉。
- 识别 MySQL 唯一键冲突时，必须优先复用 `errorsx.IsMySQLDuplicateKey(err)`，不要在业务代码里重复写数据库错误码判断。
- 前端是否展示弹窗、表单提示、跳转登录，必须由 `code/reason/metadata` 决定；后端不要新增依赖中文 `message` 才能稳定分支的协议约定。

## 数据库索引命名规则
- 唯一索引命名格式：`unique_表名`，例如 `unique_order`、`unique_goods`。
- 普通索引命名格式：`idx_表名_字段1_字段2_...`，例如 `idx_order_status`、`idx_order_user_id_created_at`。
- 在 GORM 模型定义中添加索引时，必须严格遵守此命名规范：
  - 唯一索引：`gorm:"column:字段名;type:字段类型;uniqueIndex:unique_表名,priority:N;comment:注释"`
  - 普通索引：`gorm:"column:字段名;type:字段类型;index:idx_表名_字段1_字段2,priority:N;comment:注释"`

## 数据库命名规则
- 表名全部小写，单词间用下划线分隔，例如 `order`、`order_goods`、`base_user`、`goods_category`。
- 字段名全部小写，单词间用下划线分隔，例如 `user_id`、`order_no`、`created_at`、`category_id`。
- 命名使用有意义的英文单词，避免拼音；优先使用名词，并保持同一概念命名一致。

## Proto 契约补充约定
- Proto 契约通用命名规则以 `.codex` 为主，本节仅保留当前项目补充约束。
- proto package 必须带版本号，按终端或模块分层命名，例如 `admin.v1`、`app.v1`、`base.v1`、`common.v1`。
- proto 文件目录必须与 package 对齐，版本号必须落到目录层级，例如 `api/protos/admin/v1/base_api.proto` 对应 `package admin.v1;`。
- 当前项目内部 proto package 不加 `shop` 前缀；禁止新增 `shop.admin.v1`、`shop.app.v1`、`shop.base.v1`、`shop.common.v1`、`shop.conf.v1` 这类历史/已废弃写法。
- Go 生成包 import 必须使用真实包名别名：`adminv1`、`appv1`、`basev1`、`commonv1`、`confv1`；生成 import path 必须带 `/v1`，禁止退回不带版本层级的历史路径。
- 前端 TS 生成类型与业务 import 必须带 `/v1/` 目录层级，例如 `@/rpc/admin/v1/auth`、`@/rpc/app/v1/auth`、`@/rpc/base/v1/login`、`@/rpc/common/v1/enum`；禁止新增 `@/rpc/admin/auth` 这类历史/已废弃路径。
- 修改或新增 proto 时，必须同步检查 `api/gen`、后端 service/biz、前端 RPC 类型与调用处、OpenAPI、`sql/default-data.sql` 中的接口权限数据是否需要更新。

## HTTP 接口命名规范
- 后续新增或修改 HTTP 接口路径时，必须遵循 RESTful API 协议，优先表达“资源”而不是“动作”。
- 路径统一使用小写英文与中划线或现有项目已使用的下划线/单词组合风格，不得出现驼峰命名、中文、空格。
- 路径中的资源名统一使用当前业务约定的常规名词；当前项目默认继续使用单数或不可数资源段，避免在路径中混用复数形式。
- 路径层级统一使用带版本号的前缀结构：`/api/v1/{terminal}/{module}/{resource}`。
- 迁移旧接口时，新的 `google.api.http` 主路径必须使用 `/api/v1/...`；为保证迁移过程中旧功能仍可访问，应通过 `additional_bindings` 暂时保留旧路径 `/api/...`。
- 标准 CRUD 场景必须优先通过 HTTP Method 表达动作语义，不要把动作写进 path。
- RESTful 判断以“资源建模”而不是“动词翻译”为准；业务动作必须优先抽象为会话、令牌、绑定关系、执行记录、确认单、支付单等资源或子资源。
- 资源唯一标识必须优先放在 path 中，筛选、分页、排序、统计维度等集合查询条件应优先放在 query 或请求参数中。
- 同一资源的查询、创建、更新、删除应尽量复用同一主路径，只通过 HTTP Method 区分动作。
- 状态切换、排序、导出、树结构、选项集、统计汇总等非标准 CRUD 场景，必须抽象成资源或子资源，禁止直接使用动词式路径。
- 同一业务若同时存在列表、汇总、图表、明细等接口，路径命名必须保持并列、稳定、可预测，例如 `/summary`、`/trend`、`/rank`、`/metric`、`/risk`、`/todo`。
- 批量操作若无法自然映射为单资源操作，优先使用资源集合语义或明确的子资源命名。
- `proto` 中的 RPC 方法名应与最终资源语义保持一致，避免保留历史动作式或歧义命名。
- HTTP 路径命名确定后，`proto` 中的 `google.api.http` 映射、`sql/default-data.sql` 中的 `base_api.path`、前端请求地址必须同步保持一致，禁止只改其中一处。
