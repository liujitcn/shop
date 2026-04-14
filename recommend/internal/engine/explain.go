package engine

import (
	"context"
	"errors"
	"fmt"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	cachex "recommend/internal/cache"
	cacheleveldb "recommend/internal/cache/leveldb"
	"recommend/internal/core"
	"recommend/internal/model"
	"time"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Explain 查询已持久化的推荐追踪结果。
func Explain(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.ExplainRequest) (*core.ExplainResult, error) {
	manager, err := openCacheManager(ctx, dependencies)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.TraceStore{Driver: manager}
	traceId := request.TraceId
	requestId := request.RequestId
	var detail *recommendv1.RecommendTraceDetail

	// 优先按追踪编号精确查询，便于直接通过推荐结果返回的 traceId 回查。
	if traceId != "" {
		detail, err = store.GetTraceDetail(traceId)
		if err == nil {
			return buildExplainResult(traceId, detail), nil
		}
		// 追踪编号命中不存在时，如果没有请求编号可回退，则直接返回原始错误。
		if !errors.Is(err, goleveldb.ErrNotFound) || requestId == "" {
			return nil, err
		}
	}

	// 未提供追踪编号或追踪编号不存在时，允许按请求编号回查最后一次已保存的追踪详情。
	if requestId == "" {
		return nil, errors.New("recommend: explain 请求必须提供 traceId 或 requestId")
	}

	detail, err = store.GetTraceDetailByRequestId(requestId)
	if err != nil {
		return nil, err
	}
	if traceId == "" {
		traceId = requestId
	}
	return buildExplainResult(traceId, detail), nil
}

// saveRecommendTrace 保存在线推荐产生的追踪详情。
func saveRecommendTrace(
	ctx context.Context,
	dependencies core.Dependencies,
	config core.ServiceConfig,
	request model.Request,
	traceId string,
	candidates []*model.Candidate,
	currentPage []*model.Candidate,
) error {
	// 没有可用追踪编号时，不保存 explain 明细，避免写入无法回查的数据。
	if traceId == "" {
		return nil
	}
	// 当前未配置缓存数据源时，只跳过 trace 持久化，不阻塞推荐主链路。
	if dependencies.Cache == nil {
		return nil
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.TraceStore{Driver: manager}
	return store.SaveTraceDetail(traceId, request.Context.RequestId, buildTraceDetail(config, request, candidates, currentPage))
}

// appendTraceStep 追加一条追踪步骤。
func appendTraceStep(
	ctx context.Context,
	dependencies core.Dependencies,
	requestId string,
	stage string,
	reason string,
	goodsIds []int64,
) error {
	// 没有请求编号时，无法定位需要补充的 trace 记录。
	if requestId == "" {
		return nil
	}

	manager, err := openCacheManager(ctx, dependencies)
	if err != nil {
		return err
	}
	defer func() {
		_ = manager.Close()
	}()

	return appendTraceStepByStore(&cachex.TraceStore{Driver: manager}, requestId, stage, reason, goodsIds)
}

// appendTraceStepByStore 基于已打开的追踪缓存写入一条追踪步骤。
func appendTraceStepByStore(
	store *cachex.TraceStore,
	requestId string,
	stage string,
	reason string,
	goodsIds []int64,
) error {
	detail, err := store.GetTraceDetailByRequestId(requestId)
	if err != nil {
		// 推荐主链路可能尚未开启 trace 持久化；这种情况下跳过补充，不视为同步失败。
		if errors.Is(err, goleveldb.ErrNotFound) {
			return nil
		}
		return err
	}

	detail.Steps = append(detail.Steps, &recommendv1.RecommendTraceStep{
		Stage:    stage,
		Reason:   reason,
		GoodsIds: append([]int64(nil), goodsIds...),
	})
	detail.Meta = buildCacheMeta(detail.GetScene(), detail.GetMeta().GetActorType(), detail.GetMeta().GetActorId(), time.Now())
	return store.SaveTraceDetail(requestId, detail.GetRequestId(), detail)
}

// openCacheManager 打开推荐模块所需的 LevelDB 管理器。
func openCacheManager(ctx context.Context, dependencies core.Dependencies) (*cacheleveldb.Manager, error) {
	// explain 和运行态同步都依赖缓存持久化，未配置缓存数据源时直接返回错误。
	if dependencies.Cache == nil {
		return nil, errors.New("recommend: 缓存数据源未配置")
	}
	return cacheleveldb.OpenManager(ctx, dependencies.Cache)
}

// buildExplainResult 将缓存中的追踪详情转换为对外结果。
func buildExplainResult(traceId string, detail *recommendv1.RecommendTraceDetail) *core.ExplainResult {
	result := &core.ExplainResult{
		TraceId:        traceId,
		Scene:          core.Scene(detail.GetScene()),
		ResultGoodsIds: append([]int64(nil), detail.GetResultGoodsIds()...),
	}
	if len(detail.GetSteps()) > 0 {
		result.Steps = make([]core.TraceStep, 0, len(detail.GetSteps()))
		for _, item := range detail.GetSteps() {
			// 空步骤不会给调用方带来有效 explain 信息，直接跳过。
			if item == nil {
				continue
			}
			result.Steps = append(result.Steps, core.TraceStep{
				Stage:    item.GetStage(),
				Reason:   item.GetReason(),
				GoodsIds: append([]int64(nil), item.GetGoodsIds()...),
			})
		}
	}
	if len(detail.GetScoreDetails()) > 0 {
		result.ScoreDetails = make([]core.ScoreDetail, 0, len(detail.GetScoreDetails()))
		for _, item := range detail.GetScoreDetails() {
			// 空评分明细没有业务价值，不继续暴露给 explain 调用方。
			if item == nil {
				continue
			}
			result.ScoreDetails = append(result.ScoreDetails, core.ScoreDetail{
				GoodsId:            item.GetGoodsId(),
				FinalScore:         item.GetFinalScore(),
				RelationScore:      item.GetRelationScore(),
				UserGoodsScore:     item.GetUserGoodsScore(),
				CategoryScore:      item.GetCategoryScore(),
				SceneHotScore:      item.GetSceneHotScore(),
				GlobalHotScore:     item.GetGlobalHotScore(),
				FreshnessScore:     item.GetFreshnessScore(),
				SessionScore:       item.GetSessionScore(),
				ExternalScore:      item.GetExternalScore(),
				CollaborativeScore: item.GetCollaborativeScore(),
				UserNeighborScore:  item.GetUserNeighborScore(),
				VectorScore:        item.GetVectorScore(),
				ExposurePenalty:    item.GetExposurePenalty(),
				RepeatPenalty:      item.GetRepeatPenalty(),
				RuleScore:          item.GetRuleScore(),
				FmScore:            item.GetFmScore(),
				LlmScore:           item.GetLlmScore(),
				RecallSources:      append([]string(nil), item.GetRecallSources()...),
			})
		}
	}
	return result
}

// buildTraceDetail 构建一次在线推荐的追踪详情。
func buildTraceDetail(
	config core.ServiceConfig,
	request model.Request,
	candidates []*model.Candidate,
	currentPage []*model.Candidate,
) *recommendv1.RecommendTraceDetail {
	return &recommendv1.RecommendTraceDetail{
		Meta:           buildCacheMeta(request.Scene.String(), int32(request.Actor.Type), request.Actor.Id, time.Now()),
		RequestId:      request.Context.RequestId,
		Scene:          request.Scene.String(),
		Steps:          buildTraceSteps(config, candidates, currentPage),
		ScoreDetails:   buildTraceScoreDetails(candidates),
		ResultGoodsIds: buildCandidateGoodsIds(currentPage),
	}
}

// buildTraceSteps 生成推荐 explain 使用的步骤列表。
func buildTraceSteps(config core.ServiceConfig, candidates []*model.Candidate, currentPage []*model.Candidate) []*recommendv1.RecommendTraceStep {
	rankingStageReason := "场景流水线产出的最终候选列表"
	// 当实例启用了二阶段排序时，把具体模式写入 trace，便于 explain 明确说明最终结果来自哪条排序链路。
	if config.Ranking.Mode != "" && config.Ranking.Mode != core.RankingModeRule {
		rankingStageReason = fmt.Sprintf("场景流水线产出的最终候选列表，当前排序模式=%s", config.Ranking.Mode)
	}
	return []*recommendv1.RecommendTraceStep{
		{
			Stage:    "ranked_candidates",
			Reason:   rankingStageReason,
			GoodsIds: buildCandidateGoodsIds(candidates),
		},
		{
			Stage:    "page_result",
			Reason:   "当前分页返回结果",
			GoodsIds: buildCandidateGoodsIds(currentPage),
		},
	}
}

// buildTraceScoreDetails 生成追踪明细中的评分详情。
func buildTraceScoreDetails(candidates []*model.Candidate) []*recommendv1.RecommendScoreDetail {
	details := make([]*recommendv1.RecommendScoreDetail, 0, len(candidates))
	for _, item := range candidates {
		// 空候选或缺失商品实体时，无法输出有效评分明细。
		if item == nil || item.Goods == nil {
			continue
		}
		details = append(details, &recommendv1.RecommendScoreDetail{
			GoodsId:            item.GoodsId(),
			FinalScore:         item.Score.FinalScore,
			RelationScore:      item.Score.RelationScore,
			UserGoodsScore:     item.Score.UserGoodsScore,
			CategoryScore:      item.Score.CategoryScore,
			SceneHotScore:      item.Score.SceneHotScore,
			GlobalHotScore:     item.Score.GlobalHotScore,
			FreshnessScore:     item.Score.FreshnessScore,
			SessionScore:       item.Score.SessionScore,
			ExternalScore:      item.Score.ExternalScore,
			CollaborativeScore: item.Score.CollaborativeScore,
			UserNeighborScore:  item.Score.UserNeighborScore,
			VectorScore:        item.Score.VectorScore,
			ExposurePenalty:    item.Score.ExposurePenalty,
			RepeatPenalty:      item.Score.RepeatPenalty,
			RuleScore:          item.Score.RuleScore,
			FmScore:            item.Score.FmScore,
			LlmScore:           item.Score.LlmScore,
			RecallSources:      item.RecallSourceList(),
		})
	}
	return details
}

// buildCandidateGoodsIds 提取候选列表中的商品编号。
func buildCandidateGoodsIds(candidates []*model.Candidate) []int64 {
	goodsIds := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		goodsId := item.GoodsId()
		// 非法商品编号不进入 trace 输出，避免 explain 中出现无效条目。
		if goodsId <= 0 {
			continue
		}
		goodsIds = append(goodsIds, goodsId)
	}
	return goodsIds
}

// buildCacheMeta 构建缓存消息统一使用的元信息。
func buildCacheMeta(scene string, actorType int32, actorId int64, updatedAt time.Time) *recommendv1.CacheMeta {
	// 调用方未显式传入更新时间时，统一回退到当前时间。
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	return &recommendv1.CacheMeta{
		SchemaVersion: "v1",
		Scene:         scene,
		ActorType:     actorType,
		ActorId:       actorId,
		UpdatedAt:     timestamppb.New(updatedAt),
	}
}

// buildGeneratedTraceId 生成兜底使用的追踪编号。
func buildGeneratedTraceId() string {
	return fmt.Sprintf("trace-%d", time.Now().UnixNano())
}
