# 错误处理最小化方案

本文基于当前工作区实现编写，结论时间为 2026-04-13。目标是重新设计一套最小必要的错误模型，不考虑历史兼容，只保留系统内真正需要稳定识别的错误类型。

## 设计目标

错误体系只解决两件事：

- 让前后端能稳定判断少量必要场景。
- 让用户看到合适的提示文案。

不解决的事：

- 不为每一句业务文案单独设计枚举。
- 不为每一种资源单独设计枚举。
- 不为验证码、token、密码、手机号、重复键等细项各自设计一套 reason。

结论是：系统只需要 6 类错误。

## 最小错误集合

建议 `error.proto` 只保留以下 6 个错误类型：

```proto
syntax = "proto3";

package common;

import "errors/errors.proto";

enum ErrorReason {
  option (errors.default_code) = 500;

  INVALID_ARGUMENT = 100 [(errors.code) = 400];
  UNAUTHENTICATED = 200 [(errors.code) = 401];
  PERMISSION_DENIED = 300 [(errors.code) = 403];
  RESOURCE_NOT_FOUND = 400 [(errors.code) = 404];
  CONFLICT = 500 [(errors.code) = 409];
  INTERNAL_ERROR = 600 [(errors.code) = 500];
}
```

这 6 类已经足够覆盖当前系统。

## 每类错误的使用边界

### 1. `INVALID_ARGUMENT`

表示“请求本身有问题，用户可以修改后重试”。

适用场景：

- 参数为空
- 参数格式错误
- 参数取值非法
- 验证码错误
- 验证码过期
- 手机号格式错误
- 两次密码不一致

当前系统中的典型文案：

- `验证码错误`
- `验证码已过期`
- `验证码不能为空`
- `手机号格式错误`
- `位置不能为空`
- `订单商品信息不能为空`
- `原密码不能为空`
- `新密码不能为空`
- `两次输入的密码不一致`

说明：

- 验证码错误、验证码过期、手机号格式错误都不需要单独枚举。
- 这类差异由 `message` 表达即可。

### 2. `UNAUTHENTICATED`

表示“认证失败，当前请求无法证明调用者身份”。

适用场景：

- 未登录
- token 无效
- token 过期
- 刷新令牌无效
- 登录时用户名或密码错误

当前系统中的典型文案：

- `用户认证失败`
- `用户名或密码错误`
- `刷新认证令牌失败`

说明：

- 不再拆分 `NOT_LOGGED_IN`、`TOKEN_EXPIRED`、`INCORRECT_REFRESH_TOKEN`、`INCORRECT_PASSWORD` 等独立枚举。
- 对前端来说，这些都属于认证失败。
- 登录场景中的“用户不存在”和“密码错误”统一收敛为 `用户名或密码错误`，避免账号探测。

### 3. `PERMISSION_DENIED`

表示“身份成立，但没有权限或当前主体不允许执行该操作”。

适用场景：

- 无权限访问
- 账号被禁用
- 角色被禁用
- 当前用户不允许操作超级管理员

当前系统中的典型文案：

- `用户权限不存在`
- `用户状态错误`
- `角色状态错误`
- `不能操作超级管理员`
- `不能操作超级管理员角色`

说明：

- “账号禁用”和“无权限”都可以先归入这一类。
- 只有当后续前端确实需要针对“禁用账号”做特殊流程时，才值得再细分。

### 4. `RESOURCE_NOT_FOUND`

表示“请求的目标资源不存在”。

适用场景：

- 按 ID 查用户不存在
- 按 ID 查订单不存在
- 按 ID 查角色不存在
- 按编号查任务不存在

当前系统中的典型文案：

- `用户不存在`
- `角色不存在`
- `部门不存在`
- `定时任务不存在`
- `调用目标不存在`

说明：

- 不再保留 `USER_NOT_FOUND`、`ROLE_NOT_FOUND` 这种按实体拆分的设计。
- 所有“资源不存在”统一用一个 reason。

### 5. `CONFLICT`

表示“请求本身合法，但和当前系统状态冲突，不能执行”。

适用场景：

- 唯一键重复
- 资源已存在
- 当前状态不允许操作
- 资源被保护
- 资源下还有子项，不能删除

当前系统中的典型文案：

- `用户账号重复`
- `角色编码重复`
- `配置key重复`
- `SKU编码重复`
- `商品属性重复`
- `商品规格重复`
- `订单已支付，不能重复支付`
- `订单已退款，不能重复退款`
- `订单已支付，无法取消`
- `删除部门失败,下面有部门`
- `删除菜单失败,下面有菜单`
- `删除字典失败,下面有属性`

说明：

- 不再区分“重复”、“状态冲突”、“受保护资源”、“存在子节点”多种 reason。
- 这些都属于冲突，只保留一个枚举。

