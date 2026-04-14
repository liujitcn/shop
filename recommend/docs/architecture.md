# 架构设计

## 模块定位

`recommend` 是 `shop` 的独立推荐工具模块，不是服务。推荐工具只负责推荐逻辑、缓存、离线构建和评估，不负责：

- 对前端暴露 HTTP、gRPC 接口
- 认证、鉴权、事务
- 推荐事实主表落库
- 商品、订单、购物车、收藏等业务主流程

当前模块对外只暴露包级方法，不提供 `NewXXX` 构造器作为主入口。

## 总体职责

### backend 负责

- `backend/service/app/biz` 解析主体、场景和上下文
- 推荐请求、曝光、行为事实继续落现有 `recommend_*` 表
- 商品详情组装、接口返回、业务事务控制

### recommend 负责

- 基于业务事实和推荐事实执行场景化推荐
- 构建多路召回候选
- 统一排序
- 执行 replacement/fallback/diversify
- 将离线结果、运行态和 trace 持久化到 LevelDB

## 对外 API

### 在线能力

- `Recommend(...)`
- `Explain(...)`
- `SyncExposure(...)`
- `SyncBehavior(...)`
- `SyncActorBind(...)`

### 离线能力

- `BuildNonPersonalized(...)`
- `BuildUserCandidate(...)`
- `BuildGoodsRelation(...)`
- `BuildUserToUser(...)`
- `BuildCollaborative(...)`
- `BuildExternal(...)`
- `EvaluateOffline(...)`

## 目录职责

### 根包

- `recommend.go`：根包与错误定义
- `build.go`：离线构建入口
- `explain.go`：explain 入口
- `evaluate.go`：离线评估入口
- `sync.go`：运行态同步入口
- `types.go`：公共 DTO、请求与结果结构

### contract

由 `backend` 实现的数据契约。当前模块不直接依赖 `backend` 内部 repo 或 biz，只依赖这组公开接口。

- `goods.go`
- `user.go`
- `order.go`
- `behavior.go`
- `recommend.go`
- `cache.go`

### internal/engine

推荐总入口，负责把场景、召回、排序、替换、缓存、explain 串起来。

### internal/scene

按商城业务场景组织 pipeline：

- `home`
- `goods_detail`
- `cart`
- `profile`
- `order_detail`
- `order_paid`

### internal/recall

按商城业务信号组织召回器：

- `latest`
- `scene_hot`
- `global_hot`
- `goods_relation`
- `user_goods_pref`
- `user_category_pref`
- `session_context`
- `user_to_user`
- `collaborative`
- `external`

### internal/rank

统一打分排序，第一阶段优先规则排序，同时预留 `fm`、`llm` 扩展位。

### internal/replace

负责：

- 上下架过滤
- 库存过滤
- 当前上下文过滤
- 惩罚与降权
- fallback 兜底
- 类目/品牌/价格带打散

### internal/cache

基于 LevelDB 的缓存存储。当前固定三类库：

- `pool.db`
- `runtime.db`
- `trace.db`

当前缓存层已经拆为两层：

- `internal/cache/driver`
  - 定义缓存驱动抽象
- `internal/cache/leveldb`
  - 当前默认实现

这样后续如果要增加 `redis` 驱动，可以只替换缓存驱动层，不改推荐主链路接口。

### internal/materialize

离线物化缓存，负责从现有业务表和 `recommend_*` 表生成在线可直接消费的缓存结果。

### internal/evaluate

离线评估能力，复用当前请求、曝光、行为事实表的评估口径。

## 代码结构

```text
recommend
├── recommend.go
├── types.go
├── build.go
├── explain.go
├── evaluate.go
├── sync.go
├── contract
│   ├── goods.go
│   ├── user.go
│   ├── order.go
│   ├── behavior.go
│   ├── recommend.go
│   └── cache.go
├── internal
│   ├── engine
│   ├── scene
│   ├── recall
│   ├── rank
│   ├── replace
│   ├── cache
│   ├── materialize
│   ├── evaluate
│   └── model
└── docs
```

## 切流原则

- `backend/service/app/biz` 和 `backend/pkg/job` 直接调用 `shop/recommend`
- 不再围绕旧 `backend/pkg/recommend` 设计新能力
- 新能力落地并完成切流后，删除旧 `backend/pkg/recommend/*`

## 与 gorse 的关系

当前模块参考 `gorse` 的推荐核心思路，但不复制它的服务化外壳：

- 保留：多路召回、排序、replacement/fallback、离线物化、评估、explain
- 不保留：`master`、`server`、`worker` 进程、dashboard、通用数据平台

换句话说，`recommend` 是 `shop` 的推荐工具库，不是 `gorse` 式独立推荐系统。
