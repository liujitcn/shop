package recommendcontext

import (
	"strconv"
	"strings"

	"shop/api/gen/go/common"
)

// ParseScene 解析任务入参中的推荐场景。
func ParseScene(scene string) int32 {
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

// NormalizeSceneEnum 统一 proto 场景枚举到存储值。
func NormalizeSceneEnum(scene common.RecommendScene) int32 {
	return int32(scene)
}

// HasRequest 判断推荐请求 ID 是否有效。
func HasRequest(requestID string) bool {
	return strings.TrimSpace(requestID) != ""
}

// FormatScene 将存储值格式化为可读场景名。
func FormatScene(scene int32) string {
	sceneValue := common.RecommendScene(scene)
	if sceneValue == common.RecommendScene_RECOMMEND_SCENE_UNKNOWN {
		return ""
	}
	if _, ok := common.RecommendScene_name[int32(sceneValue)]; ok {
		return sceneValue.String()
	}
	return ""
}
