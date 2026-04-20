# 商城推荐接入 Gorse 从零设计文档

## 1. 文档目标

本文档用于描述当前商城系统接入 Gorse 的一期落地方案。

本次设计明确采用“从 0 设计”的方式，不继承当前仓库中已有的推荐域模型、历史推荐表或半成品实现，只保留对现有移动端接口、页面场景、埋点事件和 Gorse 部署现状的兼容考虑。

本文档覆盖以下内容：

- 推荐系统的一期目标与边界
- 业务库表结构设计
- Gorse 数据模型映射方案
- 推荐请求、曝光、行为、匿名绑定的接入链路
- `recommend.proto` 的修改建议
- 配置项调整建议
- 实施顺序建议

## 2. 现状与约束

### 2.1 当前移动端已具备的推荐能力

当前移动端已接入以下推荐相关能力：

- 匿名登录状态：
  - 首页推荐
  - 商品详情推荐
  - 购物车推荐
  - 我的推荐
- 登录成功后：
  - 首页推荐
  - 商品详情推荐
  - 购物车推荐
  - 我的推荐
  - 支付成功推荐

页面补充说明：

- 匿名用户可以进入购物车页面
- 匿名购物车页不展示购物车商品明细，只展示登录提示和推荐商品区域
- 下单确认页属于登录后场景，但当前页面没有独立推荐位
- 支付成功页属于登录后场景，不需要为匿名用户设计

当前移动端已上报以下埋点事件：

- 曝光
- 点击
- 浏览
- 加入购物车
- 收藏
- 下单
- 支付成功

说明：

- 上述为当前系统已实现或可承接的事件范围
- 一期接入 Gorse 时，订单类事件默认只强制同步 `支付成功`

### 2.2 当前系统约束

- 后端业务库当前使用 [backend/configs/data.yaml](/Users/liujun/workspace/shop/shop/backend/configs/data.yaml) 中的 `shop_test`
- Gorse 当前通过 [gorse/docker-compose.yml](/Users/liujun/workspace/shop/shop/gorse/docker-compose.yml) 启动
- Gorse 当前数据存储配置在 [gorse/config/config.toml](/Users/liujun/workspace/shop/shop/gorse/config/config.toml)，使用独立库 `shop_gorse`
- 当前移动端推荐接口来自 [recommend.proto](/Users/liujun/workspace/shop/shop/backend/api/protos/app/recommend.proto)

### 2.3 一期设计原则

- Gorse 只负责推荐计算和推荐存储，不承担业务留痕和业务归因
- 业务库只保存推荐请求、请求明细、行为事实、匿名绑定
- 匿名用户优先采用会话推荐，不长期把匿名主体当成 Gorse 正式用户
- 业务库与 Gorse 库必须分离，不合并数据库
- 一期先保证链路闭环和可追踪，不先做推荐运营后台、策略发布后台、日报表后台

## 3. 总体架构

推荐链路分为两层：

### 3.1 Gorse 层

Gorse 自身只维护三类核心对象：

- User
- Item
- Feedback

其中：

- User 对应商城登录用户
- Item 对应商城商品
- Feedback 对应曝光、点击、浏览、收藏、加购、下单、支付等行为

### 3.2 业务层

业务后端额外维护以下能力：

- 匿名主体生成与绑定
- 推荐请求留痕
- 推荐结果明细留痕
- 曝光和转化归因
- 向 Gorse 的直接同步调用

## 4. 标识设计

### 4.1 推荐主体设计

业务层统一抽象两类主体：

- 匿名主体
- 登录用户主体

建议约定如下：

- 匿名主体：
  - 业务库中使用 `anonymous_id bigint`
  - 前端继续通过请求头 `X-Recommend-Anonymous-Id` 传递
- 登录用户：
  - 业务库中直接使用 `user_id bigint`
  - Gorse 中使用字符串形式 `user:{userId}`

补充约定：

- 一个 `anonymous_id` 只服务于一次匿名会话到首次绑定用户的衔接过程
- `anonymous_id` 一旦绑定到某个 `user_id`，后续如果用户退出登录并重新进入匿名状态，建议重新生成新的 `anonymous_id`
- 这样可以避免同一匿名主体跨多个账号串联，导致推荐画像污染

### 4.2 商品标识设计

