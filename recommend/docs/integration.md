# 与现有 backend 的衔接

## service/app/biz 推荐链路

以下现有业务对象后续需要调用 `recommend` 工具方法，而不是继续在业务层内拼推荐流程：

- `backend/service/app/biz/recommend.go`
- `backend/service/app/biz/recommend_request.go`

### 推荐查询

建议后续由这两个位置调用统一入口：

- `recommend.Recommend(...)`

返回：

- 推荐商品 ID 列表
- 召回来源
- 评分明细
- trace ID

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

建议映射的工具入口：

- `recommend.SyncExposure(...)`
- `recommend.SyncBehavior(...)`
- `recommend.SyncActorBind(...)`

同步内容包括：

- `session_context`
- `actor penalty`
- trace 补充
- 用户候选池失效或重建标记

## pkg/job 推荐任务

当前这些任务仍保留在 `backend/pkg/job/task`，但任务执行完成后建议触发推荐缓存构建：

- `recommend_goods_stat_day.go`
- `recommend_user_preference_rebuild.go`
- `recommend_goods_relation_rebuild.go`
- `recommend_eval_report.go`

建议衔接关系：

- 推荐商品日统计完成后：
  - `recommend.BuildNonPersonalized(...)`
- 用户偏好重建完成后：
  - `recommend.BuildUserCandidate(...)`
  - `recommend.BuildUserToUser(...)`
  - `recommend.BuildCollaborative(...)`
- 商品关联重建完成后：
  - `recommend.BuildGoodsRelation(...)`
- 活动池、营销池、人工池重建完成后：
  - `recommend.BuildExternal(...)`
- 离线评估报告：
  - `recommend.EvaluateOffline(...)`

## 推荐工具不承担的职责

以下能力继续留在 `backend`，不要搬到 `recommend`：

- 推荐接口 HTTP / gRPC 暴露
- 认证与匿名主体解析
- 事务控制
- 推荐事实主表落库
- 商品、订单、购物车、收藏等业务写操作

## 最终切流

- `backend/service/app/biz` 与 `backend/pkg/job` 完成接入后，推荐主链路以 `shop/recommend` 为唯一工具入口
- 新链路稳定后，删除历史 `backend/pkg/recommend/*`
