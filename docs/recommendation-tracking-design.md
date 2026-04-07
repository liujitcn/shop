# 移动端推荐能力与埋点归因设计

## 1. 背景

当前移动端首页、购物车、我的、订单详情、订单支付成功页都存在推荐商品区，但现状是：

- 前端复用同一个推荐组件；
- 后端推荐接口没有区分页面场景；
- 推荐结果缺少有效归因；
- 无法准确判断推荐是否被看到、是否被点击、是否带来了后续浏览和转化。

本方案目标是在保留推荐区通用组件的前提下，建设一套适合个性化推荐的最小埋点与归因能力。

本次设计遵循一个明确约束：

- 尽量让前端少参与；
- 优先通过后端直接记录行为；
- 前端只在无法由后端感知的地方传递最少上下文参数；
- 商品详情页允许增加参数用于归因。

## 2. 设计目标

### 2.1 业务目标

- 支持首页、购物车、我的、订单详情、支付成功五个页面的个性化推荐；
- 支持不同页面使用不同推荐文案；
- 支持后续评估推荐效果，包括曝光、点击、详情浏览、加购、下单、支付。

### 2.2 工程目标

- 不依赖大量前端主动埋点接口；
- 交易类行为尽可能由后端直接记录；
- 推荐归因链路可以串起来；
- 为后续推荐排序优化预留数据基础。

## 3. 核心结论

### 3.1 哪些行为可以由后端直接记录

以下行为天然经过业务接口，后端应直接记录，不需要前端额外埋点：

- 加购
- 收藏
- 下单
- 支付
- 退款
- 再次购买

这些行为可靠、明确、可直接入库，适合用于用户个性化画像。

### 3.2 哪些行为不能完全只靠后端

以下行为如果完全不靠前端，后端无法准确判断：

- 推荐曝光
- 推荐点击来源
- 商品详情来源归因

原因如下：

- 推荐曝光：后端只能知道推荐结果返回了，不知道用户是否真的滑到该区域并看到；
- 推荐点击：后端只能看到详情接口被请求，无法区分是从推荐位进入还是从搜索、收藏、购物车进入；
- 商品详情浏览：后端可以知道详情接口被调用，但不知道来源链路，除非前端把来源信息带上。

因此，完全零前端参与不可行。

### 3.3 最小前端参与原则

本方案采用“曝光前端埋点，点击与浏览后端归因，交易行为后端直记”的方式：

- 推荐列表请求时，后端生成 `requestId`；
- 推荐区首次进入可视区域时，前端只调用一次曝光接口；
- 点击推荐商品时，前端不单独调点击接口，而是在跳商品详情时把归因参数拼到详情页 URL；
- 商品详情页请求详情接口时，前端把来源参数一并带给后端；
- 后端在详情接口中自动记录“推荐点击 + 商品浏览”。

这样可以把前端参与压缩到最低，同时保证归因闭环成立。

## 4. 页面范围

本方案覆盖以下推荐页面：

- 首页
- 购物车
- 我的
- 订单详情
- 支付成功

场景枚举建议如下：

- `HOME`
- `CART`
- `PROFILE`
- `ORDER_DETAIL`
- `ORDER_PAID`

## 5. 推荐归因整体链路

### 5.1 事件流

推荐链路按如下顺序发生：

1. 用户进入含推荐位的页面；
2. 页面请求推荐接口；
3. 后端返回推荐列表与 `requestId`；
4. 推荐区进入可视区域，前端上报一次曝光；
5. 用户点击推荐商品；
6. 前端跳商品详情，并把推荐来源参数带上；
7. 商品详情接口接收到这些参数；
8. 后端在商品详情接口内自动记录：
   - 一次推荐点击；
   - 一次商品浏览；
9. 用户后续加购、收藏、下单、支付时，后端继续记录业务行为；
10. 推荐服务根据这些行为进行个性化推荐。

### 5.2 设计取舍

本方案不要求前端在点击时额外调一个“推荐点击埋点接口”，原因是：

- 会增加前端复杂度；
- 弱网下容易丢；
- 点击行为本质上可以通过“详情请求 + 来源参数”由后端归因。

代价是：

- 只能把“进入详情页”视为一次有效点击；
- 无法记录“点了但跳转失败”的极端情况。

这个取舍对当前阶段是合理的。

## 6. 前后端职责划分

### 6.1 前端职责

前端只负责以下最小事项：

