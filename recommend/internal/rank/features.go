package rank

import (
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/internal/model"
	"sort"
)

const (
	// featureRelation 表示商品关联信号特征。
	featureRelation = "relation"
	// featureUserGoods 表示用户商品偏好信号特征。
	featureUserGoods = "user_goods"
	// featureCategory 表示用户类目偏好信号特征。
	featureCategory = "category"
	// featureSceneHot 表示场景热度信号特征。
	featureSceneHot = "scene_hot"
	// featureGlobalHot 表示全站热度信号特征。
	featureGlobalHot = "global_hot"
	// featureFreshness 表示新鲜度信号特征。
	featureFreshness = "freshness"
	// featureSession 表示会话上下文信号特征。
	featureSession = "session"
	// featureExternal 表示外部召回信号特征。
	featureExternal = "external"
	// featureCollaborative 表示协同过滤信号特征。
	featureCollaborative = "collaborative"
	// featureUserNeighbor 表示相似用户信号特征。
	featureUserNeighbor = "user_neighbor"
	// featureVector 表示向量召回信号特征。
	featureVector = "vector"
	// featureExposure 表示曝光惩罚特征。
	featureExposure = "exposure_penalty"
	// featureRepeat 表示重复购买惩罚特征。
	featureRepeat = "repeat_penalty"
)

var rankingFeatureNames = []string{
	featureRelation,
	featureUserGoods,
	featureCategory,
	featureSceneHot,
	featureGlobalHot,
	featureFreshness,
	featureSession,
	featureExternal,
	featureCollaborative,
	featureUserNeighbor,
	featureVector,
	featureExposure,
	featureRepeat,
}

// RankingFeatureNames 返回学习排序使用的全部基础特征名。
func RankingFeatureNames() []string {
	return append([]string(nil), rankingFeatureNames...)
}

// BuildFeatureMap 根据内部候选结构提取学习排序特征。
func BuildFeatureMap(candidate *model.Candidate) map[string]float64 {
	if candidate == nil {
		return nil
	}
	return map[string]float64{
		featureRelation:      candidate.Score.RelationScore,
		featureUserGoods:     candidate.Score.UserGoodsScore,
		featureCategory:      candidate.Score.CategoryScore,
		featureSceneHot:      candidate.Score.SceneHotScore,
		featureGlobalHot:     candidate.Score.GlobalHotScore,
		featureFreshness:     candidate.Score.FreshnessScore,
		featureSession:       candidate.Score.SessionScore,
		featureExternal:      candidate.Score.ExternalScore,
		featureCollaborative: candidate.Score.CollaborativeScore,
		featureUserNeighbor:  candidate.Score.UserNeighborScore,
		featureVector:        candidate.Score.VectorScore,
		featureExposure:      candidate.Score.ExposurePenalty,
		featureRepeat:        candidate.Score.RepeatPenalty,
	}
}

// BuildFeatureMapFromScoreDetail 根据 trace 评分明细提取学习排序特征。
func BuildFeatureMapFromScoreDetail(detail *recommendv1.RecommendScoreDetail) map[string]float64 {
	if detail == nil {
		return nil
	}
	return map[string]float64{
		featureRelation:      detail.GetRelationScore(),
		featureUserGoods:     detail.GetUserGoodsScore(),
		featureCategory:      detail.GetCategoryScore(),
		featureSceneHot:      detail.GetSceneHotScore(),
		featureGlobalHot:     detail.GetGlobalHotScore(),
		featureFreshness:     detail.GetFreshnessScore(),
		featureSession:       detail.GetSessionScore(),
		featureExternal:      detail.GetExternalScore(),
		featureCollaborative: detail.GetCollaborativeScore(),
		featureUserNeighbor:  detail.GetUserNeighborScore(),
		featureVector:        detail.GetVectorScore(),
		featureExposure:      detail.GetExposurePenalty(),
		featureRepeat:        detail.GetRepeatPenalty(),
	}
}

// PairFeatureKey 返回稳定有序的二阶交叉特征键。
func PairFeatureKey(leftFeature, rightFeature string) string {
	if leftFeature <= rightFeature {
		return leftFeature + "*" + rightFeature
	}
	return rightFeature + "*" + leftFeature
}

// SortedInteractionKeys 返回稳定排序后的二阶交叉特征键列表。
func SortedInteractionKeys() []string {
	keys := make([]string, 0, len(rankingFeatureNames)*len(rankingFeatureNames))
	for leftIndex := 0; leftIndex < len(rankingFeatureNames); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(rankingFeatureNames); rightIndex++ {
			keys = append(keys, PairFeatureKey(rankingFeatureNames[leftIndex], rankingFeatureNames[rightIndex]))
		}
	}
	sort.Strings(keys)
	return keys
}
