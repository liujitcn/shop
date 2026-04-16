package materialize

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"time"

	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
)

// Materializer 负责把离线聚合结果写入推荐缓存。
type Materializer struct {
	store recommendCache.Store // 推荐缓存存储实现。
}

// NewMaterializer 创建推荐缓存写入服务。
func NewMaterializer(store recommendCache.Store) *Materializer {
	// 推荐缓存必须显式注入真实实现，避免静默回退到空对象。
	if store == nil {
		panic("recommend cache store is nil")
	}
	return &Materializer{
		store: store,
	}
}

// MaterializeSceneHot 发布场景热门榜缓存。
func (m *Materializer) MaterializeSceneHot(ctx context.Context, scene int32, version string, list []*models.RecommendGoodsStatDay) error {
	documents := make([]recommendCache.Score, 0, len(list))
	for _, item := range list {
		// 非法统计记录不参与热门榜缓存发布。
		if item == nil || item.GoodsID <= 0 || item.Scene != scene {
			continue
		}
		documents = append(documents, recommendCache.Score{
			Id:        strconv.FormatInt(item.GoodsID, 10),
			Score:     item.Score,
			Timestamp: item.UpdatedAt,
		})
	}
	return m.publishScores(ctx, recommendCache.NonPersonalized, recommendCache.SceneHotSubset(scene, version), documents)
}

// MaterializeSceneLatest 发布场景最新榜缓存。
func (m *Materializer) MaterializeSceneLatest(ctx context.Context, scene int32, version string, list []*models.GoodsInfo) error {
	documents := make([]recommendCache.Score, 0, len(list))
	for _, item := range list {
		// 非法商品不参与最新榜缓存发布。
		if item == nil || item.ID <= 0 {
			continue
		}
		scoreTime := item.CreatedAt
		// 创建时间缺失时，回退到更新时间，避免旧数据无法进入最新榜。
		if scoreTime.IsZero() {
			scoreTime = item.UpdatedAt
		}
		documents = append(documents, recommendCache.Score{
			Id:         strconv.FormatInt(item.ID, 10),
			Score:      float64(scoreTime.Unix()),
			Categories: []string{strconv.FormatInt(item.CategoryID, 10)},
			Timestamp:  scoreTime,
		})
	}
	return m.publishScores(ctx, recommendCache.NonPersonalized, recommendCache.SceneLatestSubset(scene, version), documents)
}

// MaterializeSimilarItems 发布相似商品缓存。
func (m *Materializer) MaterializeSimilarItems(ctx context.Context, goodsId int64, version string, list []*models.RecommendGoodsRelation) error {
	documents := make([]recommendCache.Score, 0, len(list))
	for _, item := range list {
		// 非法关系或非目标主商品的关系记录不参与缓存发布。
		if item == nil || item.GoodsID != goodsId || item.RelatedGoodsID <= 0 {
			continue
		}
		documents = append(documents, recommendCache.Score{
			Id:        strconv.FormatInt(item.RelatedGoodsID, 10),
			Score:     item.Score,
			Timestamp: item.UpdatedAt,
		})
	}
	return m.publishScores(ctx, recommendCache.ItemToItem, recommendCache.SimilarItemSubset(goodsId, version), documents)
}

// MaterializeSimilarUsers 发布相似用户缓存。
func (m *Materializer) MaterializeSimilarUsers(ctx context.Context, userId int64, version string, documents []recommendCache.Score) error {
	// 用户编号非法时，不继续发布相似用户缓存。
	if userId <= 0 {
		return nil
	}
	return m.publishScores(ctx, recommendCache.UserToUser, recommendCache.SimilarUserSubset(userId, version), documents)
}

// MaterializeCollaborativeFiltering 发布协同过滤缓存。
func (m *Materializer) MaterializeCollaborativeFiltering(ctx context.Context, userId int64, version string, documents []recommendCache.Score) error {
	// 用户编号非法时，不继续发布协同过滤缓存。
	if userId <= 0 {
		return nil
	}
	return m.publishScores(ctx, recommendCache.CollaborativeFiltering, recommendCache.CollaborativeFilteringSubset(userId, version), documents)
}

// MaterializeContentBased 发布内容相似缓存。
func (m *Materializer) MaterializeContentBased(ctx context.Context, goodsId int64, version string, documents []recommendCache.Score) error {
	// 商品编号非法时，不继续发布内容相似缓存。
	if goodsId <= 0 {
		return nil
	}
	return m.publishScores(ctx, recommendCache.ContentBased, recommendCache.ContentBasedSubset(goodsId, version), documents)
}

// publishScores 写入排序型缓存并补齐摘要与更新时间。
func (m *Materializer) publishScores(ctx context.Context, collection, subset string, documents []recommendCache.Score) error {
	collectionKey := recommendCache.CollectionKey(collection)
	if len(documents) == 0 {
		return m.clearSubset(ctx, collectionKey, collection, subset)
	}

	recommendCache.SortDocuments(documents)
	err := m.store.AddScores(ctx, collectionKey, subset, documents)
	if err != nil {
		return err
	}
	err = m.store.Set(recommendCache.DigestKey(collection, subset), buildDigest(documents), 0)
	if err != nil {
		return err
	}
	err = m.store.Set(recommendCache.DocumentCountKey(collection, subset), strconv.Itoa(len(documents)), 0)
	if err != nil {
		return err
	}
	return m.store.Set(recommendCache.UpdateTimeKey(collection, subset), time.Now().Format(time.RFC3339Nano), 0)
}

// clearSubset 清理空缓存结果对应的子集合和元信息。
func (m *Materializer) clearSubset(ctx context.Context, collectionKey, collection, subset string) error {
	err := m.store.DeleteScores(ctx, []string{collectionKey}, recommendCache.ScoreCondition{Subset: &subset})
	if err != nil {
		return err
	}
	err = m.store.Del(recommendCache.DigestKey(collection, subset))
	if err != nil {
		return err
	}
	err = m.store.Del(recommendCache.DocumentCountKey(collection, subset))
	if err != nil {
		return err
	}
	return m.store.Del(recommendCache.UpdateTimeKey(collection, subset))
}

// buildDigest 根据排序型缓存内容生成稳定摘要。
func buildDigest(documents []recommendCache.Score) string {
	cloned := make([]recommendCache.Score, 0, len(documents))
	cloned = append(cloned, documents...)
	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Id == cloned[j].Id {
			return cloned[i].Score > cloned[j].Score
		}
		return cloned[i].Id < cloned[j].Id
	})

	hasher := sha1.New()
	for _, item := range cloned {
		_, _ = hasher.Write([]byte(fmt.Sprintf("%s|%.6f|%d;", item.Id, item.Score, item.Timestamp.UnixNano())))
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