- 请求推荐接口；
- 保存后端返回的 `requestId`；
- 推荐区首次进入可视区域时上报一次曝光；
- 点击推荐商品跳详情时，拼接归因参数；
- 商品详情请求时，把归因参数一起带给后端。

前端不负责：

- 单独落库；
- 推荐点击独立埋点；
- 商品浏览独立埋点；
- 行为计算和画像处理。

### 6.2 后端职责

后端负责：

- 根据场景返回推荐列表；
- 为每次推荐请求生成 `requestId`；
- 记录推荐结果下发；
- 接收曝光事件并落库；
- 在商品详情接口内根据归因参数自动记录推荐点击和商品浏览；
- 在收藏、加购、下单、支付等业务接口内直接记录强行为；
- 基于行为数据构建用户画像和推荐召回排序。

## 7. 接口设计

### 7.1 推荐接口

建议新增独立推荐接口，不复用商品分页接口。

接口建议：

- `GET /api/app/recommend/goods`

请求字段建议：

```proto
message RecommendGoodsRequest {
  RecommendScene scene = 1;
  int64 orderId = 2;
  repeated int64 cartGoodsIds = 3;
  int64 currentGoodsId = 4;
  repeated int64 currentCategoryIds = 5;
  int64 pageNum = 101;
  int64 pageSize = 102;
}
```

返回字段建议：

```proto
message RecommendGoodsItem {
  int64 id = 1;
  string name = 2;
  string desc = 3;
  string picture = 4;
  int64 price = 5;
  int64 saleNum = 6;
  string reason = 7;
}

message RecommendGoodsResponse {
  repeated RecommendGoodsItem list = 1;
  int32 total = 2;
  string requestId = 3;
}
```

### 7.2 推荐曝光接口

曝光无法由后端直接感知，因此保留一个最小曝光接口。

接口建议：

- `POST /api/app/recommend/expose`

请求字段建议：

```proto
message RecommendExposeRequest {
  string requestId = 1;
  RecommendScene scene = 2;
  repeated int64 goodsIds = 3;
}
```

说明：

- 每次推荐区首次进入可视区域只上报一次；
- 可以按整组商品上报，不要求每个商品单独发一次。

### 7.3 商品详情接口增加归因参数

商品详情接口建议支持以下归因参数：

接口形式可以是：

- `GET /api/app/goods/info/{id}?source=...&scene=...&requestId=...&index=...`

参数建议：

- `source`
- `scene`
- `requestId`
- `index`

示例：

```text
/api/app/goods/info/123?source=recommend&scene=HOME&requestId=req_abc123&index=2
```

说明：

- `source` 表示来源，例如 `recommend`、`search`、`cart`、`favorite`、`direct`；
- `scene` 表示推荐位所属页面场景；
- `requestId` 用于串联同一次推荐请求；
- `index` 表示该商品在推荐列表中的位置。

约束建议：

- 当 `source != recommend` 时，`scene` 可为空；
- 当 `source = recommend` 时，`scene` 必填；
- `source` 不由后端自动推断，而是由详情页上游入口在跳转时显式传递；
- 未接入来源参数的入口统一降级为 `direct`。

### 7.4 点击与浏览的后端自动记录

当商品详情接口收到如下参数时：

- `source=recommend`
- `scene` 非空
- `requestId` 非空

后端自动记录两类事件：

1. `recommend_click`
2. `goods_view`

其中：

- `recommend_click` 代表“用户从推荐位进入详情”；
- `goods_view` 代表“商品详情被浏览”。

如果详情接口未携带推荐来源，但正常访问，则仍记录一条普通 `goods_view`。

## 8. 前后端参与边界

最终口径如下：

- 推荐曝光：前端埋点，上报给后端；
- 推荐点击：不单独前端埋点，通过跳详情携带参数，由后端自动归因记录；
- 商品浏览：详情接口携带来源参数，由后端自动记录；
- 加购、收藏、下单、支付：后端业务接口直接记录。

说明：

- 曝光必须由前端参与，因为只有前端知道推荐区是否真正进入了用户可视区域；
- 点击和浏览可通过详情链路归因，避免前端增加额外点击埋点逻辑；
- 这样既保证数据可用，也控制了前端改造成本。

## 9. source 与 scene 约定

### 9.1 字段职责

`source` 表示入口来源分类，适合做全站统一归因。

`scene` 表示推荐位所属业务场景，只在推荐来源下有强业务意义。

两者不能互相替代。

示例：

- `source=recommend, scene=HOME`
- `source=recommend, scene=ORDER_PAID`

