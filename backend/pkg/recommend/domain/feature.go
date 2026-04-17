package domain

// PersonalizedSignals 表示登录态候选所需的评分信号。
type PersonalizedSignals struct {
	RelationScores         map[int64]float64  // 商品关系分映射：商品编号 -> 分数
	UserGoodsScores        map[int64]float64  // 用户商品偏好分映射：商品编号 -> 分数
	SimilarUserScores      map[int64]float64  // 相似用户偏好分映射：商品编号 -> 分数
	ProfileScores          map[int64]float64  // 用户类目画像分映射：类目编号 -> 分数
	ScenePopularityScores  map[int64]float64  // 场景热度分映射：商品编号 -> 分数
	GlobalPopularityScores map[int64]float64  // 全站热度分映射：商品编号 -> 分数
	SceneExposurePenalties map[int64]float64  // 场景曝光惩罚映射：商品编号 -> 惩罚分
	ActorExposurePenalties map[int64]float64  // 主体曝光惩罚映射：商品编号 -> 惩罚分
	RecentPaidGoods        map[int64]struct{} // 近期已支付商品集合：商品编号 -> 占位值
}

// AnonymousSignals 表示匿名候选所需的评分信号。
type AnonymousSignals struct {
	RelationScores         map[int64]float64 // 商品关系分映射：商品编号 -> 分数
	ScenePopularityScores  map[int64]float64 // 场景热度分映射：商品编号 -> 分数
	GlobalPopularityScores map[int64]float64 // 全站热度分映射：商品编号 -> 分数
	SceneExposurePenalties map[int64]float64 // 场景曝光惩罚映射：商品编号 -> 惩罚分
	ActorExposurePenalties map[int64]float64 // 主体曝光惩罚映射：商品编号 -> 惩罚分
}

// TrainArtifactMeta 表示离线训练产物的元信息。
type TrainArtifactMeta struct {
	Scene         int32  // 推荐场景
	ArtifactType  string // 产物类型：相似商品、相似用户、协同过滤等
	StrategyName  string // 产出该训练结果的策略名称
	StrategyKey   string // 训练策略唯一键
	CacheVersion  string // 对应缓存版本号
	ArtifactCount int64  // 当前产物数量
}
