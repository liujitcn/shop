package biz

import (
	"context"
	"time"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	appdto "shop/service/app/dto"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendExposureItemCase 推荐曝光逐商品明细业务处理对象。
type RecommendExposureItemCase struct {
	*biz.BaseCase
	*data.RecommendExposureItemRepo
	recommendExposureRepo    *data.RecommendExposureRepo
	recommendRequestItemCase *RecommendRequestItemCase
}

// NewRecommendExposureItemCase 创建推荐曝光逐商品明细业务处理对象。
func NewRecommendExposureItemCase(
	baseCase *biz.BaseCase,
	recommendExposureItemRepo *data.RecommendExposureItemRepo,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendRequestItemCase *RecommendRequestItemCase,
) *RecommendExposureItemCase {
	return &RecommendExposureItemCase{
		BaseCase:                  baseCase,
		RecommendExposureItemRepo: recommendExposureItemRepo,
		recommendExposureRepo:     recommendExposureRepo,
		recommendRequestItemCase:  recommendRequestItemCase,
	}
}

// batchCreateByRecommendExposure 根据曝光结果批量写入逐商品明细。
func (c *RecommendExposureItemCase) batchCreateByRecommendExposure(ctx context.Context, recommendExposureId int64, requestId string, goodsIds []int64) error {
	// 主表编号或曝光商品缺失时，不生成逐商品明细。
	if recommendExposureId <= 0 || len(goodsIds) == 0 {
		return nil
	}

	positionMap, err := c.recommendRequestItemCase.loadPositionMapByRequestId(ctx, requestId, goodsIds)
	if err != nil {
		return err
	}

	seenGoodsIds := make(map[int64]struct{}, len(goodsIds))
	exposureItemList := make([]*models.RecommendExposureItem, 0, len(goodsIds))
	for index, goodsId := range goodsIds {
		// 非法商品直接跳过，避免曝光明细写入脏数据。
		if goodsId <= 0 {
			continue
		}
		_, ok := seenGoodsIds[goodsId]
		// 同一批曝光中重复商品只保留第一次，避免统计口径放大。
		if ok {
			continue
		}
		seenGoodsIds[goodsId] = struct{}{}

		position, ok := positionMap[goodsId]
		// 请求明细里没有该商品时，退回前端上报顺序作为曝光位次。
		if !ok {
			position = int32(index)
		}

		exposureItemList = append(exposureItemList, &models.RecommendExposureItem{
			RecommendExposureID: recommendExposureId,
			GoodsID:             goodsId,
			Position:            position,
		})
	}
	// 当前曝光没有有效商品明细时，只保留主曝光记录。
	if len(exposureItemList) == 0 {
		return nil
	}
	return c.RecommendExposureItemRepo.BatchCreate(ctx, exposureItemList)
}

// loadRecommendExposureCountMap 加载指定主体的商品曝光次数。
func (c *RecommendExposureItemCase) loadRecommendExposureCountMap(ctx context.Context, actor *appdto.RecommendActor, scene int32, cutoff time.Time, goodsIds []int64) (map[int64]int64, error) {
	recommendExposureQuery := c.recommendExposureRepo.Query(ctx).RecommendExposure
	exposureOpts := make([]repo.QueryOption, 0, 4)
	exposureOpts = append(exposureOpts, repo.Where(recommendExposureQuery.ActorType.Eq(actor.ActorType)))
	exposureOpts = append(exposureOpts, repo.Where(recommendExposureQuery.ActorID.Eq(actor.ActorId)))
	exposureOpts = append(exposureOpts, repo.Where(recommendExposureQuery.Scene.Eq(scene)))
	exposureOpts = append(exposureOpts, repo.Where(recommendExposureQuery.CreatedAt.Gte(cutoff)))
	exposureList, err := c.recommendExposureRepo.List(ctx, exposureOpts...)
	if err != nil {
		return nil, err
	}

	countMap := make(map[int64]int64, len(goodsIds))
	// 当前主体没有曝光主记录时，直接返回空统计。
	if len(exposureList) == 0 {
		return countMap, nil
	}

	exposureIds := make([]int64, 0, len(exposureList))
	for _, item := range exposureList {
		// 非法主记录直接跳过，避免污染逐商品明细查询条件。
		if item.ID <= 0 {
			continue
		}
		exposureIds = append(exposureIds, item.ID)
	}
	// 没有可用主表编号时，说明曝光主记录都不合法。
	if len(exposureIds) == 0 {
		return countMap, nil
	}

	recommendExposureItemQuery := c.Query(ctx).RecommendExposureItem
	exposureItemOpts := make([]repo.QueryOption, 0, 2)
	exposureItemOpts = append(exposureItemOpts, repo.Where(recommendExposureItemQuery.RecommendExposureID.In(exposureIds...)))
	exposureItemOpts = append(exposureItemOpts, repo.Where(recommendExposureItemQuery.GoodsID.In(goodsIds...)))
	exposureItemList, err := c.List(ctx, exposureItemOpts...)
	if err != nil {
		return nil, err
	}

	for _, item := range exposureItemList {
		// 非法商品不参与曝光惩罚统计。
		if item.GoodsID <= 0 {
			continue
		}
		countMap[item.GoodsID]++
	}
	return countMap, nil
}