这两条记录都属于推荐来源，但业务场景不同，必须通过 `scene` 区分。

### 9.2 source 取值建议

建议统一如下取值：

- `recommend`
- `search`
- `cart`
- `favorite`
- `order`
- `direct`

### 9.3 source 来源规则

`source` 不是由后端自动推断，而是由“跳转到商品详情的上游入口”显式传递。

来源规则建议如下：

- 推荐位跳详情：传 `source=recommend`
- 搜索结果页跳详情：传 `source=search`
- 购物车页跳详情：传 `source=cart`
- 收藏页跳详情：传 `source=favorite`
- 订单列表或订单详情页跳详情：传 `source=order`
- 无法识别来源或老链路未接入时：传 `source=direct`

### 9.4 详情归因规则

建议统一约定：

- 当 `source = recommend` 时，`scene`、`requestId` 必填，`index` 建议传；
- 当 `source != recommend` 时，`scene`、`requestId`、`index` 可为空；
- 详情接口收到参数后，统一由后端做来源归因和行为记录。

## 10. 页面跳转参数规范

推荐位跳商品详情时，建议统一使用如下路由参数：

- `id`
- `source=recommend`
- `scene`
- `requestId`
- `index`

示例：

```text
/pages/goods/goods?id=123&source=recommend&scene=ORDER_PAID&requestId=req_abc123&index=1
```

这样详情页只需要把这些参数原样传给详情接口即可。

## 11. 事件模型设计

### 11.1 推荐结果下发

事件名建议：

- `recommend_served`

记录时机：

- 推荐接口返回结果时，由后端直接记录。

字段建议：

- `request_id`
- `user_id`
- `scene`
- `goods_ids`
- `page_num`
- `page_size`
- `served_at`

说明：

- 这不是曝光；
- 只是说明该次推荐结果已经返回给客户端。

### 11.2 推荐曝光

事件名建议：

- `recommend_exposed`

记录时机：

- 推荐区首次进入视口，由前端调用曝光接口，后端落库。

字段建议：

- `request_id`
- `user_id`
- `scene`
- `goods_ids`
- `exposed_at`

### 11.3 推荐点击

事件名建议：

- `recommend_clicked`

记录时机：

- 商品详情接口接收 `source=recommend` 和归因参数时，由后端自动记录。

字段建议：

- `request_id`
- `user_id`
- `scene`
- `goods_id`
- `index`
- `clicked_at`

### 11.4 商品浏览

事件名建议：

- `goods_viewed`

记录时机：

- 商品详情接口被访问时，由后端自动记录。

字段建议：

- `user_id`
- `goods_id`
- `source`
- `scene`
- `request_id`
- `viewed_at`

### 11.5 交易强行为

事件名建议：

- `goods_collected`
- `goods_cart_added`
- `order_created`
- `order_paid`

记录时机：

- 各自业务接口成功后，由后端直接记录。

## 12. 推荐场景与文案

建议文案如下：

- 首页：`为你推荐`
- 购物车：`搭配着买`
- 我的：`根据你的偏好推荐`
- 订单详情：`买过这单的人还会买`
- 支付成功：`顺手再带两件`

说明：

- 标题不必统一；
- 不同文案有助于向用户解释推荐来源和意图；
- 这些文案不影响推荐接口，只影响前端展示。

## 13. 个性化推荐策略

### 13.1 数据来源优先级

按推荐价值从高到低建议如下：

- 支付
- 下单
- 加购
- 收藏
- 浏览
- 热门商品兜底

### 13.2 召回源

第一版建议采用多路召回：

- 用户近 30 天支付商品关联召回
- 用户近 30 天下单商品关联召回
- 用户当前购物车商品关联召回
- 用户收藏商品同类召回
- 用户近期浏览商品同类召回
- 热门商品召回
- 新品商品召回

### 13.3 排序思路

建议使用可解释打分公式：

```text
score = 行为权重 + 场景权重 + 类目匹配权重 + 热度权重 + 时间衰减权重 - 去重惩罚
```

建议权重：

- 支付：10
- 下单：8
- 加购：6
- 收藏：4
- 浏览：2

时间衰减建议：

- 7 天内：1.0
- 8-15 天：0.7
- 16-30 天：0.4
- 30 天外：0.1

### 13.4 场景差异

首页：

- 强调偏好覆盖和探索性；
- 优先常买类目与近期兴趣；
- 兜底热门。

购物车：

