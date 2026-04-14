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

- 当前阶段：批次 4，评估任务与版本记录。
- 当前任务：`B5-01` 定义推荐内部 `User / Item / Feedback / Context` 语义结构。
- 当前状态：`未开始`
- 上次结束位置：`B4-03`、`B4-04` 已确认完成，下一步从 `B5-01` 开始。

## 执行清单

| ID | 批次 | 任务 | 主要范围 | 状态 |
| --- | --- | --- | --- | --- |
| B1-01 | 批次 1 | 新增推荐行为后端内部写入入口，供业务服务直接调用 | `backend/service/app/biz/recommend_goods_action.go` | 已完成 |
| B1-02 | 批次 1 | 收藏成功后端回写 `COLLECT` | `backend/service/app/biz/user_collect.go` | 已完成 |
| B1-03 | 批次 1 | 加购成功后端回写 `ADD_CART` | `backend/service/app/biz/user_cart.go` | 已完成 |
| B1-04 | 批次 1 | 下单成功后端回写 `ORDER_CREATE` | `backend/service/app/biz/order_info.go` | 已完成 |
| B1-05 | 批次 1 | 支付成功后端回写 `ORDER_PAY`，并补幂等保护 | `backend/service/app/biz/pay.go` | 已完成 |
| B1-06 | 批次 1 | 删除前端 `COLLECT`、`ADD_CART`、`ORDER_CREATE`、`ORDER_PAY` 上报 | `frontend/app/src/pages/goods/goods.vue` `frontend/app/src/pagesOrder/create/create.vue` `frontend/app/src/pagesOrder/payment/payment.vue` | 已完成 |
| B1-07 | 批次 1 | 批次 1 校验与回归 | `backend` `frontend/app` | 已完成 |
| B2-01 | 批次 2 | 新增 `recommend_request_item` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 已完成 |
| B2-02 | 批次 2 | 推荐请求主表落库时同步写 `recommend_request_item` | `backend/service/app/biz/recommend_request.go` | 已完成 |
| B2-03 | 批次 2 | 新增 `recommend_exposure_item` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 已完成 |
| B2-04 | 批次 2 | 曝光上报时同步写 `recommend_exposure_item` | `backend/service/app/biz/recommend_exposure.go` | 已完成 |
| B2-05 | 批次 2 | 批次 2 校验与回归 | `backend` | 已完成 |
| B3-01 | 批次 3 | 新增 `recommend_actor_bind_log` 生成代码 | `backend/pkg/gen` | 已完成 |
| B3-02 | 批次 3 | 匿名绑定时写入绑定日志 | `backend/service/app/biz/recommend.go` | 已完成 |
| B3-03 | 批次 3 | 实现 `RecommendUserPreferenceRebuild` | `backend/pkg/job/task` `backend/service/app/biz` | 已完成 |
| B3-04 | 批次 3 | 实现 `RecommendGoodsRelationRebuild` | `backend/pkg/job/task` `backend/service/app/biz` | 已完成 |
| B3-05 | 批次 3 | 注册推荐重建任务并校验 | `backend/pkg/job/init.go` `backend/pkg/job/task/task.go` | 已完成 |
| B4-01 | 批次 4 | 新增 `recommend_eval_report` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 已完成 |
| B4-02 | 批次 4 | 新增 `recommend_model_version` 表结构与生成代码 | `sql/default-data.sql` `backend/pkg/gen` | 已完成 |
| B4-03 | 批次 4 | 实现 `RecommendEvalReport` 任务 | `backend/pkg/job/task` | 已完成 |
| B4-04 | 批次 4 | 注册评估任务并校验 | `backend/pkg/job/init.go` `backend/pkg/job/task/task.go` | 已完成 |
| B5-01 | 批次 5 | 定义推荐内部 `User / Item / Feedback / Context` 语义结构 | `backend/pkg/recommend` | 未开始 |
| B5-02 | 批次 5 | 实现业务事实到推荐语义的代码适配层 | `backend/pkg/recommend` `backend/service/app/biz` | 未开始 |
| B5-03 | 批次 5 | 梳理后续 Gorse 能力接入入口 | `backend/docs` `backend/pkg/recommend` | 未开始 |

## 最近一次更新记录

