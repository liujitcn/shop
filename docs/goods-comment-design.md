# 商品评价功能设计方案

## 1. 文档目标

本文档用于设计商城“商品评价”功能，覆盖以下内容：

- App 端商品详情评价摘要、评价列表、订单评价、追评、讨论回复。
- Admin 端评价审核、精选、隐藏、统计。
- Backend 端评价域的数据模型、聚合逻辑、接口契约。
- SQL 侧需要新增或调整的表结构。

本文档只讨论设计方案，不包含本次直接开发实现。

## 2. 现状说明

结合当前仓库实现，评价功能具备以下基础条件：

- App 端订单列表与订单详情已经存在“去评价”业务入口位，但尚未接入真实评价流程。
- 订单状态 `RECEIVED` 当前表示“已收货/已完成”，不适合直接复用为“已评价”状态。
- 订单商品 `order_goods` 已保存商品快照、规格快照、推荐归因字段，适合作为评价资格校验的基础表。
- 商品详情页当前已有“商品 / 详情 / 推荐”分段，但缺少“评价”分段和评价摘要区。
- 后台当前没有独立的评价审核、精选、回复管理页面。

这意味着当前系统“具备评价入口和订单基础数据”，但“缺少完整评价域模型与前后端业务闭环”。

## 3. 设计目标

### 3.1 目标

- 支持按订单商品维度提交评价，避免一个订单多商品时评价关系混乱。
- 支持主评、追评、图文/视频评价、标签评价。
- 支持商品详情页展示评价摘要和评价列表。
- 支持评价下的讨论回复，效果贴近京东“全部讨论”弹层。
- 支持后台审核、隐藏、精选、排序和统计。
- 支持后续把评价质量、评价标签、讨论热度接入推荐或搜索排序，但不与现有推荐服务强耦合。

### 3.2 非目标

- 一期不做复杂的 AI 评价总结、情感分析、内容改写。
- 一期不做店铺回复、客服身份回复、多角色对话体系。
- 一期不做评价有奖、评价积分、评价返现等营销能力。
- 一期不修改现有推荐主链路，不把评价接口塞入 `RecommendService`。

## 4. 功能范围

### 4.1 App 端

- 商品详情页评价摘要区。
- 商品评价列表页。
- 订单评价提交页。
- 追评提交能力。
- 评价讨论弹层和回复能力。
- 评价有用、点踩、点赞等轻互动能力。

### 4.2 Admin 端

- 商品评价分页管理。
- 评价详情查看。
- 评价审核通过、驳回、隐藏。
- 精选评价、排序调整。
- 讨论回复管理。
- 评价统计看板。

### 4.3 Backend 端

- 评价资格校验。
- 评价写入、追评写入、回复写入。
- 评价聚合统计。
- 商品详情评价摘要接口。
- 商品评价列表接口。
- 后台审核与统计接口。

## 5. 页面与交互设计

### 5.1 商品详情页

建议将当前分段从：

```text
商品 / 详情 / 推荐
```

调整为：

```text
商品 / 评价 / 详情 / 推荐
```

评价区展示内容：

- 评价总数。
- 好评率。
- 图/视频评价数量。
- 追评数量。
- 高频标签。
- 2 到 3 条精选评价预览。
- “查看全部评价”入口。

### 5.2 评价列表页

顶部筛选建议参考京东，但先做轻量版：

- 全部
- 图/视频
- 追评
- 当前规格
- 标签筛选

排序建议：

- 默认
- 最新
- 最热

列表项展示：

- 用户昵称或匿名昵称。
- 用户头像。
- 评分星级。
- 规格快照。
- 评价正文。
- 图片/视频列表。
- 创建时间。
- 点赞/有用数。
- 追评内容。
- 讨论数。

### 5.3 订单评价页

提交入口：

- 订单详情页“去评价”。
- 订单列表页待评价商品入口。

提交形式：