- 强调连带销售；
- 优先购物车商品关联与补充商品；
- 排除购物车已有商品。

我的：

- 强调用户历史偏好；
- 优先支付、收藏、浏览的稳定偏好类目；
- 无行为则退化到热门。

订单详情：

- 强调复购和关联购买；
- 基于当前订单商品召回；
- 排除当前订单内商品。

支付成功：

- 强调趁热转化；
- 优先已支付订单商品的强关联商品、配件商品、搭配商品；
- 场景权重应最高。

## 14. 数据表建议

建议新增推荐域统一前缀的数据表，而不是把所有行为散落在日志中。

统一前缀建议：

- 表前缀：`recommend_`
- 服务前缀：`Recommend`

这样可以保证推荐域的数据模型、代码结构、统计口径都集中管理。

命名约定建议：

- 事件明细表统一使用 `created_at`，不再混用 `clicked_at`、`viewed_at`、`served_at`；
- 推荐位序号统一使用落库字段 `position`，前端与接口层可继续传 `index`，入库时映射为 `position`；
- JSON 字段统一使用 `_json` 后缀；
- 能通过 `request_id` 稳定回查的字段，优先不重复落库，避免冗余。

### 14.1 推荐请求表

表名建议：

- `recommend_request`

用途：

- 记录一次推荐结果下发；
- 为后续曝光、点击、浏览、转化归因提供母记录；
- 为排查推荐结果内容、排序和分页问题提供依据。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 后端生成 |
| `request_id` | varchar(64) | 一次推荐请求的唯一标识 | `RecommendService` 生成 |
| `user_id` | bigint | 用户 ID，未登录可为 `0` | 登录态解析 |
| `scene` | varchar(32) | 推荐场景，如 `HOME`、`CART` | 推荐接口入参 |
| `source_context_json` | json | 场景上下文，如 `orderId`、`cartGoodsIds`、`currentGoodsId` | 推荐接口入参组装 |
| `page_num` | int | 页码 | 推荐接口入参 |
| `page_size` | int | 每页数量 | 推荐接口入参 |
| `goods_ids_json` | json | 本次下发商品 ID 列表 | 推荐结果 |
| `strategy_version` | varchar(32) | 推荐策略版本号 | `RecommendService` 配置 |
| `recall_sources_json` | json | 本次结果涉及的召回来源集合 | `RecommendService` 计算 |
| `created_at` | datetime | 创建时间 | 后端写入 |

建议索引：

- 唯一索引：`uk_request_id(request_id)`
- 普通索引：`idx_user_scene_created(user_id, scene, created_at)`

约定：

- 已登录用户写真实 `user_id`；
- 未登录用户统一写 `user_id = 0`；
- 不再额外存储 `is_login`，避免冗余和数据不一致。

### 14.2 推荐曝光表

表名建议：

- `recommend_exposure`

用途：

- 记录推荐区被用户实际看到；
- 支撑曝光、点击、转化漏斗分析；
- 作为推荐效果评估的起点数据。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 后端生成 |
| `request_id` | varchar(64) | 对应 `recommend_request.request_id` | 前端曝光接口入参 |
| `user_id` | bigint | 用户 ID，未登录可为 `0` | 登录态解析 |
| `scene` | varchar(32) | 推荐场景 | 前端曝光接口入参 |
| `goods_ids_json` | json | 本次曝光的商品 ID 列表 | 前端曝光接口入参 |
| `expose_mode` | varchar(16) | 曝光模式，首版可固定 `viewport_once` | 前端约定 + 后端落库 |
| `created_at` | datetime | 曝光时间 | 后端写入 |

建议索引：

- 普通索引：`idx_request_id(request_id)`
- 普通索引：`idx_user_scene_created(user_id, scene, created_at)`

### 14.3 推荐点击表

表名建议：

- `recommend_click`

用途：

- 记录用户从推荐位进入商品详情；
- 衡量推荐点击率、位置效果；
- 与浏览、加购、下单、支付做后续转化归因。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 后端生成 |
| `request_id` | varchar(64) | 对应推荐请求 ID | 商品详情接口参数 |
| `user_id` | bigint | 用户 ID，未登录可为 `0` | 登录态解析 |
| `scene` | varchar(32) | 推荐场景 | 商品详情接口参数 |
| `goods_id` | bigint | 被点击的商品 ID | 商品详情接口路径参数 |
| `position` | int | 推荐位序号，从 `0` 或 `1` 开始需统一 | 商品详情接口参数 `index` |
| `source` | varchar(32) | 固定为 `recommend` | 商品详情接口参数 |
| `created_at` | datetime | 点击时间 | 后端写入 |

