# 推荐行为明细表收敛方案

## 背景

当前推荐行为明细涉及 3 张表：

- `recommend_click`
- `recommend_goods_view`
- `recommend_goods_action`

其中 `click` 与 `view` 事件存在双写：

- `recommend_click` / `recommend_goods_view` 保存专项明细
- `recommend_goods_action` 同时再保存一份统一行为明细

现状会带来以下问题：

- 数据语义重叠，理解成本高
- 统计口径容易分叉
- 写入链路存在双写成本
- 后续扩展事件字段时维护成本高

## 当前现状

### 写入现状

后端推荐事件消费逻辑中：

- `recommend_click` 事件会写入 `recommend_click`，随后再写入 `recommend_goods_action`
- `goods_view` 事件会写入 `recommend_goods_view`，随后再写入 `recommend_goods_action`
- `collect/cart/order/pay` 只写入 `recommend_goods_action`

对应代码：

- [recommend.go](/Users/liujun/workspace/shop/shop/backend/service/app/biz/recommend.go#L560)

### 表职责现状

#### `recommend_click`

职责：推荐点击专项明细表。

主要字段：

- `actor_type`
- `actor_id`
- `scene`
- `goods_id`
- `request_id`
- `position`
- `source`
- `created_at`

定义：

- [recommend_click.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/recommend_click.gen.go#L11)

#### `recommend_goods_view`

职责：商品浏览专项明细表。

主要字段：

- `actor_type`
- `actor_id`
- `goods_id`
- `source`
- `scene`
- `request_id`
- `position`
- `view_mode`
- `created_at`

定义：

- [recommend_goods_view.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/recommend_goods_view.gen.go#L11)

#### `recommend_goods_action`

职责：统一推荐行为明细表。

已覆盖事件：

- `recommend_click`
- `goods_view`
- `goods_collect`
- `goods_cart`
- `order_create`
- `order_pay`

主要字段：

- `event_type`
- `actor_type`
- `actor_id`
- `goods_id`
- `goods_num`
- `source`
- `scene`
- `request_id`
- `position`
- `created_at`

定义：

- [recommend_goods_action.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/recommend_goods_action.gen.go#L11)

## 是否重复

结论：**部分重复，且属于有意冗余设计。**

### 重复部分

对于 `click` 和 `view` 两类事件：

- `recommend_click` 与 `recommend_goods_action(event_type='recommend_click')` 语义重叠
- `recommend_goods_view` 与 `recommend_goods_action(event_type='goods_view')` 语义重叠

两者都保存了大部分核心字段：

- 主体
- 商品
- 来源
- 场景
- 请求 ID
- 推荐位序号
- 时间

### 不完全重复的部分

- `recommend_goods_action` 还承载 `collect/cart/order/pay`
- `recommend_goods_view` 独有 `view_mode`
- 专表在当前实现里承担了部分专项查询和索引优化职责

## 当前读取依赖

在决定收敛前，必须先识别读取方。

### 仍直接依赖 `recommend_click` 的逻辑

#### 1. 在线排序中的点击惩罚

当前 `loadActorExposurePenalties` 直接查询 `recommend_click` 统计点击次数：

- [recommend_rank.go](/Users/liujun/workspace/shop/shop/backend/service/app/biz/recommend_rank.go#L482)

#### 2. 推荐商品日统计任务

当前 `click_count` 直接来自 `recommend_click`：

- [recommend_goods_stat_day.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_goods_stat_day.go#L131)

#### 3. 商品关联重建任务

当前 `co_click` 关系直接来自 `recommend_click`：

- [recommend_rebuild.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_rebuild.go#L401)

### 已经依赖 `recommend_goods_action` 的逻辑

#### 1. 用户商品偏好重建

- [recommend_rebuild.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_rebuild.go#L222)

#### 2. 用户类目偏好重建

- [recommend_rebuild.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_rebuild.go#L289)

#### 3. 推荐商品日统计中的浏览、收藏、加购、下单、支付统计

- [recommend_goods_stat_day.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_goods_stat_day.go#L138)

## 建议目标

建议将 `recommend_goods_action` 收敛为**唯一行为事实表**。

最终目标：

- 保留 `recommend_goods_action`
- 下线 `recommend_click`
- 下线 `recommend_goods_view`

### 为什么建议这样做

原因如下：

- 统一事实表后，统计口径更清晰
- 新增事件类型时不需要继续加专表
- 画像、统计、关联、惩罚都可以走统一行为表
- 可以减少双写和潜在不一致问题

## 目标表设计建议

如果 `recommend_goods_action` 成为唯一事实表，需要补齐 `view` 的专有信息。

推荐两种方案：

### 方案 A：新增 `view_mode`

新增字段：

- `view_mode varchar(16) null`

优点：

- 改动小
- 只解决当前 `goods_view` 的字段缺口

缺点：

- 后续其他事件若有专有字段，还要继续加列

### 方案 B：新增 `event_meta_json`

新增字段：

- `event_meta_json json null`

示例：

```json
{
  "viewMode": "detail_open"
}
```

优点：

- 扩展性更好
- 不需要为每种行为类型反复加字段

缺点：

- 查询专项字段时比独立列稍弱

### 推荐选择

建议优先采用 **方案 B：`event_meta_json`**。

原因：

- 当前项目推荐事件还在演进
- 后续可能出现更多行为扩展字段
- 用统一扩展字段更稳

## 迁移原则

迁移必须遵循：

- **先改读，后停写**
- **先对账，后删表**
- **确保可回滚**

不建议直接删除专表并一次性切换。

## 迁移步骤

### 第 1 步：补齐统一表字段

在 `recommend_goods_action` 上新增：

- `event_meta_json`

同时调整写入逻辑：

- `goods_view` 事件写入 `event_meta_json.viewMode`

此阶段仍保留旧的双写逻辑，不改变读路径。

### 第 2 步：把读取方切换到统一表

逐步将所有依赖 `recommend_click` / `recommend_goods_view` 的读取改为依赖 `recommend_goods_action`。

建议优先改以下位置：

#### 在线排序

将：

- `recommend_click`

替换为：

- `recommend_goods_action where event_type = 'recommend_click'`

对应位置：

- [recommend_rank.go](/Users/liujun/workspace/shop/shop/backend/service/app/biz/recommend_rank.go#L482)

#### 推荐商品日统计

将：

- `click_count` 来自 `recommend_click`

替换为：

- `recommend_goods_action where event_type = 'recommend_click'`

对应位置：

- [recommend_goods_stat_day.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_goods_stat_day.go#L131)

#### 商品关联重建

将：

- `co_click` 来自 `recommend_click`

替换为：

- `recommend_goods_action where event_type = 'recommend_click'`

对应位置：

- [recommend_rebuild.go](/Users/liujun/workspace/shop/shop/backend/service/admin/task/recommend_rebuild.go#L401)

如果未来有浏览专项读取，也统一改成：

- `recommend_goods_action where event_type = 'goods_view'`

### 第 3 步：保留双写并做一致性校验

在读已切换、写仍双写的阶段，持续做对账。

建议至少校验最近 7 天数据：

- `recommend_click` 行数 vs `recommend_goods_action(event_type='recommend_click')`
- `recommend_goods_view` 行数 vs `recommend_goods_action(event_type='goods_view')`
- 分天统计是否一致
- 分场景统计是否一致
- 分商品统计是否一致

建议额外观察：

- 首页 CTR 是否异常
- 排序结果是否漂移
- `loadActorExposurePenalties` 的惩罚分布是否异常

### 第 4 步：停专表写入

确认一致性后，再移除双写逻辑：

- `recommend_click` 不再写入
- `recommend_goods_view` 不再写入

只保留写入 `recommend_goods_action`。

### 第 5 步：保留观察期

停写后建议保留旧表 2 到 4 周：

- 不再作为业务依赖
- 仅作为回滚缓冲和人工核对使用

### 第 6 步：清理

确认无读写依赖后，执行最终清理：

- 删除 `RecommendClickRepo`
- 删除 `RecommendGoodsViewRepo`
- 删除旧表

保守做法可以先：

- `rename table recommend_click to recommend_click_bak`
- `rename table recommend_goods_view to recommend_goods_view_bak`

## 推荐后的统一口径

统一为 `recommend_goods_action` 后，事件口径如下：

- 点击：`event_type = 'recommend_click'`
- 浏览：`event_type = 'goods_view'`
- 收藏：`event_type = 'goods_collect'`
- 加购：`event_type = 'goods_cart'`
- 下单：`event_type = 'order_create'`
- 支付：`event_type = 'order_pay'`

统计、画像、关联、排序惩罚都应基于同一行为事实表建设。

## 风险点

### 1. `view_mode` 丢失风险

如果不补齐 `view_mode` 对应字段，`recommend_goods_view` 的专项信息会丢失。

### 2. 索引压力上升

专表下线后，所有行为查询会集中到 `recommend_goods_action`。

建议至少补充或评估以下索引：

- `(event_type, created_at)`
- `(actor_type, actor_id, scene, created_at, event_type)`
- `(request_id, goods_id, event_type)`
- `(goods_id, created_at, event_type)`

### 3. SQL 任务回归风险

离线任务 SQL 直接引用表名，迁移时不能只改 Go 层代码，必须同步修改任务 SQL。

### 4. 排序结果回归风险

在线排序中的点击惩罚逻辑较敏感，切换读取口径后必须观察推荐结果是否明显变化。

## 建议结论

建议采用以下方向：

1. 以 `recommend_goods_action` 作为唯一行为事实表
2. 为统一表补齐事件扩展字段，推荐 `event_meta_json`
3. 先切换读取，再移除双写
4. 对账稳定后再下线 `recommend_click` 与 `recommend_goods_view`

## 明日继续时建议优先处理的事项

1. 明确统一表扩展字段选型：`view_mode` 还是 `event_meta_json`
2. 列出迁移 SQL 草案
3. 先改读取链路，不动前端与埋点协议
4. 补一份对账 SQL，保证迁移可验证

