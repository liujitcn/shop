# TMP-DEVELOPMENT-PLAN

> 临时开发记录。
> 当前文件只用于开发过程中衔接上下文，开发完成并完成切流后删除。
> 当前文件不是正式业务文档，不加入最终业务说明体系。
> 最后更新时间：2026-04-14

## 当前目标

在 `shop/recommend` 下建设一个推荐工具模块，供 `backend/service/app/biz` 和 `backend/pkg/job` 直接调用。

明确约束：

- `recommend` 是工具，不是服务
- 不新增 `cmd`、`server`、`wire`、`bootstrap`
- 复用现有 `recommend_*` 表和商城业务表
- 缓存默认使用纯 Go `LevelDB`
- 当前 `LevelDB` 实现使用 `github.com/syndtr/goleveldb`
- `LevelDB` value 统一走 `api/protos/recommend/v1/recommend.proto`
- 缓存驱动保留抽象层，后续可增补 `Redis` 实现
- 新链路稳定后删除旧 `backend/pkg/recommend/*`

## 今天已经完成

### 1. 模块初始化已完成

已创建并落库：

- `recommend/AGENTS.md`
- `recommend/go.mod`
- `recommend/Makefile`
- `recommend/README.md`
- `recommend/recommend.go`
- `recommend/types.go`
- `recommend/build.go`
- `recommend/explain.go`
- `recommend/evaluate.go`
- `recommend/sync.go`

### 2. 协议与设计文档已完成

已创建并生成：

- `recommend/api/protos/recommend/v1/recommend.proto`
- `recommend/api/gen/go/...`

已补齐设计文档：

- `recommend/docs/architecture.md`
- `recommend/docs/pipeline.md`
- `recommend/docs/integration.md`
- `recommend/docs/capability-map.md`

### 3. 开发规则已同步到模块内

已同步到 `recommend/AGENTS.md`：

- 注释规范
- 代码修改约束
- 变量命名规则
- Proto 接口契约命名规则

已处理：

- 删除 Go 文件上的包注释，避免出现 `Package recommend ...` 形式的 lint 提示

### 4. 缓存层已完成首版实现

已完成：

- `internal/cache/driver/types.go`
- `internal/cache/leveldb/manager.go`
- `internal/cache/leveldb/keys.go`
- `internal/cache/leveldb/codec.go`
- `internal/cache/leveldb/batch.go`
- `internal/cache/pool_store.go`
- `internal/cache/runtime_store.go`
- `internal/cache/trace_store.go`
- `internal/cache/store_test.go`

当前状态：

- 已经具备 `pool.db`、`runtime.db`、`trace.db` 的读写能力
- 外层已经通过缓存驱动接口隔离，后续切 `Redis` 时不需要重写推荐主链路

### 5. 推荐内部模型已完成首版实现

已完成：

- `internal/model/actor.go`
- `internal/model/request.go`
- `internal/model/candidate.go`
- `internal/model/score.go`
- `internal/model/scene.go`
- `internal/model/trace.go`

### 6. 排序层已完成首版实现

已完成：

- `internal/rank/types.go`
- `internal/rank/weights.go`
- `internal/rank/scorer.go`
- `internal/rank/ranker.go`
- `internal/rank/scorer_test.go`
- `internal/rank/ranker_test.go`

### 7. 召回层已完成首版实现

已完成：

- `internal/recall/types.go`
- `internal/recall/latest.go`
- `internal/recall/scene_hot.go`
- `internal/recall/global_hot.go`
- `internal/recall/goods_relation.go`
- `internal/recall/user_goods_pref.go`
- `internal/recall/user_category_pref.go`
- `internal/recall/session_context.go`
- `internal/recall/user_to_user.go`
- `internal/recall/collaborative.go`
- `internal/recall/external.go`
- `internal/recall/recall_test.go`

覆盖方向：

- 非个性化召回
- 商品关联召回
- 用户偏好召回
- 会话上下文召回
- user-to-user 召回
- collaborative 召回
- external 召回

### 8. 后处理层已完成首版实现

已完成：

- `internal/replace/filter.go`
- `internal/replace/penalty.go`
- `internal/replace/fallback.go`
- `internal/replace/diversify.go`
- `internal/replace/replace_test.go`

### 9. 场景流水线已完成首版实现

