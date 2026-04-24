# 商品评价功能轻量设计方案

## 1. 目标

商品评价作为独立业务域建设，先满足“用户能评价、商品能展示、后台能审核”的闭环，不引入复杂营销、AI 分析或深层社交能力。

一期目标：

- App 端支持订单商品评价、商品详情评价摘要、评价列表。
- Admin 端支持评价列表、详情查看、审核通过、驳回、隐藏。
- Backend 端支持评价资格校验、评价写入、评价展示和基础统计。
- SQL 侧补齐评价主表、媒体表和商品评价统计表。

一期暂不做：

- 追评、楼中楼讨论、点赞点踩等互动能力。
- 评价有奖、积分、返现等营销能力。
- AI 总结、情感分析、评价质量分等智能能力。
- 与推荐主链路强绑定。

## 2. 现状判断

当前仓库已经具备评价功能的基础入口和数据条件：

- App 端订单列表、订单详情已有“去评价”入口位。
- 订单状态 `RECEIVED` 表示已收货或已完成，不能直接扩展为“已评价”。
- `order_goods` 保存了商品快照、规格快照，适合作为评价资格校验和评价归属基础。
- 商品详情页已有“商品 / 详情 / 推荐”分段，缺少评价摘要和评价列表入口。
- Admin 端暂无评价审核管理页面。

因此本方案采用 `order_goods` 粒度做评价资格，避免多商品订单下整单状态混乱。

## 3. 功能范围

### 3.1 App 端

- 商品详情页新增评价摘要区。
- 商品详情页增加“评价”入口，可进入评价列表页。
- 订单详情页或订单列表页点击“去评价”进入评价提交页。
- 一个订单可包含多个待评价商品，每个 `order_goods` 单独提交评价。
- 评价内容支持评分、文字、图片、匿名展示。

### 3.2 Admin 端

- 评价分页列表。
- 评价详情查看。
- 审核通过、驳回、隐藏。
- 查看商品维度评价统计。

一期不做精选排序、讨论回复管理和复杂看板，避免后台范围过大。

### 3.3 Backend 端

- 校验订单商品是否属于当前用户。
- 校验订单是否已收货。
- 校验同一 `order_goods` 只能提交一次主评价。
- 写入评价、图片和统计数据。
- 提供商品评价摘要、评价分页和后台审核接口。

## 4. 页面设计

### 4.1 商品详情页

当前分段建议从：

```text
商品 / 详情 / 推荐
```

调整为：

```text
商品 / 评价 / 详情 / 推荐
```

评价摘要展示：

- 总评价数。
- 好评率。
- 带图评价数。
- 高频标签，最多展示 3 个。
- 最新或优质评价预览 1 到 2 条。
- “查看全部评价”入口。

### 4.2 评价列表页

筛选先保留三类：

- 全部。
- 带图。
- 好评。

排序先保留两类：

- 默认。
- 最新。

列表项展示：

- 用户昵称或匿名昵称。
- 用户头像。
- 评分星级。
- 规格快照。
- 评价正文。
- 图片列表。
- 创建时间。

### 4.3 订单评价页

提交字段：

- 商品快照信息。
- 评分，1 到 5 分。
- 评价正文。
- 图片，建议最多 9 张。
- 匿名开关。

提交后：

- `order_goods.comment_status` 更新为已评价。
- 评价默认进入待审核状态。
- 前台商品详情只展示审核通过的评价。

## 5. 核心规则

### 5.1 评价资格

- 只有订单状态为 `RECEIVED` 的订单商品允许评价。
- 评价资格按 `order_goods` 校验，不按整单校验。
- 一条 `order_goods` 只允许一条主评价。
- 评价提交人必须是订单所属用户。

### 5.2 审核展示

- 用户提交后状态为待审核。
- 审核通过后才在商品详情和评价列表展示。
- 审核驳回和隐藏状态不对前台展示。
- 后台能看到全部状态评价。

### 5.3 匿名展示

- 匿名只影响前台展示，不影响后台真实用户归属。
- 匿名昵称统一脱敏，例如 `用户***123`。

### 5.4 统计更新

- 评价通过审核后计入商品评价统计。
- 评价被隐藏后从前台统计中扣除。
- 一期可以在审核事务后同步更新统计表，后续量大再改为异步任务。

## 6. 数据模型

