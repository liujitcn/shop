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

### 排序

- 规则排序
- 预留 `fm`
- 预留 `llm`

### replacement / fallback

- 过滤
- 惩罚
- 兜底
- 打散

### 离线能力

- 候选池构建
- 运行态构建
- trace 持久化
- 离线评估

## 在当前设计中的落点

- 在线推荐：`Recommend(...)`
- explain：`Explain(...)`
- 运行态同步：`SyncExposure(...)`、`SyncBehavior(...)`、`SyncActorBind(...)`
- 非个性化构建：`BuildNonPersonalized(...)`
- 偏好候选构建：`BuildUserCandidate(...)`
- 商品关联构建：`BuildGoodsRelation(...)`
- 相似用户构建：`BuildUserToUser(...)`
- 协同过滤构建：`BuildCollaborative(...)`
- 外部池构建：`BuildExternal(...)`
- 离线评估：`EvaluateOffline(...)`

## 刻意不覆盖的能力

当前模块不覆盖以下 `gorse` 系统级能力：

- `master` 节点
- `server` 节点
- 分布式 `worker` 集群
- dashboard
- 通用数据导入导出平台
- 通用 RESTful API

原因不是能力缺失，而是当前模块目标明确限定为 `shop` 的推荐工具库。