- 一个订单可同时评价多个 `order_goods`。
- 每个商品单独填写评分、内容、标签、图片/视频、匿名状态。
- 支持批量提交，但后台仍然按商品拆成独立评价记录。

### 5.4 讨论弹层

对标你给的截图，采用“评价详情内单独讨论流”而不是把所有回复平铺到评价列表。

建议交互：

- 打开“全部讨论”底部弹层。
- 首屏显示一级回复列表。
- 某条一级回复下仅预览最近 1 到 2 条子回复。
- 点击“展开”后异步拉取该一级回复的子回复列表。

这样既能还原截图效果，也便于接口分页和性能控制。

## 6. 核心业务规则

### 6.1 评价资格

- 只有订单状态为 `RECEIVED` 的订单商品允许发起评价。
- 评价资格按 `order_goods` 粒度校验，不按整单校验。
- 同一条 `order_goods` 只允许存在一条主评。
- 同一条主评只允许存在一条追评。

### 6.2 主评与追评

- 主评是首次评价。
- 追评依附于主评，不单独作为新的评价入口展示。
- 商品详情列表默认展示“主评 + 追评摘要”。
- 追评提交后，需要同步更新评价统计中的 `append_count`。

### 6.3 审核与展示

- 用户提交后先进入审核状态。
- 审核通过后才能在商品详情页和评价列表页展示。
- 审核驳回的评价默认不对外展示。
- 后台可将评价设为精选，商品详情摘要优先抽取精选评价。

### 6.4 匿名展示

- 匿名只影响前台展示，不影响后台和数据库真实 `user_id` 归属。
- 匿名昵称建议统一脱敏，例如 `颜***y`。

### 6.5 讨论规则

- 讨论只能挂在已通过审核的评价下。
- 讨论回复最多支持两层展示语义：
  - 一级回复：直接回复评价。
  - 二级回复：回复某条一级或二级回复。
- 数据表允许树形关系，但前台默认只做两层展示，避免线程过深。

## 7. 数据模型设计

### 7.1 调整现有表：`order_goods`

现有 `order_goods` 保存了订单商品快照，非常适合作为评价资格与评价归属基础。

建议新增字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `comment_status` | `tinyint` | 评价状态：`0 未评价 / 1 已评价 / 2 已追评` |
| `comment_id` | `bigint` | 主评 ID |
| `commented_at` | `datetime` | 主评时间 |
| `append_status` | `tinyint` | 追评状态：`0 未追评 / 1 已追评` |
| `appended_at` | `datetime` | 追评时间 |

建议索引：

- `idx_order_goods_order_id_comment_status(order_id, comment_status)`
- `idx_order_goods_goods_id_comment_status(goods_id, comment_status)`

### 7.2 新增表：`goods_comment`

用于保存主评和追评主记录。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `root_comment_id` | `bigint` | 主评 ID，主评时等于自身 |
| `comment_type` | `tinyint` | `1 主评 / 2 追评` |
| `order_id` | `bigint` | 订单 ID |
| `order_goods_id` | `bigint` | 订单商品 ID |
| `goods_id` | `bigint` | 商品 ID |
| `user_id` | `bigint` | 用户 ID |
| `sku_code` | `varchar(64)` | 规格编号 |
| `sku_desc_snapshot` | `varchar(255)` | 规格文案快照 |
| `goods_name_snapshot` | `varchar(255)` | 商品名称快照 |
| `goods_picture_snapshot` | `varchar(1024)` | 商品图片快照 |
| `score` | `tinyint` | 评分，1 到 5 |
| `content` | `varchar(2000)` | 评价正文 |
| `is_anonymous` | `tinyint(1)` | 是否匿名 |
| `status` | `tinyint` | `1 待审核 / 2 已通过 / 3 已驳回 / 4 已隐藏` |
| `is_featured` | `tinyint(1)` | 是否精选 |
| `featured_sort` | `int` | 精选排序 |
| `helpful_count` | `int` | 有用数 |
| `dislike_count` | `int` | 点踩数 |
| `reply_count` | `int` | 讨论数 |
| `media_count` | `int` | 媒体数 |
| `created_at` | `datetime` | 创建时间 |
| `updated_at` | `datetime` | 更新时间 |
| `deleted_at` | `datetime` | 删除时间 |

