package biz

import (
	"context"
	"encoding/json"
	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
	recommendEvent "shop/pkg/recommend/event"
	appDto "shop/service/app/dto"
	"sort"
	"time"

	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRequestCase 推荐请求子业务处理对象。
type RecommendRequestCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepo
	recommendRequestItemCase         *RecommendRequestItemCase
	goodsInfoCase                    *GoodsInfoCase
	orderGoodsCase                   *OrderGoodsCase
	userCartCase                     *UserCartCase
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
	recommendRequestItemCase *RecommendRequestItemCase,
	goodsInfoCase *GoodsInfoCase,
	orderGoodsCase *OrderGoodsCase,
	userCartCase *UserCartCase,
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
		recommendRequestItemCase:         recommendRequestItemCase,
		goodsInfoCase:                    goodsInfoCase,
		orderGoodsCase:                   orderGoodsCase,
		userCartCase:                     userCartCase,
		recommendExposureCase:            recommendExposureCase,
		recommendUserGoodsPreferenceCase: recommendUserGoodsPreferenceCase,
		recommendUserPreferenceCase:      recommendUserPreferenceCase,
		recommendGoodsRelationCase:       recommendGoodsRelationCase,
		recommendGoodsStatDayCase:        recommendGoodsStatDayCase,
		goodsStatDayCase:                 goodsStatDayCase,
	}
}

