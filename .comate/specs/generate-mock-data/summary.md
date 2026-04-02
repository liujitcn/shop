# 模拟数据生成任务总结

## 任务完成情况

所有任务已完成，共生成 1 个 SQL 文件。

## 生成的数据统计

| 数据类型 | 数量 |
|----------|------|
| 用户 (base_user) | 5,000 |
| 订单 (order) | 9,100 |
| 订单商品 (order_goods) | 18,175 |
| 订单支付 (order_payment) | 7,357 |
| 订单地址 (order_address) | 9,100 |
| 订单退款 (order_refund) | 926 |
| 订单物流 (order_logistics) | 5,015 |
| 订单取消 (order_cancel) | 701 |

## 订单状态分布

| 状态 | 数量 | 占比 |
|------|------|------|
| CREATED (待付款) | 882 | 9.7% |
| PAID (待发货) | 1,416 | 15.6% |
| SHIPPED (待收货) | 1,835 | 20.2% |
| RECEIVED (已完成) | 3,180 | 34.9% |
| REFUNDING (已退款) | 926 | 10.2% |
| CANCELED (已取消) | 701 | 7.7% |
| DELETED (已删除) | 160 | 1.8% |

## 输出文件

- **SQL 文件**：`generate_mock_data.sql`（约 16MB）
- **项目路径**：`.comate/specs/generate-mock-data/`
- **独立项目**：包含独立的 `go.mod`，不依赖主项目

## 使用方式

将生成的 SQL 文件导入数据库即可：

```bash
mysql -u username -p database < generate_mock_data.sql
```