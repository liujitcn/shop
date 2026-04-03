# 工作台改版设计记录

## 1. 目标

当前工作台页面 [index.vue](/Users/liujun/workspace/shop/shop/frontend/admin/src/views/dashboard/workspace/index.vue) 主要是静态欢迎语、固定建议和快捷入口，存在几个问题：

- 没有真正利用当前商城后台已有数据表。
- 页面内容偏“说明文字”，缺少运营判断和待处理事项。
- 与订单、商品、运营、财务之间没有形成真实联动。
- 用户进入工作台后，不能快速回答“现在最该处理什么”。

本次改版目标不是再做一个“经营分析页”，而是做一个 **运营工作台**：

- 第一优先级：告诉管理员当前最需要处理的事项。
- 第二优先级：提供关键业务状态的即时总览。
- 第三优先级：提供常用操作入口和处理路径。

## 2. 现有数据基础

基于当前 `backend/pkg/gen/models`，工作台可直接依赖的业务域已经足够完整：

### 2.1 订单域

- [order.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order.gen.go)
- [order_goods.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order_goods.gen.go)
- [order_logistics.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order_logistics.gen.go)
- [order_refund.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order_refund.gen.go)
- [order_cancel.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order_cancel.gen.go)
- [order_payment.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order_payment.gen.go)

可支撑信息：

- 待发货、待收货、已完成、已退款、已取消订单规模
- 今日订单数、今日成交额、客单价
- 退款申请、退款成功、取消单趋势
- 物流发货节奏
- 异常订单巡检入口

### 2.2 商品域

- [goods.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/goods.gen.go)
- [goods_category.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/goods_category.gen.go)
- [goods_sku.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/goods_sku.gen.go)
- [goods_prop.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/goods_prop.gen.go)
- [goods_spec.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/goods_spec.gen.go)

可支撑信息：

- 商品总量、上架商品、下架商品、新增商品
- 动销商品、销量、分类活跃度
- 低活跃分类、待上架商品巡检
- SKU/属性配置完整度检查入口

### 2.3 用户与门店域

- [base_user.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/base_user.gen.go)
- [user_address.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/user_address.gen.go)
- [user_cart.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/user_cart.gen.go)
- [user_collect.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/user_collect.gen.go)
- [user_store.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/user_store.gen.go)

可支撑信息：

- 新增用户、活跃行为用户、下单用户
- 地址、收藏、加购、门店申请等行为覆盖
- 待审核门店申请

### 2.4 运营域

- [shop_banner.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/shop_banner.gen.go)
- [shop_hot.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/shop_hot.gen.go)
- [shop_hot_item.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/shop_hot_item.gen.go)
- [shop_hot_goods.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/shop_hot_goods.gen.go)
- [shop_service.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/shop_service.gen.go)

可支撑信息：

- 首页轮播图启用数量
- 热门推荐位配置情况
- 商城服务项维护状态
- 今日应检查的运营位

### 2.5 财务与对账域

- [pay_bill.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/pay_bill.gen.go)
- [order_refund.gen.go](/Users/liujun/workspace/shop/shop/backend/pkg/gen/models/order_refund.gen.go)

可支撑信息：

- 最近对账状态
- 有误差账单数量
- 退款状态与处理节奏

## 3. 设计原则

### 3.1 工作台不是分析页

工作台不承载完整趋势分析，也不替代数据分析模块。  
工作台应该回答的是：

- 今天先看什么
- 哪些数据异常
- 哪些模块需要立刻进入处理

### 3.2 模块必须能落到真实页面

工作台上的每个卡片和待办都要能跳转到现有页面：

- `/order/info`
- `/goods/info`
- `/user/store`
- `/shop/banner`
- `/shop/hot`
- `/base/config`

不能再出现只描述、不行动的模块。

### 3.3 指标尽量用“状态量 + 处理量”

不追求堆很多 KPI，而是优先显示：

- 当前积压量
- 当日新增量
- 是否异常
- 去哪处理

### 3.4 和当前后台风格一致

工作台应延续当前后台页面语言：

- 以卡片式布局为主
- 左主右辅
- 每张卡片只讲一件事
- 避免大段说明文案

## 4. 新工作台信息架构

建议改成 5 个区域。

### 4.1 顶部：值班概览

用途：进入页面后 3 秒内建立全局认知。

建议内容：

- 今日订单
- 今日成交额
- 待发货订单
- 待审核门店

每张卡展示：

- 主值
- 辅助说明
- 风险提示色

数据来源：

- `order`
- `user_store`

### 4.2 左主区：待处理事项

这是工作台核心区域，优先级最高。

建议拆成 4 张任务卡：

1. 订单履约
   - 待发货订单数
   - 待收货订单数
   - 今日取消订单数
   - 跳转 `/order/info`

2. 退款处理
   - 退款中数量
   - 今日退款成功数
   - 最近异常退款/异常对账提示
   - 跳转订单退款处理页或订单详情页

3. 商品巡检
   - 新增商品
   - 未上架商品
   - 动销商品数
   - 跳转 `/goods/info`

4. 门店审核
   - 待审核门店申请数
   - 今日新增门店申请
   - 跳转 `/user/store`

