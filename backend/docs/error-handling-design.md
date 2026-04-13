# 错误处理最小化与落地方案

本文基于当前工作区实现编写，结论时间为 2026-04-13。本文不是“枚举清单”，而是当前项目后续统一错误设计、错误分类和错误迁移的落地方案。

## 当前问题

当前项目的错误体系存在 3 个明显问题：

- `error.proto` 仍然保留了大量按句子、按场景、按资源拆分的旧枚举。
- 后端大量代码仍直接返回 `errors.New("中文文案")`，机器无法稳定识别错误类别。
- 前端仍有一部分逻辑依赖旧的 `reason` 或旧的错误编号，导致后端不敢收敛错误模型。

本方案的目标不是把每一句错误文案都枚举化，而是先统一出一套最小可用、长期稳定的错误模型，再逐步迁移当前实现。

## 设计目标

错误体系只解决 3 件事：

- 让前后端能稳定判断少量必要场景。
- 让用户看到合适的提示文案。
- 让日志里保留足够的排障信息。

错误体系不解决的事：

- 不为每一句中文提示单独设计 `reason`。
- 不为每一种资源单独设计 `reason`。
- 不为验证码、token、密码、手机号、唯一键、子节点删除限制等细项各自设计一套顶层枚举。

结论是：系统只需要 6 类顶层错误。

## 核心原则

### 1. `reason` 只负责“稳定分类”

`reason` 只能表达调用方需要稳定识别的一级语义，例如：

- 请求参数有问题
- 认证失败
- 权限不足
- 资源不存在
- 当前状态冲突
- 服务端内部异常

`reason` 不负责表达具体业务句子。

### 2. `message` 只负责“用户提示”

`message` 用来承载对用户友好的文案，例如：

- `验证码错误`
- `用户名或密码错误`
- `配置key重复`
- `删除菜单失败，存在子菜单`

前端可以展示 `message`，但不允许根据中文 `message` 做逻辑分支。

### 3. `metadata` 只负责“可选的二级结构化信息”

当同一个 `reason` 下，调用方确实需要更细但仍然稳定的机器可识别信息时，再使用 `metadata`。

例如：

- `CONFLICT` 下区分 `unique_violation`
- `CONFLICT` 下区分 `has_children`
- `CONFLICT` 下区分 `state_conflict`

`metadata` 是可选补充，不是新增顶层 `reason` 的替代枚举池。

### 4. `cause` 和日志负责“技术细节”

数据库错误、Redis 错误、第三方错误、底层堆栈、SQL 唯一索引名等技术细节，不应该全部暴露到顶层 `reason`，而应保留在：

- `cause`
- 结构化日志
- 链路日志

### 5. 新增错误类型的门槛要非常高

只有当以下条件同时成立时，才考虑新增顶层 `reason`：

- 现有 6 类无法准确表达该错误的一级语义。
- 调用方存在明确、稳定、跨页面或跨端的分支处理需求。
- 该需求不能用现有 `reason + metadata` 表达。
- 该分类具备长期稳定性，不是某个页面、某个资源、某句文案的临时差异。

只要上述条件缺少任意一项，就不要新增新的顶层 `reason`。

## 目标错误模型

建议新的 `error.proto` 最终只保留以下 6 个错误类型：

```proto
syntax = "proto3";

package common;

import "errors/errors.proto";

enum ErrorReason {
  option (errors.default_code) = 500;

  INVALID_ARGUMENT = 0 [(errors.code) = 400];
  UNAUTHENTICATED = 1 [(errors.code) = 401];
  PERMISSION_DENIED = 2 [(errors.code) = 403];
  RESOURCE_NOT_FOUND = 3 [(errors.code) = 404];
  CONFLICT = 4 [(errors.code) = 409];
  INTERNAL_ERROR = 5 [(errors.code) = 500];
}
```

说明：

