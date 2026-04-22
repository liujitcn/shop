# 商城移动端推荐标题语义与推荐链路设计

## 1. 文档目标

本文档用于基于当前移动端已经上线或已经写在页面中的推荐标题，规划推荐链路、推荐策略路由和 Gorse 配置取舍。

本次设计有两个前提：

- 不改当前移动端标题文案，只按当前标题寓意反推推荐链路。
- 不为每个页面都单独建设一套推荐器，而是优先复用 Gorse 现有能力，在业务后端做场景路由。

本文档重点回答以下问题：

- 当前每个页面标题分别表达了什么推荐语义
- 这些语义分别应该走哪条推荐链路
- 最终上线目标下，Gorse 推荐配置应如何规划
- 为支撑最终上线目标，后端还需要补齐哪些能力

## 2. 设计依据

### 2.1 当前移动端标题现状

当前移动端推荐标题如下：

| 页面 | 登录标题 | 未登录标题 | 当前场景 |
| --- | --- | --- | --- |
| 首页 | 为你推荐 | 为你推荐 | `HOME` |
| 商品详情 | 看了又看 | 看了又看 | `GOODS_DETAIL` |
| 购物车 | 搭配着买 | 大家都在买 | `CART` |
| 我的 | 根据你的偏好推荐 | 热门好物推荐 | `PROFILE` |
| 订单详情 | 买过这单的人还会买 | 无未登录场景 | `ORDER_DETAIL` |
| 支付成功 | 顺手再带两件 | 无未登录场景 | `ORDER_PAID` |

补充说明：

- 下单确认页当前没有独立推荐位，因此当前标题体系里不包含“创建订单页标题”。
- 若创建订单页后续补推荐位，应单独定义标题并新增对应场景，不建议直接挪用已有标题。

### 2.2 标题语义解释

本次设计按以下方式理解当前标题寓意：

| 标题 | 语义解释 | 不应偏离的推荐方向 |
| --- | --- | --- |
| 为你推荐 | 由系统为当前用户或当前会话挑选的一组综合推荐 | 可以个性化，也可以会话化，但不能明显像“热榜区” |
| 看了又看 | 当前商品引出的继续浏览、相似浏览 | 应优先围绕当前商品做相似或邻近推荐 |
| 搭配着买 | 基于当前购物车商品做搭配、凑单、连带购买推荐 | 应优先围绕购物车商品做会话/购物篮推荐 |
| 大家都在买 | 面向游客的热门购买推荐 | 应优先走热门、非个性化推荐 |
| 根据你的偏好推荐 | 明确表达登录用户画像驱动的个性化推荐 | 应优先走用户推荐 |
| 热门好物推荐 | 明确表达热门商品推荐 | 应优先走非个性化热榜 |
| 买过这单的人还会买 | 强调“买过这些商品的其他人还会继续买什么” | 应优先走 item-to-item / also-buy 逻辑 |
| 顺手再带两件 | 强调支付成功后的即时加购、补单、凑单 | 应优先走基于订单商品的会话推荐 |

### 2.3 当前已有能力

当前项目已经具备以下基础能力：

- 后端已区分推荐主体：登录用户、匿名主体
- 后端已区分推荐场景：`HOME`、`GOODS_DETAIL`、`CART`、`PROFILE`、`ORDER_DETAIL`、`ORDER_PAID`
- 后端已具备推荐上下文提取能力：商品详情、购物车、订单商品、最近行为
- 后端已具备本地兜底能力：同类目兜底、最新热销兜底
- Gorse 已配置：
  - `collaborative`
  - `item-to-item/goods_relation`
  - `user-to-user/similar_users`
  - `non-personalized/hot_30d`
  - `latest`

当前配置见 [gorse/config/config.toml](../gorse/config/config.toml)。

## 3. 推荐能力分层

为避免页面一多就变成“一页一策略”，建议把所有页面收敛为 4 类推荐能力。

### 3.1 用户推荐

含义：

- 根据登录用户长期或中期兴趣进行推荐
- 更适合首页、我的这类 feed 场景

对应 Gorse 能力：

