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
| 模块设计 | [后端服务设计](docs/后端服务设计.md)、[管理后台设计](docs/管理后台设计.md)、[商城端设计](docs/商城端设计.md)、[数据库与初始化数据设计](docs/数据库与初始化数据设计.md) |
| 订单链路 | [订单数据流转设计](docs/订单数据流转设计.md) |
| 推荐链路 | [推荐系统设计](docs/推荐系统设计.md)、[推荐数据流转设计](docs/推荐数据流转设计.md) |
| 统计报表 | [统计数据流转设计](docs/统计数据流转设计.md) |
| 评价审核 | [评价与审核数据流转设计](docs/评价与审核数据流转设计.md) |

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

1. 创建 `shop_test` 数据库，启动后端并导入 `sql/default-data.sql`、`sql/base_area.sql`，演示数据可导入 `sql/shop.sql`。
2. 启动 [后端服务](backend/README.md)，默认 HTTP 地址为 `http://localhost:7001`。
3. 启动 [管理后台](frontend/admin/README.md)，默认地址为 `http://localhost:8848`。
4. 启动 [商城端](frontend/app/README.md)，H5 默认地址为 `http://localhost:5002`，微信小程序需用微信开发者工具导入构建目录。
5. 需要 Gorse 推荐联调时，再启动 [gorse](gorse/README.md)。

默认后台账号来自 `sql/default-data.sql`：

- `super / 112233`
- `admin / 112233`

## 共享说明

- 初始化 SQL 位于 `sql`，其中 `casbin_rule.sql` 当前为空文件，权限和菜单初始化主要维护在 `default-data.sql`。
- 后端会托管 `backend/data/admin` 与 `backend/data/app` 下的前端构建产物，分别对应 `/admin` 和 `/app`。
- 推荐服务为可选联调模块；未启动 Gorse 时，推荐链路应依赖后端本地兜底策略保证页面可用。
