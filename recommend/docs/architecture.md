# 架构设计

## 模块定位

`recommend` 是 `shop` 的独立推荐工具模块，不是服务。推荐工具只负责推荐逻辑、缓存、离线构建和评估，不负责：

- 对前端暴露 HTTP、gRPC 接口
- 认证、鉴权、事务
- 推荐事实主表落库
- 商品、订单、购物车、收藏等业务主流程

当前模块对外以 `recommend.New(...Option)` 创建的 `*Recommend` 实例作为统一入口，不再推荐使用散落的工具函数式调用。

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

- `Rebuild(...)`
- `BuildNonPersonalized(...)`
- `BuildUserCandidate(...)`
- `BuildGoodsRelation(...)`
- `BuildUserToUser(...)`
- `BuildCollaborative(...)`
- `BuildExternal(...)`
- `EvaluateOffline(...)`

## 目录职责

### 根包

- `recommend.go`：实例定义、option、默认配置与依赖装配
- `build.go`：离线构建与 `Rebuild(...)` 入口
- `explain.go`：实例 explain 入口
- `evaluate.go`：实例离线评估入口
- `sync.go`：实例在线推荐与运行态同步入口
- `types.go`：公共 DTO、请求、结果和配置类型别名，对外保持稳定边界

### internal/core

根包和内部执行链路共用的公共类型边界。`engine`、`scene`、`recall`、`model` 统一依赖这一层，根包只做对外适配，不再反向被内部包引用。

### internal/model

`internal/model` 会保留一份与 `internal/core` 接近但不完全相同的内部运行态结构。两者不是无意义重复：

- `internal/core` 负责公开 DTO 和内核共享边界
- `internal/model` 负责归一化后的内部状态、方法和排序中间结果

例如 `Request`、`Actor`、`Scene` 在 `model` 中会额外承载分页偏移、主体判断、场景字符串化这类仅服务内部执行链路的方法。

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

当前已经接通：

- 在线推荐 trace 持久化
- explain 回查转换
- 运行态同步入口
- 运行态曝光惩罚、复购惩罚回灌在线排序
- 离线构建入口到 `pool.db` 的落库
- 离线评估入口到统一指标结果的计算
- 一键 `Rebuild(...)` 对多路离线构建和离线评估的统一编排

### internal/scene

按商城业务场景组织 pipeline：

- `home`
- `goods_detail`
- `cart`
- `profile`
- `order_detail`
- `order_paid`

当前场景层会在单次请求内共享同一个 LevelDB manager，同时复用：

- `pool.db` 的离线候选池
- `runtime.db` 的惩罚态

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
- `vector`

当前已接通的读池优先召回：

- `latest`、`scene_hot`、`global_hot` 优先读取匿名通用候选池，缺失或无法恢复来源分值时回退事实源
- `user_goods_pref`、`user_category_pref` 优先读取用户候选池，缺失或无法恢复来源分值时回退事实源
- `session_context` 优先读取 `runtime.db` 中的具体会话态或共享会话态，缺失时回退行为事实源
- `user_to_user` 优先读取相似用户池中的商品项，缺失时回退事实源
- `goods_relation` 优先读取商品关联池，缺失时回退事实源
- `collaborative` 优先读取协同过滤池，缺失时回退事实源
- `external` 优先读取外部推荐池，缺失时回退事实源

候选池商品项当前会同时保留：

- 合并后的总分
- `source_scores` 形式的来源原始分值

这样在线召回在消费离线合并池时，仍然可以恢复各路排序信号，不会把总分重复计入多路权重。

### internal/rank

统一打分排序，当前已支持实例级场景权重配置、类目打散上限配置、轻量 `fm` 学习排序和 `llm` 二阶段重排，同时保留 `custom` 排序模式入口。

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

当前已经接通：

- 非个性化池构建
- 用户候选池构建
- 商品关联池构建
- 相似用户池构建
- 协同过滤池构建
- 外部推荐池构建
- `Rebuild(...)` 对多类离线池构建的统一调度

### internal/evaluate

离线评估能力，复用当前请求、曝光、行为事实表的评估口径。

当前已经接通：

- 按场景按天拉取请求、曝光、行为事实
- precision / recall / NDCG 计算
- CTR、下单率、支付率计算

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
│   ├── core
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

- `backend/service/app/biz` 和 `backend/pkg/job` 建议各自持有一个 `*recommend.Recommend` 实例
- 不再围绕旧 `backend/pkg/recommend` 设计新能力
- 新能力落地并完成切流后，删除旧 `backend/pkg/recommend/*`

## 与 gorse 的关系

当前模块参考 `gorse` 的推荐核心思路，但不复制它的服务化外壳：

- 保留：多路召回、排序、replacement/fallback、离线物化、评估、explain
- 不保留：`master`、`server`、`worker` 进程、dashboard、通用数据平台

换句话说，`recommend` 是 `shop` 的推荐工具库，不是 `gorse` 式独立推荐系统。
