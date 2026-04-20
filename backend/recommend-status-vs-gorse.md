# 推荐接入现状（gorse 版）

## 当前目标

当前仓库已经从“系统内置推荐”切换为“业务后端保留最小事实链路，推荐结果交给 gorse”。

当前阶段的重点不是把 gorse 全量能力一次性接完，而是先保证：

- 后端不再依赖旧推荐表和旧离线任务
- 商城推荐接口、曝光埋点、行为埋点继续可用
- 项目能正常编译、测试、启动
- 后续接 gorse 时不需要重做推荐上下文和归因表结构

## 当前保留的数据表

推荐域当前只保留以下 6 张表：

1. `recommend_request`
2. `recommend_request_item`
3. `recommend_actor_bind_log`
4. `recommend_feedback_event`
5. `recommend_strategy_release`
6. `recommend_metrics_day`

补充说明：

- `goods_stat_day` 继续保留，但它属于商城商品统计表，不属于 gorse 推荐域事实表。
- 旧的 `recommend_model_version`、`recommend_eval_report`、`recommend_exposure`、`recommend_exposure_item`、`recommend_goods_action`、`recommend_goods_relation`、`recommend_user_preference`、`recommend_user_goods_preference` 已经不再是当前方案的一部分。

## 当前后端职责

### 1. 推荐请求

- 入口：`backend/service/app/biz/recommend_request.go`
- 当前会为首页、商品详情、购物车、我的、下单、支付成功等场景生成 `requestId`
- 当前先从 `recommend_strategy_release` 读取场景策略码
- 当前在 gorse API 尚未完全接入时，先使用商品分页兜底返回，保证接口不报错
- 返回结果同时写入 `recommend_request` 和 `recommend_request_item`

### 2. 曝光与行为反馈

- 曝光入口：`backend/service/app/biz/recommend_exposure.go`
- 行为入口：`backend/service/app/biz/recommend_goods_action.go`
- 当前统一写入 `recommend_feedback_event`
- 已覆盖曝光、点击、浏览、收藏、加购、下单、支付
- 若命中 `requestId + goodsId`，会回填 `request_item_id` 和 `position`

### 3. 匿名主体绑定

- 入口：`backend/service/app/biz/recommend_actor_bind_log.go`
- 登录后会把匿名主体下的推荐请求和反馈事件改绑到登录主体
- 同时写入 `recommend_actor_bind_log`

### 4. 商品统计

- 任务：`backend/pkg/job/task/goods_stat_day.go`
- 当前从 `recommend_feedback_event` 的 `view` 事件读取浏览数据
- 再结合 `user_collect`、`user_cart`、`order_info`、`order_goods` 汇总到 `goods_stat_day`

## 当前 gorse 接入边界

当前仓库对 gorse 的边界约定如下：

- gorse 的部署入口位于仓库根目录 `gorse/docker-compose.yml`
- gorse 配置文件位于 `gorse/config/config.toml`
- 场景到 gorse 策略码的映射由 `recommend_strategy_release` 维护
- 后端现阶段先保留请求留痕和反馈留痕，不再在本地训练、发布、评估推荐模型

## 分步骤接入建议

### 第一步：先保证后端链路稳定

- 保持当前 6 张推荐表结构
- 保持 `RecommendGoods`、曝光、行为、匿名绑定接口可用
- 让项目先稳定启动，避免继续依赖旧推荐表和旧任务

### 第二步：启动 gorse 基础服务

- 使用 `gorse/docker-compose.yml` 启动 master、server、worker
- 根据实际环境调整 `gorse/config/config.toml`
- 确认 gorse 能正常接收用户、物品、反馈数据

### 第三步：把反馈事件同步给 gorse

- 以 `recommend_feedback_event` 为后端统一事实表
- 将曝光、点击、收藏、加购、下单、支付等事件转换为 gorse 需要的反馈类型
- 匿名主体和登录主体统一通过当前 actor 体系映射

### 第四步：把推荐请求切到 gorse

- 在 `recommend_request.go` 中把当前商品分页兜底替换为 gorse 推荐接口调用
- 继续保留 `recommend_request` 和 `recommend_request_item` 的落库逻辑
- 保证前端仍然拿到稳定的 `requestId` 和 `recommendContext`

### 第五步：再补充指标与运维

- 使用 `recommend_metrics_day` 做场景级指标日报
- 后续如需要，再增加 gorse 同步状态、失败重试、对账任务

## 当前结论

当前后端已经完成“先去掉旧内置推荐依赖、保留 gorse 接入最小骨架”的收口工作。

这套状态的特点是：

- 项目可以先不依赖旧推荐表和旧离线任务
- 推荐埋点链路仍然完整
- 后续接 gorse 时，不需要再重做前端推荐上下文和行为归因结构
