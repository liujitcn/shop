package biz

import (
	"context"
	"encoding/json"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/util"

	"github.com/google/uuid"
	"github.com/liujitcn/gorm-kit/repo"
	auth "github.com/liujitcn/kratos-kit/auth"
)

const recommendStrategyVersion = "v1"

// RecommendCase 推荐业务处理对象。
type RecommendCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepo
	goodsInfoRepo     *data.GoodsInfoRepo
	orderGoodsRepo    *data.OrderGoodsRepo
	userCartRepo      *data.UserCartRepo
	recommendProfile  *RecommendProfileCase
	recommendRelation *RecommendRelationCase
}

// NewRecommendCase 创建推荐业务处理对象。
func NewRecommendCase(
	baseCase *biz.BaseCase,
	recommendRequestRepo *data.RecommendRequestRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
	userCartRepo *data.UserCartRepo,
	recommendProfile *RecommendProfileCase,
	recommendRelation *RecommendRelationCase,
) *RecommendCase {
	return &RecommendCase{
		BaseCase:             baseCase,
		RecommendRequestRepo: recommendRequestRepo,
		goodsInfoRepo:        goodsInfoRepo,
		orderGoodsRepo:       orderGoodsRepo,
		userCartRepo:         userCartRepo,
		recommendProfile:     recommendProfile,
		recommendRelation:    recommendRelation,
	}
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendCase) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	// 统一兜底分页参数，避免前端漏传导致查询异常。
	pageNum := req.GetPageNum()
	if pageNum <= 0 {
		pageNum = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 10
	}
	// 每次推荐请求都生成独立 requestID，用于后续曝光归因。
	requestID := uuid.NewString()
	userID := c.getRecommendUserID(ctx)
	sceneGoodsIds, sceneCategoryIds, sourceContext, recallSources, err := c.resolveSceneContext(ctx, req, userID, int(pageSize))
	if err != nil {
		return nil, err
	}
	var list []*app.GoodsInfo
	var total int64
	list, total, err = c.listRecommendGoods(ctx, sceneGoodsIds, sceneCategoryIds, pageNum, pageSize)
	if err != nil {
		return nil, err
	}
	err = c.saveRecommendRequest(ctx, requestID, userID, req, sourceContext, list, recallSources)
	if err != nil {
		return nil, err
	}
	return &app.RecommendGoodsResponse{
		List:      list,
		Total:     int32(total),
		RequestId: requestID,
	}, nil
}

// RecommendExposure 记录推荐曝光。
func (c *RecommendCase) RecommendExposure(ctx context.Context, req *app.RecommendExposureRequest) error {
	// 第一版曝光固定按整组推荐位一次记录，入队异步落库。
	publishRecommendExposureEvent(c.getRecommendUserID(ctx), req.GetRequestId(), req.GetScene().String(), req.GetGoodsIds())
	return nil
}

