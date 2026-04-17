package rank

import (
	app "shop/api/gen/go/app"
	"shop/api/gen/go/conf"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendcore "shop/pkg/recommend/core"
	recommendDomain "shop/pkg/recommend/domain"
)

// CandidateMarker 表示候选构建完成后的附加处理函数。
type CandidateMarker func(candidates map[int64]*recommendcore.Candidate)

// RankingResult 表示在线排序执行后的结果快照。
type RankingResult struct {
	Candidates      map[int64]*recommendcore.Candidate
	PageSnapshot    RankedPageSnapshot
	ExplainSnapshot PageExplainSnapshot
	StageContext    map[string]any
}

// ExecuteAnonymousRanking 执行匿名态候选构建、排序分页和 explain 快照组装。
func ExecuteAnonymousRanking(
	request *recommendDomain.GoodsRequest,
	goodsList []*app.GoodsInfo,
	signals recommendDomain.AnonymousSignals,
	rankWeightConfig *conf.GoodsRecommendAnonymousRankWeightConfig,
	marker CandidateMarker,
	strategy *recommendDomain.StrategyVersionConfig,
	stageScores StageScoreSet,
) RankingResult {
	candidates := recommendCandidate.BuildAnonymous(goodsList, signals, rankWeightConfig)
	// 当前存在附加标记逻辑时，在排序前补齐 explain 来源。
	if marker != nil {
		marker(candidates)
	}
	stageContext := ApplyRankingStrategy(candidates, strategy, stageScores)
	rankedGoods := recommendCandidate.RankGoods(candidates)
	pageSnapshot := BuildRankedPageSnapshot(request, rankedGoods)
	explainSnapshot := BuildPageExplainSnapshot(pageSnapshot.PageGoods, candidates)
	return RankingResult{
		Candidates:      candidates,
		PageSnapshot:    pageSnapshot,
		ExplainSnapshot: explainSnapshot,
		StageContext:    stageContext,
	}
}

// ExecutePersonalizedRanking 执行登录态候选构建、排序分页和 explain 快照组装。
func ExecutePersonalizedRanking(
	request *recommendDomain.GoodsRequest,
	goodsList []*app.GoodsInfo,
	signals recommendDomain.PersonalizedSignals,
	rankWeightConfig *conf.GoodsRecommendPersonalizedRankWeightConfig,
	marker CandidateMarker,
	strategy *recommendDomain.StrategyVersionConfig,
	stageScores StageScoreSet,
) RankingResult {
	candidates := recommendCandidate.BuildPersonalized(goodsList, signals, rankWeightConfig)
	// 当前存在附加标记逻辑时，在排序前补齐 explain 来源。
	if marker != nil {
		marker(candidates)
	}
	stageContext := ApplyRankingStrategy(candidates, strategy, stageScores)
	rankedGoods := recommendCandidate.RankGoods(candidates)
	pageSnapshot := BuildRankedPageSnapshot(request, rankedGoods)
	explainSnapshot := BuildPageExplainSnapshot(pageSnapshot.PageGoods, candidates)
	return RankingResult{
		Candidates:      candidates,
		PageSnapshot:    pageSnapshot,
		ExplainSnapshot: explainSnapshot,
		StageContext:    stageContext,
	}
}
