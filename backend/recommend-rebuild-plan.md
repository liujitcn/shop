# 推荐系统重构开发计划

## 文档目的

本文档用于指导当前项目推荐系统的渐进式重构与能力补齐，目标是：

- 以当前商城推荐链路为基础，逐步补齐 `recommend-vs-gorse.md` 差距对比表中的能力。
- 坚持“先底层、后上层；先并行、后替换；先可运行、后切主链路”的演进原则。
- 在整个改造过程中保持项目始终可运行、可验证、可回滚。
- 将阶段目标、替换顺序、验证方法、风险点、开发进度统一记录在案，便于中断后继续推进。

## 基本约束

- 不允许上来直接重写 `service/app/biz/recommend*.go` 主链路，必须先补底层能力，再逐步切流。
- 每个阶段都必须保留旧实现可用，推荐链路切换优先通过版本配置、开关或并行执行结果对比完成。
- 每次改动都只替换一层职责，不同时改“在线引擎 + 训练链路 + 后台界面”三层。
- 所有新增推荐能力优先落在 `pkg/recommend`，`service/app` 只保留接口编排和业务桥接。
- 每个阶段结束后都要补充本文件中的“开发进度记录”和“阶段结论”。

## 实现参照与复用原则

- 内部实现参照仓库固定为 `/Users/liujun/workspace/github/gorse`，用于参考训练任务拆分、候选集合处理、缓存写入和评估口径；当前项目对外命名、目录命名和业务文案中不引入该名称。
- 基础 KV / Hash 缓存统一优先复用 `github.com/liujitcn/kratos-kit/cache`，推荐层仅补齐排序集合、版本、缓存发布等推荐专用语义。
- 去重、排除集、候选集合并交等集合操作，优先使用 `github.com/deckarep/golang-set/v2 v2.8.0`，避免在业务层重复手写 `map` 去重逻辑。
- 稠密布尔标记、可预测位图、训练阶段批量成员判定等位图场景，优先使用 `github.com/bits-and-blooms/bitset v1.24.4`，作为后续单机训练和评估阶段的基础位图组件。

## 当前基线

### 已有能力

- 已有推荐请求、曝光、点击、浏览、收藏、加购、下单、支付整条事实链路。
- 已有匿名主体与登录主体的统一推荐入口。
- 已有用户类目偏好、用户商品偏好、商品关系、商品热度、评估报表、模型版本表。
- 已有推荐任务：
  - `RecommendGoodsStatDay`
  - `RecommendUserPreferenceRebuild`
  - `RecommendGoodsRelationRebuild`
  - `RecommendEvalReport`

### 当前主要问题

- 在线推荐入口职责过重，召回、特征、排序、落库耦合在一起。
- 行为消费链路同时承担事实落库和聚合更新，难以继续扩展训练能力。
- 现有“重建”属于统计重算，不等于单机训练平台。
- 已有版本表和评估报表，但还没有形成“按版本驱动在线推荐”的闭环。
- 当前服务形态偏在线现算，缺少预计算结果缓存、相似用户、协同过滤、内容相似、模型排序、自动调参、推荐后台。

## 目标能力清单

本次重构最终要补齐以下能力：

1. 多路召回：场景关系、用户偏好、相似用户、协同过滤、内容相似、热门、最新。
2. 商品相似：行为共现相似、属性相似、向量相似。
3. 相似用户：基于行为与偏好生成相似用户 TopN。
4. 自动学习用户口味：从行为事实训练用户兴趣表示，而不是只做行为加权累计。
5. 模型排序：规则粗排 + 模型精排 + LLM TopN 二次重排。
6. 更灵活的去重、惩罚、替换和多样性策略。
7. 更完整的冷启动：匿名、新用户、新商品。
8. 单机离线训练平台：聚合、训练、写缓存、发布。
9. 自动调参：用离线评估结果自动选择更优参数集。
10. 版本驱动推荐：训练版本、缓存版本、在线策略联动。
11. 离线缓存推荐：结果预生成并落推荐缓存，基础 KV / Hash 优先复用 `kratos-kit/cache`，后续在该模块补齐 `LevelDB` 实现，并保持兼容 `Redis`。
12. 推荐后台：任务、指标、版本、缓存、实验、调参结果可视化。

## 差距到阶段映射

`recommend-vs-gorse.md` 里提到的主要差距，不是并列散点，而是要映射到固定阶段里逐项收敛。

| 差距项 | 对应阶段 | 当前状态 | 阶段输出物 |
| --- | --- | --- | --- |
| 在线推荐主链路职责过重 | 阶段 1、阶段 6 | 阶段 1 已开始抽离领域边界，主链路尚未切换 | 统一领域对象、统一在线引擎入口、旧链路下线计划 |
| 行为事实与聚合耦合 | 阶段 2 | 已开始拆分，当前仍保留原队列入口 | 事实事件对象、投影器、离线重建复用聚合函数 |
| 缺少缓存优先和写缓存发布 | 阶段 3 | 已完成 `hot`、`latest`、`similar_item` 三类结果的写缓存调度与在线读取挂点 | 推荐缓存协议、写缓存任务、缓存版本规范、缓存命中指标 |
| 缺少相似用户和协同过滤 | 阶段 4、阶段 5 | 已接入按版本控制的相似用户与协同过滤召回探针，`GOODS_DETAIL` 可按版本灰度并入协同过滤候选 | 相似用户召回、CF 召回、训练产物、候选缓存 |
| 商品相似仍偏行为统计 | 阶段 4、阶段 5 | 已接入内容相似探针读取挂点，`GOODS_DETAIL` 可按版本灰度并入内容相似候选 | 行为相似、属性相似、内容相似三类产物 |
| 缺少模型排序 | 阶段 5、阶段 7 | 尚未开始 | CTR / CVR 轻量模型、排序特征、精排服务 |
| 缺少自动调参与版本联动 | 阶段 8 | 尚未开始 | 参数搜索任务、评估结果回写、版本发布和回滚 |
| 缺少推荐后台和监控 | 阶段 9 | 尚未开始 | 推荐任务后台、指标面板、发布记录、排障入口 |

## 当前阶段状态

为了便于中断后继续推进，阶段状态统一按下表维护：

