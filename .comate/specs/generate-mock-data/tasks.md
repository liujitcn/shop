# 模拟数据生成任务计划

## 任务概述

生成测试用模拟数据脚本，覆盖2026-01-01至2026-04-02，共92天。

- [x] 任务1：创建模拟数据生成工具（独立项目，不影响主项目）
- [x] 任务2：生成用户数据 SQL
- [x] 任务3：生成订单数据 SQL
- [x] 任务4：生成订单商品数据 SQL
- [x] 任务5：生成订单支付数据 SQL
- [x] 任务6：生成订单地址数据 SQL
- [x] 任务7：生成订单退款数据 SQL
- [x] 任务8：生成订单物流数据 SQL
- [x] 任务9：生成订单取消数据 SQL
- [x] 任务10：整合输出完整 SQL 文件

---

## 任务1：创建模拟数据生成工具

### 1.1 创建独立项目结构
- 项目路径：`.comate/specs/generate-mock-data/`
- 创建 `go.mod` 文件，独立于主项目
- 创建 `main.go` 主程序

### 1.2 实现用户数据生成函数
- 生成5000个用户
- 每天约54个用户，从2026-01-01到2026-04-02
- 随机分配性别、角色、部门

### 1.3 实现订单数据生成函数
- 每日约100单
- 按照状态占比分配订单状态

### 1.4 实现关联表数据生成函数
- order_goods、order_payment、order_address
- order_refund、order_logistics、order_cancel

---

## 任务2-10：生成各表数据

按照数据依赖顺序依次生成：
1. 用户数据 (base_user)
2. 订单数据 (order)
3. 订单商品 (order_goods)
4. 订单支付 (order_payment)
5. 订单地址 (order_address)
6. 订单退款 (order_refund)
7. 订单物流 (order_logistics)
8. 订单取消 (order_cancel)

最终输出完整 SQL 文件：`generate_mock_data.sql`（在当前目录下）