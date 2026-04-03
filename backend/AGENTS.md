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
