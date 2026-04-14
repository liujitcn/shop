# gorse 推荐能力映射

## 已规划覆盖的能力

### 多路召回

- `non-personalized`
  - 对应：`latest`、`scene_hot`、`global_hot`
- `item-to-item`
  - 对应：`goods_relation`
- `session recommendation`
  - 对应：`session_context`
- `user-to-user`
  - 对应：`user_to_user`
- `collaborative filtering`
  - 对应：`collaborative`
- `external recommender`
  - 对应：`external`
- `vector recall`
  - 对应：`vector`

### 排序

- 规则排序
- 实例级场景权重配置
- `fm`
- `llm`

### replacement / fallback

- 过滤
- 惩罚
- 兜底
- 打散

### 离线能力

- 候选池构建
- 一键 `Rebuild(...)` 编排
- 运行态构建
- trace 持久化
- 离线评估

## 在当前设计中的落点

- 类型边界：根包 DTO 对外稳定，内部统一经 `internal/core` 进入推荐内核，避免在线入口和内核实现相互依赖
- 在线推荐：`Recommend(...)`
- explain：`Explain(...)`
- 运行态同步：`SyncExposure(...)`、`SyncBehavior(...)`、`SyncActorBind(...)`
- 一键重建：`Rebuild(...)`
- 非个性化构建：`BuildNonPersonalized(...)`
- 偏好候选构建：`BuildUserCandidate(...)`
- 商品关联构建：`BuildGoodsRelation(...)`
- 相似用户构建：`BuildUserToUser(...)`
- 协同过滤构建：`BuildCollaborative(...)`
- 外部池构建：`BuildExternal(...)`
- 向量池构建：`BuildVector(...)`
- 学习排序训练：`TrainRanking(...)`
- 离线评估：`EvaluateOffline(...)`

## 当前已落地的非骨架能力

- `Recommend(...)` 已写入 trace 明细
- `Explain(...)` 已支持缓存回查
- `SyncExposure(...)` 已更新曝光惩罚并补 trace
- `SyncBehavior(...)` 已更新会话态、复购惩罚并补 trace
- `SyncActorBind(...)` 已归并匿名态到登录态
- `Recommend(...)` 已优先读取匿名通用候选池和用户候选池，并按 `source_scores` 恢复独立排序信号
- `Recommend(...)` 已优先读取 `runtime.db` 中的会话态并构造 session_context 召回
- `Recommend(...)` 已优先读取相似用户池中的 user-to-user 商品项
- `Recommend(...)` 已优先读取商品关联池、协同过滤池、外部推荐池，缺失时回退事实源
- `Recommend(...)` 已优先读取向量召回池，缺失时回退向量 provider
- `Recommend(...)` 已支持 `fm` 学习排序模型读取与在线打分
- `Recommend(...)` 已支持 `llm` 二阶段重排
- `BuildNonPersonalized(...)` 已写入匿名通用候选池
- `BuildUserCandidate(...)`、`BuildGoodsRelation(...)`、`BuildUserToUser(...)`、`BuildCollaborative(...)`、`BuildExternal(...)` 已写入对应离线池
- `BuildVector(...)` 已写入用户向量池和商品向量池
- `TrainRanking(...)` 已根据 trace 和行为事实训练轻量 FM 模型并写入 `runtime.db`
- `Rebuild(...)` 已可统一编排多类离线池构建、学习排序训练并串联离线评估
- `EvaluateOffline(...)` 已返回场景级排序指标和转化指标

## 刻意不覆盖的能力

当前模块不覆盖以下 `gorse` 系统级能力：

- `master` 节点
- `server` 节点
- 分布式 `worker` 集群
- dashboard
- 通用数据导入导出平台
- 通用 RESTful API

原因不是能力缺失，而是当前模块目标明确限定为 `shop` 的推荐工具库。
