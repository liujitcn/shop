package biz

import (
	"context"
	"encoding/json"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendactor "shop/pkg/recommend/actor"
	recommendcandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
	recommendevent "shop/pkg/recommend/event"
	recommendexplain "shop/pkg/recommend/explain"
	"shop/service/app/util"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRequestCase 推荐请求子业务处理对象。
type RecommendRequestCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepo
	goodsInfoCase                    *GoodsInfoCase
	orderGoodsRepo                   *data.OrderGoodsRepo
	userCartRepo                     *data.UserCartRepo
	recommendExposureCase            *RecommendExposureCase
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase
	recommendUserPreferenceCase      *RecommendUserPreferenceCase
	recommendGoodsRelationCase       *RecommendGoodsRelationCase
	recommendGoodsStatDayCase        *RecommendGoodsStatDayCase
	goodsStatDayCase                 *GoodsStatDayCase
}

// NewRecommendRequestCase 创建推荐请求子业务处理对象。
func NewRecommendRequestCase(
	baseCase *biz.BaseCase,
	recommendRequestRepo *data.RecommendRequestRepo,
	goodsInfoCase *GoodsInfoCase,
	orderGoodsRepo *data.OrderGoodsRepo,
	userCartRepo *data.UserCartRepo,
	recommendExposureCase *RecommendExposureCase,
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase,
	recommendUserPreferenceCase *RecommendUserPreferenceCase,
	recommendGoodsRelationCase *RecommendGoodsRelationCase,
	recommendGoodsStatDayCase *RecommendGoodsStatDayCase,
	goodsStatDayCase *GoodsStatDayCase,
) *RecommendRequestCase {
	return &RecommendRequestCase{
		BaseCase:                         baseCase,
		RecommendRequestRepo:             recommendRequestRepo,
		goodsInfoCase:                    goodsInfoCase,
		orderGoodsRepo:                   orderGoodsRepo,
		userCartRepo:                     userCartRepo,
		recommendExposureCase:            recommendExposureCase,
		recommendUserGoodsPreferenceCase: recommendUserGoodsPreferenceCase,
		recommendUserPreferenceCase:      recommendUserPreferenceCase,
		recommendGoodsRelationCase:       recommendGoodsRelationCase,
		recommendGoodsStatDayCase:        recommendGoodsStatDayCase,
		goodsStatDayCase:                 goodsStatDayCase,
	}
}

// RecommendGoods 查询推荐商品列表。
func (c *RecommendRequestCase) RecommendGoods(ctx context.Context, req *app.RecommendGoodsRequest) (*app.RecommendGoodsResponse, error) {
	// 统一兜底分页参数，避免前端漏传导致查询异常。
	pageNum := req.GetPageNum()
	// 页码非法时回退到首页，保证分页查询始终可执行。
	if pageNum <= 0 {
		pageNum = 1
	}
	pageSize := req.GetPageSize()
	// 每页数量非法时使用默认值，避免查全表或空分页。
	if pageSize <= 0 {
		pageSize = 10
	}
	req.PageNum = pageNum
	req.PageSize = pageSize
	// 每次推荐请求都生成独立 requestID，用于后续曝光归因。
	requestId := id.NewShortUUID()
	actor := recommendactor.Resolve(ctx)

	list := make([]*app.GoodsInfo, 0)
	total := int64(0)
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	recallSources := make([]string, 0, 4)
	var err error
	// 匿名主体统一走公共推荐池，减少首页、购物车、我的三端内容分裂。
	if actor.ActorType == recommendevent.ActorTypeAnonymous {
		list, total, recallSources, sourceContext, err = c.listAnonymousRecommendGoods(ctx, actor, req, pageNum, pageSize)
	} else {
		sceneGoodsIds := make([]int64, 0)
		sceneCategoryIds := make([]int64, 0)
		sceneGoodsIds, sceneCategoryIds, sourceContext, recallSources, err = c.resolveSceneContext(ctx, req, actor.UserId, int(pageSize))
		if err == nil {
			list, total, recallSources, sourceContext, err = c.listRecommendGoods(ctx, actor, req, actor.UserId, sceneGoodsIds, sceneCategoryIds, pageNum, pageSize)
		}
	}
	if err != nil {
		return nil, err
	}
	sourceContext["actorType"] = actor.ActorType
	sourceContext["actorId"] = actor.ActorId

	err = c.saveRecommendRequest(ctx, requestId, actor, req, sourceContext, list, recallSources)
	if err != nil {
		return nil, err
	}

	return &app.RecommendGoodsResponse{
		List:      list,
		Total:     int32(total),
		RequestId: requestId,
	}, nil
}