| 阶段 | 状态 | 说明 |
| --- | --- | --- |
| 阶段 0 | 已完成 | 已完成建档、README 入口和基线固化 |
| 阶段 1 | 已完成 | 领域对象与缓存边界已落到 `pkg/recommend` |
| 阶段 2 | 进行中 | 商品行为投影器已下沉，其他聚合逻辑仍待继续收敛 |
| 阶段 3 | 进行中 | 已补缓存 key 规范、缓存后端适配、首批写缓存任务调度，并接入 `hot`、`latest`、`similar_item` 在线缓存优先读取 |
| 阶段 4 | 进行中 | 已补相似用户、协同过滤、内容相似探针和版本控制入口，并在 `GOODS_DETAIL` 接入首批灰度候选融合 |
| 阶段 5 | 进行中 | 已补首批相似用户、协同过滤、内容相似训练与写缓存任务，排序模型仍待后续补齐 |
| 阶段 6 | 进行中 | 已开始将召回探针解析与请求计划对象下沉到 `pkg/recommend/online`，主链路尚未切换 |
| 阶段 7 | 未开始 | 依赖阶段 5 的排序特征与模型产物 |
| 阶段 8 | 未开始 | 依赖评估指标、训练版本、发布协议打通 |
| 阶段 9 | 未开始 | 依赖前面阶段输出稳定指标和操作面 |

## 当前代码锚点

为了避免后续推进时反复重新定位，先把当前推荐链路的主要落点固定下来：

| 职责 | 当前文件 | 当前作用 | 后续阶段处理方式 |
| --- | --- | --- | --- |
| 在线推荐总入口 | `service/app/biz/recommend_request.go` | 同时承担场景规划、召回、候选合并、排序、明细落库 | 阶段 6 拆成 planner / recall / rank / record，多数逻辑迁到 `pkg/recommend/online` |
| 在线缓存读取桥接 | `service/app/biz/recommend_cache.go` | 负责 `hot`、`latest`、`similar_item` 的缓存读取、版本回退和排除过滤 | 阶段 6 再迁到 `pkg/recommend/online/cache` |
| 召回探针桥接 | `service/app/biz/recommend_recall_probe.go` | 负责按版本配置读取相似用户、协同过滤、内容相似三类召回探针，并把命中信息写入请求上下文 | 阶段 6 再迁到 `pkg/recommend/online/recall` |
| 商品行为事实入口 | `service/app/biz/recommend_goods_action.go` | 消费队列、写 `recommend_goods_action`、调用投影器 | 阶段 2 保持这里只做事实入库和桥接，不继续扩展聚合细节 |
| 实时投影器 | `pkg/recommend/offline/aggregate/goods_action_projector.go` | 把行为事实投影到用户偏好、商品偏好、商品关系 | 阶段 2 继续扩展为离线重建和实时更新共用聚合入口 |
| 用户类目偏好重建 | `service/app/biz/recommend_user_preference.go` | 仍在 `biz` 层直接读事实表并重建聚合 | 阶段 2 继续下沉到 `pkg/recommend/offline/aggregate` |
| 商品关联重建 | `service/app/biz/recommend_goods_relation.go` | 仍在 `biz` 层直接读事实表并重建商品关系 | 阶段 2 继续下沉到 `pkg/recommend/offline/aggregate` |
| 候选构建 | `pkg/recommend/candidate/logic.go` | 已下沉匿名/登录态候选构建和基础打散排序 | 阶段 4 继续从“单函数候选构建”演进到多召回组合 |
| 基础排序函数 | `pkg/recommend/rank/weight_ranker.go` | 负责当前规则排序中的新鲜度、曝光惩罚等基础分 | 阶段 7 在此基础上补模型精排和 LLM 重排挂点 |
| 推荐缓存协议 | `pkg/recommend/cache/types.go`、`pkg/recommend/cache/key.go`、`pkg/recommend/cache/store.go` | 已定义推荐缓存语义、固定 key 前缀并接入基础缓存实现 | 阶段 3 在不改协议前提下继续补写缓存任务与读缓存桥接 |
| 推荐领域对象 | `pkg/recommend/domain/*.go` | 已承载请求、上下文、候选、特征、结果、版本、行为事件 | 阶段 4 之后继续作为统一在线/离线边界 |
| 依赖注入入口 | `service/app/init.go` | 汇总 `biz` 与 `pkg/recommend` 推荐依赖 | 新增在线引擎、缓存服务、写缓存任务时在这里接线 |

## 目标目录结构

推荐相关代码最终收敛到以下结构：

```text
backend/pkg/recommend/
  actor/
  event/
  domain/
    request.go
    context.go
    candidate.go
    feature.go
    strategy.go
    version.go
  online/
    engine/
    planner/
    recall/
      relation/
      user_cf/
      user_neighbor/
      content/
      hot/
      latest/
    feature/
    rank/
      rule/
      model/
      llm/
    diversify/
    filter/
    record/
    cache/
  offline/
    aggregate/
    train/
      similarity/
      cf/
      ctr/
      tune/
    materialize/
    evaluate/
    publish/
  admin/
    dto/
    service/
```

## 演进原则

### 原则一：先补基础设施，再替换主链路

必须先完成以下底层能力，才能动在线推荐主入口：

- 推荐领域模型
- 推荐专用缓存接口
- 推荐版本配置协议
- 推荐训练产物读写协议
- 推荐聚合器与训练任务基座

### 原则二：先并行运行，再切换默认实现

每次替换主链路前都要经历三个步骤：

1. 新能力只写入、不读取。
2. 新旧能力并行计算，记录结果差异。
3. 差异稳定后再切默认读取路径。

### 原则三：每一阶段都可单独停止

任何一个阶段完成后，即使后续停工，项目也必须满足：

- 服务能正常启动。
- 旧接口不被破坏。
- 数据结构兼容。
- 当前阶段的新增能力可独立验证。

## 总体阶段规划

### 阶段 0：建档与稳定基线

目标：

- 建立完整规划文档、进度记录、阶段结论记录方式。
- 固化当前推荐链路的现状、入口文件、任务、验证口径。

输出物：

- 本文档。
- `README` 入口。
- 当前推荐链路基线清单。

不改动：

- 在线推荐行为。
- 数据库表结构。

### 阶段 1：抽离推荐底层领域层

目标：

- 把推荐领域模型、策略配置、缓存协议、训练产物协议从 `service/app/biz` 中抽离到 `pkg/recommend/domain`。
- 为后续分层替换建立稳定边界。

重点任务：

- 新增推荐领域请求、候选、特征、结果、版本对象。
- 定义 `RecommendCache` 接口，首版只约束能力，不急着切主链路。
- 定义在线引擎统一输入输出协议。
- 定义离线训练产物的统一描述对象。

阶段完成标准：

- `service/app/biz` 不再直接依赖零散的 `map[string]any` 传递核心推荐上下文。
- 后续召回、排序、缓存都能使用统一领域对象。

### 阶段 2：重构行为事实层与聚合层

目标：

- 将当前推荐行为消费从“事实落库 + 在线聚合”拆成“事实层 + 投影层”。
- 为单机训练提供稳定输入。

