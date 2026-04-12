# 推荐链路闭环检查与 Gorse 差距分析

本文基于 `backend/service/app/recommend_service.go` 及其当前工作区关联实现编写，结论时间为 2026-04-12。对照项使用的是当天可访问的 Gorse 官方 README 与文档页面。

阅读说明：

- 本文负责回答“当前链路是否闭环、和 Gorse 还差什么”。
- 如果要看后续应该按什么目标继续推进，请优先阅读 [recommendation-gorse-mall-roadmap.md](recommendation-gorse-mall-roadmap.md)。

## 结论

当前仓库的推荐能力已经形成了一条“基础业务闭环”：

`推荐请求 -> requestId/推荐上下文 -> 曝光/点击/浏览/收藏/加购/下单/支付 -> 原始事件落库 -> 在线聚合用户偏好/商品关系 -> 日统计回流 -> 下一次推荐读取聚合结果`

但它还不是一条“完整工程闭环”或“可运营闭环”。主要缺口在于：

- 缺少离线重建与评估执行器，导致原始数据无法稳定回放校正。
- 匿名主体转登录主体后，只合并了原始表，没有回补画像聚合表。
- 下单、支付转化事件依赖前端主动上报，不是以后端订单事实为准。
- 没有评估看板、实验体系、配置化 pipeline 和统一运维面板。

## 当前链路梳理

### 1. 请求入口与主体识别

- 前端通过 `frontend/app/src/stores/modules/recommend.ts` 获取并缓存匿名主体，统一在请求头透传 `X-Recommend-Anonymous-Id`。
- `backend/pkg/recommend/actor/actor.go` 负责从登录态或匿名请求头解析推荐主体。
- `backend/service/app/recommend_service.go` 只做服务编排，核心逻辑下沉到 `backend/service/app/biz/recommend.go`。

### 2. 推荐请求与召回排序

- `RecommendGoods` 每次生成独立 `requestId`，并保存到 `recommend_request`。
- 匿名态走 `listAnonymousRecommendGoods`：
  - 场景热度 `recommend_goods_stat_day`
  - 全站热度 `goods_stat_day`
  - 新鲜度
  - 场景曝光惩罚
  - 主体曝光惩罚
- 登录态走 `listRecommendGoods`：
  - 购物车 / 订单上下文触发的商品关联召回
  - 用户类目偏好补足
  - latest 兜底
  - 排序信号包括商品关联分、用户商品偏好分、类目偏好分、场景热度、全站热度、新鲜度、曝光惩罚、重复购买惩罚
- 候选构建与打散逻辑集中在 `backend/pkg/recommend/candidate/logic.go`，排序权重集中在代码常量和公式中。

### 3. 推荐上下文透传

- 前端 `frontend/app/src/components/XtxGuess.vue` 会把 `scene + requestId + position` 写入商品点击行为，并在跳转商品详情时带到路由。
- 商品详情、购物车、收藏、下单确认、支付成功页都会继续透传或回填 `recommendContext`。
- 后端会把推荐上下文持久化到：
  - `user_cart`
  - `user_collect`
  - `order_goods`

这一步保证了后续加购、收藏、下单、支付仍然能追溯回最初的推荐请求。

### 4. 曝光与行为采集

- `RecommendExposureReport` 会把曝光事件异步投递到队列，再落到 `recommend_exposure`。
- `RecommendGoodsActionReport` 会把点击、浏览、收藏、加购、下单、支付等行为异步投递到队列，再落到 `recommend_goods_action`。
- 当前行为采集已覆盖：
  - 推荐曝光
  - 推荐点击
  - 商品浏览
  - 商品收藏
  - 商品加购
  - 下单
  - 支付

### 5. 在线聚合

`backend/service/app/biz/recommend_goods_action.go` 在消费行为事件时，会同步更新：

- `recommend_user_goods_preference`
- `recommend_user_preference`
- `recommend_goods_relation`

这里的关键限制是：只有登录主体才会聚合画像与偏好，匿名主体只保留原始行为明细。

### 6. 日统计回流

- `backend/pkg/job/task/recommend_goods_stat_day.go` 会按天重算 `recommend_goods_stat_day`，用于场景热度、场景曝光惩罚等信号。
- `backend/pkg/job/task/goods_stat_day.go` 会按天重算 `goods_stat_day`，用于全站热度信号。
- 下一次推荐请求再次读取这些统计表，形成 T+1 的热度回流。

## 闭环判断

### 已闭合的部分

- 推荐结果有唯一 `requestId`，后续曝光和行为能归因到具体推荐请求。
- 推荐上下文可以跨页面、跨业务动作继续透传到加购、收藏、下单、支付。
- 原始曝光与行为会回流到推荐相关聚合表和统计表。
- 下一次推荐请求会消费画像、商品关系、热度统计结果。

### 未闭合的部分

#### 1. 匿名转登录只合并原始表，没有回补画像表

`BindRecommendAnonymousActor` 当前只更新：

- `recommend_request`
- `recommend_exposure`
- `recommend_goods_action`

但不会回放这些已改绑的匿名行为去重建：

- `recommend_user_goods_preference`
- `recommend_user_preference`
- `recommend_goods_relation`

结果是：匿名阶段积累的浏览、点击、加购，在用户登录后并不会立即进入登录态画像。只有未来补齐“重建任务”或手动重放原始行为后，这部分价值才会回流。

#### 2. 下单与支付事件不是以后端事实为准

当前下单和支付行为由前端页面主动调用 `RecommendGoodsActionReport` 上报。只要前端请求丢失、页面提前关闭、客户端版本未跟进，推荐链路就会丢失高价值转化信号。

