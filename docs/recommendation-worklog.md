# 推荐能力工作记录

## 当前主题

移动端推荐能力与埋点归因设计。

覆盖页面：

- 首页
- 购物车
- 我的
- 订单详情
- 支付成功

关联设计文档：

- [recommendation-tracking-design.md](/Users/liujun/workspace/shop/shop/docs/recommendation-tracking-design.md)

## 本轮已确认结论

### 1. 推荐能力方向

- 目标不是简单规则推荐过渡，而是直接为后续“用户行为个性化推荐”铺路；
- 不同页面允许使用不同推荐文案；
- 推荐能力需要支持场景化，而不是所有页面共用一套无差别商品流。

### 2. 埋点与归因边界

最终口径已确认：

- 推荐曝光：前端埋点，上报给后端；
- 推荐点击：不单独前端埋点，通过跳商品详情时携带参数，由后端在详情接口中自动归因记录；
- 商品浏览：详情接口携带来源参数，由后端自动记录；
- 加购、收藏、下单、支付：后端业务接口直接记录。

### 3. 商品详情归因参数

详情接口建议支持以下参数：

- `source`
- `scene`
- `requestId`
- `index`

推荐位跳商品详情时，页面路由也应带上这些参数。

### 4. source 与 scene 的定义

`source`：

- 表示入口来源分类；
- 用于全站统一归因；
- 来源不是后端猜测，而是上游跳转入口显式传递。

建议值：

- `recommend`
- `search`
- `cart`
- `favorite`
- `order`
- `direct`

`scene`：

- 表示推荐位所属业务场景；
- 仅在 `source=recommend` 时具有强业务意义；
- 不能替代 `source`。

约束：

- 当 `source = recommend` 时，`scene`、`requestId` 必填，`index` 建议传；
- 当 `source != recommend` 时，`scene`、`requestId`、`index` 可为空；
- 未接入来源参数的入口统一降级为 `direct`。

### 5. 表结构统一前缀

已确认推荐域统一前缀方案：

- 表前缀：`recommend_`
- 服务前缀：`Recommend`

当前文档中已建议的表：

- `recommend_request`
- `recommend_exposure`
- `recommend_click`
- `recommend_goods_view`
- `recommend_user_preference`
- `recommend_user_goods_preference`
- `recommend_goods_relation`

### 6. 服务统一前缀

当前文档中已建议的服务：

- `RecommendService`
- `RecommendProfileService`
- `RecommendRelationService`

## 当前设计状态

已完成：

- 推荐场景与推荐文案方向梳理
- 曝光 / 点击 / 浏览 / 交易行为的职责边界梳理
- `source` / `scene` 的归因规则梳理
- 推荐域表结构与服务命名统一规则设计
- 推荐域表结构字段级草案补充完成
- 推荐域服务接口定义草案补充完成
- 推荐域典型使用场景补充完成
- 推荐域建表 SQL 已生成并执行到本地数据库
- 设计文档初稿完成

未开始：

- proto / API 具体草案
- 数据库 DDL 草案
- 后端服务落地方案
- 前端推荐组件改造方案
- 行为聚合任务设计

## 本轮新增进展

本轮已把设计文档中的表结构部分进一步细化，补充了：

- 每张表的用途
- 详细字段名
- 类型建议
- 字段说明
- 数据来源
- 建议索引
- 表字段去冗余与命名统一审查

本轮还新增了：

- `RecommendService` 接口定义建议
- `RecommendProfileService` 接口定义建议
- `RecommendRelationService` 接口定义建议
- 首页、购物车、我的、订单详情、支付成功五个页面的服务使用场景
- 商品详情接口中的统一归因调用链建议
- “只保留前端埋点接口，不单独定义后端埋点服务接口” 口径确认
- 推荐曝光接口归属 `RecommendService` 已确认
- `RecommendProfileService`、`RecommendRelationService` 的典型使用场景已补充
- `sql/recommend.sql` 已创建
- `shop_test` 数据库已创建推荐域 7 张表

已细化的表包括：

