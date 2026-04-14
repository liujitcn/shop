package materialize

import (
	"context"
	"errors"
	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	cachex "recommend/internal/cache"
	cacheleveldb "recommend/internal/cache/leveldb"
	"recommend/internal/core"
	"recommend/internal/recall"
	"time"
)

const (
	// vectorTargetTypeActor 表示基于主体画像构建的向量池。
	vectorTargetTypeActor = "actor"
	// vectorTargetTypeGoods 表示基于商品锚点构建的向量池。
	vectorTargetTypeGoods = "goods"
)

// BuildVector 构建向量召回池。
func BuildVector(ctx context.Context, dependencies core.Dependencies, _ core.ServiceConfig, request core.BuildVectorRequest) (*core.BuildResult, error) {
	err := validateVectorDependencies(dependencies)
	if err != nil {
		return nil, err
	}

	manager, err := cacheleveldb.OpenManager(ctx, dependencies.Cache)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = manager.Close()
	}()

	store := &cachex.PoolStore{Driver: manager}
	scenes := normalizeScenes(request.Scenes, defaultScenes)
	userIds := normalizeIds(request.UserIds)
	goodsIds := normalizeIds(request.GoodsIds)
	limit := normalizeBuildLimit(request.Limit)
	updatedAt := time.Now()
	keyCount := int64(0)

	for _, scene := range scenes {
		if isActorVectorScene(scene) {
			for _, userId := range userIds {
				err = saveActorVectorPool(ctx, dependencies, store, scene, userId, limit, updatedAt)
				if err != nil {
					return nil, err
				}
				keyCount++
			}
			continue
		}

		for _, goodsId := range goodsIds {
			err = saveGoodsVectorPool(ctx, dependencies, store, scene, goodsId, limit, updatedAt)
			if err != nil {
				return nil, err
			}
			keyCount++
		}
	}

	return buildResult("vector", keyCount, updatedAt), nil
}

// validateVectorDependencies 校验向量召回池构建依赖。
func validateVectorDependencies(dependencies core.Dependencies) error {
	if dependencies.Cache == nil {
		return errors.New("recommend: 缓存数据源未配置")
	}
	if dependencies.Goods == nil {
		return errors.New("recommend: 商品数据源未配置")
	}
	if dependencies.Vector == nil {
		return errors.New("recommend: 向量数据源未配置")
	}
	return nil
}

// isActorVectorScene 判断当前场景是否使用用户向量池。
func isActorVectorScene(scene core.Scene) bool {
	switch scene {
	case core.SceneHome, core.SceneProfile:
		return true
	default:
		return false
	}
}

// saveActorVectorPool 构建并保存用户向量池。
func saveActorVectorPool(
	ctx context.Context,
	dependencies core.Dependencies,
	store *cachex.PoolStore,
	scene core.Scene,
	userId int64,
	limit int32,
	updatedAt time.Time,
) error {
	rows, err := dependencies.Vector.ListVectorGoods(ctx, contract.VectorRecallRequest{
		Scene:     string(scene),
		ActorType: int32(core.ActorTypeUser),
		ActorId:   userId,
		Limit:     limit,
	})
	if err != nil {
		return err
	}
	items, err := buildWeightedGoodsItems(ctx, dependencies.Goods, rows, recall.RecallSourceVector)
	if err != nil {
		return err
	}
	return store.SaveVectorPool(string(scene), vectorTargetTypeActor, userId, &recommendv1.RecommendVectorPool{
		Meta:       buildPoolMeta(string(scene), int32(core.ActorTypeUser), userId, updatedAt),
		TargetType: vectorTargetTypeActor,
		TargetId:   userId,
		Items:      items,
	})
}

// saveGoodsVectorPool 构建并保存商品向量池。
func saveGoodsVectorPool(
	ctx context.Context,
	dependencies core.Dependencies,
	store *cachex.PoolStore,
	scene core.Scene,
	goodsId int64,
	limit int32,
	updatedAt time.Time,
) error {
	rows, err := dependencies.Vector.ListVectorGoods(ctx, contract.VectorRecallRequest{
		Scene:          string(scene),
		SourceGoodsIds: []int64{goodsId},
		Limit:          limit,
	})
	if err != nil {
		return err
	}
	items, err := buildWeightedGoodsItems(ctx, dependencies.Goods, rows, recall.RecallSourceVector)
	if err != nil {
		return err
	}
	return store.SaveVectorPool(string(scene), vectorTargetTypeGoods, goodsId, &recommendv1.RecommendVectorPool{
		Meta:       buildPoolMeta(string(scene), 0, 0, updatedAt),
		TargetType: vectorTargetTypeGoods,
		TargetId:   goodsId,
		Items:      items,
	})
}