已完成：

- `internal/scene/registry.go`
- `internal/scene/scene.go`
- `internal/scene/home.go`
- `internal/scene/goods_detail.go`
- `internal/scene/cart.go`
- `internal/scene/profile.go`
- `internal/scene/order_detail.go`
- `internal/scene/order_paid.go`

当前已经覆盖的业务场景：

- 首页推荐
- 商品详情页推荐
- 购物车推荐
- 个人中心推荐
- 订单详情推荐
- 支付完成推荐

### 10. 引擎层已完成内部首版实现

已完成：

- `internal/engine/recommend.go`
- `internal/engine/explain.go`
- `internal/engine/build.go`
- `internal/engine/recommend_test.go`

当前状态：

- 内部主链路已经能串起 `scene -> recall -> rank -> replace -> result`
- 对外入口接线过程中出现依赖环，还没有收口完成

## 当前未完成

以下事项仍未完成：

- 根包 `Recommend(...)` 正式接通
- `SyncExposure(...)`
- `SyncBehavior(...)`
- `SyncActorBind(...)`
- explain / trace 持久化联动
- materialize 离线构建能力落地
- evaluate 离线评估能力落地
- `backend/service/app/biz` 接入新模块
- `backend/pkg/job` 接入新模块
- 删除旧 `backend/pkg/recommend/*`

## 当前阻塞点

### 阻塞问题：根包与内部引擎出现 import cycle

当前执行 `go test ./...` 失败，错误链路如下：

```text
recommend
-> recommend/internal/engine
-> recommend
```

触发点：

- `sync.go` 中根包开始直接引用 `internal/engine`
- 同时 `internal/engine`、`internal/model`、`internal/scene`、`internal/recall` 里仍然直接依赖根包 `recommend` 中的对外类型

当前最明显的循环链路：

- `recommend/sync.go`
- `recommend/internal/engine/recommend.go`
- `recommend/internal/model/*.go`
- `recommend/internal/scene/*.go`
- `recommend/internal/recall/*.go`

## 明天第一步怎么接着做

优先按下面顺序处理：

### 1. 先拆掉根包和内部包的共享类型耦合

建议做法：

- 新建一个内部共享类型目录，例如 `internal/app` 或 `internal/types`
- 把内部运行需要的 `request/result/context/dependencies` 迁过去
- 让 `internal/engine`、`internal/model`、`internal/scene`、`internal/recall` 只依赖内部共享类型

### 2. 根包只保留对外适配职责

目标：

- 根包 `recommend` 只做对外方法和对外类型暴露
- 根包负责把外部请求转换为内部请求
- 根包负责把内部结果转换为外部结果

### 3. 拆完依赖环后先恢复测试

依次执行：

```bash
make fmt
go test ./...
```

### 4. 测试恢复后再继续补主入口

继续实现：

- `Recommend(...)`
- `SyncExposure(...)`
- `SyncBehavior(...)`
- `SyncActorBind(...)`

### 5. 再进入离线构建和评估

继续实现：

- `internal/materialize/*`
- `internal/evaluate/*`

### 6. 最后接入 backend 并删除旧模块

接入顺序：

1. `backend/service/app/biz`
2. `backend/pkg/job`
3. 删除 `backend/pkg/recommend/*`

## 当前测试状态

2026-04-14 在 `recommend` 目录执行：

```bash
go test ./...
```

当前结果：

- 失败
- 原因是 import cycle

补充说明：

- 在根包接入 `internal/engine` 之前，缓存、排序、召回、后处理相关测试是通过的
- 当前优先级不是回退代码，而是先完成内部 DTO 拆分，彻底解掉依赖环

## 下次恢复开发先看这几个文件

- `recommend/AGENTS.md`
- `recommend/TMP-DEVELOPMENT-PLAN.md`
- `recommend/README.md`
- `recommend/docs/architecture.md`
- `recommend/docs/pipeline.md`
- `recommend/docs/integration.md`
- `recommend/sync.go`
- `recommend/internal/engine/recommend.go`
- `recommend/internal/model/request.go`
- `recommend/internal/scene/scene.go`
- `recommend/internal/recall/types.go`

## 当前可用命令

在 `recommend` 目录执行：

```bash
make proto
make fmt
go test ./...
```
