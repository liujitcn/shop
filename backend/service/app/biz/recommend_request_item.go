package biz

import (
	"context"
	"errors"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendOnlineRecord "shop/pkg/recommend/online/record"

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
	requestItemList := recommendOnlineRecord.BuildRecommendRequestItems(
		recommendRequestId,
		req.GetPageNum(),
		req.GetPageSize(),
		sourceContext,
		list,
		recallSources,
	)
	// 当前页没有有效逐商品明细时，只保留主请求记录。
	if len(requestItemList) == 0 {
		return nil
	}
	return c.RecommendRequestItemRepo.BatchCreate(ctx, requestItemList)
}

// listRelatedGoodsIdsByRequestId 读取推荐请求中与当前商品共同出现的其他商品。
func (c *RecommendRequestItemCase) listRelatedGoodsIdsByRequestId(ctx context.Context, requestId string, goodsId int64) ([]int64, error) {
	requestItemList, err := c.listRecommendRequestItemsByRequestId(ctx, requestId, nil)
	if err != nil {
		return nil, err
	}
	return recommendOnlineRecord.BuildRelatedGoodsIds(requestItemList, goodsId), nil
}

// loadPositionMapByRequestId 加载推荐请求中的商品位次映射。
func (c *RecommendRequestItemCase) loadPositionMapByRequestId(ctx context.Context, requestId string, goodsIds []int64) (map[int64]int32, error) {
	positionMap := make(map[int64]int32, len(goodsIds))
	// 请求编号或商品列表为空时，说明没有可复用的请求位次。
	if requestId == "" || len(goodsIds) == 0 {
		return positionMap, nil
	}
	requestItemList, err := c.listRecommendRequestItemsByRequestId(ctx, requestId, goodsIds)
	if err != nil {
		return nil, err
	}
	return recommendOnlineRecord.BuildPositionMap(requestItemList, goodsIds), nil
}

// findRecommendRequestEntityByRequestId 按请求编号查询推荐请求主表。
func (c *RecommendRequestItemCase) findRecommendRequestEntityByRequestId(ctx context.Context, requestId string) (*models.RecommendRequest, error) {
	recommendRequestQuery := c.recommendRequestRepo.Query(ctx).RecommendRequest
	requestOpts := make([]repo.QueryOption, 0, 1)
	requestOpts = append(requestOpts, repo.Where(recommendRequestQuery.RequestID.Eq(requestId)))
	requestEntity, err := c.recommendRequestRepo.Find(ctx, requestOpts...)
	// 推荐请求不存在时，直接回退为空记录，避免影响回查主流程。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return requestEntity, nil
}

// listRecommendRequestItemsByRequestId 按请求编号查询推荐请求逐商品明细。
func (c *RecommendRequestItemCase) listRecommendRequestItemsByRequestId(ctx context.Context, requestId string, goodsIds []int64) ([]*models.RecommendRequestItem, error) {
	// 请求编号为空时，不需要继续回查推荐链路。
	if requestId == "" {
		return []*models.RecommendRequestItem{}, nil
	}
	requestEntity, err := c.findRecommendRequestEntityByRequestId(ctx, requestId)
	if err != nil {
		return nil, err
	}
	// 推荐请求不存在时，说明当前链路没有可复用的逐商品明细。
	if requestEntity == nil {
		return []*models.RecommendRequestItem{}, nil
	}
	return c.listRecommendRequestItems(ctx, requestEntity.ID, goodsIds)
}

// listRecommendRequestItems 按请求主表编号查询推荐请求逐商品明细。
func (c *RecommendRequestItemCase) listRecommendRequestItems(ctx context.Context, recommendRequestId int64, goodsIds []int64) ([]*models.RecommendRequestItem, error) {
	recommendRequestItemQuery := c.Query(ctx).RecommendRequestItem
	requestItemOpts := make([]repo.QueryOption, 0, 2)
	requestItemOpts = append(requestItemOpts, repo.Where(recommendRequestItemQuery.RecommendRequestID.Eq(recommendRequestId)))
	// 当前存在商品编号过滤条件时，只回查指定商品的逐商品明细。
	if len(goodsIds) > 0 {
		requestItemOpts = append(requestItemOpts, repo.Where(recommendRequestItemQuery.GoodsID.In(goodsIds...)))
	}
	return c.List(ctx, requestItemOpts...)
}