### 6. `INTERNAL_ERROR`

表示“服务端内部异常，调用方无法通过修改请求解决”。

适用场景：

- 数据库错误
- Redis 错误
- RPC 错误
- 第三方服务错误
- 文件系统错误
- 内部数据不一致
- 未分类异常

当前系统中的典型文案：

- `登录失败`
- `支付失败`
- `查询支付失败`
- `发送验证码失败`
- `文件上传失败`
- `文件下载失败`
- `查询失败`
- 各类 `创建/更新/删除/查询 xxx 失败`

说明：

- 第三方服务错误不单独建枚举。
- 内部数据不一致也不单独建枚举。
- 先统一为 `INTERNAL_ERROR`，日志里保留详细 cause。

## 明确不保留的错误类型

以下类型不建议存在于新的 `error.proto` 中：

- `USER_NOT_FOUND`
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

原因只有一个：

- 这些类型对当前系统来说，不是必须稳定分支处理的信息。

如果确实要表达：

- 用 HTTP 状态码表达通用协议层语义。
- 用 `message` 表达具体用户提示。
- 用日志表达内部原因。

## `message` 的定位

最小错误模型下，`reason` 负责分类，`message` 负责具体提示。

例如：

- `INVALID_ARGUMENT` + `验证码错误`
- `INVALID_ARGUMENT` + `手机号格式错误`
- `UNAUTHENTICATED` + `用户名或密码错误`
- `PERMISSION_DENIED` + `账号已被禁用`
- `RESOURCE_NOT_FOUND` + `用户不存在`
- `CONFLICT` + `订单已支付，无法取消`
- `INTERNAL_ERROR` + `支付失败`

规则：

- 前端逻辑只依赖 `reason` 或 HTTP 状态码。
- 前端展示文案使用 `message`。
- 不允许前端根据中文 `message` 做逻辑分支。

## 分层职责

### `repo` / 第三方层

- 返回原始错误
- 不决定对外 reason

### `biz` 层

- 统一做错误分类
- 把错误归入上述 6 类之一
- 决定对外 `message`

### `service` 层

- 不记录错误日志
- 不再手写 `errors.New("xxx失败")`
- 不再做业务判断
- 只直接返回 `biz` 层错误

理想形态：

```go
func (s *LoginService) Login(ctx context.Context, req *base.LoginRequest) (*base.LoginResponse, error) {
    return s.loginCase.Login(ctx, req)
}
```

### `middleware` 层

- 统一记录 `code`
- 统一记录 `reason`
- 统一记录原始错误堆栈

## 当前系统的最小落地规则

如果后续开始改代码，统一按下面规则执行：

### 登录与认证

- 验证码错、验证码过期：`INVALID_ARGUMENT`
- 用户名或密码错误：`UNAUTHENTICATED`
- 未登录、token 无效、token 过期、刷新令牌无效：`UNAUTHENTICATED`
- 账号禁用、角色禁用：`PERMISSION_DENIED`
- 角色缺失、部门缺失、token 生成失败：`INTERNAL_ERROR`

### 后台和商城 CRUD

- 表单参数错误：`INVALID_ARGUMENT`
- 资源不存在：`RESOURCE_NOT_FOUND`
- 重复、状态冲突、禁止删除：`CONFLICT`
- 其余查询/保存失败：`INTERNAL_ERROR`

### 订单与支付

- 地址错误、订单商品为空：`INVALID_ARGUMENT`
- 未登录：`UNAUTHENTICATED`
- 无权限：`PERMISSION_DENIED`
- 订单不存在：`RESOURCE_NOT_FOUND`
- 重复支付、重复退款、不可取消：`CONFLICT`
- 渠道异常、内部异常：`INTERNAL_ERROR`

## 为什么只保留这 6 类

因为从当前系统真实需求看，机器层面只需要识别这几件事：

- 请求是不是有问题
- 用户是不是没通过认证
- 用户是不是没权限
- 资源是不是不存在
- 当前状态是不是冲突
- 是不是服务端内部异常

除此之外的细节：

- 是验证码错还是验证码过期
- 是 token 过期还是 refresh token 错
- 是账号重复还是 SKU 重复
- 是有子节点不能删还是超级管理员不能删

这些都不需要占用新的枚举，它们只需要体现在 `message` 里。

## 本文结论

新的错误体系不追求完整枚举所有业务句子，而是只保留最小必要分类：

- `INVALID_ARGUMENT`
- `UNAUTHENTICATED`
- `PERMISSION_DENIED`
- `RESOURCE_NOT_FOUND`
- `CONFLICT`
- `INTERNAL_ERROR`

后续如果没有明确、稳定、跨页面的分支处理需求，不再新增新的错误类型。先用好这 6 类，再谈扩展。