- `errors.code` 对应对外 HTTP 语义。
- `proto3` 要求首个枚举值必须为 `0`，因此这里使用 `0..5` 的稳定编号。
- enum 数值只作为稳定编号，不等同于 HTTP 状态码。
- 一旦切换到新模型，顶层 `reason` 集合必须冻结，后续不要随意扩充。

## 6 类错误的边界

### 1. `INVALID_ARGUMENT`

含义：请求本身有问题，调用方修改请求后可以重试。

适用场景：

- 参数为空
- 参数格式错误
- 参数取值非法
- 验证码错误
- 验证码过期
- 手机号格式错误
- 两次密码不一致
- 地址错误
- 订单商品信息为空

典型文案：

- `验证码错误`
- `验证码已过期`
- `手机号格式错误`
- `位置不能为空`
- `订单商品信息不能为空`
- `两次输入的密码不一致`

### 2. `UNAUTHENTICATED`

含义：调用方无法证明自己的身份，或者登录身份校验失败。

适用场景：

- 未登录
- token 无效
- token 过期
- 刷新令牌无效
- 登录时用户名或密码错误

典型文案：

- `用户认证失败`
- `用户名或密码错误`
- `刷新认证令牌失败`

补充规则：

- 登录场景中的“用户不存在”和“密码错误”统一收敛为 `用户名或密码错误`，避免账号探测。
- 不再拆分 `TOKEN_EXPIRED`、`INCORRECT_REFRESH_TOKEN`、`INCORRECT_PASSWORD` 等旧枚举。

### 3. `PERMISSION_DENIED`

含义：身份成立，但没有权限，或者当前主体不允许执行该操作。

适用场景：

- 无权限访问
- 账号被禁用
- 角色被禁用
- 不允许操作超级管理员
- 不允许操作超级管理员角色

典型文案：

- `用户权限不存在`
- `用户状态错误`
- `角色状态错误`
- `不能操作超级管理员`
- `不能操作超级管理员角色`

补充规则：

- “账号禁用”和“无权限”在一级语义上都属于权限拒绝。
- “不能操作超级管理员”属于权限或主体限制，不属于状态冲突。

### 4. `RESOURCE_NOT_FOUND`

含义：请求目标资源不存在。

适用场景：

- 用户不存在
- 角色不存在
- 部门不存在
- 订单不存在
- 定时任务不存在
- 调用目标不存在

典型文案：

- `用户不存在`
- `角色不存在`
- `部门不存在`
- `定时任务不存在`
- `调用目标不存在`

补充规则：

- 不再保留 `USER_NOT_FOUND`、`ROLE_NOT_FOUND` 这种按实体拆分的顶层枚举。
- “登录失败时查不到用户”属于认证语义，应归入 `UNAUTHENTICATED`，而不是 `RESOURCE_NOT_FOUND`。

### 5. `CONFLICT`

含义：请求本身合法，但与当前系统状态冲突，因此不能执行。

适用场景：

- 唯一键重复
- 资源已存在
- 当前状态不允许操作
- 资源被保护
- 资源下仍有子项，不能删除

典型文案：

- `用户账号重复`
- `角色编码重复`
- `配置key重复`
- `SKU编码重复`
- `商品属性重复`
- `商品规格重复`
- `订单已支付，不能重复支付`
- `订单已退款，不能重复退款`
- `订单已支付，无法取消`
- `删除部门失败，下面有部门`
- `删除菜单失败，下面有菜单`
- `删除字典失败，下面有属性`

补充规则：

- 唯一约束冲突归 `CONFLICT`，不要单独新建 `UNIQUE_VIOLATION` 顶层类型。
- “父节点下还有子资源，不能删除父节点”归 `CONFLICT`，不要单独新建 `HAS_CHILDREN` 顶层类型。
- “资源被保护，禁止删除”归 `CONFLICT`；若本质是操作主体权限问题，则应改归 `PERMISSION_DENIED`。

### 6. `INTERNAL_ERROR`

含义：服务端内部异常，调用方无法通过修改请求解决。

适用场景：

