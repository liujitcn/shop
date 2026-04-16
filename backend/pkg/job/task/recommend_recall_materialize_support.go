package task

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendEvent "shop/pkg/recommend/event"

	"github.com/liujitcn/gorm-kit/repo"
)

const (
	// recommendSimilarUserGoodsWeight 表示商品偏好重叠在相似用户中的权重。
	recommendSimilarUserGoodsWeight = 1.5
	// recommendContentPriceWeight 表示价格接近度在内容相似中的权重。
	recommendContentPriceWeight = 0.7
	// recommendContentFreshnessWeight 表示新鲜度接近度在内容相似中的权重。
	recommendContentFreshnessWeight = 0.3
)

// recommendNeighborScore 表示用户邻居分数及更新时间。
type recommendNeighborScore struct {
	Score     float64
	UpdatedAt time.Time
}

// recommendGoodsScore 表示商品分数及更新时间。
type recommendGoodsScore struct {
	Score     float64
	UpdatedAt time.Time
}

// loadEnabledRecommendVersions 查询当前启用的推荐版本列表。
func loadEnabledRecommendVersions(ctx context.Context, recommendModelVersionRepo *data.RecommendModelVersionRepo) ([]string, error) {
	sceneList := listRecommendMaterializeScenes()
	versionMap, err := loadEnabledRecommendVersionMap(ctx, recommendModelVersionRepo, sceneList)
	if err != nil {
		return nil, err
	}

	versionSet := make(map[string]struct{}, len(versionMap))
	versionList := make([]string, 0, len(versionMap))
	for _, version := range versionMap {
		normalizedVersion := recommendCache.NormalizeVersion(version)
		_, exists := versionSet[normalizedVersion]
		// 相同版本只保留一份，避免重复发布相同缓存空间。
		if exists {
			continue
		}
		versionSet[normalizedVersion] = struct{}{}
		versionList = append(versionList, normalizedVersion)
	}
	sort.Strings(versionList)
	return versionList, nil
}

// loadRecommendCategoryPreferenceList 加载用户类目偏好训练输入。
func loadRecommendCategoryPreferenceList(ctx context.Context, recommendUserPreferenceRepo *data.RecommendUserPreferenceRepo) ([]*models.RecommendUserPreference, error) {
	query := recommendUserPreferenceRepo.Query(ctx).RecommendUserPreference
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.PreferenceType.Eq(recommendEvent.PreferenceTypeCategory)))
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	return recommendUserPreferenceRepo.List(ctx, opts...)
}

// loadRecommendUserGoodsPreferenceList 加载用户商品偏好训练输入。
func loadRecommendUserGoodsPreferenceList(ctx context.Context, recommendUserGoodsPreferenceRepo *data.RecommendUserGoodsPreferenceRepo) ([]*models.RecommendUserGoodsPreference, error) {
	query := recommendUserGoodsPreferenceRepo.Query(ctx).RecommendUserGoodsPreference
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	return recommendUserGoodsPreferenceRepo.List(ctx, opts...)
}

// loadPutOnGoodsList 查询当前全部上架商品。
func loadPutOnGoodsList(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo) ([]*models.GoodsInfo, error) {
	query := goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	return goodsInfoRepo.List(ctx, opts...)
}

// buildSimilarUserScoreMap 基于类目偏好和商品偏好构建相似用户分数。
func buildSimilarUserScoreMap(
	categoryPreferenceList []*models.RecommendUserPreference,
	userGoodsPreferenceList []*models.RecommendUserGoodsPreference,
) map[int64]map[int64]recommendNeighborScore {
	scoreMap := make(map[int64]map[int64]recommendNeighborScore)

	categoryIndex := make(map[int64][]*models.RecommendUserPreference)
	for _, item := range categoryPreferenceList {
		// 非法用户、类目或分数记录不参与相似用户训练。
		if item == nil || item.UserID <= 0 || item.TargetID <= 0 || item.Score <= 0 {
			continue
		}
		categoryIndex[item.TargetID] = append(categoryIndex[item.TargetID], item)
	}
	for _, list := range categoryIndex {
		buildCategoryNeighborScores(scoreMap, list)
	}

	goodsIndex := make(map[int64][]*models.RecommendUserGoodsPreference)
	for _, item := range userGoodsPreferenceList {
		// 非法用户、商品或分数记录不参与相似用户训练。
		if item == nil || item.UserID <= 0 || item.GoodsID <= 0 || item.Score <= 0 {
			continue
		}
		goodsIndex[item.GoodsID] = append(goodsIndex[item.GoodsID], item)
	}
	for _, list := range goodsIndex {
		buildGoodsNeighborScores(scoreMap, list)
	}
	return scoreMap
}