重点任务：

- 拆分当前推荐行为消费者职责。
- 保留 `recommend_goods_action`、`recommend_request`、`recommend_exposure` 等事实表作为事实源。
- 将用户偏好、商品偏好、商品关系、商品热度明确为投影结果。
- 把离线重建与实时投影统一到同一套聚合逻辑。
- 行为投影里的去重、排除集和候选集合优先复用 `golang-set/v2`，后续训练阶段的位图判定优先复用 `bitset`。

阶段完成标准：

- 新行为写入不再和所有聚合逻辑强绑定在一个大事务里。
- 离线重建和实时更新共用统一聚合函数或聚合服务。

阶段 2 文件级执行清单：

1. 保持 `service/app/biz/recommend_goods_action.go` 只负责“队列消费 -> 事实落库 -> 调用投影器”，不要再继续把偏好或关系聚合写回这个 Case。
2. 继续扩展 `pkg/recommend/offline/aggregate/goods_action_projector.go`，把“实时投影”和“离线重建”需要复用的聚合函数沉到这里或其同级文件。
3. 把 `service/app/biz/recommend_user_preference.go` 中的重建逻辑改成调用 `pkg/recommend/offline/aggregate`，避免 `biz` 层继续维护第二套聚合实现。
4. 把 `service/app/biz/recommend_goods_relation.go` 中的重建逻辑改成调用 `pkg/recommend/offline/aggregate`，统一请求共现和订单共现的累计规则。
5. 在阶段 2 内暂时不要改 `service/app/biz/recommend_request.go` 的在线读取路径，避免事实层拆分和在线链路改造叠在一起。

阶段 2 完成后的代码形态要求：

- `service/app/biz` 中的推荐 Case 只保留接口编排、任务入口、事实桥接。
- 所有推荐聚合规则只在 `pkg/recommend/offline/aggregate` 维护一份。
- 离线重建任务和实时投影调用同一套聚合函数，而不是各自复制一份规则。

### 阶段 3：落地推荐专用缓存与写缓存层

目标：

- 引入推荐专用缓存层。
- 基础 KV / Hash 能力优先复用 `kratos-kit/cache`。
- 后续在 `kratos-kit/cache` 模块中补齐 `LevelDB` 实现，同时保持兼容 `Redis`。
- 支持热门榜、相似商品、相似用户、协同过滤候选、最终推荐结果缓存。

重点任务：

- 定义驱动无关的推荐缓存协议。
- 首先接入 `kratos-kit/cache` 作为基础缓存模块。
- 后续在 `kratos-kit/cache` 中补齐 `LevelDB` 实现，再回接推荐缓存层。
- 预留 `Redis` 缓存实现接入位。
- 定义缓存 key 规范和版本规范。
- 支持写缓存任务将结果写入缓存。
- 在线引擎支持“缓存优先，未命中查库”。

建议缓存内容：

- `recommend:user:{scene}:{actor}:{version}`
- `similar_item:{goods_id}:{version}`
- `similar_user:{user_id}:{version}`
- `cf_candidate:{user_id}:{version}`
- `content_candidate:{goods_id}:{version}`
- `hot:{scene}:{version}`
- `latest:{scene}:{version}`
- `llm_rerank:{scene}:{actor}:{request_hash}:{version}`

阶段完成标准：

- 推荐结果支持离线预热与缓存命中。
- 缓存接口层不依赖具体存储实现，可平滑接入 `LevelDB` 与 `Redis`。
- 当前项目推荐服务形态从“纯在线现算”升级为“缓存优先 + 在线补算”。

阶段 3 启动前置条件：

- 阶段 2 至少完成商品行为投影、用户类目偏好、用户商品偏好、商品关系三类聚合逻辑的下沉或桥接收口。
- 在线推荐主链路暂不改协议，只增加缓存读写挂点。
- 版本对象、缓存 key 规范和写缓存任务输入输出先在 `pkg/recommend` 内固定。

阶段 3 实施拆解：

1. 先定义推荐缓存实体和缓存键构造器，固定用户推荐、相似商品、相似用户、热门榜、最新榜、协同过滤候选的 key 规范。
2. 再补缓存读写适配层，基础 KV / Hash 直接复用 `kratos-kit/cache`，推荐层只补排序集合、多值列表、版本摘要这些语义。
3. 再落写缓存任务，把热门榜、最新榜、相似商品等当前最容易稳定的结果先写入缓存。
4. 然后在在线推荐链路增加“缓存优先，未命中查库”的只读挂点，先记录命中来源和请求上下文，不急着重写在线引擎。
5. 最后补版本发布和失效策略，确保新版本发布只影响对应 key 空间，不污染旧版本缓存。

阶段 3 验证重点：

- 同一版本下重复写缓存不会产生脏写和重复 key。
- 缓存未命中时，旧在线现算结果保持不变。
- 缓存命中后，推荐返回结构、曝光入库、行为回流不受影响。
- 可按场景、按版本查看写缓存数量、发布时间和命中率。

阶段 3 文件级执行清单：

1. 基于 `pkg/recommend/cache/types.go`、`pkg/recommend/cache/key.go` 与 `pkg/recommend/cache/store.go` 固定缓存协议，不在业务 Case 中直接拼接缓存 key。
2. 在 `pkg/recommend` 下新增写缓存与缓存读写承接目录，优先落到 `offline/materialize`、`online/cache` 或同等职责目录，不把缓存实现散落回 `service/app/biz`。
3. 首批写缓存对象只覆盖 `hot`、`latest`、`similar_item` 三类稳定结果，避免一开始就把 `recommend`、`cf_candidate`、`llm_rerank` 全量做完。
4. `service/app/biz/recommend_request.go` 在阶段 3 只允许增加“缓存读取挂点”和命中日志，不允许直接改造成全新在线引擎。
5. 如需新增缓存服务或写缓存任务注入，统一在 `service/app/init.go` 接线，保证依赖入口单一。

阶段 3 暂不处理项：

- 不在本阶段引入协同过滤训练结果。
- 不在本阶段引入相似用户缓存。
- 不在本阶段替换 `RecommendGoods` 主流程。
- 不在本阶段引入后台管理页面。

### 阶段 4：补齐召回层能力

目标：

- 将当前单一路径召回扩展为多路召回框架。

重点任务：

- 保留现有关系召回、偏好召回、热门召回、最新召回。
- 新增相似用户召回。
- 新增协同过滤召回。
- 新增内容相似召回。
- 建立召回融合与解释信息输出能力。

替换策略：

- 先新增新召回器，不替换旧召回。
- 先在日志和请求明细中记录新召回命中信息。
- 观察一段时间后再将其纳入默认候选融合。

阶段完成标准：

