# Codex 规则

## 适用范围
- 本规则适用于 `shop` 仓库全量目录。

## 提交与文档约定
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

## 注释规范
- 后续新增或修改代码时，代码注释统一使用中文。
- 后续新增或修改代码时，必须为每个新增或修改的方法补充中文方法注释。
- 后续新增或修改代码时，必须补充必要的中文行内注释（关键逻辑、边界条件、异常分支需明确说明）。

## 代码修改约束
- 后续新增或修改代码时，若当前方法内某个变量名已在上文声明，后续涉及该变量的多值赋值禁止继续使用 `:=` 混合短声明。
- 遇到上述场景时，必须先显式 `var` 定义新的变量，再使用 `=` 赋值，避免因重复声明触发 IDE 或 lint 警告。
- 后续新增或修改代码时，禁止使用 `var (...)` 形式成组声明一批局部变量后再集中赋值。
- 每个局部变量必须在首次使用前一行就近声明，避免在方法开头堆积无关变量定义。
- 方法内第一次出现 `err` 时，必须结合实际调用使用 `:=` 获取；后续复用同一个 `err` 时，只能使用 `=` 赋值。

## 数据库索引命名规则
- 唯一索引命名格式：`unique_表名`
  - 示例：`unique_order`、`unique_goods`
  - 参考文件：`pkg/gen/models/order.gen.go`
- 普通索引命名格式：`idx_表名_字段1_字段2_...`
  - 单字段索引：`idx_表名_字段名`
    - 示例：`idx_order_status`、`idx_goods_category_id`
  - 联合索引：`idx_表名_字段1_字段2`
    - 示例：`idx_order_user_id_created_at`、`idx_goods_category_created_at`
- 在 GORM 模型定义中添加索引时，必须严格遵守此命名规范
  - 唯一索引：`gorm:"column:字段名;type:字段类型;uniqueIndex:unique_表名,priority:N;comment:注释"`
  - 普通索引：`gorm:"column:字段名;type:字段类型;index:idx_表名_字段1_字段2,priority:N;comment:注释"`

## 数据库命名规则
- **表命名格式**：全部小写，单词间用下划线分隔
  - 格式：`aa_bb_cc`
  - 示例：`order`、`order_goods`、`base_user`、`goods_category`
  - 禁止：`Order`、`OrderGoods`、`BaseUser`
- **字段命名格式**：全部小写，单词间用下划线分隔
  - 格式：`aa_bb_cc`
  - 示例：`user_id`、`order_no`、`created_at`、`category_id`
  - 禁止：`userId`、`orderNo`、`createdAt`、`categoryId`
- **命名原则**：
  - 使用有意义的英文单词，避免拼音
  - 优先使用名词，避免动词
  - 保持一致性，同一概念使用相同命名
  - 参考文件：`pkg/gen/models/order.gen.go`、`pkg/gen/models/base_user.gen.go`

## 变量命名规则
- **Go 变量命名格式**：首字母小写的驼峰命名法（小驼峰）
  - 格式：`aaBbCc`
  - 示例：`userId`、`userName`、`orderId`、`categoryList`
  - 禁止：`user_id`、`user_name`、`order_id`、`category_list`
- **命名原则**：
  - 变量名必须见名知意，避免无意义缩写
  - 布尔变量以 `Is`、`Has`、`Can`、`Should` 等开头
    - 示例：`isActive`、`hasPermission`、`canDelete`、`shouldUpdate`
  - 常量使用全大写，单词间用下划线分隔
    - 示例：`MAX_SIZE`、`DEFAULT_TIMEOUT`、`API_VERSION`
  - 缩写词全大写时需保持一致性
    - 推荐：`userID`、`htmlContent`、`xmlParser`
  - 循环变量可使用简短名称（`i`、`j`、`k`），但需有明确上下文
- **方法命名格式**：首字母大写的驼峰命名法（大驼峰，用于公开方法）或首字母小写（用于私有方法）
  - 公开方法：`GetUser()`、`CreateOrder()`、`UpdateStatus()`
  - 私有方法：`getUser()`、`createOrder()`、`updateStatus()`
- **结构体命名格式**：首字母大写的驼峰命名法（大驼峰）
  - 示例：`User`、`Order`、`Goods`、`AnalyticsCase`

## Proto接口契约命名规则
- 标准业务默认包含以下方法：
  - `List业务名`
  - `Page业务名`
  - `Get业务名`
  - `Create业务名`
  - `Update业务名`
  - `Delete业务名`
  - `Set业务名Status`
- 上述 7 个方法视为标准业务方法，参数和返回值必须严格按照以下固定结构：
  - `List业务名`：入参 `google.protobuf.Empty`，返回 `List业务名Response`
  - `Page业务名`：入参 `Page业务名Request`，返回 `Page业务名Response`
  - `Get业务名`：入参 `google.protobuf.Int64Value`，返回 `业务名Form`
  - `Create业务名`：入参 `业务名Form`，返回 `google.protobuf.Empty`
  - `Update业务名`：入参 `业务名Form`，返回 `google.protobuf.Empty`
  - `Delete业务名`：入参 `google.protobuf.StringValue`，返回 `google.protobuf.Empty`
  - `Set业务名Status`：入参 `common.SetStatusRequest`，返回 `google.protobuf.Empty`
