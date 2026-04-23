package remote

import (
	"context"
	"strconv"

	"shop/pkg/recommend/dto"
)

// OnlineUserReceiver 表示登录用户在线推荐接收器。
type OnlineUserReceiver struct {
	recommend *Recommend
}

// NewOnlineUserReceiver 创建登录用户在线推荐接收器。
func NewOnlineUserReceiver(recommend *Recommend) *OnlineUserReceiver {
	return &OnlineUserReceiver{recommend: recommend}
}

// Enabled 判断当前登录用户在线推荐接收器是否可用。
func (r *OnlineUserReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// ListGoodsIds 查询当前用户前 N 条原始推荐商品编号。
func (r *OnlineUserReceiver) ListGoodsIds(ctx context.Context, actor *dto.RecommendActor, limit int64) ([]int64, bool, error) {
	// 客户端未启用、推荐主体无效或主体不是登录用户时，直接返回空推荐结果。
	if !r.Enabled() || !actor.IsValid() {
		return []int64{}, false, nil
	}
	// 匿名主体不走用户维度的推荐系统推荐。
	if !actor.IsUser() {
		return []int64{}, false, nil
	}
	// 请求上限非法时，直接返回空结果，避免远端收到无效参数。
	if limit <= 0 {
		return []int64{}, false, nil
	}

	rawIds, err := r.recommend.gorseClient.GetRecommend(ctx, strconv.FormatInt(actor.ActorId, 10), "", int(limit)+1, 0)
	if err != nil {
		return nil, false, err
	}
	return r.recommend.buildRecommendGoodsIds(rawIds, limit)
}

// GetGoodsIds 查询用户维度推荐商品编号列表。
func (r *OnlineUserReceiver) GetGoodsIds(ctx context.Context, actor *dto.RecommendActor, pageNum, pageSize int64) ([]int64, int64, error) {
	limit := pageNum*pageSize + 1
	rawIds, hasMore, err := r.ListGoodsIds(ctx, actor, limit)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildRecommendPageResult(rawIds, hasMore, pageNum, pageSize)
}