- `GetRecommend`

适用标题：

- 为你推荐
- 根据你的偏好推荐

### 3.2 商品相似推荐

含义：

- 基于当前锚点商品寻找相似商品
- 更适合商品详情页

对应 Gorse 能力：

- `GetNeighbors`
- 或命名推荐器 `item-to-item/goods_relation`

适用标题：

- 看了又看

### 3.3 购物篮 / 会话推荐

含义：

- 基于当前一组上下文商品做连带购买、补单、凑单推荐
- 更适合购物车、支付成功、下单前后场景

对应 Gorse 能力：

- `SessionRecommend`

适用标题：

- 搭配着买
- 顺手再带两件

### 3.4 热门推荐

含义：

- 与用户个性化弱相关，更强调当前平台热门购买趋势
- 更适合游客页、冷启动页、兜底场景

对应 Gorse 能力：

- `non-personalized/hot_30d`

适用标题：

- 大家都在买
- 热门好物推荐

## 4. 场景到链路的详细设计

本节按“标题寓意优先”的原则定义推荐链路。

### 4.1 首页：为你推荐

#### 4.1.1 登录态

标题：`为你推荐`

推荐链路建议：

1. 主链路：`GetRecommend(userId)`
2. 次链路：`SessionRecommend(recentBehaviorGoodsIds)`
3. Gorse 热门兜底：`non-personalized/hot_30d`
4. 本地兜底：`latest` 或当前后端 `pageGoodsIdsByLatest`

设计说明：

- “为你推荐”可以承接个性化推荐，也能承接用户近期会话兴趣，因此主链路优先使用用户推荐。
- 当登录用户行为不足、Gorse 用户推荐为空时，允许回退到最近行为会话推荐，标题含义不会被破坏。
- 不建议登录首页一上来直接走热榜，否则“为你推荐”会和“热门好物推荐”产生含义冲突。

#### 4.1.2 未登录态

标题：`为你推荐`

推荐链路建议：

1. 主链路：`SessionRecommend(recentAnonymousGoodsIds)`
2. 次链路：`non-personalized/hot_30d`
3. 本地兜底：`latest`

设计说明：

- 未登录态没有用户画像，因此不应调用 `GetRecommend`。
- 但“为你推荐”并不强制要求长期个性化，可以理解为“为当前会话挑选”，因此优先使用匿名近期行为的会话推荐。
- 当匿名会话没有任何历史时，再回退到热门推荐。

### 4.2 商品详情：看了又看

#### 4.2.1 登录态与未登录态统一处理

标题：`看了又看`

推荐链路建议：

1. 主链路：`GetNeighbors(goodsId)` 或命名 `item-to-item/goods_relation`
2. 次链路：`SessionRecommend([goodsId])`
3. 本地兜底：同类目商品
4. 最终兜底：`hot_30d` 或 `latest`

设计说明：

- “看了又看”最核心的含义不是“你喜欢什么”，而是“围绕这件商品继续看什么”。
- 因此详情页必须优先以当前商品作为锚点，不能让用户推荐覆盖掉商品相似关系。
- 登录态和未登录态在这里不需要拆成两套主逻辑，因为标题语义本身是商品驱动而不是用户驱动。

### 4.3 购物车：搭配着买 / 大家都在买

#### 4.3.1 登录态

标题：`搭配着买`

推荐链路建议：

1. 主链路：`SessionRecommend(cartGoodsIds)`
2. 次链路：对购物车商品逐个做 `GetNeighbors(goodsId)` 后聚合去重
3. Gorse 热门兜底：`non-personalized/hot_30d`
4. 本地兜底：同类目 + 最新热销

设计说明：

- “搭配着买”明确是购物篮语义，应优先基于购物车商品集合推荐，而不是用户长期偏好。
- `SessionRecommend` 最符合该语义，因为它天然接受一组上下文商品。
- 若 `SessionRecommend` 在购物车场景下召回不足，则以“多商品邻居聚合”作为稳定的第二链路。

#### 4.3.2 未登录态

标题：`大家都在买`

推荐链路建议：

