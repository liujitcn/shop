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
- 当前阶段 2 已将商品行为投影器、用户商品偏好、用户类目偏好、商品关联以及推荐商品日统计的离线聚合统一收口到 `pkg/recommend/offline/aggregate`，`service/app/biz` 和 `pkg/job/task` 中的推荐聚合入口仅保留事实查询、删旧数据、调用聚合器和批量落库。
- 当前阶段 3 已补齐推荐缓存 key 规范、`kratos-kit/cache` 适配层与 `hot`、`latest`、`similar_item` 三类结果的写缓存任务，并在在线推荐链路接入“缓存优先，未命中查库”的读取挂点；当前无 Redis 时走内存后端，有 Redis 时直接复用 Redis 发布缓存。
- 推荐请求的在线排障字段会统一收口到 `sourceContext.onlineDebugContext`，当前已覆盖 `cacheHitSources`、`cacheReadContext`、`recallProbeContext`、`joinRecallContext`、`similarUserObservationContext` 等调试信息。
- 当前阶段 4 已补齐相似用户、协同过滤、内容相似三类召回探针的缓存键约定与读取挂点；探针是否启用由 `recommend_model_version.config_json.recall_probe` 控制，当前会在 `onlineDebugContext.recallProbeContext` 中记录探针配置和观测结果，并支持通过 `join_candidate` 把低风险召回灰度并入 `GOODS_DETAIL` 场景候选池。
- 当前阶段 4 的灰度召回还会把 `joinRecallContext` 收口到 `onlineDebugContext`，用于区分“已并入候选”“实际进入候选池”“实际返回到当前页”三层命中情况。
- 当前阶段 4 还会把相似用户探针的观测结果收口到 `onlineDebugContext.similarUserObservationContext`，用于查看相似用户偏好商品与当前候选、当前返回页，以及协同过滤和内容相似灰度结果的重合数量和覆盖率。
- 当前阶段 4 和阶段 5 之间，在线读缓存与召回探针还会把版本号、版本发布时间、缓存发布时间、文档数量、请求返回数量等读取元信息收口到 `cacheReadContext` 与探针子上下文，便于排查“命中了哪一版缓存、这版缓存是什么时间写出来的、当前返回了多少条”。
- 当前阶段 6 已开始把在线层纯逻辑从 `service/app/biz` 下沉到 `pkg/recommend/online`；首批已新增 `pkg/recommend/online/recall`，承接探针结果解析、灰度召回上下文归一化和相似用户观测逻辑，主推荐入口暂未切换。
- 当前阶段 6 继续新增了 `pkg/recommend/online/planner`，用于承接匿名态和登录态请求的前置计划对象，把 `priority/category/recall/cache` 这些在线编排状态从 `service/app/biz/recommend_request.go` 收口到在线层，主推荐读取和结果落库流程暂未改写。
- 当前阶段 6 还继续把 `CART`、`ORDER_*`、`GOODS_DETAIL`、`profile` 和 `latest` fallback 的场景级规划动作下沉到 `pkg/recommend/online/planner`，当前 `service/app/biz/recommend_request.go` 中的场景 `switch` 已开始只保留查库和桥接职责。
- 当前阶段 6 又继续补了 `pkg/recommend/online/planner.SceneInput`，用于承接购物车商品、订单商品、源商品、场景优先候选、类目补足和缓存命中来源等桥接输入，并统一构建 `orderId`、`goodsId`、`cartGoodsIds`、`orderGoodsIds`、`sourceGoodsIds` 等在线来源上下文字段。
- 当前阶段 6 还继续把结果来源上下文和在线调试上下文的组装收口到 `pkg/recommend/online/planner`，匿名态与登录态现在都会先通过 planner 构建结果上下文，再由 planner 统一补 join recall 和 similar-user observation 调试字段。
- 当前阶段 6 又继续补了 `pkg/recommend/online/planner.ResultSnapshot`，用于承接 `candidateLimit`、`sceneHotGoodsIds`、`candidateGoodsIds`、`anonymousCandidateGoodsIds`、`returnedScoreDetails` 等结果快照字段，匿名态与登录态的最终来源上下文现在也开始通过 planner 结果对象统一构建。
- 当前阶段 6 又继续补了候选池状态方法，用于承接类目补足候选、latest 候选、latest 排除集合、匿名态场景候选合并和登录态最终候选集合合并，`service/app/biz/recommend_request.go` 中这类纯集合状态和去重规则继续从主链路函数里迁出。
- 当前阶段 6 又继续补了候选池查询参数计划，用于承接类目补足查询参数、latest 查询参数和匿名态 latest fallback 判断，`service/app/biz/recommend_request.go` 中这类“是否需要查、查多少、排除哪些商品”的纯参数决策也开始通过 planner 统一组织。
- 当前阶段 6 又继续补了共享候选池桥接查询方法，类目补足和 latest 兜底的商品 ID 提取逻辑已经开始复用统一实现，`service/app/biz/recommend_request.go` 中重复的查询拼装与结果提取代码进一步缩短。
- 当前阶段 6 又继续补了匿名态 latest 回退商品列表桥接方法，把 latest fallback 的分页查询与总数返回逻辑继续从主链路函数里抽离，匿名态 latest 回退分支现在更接近“计划决策 + 桥接调用”的结构。
- 当前阶段 6 又继续补了排序信号快照方法，用于承接匿名态与登录态候选商品列表的过滤、商品 ID 提取和类目 ID 提取，`service/app/biz/recommend_request.go` 中候选信号加载前的结果整理逻辑继续从主链路函数里迁出。
- 当前阶段 6 又继续补了分页 explain 快照方法，用于统一承接当前页召回来源列表、评分明细和返回商品编号提取，`service/app/biz/recommend_request.go` 中匿名态与登录态的 explain 组装循环继续从主链路函数里迁出。
- 当前阶段 6 又继续补了排序结果分页窗口快照方法，用于统一承接总数计算、分页窗口切片和空页判定，`service/app/biz/recommend_request.go` 中匿名态与登录态的分页窗口和空页分支继续从主链路函数里迁出。
- 当前阶段 6 又继续补了结果回写桥接方法，用于统一承接匿名态 latest fallback、匿名态空页、匿名态正常页、登录态空页和登录态正常页的 `ResultSnapshot` 构建与 `sourceContext` 回写，`service/app/biz/recommend_request.go` 中末尾结果回写分支继续从主链路函数里迁出。
- 当前阶段 6 又继续补了排序信号加载计划，用于统一承接匿名态与登录态的场景编号、候选商品编号、候选类目编号和关系分源商品编号，`service/app/biz/recommend_request.go` 中排序信号加载前的参数组织继续从主链路函数里迁出。
- 当前阶段 6 又继续补了领域信号桥接方法，用于统一承接匿名态 `AnonymousSignals` 与登录态 `PersonalizedSignals` 的组装，`service/app/biz/recommend_request.go` 中信号加载完成后的领域对象桥接继续从主链路函数里迁出。
- 当前阶段 6 又继续补了 explain 召回补标方法，用于统一承接匿名态内容相似灰度补标，以及登录态内容相似、协同过滤灰度补标，`service/app/biz/recommend_request.go` 中灰度召回 explain 补标逻辑继续从主链路函数里迁出。
- 当前阶段 6 又继续补了共享分页桥接底层方法，用于统一承接类目补足、latest 候选和 latest fallback 的 `PageGoodsInfo` 调用、排除商品过滤和分页参数拼装，`service/app/biz/recommend_request.go` 中三类分页桥接查询的底层实现继续收口。
- 当前阶段 6 已开始补 `pkg/recommend/online/record`，首批承接推荐请求主表 `sourceContext` 的持久化整理和在线调试上下文压缩，`service/app/biz/recommend_request.go` 中主表上下文裁剪逻辑已开始从保存函数里迁出。
- 当前阶段 6 又继续补了 `pkg/recommend/online/record` 的逐商品明细模型构建方法，用于统一承接 `returnedScoreDetails` 索引收敛、单商品召回来源回退和 `RecommendRequestItem` 列表组装，`service/app/biz/recommend_request_item.go` 中批量落库前的纯整理逻辑继续从业务层迁出。
- 当前阶段 6 又继续补了 `pkg/recommend/online/record` 的逐商品明细读取整理方法，用于统一承接关联商品编号提取和商品位次映射构建，`service/app/biz/recommend_request_item.go` 中按 requestId 回查明细后的纯循环整理逻辑继续从业务层迁出。
- 当前阶段 6 又继续补了 `recommend_request_item.go` 的共享读桥接方法，用于统一承接 `requestId -> requestEntity` 和 `requestEntity.ID -> requestItemList` 的查询路径，关联商品、位次映射两条回查路径的重复查询拼装继续缩短。
- 当前阶段 6 又继续补了 `pkg/recommend/online/planner.ListGoodsIds`，统一承接 explain 快照、类目补足候选和 latest 候选的商品编号提取；`recommend_request.go` 里这类纯商品 ID 提取工具已继续从业务层迁出。
- 当前阶段 6 又继续补了 `pkg/recommend/online/planner.GoodsPoolPageSnapshot` 与 `GoodsPoolQuery` 的可执行判断方法，统一承接候选池分页桥接结果中的 `list/id/total` 提取和查询启用判断；`recommend_request.go` 里类目补足、latest 候选和 latest fallback 的结果侧分支继续从业务层迁出。
- 当前阶段 6 又继续补了 `pkg/recommend/online/planner` 的场景输入构造方法，统一承接 `CART`、`ORDER_*`、`GOODS_DETAIL` 场景的 `SceneInput` 映射；`recommend_request.go` 里的场景分支当前进一步收敛为只保留查库和桥接调用。
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
