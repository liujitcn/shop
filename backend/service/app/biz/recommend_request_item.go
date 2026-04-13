package biz

import (
	"context"
	"encoding/json"
	"errors"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendcore "shop/pkg/recommend/core"

	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// RecommendRequestItemCase 推荐请求逐商品明细业务处理对象。
type RecommendRequestItemCase struct {
	*biz.BaseCase
	*data.RecommendRequestItemRepo
	recommendRequestRepo *data.RecommendRequestRepo
}

// NewRecommendRequestItemCase 创建推荐请求逐商品明细业务处理对象。
func NewRecommendRequestItemCase(
	baseCase *biz.BaseCase,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
) *RecommendRequestItemCase {
	return &RecommendRequestItemCase{
		BaseCase:                 baseCase,
		RecommendRequestItemRepo: recommendRequestItemRepo,
		recommendRequestRepo:     recommendRequestRepo,
	}
}

// batchCreateByRecommendRequest 根据推荐结果批量写入请求逐商品明细。
func (c *RecommendRequestItemCase) batchCreateByRecommendRequest(
	ctx context.Context,
	recommendRequestId int64,
	req *app.RecommendGoodsRequest,
	sourceContext map[string]any,
	list []*app.GoodsInfo,
	recallSources []string,
) error {
	// 主表编号、请求或返回结果缺失时，不生成逐商品明细。
	if recommendRequestId <= 0 || req == nil || len(list) == 0 {
		return nil
	}

	scoreDetailMap := make(map[int64]recommendcore.ScoreDetail)
	// explain 明细存在时，先收敛成本次请求的商品评分索引。
	if sourceContext != nil {
		scoreDetails, ok := sourceContext["returnedScoreDetails"].([]recommendcore.ScoreDetail)
		// explain 是当前请求可复用的逐商品排序解释时，才继续收敛成索引。
		if ok {
			for _, item := range scoreDetails {
				// 商品编号非法的 explain 明细直接忽略，避免污染后续逐商品映射。
				if item.GoodsId <= 0 {
					continue
				}
				scoreDetailMap[item.GoodsId] = item
			}
		}
	}

	positionBase := (req.GetPageNum() - 1) * req.GetPageSize()
	requestItemList := make([]*models.RecommendRequestItem, 0, len(list))
	for index, item := range list {
		// 非法商品结果直接跳过，避免脏数据写入逐商品明细表。
		if item == nil || item.GetId() <= 0 {
			continue
		}

		scoreDetail, ok := scoreDetailMap[item.GetId()]
		itemRecallSources := recallSources
		// 单商品 explain 存在时，优先落库该商品自己的召回来源。
		if ok && len(scoreDetail.RecallSources) > 0 {
			itemRecallSources = scoreDetail.RecallSources
		}
		recallSourceJson, err := json.Marshal(itemRecallSources)
		// 召回来源序列化理论上不会失败，失败时回退为空数组，避免影响主流程。
		if err != nil {
			recallSourceJson = []byte("[]")
		}

		requestItemList = append(requestItemList, &models.RecommendRequestItem{
			RecommendRequestID:    recommendRequestId,
			GoodsID:               item.GetId(),
			Position:              int32(positionBase + int64(index)),
			RecallSource:          string(recallSourceJson),
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
	// 当前页没有有效逐商品明细时，只保留主请求记录。
	if len(requestItemList) == 0 {
		return nil
	}
	return c.RecommendRequestItemRepo.BatchCreate(ctx, requestItemList)
}

// listRelatedGoodsIdsByRequestId 读取推荐请求中与当前商品共同出现的其他商品。
func (c *RecommendRequestItemCase) listRelatedGoodsIdsByRequestId(ctx context.Context, requestId string, goodsId int64) ([]int64, error) {
	// 请求编号为空时，不需要继续回查逐商品明细。
	if requestId == "" {
		return []int64{}, nil
	}

	recommendRequestQuery := c.recommendRequestRepo.Query(ctx).RecommendRequest
	requestOpts := make([]repo.QueryOption, 0, 1)
	requestOpts = append(requestOpts, repo.Where(recommendRequestQuery.RequestID.Eq(requestId)))
	requestEntity, err := c.recommendRequestRepo.Find(ctx, requestOpts...)
	// 请求主表查询失败时，仅对“未找到”场景回退为空结果。
	if err != nil {
		// 请求主表不存在时，说明当前行为无法回查推荐链路。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []int64{}, nil
		}
		return nil, err
	}

	recommendRequestItemQuery := c.Query(ctx).RecommendRequestItem
	requestItemOpts := make([]repo.QueryOption, 0, 1)
	requestItemOpts = append(requestItemOpts, repo.Where(recommendRequestItemQuery.RecommendRequestID.Eq(requestEntity.ID)))
	requestItemList, err := c.List(ctx, requestItemOpts...)
	if err != nil {
		return nil, err
	}

	relatedGoodsIds := make([]int64, 0, len(requestItemList))
	for _, item := range requestItemList {
		// 非法商品或当前商品自身都不参与关联商品集合。
		if item.GoodsID <= 0 || item.GoodsID == goodsId {
			continue
		}
		relatedGoodsIds = append(relatedGoodsIds, item.GoodsID)
	}
	return recommendcore.DedupeInt64s(relatedGoodsIds), nil
}

// loadPositionMapByRequestId 加载推荐请求中的商品位次映射。
func (c *RecommendRequestItemCase) loadPositionMapByRequestId(ctx context.Context, requestId string, goodsIds []int64) (map[int64]int32, error) {
	positionMap := make(map[int64]int32, len(goodsIds))
	// 请求编号或商品列表为空时，说明没有可复用的请求位次。
	if requestId == "" || len(goodsIds) == 0 {
		return positionMap, nil
	}

	recommendRequestQuery := c.recommendRequestRepo.Query(ctx).RecommendRequest
	requestOpts := make([]repo.QueryOption, 0, 1)
	requestOpts = append(requestOpts, repo.Where(recommendRequestQuery.RequestID.Eq(requestId)))
	requestEntity, err := c.recommendRequestRepo.Find(ctx, requestOpts...)
	// 推荐请求不存在时，说明曝光无法回查到请求链路，直接退回空映射。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return positionMap, nil
	}
	if err != nil {
		return nil, err
	}

	recommendRequestItemQuery := c.Query(ctx).RecommendRequestItem
	requestItemOpts := make([]repo.QueryOption, 0, 2)
	requestItemOpts = append(requestItemOpts, repo.Where(recommendRequestItemQuery.RecommendRequestID.Eq(requestEntity.ID)))
	requestItemOpts = append(requestItemOpts, repo.Where(recommendRequestItemQuery.GoodsID.In(goodsIds...)))
	requestItemList, err := c.List(ctx, requestItemOpts...)
	if err != nil {
		return nil, err
	}

	for _, item := range requestItemList {
		// 非法商品位次明细直接跳过，避免污染曝光位次映射。
		if item.GoodsID <= 0 {
			continue
		}
		positionMap[item.GoodsID] = item.Position
	}
	return positionMap, nil
}