1. 主链路：`non-personalized/hot_30d`
2. 次链路：`latest`
3. 本地兜底：当前 `pageGoodsIdsByLatest`

设计说明：

- 当前游客购物车页不展示购物车商品明细，只展示登录提示和推荐区域。
- 标题已经明确写成“大家都在买”，因此不应强行做会话推荐或个性化推荐。
- 若未来希望游客购物车页使用匿名购物篮推荐，应先改标题，再改策略；在当前标题不变的前提下，维持热门推荐更符合语义。

### 4.4 我的：根据你的偏好推荐 / 热门好物推荐

#### 4.4.1 登录态

标题：`根据你的偏好推荐`

推荐链路建议：

1. 主链路：`GetRecommend(userId)`
2. 次链路：`SessionRecommend(recentCollectGoodsIds or recentBehaviorGoodsIds)`
3. Gorse 热门兜底：`non-personalized/hot_30d`
4. 本地兜底：`latest`

设计说明：

- 这是当前所有标题里用户画像语义最强的一个，因此必须优先走用户推荐。
- 若登录用户历史非常少，可回退到最近收藏、浏览、加购构成的会话推荐，再之后才进入热门兜底。

#### 4.4.2 未登录态

标题：`热门好物推荐`

推荐链路建议：

1. 主链路：`non-personalized/hot_30d`
2. 次链路：`latest`
3. 本地兜底：当前 `pageGoodsIdsByLatest`

设计说明：

- 标题已经明确是热门推荐，因此未登录“我的”页不建议主走 `SessionRecommend`。
- 如果这里主走会话推荐，会造成标题语义与结果来源不一致。

### 4.5 订单详情：买过这单的人还会买

#### 4.5.1 登录态

标题：`买过这单的人还会买`

推荐链路建议：

1. 主链路：基于订单商品聚合调用 `item-to-item/goods_relation`
2. 次链路：`GetNeighbors(orderGoodsId)` 后做多商品聚合去重
3. 第三链路：`SessionRecommend(orderGoodsIds)`
4. Gorse 热门兜底：`non-personalized/hot_30d`
5. 本地兜底：同类目 + 最新热销

设计说明：

- 这个标题不是“为你推荐”，也不是“搭配着买”，而是明显的 “also-buy” 语义。
- 因此订单详情页更适合优先走 item-to-item 或同购关系，不应直接把 `SessionRecommend` 放在第一位。
- 当前配置里的 `item-to-item/goods_relation` 已经存在，因此订单详情页的主链路可以直接围绕该推荐器落地，但后端需要补命名推荐器调用封装。

### 4.6 支付成功：顺手再带两件

#### 4.6.1 登录态

标题：`顺手再带两件`

推荐链路建议：

1. 主链路：`SessionRecommend(orderGoodsIds)`
2. 次链路：基于订单商品聚合调用 `item-to-item/goods_relation`
3. Gorse 热门兜底：`non-personalized/hot_30d`
4. 本地兜底：同类目 + 最新热销

设计说明：

- 支付成功页强调即时补购、凑单和连带购买，因此优先使用订单商品的会话推荐最合适。
- 与订单详情页相比，支付成功页更强调“当下顺手买”，所以 `SessionRecommend` 应排在第一位。

## 5. 当前标题语义下的统一路由规则

为便于后端落地，建议不要把页面标题直接硬编码到推荐代码里，而是先映射成“策略族”，再执行对应链路。

建议新增一个内部策略枚举，例如：

- `feed_recommend`
- `similar_recommend`
- `bundle_recommend`
- `hot_recommend`
- `also_buy_recommend`

推荐场景到策略族的映射建议如下：

| 推荐场景 | 登录态策略族 | 未登录态策略族 |
| --- | --- | --- |
| `HOME` | `feed_recommend` | `feed_recommend` |
| `GOODS_DETAIL` | `similar_recommend` | `similar_recommend` |
| `CART` | `bundle_recommend` | `hot_recommend` |
| `PROFILE` | `feed_recommend` | `hot_recommend` |
| `ORDER_DETAIL` | `also_buy_recommend` | 无 |
| `ORDER_PAID` | `bundle_recommend` | 无 |

