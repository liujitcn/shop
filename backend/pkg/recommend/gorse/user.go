package gorse

import (
	"context"
	"strconv"

	"shop/pkg/recommend/dto"
)

// UserReceiver 表示登录用户Gorse 推荐接收器。
type UserReceiver struct {
	recommend *Recommend
}

// NewUserReceiver 创建登录用户Gorse 推荐接收器。
func NewUserReceiver(recommend *Recommend) *UserReceiver {
	return &UserReceiver{recommend: recommend}
}

// Enabled 判断当前登录用户Gorse 推荐接收器是否可用。
func (r *UserReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// ListGoodsIDs 查询当前用户前 N 条原始推荐商品编号。
func (r *UserReceiver) ListGoodsIDs(ctx context.Context, actor *dto.RecommendActor, limit int64) ([]int64, bool, error) {
	// 客户端未启用、推荐主体无效或主体不是登录用户时，直接返回空推荐结果。
	if !r.Enabled() || !actor.IsValid() {
		return []int64{}, false, nil
	}
	// 匿名主体不走用户维度的推荐系统推荐。
	if !actor.IsUser() {
		return []int64{}, false, nil
	}
	// 请求上限非法时，直接返回空结果，避免Gorse收到无效参数。
	if limit <= 0 {
		return []int64{}, false, nil
	}

	rawIDs, err := r.recommend.gorseClient.GetRecommend(ctx, strconv.FormatInt(actor.ActorID, 10), "", int(limit)+1, 0)
	if err != nil {
		return nil, false, err
	}
	return r.recommend.buildRecommendGoodsIDs(rawIDs, limit)
}

// GetGoodsIDs 查询用户维度推荐商品编号列表。
func (r *UserReceiver) GetGoodsIDs(ctx context.Context, actor *dto.RecommendActor, pageNum, pageSize int64) ([]int64, int64, error) {
	limit := pageNum*pageSize + 1
	rawIDs, hasMore, err := r.ListGoodsIDs(ctx, actor, limit)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildRecommendPageResult(rawIDs, hasMore, pageNum, pageSize)
}
