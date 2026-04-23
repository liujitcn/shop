package remote

import (
	"context"
	"strconv"
	"time"

	"shop/api/gen/go/common"

	client "github.com/gorse-io/gorse-go"
)

// OnlineSessionReceiver 表示会话级在线推荐接收器。
type OnlineSessionReceiver struct {
	recommend *Recommend
}

// NewOnlineSessionReceiver 创建会话级在线推荐接收器。
func NewOnlineSessionReceiver(recommend *Recommend) *OnlineSessionReceiver {
	return &OnlineSessionReceiver{recommend: recommend}
}

// Enabled 判断当前会话级在线推荐接收器是否可用。
func (r *OnlineSessionReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// ListGoodsIds 查询当前会话前 N 条原始推荐商品编号。
func (r *OnlineSessionReceiver) ListGoodsIds(ctx context.Context, contextGoodsIds []int64, limit int64) ([]int64, bool, error) {
	// 客户端未启用或上下文商品为空时，直接返回空会话推荐结果。
	if !r.Enabled() || len(contextGoodsIds) == 0 {
		return []int64{}, false, nil
	}
	// 请求上限非法时，直接返回空结果，避免远端接口收到无效参数。
	if limit <= 0 {
		return []int64{}, false, nil
	}

	cleanGoodsIds := r.cleanContextGoodsIds(contextGoodsIds)
	// 清洗后没有有效上下文商品时，无需继续调用远端推荐。
	if len(cleanGoodsIds) == 0 {
		return []int64{}, false, nil
	}

	excludedGoods := make(map[int64]struct{}, len(cleanGoodsIds))
	for _, goodsId := range cleanGoodsIds {
		excludedGoods[goodsId] = struct{}{}
	}

	now := time.Now()
	feedbacks := make([]client.Feedback, 0, len(cleanGoodsIds))
	for _, goodsId := range cleanGoodsIds {
		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: common.RecommendEventType_VIEW.String(),
			ItemId:       strconv.FormatInt(goodsId, 10),
			Value:        1,
			Timestamp:    now,
		})
	}

	scores, err := r.recommend.gorseClient.SessionRecommend(r.recommend.defaultContext(ctx), feedbacks, int(limit)+1)
	if err != nil {
		return nil, false, err
	}
	return r.recommend.buildGoodsIdsFromScores(scores, limit, excludedGoods)
}

// GetGoodsIds 查询会话级推荐商品编号列表。
func (r *OnlineSessionReceiver) GetGoodsIds(ctx context.Context, contextGoodsIds []int64, pageNum, pageSize int64) ([]int64, int64, error) {
	limit := pageNum*pageSize + 1
	rawIds, hasMore, err := r.ListGoodsIds(ctx, contextGoodsIds, limit)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildRecommendPageResult(rawIds, hasMore, pageNum, pageSize)
}

// GetLatestGoodsIds 查询最新商品推荐列表。
func (r *OnlineSessionReceiver) GetLatestGoodsIds(ctx context.Context, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用时，直接返回空结果，由业务侧继续走本地兜底。
	if !r.Enabled() {
		return []int64{}, 0, nil
	}
	limit := pageNum*pageSize + 1
	// 请求上限非法时，直接返回空结果。
	if limit <= 0 {
		return []int64{}, 0, nil
	}

	scores, err := r.recommend.gorseClient.GetLatestItems(r.recommend.defaultContext(ctx), "", "", int(limit)+1, 0)
	if err != nil {
		return nil, 0, err
	}
	rawIds := make([]string, 0, len(scores))
	for _, score := range scores {
		rawIds = append(rawIds, score.Id)
	}
	goodsIds, hasMore, err := r.recommend.buildRecommendGoodsIds(rawIds, limit)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildRecommendPageResult(goodsIds, hasMore, pageNum, pageSize)
}

// cleanContextGoodsIds 清洗会话上下文商品编号列表。
func (r *OnlineSessionReceiver) cleanContextGoodsIds(contextGoodsIds []int64) []int64 {
	cleanGoodsIds := make([]int64, 0, len(contextGoodsIds))
	excludedGoods := make(map[int64]struct{}, len(contextGoodsIds))
	for _, goodsId := range contextGoodsIds {
		// 非法商品编号或重复商品编号时，直接跳过当前无效值。
		if goodsId <= 0 {
			continue
		}
		// 同一个上下文商品只保留一次，避免会话推荐被重复反馈放大。
		if _, ok := excludedGoods[goodsId]; ok {
			continue
		}
		excludedGoods[goodsId] = struct{}{}
		cleanGoodsIds = append(cleanGoodsIds, goodsId)
	}
	return cleanGoodsIds
}