这层映射的意义是：

- 页面标题改动不会直接冲击底层推荐器调用逻辑
- 后端可以统一记录“这个请求实际命中了哪类推荐意图”
- 线上报表可以区分“页面场景”和“推荐意图”

## 6. 最终上线目标的 Gorse 配置规划

### 6.1 当前配置能否支撑最终目标

结论：

- 当前配置可以支撑大部分主链路
- 但若按“最终上线目标”交付，仍建议补充两类热门推荐配置，并同步优化反馈定义

原因如下：

- 首页、我的登录态可直接使用 `GetRecommend`
- 商品详情可直接使用 `GetNeighbors`
- 购物车、支付成功可直接使用 `SessionRecommend`
- 游客购物车、游客我的可直接使用 `non-personalized/hot_30d`
- 订单详情所需的 also-buy 语义，当前已有 `item-to-item/goods_relation`

当前已有配置见 [gorse/config/config.toml](../gorse/config/config.toml)：

- `non-personalized/hot_30d`
- `item-to-item/goods_relation`
- `user-to-user/similar_users`
- `collaborative`
- `latest`

### 6.2 最终上线目标下必须补齐的能力

最终上线前，最关键的不是盲目增加很多 recommenders，而是同时补齐“配置能力”和“后端路由能力”。

当前后端包装层主要只显式封装了：

- `GetRecommend`
- `SessionRecommend`

而基于当前标题语义，真正需要补齐的是：

- 命名 `item-to-item/goods_relation` 的调用封装
- 命名 `non-personalized/hot_30d` 的调用封装
- 更细粒度热门推荐器的调用封装
- 多商品聚合去重逻辑
- 场景到策略链的路由逻辑

也就是说，最终上线目标下必须补的是：

1. 后端推荐包装方法
2. 后端场景路由器
3. 推荐请求日志里记录实际命中的在线策略
4. 更细粒度热门推荐配置

## 7. 最终推荐配置方案

最终建议的 Gorse 推荐配置应拆成“基础推荐器”和“热门兜底推荐器”两层。

### 7.1 基础推荐器

基础推荐器建议保留并继续使用：

- `collaborative`
- `item-to-item/goods_relation`
- `user-to-user/similar_users`
- `latest`

用途如下：

- `collaborative`：支撑首页、我的页登录态用户推荐
- `item-to-item/goods_relation`：支撑商品详情的“看了又看”和订单详情的“买过这单的人还会买”
- `user-to-user/similar_users`：作为 `GetRecommend` 的候选来源之一，不直接暴露给业务页面
- `latest`：作为所有链路的最终保底

### 7.2 热门兜底推荐器

最终上线建议至少配置以下 3 个非个性化推荐器：

#### 7.2.1 `hot_30d`

用途：

- 作为全站稳定热门兜底
- 覆盖游客“热门好物推荐”
- 覆盖所有页面的最终热门回退

#### 7.2.2 `hot_7d`

用途：

- 更适合游客首页、游客购物车、游客我的
- 更符合“大家都在买”的近期热度感

建议用途：

- `大家都在买`
- `热门好物推荐`
- 首页匿名态在无匿名历史时的次链路

#### 7.2.3 `hot_pay_30d`

用途：

- 更强调支付成功，而不是纯点击热度
- 更适合“买过这单的人还会买”的兜底候选

建议用途：

- 订单详情兜底
- 支付成功页兜底

设计说明：

- `hot_30d` 解决“稳定热榜”问题
- `hot_7d` 解决“近期热门”问题
- `hot_pay_30d` 解决订单页和支付成功页兜底时“购买语义不够强”的问题

### 7.3 反馈定义优化

为让最终上线后的推荐语义更稳定，建议同步调整反馈定义：

- `read_feedback_types = ["EXPOSURE", "VIEW"]`
- `positive_feedback_types = ["CLICK", "COLLECT", "ADD_CART", "ORDER_CREATE", "ORDER_PAY"]`

调整原因：