建议索引：

- 普通索引：`idx_request_goods(request_id, goods_id)`
- 普通索引：`idx_user_scene_created(user_id, scene, created_at)`

### 14.4 商品浏览表

表名建议：

- `recommend_goods_view`

用途：

- 统一记录商品详情浏览；
- 既支持推荐来源浏览，也支持搜索、购物车、订单等来源浏览；
- 为用户兴趣画像和推荐召回提供基础行为数据。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 后端生成 |
| `user_id` | bigint | 用户 ID，未登录可为 `0` | 登录态解析 |
| `goods_id` | bigint | 被浏览商品 ID | 商品详情接口路径参数 |
| `source` | varchar(32) | 入口来源，如 `recommend`、`search`、`cart` | 商品详情接口参数 |
| `scene` | varchar(32) | 推荐场景，仅 `source=recommend` 时一般有值 | 商品详情接口参数 |
| `request_id` | varchar(64) | 推荐请求 ID，非推荐来源可为空 | 商品详情接口参数 |
| `position` | int | 推荐位序号，非推荐来源可为空 | 商品详情接口参数 `index` |
| `view_mode` | varchar(16) | 浏览模式，首版固定 `detail_open` | 后端写入 |
| `created_at` | datetime | 浏览时间 | 后端写入 |

建议索引：

- 普通索引：`idx_user_created(user_id, created_at)`
- 普通索引：`idx_goods_created(goods_id, created_at)`
- 普通索引：`idx_source_scene_created(source, scene, created_at)`

### 14.5 用户行为聚合表

后续可以按日或按周生成：

- `recommend_user_preference`
- `recommend_user_goods_preference`
- `recommend_goods_relation`

第一阶段不强制要求实时构建，可由定时任务生成。

说明：

- `recommend_user_preference` 用于沉淀用户类目偏好、价格带偏好、品牌偏好等；
- `recommend_user_goods_preference` 用于沉淀用户对具体商品的偏好得分；
- `recommend_goods_relation` 用于沉淀商品间关联关系，如搭配购、同类替代、共购关系。

#### 14.5.1 用户偏好表

表名建议：

- `recommend_user_preference`

用途：

- 存储用户在类目、品牌、价格带等维度上的偏好结果；
- 为首页、我的页面做稳定偏好推荐；
- 作为排序层的用户画像输入。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 聚合任务生成 |
| `user_id` | bigint | 用户 ID | 行为明细聚合 |
| `preference_type` | varchar(32) | 偏好类型，如 `category`、`brand`、`price_band` | 聚合任务定义 |
| `target_id` | bigint | 偏好对象 ID，价格带可做枚举映射 | 聚合任务计算 |
| `score` | decimal(10,4) | 偏好得分 | 聚合任务计算 |
| `behavior_summary_json` | json | 贡献该偏好的行为汇总 | 聚合任务计算 |
| `window_days` | int | 统计窗口，如 `7`、`30` | 聚合任务配置 |
| `created_at` | datetime | 首次写入时间 | 聚合任务写入 |
| `updated_at` | datetime | 更新时间 | 聚合任务写入 |

建议索引：

- 唯一索引：`uk_user_pref(user_id, preference_type, target_id, window_days)`
- 普通索引：`idx_user_type_score(user_id, preference_type, score)`

#### 14.5.2 用户商品偏好表

表名建议：

- `recommend_user_goods_preference`

用途：

- 记录用户对具体商品的兴趣强弱；
- 为个性化召回和排序提供更细粒度输入；
- 适合承接浏览、收藏、加购、下单、支付等多种行为加权。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 聚合任务生成 |
| `user_id` | bigint | 用户 ID | 行为明细聚合 |
| `goods_id` | bigint | 商品 ID | 行为明细聚合 |
| `score` | decimal(10,4) | 对该商品的兴趣得分 | 聚合任务计算 |
| `last_behavior_type` | varchar(32) | 最近一次关键行为类型 | 行为明细聚合 |
| `last_behavior_at` | datetime | 最近行为时间 | 行为明细聚合 |
| `behavior_summary_json` | json | 浏览/收藏/加购/支付贡献明细 | 聚合任务计算 |
| `window_days` | int | 统计窗口 | 聚合任务配置 |
| `created_at` | datetime | 首次写入时间 | 聚合任务写入 |
| `updated_at` | datetime | 更新时间 | 聚合任务写入 |

