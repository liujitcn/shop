package task

import (
	"context"
	"sort"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendEvent "shop/pkg/recommend/event"
	recommendRank "shop/pkg/recommend/rank"

	"github.com/liujitcn/gorm-kit/repo"
)

// listRecommendMaterializeScenes 返回当前需要写入缓存的推荐场景列表。
func listRecommendMaterializeScenes() []int32 {
	sceneList := make([]int32, 0, len(common.RecommendScene_name))
	for scene := range common.RecommendScene_name {
		// 未指定场景不参与缓存写入。
		if scene <= 0 {
			continue
		}
		sceneList = append(sceneList, scene)
	}
	sort.Slice(sceneList, func(i int, j int) bool {
		return sceneList[i] < sceneList[j]
	})
	return sceneList
}

// loadEnabledRecommendVersionMap 查询场景启用版本映射。
func loadEnabledRecommendVersionMap(ctx context.Context, recommendModelVersionRepo *data.RecommendModelVersionRepo, sceneList []int32) (map[int32]string, error) {
	versionMap := make(map[int32]string, len(sceneList))
	// 没有待查询场景时，直接返回空映射。
	if len(sceneList) == 0 {
		return versionMap, nil
	}

	query := recommendModelVersionRepo.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.Scene.In(sceneList...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Order(query.Scene.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := recommendModelVersionRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		_, exists := versionMap[item.Scene]
		// 每个场景只保留最新一条启用版本。
		if exists {
			continue
		}
		versionMap[item.Scene] = item.Version
	}
	return versionMap, nil
}

// resolveRecommendSceneVersion 返回场景最终使用的缓存版本。
func resolveRecommendSceneVersion(versionMap map[int32]string, scene int32) string {
	version, ok := versionMap[scene]
	// 场景未启用版本时，统一回退到默认缓存版本。
	if !ok {
		return recommendCache.DefaultVersion
	}
	return recommendCache.NormalizeVersion(version)
}

// loadPutOnGoodsMap 查询上架商品映射。
func loadPutOnGoodsMap(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo, goodsIds []int64) (map[int64]*models.GoodsInfo, error) {
	result := make(map[int64]*models.GoodsInfo)
	// 没有商品编号时，无需继续访问数据库。
	if len(goodsIds) == 0 {
		return result, nil
	}

	query := goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.In(goodsIds...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	list, err := goodsInfoRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range list {
		result[item.ID] = item
	}
	return result, nil
}

// loadSceneHotMaterializeList 加载场景热门榜缓存输入。
func loadSceneHotMaterializeList(
	ctx context.Context,
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	scene int32,
	startDate time.Time,
	limit int64,
) ([]*models.RecommendGoodsStatDay, error) {
	query := recommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.Scene.Eq(scene)))
	opts = append(opts, repo.Where(query.StatDate.Gte(startDate)))
	list, err := recommendGoodsStatDayRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	aggregatedMap := make(map[int64]*models.RecommendGoodsStatDay)
	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		// 非法统计记录不参与热门榜聚合。
		if item == nil || item.GoodsID <= 0 {
			continue
		}
		stat, exists := aggregatedMap[item.GoodsID]
		// 首次命中商品时，先初始化聚合容器。
		if !exists {
			stat = &models.RecommendGoodsStatDay{
				Scene:     scene,
				GoodsID:   item.GoodsID,
				UpdatedAt: item.UpdatedAt,
			}
			aggregatedMap[item.GoodsID] = stat
			goodsIds = append(goodsIds, item.GoodsID)
		}
		stat.Score += item.Score * recommendRank.CalculateDayDecay(item.StatDate)
		// 缓存发布时间取最近一条统计记录的更新时间。
		if item.UpdatedAt.After(stat.UpdatedAt) {
			stat.UpdatedAt = item.UpdatedAt
		}
	}

	putOnGoodsMap := make(map[int64]*models.GoodsInfo)
	putOnGoodsMap, err = loadPutOnGoodsMap(ctx, goodsInfoRepo, goodsIds)
	if err != nil {
		return nil, err
	}

	result := make([]*models.RecommendGoodsStatDay, 0, len(aggregatedMap))
	for goodsId, item := range aggregatedMap {
		_, exists := putOnGoodsMap[goodsId]
		// 已下架商品不继续发布到热门榜缓存。
		if !exists {
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i int, j int) bool {
		// 热门榜先按分数降序，再按更新时间降序稳定排序。
		if result[i].Score == result[j].Score {
			if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
				return result[i].GoodsID < result[j].GoodsID
			}
			return result[i].UpdatedAt.After(result[j].UpdatedAt)
		}
		return result[i].Score > result[j].Score
	})
	// 缓存结果超过上限时，只发布 TopN。
	if int64(len(result)) > limit {
		result = result[:limit]
	}
	return result, nil
}