- 业务库中商品主键使用 `goods_id bigint`
- Gorse 中商品标识统一转换为 `goods:{goodsId}`

### 4.3 requestId 设计

- 首次推荐请求由业务后端生成 `request_id`
- 同一推荐会话的后续分页继续复用同一个 `request_id`
- 当前主体、场景、锚点商品或订单发生变化时，重新生成新的 `request_id`
- `recommend_request` 每次请求都新增一条记录，只是同会话继续复用相同 `request_id`
- `request_id` 用于串联：
  - 推荐请求主记录
  - 推荐请求商品明细
  - 曝光上报
  - 点击和后续转化归因

## 5. 一期必须落地的业务表

一期建议只建设以下 4 张业务表：

1. `recommend_anonymous_actor`
2. `recommend_request`
3. `recommend_request_item`
4. `recommend_event`

说明：

- 这 4 张表全部建在业务库 `shop_test`
- Gorse 自身的 `users/items/feedback` 数据仍然存放在 `shop_gorse`
- 一期不建设推荐日报、推荐策略发布、推荐实验分流等表

## 6. 表结构设计

### 6.1 `recommend_anonymous_actor`

用途：

- 记录匿名主体
- 支撑匿名推荐请求和匿名行为留痕
- 支撑登录后匿名主体绑定到正式用户

建议字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 ID |
| `anonymous_id` | bigint | 匿名主体编号 |
| `user_id` | bigint | 绑定用户 ID，未绑定为空 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `bind_at` | datetime | 绑定时间 |

建议索引：

- 唯一索引：`unique_recommend_anonymous_actor (anonymous_id)`
- 普通索引：`idx_recommend_anonymous_actor_user_id (user_id)`
- 普通索引：`idx_recommend_anonymous_actor_updated_at (updated_at)`

建议 DDL：

```sql
CREATE TABLE `recommend_anonymous_actor` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `anonymous_id` bigint NOT NULL COMMENT '匿名主体编号',
  `user_id` bigint DEFAULT NULL COMMENT '绑定用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '更新时间',
  `bind_at` datetime DEFAULT NULL COMMENT '绑定时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `unique_recommend_anonymous_actor` (`anonymous_id`),
  KEY `idx_recommend_anonymous_actor_user_id` (`user_id`),
  KEY `idx_recommend_anonymous_actor_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='推荐匿名主体信息';