// bindRecommendRequestActor 将匿名请求主体绑定为登录主体。
func (c *RecommendRequestCase) bindRecommendRequestActor(ctx context.Context, anonymousId, userId int64) error {
	recommendRequestQuery := c.RecommendRequestRepo.Data.Query(ctx).RecommendRequest
	_, err := recommendRequestQuery.WithContext(ctx).
		Where(
			recommendRequestQuery.ActorType.Eq(recommendEvent.ActorTypeAnonymous),
			recommendRequestQuery.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendEvent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// listAnonymousRecommendGoods 查询匿名推荐商品列表并执行统一排序。
func (c *RecommendRequestCase) listAnonymousRecommendGoods(ctx context.Context, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	// 匿名态没有用户画像，候选池大小完全由分页深度决定。
	candidateLimit := recommendCandidate.ResolveCandidateLimit(req.GetPageNum(), req.GetPageSize())
	// 匿名态只使用近一段时间内的热度数据。
	startDate := time.Now().AddDate(0, 0, -recommendCandidate.AnonymousRecallDays)
	sceneGoodsIds, err := c.recommendGoodsStatDayCase.listSceneHotGoodsIds(ctx, req.GetScene(), startDate, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidateGoodsIds := make([]int64, 0)
	// 这里把场景热销和全站热销合并成匿名候选池。
	candidateGoodsIds, err = c.goodsStatDayCase.mergeAnonymousGoodsIds(ctx, sceneGoodsIds, startDate, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	// 场景热度和全站热度都没有数据时，退回最新商品分页。
	if len(candidateGoodsIds) == 0 {
		var pageGoodsInfoResponse *app.PageGoodsInfoResponse
		pageGoodsInfoResponse, err = c.goodsInfoCase.PageGoodsInfo(ctx, &app.PageGoodsInfoRequest{})
		if err != nil {
			return nil, 0, nil, nil, err
		}
		return pageGoodsInfoResponse.List, int64(pageGoodsInfoResponse.Total), []string{"latest"}, map[string]any{
			"candidateLimit":   candidateLimit,
			"sceneHotGoodsIds": sceneGoodsIds,
		}, nil
	}

	goodsList := make([]*app.GoodsInfo, 0)
	goodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	// 后续的热度分、曝光惩罚都会按商品 ID 回填。
	candidateGoodsIdList := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 非法商品不参与匿名候选排序。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidateGoodsIdList = append(candidateGoodsIdList, item.Id)
	}

	scenePopularityScores := make(map[int64]float64)
	sceneExposurePenalties := make(map[int64]float64)
	scenePopularityScores, sceneExposurePenalties, err = c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	actorExposurePenalties := make(map[int64]float64)
	actorExposurePenalties, err = c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	globalPopularityScores := make(map[int64]float64)
	globalPopularityScores, err = c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIdList)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	// 匿名态不看用户偏好，只使用公共排序信号。
	candidates := recommendCandidate.BuildAnonymous(goodsList, recommendCandidate.AnonymousSignals{
		ScenePopularityScores:  scenePopularityScores,
		GlobalPopularityScores: globalPopularityScores,
		SceneExposurePenalties: sceneExposurePenalties,
		ActorExposurePenalties: actorExposurePenalties,
	})
	// 这里不仅排序，还会顺带做类目打散。
	rankedGoods := recommendCandidate.RankGoods(candidates)
	total := int64(len(rankedGoods))
	offset := int((req.GetPageNum() - 1) * req.GetPageSize())
	// 当前页超过候选集时，返回匿名空页。
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{"anonymous_hot"}, map[string]any{}, nil
	}
	end := offset + int(req.GetPageSize())
	// 分页结束位置越界时，按候选集最后一条裁剪。
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}

	// explain 只收集当前页，避免响应上下文过大。
	pageRecallSources := make(map[string]struct{}, 6)
	scoreDetails := make([]recommendcore.ScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		candidate, ok := candidates[item.Id]
		// 候选解释缺失时仅跳过解释信息，不影响结果列表。
		if !ok {
			continue
		}
		for source := range candidate.RecallSources {
			pageRecallSources[source] = struct{}{}
		}
		scoreDetails = append(scoreDetails, recommendcore.ScoreDetail{
			GoodsId:               candidate.Goods.Id,
			FinalScore:            candidate.FinalScore,
			RelationScore:         candidate.RelationScore,
			UserGoodsScore:        candidate.UserGoodsScore,
			ProfileScore:          candidate.ProfileScore,
			ScenePopularityScore:  candidate.ScenePopularityScore,
			GlobalPopularityScore: candidate.GlobalPopularityScore,
			FreshnessScore:        candidate.FreshnessScore,
			ExposurePenalty:       candidate.ExposurePenalty,
			ActorExposurePenalty:  candidate.ActorExposurePenalty,
			RepeatPenalty:         candidate.RepeatPenalty,
		})
	}
	recallSourceList := make([]string, 0, len(pageRecallSources))
	for source := range pageRecallSources {
		recallSourceList = append(recallSourceList, source)
	}
	// 召回来源按稳定顺序返回，便于日志和前端比对。
	sort.Strings(recallSourceList)
	for i := range scoreDetails {
		candidate, ok := candidates[scoreDetails[i].GoodsId]
		// 候选解释缺失时，上面已经跳过，这里只做保护。
		if !ok {
			continue
		}
		recallSources := make([]string, 0, len(candidate.RecallSources))
		for source := range candidate.RecallSources {
			recallSources = append(recallSources, source)
		}
		// 单商品 explain 中的召回来源也保持稳定顺序。
		sort.Strings(recallSources)
		scoreDetails[i].RecallSources = recallSources
	}
	return rankedGoods[offset:end], total, recallSourceList, map[string]any{
		"candidateLimit":             candidateLimit,
		"sceneHotGoodsIds":           sceneGoodsIds,
		"anonymousCandidateGoodsIds": candidateGoodsIds,
		"returnedScoreDetails":       scoreDetails,
	}, nil
}