- 在线推荐可同时从多条召回链生成候选池。
- 单个召回器的启停可由策略版本控制。

当前进展补充：

- 已为相似用户、协同过滤、内容相似三类召回补齐缓存键约定和读取探针。
- 已支持从 `recommend_model_version.config_json.recall_probe` 读取探针启停和读取数量。
- 当前探针命中信息会统一写入推荐请求 `sourceContext.onlineDebugContext`；其中 `recallProbeContext` 记录探针命中与配置，`joinRecallContext` 区分入池、入候选、入返回页三层命中，相似用户探针的 `similarUserObservationContext` 用于观测和当前候选、返回结果，以及协同过滤和内容相似灰度结果的重合情况。
- 当前阶段仍缺训练产物写入任务，因此探针默认用于观测，不直接改主推荐结果。

### 阶段 5：补齐单机训练平台

目标：

- 构建单机可运行的训练与发布流水线。

训练任务包含：

- 商品相似训练
- 用户相似训练
- 协同过滤训练
- CTR / CVR 轻量排序模型训练
- 调参任务
- 结果写缓存任务

阶段完成标准：

- 不再只有“重建表”和“统计报表”，而是有可用于在线推荐的训练产物。
- 训练产物可落库、可缓存、可挂版本。

当前进展补充：

- 已新增相似用户、协同过滤、内容相似三类离线训练与写缓存任务。
- 当前首版训练优先复用用户类目偏好、用户商品偏好、商品属性等现有聚合结果，不新引入库表。
- 当前产物已可按启用版本发布到推荐缓存，并可直接被阶段 4 的召回探针读取。
- 已为写缓存任务补统一失败摘要，任务异常时会带出当前执行阶段、输入规模、已发布子集合数、已清理子集合数和耗时，便于排查训练或发布卡点。
- 当前仍未补 CTR / CVR 轻量排序模型、模型文件发布和自动调参链路。

### 阶段 6：重构在线引擎并灰度替换主链路

目标：

- 将现有推荐主链路切换到统一在线引擎。

重点任务：

- 新建在线引擎入口。
- 将场景规划、召回、特征、排序、打散、过滤、记录拆开。
- 保留旧 `RecommendGoods` 接口，不改 `proto`。
- 引入版本驱动策略选择。

替换顺序建议：

1. `GOODS_DETAIL`
2. `CART`
3. `ORDER_DETAIL`
4. `ORDER_PAID`
5. 其他公共场景

阶段完成标准：

- 原大函数退出主链路。
- 在线推荐按版本执行，不再靠固定流程硬编码。

### 阶段 7：补齐模型排序与 LLM 优化

目标：

- 在规则排序之上增加模型精排与 LLM TopN 重排。

排序链路建议：

1. 规则粗排
2. CTR / CVR 轻量模型精排
3. LLM 对 TopN 做二次重排

注意事项：

- LLM 只参与 TopN，不参与全量召回。
- LLM 输出必须可缓存、可追踪、可关闭。
- LLM 结果要落 explain 记录，便于排查排序变化原因。

阶段完成标准：

- 当前项目具备“规则排序 + 模型排序 + 大模型重排”三层结构。

### 阶段 8：补齐自动调参与版本发布

目标：

- 把离线评估结果接入自动调参与版本发布闭环。

重点任务：

- 枚举权重、阈值、召回组合。
- 离线计算 Precision、Recall、NDCG、CTR、CVR。
- 自动选出候选最优参数集。
- 将结果写入版本配置。
- 支持发布、灰度、回滚。

阶段完成标准：

- 推荐版本不是只记录，而是能驱动在线链路。
- 评估结果能自动反馈到参数与版本管理。

### 阶段 9：补齐推荐后台与监控

目标：

- 提供推荐专用管理与排障入口。

后台至少覆盖：

- 训练任务状态
- 当前启用版本
- 召回命中率
- CTR / CVR / Precision / Recall / NDCG 趋势
- 缓存命中率
- LLM 重排命中率与耗时
- 调参结果对比
- 版本发布与回滚记录

阶段完成标准：

- 推荐链路具备独立的管理、观测和排障入口。

## 分层替换顺序

为了避免项目失稳，必须按以下固定顺序替换：

1. 文档与边界协议
2. 领域模型
3. 行为事实层
4. 聚合层
5. 缓存层
6. 训练层
7. 召回层
8. 排序层
9. 在线引擎
10. 后台与监控

禁止跳步：

- 禁止在没有缓存层和版本层的情况下直接把在线引擎全量替换。
- 禁止在没有事实层拆分的情况下直接上协同过滤和相似用户。
- 禁止在没有评估闭环的情况下直接默认启用模型排序。

## 主链路替换策略

每个主链路替换阶段都要执行以下动作：

### 第一步：并行写

- 新能力产生的数据与产物先写入，但不参与线上读。

### 第二步：并行算

- 新旧链路同时计算。
- 结果差异写日志、写对比表或写评估记录。

### 第三步：灰度读

- 通过策略版本或显式开关让部分场景读新链路。

### 第四步：默认切换

- 新链路稳定后再切换默认读取。

### 第五步：旧链路下线

- 只有在至少一个阶段周期内稳定运行后，才允许删除旧实现。

## 风险控制

### 风险一：在线推荐改动过大导致接口可用性下降

控制措施：

- 接口契约保持不变。
- 在线主入口最后替换。
- 每个场景单独切换，不做全场景一次切换。

### 风险二：行为链路改动导致推荐事实丢失

控制措施：

- 事实落库优先。
- 聚合失败不影响事实记录保留。
- 新旧聚合同期并行校验。

### 风险三：训练产物不稳定导致推荐结果抖动

控制措施：

- 训练结果必须挂版本。
- 训练结果发布前必须通过离线评估。
- 支持快速回滚上一稳定版本。

### 风险四：LLM 引入延迟和成本波动

控制措施：

- 只做 TopN 重排。
- 必须有本地缓存。
- 必须可按场景、按版本、按开关关闭。

## 验证策略

每个阶段完成后至少执行以下验证：

### 通用验证

```bash
cd backend
go test ./...
```

### 推荐链路专项验证

- 推荐接口返回结构不变。
- 请求、曝光、行为链路仍可正常落库。
- 新增训练任务和缓存任务可单独执行。
- 评估报表可正常生成。

### 重点对比验证

- 新旧推荐结果差异率
- 新旧召回命中率
- 缓存命中率
- 场景级 CTR / CVR / NDCG

## 里程碑定义

### 里程碑 M1：底层协议稳定

- 阶段 0 ~ 2 完成
- 在线推荐主链路未切换
- 训练和缓存边界已具备

### 里程碑 M2：多路召回与缓存可用

