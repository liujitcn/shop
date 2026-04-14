package scene

import (
	"context"
	"recommend"
	"recommend/internal/model"
	"recommend/internal/rank"
	"recommend/internal/recall"
	"recommend/internal/replace"
	"time"
)

const recentPaidWindowDays = 15

// Run 执行指定场景的推荐流水线。
func Run(ctx context.Context, request model.Request, dependencies recommend.Dependencies) ([]*model.Candidate, error) {
	pipeline, err := ResolvePipeline(request.Scene)
	if err != nil {
		return nil, err
	}
	return pipeline(ctx, request, dependencies)
}

// buildRecallRequest 构建召回层使用的统一请求参数。
func buildRecallRequest(request model.Request, dependencies recommend.Dependencies) recall.Request {
	return recall.Request{
		Scene:         request.Scene,
		Actor:         request.Actor,
		Context:       request.Context,
		Limit:         int32(request.Offset() + request.Limit()*4),
		ReferenceTime: time.Now(),
		Dependencies:  dependencies,
	}
}

// finalizeCandidates 执行过滤、惩罚、排序和兜底补足。
func finalizeCandidates(
	ctx context.Context,
	request model.Request,
	dependencies recommend.Dependencies,
	primary []*model.Candidate,
	fallback []*model.Candidate,
) ([]*model.Candidate, error) {
	primary = replace.FilterUnavailableGoods(primary)
	primary = replace.FilterContextGoods(request, primary)
	fallback = replace.FilterUnavailableGoods(fallback)
	fallback = replace.FilterContextGoods(request, fallback)

	recentPaidGoodsIds, err := loadRecentPaidGoodsIds(ctx, request, dependencies)
	if err != nil {
		return nil, err
	}
	replace.ApplyRepeatPenalty(primary, recentPaidGoodsIds, 1)
	replace.ApplyRepeatPenalty(fallback, recentPaidGoodsIds, 1)

	weights := rank.DefaultWeights(request.Scene)
	rank.ScoreCandidates(primary, weights, time.Now())
	rank.ScoreCandidates(fallback, weights, time.Now())

	primary = rank.RankCandidates(primary, rank.RankOptions{})
	fallback = rank.RankCandidates(fallback, rank.RankOptions{})
	return replace.MergeFallback(primary, fallback, request.Offset()+request.Limit()), nil
}

// mergeCandidates 合并多路召回结果。
func mergeCandidates(groups ...[]*model.Candidate) []*model.Candidate {
	candidateMap := make(map[int64]*model.Candidate)
	order := make([]int64, 0)

	for _, group := range groups {
		for _, item := range group {
			// 空候选或缺失商品实体时，不参与多路合并。
			if item == nil || item.Goods == nil || item.Goods.Id <= 0 {
				continue
			}

			existing, ok := candidateMap[item.Goods.Id]
			if !ok {
				candidateMap[item.Goods.Id] = cloneCandidate(item)
				order = append(order, item.Goods.Id)
				continue
			}
			mergeCandidate(existing, item)
		}
	}

	result := make([]*model.Candidate, 0, len(order))
	for _, goodsId := range order {
		result = append(result, candidateMap[goodsId])
	}
	return result
}

// cloneCandidate 复制候选对象，避免多路召回合并时复用同一个指针。
func cloneCandidate(candidate *model.Candidate) *model.Candidate {
	result := model.BuildCandidate(candidate.Goods)
	result.Score = candidate.Score
	for source := range candidate.RecallSources {
		result.AddRecallSource(source)
	}
	for _, reason := range candidate.TraceReasons {
		result.AddTraceReason(reason)
	}
	return result
}

// mergeCandidate 合并相同商品的多路召回结果。
func mergeCandidate(target *model.Candidate, source *model.Candidate) {
	target.Score.RelationScore += source.Score.RelationScore
	target.Score.UserGoodsScore += source.Score.UserGoodsScore
	target.Score.CategoryScore += source.Score.CategoryScore
	target.Score.SceneHotScore += source.Score.SceneHotScore
	target.Score.GlobalHotScore += source.Score.GlobalHotScore
	target.Score.SessionScore += source.Score.SessionScore
	target.Score.ExternalScore += source.Score.ExternalScore
	target.Score.CollaborativeScore += source.Score.CollaborativeScore
	target.Score.UserNeighborScore += source.Score.UserNeighborScore
	target.Score.ExposurePenalty += source.Score.ExposurePenalty
	target.Score.RepeatPenalty += source.Score.RepeatPenalty

	for sourceName := range source.RecallSources {
		target.AddRecallSource(sourceName)
	}
	for _, reason := range source.TraceReasons {
		target.AddTraceReason(reason)
	}
}

// withGoodsId 返回覆盖了上下文商品编号的新请求。
func withGoodsId(request model.Request, goodsId int64) model.Request {
	result := request
	result.Context.GoodsId = goodsId
	return result
}

// loadRecentPaidGoodsIds 加载最近已支付商品编号，用于重复购买惩罚。
func loadRecentPaidGoodsIds(ctx context.Context, request model.Request, dependencies recommend.Dependencies) ([]int64, error) {
	// 匿名主体没有支付历史，不需要计算重复购买惩罚。
	if !request.Actor.IsUser() {
		return nil, nil
	}
	// 订单数据源尚未接入时，跳过重复购买惩罚，避免阻塞主链路。
	if dependencies.Order == nil {
		return nil, nil
	}
	startAt := time.Now().AddDate(0, 0, -recentPaidWindowDays)
	list, err := dependencies.Order.ListRecentPaidGoods(ctx, request.Actor.Id, startAt, int32(request.Limit()))
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		// 非法商品编号不参与重复购买惩罚集合。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsId)
	}
	return goodsIds, nil
}

// loadOrderAnchorGoodsId 读取订单中的第一个有效商品编号，作为简化关联召回的锚点。
func loadOrderAnchorGoodsId(ctx context.Context, dependencies recommend.Dependencies, orderId int64) (int64, error) {
	// 订单编号缺失时，无法从订单商品中选择锚点。
	if orderId <= 0 {
		return 0, nil
	}
	// 订单数据源尚未接入时，跳过订单锚点解析，避免阻塞主链路。
	if dependencies.Order == nil {
		return 0, nil
	}
	list, err := dependencies.Order.ListOrderGoods(ctx, orderId)
	if err != nil {
		return 0, err
	}
	for _, item := range list {
		// 当前只选择第一个有效商品作为关联召回锚点。
		if item == nil || item.GoodsId <= 0 {
			continue
		}
		return item.GoodsId, nil
	}
	return 0, nil
}
