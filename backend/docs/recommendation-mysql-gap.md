# 推荐系统 MySQL 表结构差距记录

本文只讨论当前仓库在 `MySQL` 场景下的推荐表结构，不讨论 Gorse 当前使用 `SQLite` 的 `meta` 表。

结论时间：2026-04-12

阅读方式说明：

- 如果你的目标是“商城专用 Gorse 化推荐服务”，请优先看 [recommendation-gorse-mall-roadmap.md](recommendation-gorse-mall-roadmap.md)。
- 本文保留的是“从 MySQL 表结构角度看，潜在还缺什么”的记录清单。
- 本文里的缺口不代表都要一次性补齐，尤其不代表必须先把 Gorse 的通用存储层完整复制过来。

## 结论

当前仓库已经有一套可运行的自研推荐表结构，但从“表结构可闭环”和“对接 Gorse MySQL 数据层”两个角度看，仍然缺少一组关键表或标准化落库层。

如果按“商城专用 Gorse 化”目标裁剪，应该这样理解：

- 必须补的是推荐域自己的归因、模型、评估表。
- 可以后补的是 Gorse 风格的标准化兼容层。
- 可以暂时不做的是 Gorse 通用缓存表。

如果只看当前自研链路，最缺的是：

- 请求结果明细表
- 曝光明细表
- 匿名主体绑定日志 / 重放锚点表
- 评估 / 模型 / 策略快照表

如果看 Gorse 的 MySQL 数据接口，最缺的是：

- 标准化用户表
- 标准化物品表
- 标准化反馈表

如果未来要把 MySQL 同时当缓存存储，还会再缺：

- `values`
- `message`
- `documents`
- `time_series_points`

## 当前已有表

### 1. 原始请求与行为表

- `recommend_request`
- `recommend_exposure`
- `recommend_goods_action`

### 2. 业务链路透传表

- `user_cart`
- `user_collect`
- `order_goods`

### 3. 在线聚合表

- `recommend_user_preference`
- `recommend_user_goods_preference`
- `recommend_goods_relation`

### 4. 统计表

- `recommend_goods_stat_day`
- `goods_stat_day`

### 5. 基础实体表

- `base_user`
- `goods_info`

这些表已经足够支撑当前“请求 -> 曝光/行为 -> 聚合 -> 次日回流”的基础推荐链路，但还不够支撑高质量归因、离线重建、效果评估和 Gorse 兼容接入。

## Gorse 在 MySQL 下的核心表

按 Gorse 当前 `master` 分支源码，MySQL 数据存储核心只建 3 张表：

- `users`
- `items`
- `feedback`

字段语义分别是：

- `users`：`user_id`、`labels`、`comment`
- `items`：`item_id`、`is_hidden`、`categories`、`time_stamp`、`labels`、`comment`
- `feedback`：`feedback_type`、`user_id`、`item_id`、`value`、`time_stamp`、`updated`、`comment`

如果 MySQL 还承担缓存存储，Gorse 还会建 4 张 SQL 缓存表：

- `values`
- `message`
- `documents`
- `time_series_points`

参考：

- <https://github.com/gorse-io/gorse/blob/master/storage/data/sql.go>
- <https://github.com/gorse-io/gorse/blob/master/storage/cache/sql.go>

## 当前表结构缺什么

## 一. 当前自研推荐链路最缺的表

### 1. `recommend_request_item`

当前问题：

- `recommend_request` 只把 `goods_ids` 整批存成 JSON。
- `recall_sources` 也是整批 JSON，不是逐商品明细。
- 排序打分明细只存在运行时 `sourceContext`，没有单商品落库。

直接影响：

- 无法直接按 `request_id + goods_id + position` 做 MySQL 级别分析。
- 无法高效统计某个召回源、某个排序信号、某个位置的 CTR / CVR。
- 无法稳定回放当时到底给用户返回了哪些商品和对应分数。

建议补表字段：

- `request_id`
- `actor_type`
- `actor_id`
- `scene`
- `goods_id`
- `position`
- `recall_source`
- `final_score`
- `relation_score`
- `user_goods_score`
- `profile_score`
- `scene_popularity_score`
- `global_popularity_score`
- `freshness_score`
- `exposure_penalty`
- `actor_exposure_penalty`
- `repeat_penalty`
- `created_at`

### 2. `recommend_exposure_item`

当前问题：

- `recommend_exposure` 只把一次曝光批次的 `goods_ids` 存成 JSON。
- 曝光不是逐商品逐位置行存储。

直接影响：

- 商品级曝光、位置级曝光分析都要先拆 JSON。
- 曝光和点击的 SQL Join 成本高，归因不稳定。
- 无法方便地做“第几位曝光后点击率”分析。

建议补表字段：

- `request_id`
- `actor_type`
- `actor_id`
- `scene`
- `goods_id`
- `position`
- `created_at`

### 3. `recommend_actor_bind_log`

当前问题：

- 匿名绑定登录时，当前实现是直接改写 `recommend_request`、`recommend_exposure`、`recommend_goods_action`。
- 没有一张独立的绑定日志表记录“谁在什么时候被绑定到谁”。

直接影响：

- 后续很难做补偿重放。
- 很难审计某次画像为什么突然变化。
- 如果未来引入离线训练或 Gorse，同一匿名主体是否已经并入登录主体没有稳定事实表。

建议补表字段：

- `anonymous_id`
- `user_id`
- `bind_source`
- `bind_status`
- `created_at`

### 4. `recommend_eval_report`

当前问题：