```

### 6.2 `recommend_request`

用途：

- 记录一次推荐查询的主记录
- 记录查询主体、场景、锚点商品、分页信息和调用结果

建议字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 ID |
| `request_id` | bigint | 推荐请求 ID，雪花算法生成 |
| `actor_type` | tinyint | 主体类型：枚举【RecommendActorType】 |
| `actor_id` | bigint | 主体编号 |
| `scene` | tinyint | 推荐场景：枚举【RecommendScene】 |
| `page_num` | int | 页码 |
| `page_size` | int | 分页大小 |
| `total` | int | 本次返回总数 |
| `context_json` | json | 推荐上下文 JSON |
| `request_at` | datetime | 请求时间 |

说明：

- 一期固定字段只保留真正稳定的请求输入
- 场景特有字段统一收敛到 `context_json`
- `goods_id`、`order_id`、`strategy_type`、`gorse_recommender`、`status`、`error_msg` 等信息都不再单独占列
- 若后续这些 JSON 字段成为高频筛选条件，再从 JSON 提升为独立列
- 该表只保留业务时间 `request_at`，不再保留通用审计时间字段
- 对外接口里的 `requestId` 统一使用 `int64`
- 同一推荐会话翻页时，后续页请求继续复用 `request_id`
- `recommend_request` 每次请求都新增一条日志，不更新旧记录

`context_json` 建议内容：

```json
{
  "goods_id": 1001,
  "order_id": 2001,
  "strategy_type": "session_recommend",
  "gorse_recommender": "item-to-item/goods_relation",
  "source": "gorse",
  "context_goods_ids": [1001, 1002, 1003],
  "status": "fallback",
  "error_msg": "gorse timeout"
}
```

建议为 `context_json` 约定固定记录结构，可以用 proto 语义约束，但不进入当前对外接口：

```proto
message RecommendRequestContext {
  int64 goods_id = 1;  // 商品ID
  int64 order_id = 2;  // 订单ID
  repeated int64 context_goods_ids = 3;  // 上下文商品ID列表
  string strategy_type = 5;  // 策略类型
  string gorse_recommender = 6;  // 命中的推荐器名称
  string source = 7;  // 结果来源：gorse、fallback
  string status = 8;  // 结果状态：success、fallback、failed
  string error_msg = 9;  // 错误信息
}
```

建议约束：

- `context_json` 只放场景化上下文和本次推荐执行信息
- `scene`、`actor_type`、`actor_id`、`page_num`、`page_size`、`total` 这些固定字段不要重复写入 `context_json`
- 不要把会长期参与筛选统计的字段一直留在 JSON 里

建议索引：

- 主键：`primary key (id)`
- 普通索引：`idx_recommend_request_request_id (request_id)`
- 普通索引：`idx_recommend_request_actor_type_actor_id_request_at (actor_type, actor_id, request_at)`
- 普通索引：`idx_recommend_request_scene_request_at (scene, request_at)`

建议 DDL：

```sql
CREATE TABLE `recommend_request` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `request_id` bigint NOT NULL COMMENT '推荐请求ID',
  `actor_type` tinyint NOT NULL COMMENT '主体类型：枚举【RecommendActorType】',
  `actor_id` bigint NOT NULL COMMENT '主体编号',
  `scene` tinyint NOT NULL COMMENT '推荐场景：枚举【RecommendScene】',
  `page_num` int NOT NULL DEFAULT 1 COMMENT '页码',
  `page_size` int NOT NULL DEFAULT 10 COMMENT '分页大小',
  `total` int NOT NULL DEFAULT 0 COMMENT '返回总数',
  `context_json` json DEFAULT NULL COMMENT '推荐上下文JSON',
  `request_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '请求时间',
  PRIMARY KEY (`id`),
  KEY `idx_recommend_request_request_id` (`request_id`),
  KEY `idx_recommend_request_actor_type_actor_id_request_at` (`actor_type`, `actor_id`, `request_at`),
  KEY `idx_recommend_request_scene_request_at` (`scene`, `request_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='推荐请求记录信息';
