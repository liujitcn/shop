# 业务域目录拆分设计（当前实现）

## 状态

本文档记录已完成的接口与服务分域。仓库不再使用端优先的 `api/protos/admin`、`api/protos/app` 或平铺的 `service/admin`、`service/app` 结构；当前所有新代码都必须落到下列业务域。

## 目录结构

```text
backend
├── api
│   ├── proto
│   │   ├── base/v1            # 登录、文件、OAuth、SSE、AI、MCP 等公共能力
│   │   ├── common/v1          # 错误、通用类型和跨域基础枚举
│   │   ├── system
│   │   │   ├── common/v1      # 系统域枚举
│   │   │   ├── admin/v1       # 系统后台接口
│   │   │   └── app/v1         # 系统商城端接口
│   │   └── shop
│   │       ├── common/v1      # 商城域枚举和共享消息
│   │       ├── config/v1      # 商城配置模型
│   │       ├── admin/v1       # 商城运营后台接口
│   │       └── app/v1         # 商城端接口
│   └── gen/go                 # 与 proto 目录同构的生成结果
├── service
│   ├── base
│   ├── system/admin
│   ├── system/app
│   ├── shop/admin
│   └── shop/app
└── server
    ├── base
    ├── system/admin
    ├── system/app
    ├── shop/admin
    └── shop/app
```

前端的手写 API、生成 RPC 和业务页面也按相同边界组织：

```text
frontend/admin/src/{api,rpc,views}/
  base/
  system/admin/
  shop/admin/

frontend/app/src/{api,rpc}/
  base/
  system/app/
  shop/app/
```

## 域归属

| 域 | 范围 |
| --- | --- |
| `base` | 登录、OAuth、文件、SSE、MCP、AI 会话与消息等跨终端基础能力。 |
| `common` | 错误、分页、基础类型和无业务归属的公共枚举。 |
| `system` | 认证、用户、角色、部门、菜单、字典、配置、日志、任务、租户和代码生成。 |
| `shop` | 商品、门店、订单、支付、评价、推荐、工作台、统计、报表和商城运营配置。 |

`system` 不能依赖商城 Proto 或商城业务包。`shop` 可以使用 `base`、`common` 与 `system/common` 提供的类型。若一个能力只服务于商城交易或经营，不应放入 `system`，即使它同时被后台和商城端消费。

## 命名和生成约束

| 位置 | 当前约定 |
| --- | --- |
| Proto package | `system.admin.v1`、`shop.admin.v1`、`system.app.v1`、`shop.app.v1`，公共包为 `base.v1`、`common.v1`。 |
| Go import alias | `systemadminv1`、`shopadminv1`、`systemappv1`、`shopappv1`、`basev1`、`commonv1`。 |
| 管理后台 RPC | `src/rpc/system/admin/v1`、`src/rpc/shop/admin/v1`。 |
| 商城端 RPC | `src/rpc/system/app/v1`、`src/rpc/shop/app/v1`。 |
| 手写前端 API | `src/api/{base,system,shop}/...`，目录与对应 Proto 包一致。 |

接口变更在 `backend` 目录通过 `make api`、`make openapi`、`make ts` 生成。`api/gen/go`、前端 `src/rpc`、OpenAPI 均不可手工改写。

## 注册与菜单连带

新增服务或移动域时，除了修改 Proto 与 `service`，还要同步检查：

1. `server/{system,shop}/{admin,app}/register.go` 的 HTTP、gRPC 和工具注册。
2. `internal/cmd/server` 的 Wire 装配是否包含新依赖。
3. `backend/api/buf*.yaml` 的生成输入和 TypeScript 端过滤规则。
4. 管理后台 `src/api`、`src/views`、动态菜单组件路径和 `sql/default-data.sql` 中的菜单/API 权限数据。
5. 代码生成器的目标域和输出路径。当前代码生成器可识别 `system/admin/v1` 与 `shop/admin/v1`。

HTTP 路径以 Proto 的显式 `google.api.http` 为准。迁移目录不等于可以任意改变已有 HTTP 路径、OpenAPI operation 或权限数据；这些对外契约必须同步评估。

## 历史迁移说明

旧的 `api/protos`、`admin.v1`、`app.v1` 与平铺服务目录是迁移前结构，仅用于理解历史提交，不得在新增代码或文档中继续引用。若后续拆分新的可选业务域，应复用“域 common + Admin/App 终端接口 + service/server 同构注册 + 前端 API/RPC 对齐”的模式，而不是重新引入端优先目录。
