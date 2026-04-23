package remote

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"shop/pkg/recommend/dto"

	client "github.com/gorse-io/gorse-go"
)

// NamedReceiver 表示命名推荐器接收器，负责邻近、命名召回和非个性化推荐。
type NamedReceiver struct {
	recommend *Recommend
}

// NewNamedReceiver 创建命名推荐器接收器。
func NewNamedReceiver(recommend *Recommend) *NamedReceiver {
	return &NamedReceiver{recommend: recommend}
}

// Enabled 判断当前命名推荐器接收器是否可用。
func (r *NamedReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// GetNeighborsGoodsIds 查询单商品邻近推荐列表。
func (r *NamedReceiver) GetNeighborsGoodsIds(ctx context.Context, goodsId, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用或商品编号非法时，直接返回空结果。
	if !r.Enabled() || goodsId <= 0 {
		return []int64{}, 0, nil
	}
	limit := pageNum*pageSize + 1
	// 请求上限非法时，直接返回空结果。
	if limit <= 0 {
		return []int64{}, 0, nil
	}

	scores, err := r.recommend.gorseClient.GetNeighbors(ctx, strconv.FormatInt(goodsId, 10), int(limit)+1)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildPageResultFromScores(scores, limit, pageNum, pageSize, map[int64]struct{}{goodsId: {}})
}

// GetItemToItemGoodsIds 查询命名 item-to-item 推荐器结果。
func (r *NamedReceiver) GetItemToItemGoodsIds(ctx context.Context, recommenderName string, goodsId, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用、推荐器名称为空或商品编号非法时，直接返回空结果。
	if !r.Enabled() || recommenderName == "" || goodsId <= 0 {
		return []int64{}, 0, nil
	}
	limit := pageNum*pageSize + 1
	// 请求上限非法时，直接返回空结果。
	if limit <= 0 {
		return []int64{}, 0, nil
	}

	path := fmt.Sprintf("/api/item-to-item/%s/%d?%s", url.PathEscape(recommenderName), goodsId, r.recommend.buildPaginationQuery(limit))
	scores, err := r.recommend.requestScores(ctx, path)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildPageResultFromScores(scores, limit, pageNum, pageSize, map[int64]struct{}{goodsId: {}})
}

// GetUserToUserGoodsIds 查询命名 user-to-user 推荐器结果。
func (r *NamedReceiver) GetUserToUserGoodsIds(ctx context.Context, recommenderName string, actor *dto.RecommendActor, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用、推荐器名称为空、主体为空或主体不是登录用户时，直接返回空结果。
	if !r.Enabled() || recommenderName == "" || !actor.IsValid() {
		return []int64{}, 0, nil
	}
	// user-to-user 只服务登录用户，匿名主体没有稳定的用户画像可供相似用户召回。
	if !actor.IsUser() {
		return []int64{}, 0, nil
	}
	limit := pageNum*pageSize + 1
	// 请求上限非法时，直接返回空结果。
	if limit <= 0 {
		return []int64{}, 0, nil
	}

	path := fmt.Sprintf("/api/user-to-user/%s/%d?%s", url.PathEscape(recommenderName), actor.ActorId, r.recommend.buildPaginationQuery(limit))
	scores, err := r.recommend.requestScores(ctx, path)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildPageResultFromScores(scores, limit, pageNum, pageSize, nil)
}

// GetNonPersonalizedGoodsIds 查询命名非个性化推荐器结果。
func (r *NamedReceiver) GetNonPersonalizedGoodsIds(ctx context.Context, recommenderName string, pageNum, pageSize int64) ([]int64, int64, error) {
	// 客户端未启用或推荐器名称为空时，直接返回空结果。
	if !r.Enabled() || recommenderName == "" {
		return []int64{}, 0, nil
	}
	limit := pageNum*pageSize + 1
	// 请求上限非法时，直接返回空结果。
	if limit <= 0 {
		return []int64{}, 0, nil
	}

	path := fmt.Sprintf("/api/non-personalized/%s?%s", url.PathEscape(recommenderName), r.recommend.buildPaginationQuery(limit))
	scores, err := r.recommend.requestScores(ctx, path)
	if err != nil {
		return nil, 0, err
	}
	return r.recommend.buildPageResultFromScores(scores, limit, pageNum, pageSize, nil)
}

// resolveAnchorGoodsId 解析当前远程推荐使用的锚点商品编号。
func (r *Recommend) resolveAnchorGoodsId(goodsId int64, contextGoodsIds []int64) int64 {
	// 当前请求显式传入锚点商品时，优先使用该商品。
	if goodsId > 0 {
		return goodsId
	}
	for _, itemId := range contextGoodsIds {
		// 上下文中命中首个有效商品编号时，直接将其作为锚点商品。
		if itemId > 0 {
			return itemId
		}
	}
	return 0
}

// buildPaginationQuery 构建命名推荐器统一分页查询参数。
func (r *Recommend) buildPaginationQuery(limit int64) string {
	query := url.Values{}
	query.Set("n", strconv.FormatInt(limit+1, 10))
	query.Set("offset", "0")
	return query.Encode()
}

// buildPageResultFromScores 将评分列表转换为项目分页结果。
func (r *Recommend) buildPageResultFromScores(scores []client.Score, limit, pageNum, pageSize int64, excludedGoods map[int64]struct{}) ([]int64, int64, error) {
	goodsIds, hasMore, err := r.buildGoodsIdsFromScores(scores, limit, excludedGoods)
	if err != nil {
		return nil, 0, err
	}
	return r.buildRecommendPageResult(goodsIds, hasMore, pageNum, pageSize)
}

// buildGoodsIdsFromScores 清洗评分列表中的商品编号。
func (r *Recommend) buildGoodsIdsFromScores(scores []client.Score, limit int64, excludedGoods map[int64]struct{}) ([]int64, bool, error) {
	rawIds := make([]string, 0, len(scores))
	for _, score := range scores {
		goodsId, err := strconv.ParseInt(score.Id, 10, 64)
		// 推荐系统返回了非法商品编号时，直接跳过当前无效结果。
		if err != nil || goodsId <= 0 {
			continue
		}
		// 命中排除商品本身时，直接过滤，避免把上下文商品再次返回给前端。
		if len(excludedGoods) > 0 {
			if _, ok := excludedGoods[goodsId]; ok {
				continue
			}
		}
		rawIds = append(rawIds, score.Id)
	}
	return r.buildRecommendGoodsIds(rawIds, limit)
}