// 解析不同推荐场景下的上下文信息。
func (c *RecommendCase) resolveSceneContext(ctx context.Context, req *app.RecommendGoodsRequest, userID int64, limit int) ([]int64, []int64, map[string]any, []string, error) {
	relationGoodsIds := make([]int64, 0)
	categoryIds := make([]int64, 0)
	recallSources := make([]string, 0, 3)
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	switch req.GetScene() {
	case app.RecommendScene_CART:
		cartGoodsIds, err := c.listCurrentUserCartGoodsIds(ctx, userID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		sourceContext["cartGoodsIds"] = cartGoodsIds
		if len(cartGoodsIds) > 0 {
			var err error
			// 购物车场景优先取购物车商品的关联商品。
			relationGoodsIds, err = c.recommendRelation.ListRelatedGoodsIds(ctx, cartGoodsIds, limit)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			// 关联商品不足时，再用购物车商品所属类目补足候选集。
			categoryIds, err = c.listCategoryIdsByGoodsIds(ctx, cartGoodsIds)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			recallSources = append(recallSources, "cart")
		}
	case app.RecommendScene_ORDER_DETAIL, app.RecommendScene_ORDER_PAID:
		if req.GetOrderId() > 0 {
			orderGoodsIds, err := c.listOrderGoodsIds(ctx, req.GetOrderId())
			if err != nil {
				return nil, nil, nil, nil, err
			}
			// 订单详情和支付成功都优先基于订单商品做强关联召回。
			relationGoodsIds, err = c.recommendRelation.ListRelatedGoodsIds(ctx, orderGoodsIds, limit)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			categoryIds, err = c.listCategoryIdsByGoodsIds(ctx, orderGoodsIds)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			recallSources = append(recallSources, "order")
		}
	}
	profileCategoryIds, err := c.recommendProfile.ListPreferredCategoryIds(ctx, userID, 3)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if len(profileCategoryIds) > 0 {
		// 用户画像作为补充召回来源，不直接覆盖场景召回结果。
		categoryIds = append(categoryIds, profileCategoryIds...)
		recallSources = append(recallSources, "profile")
	}
	if len(recallSources) == 0 {
		// 没有任何场景或画像数据时，退化到最新商品兜底。
		recallSources = append(recallSources, "latest")
	}
	return dedupeInt64s(relationGoodsIds), dedupeInt64s(categoryIds), sourceContext, dedupeStrings(recallSources), nil
}

// 查询当前用户购物车中的商品ID列表。
func (c *RecommendCase) listCurrentUserCartGoodsIds(ctx context.Context, userID int64) ([]int64, error) {
	if userID == 0 {
		return []int64{}, nil
	}
	userCartQuery := c.userCartRepo.Query(ctx).UserCart
	list, err := c.userCartRepo.List(ctx,
		repo.Where(userCartQuery.UserID.Eq(userID)),
	)
	if err != nil {
		return nil, err
	}
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return dedupeInt64s(goodsIds), nil
}

// 查询订单中的商品ID列表。
func (c *RecommendCase) listOrderGoodsIds(ctx context.Context, orderID int64) ([]int64, error) {
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	list, err := c.orderGoodsRepo.List(ctx,
		repo.Where(orderGoodsQuery.OrderID.Eq(orderID)),
	)
	if err != nil {
		return nil, err
	}
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return dedupeInt64s(goodsIds), nil
}

// 根据商品ID列表查询分类ID列表。
func (c *RecommendCase) listCategoryIdsByGoodsIds(ctx context.Context, goodsIds []int64) ([]int64, error) {
	if len(goodsIds) == 0 {
		return []int64{}, nil
	}
	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoRepo.List(ctx,
		repo.Where(goodsQuery.ID.In(goodsIds...)),
	)
	if err != nil {
		return nil, err
	}
	categoryIds := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIds = append(categoryIds, item.CategoryID)
	}
	return dedupeInt64s(categoryIds), nil
}

// 查询推荐商品列表，优先使用关联商品，其次使用分类召回，最后回退到最新商品。
func (c *RecommendCase) listRecommendGoods(ctx context.Context, priorityGoodsIds []int64, categoryIds []int64, pageNum, pageSize int64) ([]*app.GoodsInfo, int64, error) {
	member := util.IsMember(ctx)
	list := make([]*app.GoodsInfo, 0, pageSize)
	selectedGoodsIds := make([]int64, 0, pageSize)
	var err error
	if pageNum == 1 && len(priorityGoodsIds) > 0 {
		var priorityList []*models.GoodsInfo
		priorityList, err = c.listGoodsByIds(ctx, priorityGoodsIds)
		if err != nil {
			return nil, 0, err
		}
		// 首屏优先消化场景关联商品，保证推荐语义稳定。
		for _, item := range priorityList {
			if len(list) >= int(pageSize) {
				break
			}
			selectedGoodsIds = append(selectedGoodsIds, item.ID)
			list = append(list, c.convertGoodsToProto(item, member))
		}
	}
	if len(list) < int(pageSize) {
		remainSize := pageSize - int64(len(list))
		var queryList []*models.GoodsInfo
		var total int64
		// 场景商品不足时，使用类目和最新商品继续补齐分页结果。
		queryList, total, err = c.pageGoods(ctx, categoryIds, selectedGoodsIds, pageNum, remainSize)
		if err != nil {
			return nil, 0, err
		}
		for _, item := range queryList {
			selectedGoodsIds = append(selectedGoodsIds, item.ID)
			list = append(list, c.convertGoodsToProto(item, member))
		}
		return list, total + int64(len(priorityGoodsIds)), nil
	}
	return list, int64(len(list)), nil
}

