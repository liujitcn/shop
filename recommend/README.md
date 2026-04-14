# recommend

`recommend` 是 `shop` 下独立维护的商城推荐工具模块，目标是在不引入独立推荐服务的前提下，复用当前商城业务表、推荐事实表和定时任务能力，重构一套接近 `gorse` 核心推荐链路的推荐工具库。

当前模块定位：

- 不是服务，不提供 HTTP、gRPC、CLI 启动入口。
- 由 `backend` 提供业务数据实现、事实落库和对外接口。
- 由 `recommend` 提供场景化 pipeline、多路召回、统一排序、replacement/fallback、LevelDB 缓存、离线构建和评估。

## 设计目标

- 覆盖 `gorse` 的核心推荐能力：多路召回、排序、replacement/fallback、离线物化、评估、explain。
- 保持商城业务语义清晰，优先围绕 `home`、`goods_detail`、`cart`、`profile`、`order_detail`、`order_paid` 等场景设计。
- 复用现有 `recommend_*` 表，不单独建设通用推荐平台。
- 使用 `LevelDB` 作为推荐缓存与 trace 存储，内部 value 统一由 `recommend.proto` 的 `message` 表达。

## 对外方法

当前模块以包级方法作为唯一对外入口，不使用 `NewXXX` 作为主入口：

- `Recommend(...)`
- `Explain(...)`
- `BuildNonPersonalized(...)`
- `BuildUserCandidate(...)`
- `BuildGoodsRelation(...)`
- `BuildUserToUser(...)`
- `BuildCollaborative(...)`
- `BuildExternal(...)`
- `EvaluateOffline(...)`
- `SyncExposure(...)`
- `SyncBehavior(...)`
- `SyncActorBind(...)`

## 目录说明

```text
recommend
├── contract      # 由 backend 实现的数据契约
├── api           # proto 与生成产物
├── build.go      # 对外离线构建入口
├── evaluate.go   # 对外离线评估入口
├── explain.go    # 对外 explain 入口
├── sync.go       # 对外运行态同步入口
├── types.go      # 对外公共 DTO
├── internal
│   ├── engine      # 推荐、构建、评估总入口
│   ├── scene       # 场景 pipeline
│   ├── recall      # 多路召回
│   ├── rank        # 统一排序
│   ├── replace     # 过滤、补足、打散
│   ├── cache       # LevelDB 缓存
│   ├── materialize # 离线物化
│   ├── evaluate    # 离线评估
│   └── model       # 内部 DTO
└── docs            # 设计文档
```

## 场景

第一阶段统一纳入以下推荐场景：

- `home`
- `goods_detail`
- `cart`
- `profile`
- `order_detail`
- `order_paid`

## 推荐能力映射

当前模块规划覆盖的核心能力如下：

- 非个性化召回：`latest`、`scene_hot`、`global_hot`
- 业务关联召回：`goods_relation`
- 偏好召回：`user_goods_pref`、`user_category_pref`
- 会话召回：`session_context`
- 增强召回：`user_to_user`、`collaborative`、`external`
- 排序：规则排序、预留 `fm`、`llm` 排序扩展位
- replacement：过滤、补足、惩罚、打散
- 离线构建：候选池、运行态、trace
- 评估：离线指标和 explain

## LevelDB 规划

缓存固定分为三类库：

- `pool.db`：候选池、邻居结果、非个性化结果
- `runtime.db`：会话态、惩罚态等短生命周期数据
- `trace.db`：请求 explain 与 trace

缓存 value 统一使用 [recommend.proto](/Users/liujun/workspace/shop/shop/recommend/api/protos/recommend/v1/recommend.proto) 中定义的消息结构序列化。

当前缓存实现默认使用 `LevelDB`，但 `internal/cache` 已经按驱动层拆分：

- `internal/cache/driver`：缓存驱动抽象
- `internal/cache/leveldb`：当前默认实现

后续如果要切 `Redis`，补充新的缓存驱动即可，不需要改推荐对外方法签名。

## 与 backend 的关系

- `backend/service/app/biz` 负责：
  - 推荐接口对外暴露
  - 主体解析
  - 推荐请求、曝光、行为事实落库
- `backend/pkg/job` 负责：
  - 推荐事实表的日统计、偏好重建、商品关联重建、离线评估任务
- `recommend` 负责：
  - 基于上述事实和业务数据生成可直接消费的推荐结果
  - 供 `backend/service/app/biz` 和 `backend/pkg/job` 直接调用

## 对接原则

- `backend/service/app/biz` 直接调用 `recommend.Recommend(...)`、`recommend.SyncExposure(...)`、`recommend.SyncBehavior(...)`、`recommend.SyncActorBind(...)`
- `backend/pkg/job` 直接调用 `recommend.Build*` 和 `recommend.EvaluateOffline(...)`
- 新模块能力完成并完成切流后，删除历史 `backend/pkg/recommend/*`

后续建议的接入点见 [architecture.md](/Users/liujun/workspace/shop/shop/recommend/docs/architecture.md)、[integration.md](/Users/liujun/workspace/shop/shop/recommend/docs/integration.md) 和 [capability-map.md](/Users/liujun/workspace/shop/shop/recommend/docs/capability-map.md)。