建议索引：

- 唯一索引：`uk_user_goods(user_id, goods_id, window_days)`
- 普通索引：`idx_user_score(user_id, score)`

#### 14.5.3 商品关联表

表名建议：

- `recommend_goods_relation`

用途：

- 存储商品与商品之间的关联关系；
- 支撑购物车、订单详情、支付成功页的强关联召回；
- 可承载共购、搭配购、同类替代等关系。

字段草案：

| 字段名 | 类型建议 | 说明 | 数据来源 |
| --- | --- | --- | --- |
| `id` | bigint | 自增主键 | 聚合任务生成 |
| `goods_id` | bigint | 主商品 ID | 订单/购物车/行为聚合 |
| `related_goods_id` | bigint | 关联商品 ID | 订单/购物车/行为聚合 |
| `relation_type` | varchar(32) | 关系类型，如 `co_buy`、`bundle`、`substitute` | 聚合任务定义 |
| `score` | decimal(10,4) | 关联强度得分 | 聚合任务计算 |
| `evidence_json` | json | 关联证据，如共购次数、同类相似度等 | 聚合任务计算 |
| `window_days` | int | 统计窗口 | 聚合任务配置 |
| `created_at` | datetime | 首次写入时间 | 聚合任务写入 |
| `updated_at` | datetime | 更新时间 | 聚合任务写入 |

建议索引：

- 唯一索引：`uk_goods_relation(goods_id, related_goods_id, relation_type, window_days)`
- 普通索引：`idx_goods_type_score(goods_id, relation_type, score)`

## 15. 服务层设计建议

建议新增推荐域统一前缀服务，而不是继续塞进商品分页服务：

- `RecommendService`
- `RecommendProfileService`
- `RecommendRelationService`

职责建议：

`RecommendService`

- 按场景获取推荐结果；
- 负责召回、打分、去重、兜底；
- 生成 `requestId`；
- 记录 `recommend_served`。

`RecommendProfileService`

- 汇总支付、下单、加购、收藏、浏览；
- 计算用户偏好类目、偏好商品、近期兴趣。

`RecommendRelationService`

- 维护商品关联关系；
- 支持共购、搭配购、同类替代等召回能力；
- 为购物车、订单详情、支付成功等强场景提供高质量召回。

### 15.1 服务接口定义建议

以下接口定义偏服务层语义，不强绑定具体 transport，可用于 `biz/service` 设计。

#### 15.1.1 RecommendService

职责：

- 面向页面场景提供推荐结果；
- 负责召回、排序、去重、兜底；
- 生成并返回 `requestId`；
- 写入 `recommend_request`。
- 提供前端可调用的推荐曝光接口。

接口建议：

```go
type RecommendService interface {
    GetRecommendGoods(ctx context.Context, req *RecommendGoodsRequest) (*RecommendGoodsResult, error)
    RecordRecommendExposure(ctx context.Context, req *RecommendExposureRequest) error
}
```

入参建议：

| 字段名 | 说明 | 使用场景 |
| --- | --- | --- |
| `scene` | 推荐场景 | 所有推荐页面 |
| `orderId` | 订单 ID | 订单详情、支付成功 |
| `cartGoodsIds` | 购物车商品 ID 列表 | 购物车 |
| `currentGoodsId` | 当前商品 ID | 预留给商品详情页相关推荐 |
| `currentCategoryIds` | 当前类目上下文 | 首页、搜索、分类扩展 |
| `pageNum` | 页码 | 所有推荐页面 |
| `pageSize` | 数量 | 所有推荐页面 |

返回值建议：

| 字段名 | 说明 |
| --- | --- |
| `requestId` | 本次推荐请求唯一 ID |
| `list` | 推荐商品列表 |
| `total` | 总数 |
| `strategyVersion` | 策略版本 |

补充说明：

- `GetRecommendGoods` 对应前端推荐列表获取接口；
- `RecordRecommendExposure` 对应前端推荐曝光接口 `POST /api/app/recommend/expose`；
- 推荐曝光接口归属 `RecommendService`，不单独拆分新的对外埋点服务。

#### 15.1.2 RecommendProfileService

职责：

- 汇总用户强行为和弱行为；
- 生成用户偏好画像；
- 为推荐召回与排序提供用户画像输入。

接口建议：

