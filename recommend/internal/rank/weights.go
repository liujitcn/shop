package rank

import "recommend/internal/model"

// DefaultWeights 返回指定场景的默认排序权重。
func DefaultWeights(scene model.Scene) ScoreWeights {
	// 首页场景强调用户偏好、场景热度和增强召回平衡。
	switch scene {
	case model.SceneHome:
		return ScoreWeights{
			UserGoodsWeight:     0.24,
			CategoryWeight:      0.16,
			SceneHotWeight:      0.14,
			GlobalHotWeight:     0.08,
			FreshnessWeight:     0.08,
			SessionWeight:       0.04,
			ExternalWeight:      0.10,
			CollaborativeWeight: 0.08,
			UserNeighborWeight:  0.08,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		}
	// 商品详情场景以商品关联为核心，其次叠加会话与热度信号。
	case model.SceneGoodsDetail:
		return ScoreWeights{
			RelationWeight:      0.42,
			SceneHotWeight:      0.10,
			FreshnessWeight:     0.06,
			SessionWeight:       0.16,
			ExternalWeight:      0.12,
			CollaborativeWeight: 0.08,
			UserNeighborWeight:  0.04,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		}
	// 购物车场景强调与当前购物车商品的搭配关系。
	case model.SceneCart:
		return ScoreWeights{
			RelationWeight:      0.34,
			SceneHotWeight:      0.12,
			FreshnessWeight:     0.06,
			SessionWeight:       0.18,
			ExternalWeight:      0.12,
			CollaborativeWeight: 0.08,
			UserNeighborWeight:  0.04,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		}
	// 个人中心场景强调用户偏好和增强个性化信号。
	case model.SceneProfile:
		return ScoreWeights{
			UserGoodsWeight:     0.22,
			CategoryWeight:      0.18,
			GlobalHotWeight:     0.08,
			FreshnessWeight:     0.08,
			ExternalWeight:      0.10,
			CollaborativeWeight: 0.12,
			UserNeighborWeight:  0.12,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		}
	// 订单详情场景强调与当前订单商品的关系。
	case model.SceneOrderDetail:
		return ScoreWeights{
			RelationWeight:      0.38,
			SceneHotWeight:      0.14,
			FreshnessWeight:     0.08,
			ExternalWeight:      0.14,
			CollaborativeWeight: 0.10,
			UserNeighborWeight:  0.06,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		}
	// 支付完成场景强调复购类召回和增强召回信号。
	case model.SceneOrderPaid:
		return ScoreWeights{
			RelationWeight:      0.24,
			SceneHotWeight:      0.12,
			FreshnessWeight:     0.08,
			ExternalWeight:      0.16,
			CollaborativeWeight: 0.14,
			UserNeighborWeight:  0.14,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.2,
		}
	// 未识别场景时，回退到通用权重，保证排序链路仍可执行。
	default:
		return ScoreWeights{
			RelationWeight:      0.20,
			UserGoodsWeight:     0.15,
			CategoryWeight:      0.10,
			SceneHotWeight:      0.12,
			GlobalHotWeight:     0.08,
			FreshnessWeight:     0.08,
			SessionWeight:       0.06,
			ExternalWeight:      0.08,
			CollaborativeWeight: 0.06,
			UserNeighborWeight:  0.05,
			ExposurePenalty:     1.0,
			RepeatPenalty:       1.0,
		}
	}
}