- 数据库错误
- Redis 错误
- RPC 错误
- 第三方服务错误
- 文件系统错误
- 内部数据不一致
- 未分类异常

典型文案：

- `登录失败`
- `支付失败`
- `查询支付失败`
- `发送验证码失败`
- `文件上传失败`
- `文件下载失败`
- `查询失败`
- 各类 `创建/更新/删除/查询 xxx 失败`

补充规则：

- 第三方服务错误不单独建顶层枚举。
- 用户已存在但角色、部门等关联数据缺失，属于内部数据不一致，应归 `INTERNAL_ERROR`。

## 重点问题：唯一约束错误和“存在子节点不能删除”怎么归类

这两个问题都不需要新增新的顶层错误类型。

### 唯一约束错误

统一归类为：

- `reason = CONFLICT`
- `code = 409`
- `message = 具体重复提示`

示例：

- `CONFLICT` + `配置key重复`
- `CONFLICT` + `角色编码重复`
- `CONFLICT` + `商品属性重复`

### 父节点下仍有资源，不能删除父节点

统一归类为：

- `reason = CONFLICT`
- `code = 409`
- `message = 具体禁止删除提示`

示例：

- `CONFLICT` + `删除菜单失败，下面有菜单`
- `CONFLICT` + `删除部门失败，下面有部门`
- `CONFLICT` + `删除字典失败，下面有属性`

原因很简单：

- 这两类错误的一级语义都是“当前状态冲突，导致操作不能执行”。
- 它们的差异是二级业务语义，不足以支撑新增顶层 `reason`。

## `metadata` 的推荐用法

默认情况下，只用 `reason + message` 即可。

只有当调用方确实需要在同一个 `reason` 下做二级稳定分支时，才增加 `metadata`。推荐优先在 `CONFLICT` 下使用有限、可控的结构化键。

推荐约定：

- `conflict_type`
- `resource`
- `field`
- `constraint`
- `child_resource`
- `current_state`
- `expected_state`

推荐值：

- `unique_violation`
- `has_children`
- `state_conflict`
- `protected_resource`

示例 1：唯一键冲突

```json
{
  "reason": "CONFLICT",
  "message": "配置key重复",
  "metadata": {
    "conflict_type": "unique_violation",
    "resource": "base_config",
    "field": "key",
    "constraint": "unique_base_config"
  }
}
```

示例 2：存在子节点不能删除

```json
{
  "reason": "CONFLICT",
  "message": "删除菜单失败，下面有菜单",
  "metadata": {
    "conflict_type": "has_children",
    "resource": "base_menu",
    "child_resource": "base_menu"
  }
}
```

使用边界：

- 不要把整句中文提示再复制一份到 `metadata`。
- 不要无限扩展 `metadata` 键，避免形成“第二套枚举系统”。
- 如果没有明确调用方依赖，宁可不加 `metadata`。

## 前端展示规范

前端展示错误时，统一遵循以下顺序：

1. 优先看 HTTP 状态码。
2. 再看顶层 `reason`。
3. 最后按需读取 `metadata`。

同时遵循以下原则：

- 用户默认只看 `message`。
- 前端逻辑只依赖 `code`、`reason`、`metadata`。
- 不允许前端根据中文 `message` 做业务分支。
- 不允许把 `metadata` 原样直接展示给用户。
- 不允许使用 `200` 包裹业务错误。

### `400 INVALID_ARGUMENT`

含义：

- 请求参数本身有问题。
- 用户修改输入后通常可以立即重试。

前端默认行为：

- 直接提示 `message`。
- 若当前是表单页，可优先把错误聚焦到对应字段。

推荐交互：

- toast / message：展示 `message`
- 表单校验：若 `metadata.field` 存在，则将对应字段标红
- 可选：若 `metadata.expected_format` 存在，可用于补充表单提示，但不直接暴露底层格式规则

示例：

```json
{
  "code": 400,
  "reason": "INVALID_ARGUMENT",
  "message": "配置key不能为空",
  "metadata": {
    "field": "key"
  }
}
```

