package biz

import (
	"strconv"
	"strings"

	"shop/api/gen/go/common"
)

func ParseRecommendSceneForTask(scene string) int32 {
	return parseRecommendScene(scene)
}

func normalizeRecommendSceneEnum(scene common.RecommendScene) int32 {
	return int32(scene)
}

func hasRecommendRequest(requestID string) bool {
	return strings.TrimSpace(requestID) != ""
}

func parseRecommendScene(scene string) int32 {
	value := strings.TrimSpace(scene)
	if value == "" {
		return int32(common.RecommendScene_RECOMMEND_SCENE_UNKNOWN)
	}
	if number, err := strconv.Atoi(value); err == nil {
		sceneValue := common.RecommendScene(number)
		if _, ok := common.RecommendScene_name[int32(sceneValue)]; ok {
			return int32(sceneValue)
		}
		return int32(common.RecommendScene_RECOMMEND_SCENE_UNKNOWN)
	}
	switch value {
	case common.RecommendScene_HOME.String():
		return int32(common.RecommendScene_HOME)
	case common.RecommendScene_CART.String():
		return int32(common.RecommendScene_CART)
	case common.RecommendScene_PROFILE.String():
		return int32(common.RecommendScene_PROFILE)
	case common.RecommendScene_ORDER_DETAIL.String():
		return int32(common.RecommendScene_ORDER_DETAIL)
	case common.RecommendScene_ORDER_PAID.String():
		return int32(common.RecommendScene_ORDER_PAID)
	default:
		return int32(common.RecommendScene_RECOMMEND_SCENE_UNKNOWN)
	}
}

func formatRecommendScene(scene int32) string {
	sceneValue := common.RecommendScene(scene)
	if sceneValue == common.RecommendScene_RECOMMEND_SCENE_UNKNOWN {
		return ""
	}
	if _, ok := common.RecommendScene_name[int32(sceneValue)]; ok {
		return sceneValue.String()
	}
	return ""
}

func FormatRecommendSceneForTask(scene int32) string {
	return formatRecommendScene(scene)
}