// BindRecommendRequestActor 将匿名请求主体绑定为登录主体。
func (c *RecommendRequestCase) BindRecommendRequestActor(ctx context.Context, anonymousId, userId int64) error {
	recommendRequestQuery := c.RecommendRequestRepo.Data.Query(ctx).RecommendRequest
	_, err := recommendRequestQuery.WithContext(ctx).
		Where(
			recommendRequestQuery.ActorType.Eq(recommendevent.ActorTypeAnonymous),
			recommendRequestQuery.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendevent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// resolveSceneContext 解析不同推荐场景下的上下文信息。
func (c *RecommendRequestCase) resolveSceneContext(ctx context.Context, req *app.RecommendGoodsRequest, userId int64, limit int) ([]int64, []int64, map[string]any, []string, error) {
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	relationGoodsIds, categoryIds, recallSources, err := c.resolveSceneRecall(ctx, req, userId, sourceContext, limit)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	profileCategoryIds := make([]int64, 0)
	profileCategoryIds, err = c.recommendUserPreferenceCase.listPreferredCategoryIds(ctx, userId, 3)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	// 用户画像只作为补充召回来源，不与场景召回互斥。
	if len(profileCategoryIds) > 0 {
		// 用户画像作为补充召回来源，不直接覆盖场景召回结果。
		categoryIds = append(categoryIds, profileCategoryIds...)
		recallSources = append(recallSources, "profile")
	}
	// 当场景和画像都没有可用数据时，最终标记为最新商品兜底。
	if len(recallSources) == 0 {
		// 没有任何场景或画像数据时，退化到最新商品兜底。
		recallSources = append(recallSources, "latest")
	}

	return recommendcore.DedupeInt64s(relationGoodsIds), recommendcore.DedupeInt64s(categoryIds), sourceContext, recommendcore.DedupeStrings(recallSources), nil
}

// listRecommendGoods 查询推荐商品列表并执行统一排序。
func (c *RecommendRequestCase) listRecommendGoods(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	userId int64,
	priorityGoodsIds []int64,
	categoryIds []int64,
	pageNum, pageSize int64,
) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	// 分页数量非法时直接返回空结果，避免继续构造候选集。
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	candidateLimit := recommendcandidate.ResolveCandidateLimit(pageNum, pageSize)
	excludeGoodsIds := recommendcore.DedupeInt64s(priorityGoodsIds)
	categoryCandidateIds := make([]int64, 0)
	var err error
	categoryCandidateIds, err = c.listCategoryCandidateGoodsIds(ctx, categoryIds, excludeGoodsIds, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	excludeGoodsIds = recommendcore.DedupeInt64s(append(excludeGoodsIds, categoryCandidateIds...))
	latestCandidateIds := make([]int64, 0)
	latestCandidateIds, err = c.listLatestCandidateGoodsIds(ctx, excludeGoodsIds, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	allCandidateIds := recommendcore.DedupeInt64s(append(append(priorityGoodsIds, categoryCandidateIds...), latestCandidateIds...))
	// 没有候选商品时，直接返回空结果。
	if len(allCandidateIds) == 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	goodsList := make([]*models.GoodsInfo, 0)
	goodsList, err = c.listGoodsByIds(ctx, allCandidateIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidates := make(map[int64]*recommendcore.Candidate)
	candidates, err = c.buildRecommendCandidates(ctx, actor, req, userId, priorityGoodsIds, goodsList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	rankedGoods := recommendcandidate.RankGoods(candidates)
	total := int64(len(rankedGoods))

	offset := int((pageNum - 1) * pageSize)
	// 当前页超出候选范围时，返回空页但保留总数。
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{}, map[string]any{}, nil
	}
	end := offset + int(pageSize)
	// 分页结束位置超过候选集时，按最后一条候选截断。
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}

	member := util.IsMember(ctx)
	list := make([]*app.GoodsInfo, 0, end-offset)
	pageRecallSources := make(map[string]struct{}, 8)
	scoreDetails := make([]recommendcore.ScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		list = append(list, c.goodsInfoCase.convertToProto(item, member))
		candidate, ok := candidates[item.ID]
		// 候选明细缺失时，仅跳过解释信息，不影响商品返回。
		if !ok {
			continue
		}
		for source := range candidate.RecallSources {
			pageRecallSources[source] = struct{}{}
		}
		scoreDetails = append(scoreDetails, recommendexplain.BuildScoreDetail(candidate))
	}
	return list, total, recommendexplain.ListRecallSources(pageRecallSources), map[string]any{
		"candidateLimit":       candidateLimit,
		"priorityGoodsIds":     priorityGoodsIds,
		"categoryIds":          categoryIds,
		"returnedScoreDetails": scoreDetails,
	}, nil
}

// listAnonymousRecommendGoods 查询匿名推荐商品列表并执行统一排序。
func (c *RecommendRequestCase) listAnonymousRecommendGoods(ctx context.Context, actor *RecommendActor, req *app.RecommendGoodsRequest, pageNum, pageSize int64) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	// 分页数量非法时直接返回匿名空页。
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{"anonymous_hot"}, map[string]any{}, nil
	}

	candidateLimit := recommendcandidate.ResolveCandidateLimit(pageNum, pageSize)
	startDate := time.Now().AddDate(0, 0, -recommendcandidate.AnonymousRecallDays)
	sceneGoodsIds := make([]int64, 0)
	var err error
	sceneGoodsIds, err = c.recommendGoodsStatDayCase.listSceneHotGoodsIds(ctx, req.GetScene(), startDate, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidateGoodsIds := make([]int64, 0)
	candidateGoodsIds, err = c.goodsStatDayCase.mergeAnonymousGoodsIds(ctx, sceneGoodsIds, startDate, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	// 场景热度和全站热度都没有数据时，退回最新商品分页。
	if len(candidateGoodsIds) == 0 {
		fallbackList, fallbackTotal, fallbackErr := c.pageGoods(ctx, nil, nil, pageNum, pageSize)
		if fallbackErr != nil {
			return nil, 0, nil, nil, fallbackErr
		}
		return c.convertGoodsListToProto(ctx, fallbackList), fallbackTotal, []string{"latest"}, map[string]any{
			"candidateLimit":   candidateLimit,
			"sceneHotGoodsIds": sceneGoodsIds,
		}, nil
	}

	goodsList := make([]*models.GoodsInfo, 0)
	goodsList, err = c.listGoodsByIds(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidates := make(map[int64]*recommendcore.Candidate)
	candidates, err = c.buildAnonymousRecommendCandidates(ctx, actor, req, goodsList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	rankedGoods := recommendcandidate.RankGoods(candidates)
	total := int64(len(rankedGoods))
	offset := int((pageNum - 1) * pageSize)
	// 当前页超过候选集时，返回匿名空页。
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{"anonymous_hot"}, map[string]any{}, nil
	}
	end := offset + int(pageSize)
	// 分页结束位置越界时，按候选集最后一条裁剪。
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}

	pageRecallSources := make(map[string]struct{}, 6)
	scoreDetails := make([]recommendcore.ScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		candidate, ok := candidates[item.ID]
		// 候选解释缺失时仅跳过解释信息，不影响结果列表。
		if !ok {
			continue
		}
		for source := range candidate.RecallSources {
			pageRecallSources[source] = struct{}{}
		}
		scoreDetails = append(scoreDetails, recommendexplain.BuildScoreDetail(candidate))
	}
	return c.convertGoodsListToProto(ctx, rankedGoods[offset:end]), total, recommendexplain.ListRecallSources(pageRecallSources), map[string]any{
		"candidateLimit":             candidateLimit,
		"sceneHotGoodsIds":           sceneGoodsIds,
		"anonymousCandidateGoodsIds": candidateGoodsIds,
		"returnedScoreDetails":       scoreDetails,
	}, nil
}

// buildRecommendCandidates 构建登录态推荐候选集。
func (c *RecommendRequestCase) buildRecommendCandidates(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	userId int64,
	priorityGoodsIds []int64,
	goodsList []*models.GoodsInfo,
) (map[int64]*recommendcore.Candidate, error) {
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	categoryIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 非法商品不参与候选信号计算。
		if item == nil || item.ID <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.ID)
		categoryIds = append(categoryIds, item.CategoryID)
	}

	relationScores := make(map[int64]float64)
	var err error
	relationScores, err = c.recommendGoodsRelationCase.loadRelationScores(ctx, priorityGoodsIds)
	if err != nil {
		return nil, err
	}
	userGoodsScores := make(map[int64]float64)
	recentPaidGoodsMap := make(map[int64]struct{})
	userGoodsScores, recentPaidGoodsMap, err = c.recommendUserGoodsPreferenceCase.loadUserGoodsSignals(ctx, userId, candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	actorExposurePenalties := make(map[int64]float64)
	actorExposurePenalties, err = c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	profileScores := make(map[int64]float64)
	profileScores, err = c.recommendUserPreferenceCase.loadProfileScores(ctx, userId, categoryIds)
	if err != nil {
		return nil, err
	}
	scenePopularityScores := make(map[int64]float64)
	sceneExposurePenalties := make(map[int64]float64)
	scenePopularityScores, sceneExposurePenalties, err = c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	globalPopularityScores := make(map[int64]float64)
	globalPopularityScores, err = c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, err
	}

	return recommendcandidate.BuildPersonalized(goodsList, recommendcandidate.PersonalizedSignals{
		RelationScores:         relationScores,
		UserGoodsScores:        userGoodsScores,
		ProfileScores:          profileScores,
		ScenePopularityScores:  scenePopularityScores,
		GlobalPopularityScores: globalPopularityScores,
		SceneExposurePenalties: sceneExposurePenalties,
		ActorExposurePenalties: actorExposurePenalties,
		RecentPaidGoods:        recentPaidGoodsMap,
	}), nil
}

// buildAnonymousRecommendCandidates 构建匿名推荐候选集。
func (c *RecommendRequestCase) buildAnonymousRecommendCandidates(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	goodsList []*models.GoodsInfo,
) (map[int64]*recommendcore.Candidate, error) {
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 非法商品不参与匿名候选排序。
		if item == nil || item.ID <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.ID)
	}

	scenePopularityScores := make(map[int64]float64)
	sceneExposurePenalties := make(map[int64]float64)
	var err error
	scenePopularityScores, sceneExposurePenalties, err = c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	actorExposurePenalties := make(map[int64]float64)
	actorExposurePenalties, err = c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	globalPopularityScores := make(map[int64]float64)
	globalPopularityScores, err = c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, err
	}

	return recommendcandidate.BuildAnonymous(goodsList, recommendcandidate.AnonymousSignals{
		ScenePopularityScores:  scenePopularityScores,
		GlobalPopularityScores: globalPopularityScores,
		SceneExposurePenalties: sceneExposurePenalties,
		ActorExposurePenalties: actorExposurePenalties,
	}), nil
}

