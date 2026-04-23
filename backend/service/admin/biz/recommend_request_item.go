package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRequestItemCase 推荐请求商品管理业务实例。
type RecommendRequestItemCase struct {
	*biz.BaseCase
	*data.RecommendRequestItemRepo
	goodsInfoRepo      *data.GoodsInfoRepo
	recommendEventCase *RecommendEventCase
}

// NewRecommendRequestItemCase 创建推荐请求商品管理业务实例。
func NewRecommendRequestItemCase(
	baseCase *biz.BaseCase,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	recommendEventCase *RecommendEventCase,
) *RecommendRequestItemCase {
	return &RecommendRequestItemCase{
		BaseCase:                 baseCase,
		RecommendRequestItemRepo: recommendRequestItemRepo,
		goodsInfoRepo:            goodsInfoRepo,
		recommendEventCase:       recommendEventCase,
	}
}

// ListRecommendRequestItems 查询当前请求页的推荐商品列表。
func (c *RecommendRequestItemCase) ListRecommendRequestItems(
	ctx context.Context,
	requestModel *models.RecommendRequest,
) ([]*admin.RecommendRequestItem, error) {
	resList := make([]*admin.RecommendRequestItem, 0)
	// 请求实体为空时，不存在可查询的推荐商品列表。
	if requestModel == nil {
		return resList, nil
	}

	startPosition, endPosition := c.resolveRequestPositionRange(requestModel)
	// 当前请求页没有有效位置区间时，不继续查询推荐商品列表。
	if endPosition <= startPosition {
		return resList, nil
	}

	query := c.RecommendRequestItemRepo.Query(ctx).RecommendRequestItem
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.Position.Asc()))
	opts = append(opts, repo.Order(query.ID.Asc()))
	opts = append(opts, repo.Where(query.RequestID.Eq(requestModel.RequestID)))
	opts = append(opts, repo.Where(query.Position.Gte(startPosition)))
	opts = append(opts, repo.Where(query.Position.Lt(endPosition)))
	itemList, err := c.RecommendRequestItemRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// 当前请求页没有推荐商品时，直接返回空列表。
	if len(itemList) == 0 {
		return resList, nil
	}

	goodsIds := make([]int64, 0, len(itemList))
	positionList := make([]int32, 0, len(itemList))
	for _, item := range itemList {
		// 请求商品为空或商品编号非法时，直接跳过无效明细。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsID)
		positionList = append(positionList, item.Position)
	}

	goodsMap, err := c.getGoodsInfoMap(ctx, goodsIds)
	if err != nil {
		return nil, err
	}
	eventCountMap, err := c.recommendEventCase.getRecommendEventCountMap(ctx, requestModel.RequestID, goodsIds, positionList)
	if err != nil {
		return nil, err
	}

	for _, item := range itemList {
		// 请求商品为空或商品编号非法时，直接跳过无效明细。
		if item == nil || item.GoodsID <= 0 {
			continue
		}

		goodsInfo := goodsMap[item.GoodsID]
		requestItem := &admin.RecommendRequestItem{
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
			requestItem.GoodsStatus = common.GoodsStatus(goodsInfo.Status)
		}
		resList = append(resList, requestItem)
	}
	return resList, nil
}

// resolveRequestPositionRange 计算当前请求页的结果位置区间。
func (c *RecommendRequestItemCase) resolveRequestPositionRange(requestModel *models.RecommendRequest) (int32, int32) {
	// 请求实体为空时，不存在可用的位置区间。
	if requestModel == nil {
		return 0, 0
	}

	pageNum := requestModel.PageNum
	pageSize := requestModel.PageSize
	// 页码非法时，统一回退到第一页。
	if pageNum <= 0 {
		pageNum = 1
	}
	// 分页大小非法时，不继续计算结果位置区间。
	if pageSize <= 0 {
		return 0, 0
	}
	startPosition := (pageNum - 1) * pageSize
	return startPosition, startPosition + pageSize
}

// getGoodsInfoMap 构建商品信息映射。
func (c *RecommendRequestItemCase) getGoodsInfoMap(ctx context.Context, goodsIds []int64) (map[int64]*models.GoodsInfo, error) {
	goodsMap := make(map[int64]*models.GoodsInfo)
	// 商品编号列表为空时，无需继续查询商品信息。
	if len(goodsIds) == 0 {
		return goodsMap, nil
	}

	query := c.goodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoRepo.List(ctx, repo.Where(query.ID.In(goodsIds...)))
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
