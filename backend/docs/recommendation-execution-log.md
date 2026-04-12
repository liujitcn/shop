# 推荐改造执行记录

本文用于记录“商城专用 Gorse 化推荐改造”的执行进度，保证后续按项推进、逐项确认、按上次停留点继续。

## 记录规则

- 每次只推进一个明确任务项，不并行跨项收尾。
- 代码完成后，先把任务状态改成 `待确认`，等待用户确认。
- 用户确认后，再把该项改成 `已完成`，并启动下一项。
- 用户确认当前任务且明确要求提交时，直接提交当前任务对应代码，并在记录中更新停留点。
- 下次继续前，先看“当前停留点”和“最近一次更新记录”。

## 状态约定

- `未开始`：还没有进入实现。
- `进行中`：当前正在实现。
- `待确认`：代码已完成，等待用户确认是否进入下一项。
- `已完成`：用户已确认，任务正式收口。
- `阻塞`：因环境、历史问题或设计分歧暂停。

## 当前共识

- 前端保留：曝光、点击、浏览。
- 后端回写：收藏、加购、下单、支付。
- 推荐改造顺序：先闭环，再补归因，再补重建与评估，最后做语义适配和 Gorse 能力继承。

## 当前停留点

- 当前阶段：批次 1，后端事实回写。
- 当前任务：`B1-02` 收藏成功后端回写 `COLLECT`。
- 当前状态：`未开始`
- 上次结束位置：`B1-01` 已确认完成，等待进入 `B1-02`。

## 执行清单

| ID | 批次 | 任务 | 主要范围 | 状态 |
| --- | --- | --- | --- | --- |
| B1-01 | 批次 1 | 新增推荐行为后端内部写入入口，供业务服务直接调用 | `backend/service/app/biz/recommend_goods_action.go` | 已完成 |
| B1-02 | 批次 1 | 收藏成功后端回写 `COLLECT` | `backend/service/app/biz/user_collect.go` | 未开始 |
| B1-03 | 批次 1 | 加购成功后端回写 `ADD_CART` | `backend/service/app/biz/user_cart.go` | 未开始 |
| B1-04 | 批次 1 | 下单成功后端回写 `ORDER_CREATE` | `backend/service/app/biz/order_info.go` | 未开始 |
| B1-05 | 批次 1 | 支付成功后端回写 `ORDER_PAY`，并补幂等保护 | `backend/service/app/biz/pay.go` | 未开始 |
| B1-06 | 批次 1 | 删除前端 `COLLECT`、`ADD_CART`、`ORDER_CREATE`、`ORDER_PAY` 上报 | `frontend/app/src/pages/goods/goods.vue` `frontend/app/src/pagesOrder/create/create.vue` `frontend/app/src/pagesOrder/payment/payment.vue` | 未开始 |
| B1-07 | 批次 1 | 批次 1 校验与回归 | `backend` `frontend/app` | 未开始 |
| B2-01 | 批次 2 | 新增 `recommend_request_item` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 未开始 |
| B2-02 | 批次 2 | 推荐请求主表落库时同步写 `recommend_request_item` | `backend/service/app/biz/recommend_request.go` | 未开始 |
| B2-03 | 批次 2 | 新增 `recommend_exposure_item` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 未开始 |
| B2-04 | 批次 2 | 曝光上报时同步写 `recommend_exposure_item` | `backend/service/app/biz/recommend_exposure.go` | 未开始 |
| B2-05 | 批次 2 | 批次 2 校验与回归 | `backend` | 未开始 |
| B3-01 | 批次 3 | 新增 `recommend_actor_bind_log` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 未开始 |
| B3-02 | 批次 3 | 匿名绑定时写入绑定日志 | `backend/service/app/biz/recommend.go` | 未开始 |
| B3-03 | 批次 3 | 实现 `RecommendUserPreferenceRebuild` | `backend/pkg/job/task` `backend/service/app/biz` | 未开始 |
| B3-04 | 批次 3 | 实现 `RecommendGoodsRelationRebuild` | `backend/pkg/job/task` `backend/service/app/biz` | 未开始 |
| B3-05 | 批次 3 | 注册推荐重建任务并校验 | `backend/pkg/job/init.go` `backend/pkg/job/task/task.go` | 未开始 |
| B4-01 | 批次 4 | 新增 `recommend_eval_report` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 未开始 |
| B4-02 | 批次 4 | 新增 `recommend_model_version` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 未开始 |
| B4-03 | 批次 4 | 实现 `RecommendEvalReport` 任务 | `backend/pkg/job/task` | 未开始 |
| B4-04 | 批次 4 | 注册评估任务并校验 | `backend/pkg/job/init.go` `backend/pkg/job/task/task.go` | 未开始 |
| B5-01 | 批次 5 | 定义推荐内部 `User / Item / Feedback / Context` 语义结构 | `backend/pkg/recommend` | 未开始 |
| B5-02 | 批次 5 | 实现业务事实到推荐语义的代码适配层 | `backend/pkg/recommend` `backend/service/app/biz` | 未开始 |
| B5-03 | 批次 5 | 梳理后续 Gorse 能力接入入口 | `backend/docs` `backend/pkg/recommend` | 未开始 |

## 最近一次更新记录

- 2026-04-12：创建执行记录文档；当前停留在 `B1-01`，尚未开始编码。
- 2026-04-12：完成 `B1-01` 代码实现；已新增推荐行为后端内部写入入口，并按确认意见复用现有 `RecommendGoodsActionReportRequest`，未新增写入 DTO，等待用户确认。
- 2026-04-12：`B1-01` 已确认完成；当前停留点推进到 `B1-02`，并增加“确认后可直接提交当前项代码”的记录规则。
