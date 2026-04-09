package biz

import (
	"context"
	"time"

	"shop/api/gen/go/app"
	"shop/pkg/gen/models"
	recommendcore "shop/pkg/recommend/core"
	recommendexplain "shop/pkg/recommend/explain"
	recommendfilter "shop/pkg/recommend/filter"
	recommendrank "shop/pkg/recommend/rank"
	"shop/service/app/util"
)

const (
	recommendCandidatePoolMultiplier = 8
	recommendCandidatePoolMin        = 80
	recommendCandidatePoolMax        = 240
)

// listRecommendGoods 查询推荐商品列表并执行统一排序。
func (c *RecommendCase) listRecommendGoods(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	userID int64,
	priorityGoodsIds []int64,
	categoryIds []int64,
	pageNum, pageSize int64,
) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	// 每页数量非法时直接返回空结果，避免继续做无意义的候选计算。
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	candidateLimit := c.resolveCandidateLimit(pageNum, pageSize)
	excludeGoodsIds := dedupeInt64s(priorityGoodsIds)
	categoryCandidateIds, err := c.listCategoryCandidateGoodsIds(ctx, categoryIds, excludeGoodsIds, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	excludeGoodsIds = dedupeInt64s(append(excludeGoodsIds, categoryCandidateIds...))
	latestCandidateIds, err := c.listLatestCandidateGoodsIds(ctx, excludeGoodsIds, candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	allCandidateIds := dedupeInt64s(append(append(priorityGoodsIds, categoryCandidateIds...), latestCandidateIds...))
	// 场景召回和补足召回都为空时，直接返回空页。
	if len(allCandidateIds) == 0 {
		return []*app.GoodsInfo{}, 0, []string{}, map[string]any{}, nil
	}

	goodsList, err := c.listGoodsByIds(ctx, allCandidateIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidates, err := c.buildRecommendCandidates(ctx, actor, req, userID, priorityGoodsIds, goodsList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	rankedGoods := c.rankRecommendCandidates(candidates)
	total := int64(len(rankedGoods))

	offset := int((pageNum - 1) * pageSize)
	// 分页偏移超出范围时返回空列表，但仍返回总数。
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{}, map[string]any{}, nil
	}
	end := offset + int(pageSize)
	// 末页需要截断到真实结果上界。
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}

	member := util.IsMember(ctx)
	list := make([]*app.GoodsInfo, 0, end-offset)
	pageRecallSources := make(map[string]struct{}, 8)
	scoreDetails := make([]recommendcore.ScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		list = append(list, c.convertGoodsToProto(item, member))
		candidate, ok := candidates[item.ID]
		// 理论上排序结果都应能在候选集中找到，这里额外兜底防脏数据。
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

// resolveCandidateLimit 计算当前分页请求需要的候选池大小。
func (c *RecommendCase) resolveCandidateLimit(pageNum, pageSize int64) int {
	limit := int(pageNum * pageSize * recommendCandidatePoolMultiplier)
	// 小分页场景也要保留最小候选池，避免排序质量过低。
	if limit < recommendCandidatePoolMin {
		limit = recommendCandidatePoolMin
	}
	// 候选池过大时需要截断，避免一次请求拉取过多商品。
	if limit > recommendCandidatePoolMax {
		limit = recommendCandidatePoolMax
	}
	return limit
}

// listCategoryCandidateGoodsIds 查询类目补足候选商品。
func (c *RecommendCase) listCategoryCandidateGoodsIds(ctx context.Context, categoryIds []int64, excludeGoodsIds []int64, limit int) ([]int64, error) {
	// 没有类目上下文或候选上限时，不再补类目商品。
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
	return dedupeInt64s(goodsIds), nil
}

// listLatestCandidateGoodsIds 查询最新商品补足候选。
func (c *RecommendCase) listLatestCandidateGoodsIds(ctx context.Context, excludeGoodsIds []int64, limit int) ([]int64, error) {
	// 候选上限非法时直接返回空集合。
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
	return dedupeInt64s(goodsIds), nil
}

// buildRecommendCandidates 构建登录态推荐候选集。
func (c *RecommendCase) buildRecommendCandidates(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	userID int64,
	priorityGoodsIds []int64,
	goodsList []*models.GoodsInfo,
) (map[int64]*recommendcore.Candidate, error) {
	candidates := make(map[int64]*recommendcore.Candidate, len(goodsList))
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	categoryIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 跳过空商品和非法商品 ID，避免后续 map 污染。
		if item == nil || item.ID <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.ID)
		categoryIds = append(categoryIds, item.CategoryID)
		candidates[item.ID] = &recommendcore.Candidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 4),
		}
	}

	relationScores, err := c.loadRelationScores(ctx, priorityGoodsIds)
	if err != nil {
		return nil, err
	}
	userGoodsScores, recentPaidGoodsMap, err := c.loadUserGoodsSignals(ctx, userID, candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	actorExposurePenalties, err := c.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	profileScores, err := c.loadProfileScores(ctx, userID, categoryIds)
	if err != nil {
		return nil, err
	}
	scenePopularityScores, sceneExposurePenalties, err := c.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	globalPopularityScores, err := c.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, err
	}

	for goodsId, candidate := range candidates {
		candidate.RelationScore = relationScores[goodsId]
		candidate.UserGoodsScore = userGoodsScores[goodsId]
		candidate.ProfileScore = profileScores[candidate.Goods.CategoryID]
		candidate.ScenePopularityScore = scenePopularityScores[goodsId]
		candidate.GlobalPopularityScore = globalPopularityScores[goodsId]
		candidate.FreshnessScore = recommendrank.CalculateFreshnessScore(candidate.Goods.CreatedAt)
		candidate.ExposurePenalty = sceneExposurePenalties[goodsId]
		candidate.ActorExposurePenalty = actorExposurePenalties[goodsId]
		candidate.RepeatPenalty = c.calculateRepeatPenalty(goodsId, recentPaidGoodsMap)
		candidate.FinalScore = recommendrank.CalculateFinalScore(candidate)
		// 命中商品关联信号时记录对应召回来源。
		if candidate.RelationScore > 0 {
			candidate.RecallSources["relation"] = struct{}{}
		}
		// 命中用户商品偏好时记录用户强兴趣来源。
		if candidate.UserGoodsScore > 0 {
			candidate.RecallSources["user_goods"] = struct{}{}
		}
		// 命中用户画像偏好时补充画像来源。
		if candidate.ProfileScore > 0 {
			candidate.RecallSources["profile"] = struct{}{}
		}
		// 命中场景热度时记录场景公共召回。
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources["scene_hot"] = struct{}{}
		}
		// 命中全站热度时记录全局热度来源。
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources["global_hot"] = struct{}{}
		}
		// 没有任何强信号时使用最新商品兜底。
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources["latest"] = struct{}{}
		}
		// 命中个体曝光惩罚时补充解释来源，方便回溯。
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources["actor_penalty"] = struct{}{}
		}
	}
	return candidates, nil
}

