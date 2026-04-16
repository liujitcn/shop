package record

import (
	"encoding/json"

	"shop/api/gen/go/app"
	"shop/pkg/gen/models"
	recommendcore "shop/pkg/recommend/core"
)

// BuildRecommendRequestItems 构建推荐请求逐商品明细模型列表。
func BuildRecommendRequestItems(
	recommendRequestId int64,
	pageNum int64,
	pageSize int64,
	sourceContext map[string]any,
	list []*app.GoodsInfo,
	recallSources []string,
) []*models.RecommendRequestItem {
	// 主表编号或返回结果缺失时，不生成逐商品明细。
	if recommendRequestId <= 0 || len(list) == 0 {
		return []*models.RecommendRequestItem{}
	}

	scoreDetailMap := buildScoreDetailMap(sourceContext)
	positionBase := (pageNum - 1) * pageSize
	requestItemList := make([]*models.RecommendRequestItem, 0, len(list))
	for index, item := range list {
		// 非法商品结果直接跳过，避免脏数据写入逐商品明细表。
		if item == nil || item.GetId() <= 0 {
			continue
		}

		scoreDetail, ok := scoreDetailMap[item.GetId()]
		recallSourceJson := buildRecallSourceJSON(recallSources, scoreDetail, ok)
		requestItemList = append(requestItemList, &models.RecommendRequestItem{
			RecommendRequestID:    recommendRequestId,
			GoodsID:               item.GetId(),
			Position:              int32(positionBase + int64(index)),
			RecallSource:          recallSourceJson,
			FinalScore:            scoreDetail.FinalScore,
			RelationScore:         scoreDetail.RelationScore,
			UserGoodsScore:        scoreDetail.UserGoodsScore,
			ProfileScore:          scoreDetail.ProfileScore,
			ScenePopularityScore:  scoreDetail.ScenePopularityScore,
			GlobalPopularityScore: scoreDetail.GlobalPopularityScore,
			FreshnessScore:        scoreDetail.FreshnessScore,
			ExposurePenalty:       scoreDetail.ExposurePenalty,
			ActorExposurePenalty:  scoreDetail.ActorExposurePenalty,
			RepeatPenalty:         scoreDetail.RepeatPenalty,
		})
	}
	return requestItemList
}

// buildScoreDetailMap 构建当前请求的逐商品评分明细索引。
func buildScoreDetailMap(sourceContext map[string]any) map[int64]recommendcore.ScoreDetail {
	scoreDetailMap := make(map[int64]recommendcore.ScoreDetail)
	// explain 明细存在时，先收敛成本次请求的商品评分索引。
	if sourceContext == nil {
		return scoreDetailMap
	}
	scoreDetails, ok := sourceContext["returnedScoreDetails"].([]recommendcore.ScoreDetail)
	// explain 是当前请求可复用的逐商品排序解释时，才继续收敛成索引。
	if !ok {
		return scoreDetailMap
	}
	for _, item := range scoreDetails {
		// 商品编号非法的 explain 明细直接忽略，避免污染后续逐商品映射。
		if item.GoodsId <= 0 {
			continue
		}
		scoreDetailMap[item.GoodsId] = item
	}
	return scoreDetailMap
}

// buildRecallSourceJSON 构建逐商品明细需要落库的召回来源 JSON。
func buildRecallSourceJSON(defaultRecallSources []string, scoreDetail recommendcore.ScoreDetail, hasScoreDetail bool) string {
	itemRecallSources := defaultRecallSources
	// 单商品 explain 存在时，优先落库该商品自己的召回来源。
	if hasScoreDetail && len(scoreDetail.RecallSources) > 0 {
		itemRecallSources = scoreDetail.RecallSources
	}
	recallSourceJson, err := json.Marshal(itemRecallSources)
	// 召回来源序列化理论上不会失败，失败时回退为空数组，避免影响主流程。
	if err != nil {
		return "[]"
	}
	return string(recallSourceJson)
}
