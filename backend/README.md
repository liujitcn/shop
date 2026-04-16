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

当前推荐重构另外补充以下边界约定：

- `pkg/recommend` 只允许放领域对象、纯逻辑、纯规则和纯编排，不允许放 Repo、GORM、SQL 条件拼装、查询桥接接口，也不允许设计查询数据库语义。
- `service/app/biz` 只允许保留表名相关 Case，不允许继续新增 `recommendAnonymousCandidateLoader`、`recommendCategoryCandidateLoader` 这类非表名相关结构。
- `RecommendCase` 不允许继续新增新的辅助方法承载推荐主链路细节，只允许引入并调用其他已有 Case。
- 现有不符合上述边界的 loader / adapter 风格实现视为阶段 6 过渡代码，后续不再继续扩张，优先做收口和回退。
- 当前阶段 2 已将商品行为投影器、用户商品偏好、用户类目偏好、商品关联以及推荐商品日统计的离线聚合统一收口到 `pkg/recommend/offline/aggregate`，`service/app/biz` 和 `pkg/job/task` 中的推荐聚合入口仅保留事实查询、删旧数据、调用聚合器和批量落库。
- 当前阶段 3 已补齐推荐缓存 key 规范、`kratos-kit/cache` 适配层与 `hot`、`latest`、`similar_item` 三类结果的写缓存任务，并在在线推荐链路接入“缓存优先，未命中查库”的读取挂点；当前无 Redis 时走内存后端，有 Redis 时直接复用 Redis 发布缓存。
- 推荐请求的在线排障字段会统一收口到 `sourceContext.onlineDebugContext`，当前已覆盖 `cacheHitSources`、`cacheReadContext`、`recallProbeContext`、`joinRecallContext`、`similarUserObservationContext` 等调试信息。
- 当前阶段 4 已补齐相似用户、协同过滤、内容相似三类召回探针的缓存键约定与读取挂点；探针是否启用由 `recommend_model_version.config_json.recall_probe` 控制，当前会在 `onlineDebugContext.recallProbeContext` 中记录探针配置和观测结果，并支持通过 `join_candidate` 把低风险召回灰度并入 `GOODS_DETAIL` 场景候选池。
- 当前阶段 4 的灰度召回还会把 `joinRecallContext` 收口到 `onlineDebugContext`，用于区分“已并入候选”“实际进入候选池”“实际返回到当前页”三层命中情况。
- 当前阶段 4 还会把相似用户探针的观测结果收口到 `onlineDebugContext.similarUserObservationContext`，用于查看相似用户偏好商品与当前候选、当前返回页，以及协同过滤和内容相似灰度结果的重合数量和覆盖率。
- 当前阶段 4 和阶段 5 之间，在线读缓存与召回探针还会把版本号、版本发布时间、缓存发布时间、文档数量、请求返回数量等读取元信息收口到 `cacheReadContext` 与探针子上下文，便于排查“命中了哪一版缓存、这版缓存是什么时间写出来的、当前返回了多少条”。
- 当前阶段 6 保留了 `pkg/recommend/online/recall`、`pkg/recommend/online/planner`、`pkg/recommend/online/feature`、`pkg/recommend/online/rank`、`pkg/recommend/online/record` 这些纯逻辑模块，分别承接探针上下文收口、请求计划状态、排序信号装配、分页 explain 和记录整理。
- 当前阶段 6 已按最新边界完成一次代码收口：`service/app/biz/recommend_request.go` 重新直接调用 `GoodsInfoCase`、`GoodsStatDayCase`、`RecommendGoodsRelationCase`、`RecommendUserPreferenceCase`、`RecommendUserGoodsPreferenceCase` 等表级 Case 组织场景查询、类目补足、latest 兜底、信号读取和排序执行。
- 原先为 `pkg/recommend` 适配而新增的 `recommendSceneLoader`、`recommendLatestLoader`、`recommendCategoryCandidateLoader`、`recommendAnonymousCandidateLoader`、`recommendCompositeCandidateLoader`、`recommendSimilarUserObservationLoader`、`recommendGoodsPoolPager`，以及 `pkg/recommend/online/cache`、`pkg/recommend/online/engine` 中的查询桥接实现，当前都已回退删除。
- 当前 `pkg/recommend` 不再承载数据库查询计划、分页桥接参数、Repo 适配接口和 DB 读取语义；推荐查询职责重新收口到 `service/app/biz` 的表名 Case 与 `RecommendRequestCase`。
- 当前 `RecommendCase` 继续只保留分页参数兜底、`requestId` 生成、请求落库和响应组装，不再承载新的推荐主链路细节。
- 当前阶段 5 已补齐相似用户、协同过滤、内容相似三类离线训练与写缓存任务，首版直接复用现有偏好聚合和商品属性做轻量训练，训练结果按版本发布到推荐缓存。
- 当前阶段 5 的写缓存任务已补最小运行摘要日志，会统一输出训练输入规模、版本数、发布子集合数、发布文档数、清理子集合数和总耗时，便于排查训练发布链路。
- 当前阶段 5 的写缓存任务在失败时也会统一输出失败摘要，包含当前执行阶段、已统计的输入规模、已发布进度和清理进度，便于快速定位卡在哪一步。
- 当前已维护的推荐域表包括：
  - 原始事实：`recommend_request`、`recommend_request_item`、`recommend_exposure`、`recommend_exposure_item`、`recommend_goods_action`
  - 聚合结果：`recommend_user_preference`、`recommend_user_goods_preference`、`recommend_goods_relation`、`recommend_goods_stat_day`
  - 重建与评估：`recommend_actor_bind_log`、`recommend_eval_report`、`recommend_model_version`

推荐系统后续的重构与能力补齐计划见：

- `backend/recommend-rebuild-plan.md`
- `backend/recommend-vs-gorse.md`

## 推荐任务

当前代码已接入以下推荐相关后台任务：

- `RecommendGoodsStatDay`：推荐商品日统计
- `RecommendUserPreferenceRebuild`：推荐用户偏好重建，固定 30 天窗口
- `RecommendGoodsRelationRebuild`：推荐商品关联重建，固定 30 天窗口
- `RecommendHotMaterialize`：推荐热门榜写缓存，按场景发布 `hot` 缓存
- `RecommendLatestMaterialize`：推荐最新榜写缓存，按场景发布 `latest` 缓存
- `RecommendSimilarItemMaterialize`：相似商品写缓存，按商品详情场景版本发布 `similar_item` 缓存
- `RecommendSimilarUserMaterialize`：相似用户写缓存，按启用版本发布 `user-to-user` 缓存
- `RecommendCollaborativeFilteringMaterialize`：协同过滤写缓存，按启用版本发布 `collaborative-filtering` 缓存
- `RecommendContentBasedMaterialize`：内容相似写缓存，按启用版本发布 `content-based` 缓存
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