推荐展示：

- 默认提示：`配置key不能为空`
- 若当前是配置表单页：将 `key` 输入框标红并显示同样的校验文案

### `401 UNAUTHENTICATED`

含义：

- 当前请求没有通过认证，或者登录身份已失效。

前端默认行为：

- 清理登录态。
- 跳转登录页或弹出重新登录提示。

推荐交互：

- 不要停留在当前页反复 toast
- 统一进入重新登录流程

### `403 PERMISSION_DENIED`

含义：

- 用户已登录，但没有权限执行当前操作。

前端默认行为：

- 展示 `message`
- 保持当前页面，不自动退出登录

推荐交互：

- 按钮操作失败时：toast 或 message 提示无权限
- 页面初始化失败时：可展示无权限空态页

### `404 RESOURCE_NOT_FOUND`

含义：

- 目标资源不存在，或者已被删除。

前端默认行为：

- 展示 `message`
- 按页面语义决定是否返回列表页或关闭详情页

推荐交互：

- 列表里的详情页：提示后返回列表
- 独立详情页：提示后跳转到上一级或空态页

### `409 CONFLICT`

含义：

- 请求格式正确，但和当前系统状态冲突，操作无法完成。

前端默认行为：

- 直接提示 `message`
- 若当前页面需要更强交互，再按 `metadata` 做增强

推荐交互：

- 唯一键冲突：显示 `message`，必要时将对应字段标红
- 存在子节点不能删除：显示 `message`，必要时补充“请先删除子节点”的引导
- 状态冲突：显示 `message`，必要时刷新当前数据

示例 1：唯一键冲突

```json
{
  "code": 409,
  "reason": "CONFLICT",
  "message": "配置key重复",
  "metadata": {
    "conflict_type": "unique_violation",
    "resource": "base_config",
    "field": "key",
    "constraint": "unique_base_config"
  }
}
```

推荐展示：

- 默认提示：`配置key重复`
- 若当前是配置表单页：将 `key` 输入框标红

示例 2：存在子节点不能删除

```json
{
  "code": 409,
  "reason": "CONFLICT",
  "message": "删除菜单失败，下面有菜单",
  "metadata": {
    "conflict_type": "has_children",
    "resource": "base_menu",
    "child_resource": "base_menu"
  }
}
```

推荐展示：

- 默认提示：`删除菜单失败，下面有菜单`
- 若当前是删除确认场景：可追加提示“请先删除子菜单后再重试”

### `500 INTERNAL_ERROR`

含义：

- 服务端内部异常，用户通常无法自行修复。

前端默认行为：

- 展示通用失败提示，或展示后端已收敛好的用户提示
- 不对 `metadata` 做业务分支

推荐交互：

- 优先展示后端返回的用户友好 `message`
- 若 `message` 为空或明显属于技术细节，则回退为统一文案，如“系统出错，请稍后再试”

## 前端实现建议

前端统一按下面的优先级处理：

- `400`：提示 `message`，表单页可结合 `metadata.field` 做字段高亮
- `401`：统一重新登录
- `403`：提示无权限，不清理登录态
- `404`：提示资源不存在，必要时返回列表页或上一级
- `409`：提示 `message`，必要时按 `metadata.conflict_type` 做增强交互
- `500`：提示通用失败或后端已收敛后的用户提示

特别说明：

- `400` 和 `409` 都可以展示 `message`，但语义不同。
- `400` 表示“请求填错了”，适合表单纠正。
- `409` 表示“请求没填错，但当前状态不允许”，适合冲突提示和操作引导。
- 因此 `配置key重复` 应归 `409`，不是 `400`。
- `删除菜单失败，下面有菜单` 也应归 `409`，不是 `400`。

## 分类决策规则

后续判断错误归类时，优先按以下顺序思考：

1. 调用方修改请求后能否立刻解决？
2. 如果能解决，是“参数错”还是“认证/权限问题”？
3. 如果请求没问题，资源是否不存在？
4. 如果资源存在，是否只是因为当前状态冲突而不能执行？
5. 如果都不是，默认归入内部异常。