// resolveSceneRecall 解析推荐场景对应的召回商品与类目。
func (c *RecommendRequestCase) resolveSceneRecall(ctx context.Context, req *app.RecommendGoodsRequest, userId int64, sourceContext map[string]any, limit int) ([]int64, []int64, []string, error) {
	relationGoodsIds := make([]int64, 0)
	categoryIds := make([]int64, 0)
	recallSources := make([]string, 0, 3)
	var err error

	// 根据推荐场景选择不同的召回入口，优先复用最强业务上下文。
	switch req.GetScene() {
	case common.RecommendScene_CART:
		cartGoodsIds := make([]int64, 0)
		cartGoodsIds, err = c.listCurrentUserCartGoodsIds(ctx, userId)
		if err != nil {
			return nil, nil, nil, err
		}
		sourceContext["cartGoodsIds"] = cartGoodsIds
		// 当前购物车为空时，不再执行关联召回，交给后续画像或兜底逻辑处理。
		if len(cartGoodsIds) == 0 {
			return relationGoodsIds, categoryIds, recallSources, nil
		}

		// 购物车场景优先取购物车商品的关联商品。
		relationGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, cartGoodsIds, limit)
		if err != nil {
			return nil, nil, nil, err
		}
		// 关联商品不足时，再用购物车商品所属类目补足候选集。
		categoryIds, err = c.listCategoryIdsByGoodsIds(ctx, cartGoodsIds)
		if err != nil {
			return nil, nil, nil, err
		}
		recallSources = append(recallSources, "cart")
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 订单场景没有订单号时无法做强关联召回，直接返回空场景结果。
		if req.GetOrderId() <= 0 {
			return relationGoodsIds, categoryIds, recallSources, nil
		}

		orderGoodsIds := make([]int64, 0)
		orderGoodsIds, err = c.listOrderGoodsIds(ctx, req.GetOrderId())
		if err != nil {
			return nil, nil, nil, err
		}
		// 订单详情和支付成功都优先基于订单商品做强关联召回。
		relationGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, orderGoodsIds, limit)
		if err != nil {
			return nil, nil, nil, err
		}
		categoryIds, err = c.listCategoryIdsByGoodsIds(ctx, orderGoodsIds)
		if err != nil {
			return nil, nil, nil, err
		}
		recallSources = append(recallSources, "order")
	}

	return relationGoodsIds, categoryIds, recallSources, nil
}

