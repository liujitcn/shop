# 场景 Pipeline

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

### 排序信号

- 用户商品偏好
- 用户类目偏好
- 场景热度
- 全站热度
- 新鲜度
- 曝光惩罚
- 重复购买惩罚

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

### 排序信号

- 商品关联得分
- 当前会话得分
- 场景热度
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

### 排序信号

- 购物车商品关联度
- 当前会话得分
- 场景热度
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

### 排序信号

- 用户商品偏好
- 用户类目偏好
- 全站热度
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

### 排序信号

- 订单商品关联度
- 场景热度
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

### 排序信号

- 订单商品关联度
- 复购惩罚
- 场景热度
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