- 标准业务方法禁止改成 `RPC方法名Request`、`RPC方法名Response` 这一套。
- 标准业务以外的方法，统一遵循通用规则：
  - `RPC方法名`
  - `RPC方法名Request`
  - `RPC方法名Response`
- 标准以外的方法包括但不限于：
  - `TreeBaseDept`
  - `TreeBaseDeptRequest`
  - `TreeBaseDeptResponse`
  - `OptionBaseDept`
  - `OptionBaseDeptRequest`
  - `OptionBaseDeptResponse`
- 例如：
  - `OrderMonthReportSummary` / `OrderMonthReportSummaryRequest` / `OrderMonthReportSummaryResponse`
  - `OrderMonthReportList` / `OrderMonthReportListRequest` / `OrderMonthReportListResponse`
- RPC 方法名、接口入参、返回值消息名主体必须与当前业务语义保持一致。
- 禁止命名：
  - `GetOrderRequest`
  - `CreateGoodsResponse`
  - `UpdateUserReply`
- 推荐命名：
  - `OrderDetail`
  - `OrderDetailRequest`
  - `OrderDetailResponse`
  - `OrderList`
  - `OrderListRequest`
  - `OrderListResponse`
- 若同一业务同时存在汇总、列表、详情等结构，优先使用业务名 + 结果形态命名：
  - `OrderMonthReportSummary`
  - `OrderMonthReportSummaryRequest`
  - `OrderMonthReportSummaryResponse`
  - `OrderMonthReportList`
  - `OrderMonthReportListRequest`
  - `OrderMonthReportListResponse`
- 禁止 RPC 方法名或消息名出现 `Get`、`Create`、`Update`、`Delete` 等动词前缀。

## HTTP接口命名规范
- 后续新增或修改 HTTP 接口路径时，必须遵循 RESTful API 协议，优先表达“资源”而不是“动作”。
- 路径统一使用小写英文与中划线或现有项目已使用的下划线/单词组合风格，不得出现驼峰命名、中文、空格。
- 路径中的资源名统一使用当前业务约定的常规名词，不使用英文复数形式。
  - 推荐：`/api/admin/goods`、`/api/admin/order`、`/api/admin/base/dict`
  - 禁止：`/api/admin/goodses`、`/api/admin/orders`、`/api/admin/dicts`
- 上述“不使用英文复数形式”同时适用于主资源和子资源，禁止出现 `/metrics`、`/todos`、`/risks`、`/menus`、`/orders` 这类复数片段。
  - 推荐：`/api/admin/workspace/metric`、`/api/admin/workspace/todo`、`/api/admin/workspace/risk`
  - 禁止：`/api/admin/workspace/metrics`、`/api/admin/workspace/todos`、`/api/admin/workspace/risks`
- 路径层级统一延续当前项目前缀结构：`/api/{terminal}/{module}/{resource}`。
  - `terminal` 取值示例：`admin`、`app`、`base`
  - `module`、`resource` 必须使用有业务语义的名词或名词短语，禁止使用 `get`、`create`、`update`、`delete`、`list` 等动词作为路径片段
- 标准 CRUD 场景必须优先通过 HTTP Method 表达动作语义，不要把动作写进 path：
  - 查询列表：`GET /api/admin/base/menu`
  - 查询详情：`GET /api/admin/base/menu/{id}`
  - 新增资源：`POST /api/admin/base/menu`
  - 更新资源：`PUT /api/admin/base/menu/{id}`
  - 删除资源：`DELETE /api/admin/base/menu/{id}`
- 状态切换、排序、导出、树结构、选项集、统计汇总等非标准 CRUD 场景，必须抽象成资源或子资源，禁止直接使用动词式路径：
  - 推荐：`PUT /api/admin/base/menu/{id}/status`
  - 推荐：`GET /api/admin/base/menu/tree`
  - 推荐：`GET /api/admin/base/menu/option`
  - 推荐：`GET /api/admin/workspace/metric`
  - 禁止：`POST /api/admin/base/menu/updateStatus`
  - 禁止：`GET /api/admin/workspace/getMetrics`
- 同一业务若同时存在列表、汇总、图表、明细等接口，路径命名必须保持并列、稳定、可预测：
  - 示例：`/summary`、`/trend`、`/rank`、`/metric`、`/risk`、`/todo`
- 批量操作若无法自然映射为单资源操作，优先使用资源集合语义或明确的子资源命名：
  - 推荐：`DELETE /api/admin/goods`
  - 推荐：`PUT /api/admin/base/role/{id}/menu`
  - 禁止：`POST /api/admin/goods/batchDelete`
- HTTP 路径命名确定后，`proto` 中的 `google.api.http` 映射、`sql/default-data.sql` 中的 `base_api.path`、前端请求地址必须同步保持一致，禁止只改其中一处。