// listCategoryCandidateGoodsIds 查询类目补足候选商品。
func (c *RecommendRequestCase) listCategoryCandidateGoodsIds(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, limit int) ([]int64, error) {
	// 没有类目或候选数量非法时，不再查询类目补足商品。
	if len(categoryIds) == 0 || limit <= 0 {
		return []int64{}, nil
	}
	list, _, err := c.pageGoods(ctx, categoryIds, excludeGoodsIds, 1, int64(limit))
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.ID)
	}
	return recommendcore.DedupeInt64s(goodsIds), nil
}

// listLatestCandidateGoodsIds 查询最新商品补足候选。
func (c *RecommendRequestCase) listLatestCandidateGoodsIds(ctx context.Context, excludeGoodsIds []int64, limit int) ([]int64, error) {
	// 候选数量非法时，不再查询最新商品兜底。
	if limit <= 0 {
		return []int64{}, nil
	}
	list, _, err := c.pageGoods(ctx, nil, excludeGoodsIds, 1, int64(limit))
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.ID)
	}
	return recommendcore.DedupeInt64s(goodsIds), nil
}

// listCurrentUserCartGoodsIds 查询当前用户购物车中的商品ID列表。
func (c *RecommendRequestCase) listCurrentUserCartGoodsIds(ctx context.Context, userId int64) ([]int64, error) {
	// 未登录用户没有专属购物车，直接返回空集合。
	if userId == 0 {
		return []int64{}, nil
	}

	userCartQuery := c.userCartRepo.Query(ctx).UserCart
	list, err := c.userCartRepo.List(ctx,
		repo.Where(userCartQuery.UserID.Eq(userId)),
	)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return recommendcore.DedupeInt64s(goodsIds), nil
}