- README 和 `sql/default-data.sql` 已经出现 `RecommendEvalReport` 任务名。
- 代码里没有对应执行器，也没有结果表。

直接影响：

- 没有表结构承接离线评估结果。
- Precision / Recall / NDCG / CTR / CVR 等指标无法稳定沉淀。

建议补表字段：

- `report_date`
- `scene`
- `strategy_name`
- `sample_size`
- `request_count`
- `exposure_count`
- `click_count`
- `order_count`
- `pay_count`
- `ctr`
- `cvr`
- `ndcg`
- `precision_score`
- `recall_score`
- `extra_json`
- `created_at`

### 5. `recommend_model_version` 或 `recommend_strategy_snapshot`

当前问题：

- 当前排序权重全写在代码里。
- 没有一张表记录每次线上策略或模型版本。

直接影响：

- 无法回答某天效果变动对应哪套排序权重。
- 无法做策略回滚、灰度或效果对比。

建议补表字段：

- `model_name`
- `model_type`
- `version`
- `scene`
- `config_json`
- `status`
- `published_at`
- `created_at`

## 二. 对接 Gorse MySQL 时最缺的标准化表

这里说的“缺”，不是指当前业务没有用户、商品、行为数据，而是指当前没有一层能直接按 Gorse MySQL 语义喂数的标准化表或视图。

### 1. 标准化用户表

可以叫：

- `recommend_mysql_user`
- 或直接 `gorse_users`

当前映射来源主要是 `base_user`，但还缺：

- `labels`
- `comment`
- 稳定的推荐用户 ID 映射规则

原因：

- `base_user` 只有账号、昵称、手机号、性别、角色等基础字段。
- 当前没有把这些字段整理成 Gorse 风格的 `labels` JSON 数组。

### 2. 标准化商品表

可以叫：

- `recommend_mysql_item`
- 或直接 `gorse_items`

当前映射来源主要是 `goods_info`，但还缺：

- `is_hidden`
- `categories` 的 JSON 数组表达
- `labels`
- `comment`
- 明确用于推荐的 `time_stamp`

原因：

- `goods_info` 当前只有单值 `category_id`，没有标准化 `categories[]`。
- 也没有商品标签数组、推荐备注、隐藏态语义字段。

### 3. 标准化反馈表

可以叫：

- `recommend_mysql_feedback`
- 或直接 `gorse_feedback`

当前映射来源主要会来自：

- `recommend_goods_action`
- `recommend_exposure`
- `order_goods`

但还缺：

- 标准 `feedback_type`
- 统一 `value`
- `updated`
- `comment`
- 稳定的 `user_id`

原因：

- `recommend_goods_action` 已有 `event_type`，但没有标准 `value` 和 `updated`。
- 匿名主体与登录主体混在 `actor_type + actor_id` 中，不能直接等价成 Gorse 的 `user_id`。
- 当前也没有一层表明确区分“只导出登录主体”还是“匿名主体也作为 user_id 导出”。

## 三. MySQL 缓存层当前缺的表

只有在未来明确把 MySQL 同时当作推荐缓存存储时，这部分才需要补。

### 1. `recommend_cache_values`

用于键值缓存。

### 2. `recommend_cache_message`

用于消息或事件型缓存。

### 3. `recommend_cache_documents`

用于文档型检索缓存。

### 4. `recommend_cache_time_series_points`

用于时序指标与监控点写入。

当前自研推荐系统没有这些 MySQL 缓存表，说明现在还不是“带缓存层的通用推荐基础设施”，而是“内嵌业务后端的推荐实现”。

## 当前表字段层面的核心缺口

### 1. `base_user` 缺推荐标签字段

当前只有：

- `nick_name`
- `gender`
- `role_id`
- `dept_id`
- `phone`

缺：

- `labels`
- `profile_json`
- `comment`

### 2. `goods_info` 缺推荐内容特征字段

当前只有：

- `category_id`
- `name`
- `desc`
- `detail`
- `price`
- `status`
- `created_at`
- `updated_at`

缺：

- `is_hidden`
- `categories[]`
- `labels[]`
- `recommend_comment`
- 内容向量或向量外键

### 3. `recommend_goods_action` 缺标准反馈语义字段

当前只有：

- `event_type`
- `goods_id`
- `goods_num`
- `scene`
- `request_id`
- `position`
- `created_at`

缺：

- `value`
- `updated`
- `source`
- `channel`
- `device`
- `comment`

### 4. `recommend_request` / `recommend_exposure` 过度依赖 JSON 批量字段

当前：

- `recommend_request.goods_ids` 是 JSON
- `recommend_request.recall_sources` 是 JSON
- `recommend_exposure.goods_ids` 是 JSON

这类字段适合快速落地，但不适合 MySQL 作为长期分析底座。

## 建议优先级

### P0

- 新增 `recommend_request_item`
- 新增 `recommend_exposure_item`
- 新增 `recommend_actor_bind_log`
- 新增 `recommend_eval_report`

### P1

- 新增 `recommend_mysql_user`
- 新增 `recommend_mysql_item`
- 新增 `recommend_mysql_feedback`
- 给 `goods_info` 和 `base_user` 补推荐标签或特征字段

### P2

- 新增 `recommend_model_version` / `recommend_strategy_snapshot`
- 视是否把 MySQL 当缓存层，再补 `values`、`message`、`documents`、`time_series_points`

## 一句话判断

只看当前 MySQL 表结构，现有推荐系统“不缺能跑起来的基础表”，但明显缺“可分析、可重放、可评估、可对接 Gorse MySQL”的标准化表层。