- `VIEW` 更适合表达“看过”，不适合作为强正反馈长期污染 item-to-item 和 collaborative
- `ORDER_PAY` 应是最强正反馈，用于强化订单详情和支付成功相关语义
- `ADD_CART` 和 `ORDER_CREATE` 对“搭配着买”“顺手再带两件”也有正向价值

### 7.4 最终上线推荐的配置结论

最终上线目标下，推荐配置建议明确包含：

- `collaborative`
- `item-to-item/goods_relation`
- `user-to-user/similar_users`
- `latest`
- `non-personalized/hot_30d`
- `non-personalized/hot_7d`
- `non-personalized/hot_pay_30d`

## 8. 后端实现建议

### 8.1 `pageGoodsIdsByOnlineRecommend` 入参建议补齐

当前在线推荐方法只有：

- `actor`
- `contextGoodsIds`
- `pageNum`
- `pageSize`

建议补充：

- `scene`
- `goodsId`
- `orderId`

原因：

- 不带 `scene`，无法判断该走 feed、similar、bundle 还是 also-buy
- 不带 `goodsId`，详情页无法明确使用商品锚点
- 不带 `orderId`，订单类页面无法区分订单上下文和普通会话上下文

### 8.2 建议新增在线策略记录字段

建议在推荐请求上下文里增加以下信息：

- `online_strategy`
- `gorse_recommender`

例如：

- `recommend`
- `session`
- `item_neighbors`
- `item_to_item/goods_relation`
- `non_personalized/hot_30d`

这样线上可以直接回答：

- 首页“为你推荐”到底有多少比例真正命中了用户推荐
- 游客购物车“大家都在买”是否真的走了热榜
- 订单详情“买过这单的人还会买”有多少比例退化成了热榜

### 8.3 建议补充的包装方法

建议在 `backend/pkg/recommend` 增加以下包装能力：

- `GetItemToItemGoodsIds`
- `GetNonPersonalizedGoodsIds`
- `GetLatestGoodsIds`
- `MergeNeighborGoodsIds`

说明：

- `GetLatestGoodsIds` 当前 Gorse Go SDK 已支持 `GetLatestItems`
- `GetItemToItemGoodsIds` 和 `GetNonPersonalizedGoodsIds` 需要补自定义调用封装
- `MergeNeighborGoodsIds` 用于订单详情、多商品购物车等需要多锚点聚合的场景

## 9. 最终上线前必须完成的实现项

要满足最终上线目标，建议至少完成以下实现项：

1. 固定当前前端标题，不再让标题与推荐语义脱节
2. 后端补齐“场景 -> 策略族 -> 在线策略链”的路由代码
3. 首页和我的页打通登录/匿名的 feed 与热门分流
4. 商品详情页打通 `GetNeighbors`
5. 购物车和支付成功页打通 `SessionRecommend`
6. 订单详情页打通 `goods_relation` / item-to-item 聚合
7. 后端补齐 `non-personalized/hot_30d`、`hot_7d`、`hot_pay_30d` 调用封装
8. 调整 Gorse 反馈定义，降低 `VIEW` 对强正反馈的干扰
9. 推荐请求日志补齐 `online_strategy` 和 `gorse_recommender`

## 10. 最终结论

基于当前标题寓意，推荐链路应按以下原则落地：

- `为你推荐`：登录优先用户推荐，未登录优先会话推荐
- `看了又看`：优先商品相似推荐
- `搭配着买`：优先购物篮 / 会话推荐
- `大家都在买`：优先热门推荐
- `根据你的偏好推荐`：优先用户推荐
- `热门好物推荐`：优先热门推荐
- `买过这单的人还会买`：优先 also-buy / item-to-item
- `顺手再带两件`：优先订单商品的会话推荐

配置层结论如下：

- 当前配置是一个可运行起点，但还不是最终上线目标
- 最终上线目标建议明确增加 `hot_7d` 和 `hot_pay_30d`
- 最终上线目标建议同步调整 `positive_feedback_types` 与 `read_feedback_types`
- 最终最该补的是后端调用路由、命名推荐器封装和在线策略留痕
