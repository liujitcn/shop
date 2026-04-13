# backend

`backend` 是 `shop` 项目的后端服务，基于 `Go + Kratos`，同时提供 HTTP、gRPC、OpenAPI、文件上传和本地静态资源托管能力。

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
├── api                   # proto 与生成配置
├── pkg                   # 业务公共层、中间件、模型与查询代码
├── data                  # 本地上传目录与前端静态资源目录
├── certs                 # 证书目录
└── Makefile              # 后端常用命令
```

## 已覆盖模块

- 后台基础能力：登录、验证码、Token 刷新、用户、角色、菜单、部门、岗位、字典、配置、日志。
- 商城管理能力：商品分类、商品信息、规格、属性、SKU、轮播图、热门推荐、商城服务、门店管理。
- 商城端能力：分类、商品详情、推荐、购物车、收藏、地址、订单、支付、门店认证。
- 统计分析：工作台、用户分析、商品分析、订单分析、支付账单。

## 推荐能力

- 推荐主体：同时支持登录用户与匿名主体，匿名主体通过请求头 `X-Recommend-Anonymous-Id` 透传。
- 行为链路：已覆盖推荐请求、曝光、点击、浏览、收藏、加购、下单、支付明细采集。
- 排序能力：当前采用“场景关联 + 用户商品偏好 + 类目偏好 + 场景热度 + 全站热度 + 新鲜度”的统一排序，并带有重复购买降权、曝光惩罚和类目打散。
- 模块结构：推荐公共类型、排序、解释与过滤能力已下沉到 `pkg/recommend`，推荐链路 DTO 统一放在 `service/app/dto`，`service/app/biz` 侧按表对应的 Case 拆分商城场景召回与事件处理。
- 聚合能力：当前已在线增量维护 `recommend_user_preference`、`recommend_user_goods_preference`、`recommend_goods_relation`，离线重建所需原始表已具备，但重建执行器尚未补齐。
- 改造方向：推荐能力后续按“商城专用 Gorse 化推荐服务”演进，不走通用推荐平台路线，并严格区分“商城业务事实层 / 商城推荐域层 / Gorse 内核层”三层边界，见 [docs/recommendation-gorse-mall-roadmap.md](docs/recommendation-gorse-mall-roadmap.md)。
- 现状评估：推荐链路闭环检查与 Gorse 差距分析见 [docs/recommendation-chain-review.md](docs/recommendation-chain-review.md)。
- 表结构差距：MySQL 场景下的推荐表结构缺口记录见 [docs/recommendation-mysql-gap.md](docs/recommendation-mysql-gap.md)。
- 执行记录：分批改造进度、当前停留点与确认状态见 [docs/recommendation-execution-log.md](docs/recommendation-execution-log.md)。

## 推荐任务

当前代码中已接入以下推荐相关后台任务：

- `RecommendGoodsStatDay`：推荐商品日统计
- `RecommendUserPreferenceRebuild`：推荐用户偏好重建，按绑定日志回放受影响用户，固定 30 天窗口
- `RecommendGoodsRelationRebuild`：推荐商品关联重建，按绑定日志触发全量关系重建，固定 30 天窗口

当前仍未实现的推荐任务：

- `RecommendEvalReport`：推荐离线评估报告

## 环境要求

- Go `1.26+`
- MySQL `8.x`

## 配置说明

启动命令固定使用：

```bash
go run ./internal/cmd/server -conf ./configs
```

主要配置文件：

| 文件 | 作用 | 关键说明 |
| --- | --- | --- |
| `configs/data.yaml` | 数据库配置 | 默认数据库为 `shop_test`，`enable_migrate: true` |
| `configs/server.yaml` | HTTP / gRPC 端口 | HTTP `7001`，gRPC `6001` |
| `configs/auth.yaml` | JWT 配置 | 包含白名单接口 |
| `configs/oss.yaml` | 文件存储 | 默认 `type: local`，根目录 `./data` |
| `configs/configs.yaml` | 商城自定义配置 | 微信小程序与微信支付配置 |

补充：

- 本地上传目录会映射到 `/shop/*`，即 `backend/data/shop/... -> http://localhost:7001/shop/...`。
- 后端会自动扫描 `backend/data` 下包含 `index.html` 的一级子目录，并按目录名挂载单页应用。
- 因此 `backend/data/shop/index.html` 对应 `/shop`，`backend/data/app/index.html` 对应 `/app`。
- `configs/configs.yaml` 中的微信配置当前要求非空，联调阶段可先填占位值。
- `GET /api/admin/base/api` 返回给菜单管理的接口列表时，会自动过滤 `configs/auth.yaml` 中配置为白名单或可选鉴权的接口。

## 数据库初始化

### 1. 创建数据库

```sql
CREATE DATABASE shop_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

然后修改 `configs/data.yaml` 中的 `source`。

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

## 前端静态资源协作

- 管理后台构建产物输出到 `backend/data/shop`。
- 商城 H5 构建产物输出到 `backend/data/app`。
- 后端运行后会自动托管这两个目录，不需要额外配置 Nginx 才能本地联调。

## 常用命令

以下命令都在 `backend` 目录执行：

```bash
make init
make fmt
make api
make openapi
make ts
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
- `make gen`：一键生成 Go / OpenAPI / TypeScript 产物。
- `make wire`：生成依赖注入代码。
- `make docker-build`：构建 Docker 镜像。

注意：

- `Makefile` 里的 `run` 目标仍然指向 `./cmd/server`，和当前真实入口不一致。
- 当前实际可用启动命令仍然是 `go run ./internal/cmd/server -conf ./configs`。

## Docker 打包

```bash
cd backend
make docker-build
```

默认会先构建：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./internal/cmd/server
```

然后使用 `Dockerfile` 打包 `bin/server`、`configs`、`certs` 到运行时镜像。

如需自定义镜像名和标签：

```bash
make docker-build IMAGE=your-registry/backend TAG=v1.0.0
```