```go
type RecommendProfileService interface {
    BuildUserPreference(ctx context.Context, userId int64, windowDays int) error
    ListUserPreferences(ctx context.Context, userId int64, preferenceType string, limit int) ([]*UserPreference, error)
    ListUserGoodsPreferences(ctx context.Context, userId int64, limit int) ([]*UserGoodsPreference, error)
}
```

说明：

- `BuildUserPreference` 更适合由定时任务或异步任务驱动；
- `ListUserPreferences` 用于按类目、品牌、价格带获取偏好；
- `ListUserGoodsPreferences` 用于个性化召回或排序加权。

典型使用场景：

| 场景 | 用法 |
| --- | --- |
| 首页 | 获取用户稳定偏好类目、品牌、价格带，用于个性化召回和排序 |
| 我的 | 作为“根据你的偏好推荐”的核心输入 |
| 购物车 | 当强关联商品不足时，用用户偏好补足推荐结果 |
| 订单详情 | 当前订单关联召回不足时，用用户长期偏好做补充 |
| 支付成功 | 在强关联召回之外，用偏好补足探索型商品 |
| 定时任务 | 周期性计算并更新用户偏好画像 |

#### 15.1.3 RecommendRelationService

职责：

- 维护商品关联关系；
- 为强场景召回提供商品到商品的关系输入。

接口建议：

```go
type RecommendRelationService interface {
    BuildGoodsRelation(ctx context.Context, windowDays int) error
    ListRelatedGoods(ctx context.Context, goodsIds []int64, relationTypes []string, limit int) ([]*GoodsRelation, error)
}
```

说明：

- `BuildGoodsRelation` 建议由离线任务生成；
- `ListRelatedGoods` 适合购物车、订单详情、支付成功等页面调用。

典型使用场景：

| 场景 | 用法 |
| --- | --- |
| 购物车 | 根据购物车内商品召回搭配购、共购商品 |
| 订单详情 | 根据订单内商品召回“买过这单的人还会买” |
| 支付成功 | 根据已支付商品召回强关联商品，提高趁热转化 |
| 商品详情扩展 | 预留给未来商品详情页“看了还看”或“相关推荐” |
| 聚合任务 | 周期性重建商品关联关系 |

### 15.2 使用场景建议

#### 15.2.1 首页

调用链建议：

1. 页面调用 `RecommendService.GetRecommendGoods`
2. 入参 `scene=HOME`
3. 服务基于用户偏好、近期兴趣、热门商品进行召回排序
4. 返回 `requestId + list`
5. 推荐区进入视口后，前端调用曝光接口

适合依赖的服务能力：

- `RecommendService`
- `RecommendProfileService`

#### 15.2.2 购物车

调用链建议：

1. 页面整理当前购物车商品 ID
2. 调用 `RecommendService.GetRecommendGoods`
3. 入参 `scene=CART, cartGoodsIds=[...]`
4. 服务优先走关联商品召回，再叠加用户偏好
5. 返回 `requestId + list`
6. 曝光由前端曝光接口上报

适合依赖的服务能力：

- `RecommendService`
- `RecommendRelationService`
- `RecommendProfileService`

#### 15.2.3 我的

调用链建议：

1. 页面调用 `RecommendService.GetRecommendGoods`
2. 入参 `scene=PROFILE`
3. 服务优先读取 `RecommendProfileService` 输出的用户偏好
4. 用户无有效画像时退化到热门兜底

适合依赖的服务能力：

- `RecommendService`
- `RecommendProfileService`

#### 15.2.4 订单详情

调用链建议：

1. 页面获取当前订单 ID
2. 调用 `RecommendService.GetRecommendGoods`
3. 入参 `scene=ORDER_DETAIL, orderId=...`
4. 服务解析订单商品，调用 `RecommendRelationService.ListRelatedGoods`
5. 关联商品不足时再用用户偏好或热门商品补足

适合依赖的服务能力：

- `RecommendService`
- `RecommendRelationService`
- `RecommendProfileService`

#### 15.2.5 支付成功

调用链建议：

1. 页面获取已支付订单 ID
2. 调用 `RecommendService.GetRecommendGoods`
3. 入参 `scene=ORDER_PAID, orderId=...`
4. 服务优先使用订单内商品强关联召回
5. 返回更偏转化导向的推荐结果

适合依赖的服务能力：

- `RecommendService`
- `RecommendRelationService`

### 15.3 前端埋点接口约定

本方案只保留前端调用的埋点接口，不再单独定义后端埋点服务接口。

前端埋点接口建议保留：

- 推荐曝光接口：`POST /api/app/recommend/expose`