// listRecommendGoods 查询推荐商品列表并执行统一排序。
func (c *RecommendRequestCase) listRecommendGoods(
	ctx context.Context,
	actor *appDto.RecommendActor,
	req *app.RecommendGoodsRequest,
	userId int64,
) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	pageNum := req.GetPageNum()
	pageSize := req.GetPageSize()
	// 分页数量非法时直接返回空结果，避免继续构造候选集。
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	// 这部分上下文会直接写入推荐请求表。
	sourceContext := map[string]any{
		"orderId": req.GetOrderId(),
	}
	// 这里放强业务上下文直接召回出的商品。
	priorityGoodsIds := make([]int64, 0)
	// 这里放用于补足候选池的类目。
	categoryIds := make([]int64, 0)
	// 这里记录本次命中的召回入口。
	recallSources := make([]string, 0, 4)

	// 登录态优先消费当前页面最强的业务上下文。
	switch req.GetScene() {
	case common.RecommendScene_CART:
		cartGoodsIds, err := c.userCartCase.listGoodsIdsByUserId(ctx, userId)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		sourceContext["cartGoodsIds"] = cartGoodsIds
		// 购物车为空时，跳过购物车关联召回。
		if len(cartGoodsIds) > 0 {
			priorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, cartGoodsIds, pageSize)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			categoryIds, err = c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, cartGoodsIds)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			recallSources = append(recallSources, "cart")
		}
	case common.RecommendScene_ORDER_DETAIL, common.RecommendScene_ORDER_PAID:
		// 没有订单号就无法恢复订单商品上下文。
		if req.GetOrderId() > 0 {
			orderGoodsIds, err := c.orderGoodsCase.listGoodsIdsByOrderId(ctx, req.GetOrderId())
			if err != nil {
				return nil, 0, nil, nil, err
			}
			priorityGoodsIds, err = c.recommendGoodsRelationCase.listRelatedGoodsIds(ctx, orderGoodsIds, pageSize)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			categoryIds, err = c.goodsInfoCase.listCategoryIdsByGoodsIds(ctx, orderGoodsIds)
			if err != nil {
				return nil, 0, nil, nil, err
			}
			recallSources = append(recallSources, "order")
		}
	}

	// 用户画像只负责补足，不覆盖强场景召回。
	profileCategoryIds, err := c.recommendUserPreferenceCase.listPreferredCategoryIds(ctx, userId, 3)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	if len(profileCategoryIds) > 0 {
		categoryIds = append(categoryIds, profileCategoryIds...)
		recallSources = append(recallSources, "profile")
	}
	if len(recallSources) == 0 {
		// 场景和画像都没有命中时，回退到 latest。
		recallSources = append(recallSources, "latest")
	}

	// 这里统一去重，避免同一商品或类目重复参与候选计算。
	priorityGoodsIds = recommendcore.DedupeInt64s(priorityGoodsIds)
	categoryIds = recommendcore.DedupeInt64s(categoryIds)
	recallSources = recommendcore.DedupeStrings(recallSources)

	// 分页越深，候选池越大，避免深页直接无货可排。
	candidateLimit := recommendCandidate.ResolveCandidateLimit(pageNum, pageSize)
	excludeGoodsIds := recommendcore.DedupeInt64s(priorityGoodsIds)
	categoryCandidateIds := make([]int64, 0)
	if len(categoryIds) > 0 && candidateLimit > 0 {
		goodsQuery := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
		opts := make([]repo.QueryOption, 0, 2)
		opts = append(opts, repo.Where(goodsQuery.CategoryID.In(categoryIds...)))
		if len(excludeGoodsIds) > 0 {
			opts = append(opts, repo.Where(goodsQuery.ID.NotIn(excludeGoodsIds...)))
		}
		pageResp, err := c.goodsInfoCase.PageGoodsInfo(ctx, &app.PageGoodsInfoRequest{
			PageNum:  1,
			PageSize: int64(candidateLimit),
		}, opts...)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		// 类目补足阶段只取商品 ID 进入后续候选池。
		for _, item := range pageResp.List {
			categoryCandidateIds = append(categoryCandidateIds, item.GetId())
		}
		categoryCandidateIds = recommendcore.DedupeInt64s(categoryCandidateIds)
	}
	// latest 兜底前先排除已召回商品，避免重复补进来。
	excludeGoodsIds = recommendcore.DedupeInt64s(append(excludeGoodsIds, categoryCandidateIds...))

	latestCandidateIds := make([]int64, 0)
	if candidateLimit > 0 {
		goodsQuery := c.goodsInfoCase.GoodsInfoRepo.Query(ctx).GoodsInfo
		opts := make([]repo.QueryOption, 0, 1)
		if len(excludeGoodsIds) > 0 {
			opts = append(opts, repo.Where(goodsQuery.ID.NotIn(excludeGoodsIds...)))
		}
		pageResp, err := c.goodsInfoCase.PageGoodsInfo(ctx, &app.PageGoodsInfoRequest{
			PageNum:  1,
			PageSize: int64(candidateLimit),
		}, opts...)
		if err != nil {
			return nil, 0, nil, nil, err
		}
		// latest 兜底阶段也只需要商品 ID。
		for _, item := range pageResp.List {
			latestCandidateIds = append(latestCandidateIds, item.GetId())
		}
		latestCandidateIds = recommendcore.DedupeInt64s(latestCandidateIds)
	}

	// 最终候选池按 强召回 + 类目补足 + latest 兜底 合并。
	allCandidateIds := recommendcore.DedupeInt64s(append(append(priorityGoodsIds, categoryCandidateIds...), latestCandidateIds...))
	// 没有候选商品时，直接返回空结果。
	if len(allCandidateIds) == 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	goodsList := make([]*app.GoodsInfo, 0)
	goodsList, err = c.goodsInfoCase.listByGoodsIds(ctx, allCandidateIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	// 这份商品 ID 用来对齐各种商品级排序信号。
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	// 这份类目 ID 用来对齐画像类偏好分。
	candidateCategoryIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 非法商品不参与候选信号计算。
		if item == nil || item.Id <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.Id)
		candidateCategoryIds = append(candidateCategoryIds, item.CategoryId)
	}

	relationScores := make(map[int64]float64)
	relationScores, err = c.recommendGoodsRelationCase.loadRelationScores(ctx, priorityGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	userGoodsScores := make(map[int64]float64)
	recentPaidGoodsMap := make(map[int64]struct{})
	userGoodsScores, recentPaidGoodsMap, err = c.recommendUserGoodsPreferenceCase.loadUserGoodsSignals(ctx, userId, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	actorExposurePenalties := make(map[int64]float64)
	actorExposurePenalties, err = c.recommendExposureCase.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	profileScores := make(map[int64]float64)
	profileScores, err = c.recommendUserPreferenceCase.loadProfileScores(ctx, userId, candidateCategoryIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	scenePopularityScores := make(map[int64]float64)
	sceneExposurePenalties := make(map[int64]float64)
	scenePopularityScores, sceneExposurePenalties, err = c.recommendGoodsStatDayCase.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	globalPopularityScores := make(map[int64]float64)
	globalPopularityScores, err = c.goodsStatDayCase.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	// 登录态会融合关系分、偏好分、热度分和惩罚分。
	candidates := recommendCandidate.BuildPersonalized(goodsList, recommendCandidate.PersonalizedSignals{
		RelationScores:         relationScores,
		UserGoodsScores:        userGoodsScores,
		ProfileScores:          profileScores,
		ScenePopularityScores:  scenePopularityScores,
		GlobalPopularityScores: globalPopularityScores,
		SceneExposurePenalties: sceneExposurePenalties,
		ActorExposurePenalties: actorExposurePenalties,
		RecentPaidGoods:        recentPaidGoodsMap,
	})
	// 这里同时完成最终排序和类目去扎堆。
	rankedGoods := recommendCandidate.RankGoods(candidates)
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

	// 当前页才需要 explain，整池 explain 没必要返回。
	list := make([]*app.GoodsInfo, 0, end-offset)
	pageRecallSources := make(map[string]struct{}, 8)
	scoreDetails := make([]recommendcore.ScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		list = append(list, item)
		candidate, ok := candidates[item.Id]
		// 候选明细缺失时，仅跳过解释信息，不影响商品返回。
		if !ok {
			continue
		}
		for source := range candidate.RecallSources {
			pageRecallSources[source] = struct{}{}
		}
		scoreDetails = append(scoreDetails, recommendcore.ScoreDetail{
			GoodsId:               candidate.Goods.Id,
			FinalScore:            candidate.FinalScore,
			RelationScore:         candidate.RelationScore,
			UserGoodsScore:        candidate.UserGoodsScore,
			ProfileScore:          candidate.ProfileScore,
			ScenePopularityScore:  candidate.ScenePopularityScore,
			GlobalPopularityScore: candidate.GlobalPopularityScore,
			FreshnessScore:        candidate.FreshnessScore,
			ExposurePenalty:       candidate.ExposurePenalty,
			ActorExposurePenalty:  candidate.ActorExposurePenalty,
			RepeatPenalty:         candidate.RepeatPenalty,
		})
	}
	recallSourceList := make([]string, 0, len(pageRecallSources))
	for source := range pageRecallSources {
		recallSourceList = append(recallSourceList, source)
	}
	// 召回来源按稳定顺序返回，便于日志和前端比对。
	sort.Strings(recallSourceList)
	for i := range scoreDetails {
		candidate, ok := candidates[scoreDetails[i].GoodsId]
		// 候选解释缺失时，上面已经跳过，这里只做保护。
		if !ok {
			continue
		}
		recallSources := make([]string, 0, len(candidate.RecallSources))
		for source := range candidate.RecallSources {
			recallSources = append(recallSources, source)
		}
		// 单商品 explain 中的召回来源也保持稳定顺序。
		sort.Strings(recallSources)
		scoreDetails[i].RecallSources = recallSources
	}
	return list, total, recallSourceList, map[string]any{
		"candidateLimit":       candidateLimit,
		"priorityGoodsIds":     priorityGoodsIds,
		"categoryIds":          categoryIds,
		"orderId":              req.GetOrderId(),
		"returnedScoreDetails": scoreDetails,
	}, nil
}

// saveRecommendRequest 保存推荐请求记录。
func (c *RecommendRequestCase) saveRecommendRequest(ctx context.Context, requestId string, actor *appDto.RecommendActor, req *app.RecommendGoodsRequest, sourceContext map[string]any, list []*app.GoodsInfo, recallSources []string) error {
	// 主表只保留排查请求所需的精简上下文，大体量 explain 明细下沉到 item 表。
	persistedSourceContext := make(map[string]any, len(sourceContext))
	for key, value := range sourceContext {
		// 逐商品 explain 明细已经落到 item 表，这里不再重复保存。
		if key == "returnedScoreDetails" {
			continue
		}
		// 主体信息已经有独立列，不再在上下文里重复冗余。
		if key == "actorType" || key == "actorId" {
			continue
		}
		persistedSourceContext[key] = value
	}
	sourceContextJson, err := json.Marshal(persistedSourceContext)
	if err != nil {
		return err
	}

	createdAt := time.Now()
	// 这条记录会被曝光、点击、下单链路按 requestId 回查。
	entity := &models.RecommendRequest{
		RequestID: requestId,
		ActorType: actor.ActorType,
		ActorID:   actor.ActorId,
		Scene:     int32(req.GetScene()),
		// 精简后的上下文仍然保留场景调试信息。
		SourceContext: string(sourceContextJson),
		PageNum:       int32(req.GetPageNum()),
		PageSize:      int32(req.GetPageSize()),
		CreatedAt:     createdAt,
	}
	return c.RecommendRequestRepo.Data.Transaction(ctx, func(ctx context.Context) error {
		err = c.RecommendRequestRepo.Create(ctx, entity)
		if err != nil {
			return err
		}
		return c.recommendRequestItemCase.batchCreateByRecommendRequest(ctx, entity.ID, req, sourceContext, list, recallSources)
	})
}