### 6.1 调整 `order_goods`

新增字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `comment_status` | `tinyint` | 评价状态：`0 未评价 / 1 已评价` |
| `comment_id` | `bigint` | 主评价 ID |
| `commented_at` | `datetime` | 评价时间 |

建议索引：

- `idx_order_goods_order_id_comment_status(order_id, comment_status)`

### 6.2 新增 `goods_comment`

用于保存评价主记录。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `order_id` | `bigint` | 订单 ID |
| `order_goods_id` | `bigint` | 订单商品 ID |
| `goods_id` | `bigint` | 商品 ID |
| `user_id` | `bigint` | 用户 ID |
| `sku_code` | `varchar(64)` | 规格编码 |
| `sku_desc_snapshot` | `varchar(255)` | 规格快照 |
| `goods_name_snapshot` | `varchar(255)` | 商品名称快照 |
| `goods_picture_snapshot` | `varchar(1024)` | 商品图片快照 |
| `score` | `tinyint` | 评分，1 到 5 |
| `content` | `varchar(2000)` | 评价正文 |
| `is_anonymous` | `tinyint(1)` | 是否匿名 |
| `status` | `tinyint` | `1 待审核 / 2 已通过 / 3 已驳回 / 4 已隐藏` |
| `media_count` | `int` | 图片数量 |
| `created_at` | `datetime` | 创建时间 |
| `updated_at` | `datetime` | 更新时间 |
| `deleted_at` | `datetime` | 删除时间 |

建议索引：

- 唯一索引 `unique_goods_comment_order_goods_id(order_goods_id)`
- 普通索引 `idx_goods_comment_goods_id_status_created_at(goods_id, status, created_at)`
- 普通索引 `idx_goods_comment_user_id_created_at(user_id, created_at)`

### 6.3 新增 `goods_comment_media`

用于保存评价图片。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `comment_id` | `bigint` | 评价 ID |
| `url` | `varchar(1024)` | 图片地址 |
| `sort` | `int` | 排序 |
| `created_at` | `datetime` | 创建时间 |

建议索引：

- `idx_goods_comment_media_comment_id_sort(comment_id, sort)`

### 6.4 新增 `goods_comment_stat`

用于商品详情页快速读取评价统计。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `goods_id` | `bigint` | 商品 ID |
| `total_count` | `int` | 审核通过评价数 |
| `good_count` | `int` | 好评数，建议评分 4 到 5 |
| `middle_count` | `int` | 中评数，建议评分 3 |
| `bad_count` | `int` | 差评数，建议评分 1 到 2 |
| `with_media_count` | `int` | 带图评价数 |
| `good_rate` | `int` | 好评率，百分比整数 |
| `updated_at` | `datetime` | 更新时间 |

建议索引：

- 唯一索引 `unique_goods_comment_stat_goods_id(goods_id)`

## 7. 接口设计

建议新增独立服务，不复用推荐服务：

- App 侧：`GoodsCommentService`
- Admin 侧：`GoodsCommentAdminService`

### 7.1 App 侧接口

#### 获取商品评价摘要

`GET /api/app/goods/comment/summary`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `goodsId` | `int64` | 是 | 商品 ID |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `goodsId` | `int64` | 商品 ID |
| `totalCount` | `int32` | 总评价数 |
| `goodRate` | `int32` | 好评率 |
| `withMediaCount` | `int32` | 带图评价数 |
| `previewList` | `[]CommentPreview` | 预览评价 |

#### 获取商品评价分页

`GET /api/app/goods/comment/list`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `goodsId` | `int64` | 是 | 商品 ID |
| `filter` | `string` | 否 | `ALL / MEDIA / GOOD` |
| `sort` | `string` | 否 | `DEFAULT / LATEST` |
| `pageNum` | `int64` | 是 | 页码 |
| `pageSize` | `int64` | 是 | 每页数量 |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `list` | `[]GoodsComment` | 评价列表 |
| `total` | `int32` | 总数 |

#### 获取订单评价表单

`GET /api/app/order/comment/form`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `orderId` | `int64` | 是 | 订单 ID |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `orderId` | `int64` | 订单 ID |
| `goodsList` | `[]CommentableGoods` | 可评价商品列表 |

#### 提交评价

