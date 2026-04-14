# 与现有 backend 的衔接

## service/app/biz 推荐链路

以下现有业务对象后续需要持有 `*recommend.Recommend` 实例，而不是继续在业务层内拼推荐流程：

- `backend/service/app/biz/recommend.go`
- `backend/service/app/biz/recommend_request.go`

### 推荐查询

建议后续由这两个位置初始化并复用统一实例：

- `svc.Recommend(...)`

返回：

- 推荐商品 ID 列表
- 召回来源
- 评分明细
- trace ID

说明：

- `recommend` 根包继续对 `backend` 暴露稳定 DTO 和配置类型。
- 根包内部再把请求转换到 `internal/core`，由 `engine/scene/recall` 消费，避免 `backend` 接入层感知内部重构。

随后仍由 `backend` 自己负责：

- 组装商品详情
- 落 `recommend_request`
- 落 `recommend_request_item`
- 返回接口响应

## service/app/biz 运行态同步

以下对象负责曝光、行为和匿名绑定后的事实落库，后续应在落库成功后同步通知推荐工具更新运行态缓存：

- `backend/service/app/biz/recommend_exposure.go`
- `backend/service/app/biz/recommend_goods_action.go`
- `backend/service/app/biz/recommend_actor_bind_log.go`

建议映射的实例方法：

- `svc.SyncExposure(...)`
- `svc.SyncBehavior(...)`
- `svc.SyncActorBind(...)`

同步内容包括：

- `session_context`
- `actor penalty`
- trace 补充
- 用户候选池失效或重建标记

当前实现状态：

- `SyncExposure(...)` 已落 `runtime.db` 的 `exposure_penalty`
- `SyncBehavior(...)` 已落共享/会话态和复购惩罚
- `SyncActorBind(...)` 已归并匿名主体到登录主体的共享会话态与场景惩罚
- `Recommend(...)` 在线排序已读取运行态惩罚
- `Recommend(...)` 在线召回已优先消费匿名通用候选池和用户候选池
- `Recommend(...)` 在线召回已优先消费 `runtime.db` 中的会话态
- `Recommend(...)` 在线召回已优先消费相似用户池中的 user-to-user 商品项
- `Recommend(...)` 在线召回已优先消费商品关联池、协同过滤池、外部推荐池
- `Explain(...)` 已可按 `traceId` / `requestId` 查询

## pkg/job 推荐任务

当前这些任务仍保留在 `backend/pkg/job/task`，但任务执行完成后建议触发推荐缓存构建：

- `recommend_goods_stat_day.go`
- `recommend_user_preference_rebuild.go`
- `recommend_goods_relation_rebuild.go`
- `recommend_eval_report.go`

建议衔接关系：

- 推荐商品日统计完成后：
  - `svc.BuildNonPersonalized(...)`
- 用户偏好重建完成后：
  - `svc.BuildUserCandidate(...)`
  - `svc.BuildUserToUser(...)`
  - `svc.BuildCollaborative(...)`
  - `svc.BuildVector(...)`
- 商品关联重建完成后：
  - `svc.BuildGoodsRelation(...)`
  - `svc.BuildVector(...)`
- 活动池、营销池、人工池重建完成后：
  - `svc.BuildExternal(...)`
- 推荐请求与行为事实累计到一定样本后：
  - `svc.TrainRanking(...)`
- 离线评估报告：
  - `svc.EvaluateOffline(...)`

如果希望把多类离线池重建合并成一次调用，优先走：

- `svc.Rebuild(...)`

当前实现状态：

- `BuildNonPersonalized(...)`、`BuildUserCandidate(...)`、`BuildGoodsRelation(...)`、`BuildUserToUser(...)`、`BuildCollaborative(...)`、`BuildExternal(...)`、`BuildVector(...)` 已可直接把候选结果写入 `pool.db`
- `BuildNonPersonalized(...)`、`BuildUserCandidate(...)` 产出的候选池商品项已保留 `source_scores`，供在线恢复多路排序信号
- `BuildUserToUser(...)` 产出的相似用户池已同时保留邻居列表和 user-to-user 商品项，供在线优先消费
- `BuildGoodsRelation(...)`、`BuildCollaborative(...)`、`BuildExternal(...)`、`BuildVector(...)` 产出的离线池已被在线召回优先消费，池缺失时自动回退事实源
- `TrainRanking(...)` 已可按场景读取 trace 与行为事实训练轻量 FM 模型，并在 `fm` 排序模式下被在线读取
- `EvaluateOffline(...)` 已可直接按天返回场景级 `precision / recall / NDCG / CTR / rate` 指标

## 推荐工具不承担的职责

以下能力继续留在 `backend`，不要搬到 `recommend`：

- 推荐接口 HTTP / gRPC 暴露
- 认证与匿名主体解析
- 事务控制
- 推荐事实主表落库
- 商品、订单、购物车、收藏等业务写操作

## 最终切流

- `backend/service/app/biz` 与 `backend/pkg/job` 完成接入后，推荐主链路以 `shop/recommend` 的实例 API 为唯一工具入口
- 新链路稳定后，删除历史 `backend/pkg/recommend/*`