// buildCategoryNeighborScores 基于同类目偏好重叠累计相似用户分数。
func buildCategoryNeighborScores(scoreMap map[int64]map[int64]recommendNeighborScore, list []*models.RecommendUserPreference) {
	for index := 0; index < len(list); index++ {
		left := list[index]
		// 左侧偏好为空时，直接跳过当前比较起点。
		if left == nil {
			continue
		}
		for nextIndex := index + 1; nextIndex < len(list); nextIndex++ {
			right := list[nextIndex]
			// 右侧偏好为空或命中同一用户时，不累计相似度。
			if right == nil || left.UserID == right.UserID {
				continue
			}
			score := math.Min(left.Score, right.Score)
			// 同类目重叠得分非法时，不累计相似度。
			if score <= 0 {
				continue
			}
			updatedAt := maxRecommendTime(left.UpdatedAt, right.UpdatedAt)
			accumulateNeighborScore(scoreMap, left.UserID, right.UserID, score, updatedAt)
		}
	}
}

// buildGoodsNeighborScores 基于同商品偏好重叠累计相似用户分数。
func buildGoodsNeighborScores(scoreMap map[int64]map[int64]recommendNeighborScore, list []*models.RecommendUserGoodsPreference) {
	for index := 0; index < len(list); index++ {
		left := list[index]
		// 左侧偏好为空时，直接跳过当前比较起点。
		if left == nil {
			continue
		}
		for nextIndex := index + 1; nextIndex < len(list); nextIndex++ {
			right := list[nextIndex]
			// 右侧偏好为空或命中同一用户时，不累计相似度。
			if right == nil || left.UserID == right.UserID {
				continue
			}
			score := math.Min(left.Score, right.Score) * recommendSimilarUserGoodsWeight
			// 同商品重叠得分非法时，不累计相似度。
			if score <= 0 {
				continue
			}
			updatedAt := maxRecommendTime(left.UpdatedAt, right.UpdatedAt)
			accumulateNeighborScore(scoreMap, left.UserID, right.UserID, score, updatedAt)
		}
	}
}

// accumulateNeighborScore 为两个用户对称累计邻居分数。
func accumulateNeighborScore(scoreMap map[int64]map[int64]recommendNeighborScore, leftUserId, rightUserId int64, score float64, updatedAt time.Time) {
	// 用户编号非法或分数非法时，不继续累计相似度。
	if leftUserId <= 0 || rightUserId <= 0 || score <= 0 {
		return
	}
	appendNeighborScore(scoreMap, leftUserId, rightUserId, score, updatedAt)
	appendNeighborScore(scoreMap, rightUserId, leftUserId, score, updatedAt)
}

// appendNeighborScore 追加单向邻居分数。
func appendNeighborScore(scoreMap map[int64]map[int64]recommendNeighborScore, userId, neighborUserId int64, score float64, updatedAt time.Time) {
	neighborScoreMap, exists := scoreMap[userId]
	// 当前用户还没有邻居分数表时，先初始化容器。
	if !exists {
		neighborScoreMap = make(map[int64]recommendNeighborScore)
		scoreMap[userId] = neighborScoreMap
	}
	current := neighborScoreMap[neighborUserId]
	current.Score += score
	current.UpdatedAt = maxRecommendTime(current.UpdatedAt, updatedAt)
	neighborScoreMap[neighborUserId] = current
}

// buildSimilarUserMaterializeMap 将相似用户分数转换为写缓存文档。
func buildSimilarUserMaterializeMap(scoreMap map[int64]map[int64]recommendNeighborScore, limit int64) map[int64][]recommendCache.Score {
	result := make(map[int64][]recommendCache.Score, len(scoreMap))
	for userId, neighborScoreMap := range scoreMap {
		documentList := make([]recommendCache.Score, 0, len(neighborScoreMap))
		for neighborUserId, item := range neighborScoreMap {
			// 邻居编号或相似度非法时，不写入相似用户缓存。
			if neighborUserId <= 0 || item.Score <= 0 {
				continue
			}
			documentList = append(documentList, recommendCache.Score{
				Id:        strconv.FormatInt(neighborUserId, 10),
				Score:     item.Score,
				Timestamp: item.UpdatedAt,
			})
		}
		recommendCache.SortDocuments(documentList)
		// 相似用户结果超过上限时，只保留 TopN。
		if int64(len(documentList)) > limit {
			documentList = documentList[:limit]
		}
		result[userId] = documentList
	}
	return result
}