- 阶段 3 ~ 5 完成
- 支持多路召回和单机训练
- 支持 `LevelDB` 推荐缓存

### 里程碑 M3：在线引擎切换完成

- 阶段 6 完成
- 主要场景已切到统一在线引擎

### 里程碑 M4：自动学习与优化闭环形成

- 阶段 7 ~ 8 完成
- 模型排序、LLM 重排、自动调参、版本发布全部接通

### 里程碑 M5：后台与监控完善

- 阶段 9 完成
- 推荐系统具备完整平台化运维能力

## 开发进度记录

后续每次推进本计划时，都在本节补一条记录。

| 日期 | 阶段 | 改动摘要 | 是否改动主链路 | 验证结果 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 2026-04-16 | 阶段 0 | 建立推荐系统重构开发计划文档，并在 README 增加入口 | 否 | 未执行代码验证，本次仅文档建档 | 当前项目基线仍保持不变 |
| 2026-04-16 | 阶段 1 | 新增 `pkg/recommend/domain` 与 `pkg/recommend/cache` 基础协议，`core/types` 与 `candidate` 完成兼容转接 | 否 | `cd backend && go test ./...` 通过 | 基础 KV/Hash 明确优先复用 `kratos-kit/cache`，推荐层仅保留排序集合缓存协议 |
| 2026-04-16 | 阶段 2 | 更新实现参照与集合库选型文档，并将商品行为投影器下沉到 `pkg/recommend/offline/aggregate`，`biz` 层仅保留事实入库与桥接 | 否 | `cd backend && go test ./...` 通过 | 当前不改变队列入口和主推荐读取链路，仅调整推荐聚合分层位置 |
| 2026-04-16 | 阶段 2 / 阶段 3 | 补充当前代码锚点与阶段 2、阶段 3 文件级执行清单，明确后续优先改哪些文件、暂不改哪些文件 | 否 | `cd backend && go test ./...` 通过 | 本次仍为文档细化，不涉及在线推荐主流程变更 |
| 2026-04-16 | 阶段 2 | 将 `recommend_user_goods_preference`、`recommend_user_preference`、`recommend_goods_relation` 的离线重建逻辑统一下沉到 `pkg/recommend/offline/aggregate`，`biz` 层仅保留删旧数据、调用聚合器和批量落库 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前未改在线推荐主读路径，仅继续收口聚合重建实现 |
| 2026-04-16 | 阶段 2 | 将 `RecommendGoodsStatDay` 的按天聚合逻辑下沉到 `pkg/recommend/offline/aggregate`，任务文件仅保留日期解析、删旧数据和批量回写 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前推荐统计读路径不变，继续为阶段 3 缓存写入准备稳定聚合输入 |
| 2026-04-16 | 阶段 3 | 新增推荐缓存 key 规范与写缓存服务，首批支持 `hot`、`latest`、`similar_item` 三类结果按统一协议发布到推荐缓存 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仅补齐缓存命名和写缓存基础，尚未接入具体缓存后端和任务调度 |
| 2026-04-16 | 阶段 3 | 接入 `kratos-kit/cache` 推荐缓存适配层，并将 `RecommendHotMaterialize`、`RecommendLatestMaterialize`、`RecommendSimilarItemMaterialize` 注册到调度任务 | 否 | `cd backend/internal/cmd/server && GOCACHE=/tmp/shop-go-build-cache wire`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前已形成“聚合结果 -> 写缓存服务 -> 定时任务”闭环 |
| 2026-04-16 | 阶段 3 | 在 `service/app/biz/recommend_request.go` 与 `service/app/biz/recommend_cache.go` 接入 `hot`、`latest`、`similar_item` 缓存优先读取，未命中回退原查库路径，并把 `cacheHitSources` 写入请求上下文 | 是 | `cd backend/internal/cmd/server && GOCACHE=/tmp/shop-go-build-cache wire`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍未抽离独立在线引擎，但阶段 3 的“写缓存 + 在线读取回退”闭环已经接通 |
| 2026-04-16 | 阶段 5 | 为六类写缓存任务补统一失败摘要日志，并为“无启用版本”的跳过分支补统一摘要输出，任务异常或跳过时都能看到当前阶段、输入规模、发布进度、清理进度与耗时 | 否 | `cd backend/internal/cmd/server && GOCACHE=/tmp/shop-go-build-cache wire`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前先补齐训练发布链路排障信息，排序模型和调参链路仍待后续推进 |
| 2026-04-16 | 阶段 5 / 阶段 6 准备 | 为写缓存元信息补 `document_count`，并在在线读缓存与召回探针中记录版本号、版本发布时间、缓存发布时间、文档数量、扫描数量、返回数量和命中状态，统一收口到 `onlineDebugContext` | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍未切主链路，只补阶段 6 灰度前必需的最小缓存观测字段 |
| 2026-04-16 | 阶段 6 | 新增 `pkg/recommend/online/recall`，将探针结果解析、灰度召回上下文归一化和相似用户观测逻辑从 `service/app/biz` 抽离到在线层，主链路继续复用原入口 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前先做纯逻辑下沉，不改 `proto`、不切默认读链路 |
| 2026-04-16 | 阶段 6 | 新增 `pkg/recommend/online/planner`，将匿名态和登录态的请求计划对象、灰度召回入池状态、缓存命中状态和来源上下文基础收口从 `service/app/biz/recommend_request.go` 下沉到在线层 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有查库、排序和落库流程，只收口阶段 6 需要复用的前置计划状态 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，将 `CART`、`ORDER_*`、`GOODS_DETAIL`、`profile` 和 `latest` fallback 的场景级规划动作从 `service/app/biz/recommend_request.go` 下沉到在线层 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍未拆统一在线引擎入口，但场景前置规划已经开始从大函数中的 `switch` 迁出 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增 `SceneInput` 场景桥接输入和基础来源上下文构建，把登录态与匿名态的场景原始数据映射、缓存命中来源回写和 `orderId/goodsId/cartGoodsIds/sourceGoodsIds` 调试字段继续从 `service/app/biz/recommend_request.go` 收口到在线层 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前 `recommend_request.go` 里的场景 `switch` 进一步收敛为“查原始数据 -> 调 planner”，但默认在线主链路仍未切换 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，将结果来源上下文与在线调试上下文的组装继续从 `service/app/biz/recommend_request.go` 下沉到在线层，统一承接 `BuildSceneResultSourceContext`、匿名态 join recall 上下文和登录态 similar-user observation 上下文的拼装 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前主链路仍保留查库、打分和落库，但来源上下文构建已进一步从业务入口函数里抽离 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增 `ResultSnapshot` 并把匿名态、登录态结果快照字段和最终来源上下文拼装继续从 `service/app/biz/recommend_request.go` 收口到在线层 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前主链路函数里仍保留候选查库与排序，但 `candidateLimit/sceneHotGoodsIds/anonymousCandidateGoodsIds/returnedScoreDetails` 这类结果字段已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增候选池状态方法，把类目补足候选、latest 候选、latest 排除集合、匿名态场景候选合并和登录态最终候选集合合并继续从 `service/app/biz/recommend_request.go` 收口到在线层 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前主链路里仍保留查库与排序信号加载，但候选池集合状态和去重合并规则已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增候选池查询参数计划，把类目补足查询参数、latest 查询参数和匿名态 latest fallback 判断继续从 `service/app/biz/recommend_request.go` 收口到在线层 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前 `recommend_request.go` 里仍保留具体查库桥接，但是否启用查询、使用哪组排除商品和使用哪组候选池上限，已开始通过 planner 统一决策 |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request.go`，新增共享候选池桥接查询方法，统一复用类目补足和 latest 兜底的商品 ID 提取逻辑，减少主链路中的重复查询拼装与结果提取代码 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍未把查库桥接整体迁到在线层，但主链路里重复的候选池桥接查询代码已开始收口到共享方法 |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request.go`，新增匿名态 latest 回退商品列表桥接方法，统一收口 latest fallback 的分页查询与总数返回逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有排序与落库流程，但匿名态 latest 回退分支已进一步收敛为“取计划 -> 调桥接 -> 回写上下文” |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增排序信号快照方法，统一收口匿名态与登录态候选商品列表的过滤、商品 ID 提取和类目 ID 提取逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有信号加载与排序流程，但候选信号加载前的结果整理逻辑已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增分页 explain 快照方法，统一收口当前页召回来源列表、评分明细和返回商品编号提取逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有排序与来源上下文写回流程，但分页 explain 组装逻辑已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增排序结果分页窗口快照方法，统一收口总数计算、分页窗口切片和空页判定逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有来源上下文写回流程，但匿名态与登录态的分页窗口和空页分支已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增结果回写桥接方法，统一收口匿名态 latest fallback、匿名态空页、匿名态正常页、登录态空页和登录态正常页的 `ResultSnapshot` 构建与 `sourceContext` 回写逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有排序与返回结果结构，但末尾结果回写分支已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增排序信号加载计划，统一收口匿名态与登录态的场景编号、候选商品编号、候选类目编号和关系分源商品编号组织逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有信号加载顺序与打分流程，但排序信号加载前的参数组织已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增领域信号桥接方法，统一收口匿名态 `AnonymousSignals` 与登录态 `PersonalizedSignals` 的组装逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有信号加载顺序与排序流程，但信号加载完成后的领域对象桥接已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/planner`，新增 explain 召回补标方法，统一收口匿名态内容相似灰度补标，以及登录态内容相似、协同过滤灰度补标逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有排序和 explain 输出结构，但灰度召回 explain 补标逻辑已开始通过 planner 统一组织 |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request.go`，新增共享分页桥接底层方法，统一收口类目补足、latest 候选和 latest fallback 的 `PageGoodsInfo` 调用、排除过滤和分页参数拼装 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留三类桥接查询的返回值差异，但其底层分页调用已开始复用统一桥接方法 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/record`，新增推荐请求主表 `sourceContext` 持久化整理方法，统一收口 explain 明细裁剪和在线调试上下文压缩逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有请求记录事务和 item 明细落库流程，但主表 `sourceContext` 的持久化整理已开始从业务入口函数里迁出 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/record`，新增推荐请求逐商品明细模型构建方法，统一收口 `returnedScoreDetails` 索引收敛、单商品召回来源回退和 `RecommendRequestItem` 列表组装逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有批量落库和请求回查流程，但逐商品明细模型的纯整理逻辑已开始从业务层迁出 |
| 2026-04-16 | 阶段 6 | 继续扩展 `pkg/recommend/online/record`，新增推荐请求逐商品明细读取整理方法，统一收口关联商品编号提取和商品位次映射构建逻辑 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有 requestId 回查和明细查询流程，但回查后的纯循环整理逻辑已开始从业务层迁出 |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request_item.go`，新增共享读桥接方法，统一收口 `requestId -> requestEntity` 和 `requestEntity.ID -> requestItemList` 的查询路径 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留原有 requestId 回查语义和明细查询语义，但关联商品、位次映射两条读取路径的重复查询拼装已开始收口 |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request.go` 与 `recommend_request_item.go`，统一收口商品编号提取工具和按 `requestId` 读取逐商品明细的共享桥接入口 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留查库桥接在业务层，但 `recommend_request.go` 的纯商品 ID 提取工具和 `recommend_request_item.go` 的重复 requestId 读分支已进一步缩短 |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request.go`，统一收口候选池查询计划的可执行判断和分页桥接结果快照，减少类目补足、latest 候选和 latest fallback 的结果侧重复分支 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留查库桥接在业务层，但候选池查询结果的 `list/id/total` 提取与空计划判断已进一步迁到 planner |
| 2026-04-16 | 阶段 6 | 继续压缩 `service/app/biz/recommend_request.go`，统一收口 `CART`、`ORDER_*`、`GOODS_DETAIL` 场景的 `SceneInput` 构造，减少场景分支中的字段赋值拼装 | 否 | `cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./pkg/recommend/online/...`、`cd backend && GOCACHE=/tmp/shop-go-build-cache go test ./...` 通过 | 当前仍保留查库桥接在业务层，但场景分支中的 `SceneInput` 映射已进一步迁到 planner |

## 阶段结论记录

每完成一个阶段，在下面追加结论。

### 阶段 0 结论

- 已建立统一规划文档。
- 已明确从底层开始、分阶段替换、保持项目可运行的改造原则。
- 下一步从“阶段 1：抽离推荐底层领域层”开始实施。

### 阶段 1 结论

- 已抽离推荐请求、上下文、候选、特征、结果、策略版本等领域对象。
- 已建立推荐缓存键规范和缓存协议边界。
- 已明确基础 KV/Hash 先复用 `kratos-kit/cache`，后续在该模块补齐 `LevelDB` 实现。
- 当前仍未替换在线主链路，满足“先稳边界、后切主链路”的阶段目标。
- 下一步继续推进“阶段 2：行为事实层与聚合层拆分”。

### 阶段 2 当前进展

- 已开始拆分推荐商品行为消费者内部职责，先把事实入库与投影更新分层。
- 已将商品行为投影器下沉到 `pkg/recommend/offline/aggregate`，避免继续在 `service/app/biz` 扩展推荐聚合逻辑。
- 已将 `recommend_user_goods_preference`、`recommend_user_preference`、`recommend_goods_relation` 的离线重建逻辑统一下沉到 `pkg/recommend/offline/aggregate`，`biz` 层只保留删旧数据、调用聚合器和批量落库。
- 已将 `RecommendGoodsStatDay` 的按天聚合逻辑统一下沉到 `pkg/recommend/offline/aggregate`，`pkg/job/task` 中的任务入口只保留日期解析、删旧数据和批量回写。
- 当前仍保留原有队列主题、事务入口和在线推荐主读路径，避免阶段 2 初期影响运行稳定性。
- 集合去重优先库已固定为 `golang-set/v2`，位图实现优先库已固定为 `bitset`，并记录到长期规划中。
- 已完成首轮代码回归验证，`cd backend && go test ./...` 通过。

### 阶段 3 当前进展

- 已补齐推荐缓存集合名、子集合、摘要键、更新时间键的统一命名规则。
- 已新增 `pkg/recommend/offline/materialize` 写缓存模块，首批支持 `hot`、`latest`、`similar_item` 三类结果发布到统一缓存协议。
- 已接入基于 `kratos-kit/cache` 的推荐缓存 store，当前无 Redis 时走内存后端，有 Redis 时直接复用 Redis。
- 已将 `RecommendHotMaterialize`、`RecommendLatestMaterialize`、`RecommendSimilarItemMaterialize` 注册到定时任务调度。
- 已把 `hot`、`latest`、`similar_item` 三类缓存优先读取挂到 `RecommendRequestCase`，未命中时回退原查库路径。
- 已在推荐请求 `sourceContext` 中记录 `cacheHitSources`，便于排查当前结果命中了哪类缓存。
- 当前仍未抽离独立在线引擎，也未补完整的缓存命中率指标面板。

### 阶段 4 当前进展

- 已新增 `service/app/biz/recommend_recall_probe.go`，用于按场景版本读取相似用户、协同过滤、内容相似三类召回探针。
- 已在 `pkg/recommend/cache` 中补齐相似用户、协同过滤、内容相似三类缓存集合与子集合键约定。
- 已在 `pkg/recommend/domain` 中补齐 `recall_probe` 配置结构，用于承接 `recommend_model_version.config_json` 内的版本化探针开关。
- 已将探针读取结果写入推荐请求 `sourceContext`，并在探针上下文中补充 `joinCandidate` 标记，便于区分“只观测”和“参与候选”。
- 已在 `GOODS_DETAIL` 场景接入首批灰度候选融合：匿名态可按版本并入内容相似，登录态可按版本并入内容相似和协同过滤；相似用户当前仍只做观测不直接并入候选。
- 已将灰度召回的排障信息写入 `joinRecallContext`，可以直接看到每类灰度召回“已并入候选”“实际进入候选池”“实际出现在当前页”的商品编号与来源列表。
- 已将相似用户探针的观测结果写入 `similarUserObservationContext`，可以直接看到相似用户偏好商品 TopN 与当前候选、当前返回页，以及协同过滤、内容相似灰度结果的交集商品和覆盖率。
- 已将上述在线排障字段统一收口到 `sourceContext.onlineDebugContext`，避免 `sourceContext` 顶层继续扩散调试字段。
- 已为在线读缓存和召回探针补读取元信息，当前会记录版本号、版本发布时间、缓存发布时间、文档数量、扫描数量、返回数量和命中状态，便于定位“命中了哪一版缓存”和“当前缓存是否过旧”。
- 当前探针已经有首批训练产物可读，下一步可以继续评估相似用户召回和更多场景的候选融合灰度。

### 阶段 5 当前进展

- 已新增 `RecommendSimilarUserMaterialize`，基于用户类目偏好重叠和商品偏好重叠训练相似用户结果，并按版本发布到 `user-to-user` 缓存。
- 已新增 `RecommendCollaborativeFilteringMaterialize`，基于相似用户结果和邻居商品偏好聚合协同过滤候选，并按版本发布到 `collaborative-filtering` 缓存。
- 已新增 `RecommendContentBasedMaterialize`，基于同类目商品的价格接近度与新鲜度接近度构建内容相似结果，并按版本发布到 `content-based` 缓存。
- 已将三类任务注册到定时任务调度与依赖注入。
- 已为 `hot`、`latest`、`similar_item`、`similar_user`、`collaborative_filtering`、`content_based` 六类写缓存任务补最小摘要日志；其中阶段 5 的训练发布任务会额外记录偏好记录数、候选用户数、候选商品数等输入规模，统一输出版本数、发布子集合数、发布文档数、清理子集合数和总耗时。
- 已为上述六类写缓存任务补统一失败摘要日志，任务异常时会带出当前执行阶段、已累计的输入规模、已发布进度、已清理进度和耗时，便于直接定位失败卡点。
- 已为写缓存元信息补 `document_count`，并在在线请求 `cacheReadContext` 中收口版本发布时间、缓存发布时间和文档数量，作为阶段 6 灰度切主链路前的最小观测基础。
- 当前仍未补 CTR / CVR 轻量排序模型、调参任务和模型产物发布协议。

### 阶段 6 当前进展

- 已新增 `pkg/recommend/online/recall`，开始承接在线层的纯召回逻辑。
- 已将探针结果解析、灰度召回上下文归一化和相似用户观测上下文从 `service/app/biz/recommend_recall_probe.go` 抽离到 `pkg/recommend/online/recall`。
- 已新增 `pkg/recommend/online/planner`，用于承接请求计划对象、候选前置状态、缓存命中状态和来源上下文基础收口。
- `service/app/biz/recommend_request.go` 的匿名态和登录态前置状态当前已统一改为通过 planner 对象流转，减少 `priority/category/recall/cache` 零散变量继续扩散。
- `service/app/biz/recommend_request.go` 中 `CART`、`ORDER_*`、`GOODS_DETAIL`、`profile` 和 `latest` fallback 的场景级规划动作当前也已通过 planner 方法收口，开始从场景 `switch` 中剥离纯编排逻辑。
- `pkg/recommend/online/planner` 当前已继续补 `SceneInput` 场景桥接输入，用于承接购物车商品、订单商品、源商品、场景优先候选、类目补足和缓存命中来源等原始输入；`recommend_request.go` 中的场景 `switch` 进一步收敛为只保留查库和桥接。
- 在线来源上下文中的 `orderId`、`goodsId`、`cartGoodsIds`、`orderGoodsIds`、`sourceGoodsIds` 当前也已开始通过 planner 统一构建，避免这些调试字段继续散落在主链路函数中手工拼 map。
- 结果来源上下文与在线调试上下文当前也已继续通过 planner 收口，匿名态和登录态都会先走 `BuildSceneResultSourceContext`，再由 planner 统一补 join recall 和 similar-user observation 调试字段，主链路里不再直接依赖 `online/recall` 组装这些上下文。
- `pkg/recommend/online/planner` 当前又继续补了 `ResultSnapshot`，用于承接 `candidateLimit`、`sceneHotGoodsIds`、`candidateGoodsIds`、`anonymousCandidateGoodsIds`、`returnedScoreDetails` 等结果快照字段；匿名态和登录态的最终来源上下文当前已开始统一改成通过 planner 结果对象构建。
- `pkg/recommend/online/planner` 当前又继续补了候选池状态方法，用于承接类目补足候选、latest 候选、latest 排除集合、匿名态场景候选合并和登录态最终候选集合合并；`recommend_request.go` 中这类纯集合状态和去重规则继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了候选池查询参数计划，用于承接类目补足查询参数、latest 查询参数和匿名态 latest fallback 判断；`recommend_request.go` 中这类“是否需要查、查多少、排除哪些商品”的纯参数决策继续从主链路函数里迁出。
- `service/app/biz/recommend_request.go` 当前又继续补了共享候选池桥接查询方法，类目补足和 latest 兜底的商品 ID 提取逻辑已经开始复用统一实现，主链路里的重复查询拼装和结果提取代码进一步缩短。
- `service/app/biz/recommend_request.go` 当前又继续补了匿名态 latest 回退商品列表桥接方法，把 latest fallback 的分页查询与总数返回统一收口到共享桥接层，主链路里的匿名态 latest 回退分支进一步缩短。
- `pkg/recommend/online/planner` 当前又继续补了排序信号快照方法，用于承接匿名态与登录态候选商品列表的过滤、商品 ID 提取和类目 ID 提取；`recommend_request.go` 中候选信号加载前的结果整理逻辑继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了分页 explain 快照方法，用于统一承接当前页召回来源列表、评分明细和返回商品编号提取；`recommend_request.go` 中匿名态与登录态的 explain 组装循环继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了排序结果分页窗口快照方法，用于统一承接 `total`、`offset/end` 计算、当前页商品切片和空页判定；`recommend_request.go` 中匿名态与登录态的分页窗口与空页分支继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了结果回写桥接方法，用于统一承接匿名态 latest fallback、匿名态空页、匿名态正常页、登录态空页和登录态正常页的 `ResultSnapshot` 构建与 `sourceContext` 回写；`recommend_request.go` 中末尾结果回写分支继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了排序信号加载计划，用于统一承接匿名态与登录态的场景编号、候选商品编号、候选类目编号和关系分源商品编号；`recommend_request.go` 中排序信号加载前的参数组织继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了领域信号桥接方法，用于统一承接匿名态 `AnonymousSignals` 和登录态 `PersonalizedSignals` 的组装；`recommend_request.go` 中信号加载完成后的领域对象桥接继续从主链路函数里迁出。
- `pkg/recommend/online/planner` 当前又继续补了 explain 召回补标方法，用于统一承接匿名态内容相似灰度补标，以及登录态内容相似、协同过滤灰度补标；`recommend_request.go` 中 `appendRecommendCandidateRecallSources` 的直接调用继续从主链路函数里迁出。
- `service/app/biz/recommend_request.go` 当前又继续补了共享分页桥接底层方法，用于统一承接类目补足、latest 候选和 latest fallback 的 `PageGoodsInfo` 调用、排除商品过滤和分页参数拼装；主链路末尾仅保留返回值差异。
- `pkg/recommend/online/record` 当前已开始承接结果记录前的纯整理逻辑，首批补了推荐请求主表 `sourceContext` 的精简与在线调试上下文收口；`saveRecommendRequest` 当前只保留序列化、事务和落库调用。
- `pkg/recommend/online/record` 当前又继续补了推荐请求逐商品明细模型构建方法，用于统一承接 `returnedScoreDetails` 索引收敛、单商品召回来源回退和 `RecommendRequestItem` 列表组装；`recommend_request_item.go` 中批量落库前的纯整理逻辑继续从业务层迁出。
- `pkg/recommend/online/record` 当前又继续补了推荐请求逐商品明细读取整理方法，用于统一承接关联商品编号提取和商品位次映射构建；`recommend_request_item.go` 中按 requestId 回查明细后的纯循环整理逻辑继续从业务层迁出。
- `service/app/biz/recommend_request_item.go` 当前又继续补了共享读桥接方法，用于统一承接 `requestId -> requestEntity` 和 `requestEntity.ID -> requestItemList` 的查询路径；按 requestId 回查关联商品和位次映射的重复查询拼装继续缩短。
- `pkg/recommend/online/planner` 当前又继续补了 `ListGoodsIds`，用于统一承接 explain 快照、类目补足候选和 latest 候选的商品编号提取；`recommend_request.go` 中本地的纯商品 ID 提取工具已继续从业务层迁出。
- `pkg/recommend/online/planner` 当前又继续补了 `GoodsPoolPageSnapshot` 和 `GoodsPoolQuery` 的可执行判断方法，用于统一承接候选池分页桥接结果中的 `list/id/total` 提取和空计划判断；`recommend_request.go` 中类目补足、latest 候选和 latest fallback 的结果侧分支继续从业务层迁出。
- `pkg/recommend/online/planner` 当前又继续补了 `BuildCartSceneInput`、`BuildOrderSceneInput` 和 `BuildGoodsDetailSceneInput`，用于统一承接 `CART`、`ORDER_*`、`GOODS_DETAIL` 场景的 `SceneInput` 映射；`recommend_request.go` 中场景分支里的字段赋值拼装继续从业务层迁出。
- `service/app/biz/recommend_request.go` 当前改为调用在线层的召回辅助函数，先复用原主链路入口，避免在阶段 6 初期同时改接口编排和结果落库。
- 当前仍未引入统一在线引擎入口、planner / recall / rank / record 分层和场景级灰度切换。

## 下阶段启动清单

阶段 5 继续推进与阶段 6 启动前，先完成以下准备：

- 保持现有队列入口、在线推荐接口和推荐结果落库逻辑不变，避免阶段 4 与主链路改造叠在一起。
- 继续补缓存命中率、写缓存耗时、版本发布时间等最小可观测字段，避免探针接入后仍然无法判断效果。
- 评估哪些探针可以先纳入候选融合灰度，优先选择风险较低的商品详情场景。
- 继续补 CTR / CVR 轻量排序模型、调参任务和模型产物发布协议，避免阶段 5 只停留在召回训练。
- 明确阶段 6 切主链路时的灰度范围、版本切换条件和回滚方式，避免探针和默认召回并线后难以排障。
