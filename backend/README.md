# backend

`backend` 是 `shop` 项目的后端服务，基于 `Go + Kratos`，同时提供 HTTP、gRPC、OpenAPI、文件上传、本地静态资源托管和定时任务能力。

## 技术栈

- Go `1.26`
- Kratos
- gRPC + HTTP
- GORM / GORM Gen
- MySQL
- Casbin
- Wire
- Buf / Protobuf / OpenAPI

## 目录结构

```text
backend
├── internal/cmd/server   # 实际启动入口
├── service               # admin / app / base 服务实现
├── server                # HTTP / gRPC 服务装配
├── configs               # 运行配置
├── api                   # proto、buf 配置与生成产物
├── pkg                   # 公共业务层、任务、推荐能力、生成代码
├── data                  # 本地上传目录与静态资源目录
├── certs                 # 证书目录
└── Makefile              # 常用命令
```

## 已覆盖能力

- 后台基础能力：登录、验证码、Token 刷新、用户、角色、菜单、部门、岗位、字典、配置、日志。
- 商城管理能力：商品分类、商品信息、规格、属性、SKU、轮播图、热门推荐、商城服务、门店管理。
- 商城端能力：分类、商品详情、推荐、购物车、收藏、地址、订单、支付、门店认证。
- 统计分析：工作台、用户分析、商品分析、商品月报、商品日报、订单分析、订单月报、订单日报、支付账单。

## 推荐能力

当前推荐链路已经具备以下能力：

- 推荐主体同时支持登录用户和匿名主体，匿名主体通过请求头 `X-Recommend-Anonymous-Id` 透传。
- 行为链路已覆盖推荐请求、曝光、点击、浏览、收藏、加购、下单、支付。
- 排序采用“场景关联 + 用户商品偏好 + 类目偏好 + 场景热度 + 全站热度 + 新鲜度”的统一组合，并带有重复购买降权、曝光惩罚和类目打散。
- 推荐公共能力下沉在 `pkg/recommend`，推荐链路 DTO 统一放在 `service/app/dto`，商城推荐业务按 Case 拆分在 `service/app/biz`。
- 当前已维护的推荐域表包括：
  - 原始事实：`recommend_request`、`recommend_request_item`、`recommend_exposure`、`recommend_exposure_item`、`recommend_goods_action`
  - 聚合结果：`recommend_user_preference`、`recommend_user_goods_preference`、`recommend_goods_relation`、`recommend_goods_stat_day`
  - 重建与评估：`recommend_actor_bind_log`、`recommend_eval_report`、`recommend_model_version`

## 推荐任务

当前代码已接入以下推荐相关后台任务：

- `RecommendGoodsStatDay`：推荐商品日统计
- `RecommendUserPreferenceRebuild`：推荐用户偏好重建，固定 30 天窗口
- `RecommendGoodsRelationRebuild`：推荐商品关联重建，固定 30 天窗口
- `RecommendEvalReport`：推荐离线评估报告，按天生成场景级 CTR、CVR、Precision、Recall、NDCG 指标

## 环境要求

- Go `1.26+`
- MySQL `8.x`

## 配置说明

实际启动命令使用：

```bash
go run ./internal/cmd/server -conf ./configs
```

主要配置文件：

| 文件 | 作用 | 关键说明 |
| --- | --- | --- |
| `configs/data.yaml` | 数据库配置 | 默认数据库为 `shop_test`，`enable_migrate: true` |
| `configs/server.yaml` | HTTP / gRPC 配置 | HTTP `7001`，gRPC `6001`，默认启用 Swagger 与 pprof |
| `configs/auth.yaml` | 鉴权配置 | 包含白名单接口与可选鉴权接口 |
| `configs/oss.yaml` | 文件存储配置 | 默认 `type: local`，根目录 `./data` |
| `configs/configs.yaml` | 商城业务配置 | 微信小程序、微信支付、商品推荐权重配置 |

补充说明：

- `configs/configs.yaml` 中的微信配置当前要求非空，联调阶段可先填占位值。
- `configs/configs.yaml` 中的 `shop.recommend` 当前用于维护商品热度分落库权重、推荐排序权重、行为权重、排序参数、召回参数和主体曝光惩罚参数。
- `GET /api/admin/base/api` 返回给菜单管理的接口列表时，会自动过滤 `configs/auth.yaml` 中配置为白名单或可选鉴权的接口。

## 数据库初始化

### 1. 创建数据库

```sql
CREATE DATABASE shop_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

然后按实际环境修改 `configs/data.yaml` 中的 `source`。

### 2. 启动一次服务自动建表

```bash
cd backend
go run ./internal/cmd/server -conf ./configs
```

### 3. 导入初始化数据

在仓库根目录执行：

```bash
mysql -uroot -p shop_test < sql/default-data.sql
mysql -uroot -p shop_test < sql/casbin_rule.sql
mysql -uroot -p shop_test < sql/base_area.sql
```

如需导入演示商品数据：

```bash
mysql -uroot -p shop_test < sql/shop.sql
```

默认后台账号：

- `super / 112233`
- `admin / 112233`

## 启动服务

```bash
cd backend
go run ./internal/cmd/server -conf ./configs
```

默认地址：

- HTTP：`http://localhost:7001`
- gRPC：`localhost:6001`
- Swagger UI：`http://localhost:7001/docs/`
- OpenAPI：`http://localhost:7001/docs/openapi.yaml`

## 静态资源托管

后端启动后会自动处理两类本地静态资源：

- `/shop/*` 直接映射到 `backend/data/shop/*`
- `backend/data` 下一级子目录只要存在 `index.html`，就会自动按目录名注册为单页应用路由

当前仓库中已经存在的本地单页入口包括：

- `backend/data/admin/index.html` -> `/admin`
- `backend/data/app/index.html` -> `/app`
- `backend/data/geeker/index.html` -> `/geeker`

## 常用命令

以下命令都在 `backend` 目录执行：

```bash
make init
make fmt
make api
make openapi
make ts
make gorm-gen
make gen
make wire
make docker-build
```

对应说明：

- `make init`：安装 `protoc` 相关插件、`buf`、`wire`、`goimports` 等工具。
- `make fmt`：使用 `goimports` 格式化 Go 代码。
- `make api`：生成 proto 对应 Go 代码。
- `make openapi`：生成 OpenAPI 文档。
- `make ts`：生成前端 TypeScript RPC 代码。
- `make gorm-gen`：根据当前 MySQL `shop_test` 表结构生成 `pkg/gen`。
- `make gen`：一键生成 Go / OpenAPI / TypeScript 产物。
- `make wire`：生成依赖注入代码。
- `make docker-build`：构建 Docker 镜像。

说明：

- `Makefile` 里的 `run` 目标仍然指向旧入口 `./cmd/server`，当前不要直接使用。
- 当前可用启动命令是 `go run ./internal/cmd/server -conf ./configs`。

## Docker 打包

```bash
cd backend
make docker-build
```

默认构建命令：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./internal/cmd/server
```

如需自定义镜像名和标签：

```bash
make docker-build IMAGE=your-registry/backend TAG=v1.0.0
```