- `recommend_request`
- `recommend_exposure`
- `recommend_click`
- `recommend_goods_view`
- `recommend_user_preference`
- `recommend_user_goods_preference`
- `recommend_goods_relation`

本轮审查后新增统一约定：

- 事件明细表统一使用 `created_at`
- 推荐位序号统一落库为 `position`
- JSON 字段统一使用 `_json` 后缀
- 可通过 `request_id` 回查的字段尽量不重复落库
- 未登录用户统一使用 `user_id = 0`

## 异步埋点方案补充

本轮补充确认了第一阶段埋点异步化边界：

- `recommend_request` 保持同步落库；
- `recommend_exposure` 改为异步事件落库；
- `recommend_click` 改为异步事件落库；
- `recommend_goods_view` 改为异步事件落库；
- 商品详情接口负责投递点击 / 浏览事件，但不等待埋点落库完成。

本轮新增约定：

- 新增统一推荐事件队列 `recommend_event_queue`；
- 第一阶段事件类型包含 `recommend_exposure`、`recommend_click`、`goods_view`；
- 埋点投递失败不影响主业务响应；
- 用户画像、商品关联仍属于第二阶段异步聚合任务，不放入第一阶段消费者。

## 当前代码落地状态

已落实：

1. `recommend_request` 继续同步落库
2. 推荐行为事件结构与队列已定义
3. 推荐曝光接口已改为投递异步事件
4. 商品详情接口已开始接收归因参数并投递点击 / 浏览事件

## 当前建议优先级更新

下一步建议从以下两个方向二选一继续：

1. 继续补数据库 DDL 草案
2. 开始补 proto / API 草案

如果目标是尽快推进后端落地，建议先补数据库 DDL 草案。
如果目标是尽快推进前后端协作边界，建议先补 proto / API 草案。
如果目标是先让后端内部职责稳定，建议先补 repository / usecase 分层草案。

## 当前未决问题

以下问题还没有最终确定：

### 1. 推荐接口 proto 结构

虽然文档中已有字段建议，但还没有形成可直接开发的 proto 草案。

重点待定：

- 推荐接口路径是否最终采用 `/api/app/recommend/goods`
- 推荐接口是单一接口还是按场景拆子接口
- 返回结构中是否保留 `reason`

### 2. 曝光接口粒度

当前建议是一组商品整组曝光一次，但未最终确定：

- 是否按推荐区整组曝光
- 是否需要支持分页二次曝光
- 是否需要单商品曝光明细

### 3. 行为聚合策略

当前只定了方向，没有细化到任务级别：

- 用户偏好按天聚合还是按实时更新
- 商品关联关系如何生成
- 聚合表刷新策略

### 4. 未登录用户推荐策略

当前方向是：

- 以场景上下文 + 热门商品兜底

但还未进一步细化：

- 是否记录设备级匿名行为
- 是否做会话级推荐缓存

## 建议的下一步

优先级建议如下：

1. 先补 `proto / API` 草案
2. 再补数据库表结构草案
3. 然后梳理后端服务落地结构
4. 最后再进入代码实现

原因：

- 当前最大的边界已经定清楚；
- 接口草案会直接约束前后端协作方式；
- 表结构草案会决定埋点数据能否支撑后续个性化推荐。

## 下次继续时的建议入口

下次继续时，建议直接从以下任务开始：

`为推荐能力补 proto / API 草案，基于 docs/recommendation-tracking-design.md 和 docs/recommendation-worklog.md 继续`

这样可以从当前工作结束位置直接接上。

## 最近一次停留点

本次工作结束时，已经完成：

- 推荐设计文档整理；
- 工作记录文档建立；
- 推荐域命名规范统一；
- `source / scene` 规则明确；
- 推荐域表结构字段级草案补齐；
- 推荐域服务接口与使用场景补齐；
- 推荐域数据库表已实际落库到 `shop_test`；
- “曝光前端埋点，点击和浏览后端归因”的方案定稿。

下一步尚未进入 DDL 或 proto / API 细化阶段。
