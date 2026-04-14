# 场景 Pipeline

所有在线请求会先在 `*Recommend` 实例中归一化为 `internal/core` 公共请求，再进入下面的场景 pipeline，避免场景层直接依赖根包 DTO。

## home

### 主召回

- `user_goods_pref`
- `user_category_pref`
- `scene_hot`
- `global_hot`
- `latest`
- `external`
- `user_to_user`
- `collaborative`
- `vector`

### 排序信号

- 用户商品偏好
- 用户类目偏好
- 场景热度
- 全站热度
- 新鲜度
- 向量召回
- 曝光惩罚
- 重复购买惩罚

运行态说明：

- `SyncExposure(...)` 写入的曝光惩罚会在在线排序阶段直接扣分
- `SyncBehavior(...)` 写入的下单/支付惩罚会在在线排序阶段直接扣分
- 实例级 `RankingConfig` 会覆写场景默认权重与类目打散上限

### replacement

- 已曝光未点击商品降权
- 近期已购商品降权
- 类目打散
- 品牌打散
- latest 兜底

## goods_detail

### 主召回

- `goods_relation`
- `session_context`
- `scene_hot`
- `latest`
- `external`
- `collaborative`
- `vector`

### 排序信号

- 商品关联得分
- 当前会话得分
- 场景热度
- 向量召回得分
- 新鲜度
- 曝光惩罚

### replacement

- 当前详情商品过滤
- 无库存过滤
- 同类商品打散
- latest 兜底

## cart

### 主召回

- `goods_relation`
- `session_context`
- `scene_hot`
- `external`
- `collaborative`
- `vector`

### 排序信号

- 购物车商品关联度
- 当前会话得分
- 场景热度
- 向量召回得分
- 曝光惩罚

### replacement

- 已在购物车中的商品过滤
- 库存过滤
- 类目打散
- scene_hot 兜底

## profile

### 主召回

- `user_goods_pref`
- `user_category_pref`
- `global_hot`
- `latest`
- `external`
- `user_to_user`
- `collaborative`
- `vector`

### 排序信号

- 用户商品偏好
- 用户类目偏好
- 全站热度
- 向量召回得分
- 新鲜度
- 曝光惩罚

### replacement

- 已购惩罚
- 曝光惩罚
- latest 兜底

## order_detail

### 主召回

- `goods_relation`
- `scene_hot`
- `latest`
- `external`
- `collaborative`
- `vector`

### 排序信号

- 订单商品关联度
- 场景热度
- 向量召回得分
- 新鲜度

### replacement

- 当前订单商品过滤
- 类目打散
- latest 兜底

## order_paid

### 主召回

- `goods_relation`
- `scene_hot`
- `latest`
- `external`
- `user_to_user`
- `collaborative`
- `vector`

### 排序信号

- 订单商品关联度
- 复购惩罚
- 场景热度
- 向量召回得分
- 新鲜度

### replacement

- 近期已支付商品降权
- 当前订单商品过滤
- 类目打散
- latest 兜底

## 覆盖说明

- `latest / scene_hot / global_hot` 覆盖 gorse 的非个性化推荐
- `goods_relation` 覆盖 item-to-item
- `session_context` 覆盖 session-based recommendation
- `user_to_user` 和 `collaborative` 作为增强召回，不只停留在首页
- `external` 覆盖活动池、人工池、营销池等外部推荐器

## 当前离线构建映射

- `BuildNonPersonalized(...)` 当前会按 `Strategy.NonPersonalizedSources` 指定的来源顺序写入匿名通用候选池
- `BuildUserCandidate(...)` 当前会为 `home`、`profile` 写入 `user_goods_pref + user_category_pref` 合并池
- `BuildGoodsRelation(...)` 当前会为 `goods_detail`、`cart`、`order_detail`、`order_paid` 写入关联池
- `BuildCollaborative(...)` 当前会为全部场景写入协同过滤池
- `BuildExternal(...)` 当前会按 `scene + strategy + actor` 维度写入外部池
- `BuildVector(...)` 当前会为 `home`、`profile` 写入用户向量池，并为详情类场景写入商品向量池
- `TrainRanking(...)` 当前会按场景读取 trace 和行为事实训练轻量 FM 模型
- `Rebuild(...)` 当前会把上述离线构建动作按统一请求编排，并可选串联学习排序训练与 `EvaluateOffline(...)`

## 当前在线读池映射

- `latest` 在线优先读取 `BuildNonPersonalized(...)` 产出的匿名通用候选池
- `scene_hot` 在线优先读取 `BuildNonPersonalized(...)` 产出的匿名通用候选池，并按 `source_scores.scene_hot` 恢复原始分值
- `global_hot` 在线优先读取 `BuildNonPersonalized(...)` 产出的匿名通用候选池，并按 `source_scores.global_hot` 恢复原始分值
- `user_goods_pref` 在线优先读取 `BuildUserCandidate(...)` 产出的用户候选池，并按 `source_scores.user_goods_pref` 恢复原始分值
- `user_category_pref` 在线优先读取 `BuildUserCandidate(...)` 产出的用户候选池，并按 `source_scores.user_category_pref` 恢复原始分值
- `session_context` 在线优先读取 `SyncBehavior(...)` 持续维护的运行态会话序列，缺失时回退 `ListSessionEvents(...)`
- `user_to_user` 在线优先读取 `BuildUserToUser(...)` 产出的相似用户池商品项，并按 `source_scores.user_to_user` 恢复原始分值
- `goods_relation` 在线优先读取 `BuildGoodsRelation(...)` 产出的商品关联池，缺失时回退 `ListRelatedGoods(...)`
- `collaborative` 在线优先读取 `BuildCollaborative(...)` 产出的协同过滤池，缺失时回退 `ListCollaborativeGoods(...)`
- `external` 在线优先读取 `BuildExternal(...)` 产出的外部推荐池，缺失时回退 `ListExternalGoods(...)`
- `vector` 在线优先读取 `BuildVector(...)` 产出的向量池，缺失时回退 `VectorSource.ListVectorGoods(...)`

## 排序模式

- `rule`：直接使用规则权重排序
- `fm`：先保留规则信号，再根据已训练模型重算最终分
- `llm`：先跑规则分，再对前 N 个候选执行二阶段重排