建议索引：

- 唯一索引 `unique_goods_comment(order_goods_id, comment_type)`
- 普通索引 `idx_goods_comment_goods_id_status_created_at(goods_id, status, created_at)`
- 普通索引 `idx_goods_comment_user_id_created_at(user_id, created_at)`
- 普通索引 `idx_goods_comment_root_comment_id(root_comment_id)`

### 7.3 新增表：`goods_comment_media`

用于保存评价图片和视频。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `comment_id` | `bigint` | 评价 ID |
| `media_type` | `tinyint` | `1 图片 / 2 视频` |
| `url` | `varchar(1024)` | 资源地址 |
| `cover_url` | `varchar(1024)` | 视频封面 |
| `width` | `int` | 宽度 |
| `height` | `int` | 高度 |
| `duration` | `int` | 视频时长，秒 |
| `sort` | `int` | 排序 |
| `created_at` | `datetime` | 创建时间 |

建议索引：

- `idx_goods_comment_media_comment_id_sort(comment_id, sort)`

### 7.4 新增表：`goods_comment_tag_rel`

用于保存评价与标签关系。

一期不强制单独建标签主表，标签模板可先复用字典能力。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `goods_id` | `bigint` | 商品 ID |
| `comment_id` | `bigint` | 评价 ID |
| `tag_name` | `varchar(64)` | 标签名称 |
| `source_type` | `tinyint` | `1 用户勾选 / 2 后台模板 / 3 系统回填` |
| `created_at` | `datetime` | 创建时间 |

建议索引：

- `idx_goods_comment_tag_rel_goods_id_tag_name(goods_id, tag_name)`
- `idx_goods_comment_tag_rel_comment_id(comment_id)`

### 7.5 新增表：`goods_comment_reply`

用于保存评价下的讨论与楼中楼回复。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `goods_id` | `bigint` | 商品 ID |
| `comment_id` | `bigint` | 评价 ID |
| `root_reply_id` | `bigint` | 一级回复 ID，一级回复时等于自身 |
| `parent_reply_id` | `bigint` | 父回复 ID |
| `user_id` | `bigint` | 回复用户 ID |
| `reply_to_user_id` | `bigint` | 被回复用户 ID |
| `content` | `varchar(1000)` | 回复内容 |
| `like_count` | `int` | 点赞数 |
| `status` | `tinyint` | `1 正常 / 2 隐藏 / 3 删除` |
| `created_at` | `datetime` | 创建时间 |
| `updated_at` | `datetime` | 更新时间 |
| `deleted_at` | `datetime` | 删除时间 |

建议索引：

- `idx_goods_comment_reply_comment_id_root_reply_id_created_at(comment_id, root_reply_id, created_at)`
- `idx_goods_comment_reply_user_id_created_at(user_id, created_at)`

### 7.6 新增表：`goods_comment_action`

用于记录用户对评价和回复的互动。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `user_id` | `bigint` | 用户 ID |
| `target_type` | `tinyint` | `1 评价 / 2 回复` |
| `target_id` | `bigint` | 目标 ID |
| `action_type` | `tinyint` | `1 有用 / 2 点踩 / 3 点赞` |
| `created_at` | `datetime` | 创建时间 |

建议索引：

- 唯一索引 `unique_goods_comment_action(user_id, target_type, target_id, action_type)`
- 普通索引 `idx_goods_comment_action_target_type_target_id(target_type, target_id)`

### 7.7 新增表：`goods_comment_stat`

