# 模拟数据生成文档

## 1. 需求概述

生成测试用模拟数据，用于数据分析功能。数据需要覆盖2026-01-01至2026-04-02（今天），共92天。

## 2. 数据范围

### 2.1 用户数据 (base_user)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 用户ID，自增 |
| user_name | varchar(50) | 用户账号，唯一 |
| nick_name | varchar(30) | 用户昵称 |
| openid | varchar(30) | 微信openid |
| role_id | bigint | 角色ID |
| dept_id | bigint | 部门ID |
| phone | varchar(20) | 手机号码 |
| password | varchar(100) | 密码 |
| gender | tinyint | 用户性别，枚举【BaseUserGender】 |
| avatar | varchar(255) | 头像地址 |
| status | tinyint | 状态，枚举【Status】 |
| remark | varchar(500) | 备注 |
| created_by | bigint | 创建者ID |
| updated_by | bigint | 更新者ID |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |
| deleted_at | datetime | 删除时间 |

- **数量**：5000个用户
- **时间范围**：2026-01-01 到 2026-04-02
- **生成规则**：
  - 每天新增约54个用户（5000 ÷ 92 ≈ 54）
  - 用户名格式：`user_序号`（如：user_0001, user_0002）
  - 昵称格式：`用户_序号`（如：用户_0001）
  - 性别随机分配（SECRET: 40%, BOY: 30%, GIRL: 30%）
  - 状态默认 ENABLE
  - role_id 和 dept_id 随机分配

### 2.2 订单数据 (order)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单ID，自增 |
| order_no | varchar(20) | 订单编号，唯一 |
| user_id | bigint | 用户ID |
| pay_money | bigint | 实际支付金额（分） |
| total_money | bigint | 总价（分） |
| post_fee | bigint | 优惠金额（分） |
| goods_num | bigint | 商品总数 |
| pay_type | tinyint | 支付方式，枚举【OrderPayType】 |
| pay_channel | tinyint | 支付渠道，枚举【OrderPayChannel】 |
| delivery_time | tinyint | 配送时间，枚举【OrderDeliveryTime】 |
| status | tinyint | 状态，枚举【OrderStatus】 |
| remark | varchar(255) | 订单备注 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |
| deleted_at | datetime | 删除时间 |

- **每日订单量**：约100单
- **日期范围**：2026-01-01 到 2026-04-02
- **订单状态分布**：

| 状态码 | 状态名 | 说明 | 占比 |
|--------|--------|------|------|
| 1 | CREATED | 待付款 | 10% |
| 2 | PAID | 待发货 | 15% |
| 3 | SHIPPED | 待收货 | 20% |
| 4 | RECEIVED | 已完成 | 35% |
| 97 | REFUNDING | 已退款 | 10% |
| 98 | CANCELED | 已取消 | 8% |
| 99 | DELETED | 已删除 | 2% |

- **生成规则**：
  - 订单编号格式：`O` + 年月日 + 6位序号（如：O20260101000001）
  - 用户从已存在的用户中随机选取
  - 订单金额随机生成（100-10000分，即1-100元）
  - 优惠金额为0或总价 * 5%~15%
  - 商品数量1-5个

### 2.3 订单商品数据 (order_goods)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单商品ID，自增 |
| order_id | bigint | 订单ID |
| goods_id | bigint | 商品ID |
| sku_code | varchar(50) | 规格编号 |
| spec_item | text | SKU规格组成 |
| picture | varchar(255) | 商品图片 |
| name | varchar(255) | 商品名称 |
| num | bigint | 数量 |
| price | bigint | 当前价格（分） |
| pay_price | bigint | 支付价格（分） |
| total_price | bigint | 当前金额汇总 |
| total_pay_price | bigint | 支付金额汇总 |
| deleted_at | datetime | 删除时间 |

- **生成规则**：
  - 每个订单1-3个商品
  - 从商品表随机选取商品

### 2.4 订单支付数据 (order_payment)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单支付ID，自增 |
| order_id | bigint | 订单ID，唯一 |
| order_no | varchar(32) | 订单编号 |
| third_order_no | varchar(32) | 三方订单编号 |
| trade_type | varchar(16) | 交易类型 |
| trade_state | varchar(32) | 交易状态 |
| trade_state_desc | varchar(256) | 交易状态描述 |
| bank_type | varchar(32) | 银行类型 |
| success_time | datetime | 支付完成时间 |
| payer | text | 支付者信息 |
| amount | text | 订单金额 |
| scene_info | text | 场景信息 |
| status | tinyint | 对账状态，枚举【OrderBillStatus】 |
| deleted_at | datetime | 删除时间 |

- **生成规则**：
  - 仅已支付状态（PAID, SHIPPED, RECEIVED, REFUNDING）的订单需要生成
  - 交易状态根据订单状态设置

### 2.5 订单地址数据 (order_address)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单地址ID，自增 |
| order_id | bigint | 订单ID |
| name | varchar(50) | 收货人姓名 |
| phone | varchar(20) | 收货人电话 |
| province | varchar(50) | 省份 |
| city | varchar(50) | 城市 |
| district | varchar(50) | 区/县 |
| address | varchar(255) | 详细地址 |
| deleted_at | datetime | 删除时间 |

- **生成规则**：
  - 每个订单生成一条收货地址
  - 地址信息随机生成

### 2.6 订单退款数据 (order_refund)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单退款ID，自增 |
| order_id | bigint | 订单ID，唯一 |
| order_no | varchar(32) | 支付订单编号 |
| third_order_no | varchar(32) | 三方支付订单编号 |
| refund_no | varchar(32) | 退款编号 |
| reason | tinyint | 退款原因，枚举【OrderRefundReason】 |
| third_refund_no | varchar(32) | 三方退款编号 |
| channel | varchar(32) | 退款渠道 |
| user_received_account | varchar(64) | 退款入账账户 |
| create_time | datetime | 退款创建时间 |
| success_time | datetime | 退款成功时间 |
| refund_state | varchar(32) | 退款状态 |
| funds_account | varchar(32) | 资金账户 |
| amount | text | 支付者信息 |
| status | tinyint | 对账状态，枚举【OrderBillStatus】 |
| deleted_at | datetime | 删除时间 |

- **生成规则**：
  - 仅已退款状态（REFUNDING）的订单需要生成
  - 退款原因随机分配

### 2.7 订单物流数据 (order_logistics)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单物流ID，自增 |
| order_id | bigint | 订单ID，唯一 |
| name | varchar(100) | 物流公司名 |
| no | varchar(100) | 单号 |
| contact | varchar(100) | 联系方式 |
| detail | text | 物流详情 |
| created_at | datetime | 创建时间 |
| deleted_at | datetime | 删除时间 |

- **生成规则**：
  - 仅已发货状态（SHIPPED, RECEIVED）的订单需要生成
  - 物流公司随机选择（顺丰、圆通、中通、韵达、申通等）

### 2.8 订单取消数据 (order_cancel)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint | 订单取消ID，自增 |
| order_id | bigint | 订单ID，唯一 |
| reason | tinyint | 取消原因，枚举【OrderCancelReason】 |
| created_at | datetime | 创建时间 |
| deleted_at | datetime | 删除时间 |

- **生成规则**：
  - 仅已取消状态（CANCELED）的订单需要生成
  - 取消原因随机分配

## 4. 注意事项

- 用户必须先于订单生成
- 订单状态需要与关联表数据保持一致
- 已删除订单（status=99）需要设置 deleted_at 字段
- 金额字段单位为分