不单独提供前端点击埋点接口，原因是：

- 点击通过跳商品详情时携带参数归因；
- 商品详情接口内部可自动记录推荐点击和商品浏览；
- 这样可以减少前端埋点调用数量，降低丢数风险和接入复杂度。

接口归属建议：

- `GET /api/app/recommend/goods` 归属 `RecommendService`
- `POST /api/app/recommend/expose` 归属 `RecommendService`

### 15.4 商品详情接口中的归因调用

商品详情接口建议增加一个统一处理逻辑：

1. 解析 `goodsId`
2. 解析 `source`
3. 解析 `scene`
4. 解析 `requestId`
5. 解析 `index`
6. 在详情业务内部记录一条商品浏览
7. 当 `source=recommend` 时，在详情业务内部同时记录一条推荐点击

这样可以把点击和浏览归因统一封装在商品详情业务内部，而不是额外暴露一组后端埋点服务接口。

## 16. 风险与边界

### 16.1 无法做到完全零前端参与

推荐曝光必须依赖前端触发，因为后端无法感知组件是否真的出现在用户可见区域。

### 16.2 点击定义需要统一

本方案将“成功跳入商品详情页”定义为一次推荐点击。

优点：

- 后端可自动记录；
- 前端改动极小。

缺点：

- 无法捕捉点击后跳转失败的情况；
- 但对当前阶段影响可接受。

### 16.3 曝光的准确度依赖前端可视区判断

建议第一版采用“推荐区首次进入视口时整组曝光一次”，避免复杂实现和重复上报。

### 16.4 未登录用户数据

未登录用户无法稳定建立长期画像。

第一版建议：

- 未登录用户只做会话级场景推荐；
- 以当前页面上下文和热门商品兜底为主；
- 登录后再使用完整个性化画像。

## 17. 实施顺序建议

第一阶段：

- 新增推荐接口；
- 返回 `requestId`；
- 推荐区增加一次曝光上报；
- 推荐位跳详情带归因参数；
- 商品详情接口接收参数并自动记录点击和浏览；
- 加购、收藏、下单、支付后端统一补行为记录。

第二阶段：

- 建立行为聚合任务；
- 形成用户偏好数据；
- 接入场景化召回和排序；
- 建立推荐效果分析报表。

第三阶段：

- 基于曝光、点击、转化数据优化排序；
- 增加位置偏差修正；
- 增加商品相似度和协同过滤能力。

## 18. 最终建议

在“前端尽量少参与”的前提下，最合理的方案不是完全取消前端埋点，而是把前端职责缩减到两个最小动作：

- 推荐曝光时发一次曝光请求；
- 跳详情时把推荐来源参数带上。

除此之外，其余行为全部由后端自动记录。

这是当前阶段在数据质量、实现成本、归因准确度之间最平衡的方案。

## 19. 异步落地方案

### 19.1 第一阶段同步 / 异步边界

第一阶段推荐能力按以下边界落地：

- `recommend_request` 同步落库；
- `recommend_exposure` 异步落库；
- `recommend_click` 异步落库；
- `recommend_goods_view` 异步落库；
- 用户偏好、商品关联等聚合任务继续使用定时任务或异步任务。

设计原则：

- 推荐结果下发与 `requestId` 生成必须同步完成，否则后续归因链不稳定；
- 曝光、点击、浏览属于明细行为，允许短暂延迟，优先降低页面接口 RT；
- 埋点失败不能影响推荐页、商品详情页等主业务接口返回。

### 19.2 推荐行为事件队列

第一阶段新增统一推荐事件队列：

- 队列名：`recommend_event_queue`

事件模型建议统一为：

- `eventType`
- `userId`
- `requestId`
- `scene`
- `source`
- `goodsId`
- `goodsIds`
- `position`
- `occurredAt`

首版支持事件类型：

- `recommend_exposure`
- `recommend_click`
- `goods_view`

### 19.3 第一阶段落地顺序

第一阶段按以下顺序实施：

1. 保持 `recommend_request` 同步落库不变；
2. 新增推荐行为事件结构与队列；
3. 推荐曝光接口改为投递曝光事件；
4. 商品详情接口改为投递点击 / 浏览事件。

### 19.4 第二阶段范围

第二阶段再继续补充：

- 事件消费失败重试与幂等；
- 加购、收藏、下单、支付的推荐归因异步化；
- 用户偏好画像与商品关联聚合任务；
- 基于明细事件的推荐效果报表。