```

### 6.3 `recommend_request_item`

用途：

- 记录一次推荐请求返回的商品明细
- 支撑通过 `request_id + goods_id` 反查推荐返回位置

建议字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `request_id` | bigint | 推荐请求 ID |
| `goods_id` | bigint | 商品 ID |
| `position` | int | 推荐位序号 |

建议索引：

- 主键：`(request_id, position)`
- 普通索引：`idx_recommend_request_item_request_id_goods_id (request_id, goods_id)`

建议 DDL：

```sql
CREATE TABLE `recommend_request_item` (
  `request_id` bigint NOT NULL COMMENT '推荐请求ID',
  `goods_id` bigint NOT NULL COMMENT '商品ID',
  `position` int NOT NULL COMMENT '推荐位序号',
  PRIMARY KEY (`request_id`, `position`),
  KEY `idx_recommend_request_item_request_id_goods_id` (`request_id`, `goods_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='推荐请求结果明细信息';
```

### 6.4 `recommend_event`

用途：

- 统一记录曝光、点击、浏览、收藏、加购、下单、支付事件
- 作为业务真相表
- 作为同步 Gorse 的上游事实来源

建议字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | bigint | 主键 ID |
| `actor_type` | tinyint | 主体类型：枚举【RecommendActorType】 |
| `actor_id` | bigint | 主体编号 |
| `scene` | tinyint | 推荐场景：枚举【RecommendScene】 |
| `event_type` | tinyint | 事件类型：枚举【RecommendEventType】 |
| `goods_id` | bigint | 商品 ID |
| `goods_num` | int | 商品数量 |
| `request_id` | bigint | 推荐请求 ID |
| `position` | int | 推荐位序号 |
| `event_at` | datetime | 事件发生时间 |

事件类型建议值：

- `1`：曝光
- `2`：点击
- `3`：浏览
- `4`：收藏
- `5`：加购
- `6`：下单
- `7`：支付

说明：

- 一期仍保留下单事件与支付事件，统一通过枚举值入库
- 前端行为和后端事实进入事件表后，都按同一套 `RecommendEventType` 存储

建议索引：

- 普通索引：`idx_recommend_event_actor_type_actor_id_event_at (actor_type, actor_id, event_at)`
- 普通索引：`idx_recommend_event_request_id (request_id)`
- 普通索引：`idx_recommend_event_event_type_event_at (event_type, event_at)`

建议 DDL：

```sql
CREATE TABLE `recommend_event` (
  `id` bigint NOT NULL COMMENT '主键ID',
  `actor_type` tinyint NOT NULL COMMENT '主体类型：枚举【RecommendActorType】',
  `actor_id` bigint NOT NULL COMMENT '主体编号',
  `scene` tinyint NOT NULL DEFAULT 0 COMMENT '推荐场景：枚举【RecommendScene】',
  `event_type` tinyint NOT NULL COMMENT '事件类型：枚举【RecommendEventType】',
  `goods_id` bigint NOT NULL COMMENT '商品ID',
  `goods_num` int NOT NULL DEFAULT 1 COMMENT '商品数量',
  `request_id` bigint DEFAULT NULL COMMENT '推荐请求ID',
  `position` int DEFAULT NULL COMMENT '推荐位序号',
  `event_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '事件发生时间',
  PRIMARY KEY (`id`),
  KEY `idx_recommend_event_actor_type_actor_id_event_at` (`actor_type`, `actor_id`, `event_at`),
  KEY `idx_recommend_event_request_id` (`request_id`),
  KEY `idx_recommend_event_event_type_event_at` (`event_type`, `event_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='推荐行为事件信息';
```

## 7. 一期不建议建设的表

以下表不建议在第一期和 Gorse 接入同时建设：

- 推荐策略发布表
- 推荐实验灰度表
- 推荐日报表
- 推荐效果宽表
- 推荐离线召回缓存表
- 推荐模型评估表

原因：

- 这些表不是移动端推荐可用的必要条件
- 当前最重要的是先跑通推荐请求、曝光归因、行为回写、Gorse 同步
- 一期表越少，后续越容易稳定落地

## 8. Gorse 数据映射设计

## 8.1 User 映射

Gorse `User` 建议映射为登录用户，不映射匿名主体。

建议映射：

- `UserId`：`user:{userId}`
- `Labels`：
  - 用户等级
  - 性别
  - 地区
  - 注册时长桶
  - 会员状态
- `Comment`：
  - 可选写入 JSON 字符串，便于排障

匿名主体不作为正式 Gorse 用户长期保存的原因：

- Gorse 不提供业务语义上的用户合并能力
- 匿名登录后迁移历史画像会变复杂
- 当前匿名场景更适合走会话推荐

## 8.2 Item 映射

Gorse `Item` 建议映射为商品。

建议映射：

- `ItemId`：`goods:{goodsId}`
- `IsHidden`：
  - 商品下架时为 `true`
  - 商品上架时为 `false`
- `Categories`：
  - 商品分类 ID 字符串列表
- `Labels`：
  - 品牌
  - 一级分类
  - 二级分类
  - 价格带
  - 是否新品
  - 是否会员价
- `Timestamp`：
  - 商品创建时间或上架时间

## 8.3 Feedback 映射

Gorse `Feedback` 由业务事件统一转换。

映射建议：

| 业务事件 | Gorse FeedbackType |
| --- | --- |
| 曝光 | `exposure` |
| 点击 | `click` |
| 浏览 | `view` |
| 收藏 | `collect` |
| 加购 | `add_cart` |
| 支付 | `order_pay` |

建议：

- `Timestamp` 使用业务事实发生时间
- `Value` 一期统一写 `1`
- 曝光视为 Gorse 的 `read feedback`
- 点击、浏览、收藏、加购、支付视为 `positive feedback`
- 一期建议只把 `order_pay` 作为订单类正反馈同步给 Gorse

## 9. 推荐查询设计

### 9.1 场景与策略映射

建议一期采用代码配置映射，不先建设策略表。

建议映射如下：

| 场景 | 主策略 | 说明 |
| --- | --- | --- |
| 首页 | `GetRecommend` / `SessionRecommend` | 登录态个性化，匿名态会话推荐 |
| 商品详情 | `GetNeighbors` | 基于当前商品查相似商品 |
| 购物车 | `SessionRecommend` | 基于购物车商品做会话推荐 |
| 我的 | `GetRecommend` | 登录态个性化，匿名态热榜兜底 |
| 支付成功 | `SessionRecommend` | 基于支付成功商品做会话推荐 |

### 9.2 推荐接口处理流程

`RecommendGoods` 建议按以下顺序执行：

1. 识别主体：
   - 已登录取 `user_id`
   - 未登录取匿名请求头
2. 生成 `request_id`
3. 写入 `recommend_request`
   - 每次请求都新增一条记录
   - 若属于同一翻页会话，则继续复用相同 `request_id`
4. 根据场景调用 Gorse：
   - 首页登录态：`GetRecommend`
   - 首页匿名态：`SessionRecommend`
   - 商品详情：`GetNeighbors`
   - 购物车/支付成功：`SessionRecommend`
5. 得到商品 ID 列表后，回查业务商品表
6. 写入 `recommend_request_item`
7. 返回商品详情和 `request_id`

### 9.3 匿名场景处理

匿名场景建议优先使用 `SessionRecommend`。

会话输入来源建议如下：

- 首页匿名态：
  - 若匿名主体已有最近浏览、点击、加购记录，则取最近若干个商品作为会话输入
  - 若没有历史行为，则回退热榜或最新商品
- 商品详情匿名态：
  - 用当前 `goods_id` 作为上下文
- 购物车匿名态：
  - 用购物车商品列表作为上下文

## 10. 推荐事件上报设计

### 10.1 统一事件接口

一期统一使用 `RecommendEventReport`，不再拆分“曝光上报”和“商品行为上报”两个接口。

请求结构：

1. `event_type`：事件类型，统一使用 `RecommendEventType`
2. `recommend_context`：请求级上下文，记录
   - `scene`
   - `request_id`
3. `items`：商品项列表，记录
   - `goods_id`
   - `goods_num`
   - `position`

处理建议：

1. 先读取请求级 `recommend_context`
2. 遍历 `items`
3. 必要时根据 `request_id + goods_id` 回查 `recommend_request_item` 补齐 `position`
4. 批量写入 `recommend_event`
5. 一期先完成本地事件沉淀，后续再接 Gorse feedback

### 10.2 后端事实优先原则

以下行为建议以后端事实为准，不完全依赖前端上报：

- 加购
- 收藏
- 支付

理由：

- 能保证推荐行为和业务真相一致
- 避免前端失败或重试导致推荐数据失真
- 便于订单、支付等高价值行为统一从后端主链路回写

## 11. 匿名主体绑定设计

### 11.1 绑定目标

`BindRecommendAnonymousActor` 的目标不是简单记录一条日志，而是要把匿名会话和登录用户尽可能衔接起来。

### 11.2 一期建议做法

建议绑定流程如下：

1. 校验当前登录用户
2. 从请求头读取 `anonymous_id`
3. 更新 `recommend_anonymous_actor.user_id`
4. 更新 `recommend_anonymous_actor.bind_at`
5. 查询该匿名主体最近若干天的正反馈事件
6. 将这些事件直接补写到登录用户的 Gorse feedback

### 11.3 不建议的做法

一期不建议直接做以下操作：

- 直接把匿名主体在 Gorse 中改名成正式用户
- 直接修改匿名历史事件主键
- 直接删除匿名历史事件

原因：

- 会破坏业务审计链路
- 容易造成 Gorse 与业务库事实不一致
- 实施复杂度高，收益有限

## 12. `recommend.proto` 修改建议

当前移动端继续使用 [recommend.proto](/Users/liujun/workspace/shop/shop/backend/api/protos/app/recommend.proto) 的接口，这一点不建议推翻。

建议按“尽量不改动”的方式兼容 Gorse。

### 12.1 一期结论：当前可不改

当前 [enum.proto](/Users/liujun/workspace/shop/shop/backend/api/protos/common/enum.proto) 中有：

- `HOME`
- `GOODS_DETAIL`
- `CART`
- `PROFILE`
- `ORDER_DETAIL`
- `ORDER_PAID`

结合当前移动端实现，一期建议保持现状，不新增推荐场景。

原因：

- 下单确认页当前没有独立推荐接口调用
- 下单确认页当前只负责承接来源商品的推荐上下文，并在提交订单时回传后端
- 首页、商品详情、购物车、我的、订单详情、支付成功这些现有场景已经覆盖当前页面推荐入口

### 12.2 一期结论：`RecommendGoodsRequest` 需要补充 `requestId`

当前 `RecommendGoodsRequest` 只有：

- `scene`
- `orderId`
- `goodsId`
- `pageNum`
- `pageSize`

一期建议新增：

- `requestId`

原因：

- 商品详情推荐使用 `goodsId` 已足够
- 支付成功和订单详情推荐可以通过 `orderId` 回查订单商品
- 购物车推荐可以由后端基于当前登录用户的购物车数据自行组装上下文
- 匿名购物车页当前没有真实购物车商品明细，实际展示的是泛化推荐区
- 同一推荐会话翻页时，需要由前端把首次返回的 `requestId` 带回后端继续复用

### 12.3 未来扩展时再评估的字段

如果后续新增以下能力，再评估是否调整 `recommend.proto`：

- 下单确认页新增独立推荐位
- 前端明确需要把多商品上下文直接传给推荐接口
- 匿名会话推荐需要前端实时传入浏览商品列表
### 12.4 一期修改结论

以下结构一期继续保留，其中 `requestId` 字段统一改为 `int64`：

- `RecommendEventReportRequest`
- `RecommendContext`

原因：

- 当前已经有 `scene + requestId + position`
- 归因所需信息已基本足够

## 13. 配置修改建议

### 13.1 业务后端配置

建议在业务配置中增加推荐相关配置，例如：

- `endpoint`
- `apiKey`
- `timeout`
- `feedbackBatchSize`

建议说明：

- `endpoint` 对接 Gorse HTTP 服务
- `apiKey` 用于后端访问 Gorse

### 13.2 Gorse 配置

当前 [gorse/config/config.toml](/Users/liujun/workspace/shop/shop/gorse/config/config.toml) 的推荐反馈配置大方向可以继续使用：

- `positive_feedback_types`
- `read_feedback_types`
- 非个性化热榜
- item-to-item 相似商品
- fallback 逻辑

建议补充关注点：

- 为服务访问开启 API Key 保护
- 确认 `positive_feedback_types` 包含：
  - `click`
  - `view`
  - `collect`
  - `add_cart`
  - `order_pay`
- 确认 `read_feedback_types` 包含：
  - `exposure`

## 14. Go SDK 接入建议

当前接入建议使用 Gorse Go SDK。

建议注意：

- 推荐使用的 module path 为 `github.com/gorse-io/gorse-go`
- 不建议直接使用 `github.com/zhenghaoz/gorse-io/gorse-go`

建议的客户端调用范围：

- `InsertUser`
- `UpdateUser`
- `InsertItem`
- `UpdateItem`
- `InsertFeedback`
- `GetRecommend`
- `GetNeighbors`
- `SessionRecommend`

## 15. 一期实施顺序建议

建议按以下顺序推进：

### 第一步：建业务表

- 建 `recommend_anonymous_actor`
- 建 `recommend_request`
- 建 `recommend_request_item`
- 建 `recommend_event`

### 第二步：接入商品和用户同步

- 商品创建、上下架、更新时直接同步到 Gorse
- 用户注册、登录、画像变化时直接同步到 Gorse

### 第三步：接入推荐查询

- 先打通首页、商品详情、购物车
- 再补我的、支付成功

### 第四步：接入曝光和行为上报

- 曝光写 `recommend_event`
- 点击写 `recommend_event`
- 后端事实加购、收藏、下单、支付写 `recommend_event`
- 同步反馈到 Gorse

### 第五步：接入匿名绑定

- 完成匿名主体绑定
- 完成匿名历史反馈回放

## 16. 最终结论

从 0 接入 Gorse 时，业务库不应该复制 Gorse 的 `users/items/feedback` 数据结构，而应该建设一层“业务推荐事实层”。

这一层的职责只有四类：

- 身份衔接
- 请求留痕
- 结果归因

因此，一期最小可用方案只需要 4 张业务表：

1. `recommend_anonymous_actor`
2. `recommend_request`
3. `recommend_request_item`
4. `recommend_event`

在这套模型下：

- Gorse 负责算推荐
- 业务库负责留痕和归因
- 移动端现有接口可以延续
- 未来要扩展推荐后台、推荐报表、策略灰度，也可以继续叠加，不需要推倒重来