用于保存商品评价聚合结果，降低商品详情页查询成本。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | `bigint` | 主键 |
| `goods_id` | `bigint` | 商品 ID |
| `total_count` | `int` | 总评价数 |
| `good_count` | `int` | 好评数 |
| `middle_count` | `int` | 中评数 |
| `bad_count` | `int` | 差评数 |
| `with_media_count` | `int` | 带图/视频数 |
| `with_video_count` | `int` | 带视频数 |
| `append_count` | `int` | 追评数 |
| `discussion_count` | `int` | 讨论数 |
| `helpful_count` | `int` | 有用总数 |
| `good_rate` | `int` | 好评率，百分比整数 |
| `updated_at` | `datetime` | 更新时间 |

建议索引：

- 唯一索引 `unique_goods_comment_stat(goods_id)`

## 8. 接口设计

建议新增独立服务：

- App 侧：`GoodsCommentService`
- Admin 侧：`GoodsCommentAdminService`

不要复用现有 `RecommendService`，避免评价域和推荐域语义混杂。

### 8.1 App 侧接口

#### 8.1.1 获取商品评价摘要

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
| `withMediaCount` | `int32` | 图/视频评价数 |
| `appendCount` | `int32` | 追评数 |
| `tagList` | `[]TagCount` | 标签统计 |
| `previewList` | `[]CommentPreview` | 预览评价 |

示例响应：

```json
{
  "goodsId": 10001,
  "totalCount": 218,
  "goodRate": 97,
  "withMediaCount": 18,
  "appendCount": 40,
  "tagList": [
    { "tagName": "触感超舒适", "count": 6 },
    { "tagName": "操作响应迅速", "count": 2 },
    { "tagName": "使用超便捷", "count": 3 }
  ],
  "previewList": [
    {
      "commentId": 9001,
      "score": 5,
      "userName": "阿伟***",
      "userAvatar": "https://cdn.example.com/avatar.png",
      "skuText": "K251粉色【蓝牙键盘】",
      "content": "罗技K251键盘，紧凑便携，续航很长。",
      "mediaList": [
        { "type": "IMAGE", "url": "https://cdn.example.com/c1.jpg" }
      ],
      "createdAt": "2026-04-24 10:00:00",
      "helpfulCount": 12,
      "discussionCount": 3,
      "hasAppend": true
    }
  ]
}
```

#### 8.1.2 获取商品评价分页

`GET /api/app/goods/comment/page`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `goodsId` | `int64` | 是 | 商品 ID |
| `filterType` | `string` | 否 | `ALL / WITH_MEDIA / WITH_APPEND / CURRENT_SKU` |
| `tagName` | `string` | 否 | 标签名称 |
| `skuCode` | `string` | 否 | SKU 编号 |
| `sortType` | `string` | 否 | `DEFAULT / LATEST / HOT` |
| `pageNum` | `int64` | 否 | 页码 |
| `pageSize` | `int64` | 否 | 每页数量 |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `list` | `[]GoodsComment` | 评价列表 |
| `total` | `int32` | 总数 |

#### 8.1.3 获取评价讨论分页

`GET /api/app/goods/comment/reply/page`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `commentId` | `int64` | 是 | 评价 ID |
| `pageNum` | `int64` | 否 | 页码 |
| `pageSize` | `int64` | 否 | 每页数量 |

示例响应：

```json
{
  "total": 5,
  "list": [
    {
      "replyId": 3001,
      "rootReplyId": 3001,
      "parentReplyId": 0,
      "userName": "颜***y",
      "content": "我上一台14年买的还没坏，等苹果明年换模具了再买。",
      "createdAt": "2026-04-24 12:00:00",
      "likeCount": 1,
      "childReplyCount": 2,
      "childReplyPreview": [
        {
          "replyId": 3002,
          "userName": "小璐胖胖",
          "replyToUserName": "颜先森_empty",
          "content": "这么夸张！14年不卡吗"
        }
      ],
      "hasMoreChildren": true
    }
  ]
}
```

#### 8.1.4 获取某条讨论的子回复

