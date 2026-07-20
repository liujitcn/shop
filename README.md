# shop

`shop` 是一个前后端分离的商城项目，仓库内按模块维护 Go 后端、Vue 管理后台、uni-app 商城端、Gorse 推荐联调配置、初始化 SQL 与设计文档。

根 README 只保留仓库入口说明。各模块的启动、配置、构建、生成与校验细节请进入对应模块文档查看；详细设计统一沉淀在根目录 `docs` 下。

## 模块文档

| 模块 | 文档 | 说明 |
| --- | --- | --- |
| 后端服务 | [backend/README.md](backend/README.md) | Go + Kratos 服务、配置、启动、接口生成、构建、静态资源挂载。 |
| 管理后台 | [frontend/admin/README.md](frontend/admin/README.md) | Vue 3 管理后台、环境变量、页面结构、开发与构建命令。 |
| 商城端 | [frontend/app/README.md](frontend/app/README.md) | uni-app 商城端、H5 / 微信小程序运行、分包结构、构建与校验。 |
| Gorse 推荐 | [gorse/README.md](gorse/README.md) | Gorse 本地推荐服务、端口、数据库、后端联调配置。 |

## 设计文档

| 分类 | 文档 |
| --- | --- |
| 总体设计 | [系统总体设计](docs/系统总体设计.md) |
| 模块设计 | [后端服务设计](docs/后端服务设计.md)、[管理后台设计](docs/管理后台设计.md)、[商城端设计](docs/商城端设计.md)、[数据库与初始化数据设计](docs/数据库与初始化数据设计.md)、[租户与门店体系设计](docs/租户与门店体系设计.md) |
| 订单链路 | [订单数据流转设计](docs/订单数据流转设计.md) |
| 推荐链路 | [推荐系统设计](docs/推荐系统设计.md)、[推荐数据流转设计](docs/推荐数据流转设计.md) |
| 统计报表 | [统计数据流转设计](docs/统计数据流转设计.md) |
| 评价审核 | [评价与审核数据流转设计](docs/评价与审核数据流转设计.md) |
| 智能与契约 | [AI 助手设计](docs/AI助手设计.md)、[业务域目录拆分设计](docs/Proto目录拆分设计.md) |

## 仓库结构

```text
.
├── backend          # Go + Kratos 后端服务
├── frontend
│   ├── admin       # Vue 3 管理后台
│   ├── app         # uni-app 商城端
│   └── Makefile    # 前端聚合命令
├── gorse           # 本地 Gorse 推荐服务配置
├── sql             # 初始化与演示数据 SQL
└── docs            # 仓库级业务设计文档
```

## 本地启动入口

推荐按以下顺序启动，具体命令见模块文档：

1. 创建 `shop_test` 数据库，首次启动后端完成当前模型的自动迁移，然后停止后端。
2. 在仓库根目录依次导入 `sql/default-data.sql`、`sql/base_area.sql`；需要演示商品、分类、轮播和商城服务时再导入 `sql/shop.sql`。
3. 重新启动 [后端服务](backend/README.md)。后端会从内置 OpenAPI 重建接口元数据和角色权限策略，默认 HTTP 地址为 `http://localhost:7001`。
4. 启动 [管理后台](frontend/admin/README.md)，默认地址为 `http://localhost:8848`。
5. 启动 [商城端](frontend/app/README.md)，H5 默认地址为 `http://localhost:5002`，微信小程序需用微信开发者工具导入构建目录。
6. 需要 Gorse 推荐联调时，再启动 [gorse](gorse/README.md)。

默认后台账号来自 `sql/default-data.sql`：

- `super / 112233`
- `admin / 112233`

管理后台登录页不默认填充租户编码，需要用户手动输入；默认租户编码为 `0000`。`tenant` 是系统内置的租户管理员角色编码，不是默认后台登录账号。

## 共享说明

- 接口契约位于 `backend/api/proto`，按 `base`、`common`、`system`、`shop` 分域；后端服务、前端 API 和生成 RPC 类型使用相同分层。
- 初始化 SQL 位于 `sql`：`default-data.sql` 维护默认租户、菜单和固定角色；后端每次启动使用 GORM 清空 `base_api`、`casbin_rule` 并重置自增 ID，根据当前 OpenAPI 重新生成接口元数据，再按角色菜单和接口数据重建权限策略。
- 默认角色固定为 `super(1)`、`tenant(2)`、`admin(3)`、`user(4)`、`guest(5)`；`tenant` 角色用于租户管理员，不能在角色管理中修改。
- Casbin 策略使用租户化字段，并按真实 HTTP Method 根据所有角色的菜单权限生成。
- 后端会托管 `backend/data/admin` 与 `backend/data/app` 下的前端构建产物，分别对应 `/admin` 和 `/app`。
- 推荐服务为可选联调模块；未启动 Gorse 时，推荐链路应依赖后端本地兜底策略保证页面可用。
