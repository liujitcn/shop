package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
)

// RecommendRequestItemCase 推荐请求商品管理业务实例。
type RecommendRequestItemCase struct {
	*biz.BaseCase
	*data.RecommendRequestItemRepository
	goodsInfoRepo      *data.GoodsInfoRepository
	recommendEventCase *RecommendEventCase
}

// NewRecommendRequestItemCase 创建推荐请求商品管理业务实例。
func NewRecommendRequestItemCase(
	baseCase *biz.BaseCase,
	recommendRequestItemRepo *data.RecommendRequestItemRepository,
	goodsInfoRepo *data.GoodsInfoRepository,
	recommendEventCase *RecommendEventCase,
) *RecommendRequestItemCase {
	return &RecommendRequestItemCase{
		BaseCase:                       baseCase,
		RecommendRequestItemRepository: recommendRequestItemRepo,
		goodsInfoRepo:                  goodsInfoRepo,
		recommendEventCase:             recommendEventCase,
	}
}

// ListRecommendRequestItems 查询当前请求页的推荐商品列表。
func (c *RecommendRequestItemCase) ListRecommendRequestItems(
	ctx context.Context,
	requestModel *models.RecommendRequest,
) ([]*adminv1.RecommendRequestItem, error) {
	resList := make([]*adminv1.RecommendRequestItem, 0)
	// 请求实体为空时，不存在可查询的推荐商品列表。
	if requestModel == nil {
		return resList, nil
	}

	pageNum, pageSize := repository.PageDefault(int64(requestModel.PageNum), int64(requestModel.PageSize))
	startPosition := int32((pageNum - 1) * pageSize)
	endPosition := startPosition + int32(pageSize)
	// 当前请求页没有有效位置区间时，不继续查询推荐商品列表。
	if endPosition <= startPosition {
		return resList, nil
	}

	query := c.RecommendRequestItemRepository.Query(ctx).RecommendRequestItem
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Position.Asc()))
	opts = append(opts, repository.Order(query.ID.Asc()))
	opts = append(opts, repository.Where(query.RequestID.Eq(requestModel.RequestID)))
	opts = append(opts, repository.Where(query.Position.Gte(startPosition)))
	opts = append(opts, repository.Where(query.Position.Lt(endPosition)))
	itemList, err := c.RecommendRequestItemRepository.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// 当前请求页没有推荐商品时，直接返回空列表。
	if len(itemList) == 0 {
		return resList, nil
	}

	goodsIDs := make([]int64, 0, len(itemList))
	positionList := make([]int32, 0, len(itemList))
	for _, item := range itemList {
		// 请求商品为空或商品编号非法时，直接跳过无效明细。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		goodsIDs = append(goodsIDs, item.GoodsID)
		positionList = append(positionList, item.Position)
	}

	goodsMap := make(map[int64]*models.GoodsInfo)
	goodsMap, err = c.getGoodsInfoMap(ctx, goodsIDs)
	if err != nil {
		return nil, err
	}
	eventCountMap := make(map[string]int64)
	eventCountMap, err = c.recommendEventCase.getRecommendEventCountMap(ctx, requestModel.RequestID, goodsIDs, positionList)
	if err != nil {
		return nil, err
	}

	for _, item := range itemList {
		// 请求商品为空或商品编号非法时，直接跳过无效明细。
		if item == nil || item.GoodsID <= 0 {
			continue
		}

		goodsInfo := goodsMap[item.GoodsID]
		requestItem := &adminv1.RecommendRequestItem{
			GoodsId:    item.GoodsID,
			Position:   item.Position,
			EventCount: eventCountMap[c.recommendEventCase.buildRecommendItemEventKey(item.GoodsID, item.Position)],
		}
		// 命中本地商品快照时，补齐商品基础展示信息。
		if goodsInfo != nil {
			requestItem.GoodsName = goodsInfo.Name
			requestItem.Picture = goodsInfo.Picture
			requestItem.Price = goodsInfo.Price
			requestItem.DiscountPrice = goodsInfo.DiscountPrice
			requestItem.GoodsStatus = commonv1.GoodsStatus(goodsInfo.Status)
		}
		resList = append(resList, requestItem)
	}
	return resList, nil
}

// getGoodsInfoMap 构建商品信息映射。
func (c *RecommendRequestItemCase) getGoodsInfoMap(ctx context.Context, goodsIDs []int64) (map[int64]*models.GoodsInfo, error) {
	goodsMap := make(map[int64]*models.GoodsInfo)
	// 商品编号列表为空时，无需继续查询商品信息。
	if len(goodsIDs) == 0 {
		return goodsMap, nil
	}

	query := c.goodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoRepo.List(ctx, repository.Where(query.ID.In(goodsIDs...)))
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		// 商品实体为空或商品编号非法时，直接跳过无效快照。
		if item == nil || item.ID <= 0 {
			continue
		}
		goodsMap[item.ID] = item
	}
	return goodsMap, nil
}