`GET /api/app/goods/comment/reply/children`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `rootReplyId` | `int64` | 是 | 一级回复 ID |
| `pageNum` | `int64` | 否 | 页码 |
| `pageSize` | `int64` | 否 | 每页数量 |

#### 8.1.5 获取订单评价表单

`GET /api/app/order/comment/form/{orderId}`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `orderId` | `int64` | 是 | 订单 ID |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `orderId` | `int64` | 订单 ID |
| `goodsList` | `[]CommentableGoods` | 待评价商品列表 |

#### 8.1.6 提交主评价

`POST /api/app/order/comment`

请求示例：

```json
{
  "orderId": 50001,
  "items": [
    {
      "orderGoodsId": 70001,
      "goodsId": 10001,
      "skuCode": "K251-PINK",
      "score": 5,
      "content": "颜值高，手感好，蓝牙连接很稳。",
      "tagNames": ["触感超舒适", "使用超便捷"],
      "isAnonymous": true,
      "mediaList": [
        { "type": "IMAGE", "url": "https://cdn.example.com/a.jpg" },
        {
          "type": "VIDEO",
          "url": "https://cdn.example.com/a.mp4",
          "coverUrl": "https://cdn.example.com/a-cover.jpg",
          "duration": 12
        }
      ]
    }
  ]
}
```

响应示例：

```json
{
  "successIds": [9001],
  "failList": []
}
```

#### 8.1.7 提交追评

`POST /api/app/goods/comment/append`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `rootCommentId` | `int64` | 是 | 主评 ID |
| `content` | `string` | 是 | 追评内容 |
| `mediaList` | `[]Media` | 否 | 追评媒体 |

响应示例：

```json
{
  "commentId": 9101
}
```

#### 8.1.8 发表讨论或回复

`POST /api/app/goods/comment/reply`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `commentId` | `int64` | 是 | 评价 ID |
| `rootReplyId` | `int64` | 否 | 一级回复 ID |
| `parentReplyId` | `int64` | 否 | 父回复 ID |
| `replyToUserId` | `int64` | 否 | 被回复用户 ID |
| `content` | `string` | 是 | 回复内容 |

响应示例：

```json
{
  "replyId": 3003
}
```

#### 8.1.9 评价互动

`POST /api/app/goods/comment/action`

请求参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `targetType` | `string` | 是 | `COMMENT / REPLY` |
| `targetId` | `int64` | 是 | 目标 ID |
| `actionType` | `string` | 是 | `HELPFUL / DISLIKE / LIKE` |
| `cancel` | `bool` | 否 | 是否取消 |

响应示例：

```json
{
  "targetId": 9001,
  "helpfulCount": 18,
  "dislikeCount": 0,
  "likeCount": 0,
  "currentAction": "HELPFUL"
}
```

### 8.2 Admin 侧接口

#### 8.2.1 评价分页管理

`GET /api/admin/goods/comment`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `goodsId` | `int64` | 商品 ID |
| `orderId` | `int64` | 订单 ID |
| `userKeyword` | `string` | 用户关键词 |
| `status` | `int32` | 审核状态 |
| `score` | `int32` | 评分 |
| `hasMedia` | `bool` | 是否带媒体 |
| `isFeatured` | `bool` | 是否精选 |
| `pageNum` | `int64` | 页码 |
| `pageSize` | `int64` | 每页数量 |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `list` | `[]AdminGoodsComment` | 评价列表 |
| `total` | `int32` | 总数 |

#### 8.2.2 获取评价详情

`GET /api/admin/goods/comment/{id}`

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `comment` | `AdminGoodsCommentDetail` | 评价详情 |
| `replies` | `[]AdminGoodsCommentReply` | 回复列表 |
| `actionSummary` | `CommentActionSummary` | 互动汇总 |

#### 8.2.3 审核或更新评价状态