这一区应采用“动作卡片”形式，而不是纯文本列表。

### 4.3 右上区：运营位巡检

用途：告诉运营当前首页配置是否完整。

建议内容：

- 首页轮播图启用数
- 热门推荐启用数
- 服务承诺启用数
- 是否存在空配置 / 全部禁用

数据来源：

- `shop_banner`
- `shop_hot`
- `shop_service`

对应跳转：

- `/shop/banner`
- `/shop/hot`
- `/shop/service`

### 4.4 右中区：用户与交易快照

用途：让管理员快速感知当前用户状态，而不进入分析页。

建议只保留 3 个小指标：

- 今日新增用户
- 今日下单用户
- 今日活跃行为用户

这里不做图表，只做轻量快照。

数据来源：

- `base_user`
- `order`
- `user_address`
- `user_collect`
- `user_cart`
- `user_store`

### 4.5 右下区：快捷入口

保留当前页面已有价值的“快捷入口”，但收紧到高频业务：

- 订单管理
- 商品管理
- 门店审核
- 轮播图
- 热门推荐
- 系统配置

不建议保留太多后台系统类入口，避免工作台失焦。

## 5. 页面草图

推荐布局：

```text
+--------------------------------------------------------------+
| 顶部概览：今日订单 / 今日成交额 / 待发货 / 待审核门店         |
+--------------------------------------+-----------------------+
| 待处理事项                           | 运营位巡检            |
| - 订单履约                           | - 轮播图启用数        |
| - 退款处理                           | - 热门推荐启用数      |
| - 商品巡检                           | - 服务项启用数        |
| - 门店审核                           |                       |
+--------------------------------------+-----------------------+
| 左下可扩展区                         | 用户与交易快照        |
| - 近期异常提示 / 处理建议            | - 新增用户            |
|                                      | - 下单用户            |
|                                      | - 活跃行为用户        |
+--------------------------------------+-----------------------+
| 快捷入口                                                    |
+--------------------------------------------------------------+
```

布局比例建议继续沿用当前工作台：

- 主内容区：`1.25fr`
- 侧栏区：`0.9fr`

## 6. 每个模块建议展示的数据

### 6.1 顶部概览

| 卡片 | 主值 | 辅助值 | 颜色 |
|---|---|---|---|
| 今日订单 | 今日订单数 | 较昨日 | 蓝 |
| 今日成交额 | 今日成交额 | 客单价 | 红 |
| 待发货 | 当前待发货订单数 | 今日新增待发货 | 橙 |
| 待审核门店 | 当前待审核数 | 今日新增申请 | 绿 |

### 6.2 待处理事项卡

每张卡统一结构：

- 标题
- 2~3 个关键数字
- 1 条提醒
- 1 个主操作按钮

示例：

订单履约卡：

- 待发货：X
- 待收货：Y
- 今日取消：Z
- 按钮：进入订单管理

### 6.3 运营位巡检

| 模块 | 展示值 | 风险条件 |
|---|---|---|
| 首页轮播图 | 启用数 / 总数 | 启用数为 0 |
| 热门推荐 | 启用专题数 | 全部禁用 |
| 服务承诺 | 启用项数 | 少于预期阈值 |

### 6.4 用户与交易快照

| 指标 | 含义 |
|---|---|
| 今日新增用户 | 当日注册数 |
| 今日下单用户 | 当日去重下单用户 |
| 今日活跃行为用户 | 地址/收藏/加购/下单/门店任一行为去重用户 |

## 7. 后端接口建议

建议不要复用现在的分析接口，工作台应单独定义聚合接口：

- `GetWorkspaceOverview`
- `GetWorkspaceTodo`
- `GetWorkspaceOperationHealth`
- `GetWorkspaceUserSnapshot`
- `GetWorkspaceQuickEntry`

原因：

- 工作台需要的是“聚合态结果”，不是分析页那种时间维度趋势接口。
- 工作台字段更偏“当前状态”和“待处理项”。
- 单独接口更稳定，不会被分析页演化绑死。

## 8. 实施优先级

### 第一阶段

先做真正有业务价值的静态改造：

- 顶部概览
- 待处理事项
- 快捷入口

这是工作台从“好看”变成“可用”的最小闭环。

### 第二阶段

增加运营位巡检与用户快照：

- 轮播图 / 热门推荐 / 服务项状态
- 新增用户 / 下单用户 / 活跃行为用户

### 第三阶段

补充异常与风险提示：

- 对账异常
- 退款异常
- 配置缺失提示

## 9. 不建议做的方向

- 不建议把分析页趋势图直接搬进工作台。
- 不建议做过多图表。
- 不建议继续保留“工作建议 / 今日关注”这种完全静态文案区。
- 不建议把系统管理类入口放在工作台主视觉区域。

## 10. 结论

基于当前表结构，工作台最适合的方向不是“再做一个总览页”，而是做成 **运营值班台**：

- 上面看状态
- 中间看待办
- 右边看运营健康
- 下面给入口

这个方案和当前数据表能天然对上，不需要发明新业务对象，也能和数据分析页形成清晰边界：

- 分析页：看趋势、看结构、看对比
- 工作台：看待办、看风险、去处理