// loadLatestGoodsMaterializeList 加载最新榜缓存输入。
func loadLatestGoodsMaterializeList(ctx context.Context, goodsInfoRepo *data.GoodsInfoRepo, limit int64) ([]*models.GoodsInfo, error) {
	query := goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, _, err := goodsInfoRepo.Page(ctx, 1, limit, opts...)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// loadSimilarItemMaterializeMap 加载相似商品缓存输入。
func loadSimilarItemMaterializeMap(
	ctx context.Context,
	recommendGoodsRelationRepo *data.RecommendGoodsRelationRepo,
	goodsInfoRepo *data.GoodsInfoRepo,
	limit int64,
) (map[int64][]*models.RecommendGoodsRelation, error) {
	query := recommendGoodsRelationRepo.Query(ctx).RecommendGoodsRelation
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.WindowDays.Eq(recommendEvent.AggregateWindowDays)))
	opts = append(opts, repo.Order(query.GoodsID.Asc()))
	opts = append(opts, repo.Order(query.Score.Desc()))
	list, err := recommendGoodsRelationRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list)*2)
	for _, item := range list {
		// 非法关系记录不参与相似商品缓存发布。
		if item == nil || item.GoodsID <= 0 || item.RelatedGoodsID <= 0 {
			continue
		}
		goodsIds = append(goodsIds, item.GoodsID, item.RelatedGoodsID)
	}
	putOnGoodsMap := make(map[int64]*models.GoodsInfo)
	putOnGoodsMap, err = loadPutOnGoodsMap(ctx, goodsInfoRepo, goodsIds)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]*models.RecommendGoodsRelation)
	for _, item := range list {
		_, goodsExists := putOnGoodsMap[item.GoodsID]
		_, relatedExists := putOnGoodsMap[item.RelatedGoodsID]
		// 主商品或关联商品已下架时，不发布该关系。
		if !goodsExists || !relatedExists {
			continue
		}
		currentList := result[item.GoodsID]
		// 每个主商品只保留固定数量的相似商品。
		if int64(len(currentList)) >= limit {
			continue
		}
		result[item.GoodsID] = append(currentList, item)
	}
	return result, nil
}

// clearStaleSimilarItemSubsets 清理当前版本下已经失效的相似商品缓存子集合。
func clearStaleSimilarItemSubsets(
	ctx context.Context,
	store recommendCache.Store,
	version string,
	currentSubsetMap map[string]struct{},
) (int, error) {
	collectionKey := recommendCache.CollectionKey(recommendCache.ItemToItem)
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
		// 只清理当前版本空间下的相似商品子集合。
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
		err = store.Del(recommendCache.DigestKey(recommendCache.ItemToItem, subset))
		if err != nil {
			return 0, err
		}
		err = store.Del(recommendCache.UpdateTimeKey(recommendCache.ItemToItem, subset))
		if err != nil {
			return 0, err
		}
		clearedSubsetCount++
	}
	return clearedSubsetCount, nil
}