可以进一步收敛成以下判断：

- 参数不合法：`INVALID_ARGUMENT`
- 身份不成立：`UNAUTHENTICATED`
- 身份成立但不允许：`PERMISSION_DENIED`
- 目标不存在：`RESOURCE_NOT_FOUND`
- 目标存在但当前状态冲突：`CONFLICT`
- 其余异常：`INTERNAL_ERROR`

## 分层职责

### `repo` / 第三方层

- 返回原始错误
- 不决定对外 `reason`
- 不直接生成面向前端的错误文案

### `biz` 层

- 统一做错误分类
- 把错误归入 6 类之一
- 决定对外 `message`
- 在确有需要时补充 `metadata`
- 保留原始 `cause`

### `service` 层

- 不做业务判断
- 统一记录一条原始错误定位日志
- 新链路优先直接返回 `biz` 层已分类错误
- 老链路过渡期允许统一使用 `errorsx.WrapInternal(err, "xxx失败")` 做兜底包装
- 不再手写裸 `errors.New("xxx失败")`

推荐形态：

```go
func (s *LoginService) Login(ctx context.Context, req *base.LoginRequest) (*base.LoginResponse, error) {
    res, err := s.loginCase.Login(ctx, req)
    if err != nil {
        log.Errorf("Login %v", err)
        return nil, errorsx.WrapInternal(err, "登录失败")
    }
    return res, nil
}
```

补充说明：

- `errorsx.WrapInternal` 遇到已经完成分类的 Kratos 错误会直接透传，不会覆盖已有 `reason`、`message`、`metadata`。
- 因此它适合作为 `service` 层统一兜底，避免历史 `biz` 代码把数据库、Redis、第三方原始错误直接漏给前端。
- `service` 层日志统一使用 `log.Errorf("方法名 %v", err)` 这种形式，避免再写出 `log.Error("xx err:", err.Error())` 导致的 `err:error: ...` 冗余前缀。
- 请求级错误日志仍由 `middleware` 记录，因此线上会保留两类日志：
- 一条 `service` 层原始错误日志，用于看具体哪个 Go 方法报错。
- 一条 `middleware` 请求日志，用于看请求入参、状态码、链路耗时和最终返回。

### `middleware` 层

- 统一记录 `code`
- 统一记录 `reason`
- 统一记录 `message`
- 统一记录 `metadata`
- 统一记录原始堆栈和 `cause`

## 当前系统的具体分类建议

### 登录与认证

- 验证码错误、验证码过期：`INVALID_ARGUMENT`
- 用户名或密码错误：`UNAUTHENTICATED`
- 未登录、token 无效、token 过期、刷新令牌无效：`UNAUTHENTICATED`
- 账号禁用、角色禁用：`PERMISSION_DENIED`
- 用户存在但角色缺失、部门缺失、token 生成失败：`INTERNAL_ERROR`

### 后台和商城 CRUD

- 表单参数错误：`INVALID_ARGUMENT`
- 资源不存在：`RESOURCE_NOT_FOUND`
- 唯一键冲突、状态冲突、受保护资源、存在子节点不能删除：`CONFLICT`
- 其余查询/保存失败：`INTERNAL_ERROR`

### 订单与支付

- 地址错误、订单商品为空：`INVALID_ARGUMENT`
- 未登录：`UNAUTHENTICATED`
- 无权限：`PERMISSION_DENIED`
- 订单不存在：`RESOURCE_NOT_FOUND`
- 重复支付、重复退款、不可取消：`CONFLICT`
- 支付渠道异常、内部异常：`INTERNAL_ERROR`

## 明确不建议保留的顶层错误类型

以下类型不建议继续出现在新的 `error.proto` 中：