// buildCollaborativeFilteringMaterializeMap 基于相似用户和用户商品偏好构建协同过滤结果。
func buildCollaborativeFilteringMaterializeMap(
	ctx context.Context,
	similarUserMap map[int64][]recommendCache.Score,
	userGoodsPreferenceList []*models.RecommendUserGoodsPreference,
	goodsInfoRepo *data.GoodsInfoRepo,
	limit int64,
) (map[int64][]recommendCache.Score, error) {
	goodsPreferenceByUser := make(map[int64][]*models.RecommendUserGoodsPreference)
	ownedGoodsMapByUser := make(map[int64]map[int64]struct{})
	for _, item := range userGoodsPreferenceList {
		// 非法用户、商品或分数记录不参与协同过滤构建。
		if item == nil || item.UserID <= 0 || item.GoodsID <= 0 || item.Score <= 0 {
			continue
		}
		goodsPreferenceByUser[item.UserID] = append(goodsPreferenceByUser[item.UserID], item)
		ownedGoodsMap, exists := ownedGoodsMapByUser[item.UserID]
		// 用户自有商品集合不存在时，先初始化集合。
		if !exists {
			ownedGoodsMap = make(map[int64]struct{})
			ownedGoodsMapByUser[item.UserID] = ownedGoodsMap
		}
		ownedGoodsMap[item.GoodsID] = struct{}{}
	}

	rawScoreMapByUser := make(map[int64]map[int64]recommendGoodsScore, len(similarUserMap))
	candidateGoodsIds := make([]int64, 0)
	for userId, neighborList := range similarUserMap {
		goodsScoreMap := make(map[int64]recommendGoodsScore)
		ownedGoodsMap := ownedGoodsMapByUser[userId]
		for _, neighbor := range neighborList {
			neighborUserId, parseErr := strconv.ParseInt(neighbor.Id, 10, 64)
			// 邻居用户编号非法时，直接跳过异常邻居。
			if parseErr != nil || neighborUserId <= 0 || neighbor.Score <= 0 {
				continue
			}
			preferenceList := goodsPreferenceByUser[neighborUserId]
			for _, preference := range preferenceList {
				_, exists := ownedGoodsMap[preference.GoodsID]
				// 当前用户自己已经有偏好的商品，不再重复进入协同过滤候选。
				if exists {
					continue
				}
				current := goodsScoreMap[preference.GoodsID]
				current.Score += neighbor.Score * preference.Score
				current.UpdatedAt = maxRecommendTime(current.UpdatedAt, preference.UpdatedAt)
				goodsScoreMap[preference.GoodsID] = current
			}
		}
		for goodsId := range goodsScoreMap {
			candidateGoodsIds = append(candidateGoodsIds, goodsId)
		}
		rawScoreMapByUser[userId] = goodsScoreMap
	}

	putOnGoodsMap, err := loadPutOnGoodsMap(ctx, goodsInfoRepo, candidateGoodsIds)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]recommendCache.Score, len(rawScoreMapByUser))
	for userId, goodsScoreMap := range rawScoreMapByUser {
		documentList := make([]recommendCache.Score, 0, len(goodsScoreMap))
		for goodsId, item := range goodsScoreMap {
			_, exists := putOnGoodsMap[goodsId]
			// 已下架商品不继续发布到协同过滤缓存。
			if !exists || item.Score <= 0 {
				continue
			}
			documentList = append(documentList, recommendCache.Score{
				Id:        strconv.FormatInt(goodsId, 10),
				Score:     item.Score,
				Timestamp: item.UpdatedAt,
			})
		}
		recommendCache.SortDocuments(documentList)
		// 协同过滤结果超过上限时，只保留 TopN。
		if int64(len(documentList)) > limit {
			documentList = documentList[:limit]
		}
		result[userId] = documentList
	}
	return result, nil
}

// buildContentBasedMaterializeMap 基于商品属性构建内容相似结果。
func buildContentBasedMaterializeMap(goodsList []*models.GoodsInfo, limit int64) map[int64][]recommendCache.Score {
	goodsByCategory := make(map[int64][]*models.GoodsInfo)
	for _, item := range goodsList {
		// 非法商品或非法类目不参与内容相似构建。
		if item == nil || item.ID <= 0 || item.CategoryID <= 0 {
			continue
		}
		goodsByCategory[item.CategoryID] = append(goodsByCategory[item.CategoryID], item)
	}

	result := make(map[int64][]recommendCache.Score, len(goodsList))
	for _, categoryGoodsList := range goodsByCategory {
		for _, baseGoods := range categoryGoodsList {
			// 基准商品为空时，不继续构建内容相似结果。
			if baseGoods == nil {
				continue
			}
			documentList := make([]recommendCache.Score, 0, len(categoryGoodsList))
			for _, targetGoods := range categoryGoodsList {
				// 目标商品为空或就是当前商品时，不加入内容相似结果。
				if targetGoods == nil || targetGoods.ID == baseGoods.ID {
					continue
				}
				score := calculateContentBasedScore(baseGoods, targetGoods)
				// 内容相似分数非法时，不加入缓存结果。
				if score <= 0 {
					continue
				}
				documentList = append(documentList, recommendCache.Score{
					Id:        strconv.FormatInt(targetGoods.ID, 10),
					Score:     score,
					Timestamp: maxRecommendTime(targetGoods.UpdatedAt, targetGoods.CreatedAt),
				})
			}
			recommendCache.SortDocuments(documentList)
			// 内容相似结果超过上限时，只保留 TopN。
			if int64(len(documentList)) > limit {
				documentList = documentList[:limit]
			}
			result[baseGoods.ID] = documentList
		}
	}
	return result
}

