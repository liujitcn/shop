# shop

`shop` 是一个前后端分离的商城项目，仓库包含：

- `backend`：基于 Go + Kratos 的后端服务，提供 HTTP、gRPC、OpenAPI、上传和静态资源能力。
- `frontend/admin`：基于 Vue 3 + Vite + Element Plus 的管理后台。
- `frontend/app`：基于 uni-app + Vue 3 + TypeScript 的商城端，支持微信小程序与 H5，保留 App 构建能力。
- `gorse`：推荐系统本地部署与运行配置。
- `sql`：初始化数据、权限数据、地区数据和演示商品数据。

## 主要能力

- 管理端：登录认证、菜单权限、用户/角色/部门/岗位、字典、系统配置、日志、工作台与数据分析。
- 商城端：首页、分类、搜索、商品详情、热门推荐、个性化推荐、购物车、收藏、地址、订单、支付结果、门店认证。
- 后端：管理端接口、商城端接口、文件上传、Swagger UI、OpenAPI 文档、静态资源托管。

## 推荐能力

- 已支持匿名与登录两类推荐主体，匿名主体通过 `X-Recommend-Anonymous-Id` 透传。
- 推荐主链路已覆盖 `request -> exposure -> click -> view -> collect -> cart -> order -> pay`，其中：
  - `request`、`exposure`、`click`、`view` 由商城前端调用 `/api/app/recommend/*` 完成。
  - `collect`、`cart`、`order`、`pay` 由后端在真实业务写库成功后异步回写，避免前端埋点与业务事实不一致。
- 推荐请求会落到本地 `recommend_request` / `recommend_request_item`，推荐事件会落到本地 `recommend_event`，形成可追踪的归因链路。
- 在线推荐优先走 `backend/pkg/recommend` 对 Gorse 的用户推荐或会话推荐；未命中时回退到同类目商品和最新热销商品。
- 匿名推荐历史会在登录后绑定到当前用户，并把匿名阶段积累的行为回放到登录用户画像。
- 商城前端推荐相关实现目前集中在 `frontend/app/src/api/app/recommend.ts`、`frontend/app/src/stores/modules/recommend.ts`、`frontend/app/src/utils/navigation.ts`、`frontend/app/src/components/XtxGuess.vue`。
- 推荐系统部署与配置位于仓库根目录 `gorse`，后端通过定时任务 `RecommendSync` 和异步队列同步用户、商品与行为反馈。
- 本地启用 Gorse 时，后端 `backend/configs/configs_local.yaml` 的 `shop.recommend.entryPoint` 需要配置为 `http://127.0.0.1:8088`，因为当前 Go 客户端访问的是 Gorse HTTP API 端口。

## 目录结构

```text
.
├── backend
│   ├── internal/cmd/server   # 后端实际入口
│   ├── service               # admin / app / base 服务实现
│   ├── server                # HTTP / gRPC 装配
│   ├── configs               # 配置文件
│   ├── api                   # proto、buf、生成代码
│   └── data                  # 本地静态资源与上传目录
├── frontend
│   ├── admin                 # 管理后台
│   ├── app                   # uni-app 商城端
│   └── Makefile              # 前端常用命令封装
├── gorse                     # 推荐系统本地部署配置
└── sql                       # 初始化 SQL
```

## 环境要求

- Go `1.26+`
- Node.js `16.18+`
- `pnpm`
- MySQL `8.x`

## 快速开始

### 1. 初始化数据库

先创建数据库，例如：

```sql
CREATE DATABASE shop_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

修改 `backend/configs/data.yaml` 中的 MySQL 连接串。

### 2. 启动后端并自动建表

```bash
cd backend
go run ./internal/cmd/server -conf ./configs
```

默认端口：

- HTTP：`http://localhost:7001`
- gRPC：`localhost:6001`
- Swagger UI：`http://localhost:7001/docs/`
- OpenAPI：`http://localhost:7001/docs/openapi.yaml`

### 3. 导入初始化数据

在仓库根目录执行：

```bash
mysql -uroot -p shop_test < sql/default-data.sql
mysql -uroot -p shop_test < sql/casbin_rule.sql
mysql -uroot -p shop_test < sql/base_area.sql
```

如需演示商品数据，再执行：

```bash
mysql -uroot -p shop_test < sql/shop.sql
```

默认后台账号：

- `super / 112233`
- `admin / 112233`

### 4. 启动管理后台

```bash
cd frontend/admin
pnpm install
pnpm dev
```

默认地址：`http://localhost:8848`

### 5. 启动商城 H5

```bash
cd frontend/app
pnpm install
pnpm dev:h5
```

默认地址：`http://localhost:5002`

### 6. 启动微信小程序

```bash
cd frontend/app
pnpm install
pnpm dev:mp-weixin
```

再使用微信开发者工具导入 `dist/dev/mp-weixin`。

## 构建说明

- 管理后台 `pnpm build` 会输出到 `backend/data/shop`，后端自动将其挂载到 `/shop`。
- 商城 H5 `pnpm build:h5` 会输出到 `backend/data/app`，后端自动将其挂载到 `/app`。
- 后端会扫描 `backend/data` 下包含 `index.html` 的一级子目录，并按目录名注册单页应用路由。

## 常用命令

后端：

```bash
cd backend
make init
make fmt
make api
make openapi
make ts
make gen
make docker-build
```

前端：

```bash
make -C frontend init
make -C frontend run-admin
make -C frontend run-app
make -C frontend build
```

## 说明

- 更详细的服务说明见 `backend/README.md`。
- 管理端说明见 `frontend/admin/README.md`。
- 商城端说明见 `frontend/app/README.md`。