- 2026-04-12：创建执行记录文档；当前停留在 `B1-01`，尚未开始编码。
- 2026-04-12：完成 `B1-01` 代码实现；已新增推荐行为后端内部写入入口，并按确认意见复用现有 `RecommendGoodsActionReportRequest`，未新增写入 DTO，等待用户确认。
- 2026-04-12：`B1-01` 已确认完成；当前停留点推进到 `B1-02`，并增加“确认后可直接提交当前项代码”的记录规则。
- 2026-04-12：完成 `B1-02` 代码实现；收藏成功后改为后端回写 `COLLECT`，并补了推荐上下文空值保护，等待用户确认。
- 2026-04-12：`B1-02` 已确认完成；当前停留点推进到 `B1-03`。
- 2026-04-12：完成 `B1-03`、`B1-04`、`B1-05` 代码实现；已将加购、下单、支付成功改为后端事实回写，并把支付成功回写收敛为“首次从待支付进入已支付”才写入 `ORDER_PAY`，当前停在 `B1-06` 前等待确认。
- 2026-04-12：已执行 `backend/make wire` 与 `backend/go test ./...`，结果通过。
- 2026-04-12：`B1-03`、`B1-04`、`B1-05` 已确认完成；开始推进 `B1-06`。
- 2026-04-12：完成 `B1-06` 代码实现；已删除商城前端对 `COLLECT`、`ADD_CART`、`ORDER_CREATE`、`ORDER_PAY` 的主动上报，等待确认。
- 2026-04-12：已执行 `frontend/app/pnpm lint` 与 `frontend/app/pnpm tsc`，结果通过。
- 2026-04-12：`B1-06` 已确认完成；当前停留点推进到 `B1-07`。
- 2026-04-12：已执行 `backend/go test ./...`、`frontend/app/pnpm lint` 与 `frontend/app/pnpm tsc`，结果通过；`B1-07` 已确认完成，当前停留点推进到 `B2-01`。
- 2026-04-12：完成 `B2-01` 至 `B2-04` 代码实现；已新增 `recommend_request_item`、`recommend_exposure_item`，主表改为瘦身并通过主表 ID 关联 item 表，同时将推荐请求、曝光、行为归因和推荐统计任务切到 item 表。
- 2026-04-12：已执行 `backend/make wire` 与 `backend/go test ./...`，结果通过；当前停留点推进到 `B2-05`，等待确认。
- 2026-04-13：`B2-01` 至 `B2-05` 已确认完成；执行记录已收口，当前停留点推进到 `B3-01`。
- 2026-04-13：完成 `B3-01` 代码实现；已新增 `recommend_actor_bind_log` 最小事实表，仅保留 `anonymous_id`、`user_id`、`created_at`，同步本地 `shop_test` 建表，并重新生成 `backend/pkg/gen`，等待确认。
- 2026-04-13：`B3-01` 已确认完成；按确认意见移除 `sql/default-data.sql` 中相关建表语句，本次仅提交 `backend/pkg/gen` 生成代码，当前停留点推进到 `B3-02`。
- 2026-04-13：完成 `B3-02` 代码实现；匿名主体归并到登录用户时，已在同一事务内写入 `recommend_actor_bind_log`，等待确认。
- 2026-04-13：按确认意见新增 `RecommendActorBindLogCase`，统一收敛绑定日志保存、用户偏好重建、商品关联重建，并补齐 `RecommendUserPreferenceRebuild`、`RecommendGoodsRelationRebuild` 任务注册与初始化脚本，当前停留点推进到 `B3-05`。
- 2026-04-14：`B3-02` 至 `B3-05` 已确认完成；执行记录已推进到 `B4-01`。
- 2026-04-14：完成 `B4-01` 代码实现；已在本地 `shop_test` 建立 `recommend_eval_report`，同步更新 `sql/default-data.sql`，并通过 `make gorm-gen` 生成 `backend/pkg/gen` 产物，等待确认。
- 2026-04-14：`B4-01` 已确认完成；当前停留点推进到 `B4-02`。
- 2026-04-14：完成 `B4-02` 代码实现；已在本地 `shop_test` 建立 `recommend_model_version`，补齐 `sql/default-data.sql` 中 `recommend_eval_report` 与 `recommend_model_version` 建表语句，并按确认意见移除 `published_at` 后通过 `make gorm-gen` 生成 `backend/pkg/gen` 产物，等待确认。
- 2026-04-14：`B4-02` 已确认完成；当前停留点推进到 `B4-03`。
- 2026-04-14：完成 `B4-03`、`B4-04` 代码实现；已新增 `RecommendEvalReport` 任务，按天汇总请求、曝光、点击、下单、支付与 Precision / Recall / NDCG 指标，并同步注册任务、补齐默认任务脚本与 README，等待确认。
- 2026-04-14：`B4-03`、`B4-04` 已确认完成；按确认意见不保留初始化脚本，当前停留点推进到 `B5-01`。
