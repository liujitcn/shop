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

当前仓库已经从“系统内置推荐”切换为“业务后端保留最小事实链路，推荐结果交给 gorse”。

当前后端仍然保留的推荐能力：

- 推荐主体支持匿名主体和登录主体，匿名主体通过请求头 `X-Recommend-Anonymous-Id` 透传。
- 行为链路覆盖推荐请求、曝光、点击、浏览、收藏、加购、下单、支付。
- 推荐请求会生成 `requestId`，并把请求主记录和逐商品明细落到数据库，便于后续曝光和转化归因。
- 登录后会把匿名主体产生的推荐请求和反馈事件统一绑定到登录用户。
- 商品统计任务会继续消费推荐反馈事件以及收藏、购物车、订单数据，为商城商品统计保留基础热度口径。

当前推荐相关表只保留以下 6 张：

- `recommend_request`：推荐请求主表
- `recommend_request_item`：推荐返回商品明细
- `recommend_actor_bind_log`：匿名主体和登录主体绑定日志
- `recommend_feedback_event`：曝光、点击、浏览、收藏、加购、下单、支付统一反馈事实
- `recommend_strategy_release`：场景到 gorse 策略编码的发布表
- `recommend_metrics_day`：推荐指标日报

说明：

- `goods_stat_day` 仍然保留，但它属于商城商品统计表，不再是旧内置推荐域表。
- 旧的 `recommend_model_version`、`recommend_eval_report`、`recommend_exposure`、`recommend_exposure_item`、`recommend_goods_action`、`recommend_goods_relation`、`recommend_user_preference`、`recommend_user_goods_preference` 等表和对应业务代码已从后端主链路移除。
- 当前 `RecommendGoods` 先使用“场景策略码 + 商品分页兜底”保证接口可用，后续可直接把该入口替换为 gorse API 调用，不影响现有埋点和归因表结构。

当前推荐主链路代码：

- `service/app/biz/recommend_request.go`
- `service/app/biz/recommend_exposure.go`
- `service/app/biz/recommend_goods_action.go`
- `service/app/biz/recommend_actor_bind_log.go`
- `service/app/biz/recommend_feedback_support.go`

当前接入说明见：

- `backend/recommend-status-vs-gorse.md`

## 推荐任务

当前后端已经移除旧内置推荐的训练、聚合、发布、评估任务，不再注册 `Recommend*` 系列离线任务。

当前仍保留的相关任务只有：

- `GoodsStatDay`：商品日统计任务。该任务会读取 `recommend_feedback_event` 的 `view` 事件，并结合 `user_collect`、`user_cart`、`order_info`、`order_goods` 汇总商品日统计。

推荐缓存和 gorse 相关边界：

- `pkg/recommend/cache`、`pkg/recommend/offline/materialize` 只保留通用缓存结构和轻量写回能力，便于后续对接 gorse 返回结果或附加排序缓存。
- gorse 自身的部署和配置位于仓库根目录 `gorse/` 下，启动方式以 `gorse/docker-compose.yml` 为准。
- 当前后端并不会在本地训练推荐模型，也不会再依赖推荐版本管理闭环。

## 环境要求

- Go `1.26+`
- MySQL `8.x`

## 配置说明

实际启动命令使用：

```bash
go run ./internal/cmd/server --conf ./configs
```

主要配置文件：

| 文件 | 作用 | 关键说明 |
| --- | --- | --- |
| `configs/data.yaml` | 数据库配置 | 默认数据库为 `shop_test`，`enable_migrate: true` |
| `configs/server.yaml` | HTTP / gRPC 配置 | HTTP `7001`，gRPC `6001`，默认启用 Swagger 与 pprof |
| `configs/auth.yaml` | 鉴权配置 | 包含白名单接口与可选鉴权接口 |
| `configs/oss.yaml` | 文件存储配置 | 默认 `type: local`，根目录 `./data` |
| `configs/configs.yaml` | 商城业务配置 | 微信小程序、微信支付、商品统计相关配置 |

补充说明：

- `configs/configs.yaml` 中的微信配置当前要求非空，联调阶段可先填占位值。
- `configs/data.yaml` 中的 `redis.addr` 需要使用数组格式，例如 `addr: [\"127.0.0.1:6379\"]`，否则启动时会在配置解析阶段报错。
- `configs/configs.yaml` 中的 `shop.recommend` 当前只用于商品统计等轻量配置，不再承载旧内置推荐的模型训练、召回、精排和发布配置。
- gorse 的运行参数不在 `backend/configs` 下维护，而是在仓库根目录 `gorse/config/config.toml` 和 `gorse/docker-compose.yml` 中维护。
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
go run ./internal/cmd/server --conf ./configs
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
go run ./internal/cmd/server --conf ./configs
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
- 当前可用启动命令是 `go run ./internal/cmd/server --conf ./configs`。

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
