package gorse

import (
	"context"
	"strconv"
	"time"

	_const "shop/pkg/const"

	commonv1 "shop/api/gen/go/common/v1"

	client "github.com/gorse-io/gorse-go"
)

// SessionReceiver 表示会话级Gorse 推荐接收器。
type SessionReceiver struct {
	recommend *Recommend
}

// NewSessionReceiver 创建会话级Gorse 推荐接收器。
func NewSessionReceiver(recommend *Recommend) *SessionReceiver {
	return &SessionReceiver{recommend: recommend}
}

// Enabled 判断当前会话级Gorse 推荐接收器是否可用。
func (r *SessionReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// ListGoodsIDs 查询当前会话前 N 条原始推荐商品编号。
func (r *SessionReceiver) ListGoodsIDs(ctx context.Context, contextGoodsIDs []int64, limit int64) ([]int64, bool, error) {
	// 客户端未启用或上下文商品为空时，直接返回空会话推荐结果。
	if !r.Enabled() || len(contextGoodsIDs) == 0 {
		return []int64{}, false, nil
	}
	// 请求上限非法时，直接返回空结果，避免Gorse 接口收到无效参数。
	if limit <= 0 {
		return []int64{}, false, nil
	}

	cleanGoodsIDs := make([]int64, 0, len(contextGoodsIDs))
	seenGoodsIDs := make(map[int64]struct{}, len(contextGoodsIDs))
	for _, goodsID := range contextGoodsIDs {
		// 非法商品编号或重复商品编号时，直接跳过当前无效值。
		if goodsID <= 0 {
			continue
		}
		// 同一个上下文商品只保留一次，避免会话推荐被重复反馈放大。
		if _, ok := seenGoodsIDs[goodsID]; ok {
			continue
		}
		seenGoodsIDs[goodsID] = struct{}{}
		cleanGoodsIDs = append(cleanGoodsIDs, goodsID)
	}
	// 清洗后没有有效上下文商品时，无需继续调用Gorse 推荐。
	if len(cleanGoodsIDs) == 0 {
		return []int64{}, false, nil
	}

	excludedGoods := make(map[int64]struct{}, len(cleanGoodsIDs))
	for _, goodsID := range cleanGoodsIDs {
		excludedGoods[goodsID] = struct{}{}
	}

	now := time.Now()
	feedbacks := make([]client.Feedback, 0, len(cleanGoodsIDs))
	for _, goodsID := range cleanGoodsIDs {
		feedbacks = append(feedbacks, client.Feedback{
			FeedbackType: commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_VIEW).String(),
			ItemId:       strconv.FormatInt(goodsID, 10),
			Value:        1,
			Timestamp:    now,
		})
	}

	scores, err := r.recommend.gorseClient.SessionRecommend(ctx, feedbacks, int(limit)+1)
	if err != nil {
		return nil, false, err
	}
	return r.recommend.buildGoodsIDsFromScores(scores, limit, excludedGoods)
}

// GetGoodsIDs 查询会话级推荐商品编号列表。
func (r *SessionReceiver) GetGoodsIDs(ctx context.Context, contextGoodsIDs []int64, pageNum, pageSize int64) ([]int64, int64, error) {
	limit := pageNum*pageSize + 1
	rawIDs, hasMore, err := r.ListGoodsIDs(ctx, contextGoodsIDs, limit)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildRecommendPageResult(rawIDs, hasMore, pageNum, pageSize)
}

// GetLatestGoodsIDs 查询最新商品推荐列表。
func (r *SessionReceiver) GetLatestGoodsIDs(ctx context.Context, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用时，直接返回空结果，由业务侧继续走本地兜底。
	if !r.Enabled() {
		return []int64{}, 0, nil
	}
	limit := pageNum*pageSize + 1
	// 请求上限非法时，直接返回空结果。
	if limit <= 0 {
		return []int64{}, 0, nil
	}

	scores, err := r.recommend.gorseClient.GetLatestItems(ctx, "", "", int(limit)+1, 0)
	if err != nil {
		return nil, 0, err
	}
	rawIDs := make([]string, 0, len(scores))
	for _, score := range scores {
		rawIDs = append(rawIDs, score.Id)
	}
	var goodsIDs []int64
	var hasMore bool
	goodsIDs, hasMore, err = r.recommend.buildRecommendGoodsIDs(rawIDs, limit)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildRecommendPageResult(goodsIDs, hasMore, pageNum, pageSize)
}