- `USER_NOT_FOUND`
- `ROLE_NOT_FOUND`
- `INVALID_USERID`
- `INVALID_TOKEN`
- `INVALID_PASSWORD`
- `INCORRECT_PASSWORD`
- `INCORRECT_ACCESS_TOKEN`
- `INCORRECT_REFRESH_TOKEN`
- `TOKEN_EXPIRED`
- `TOKEN_NOT_EXIST`
- `USER_FREEZE`
- `BAD_REQUEST`
- `NETWORK_ERROR`
- `SERVICE_UNAVAILABLE`
- `NETWORK_TIMEOUT`
- `REQUEST_TIMEOUT`
- `METHOD_NOT_ALLOWED`
- `NOT_IMPLEMENTED`
- `UNIQUE_VIOLATION`
- `HAS_CHILDREN`
- `STATE_CONFLICT`

原因只有一个：

- 这些类型对当前系统来说，不是必须稳定暴露给调用方的一级语义。

如果确实要表达其中的差异：

- 用 HTTP 状态码表达通用协议层语义。
- 用 `message` 表达具体用户提示。
- 用 `metadata` 表达必要的二级结构化信息。
- 用日志表达内部原因。

## 迁移方案

本方案的目标态是“只有 6 类顶层错误”，但当前仓库仍存在旧枚举、旧前端依赖和大量直接返回文本错误的实现，因此建议按以下顺序迁移。

### 第 1 步：冻结新增旧错误类型

- 从本文生效开始，不再新增旧式细粒度顶层 `reason`。
- 新增业务统一按 6 类模型分类。

### 第 2 步：先改前端判断逻辑

- 前端统一改为优先依赖 HTTP `401/403` 或新的 `UNAUTHENTICATED`、`PERMISSION_DENIED`。
- 去掉对 `TOKEN_EXPIRED`、`NOT_LOGGED_IN`、`INCORRECT_REFRESH_TOKEN` 等旧值的硬编码依赖。

### 第 3 步：收敛后端分层职责

- `biz` 层负责分类。
- `service` 层只透传。
- 清理 `service` 层的 `errors.New("xxx失败")` 和重复日志。

### 第 4 步：更新 `error.proto` 并重新生成

- 把顶层枚举收敛到 6 类。
- 同步更新 `api/gen` 与前端 `src/rpc` 生成产物。

### 第 5 步：按业务域逐步替换老错误

- 优先替换登录鉴权链路。
- 再替换后台 CRUD。
- 最后替换订单、支付、推荐等复杂链路。

### 第 6 步：收尾移除兼容代码

- 当前后端和两个前端都完成切换后，再移除旧 `reason` 兼容分支。

## 推荐的落地代码形态

建议后续统一返回 Kratos 标准错误，并在需要时带上 `metadata` 与 `cause`。

示例：唯一键冲突

```go
return errors.New(
    409,
    common.ErrorReason_CONFLICT.String(),
    "配置key重复",
).WithMetadata(map[string]string{
    "conflict_type": "unique_violation",
    "resource": "base_config",
    "field": "key",
    "constraint": "unique_base_config",
}).WithCause(err)
```

示例：存在子节点不能删除

```go
return errors.New(
    409,
    common.ErrorReason_CONFLICT.String(),
    "删除菜单失败，下面有菜单",
).WithMetadata(map[string]string{
    "conflict_type": "has_children",
    "resource": "base_menu",
    "child_resource": "base_menu",
})
```

## 本文结论

新的错误体系不追求完整枚举所有业务句子，而是只保留最小必要分类：

- `INVALID_ARGUMENT`
- `UNAUTHENTICATED`
- `PERMISSION_DENIED`
- `RESOURCE_NOT_FOUND`
- `CONFLICT`
- `INTERNAL_ERROR`

对于“唯一约束冲突”和“存在子节点不能删除父节点”这两类问题：

- 统一归入 `CONFLICT`
- 不新增新的顶层错误类型
- 需要进一步区分时，使用 `metadata`

后续如果没有明确、稳定、跨端的分支处理需求，不再新增新的顶层 `reason`。先把这 6 类真正落到实现里，再谈扩展。