这意味着当前链路对“浏览、点击”更稳，对“成交转化”并不稳，严格来说还不是完整闭环。

#### 3. README 和 SQL 已声明的任务，代码尚未实现

当前代码实际注册的推荐任务只有：

- `RecommendGoodsStatDay`

但 `backend/README.md` 和 `sql/default-data.sql` 里已经出现了：

- `RecommendEvalReport`
- `RecommendUserPreferenceRebuild`
- `RecommendGoodsRelationRebuild`

仓库当前没有对应执行器，也没有在任务列表中注册。这说明“数据修复”和“效果评估”这两段闭环仍然是缺失的。

#### 4. 缺少效果评估与运营反馈

当前实现没有看到以下能力：

- 在线正反馈率 / CTR / CVR 看板
- 离线 precision / recall / NDCG / AUC 评估
- 推荐源占比、场景效果、请求级 explain 查询入口
- A/B 实验或策略灰度

所以链路虽然能回流数据，但还不能回答“哪个策略真的更好”。

## 与 Gorse 的差距

以下对照基于 Gorse 官方 README 与文档：

- README: <https://github.com/gorse-io/gorse>
- Pipeline: <https://gorse.io/docs/concepts/pipeline>
- Data Source: <https://gorse.io/docs/concepts/data-source.html>
- Non-personalized: <https://gorse.io/docs/concepts/recommenders/non-personalized>
- Item-to-Item: <https://gorse.io/docs/concepts/recommenders/item-to-item.html>
- User-to-User: <https://gorse.io/docs/concepts/recommenders/user-to-user>
- Ranking: <https://gorse.io/docs/concepts/ranking>
- Evaluation: <https://gorse.io/docs/concepts/evaluation>
- Dashboard: <https://gorse.io/docs/0.4/gorse-dashboard.html>
- External API Recommenders: <https://gorse.io/docs/concepts/recommenders/external.html>

### 1. 召回源不够丰富

当前仓库主要是：

- 场景商品关联
- 用户类目偏好
- 场景热度
- 全站热度
- latest 兜底

而 Gorse 当前官方能力覆盖：

- latest
- 可配置 non-personalized
- item-to-item
- user-to-user
- collaborative filtering
- external recommender

当前仓库虽然已经有“商品关系”能力，但本质上更接近业务内生的共现关联，还没有形成可配置、可替换的多路召回体系。

### 2. 排序层还是人工规则，缺少模型化能力

当前排序公式写死在 `backend/pkg/recommend/candidate/logic.go` 中，属于固定权重规则排序。

Gorse 当前已经提供：

- matrix factorization 召回
- factorization machine ranker
- LLM-based reranker
- 自动训练与模型选择

相比之下，当前仓库缺少：

- 基于训练数据的 CTR/CVR 排序模型
- 模型版本管理
- 自动调参与模型选优
- rerank 层的可插拔能力

### 3. 反馈语义不够标准化

Gorse 的数据源模型中，明确区分：

- positive feedback
- read feedback
- feedback TTL
- hidden item
- user labels / item labels / embedding
- write-back read feedback
- replacement

当前仓库虽然也区分了点击、浏览、收藏、加购、下单、支付，但还缺少：

- 统一的正反馈 / 已读反馈配置层
- 已读自动回写或延迟回写机制
- 商品隐藏、召回过滤、重放 TTL 等标准能力
- 用户和商品的标签、向量、文本等内容侧特征输入

### 4. 缺少完整的 retrieval-ranking-fallback 工程化 pipeline

Gorse 的官方 pipeline 明确包含：

- 多路 retrieval
- ranking
- read filtering
- replacement
- fallback
- 离线缓存结果

当前仓库已经具备 retrieval 和 ranking 的基础形态，但缺少：

- 全量用户离线推荐缓存
- 请求时的统一 fallback 模块
- 已读过滤的标准层
- replacement 策略
- pipeline 配置化与编辑能力

### 5. 缺少评估、看板与运维能力

Gorse 自带：

- dashboard
- 数据导入导出
- 任务状态查看
- 集群状态查看
- 在线评估
- 离线评估

当前仓库没有推荐专用管理面板，也没有推荐专用系统指标，这会直接影响后续运营调优效率。

### 6. 缺少分布式推荐执行架构

Gorse 的 master / worker / server 拆分，本质上解决了：

- 模型训练
- 全量离线推荐生成
- 在线服务扩容

当前仓库推荐逻辑全部内嵌在业务后端内，适合当前阶段快速迭代，但还不具备大规模推荐服务的拆分能力。

## 优先级建议

如果目标是“先把当前链路真正闭合”，建议优先做下面三项：

1. 把下单、支付推荐行为改成以后端订单事实自动回写，不再依赖前端埋点是否成功。
2. 实现 `RecommendUserPreferenceRebuild`、`RecommendGoodsRelationRebuild`、`RecommendEvalReport`，补齐数据修复和效果评估。
3. 在匿名绑定登录后，增加原始行为重放或增量补偿逻辑，确保匿名行为能进入登录态画像。

如果目标是“向 Gorse 靠拢”，第二阶段再考虑：

1. 引入配置化召回源和 fallback。
2. 引入用户 / 商品标签、文本或 embedding 特征。
3. 把固定权重排序升级为可训练的 ranker。
4. 增加推荐看板、指标和实验体系。

## 当前判断

一句话总结：

当前实现已经是“能跑、能归因、能回流”的推荐基础闭环，但还不是“可重建、可评估、可运营、可扩展”的完整推荐系统；和 2026-04-12 可访问的 Gorse 官方能力相比，主要差在模型化、多路召回、评估体系、运维面板和工程化 pipeline。