// listAnonymousRecommendGoods 查询匿名推荐商品列表并执行统一排序。
func (c *RecommendCase) listAnonymousRecommendGoods(ctx context.Context, actor *RecommendActor, req *app.RecommendGoodsRequest, pageNum, pageSize int64) ([]*app.GoodsInfo, int64, []string, map[string]any, error) {
	// 每页数量非法时直接返回空结果。
	if pageSize <= 0 {
		return []*app.GoodsInfo{}, 0, []string{"anonymous_hot"}, map[string]any{}, nil
	}
	candidateLimit := c.resolveCandidateLimit(pageNum, pageSize)
	sceneGoodsIds, err := c.listSceneHotGoodsIds(ctx, req.GetScene(), time.Now().AddDate(0, 0, -recommendAnonymousRecallDays), candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidateGoodsIds, err := c.mergeAnonymousGoodsIds(ctx, sceneGoodsIds, time.Now().AddDate(0, 0, -recommendAnonymousRecallDays), candidateLimit)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	// 匿名推荐没有任何候选时，直接降级为最新商品分页。
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
	goodsList, err := c.listGoodsByIds(ctx, candidateGoodsIds)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	candidates, err := c.buildAnonymousRecommendCandidates(ctx, actor, req, goodsList)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	rankedGoods := c.rankRecommendCandidates(candidates)
	total := int64(len(rankedGoods))
	offset := int((pageNum - 1) * pageSize)
	// 分页偏移越界时仍返回总数，便于前端结束加载。
	if offset >= len(rankedGoods) {
		return []*app.GoodsInfo{}, total, []string{"anonymous_hot"}, map[string]any{}, nil
	}
	end := offset + int(pageSize)
	// 末页需要按真实结果截断。
	if end > len(rankedGoods) {
		end = len(rankedGoods)
	}
	pageRecallSources := make(map[string]struct{}, 6)
	scoreDetails := make([]recommendcore.ScoreDetail, 0, end-offset)
	for _, item := range rankedGoods[offset:end] {
		candidate, ok := candidates[item.ID]
		// 排序结果与候选集不一致时跳过该解释信息。
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

// buildAnonymousRecommendCandidates 构建匿名推荐候选集。
func (c *RecommendCase) buildAnonymousRecommendCandidates(
	ctx context.Context,
	actor *RecommendActor,
	req *app.RecommendGoodsRequest,
	goodsList []*models.GoodsInfo,
) (map[int64]*recommendcore.Candidate, error) {
	candidates := make(map[int64]*recommendcore.Candidate, len(goodsList))
	candidateGoodsIds := make([]int64, 0, len(goodsList))
	for _, item := range goodsList {
		// 跳过空商品和非法商品 ID，避免候选集出现脏数据。
		if item == nil || item.ID <= 0 {
			continue
		}
		candidateGoodsIds = append(candidateGoodsIds, item.ID)
		candidates[item.ID] = &recommendcore.Candidate{
			Goods:         item,
			RecallSources: make(map[string]struct{}, 3),
		}
	}
	scenePopularityScores, sceneExposurePenalties, err := c.loadScenePopularitySignals(ctx, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	actorExposurePenalties, err := c.loadActorExposurePenalties(ctx, actor, int32(req.GetScene()), candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	globalPopularityScores, err := c.loadGlobalPopularityScores(ctx, candidateGoodsIds)
	if err != nil {
		return nil, err
	}
	for goodsId, candidate := range candidates {
		candidate.ScenePopularityScore = scenePopularityScores[goodsId]
		candidate.GlobalPopularityScore = globalPopularityScores[goodsId]
		candidate.FreshnessScore = recommendrank.CalculateFreshnessScore(candidate.Goods.CreatedAt)
		candidate.ExposurePenalty = sceneExposurePenalties[goodsId]
		candidate.ActorExposurePenalty = actorExposurePenalties[goodsId]
		candidate.FinalScore = recommendrank.CalculateAnonymousFinalScore(candidate)
		// 匿名推荐优先解释场景热度来源。
		if candidate.ScenePopularityScore > 0 {
			candidate.RecallSources["scene_hot"] = struct{}{}
		}
		// 全局热度作为第二公共来源。
		if candidate.GlobalPopularityScore > 0 {
			candidate.RecallSources["global_hot"] = struct{}{}
		}
		// 没有命中任何热度信号时回退到最新商品。
		if len(candidate.RecallSources) == 0 {
			candidate.RecallSources["latest"] = struct{}{}
		}
		// 个体曝光惩罚需要进入解释信息，方便问题排查。
		if candidate.ActorExposurePenalty > 0 {
			candidate.RecallSources["actor_penalty"] = struct{}{}
		}
	}
	return candidates, nil
}

// rankRecommendCandidates 对候选商品执行统一排序和多样性过滤。
func (c *RecommendCase) rankRecommendCandidates(candidates map[int64]*recommendcore.Candidate) []*models.GoodsInfo {
	rankedCandidates := recommendrank.RankCandidates(candidates)
	return recommendfilter.DiversifyCandidates(rankedCandidates, recommendfilter.DefaultMaxPerCategory)
}

// convertGoodsListToProto 批量转换商品列表。
func (c *RecommendCase) convertGoodsListToProto(ctx context.Context, list []*models.GoodsInfo) []*app.GoodsInfo {
	member := util.IsMember(ctx)
	result := make([]*app.GoodsInfo, 0, len(list))
	for _, item := range list {
		result = append(result, c.convertGoodsToProto(item, member))
	}
	return result
}