`POST /api/app/order/comment/submit`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `orderId` | `int64` | 是 | 订单 ID |
| `items` | `[]SubmitCommentItem` | 是 | 评价商品列表 |

`SubmitCommentItem` 字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `orderGoodsId` | `int64` | 是 | 订单商品 ID |
| `score` | `int32` | 是 | 评分，1 到 5 |
| `content` | `string` | 是 | 评价正文 |
| `mediaUrls` | `[]string` | 否 | 图片地址 |
| `isAnonymous` | `bool` | 否 | 是否匿名 |

### 7.2 Admin 侧接口

#### 评价分页管理

`GET /api/admin/goods/comment/list`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `goodsId` | `int64` | 商品 ID |
| `orderId` | `int64` | 订单 ID |
| `userKeyword` | `string` | 用户关键词 |
| `status` | `int32` | 审核状态 |
| `score` | `int32` | 评分 |
| `hasMedia` | `bool` | 是否带图 |
| `pageNum` | `int64` | 页码 |
| `pageSize` | `int64` | 每页数量 |

#### 获取评价详情

`GET /api/admin/goods/comment/detail`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `commentId` | `int64` | 评价 ID |

#### 更新评价状态

`PUT /api/admin/goods/comment/status`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `commentId` | `int64` | 评价 ID |
| `status` | `int32` | `2 通过 / 3 驳回 / 4 隐藏` |
| `rejectReason` | `string` | 驳回原因 |

#### 商品评价统计

`GET /api/admin/goods/comment/stat`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `goodsId` | `int64` | 商品 ID |
| `pageNum` | `int64` | 页码 |
| `pageSize` | `int64` | 每页数量 |

## 8. 落地影响范围

### 8.1 Backend

- `backend/api/protos/app`：新增评价 App 侧 proto。
- `backend/api/protos/admin`：新增评价 Admin 侧 proto。
- `backend/service/app`：新增评价提交、摘要、列表服务。
- `backend/service/admin`：新增评价审核管理服务。
- `backend/pkg/gen`：通过生成命令补齐评价相关查询代码。
- `sql`：补充评价表和后台权限初始化脚本。

### 8.2 App

- `frontend/app/src/pages/goods/goods.vue`：新增评价摘要区和评价入口。
- `frontend/app/src/pagesOrder`：新增订单评价提交页。
- `frontend/app/src/api/app`：新增评价接口封装。
- `frontend/app/src/rpc/app`：通过生成命令更新评价 RPC 类型。

### 8.3 Admin

- `frontend/admin/src/views`：新增评价列表和详情审核页面。
- `frontend/admin/src/api/admin`：新增评价后台接口封装。
- `frontend/admin/src/rpc/admin`：通过生成命令更新评价 RPC 类型。

## 9. 推荐实施顺序

### 9.1 一期闭环

1. 新增 SQL 表结构和 `order_goods` 评价状态字段。
2. 新增 App、Admin 评价 proto，并执行项目既有生成命令。
3. 实现 App 侧评价表单、提交、摘要和列表接口。
4. 实现 Admin 侧评价列表、详情和审核接口。
5. App 商品详情页接入评价摘要和评价列表。
6. App 订单页接入评价提交页。
7. Admin 接入评价审核页面。

### 9.2 后续增强

- 追评。
- 标签模板。
- 精选评价。
- 讨论回复。
- 点赞、有用等互动。
- 评价统计看板。

## 10. 风险与注意点

- 不要把 `order_info.status` 扩展为“已评价”，订单生命周期和评价状态应分离。
- 必须按 `order_goods` 校验评价资格，避免多商品订单重复或漏评价。
- 图片上传复用现有上传能力，评价模块只保存图片 URL。
- 前台只查询审核通过评价，后台查询全部状态评价。
- 审核状态变化后必须同步刷新 `goods_comment_stat`，避免详情页统计不一致。
- 生成代码必须通过项目既有生成命令更新，不手写生成产物。

## 11. 结论

一期优先做轻量闭环：

```text
订单商品 -> 提交评价 -> 后台审核 -> 商品详情展示 -> 评价列表展示
```

该方案保持评价域独立，以 `order_goods` 作为资格基础，以 `goods_comment` 作为核心表，以 `goods_comment_stat` 支撑商品详情高频读取。后续追评、讨论、精选和互动能力可以在不推翻一期模型的基础上继续扩展。