`PUT /api/admin/goods/comment/{id}/status`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `status` | `int32` | `2 通过 / 3 驳回 / 4 隐藏` |
| `isFeatured` | `bool` | 是否精选 |
| `featuredSort` | `int32` | 精选排序 |
| `rejectReason` | `string` | 驳回原因 |

#### 8.2.4 讨论回复管理

`GET /api/admin/goods/comment/reply`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `commentId` | `int64` | 评价 ID |
| `status` | `int32` | 状态 |
| `pageNum` | `int64` | 页码 |
| `pageSize` | `int64` | 每页数量 |

#### 8.2.5 更新回复状态

`PUT /api/admin/goods/comment/reply/{id}/status`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `status` | `int32` | `1 正常 / 2 隐藏 / 3 删除` |

#### 8.2.6 评价统计

`GET /api/admin/goods/comment/stat`

请求参数：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `goodsId` | `int64` | 商品 ID |
| `dateRange` | `[]string` | 时间范围 |
| `pageNum` | `int64` | 页码 |
| `pageSize` | `int64` | 每页数量 |

响应字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `list` | `[]GoodsCommentStatItem` | 统计列表 |
| `total` | `int32` | 总数 |
| `summary` | `GoodsCommentStatSummary` | 汇总信息 |

## 9. 前后端落地影响范围

### 9.1 Backend

- `backend/api/protos/app`：新增 `goods_comment.proto`
- `backend/api/protos/admin`：新增 `goods_comment.proto`
- `backend/service/app`：新增评价服务与业务实现
- `backend/service/admin`：新增评价后台服务与业务实现
- `backend/pkg/gen`：新增评价相关模型和查询代码
- `sql/default-data.sql`：补后台菜单和接口权限初始化

### 9.2 App

- `frontend/app/src/pages/goods/goods.vue`：新增评价分段和评价摘要区
- `frontend/app/src/pagesOrder`：新增评价发布页
- `frontend/app/src/api/app`：新增评价 service
- `frontend/app/src/rpc/app`：新增评价 RPC 类型

### 9.3 Admin

- `frontend/admin/src/views`：新增评价管理与统计页面
- `frontend/admin/src/api/admin`：新增评价后台 service
- `frontend/admin/src/rpc/admin`：新增评价后台 RPC 类型

## 10. 推荐实施顺序

### 10.1 一期

- 补表结构。
- 完成订单评价提交。
- 完成商品详情评价摘要。
- 完成评价列表页。
- 完成后台审核基础能力。

### 10.2 二期

- 完成追评。
- 完成讨论弹层和楼中楼回复。
- 完成精选评价和排序。
- 完成评价统计看板。

### 10.3 三期

- 接入标签模板管理。
- 接入评价质量分析。
- 接入评价信号到推荐或搜索排序。

## 11. 风险与注意点

- `order_info.status` 不建议扩展为“已评价”，否则会和订单生命周期语义混淆。
- 评价资格必须按 `order_goods` 校验，否则多商品订单会出现重复评价或漏评价。
- 图片、视频上传建议复用现有上传能力，不单独重复设计媒体存储。
- 评价聚合统计建议异步更新或事务后更新，避免详情页实时聚合压垮查询。
- 讨论回复如果无限层展开，前端和分页复杂度会快速上升，因此默认只做两层展示。
- 审核状态切换后需要同步刷新 `goods_comment_stat`，避免前台统计和后台审核结果不一致。

## 12. 结论

商品评价功能建议作为独立业务域建设，核心思路是：

- 以 `order_goods` 为评价资格基础。
- 以 `goods_comment` 为主评/追评核心表。
- 以 `goods_comment_reply` 承接讨论楼中楼。
- 以 `goods_comment_stat` 提供商品详情高频读取能力。
- 前台强调“评价摘要 + 列表 + 讨论”，后台强调“审核 + 精选 + 统计”。

该方案与当前仓库结构兼容度较高，后续可以按“表结构 -> proto -> 后端实现 -> App 页面 -> Admin 页面”的顺序逐步落地。