// 按商品ID顺序查询商品信息。
func (c *RecommendCase) listGoodsByIds(ctx context.Context, goodsIds []int64) ([]*models.GoodsInfo, error) {
	if len(goodsIds) == 0 {
		return []*models.GoodsInfo{}, nil
	}
	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoRepo.List(ctx,
		repo.Where(goodsQuery.ID.In(goodsIds...)),
		repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))),
		repo.Order(goodsQuery.CreatedAt.Desc()),
	)
	if err != nil {
		return nil, err
	}
	goodsMap := make(map[int64]*models.GoodsInfo, len(list))
	for _, item := range list {
		goodsMap[item.ID] = item
	}
	// 数据库 IN 查询不保证原顺序，这里按输入顺序重新组装结果。
	result := make([]*models.GoodsInfo, 0, len(goodsIds))
	for _, goodsID := range goodsIds {
		item, ok := goodsMap[goodsID]
		if !ok {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

// 分页查询推荐商品。
func (c *RecommendCase) pageGoods(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, pageNum, pageSize int64) ([]*models.GoodsInfo, int64, error) {
	if pageSize <= 0 {
		return []*models.GoodsInfo{}, 0, nil
	}
	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(goodsQuery.CreatedAt.Desc()))
	if len(categoryIds) > 0 {
		opts = append(opts, repo.Where(goodsQuery.CategoryID.In(categoryIds...)))
	}
	if len(excludeGoodsIds) > 0 {
		// 已被优先命中的商品不再重复进入补足列表。
		opts = append(opts, repo.Where(goodsQuery.ID.NotIn(excludeGoodsIds...)))
	}
	return c.goodsInfoRepo.Page(ctx, pageNum, pageSize, opts...)
}

// 保存推荐请求记录。
func (c *RecommendCase) saveRecommendRequest(ctx context.Context, requestID string, userID int64, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	sourceContextJSON, err := json.Marshal(sourceContext)
	if err != nil {
		return err
	}
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GetId())
	}
	goodsIdsJSON, err := json.Marshal(goodsIds)
	if err != nil {
		return err
	}
	recallSourcesJSON, err := json.Marshal(recallSources)
	if err != nil {
		return err
	}
	// 推荐请求表保存的是本次实际下发结果，供曝光与点击链路统一回查。
	entity := &models.RecommendRequest{
		RequestID:         requestID,
		UserID:            userID,
		Scene:             req.GetScene().String(),
		SourceContextJSON: string(sourceContextJSON),
		PageNum:           int32(req.GetPageNum()),
		PageSize:          int32(req.GetPageSize()),
		GoodsIdsJSON:      string(goodsIdsJSON),
		StrategyVersion:   recommendStrategyVersion,
		RecallSourcesJSON: string(recallSourcesJSON),
	}
	return c.RecommendRequestRepo.Create(ctx, entity)
}

// 将商品模型转换为推荐商品响应。
func (c *RecommendCase) convertGoodsToProto(item *models.GoodsInfo, member bool) *app.GoodsInfo {
	price := item.Price
	if member {
		price = item.DiscountPrice
	}
	return &app.GoodsInfo{
		Id:      item.ID,
		Name:    item.Name,
		Desc:    item.Desc,
		Picture: item.Picture,
		SaleNum: item.InitSaleNum + item.RealSaleNum,
		Price:   price,
	}
}

// 获取推荐场景下的用户ID。
func (c *RecommendCase) getRecommendUserID(ctx context.Context) int64 {
	return getRecommendUserID(ctx)
}

// 获取推荐场景下的用户ID。
func getRecommendUserID(ctx context.Context) int64 {
	authInfo, err := auth.FromContext(ctx)
	if err != nil || authInfo == nil {
		return 0
	}
	return authInfo.UserId
}

// 去重整型切片。
func dedupeInt64s(values []int64) []int64 {
	result := make([]int64, 0, len(values))
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// 去重字符串切片。
func dedupeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