// countRecommendGoodsCategory 返回商品列表中的有效类目数量。
func countRecommendGoodsCategory(goodsList []*models.GoodsInfo) int {
	categorySet := make(map[int64]struct{})
	for _, item := range goodsList {
		// 非法商品或非法类目不参与类目数量统计。
		if item == nil || item.CategoryID <= 0 {
			continue
		}
		categorySet[item.CategoryID] = struct{}{}
	}
	return len(categorySet)
}

// calculateContentBasedScore 计算两件商品的内容相似分数。
func calculateContentBasedScore(baseGoods *models.GoodsInfo, targetGoods *models.GoodsInfo) float64 {
	// 商品为空或类目不同的时候，不认为存在内容相似关系。
	if baseGoods == nil || targetGoods == nil || baseGoods.CategoryID != targetGoods.CategoryID {
		return 0
	}
	maxPrice := maxRecommendInt64(baseGoods.Price, targetGoods.Price, 1)
	priceSimilarity := 1 - math.Min(1, math.Abs(float64(baseGoods.Price-targetGoods.Price))/float64(maxPrice))
	dayDiff := math.Abs(baseGoods.CreatedAt.Sub(targetGoods.CreatedAt).Hours()) / 24
	freshnessSimilarity := 1 / (1 + dayDiff/30)
	return priceSimilarity*recommendContentPriceWeight + freshnessSimilarity*recommendContentFreshnessWeight
}

// clearStaleVersionedSubsets 清理指定集合下当前版本已经失效的子集合。
func clearStaleVersionedSubsets(
	ctx context.Context,
	store recommendCache.Store,
	collection string,
	version string,
	currentSubsetMap map[string]struct{},
) (int, error) {
	collectionKey := recommendCache.CollectionKey(collection)
	subsetIndexMap, err := store.HGetAll(recommendCache.ScoreSubsetIndexKey(collectionKey))
	if err != nil {
		// 当前集合尚未建立索引时，说明没有旧缓存需要清理。
		if err == recommendCache.ErrObjectNotExist {
			return 0, nil
		}
		return 0, err
	}

	versionSuffix := "/version/" + recommendCache.NormalizeVersion(version)
	clearedSubsetCount := 0
	for subset := range subsetIndexMap {
		// 只清理当前版本空间下的子集合。
		if !strings.HasSuffix(subset, versionSuffix) {
			continue
		}
		_, exists := currentSubsetMap[subset]
		// 本次仍然产出的子集合不做清理。
		if exists {
			continue
		}
		err = store.DeleteScores(ctx, []string{collectionKey}, recommendCache.ScoreCondition{Subset: &subset})
		if err != nil {
			return 0, err
		}
		err = store.Del(recommendCache.DigestKey(collection, subset))
		if err != nil {
			return 0, err
		}
		err = store.Del(recommendCache.UpdateTimeKey(collection, subset))
		if err != nil {
			return 0, err
		}
		clearedSubsetCount++
	}
	return clearedSubsetCount, nil
}

// maxRecommendTime 返回两个时间中的较晚值。
func maxRecommendTime(left time.Time, right time.Time) time.Time {
	// 左侧时间为空时，直接回退到右侧时间。
	if left.IsZero() {
		return right
	}
	// 右侧时间为空时，直接保留左侧时间。
	if right.IsZero() {
		return left
	}
	if right.After(left) {
		return right
	}
	return left
}

// maxRecommendInt64 返回给定整数中的最大值。
func maxRecommendInt64(first int64, second int64, third int64) int64 {
	currentMax := first
	// 第二个值更大时，先替换当前最大值。
	if second > currentMax {
		currentMax = second
	}
	// 第三个值更大时，再次替换当前最大值。
	if third > currentMax {
		currentMax = third
	}
	return currentMax
}