// listOrderGoodsIds 查询订单中的商品ID列表。
func (c *RecommendRequestCase) listOrderGoodsIds(ctx context.Context, orderId int64) ([]int64, error) {
	orderGoodsQuery := c.orderGoodsRepo.Query(ctx).OrderGoods
	list, err := c.orderGoodsRepo.List(ctx,
		repo.Where(orderGoodsQuery.OrderID.Eq(orderId)),
	)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return recommendcore.DedupeInt64s(goodsIds), nil
}

// listCategoryIdsByGoodsIds 根据商品ID列表查询分类ID列表。
func (c *RecommendRequestCase) listCategoryIdsByGoodsIds(ctx context.Context, goodsIds []int64) ([]int64, error) {
	// 没有商品上下文时无需访问数据库查询类目。
	if len(goodsIds) == 0 {
		return []int64{}, nil
	}

	goodsQuery := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoCase.GoodsInfoRepo.List(ctx,
		repo.Where(goodsQuery.ID.In(goodsIds...)),
	)
	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIds = append(categoryIds, item.CategoryID)
	}
	return recommendcore.DedupeInt64s(categoryIds), nil
}

// listGoodsByIds 按商品 ID 顺序查询商品信息。
func (c *RecommendRequestCase) listGoodsByIds(ctx context.Context, goodsIds []int64) ([]*models.GoodsInfo, error) {
	// 没有候选商品时，无需访问数据库。
	if len(goodsIds) == 0 {
		return []*models.GoodsInfo{}, nil
	}

	goodsQuery := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	list, err := c.goodsInfoCase.GoodsInfoRepo.List(ctx,
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

	result := make([]*models.GoodsInfo, 0, len(goodsIds))
	for _, goodsId := range goodsIds {
		item, ok := goodsMap[goodsId]
		// 查询结果缺少对应商品时，直接跳过无效 ID。
		if !ok {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

// pageGoods 分页查询推荐候选商品。
func (c *RecommendRequestCase) pageGoods(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, pageNum, pageSize int64) ([]*models.GoodsInfo, int64, error) {
	// 分页数量非法时，直接返回空结果。
	if pageSize <= 0 {
		return []*models.GoodsInfo{}, 0, nil
	}

	goodsQuery := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(goodsQuery.CreatedAt.Desc()))
	// 传入类目过滤时，仅查询这些类目的商品。
	if len(categoryIds) > 0 {
		opts = append(opts, repo.Where(goodsQuery.CategoryID.In(categoryIds...)))
	}
	// 存在排除商品时，避免把已召回商品再次加入候选集。
	if len(excludeGoodsIds) > 0 {
		opts = append(opts, repo.Where(goodsQuery.ID.NotIn(excludeGoodsIds...)))
	}
	return c.goodsInfoCase.GoodsInfoRepo.Page(ctx, pageNum, pageSize, opts...)
}

// convertGoodsListToProto 批量转换商品列表。
func (c *RecommendRequestCase) convertGoodsListToProto(ctx context.Context, list []*models.GoodsInfo) []*app.GoodsInfo {
	member := util.IsMember(ctx)
	result := make([]*app.GoodsInfo, 0, len(list))
	for _, item := range list {
		result = append(result, c.goodsInfoCase.convertToProto(item, member))
	}
	return result
}

// saveRecommendRequest 保存推荐请求记录。
func (c *RecommendRequestCase) saveRecommendRequest(ctx context.Context, requestId string, actor *RecommendActor, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	sourceContextJson, err := json.Marshal(sourceContext)
	if err != nil {
		return err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GetId())
	}

	var goodsIdsJson []byte
	goodsIdsJson, err = json.Marshal(goodsIds)
	if err != nil {
		return err
	}
	var recallSourcesJson []byte
	recallSourcesJson, err = json.Marshal(recallSources)
	if err != nil {
		return err
	}

	// 推荐请求表保存的是本次实际下发结果，供曝光与点击链路统一回查。
	entity := &models.RecommendRequest{
		RequestID:     requestId,
		ActorType:     actor.ActorType,
		ActorID:       actor.ActorId,
		Scene:         int32(req.GetScene()),
		SourceContext: string(sourceContextJson),
		PageNum:       int32(req.GetPageNum()),
		PageSize:      int32(req.GetPageSize()),
		GoodsIds:      string(goodsIdsJson),
		RecallSources: string(recallSourcesJson),
	}
	return c.RecommendRequestRepo.Create(ctx, entity)
}
