package biz

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCache "shop/pkg/recommend/cache"
	recommendDomain "shop/pkg/recommend/domain"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendModelVersionCase 推荐版本业务处理对象。
type RecommendModelVersionCase struct {
	*biz.BaseCase
	*data.RecommendModelVersionRepo
}

// NewRecommendModelVersionCase 创建推荐版本业务处理对象。
func NewRecommendModelVersionCase(
	baseCase *biz.BaseCase,
	recommendModelVersionRepo *data.RecommendModelVersionRepo,
) *RecommendModelVersionCase {
	return &RecommendModelVersionCase{
		BaseCase:                  baseCase,
		RecommendModelVersionRepo: recommendModelVersionRepo,
	}
}

// loadEnabledSceneVersionEntity 查询当前场景启用的推荐版本记录。
func (c *RecommendModelVersionCase) loadEnabledSceneVersionEntity(ctx context.Context, scene int32) (*models.RecommendModelVersion, error) {
	query := c.Query(ctx).RecommendModelVersion
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.Scene.Eq(scene)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, _, err := c.Page(ctx, 1, 1, opts...)
	if err != nil {
		return nil, err
	}
	// 当前场景没有启用版本时，直接回退为空记录。
	if len(list) == 0 || list[0] == nil {
		return nil, nil
	}
	return list[0], nil
}

// loadSceneCacheVersion 查询当前场景启用的缓存版本信息。
func (c *RecommendModelVersionCase) loadSceneCacheVersion(ctx context.Context, scene int32) (string, time.Time, error) {
	sceneStrategyContext, err := c.loadSceneStrategyContext(ctx, scene)
	if err != nil {
		return "", time.Time{}, err
	}
	return recommendCache.NormalizeVersion(sceneStrategyContext.EffectiveVersion), sceneStrategyContext.VersionPublishedAt, nil
}

// loadSceneStrategyContext 查询当前场景启用的版本策略上下文。
func (c *RecommendModelVersionCase) loadSceneStrategyContext(ctx context.Context, scene int32) (*recommendDomain.SceneStrategyContext, error) {
	entity, err := c.loadEnabledSceneVersionEntity(ctx, scene)
	if err != nil {
		return nil, err
	}
	// 当前场景没有启用版本时，直接回退到默认版本和空策略配置。
	if entity == nil {
		return &recommendDomain.SceneStrategyContext{
			Scene:              scene,
			Version:            recommendCache.DefaultVersion,
			EffectiveVersion:   recommendCache.DefaultVersion,
			VersionPublishedAt: time.Time{},
			Config:             &recommendDomain.StrategyVersionConfig{},
		}, nil
	}
	version := recommendCache.NormalizeVersion(entity.Version)
	versionPublishedAt := entity.CreatedAt

	config := &recommendDomain.StrategyVersionConfig{}
	// 当前版本没有扩展配置时，直接返回基础版本策略上下文。
	if strings.TrimSpace(entity.ConfigJSON) == "" {
		return &recommendDomain.SceneStrategyContext{
			Scene:              scene,
			Version:            version,
			EffectiveVersion:   version,
			VersionPublishedAt: versionPublishedAt,
			Config:             config,
		}, nil
	}

	err = json.Unmarshal([]byte(entity.ConfigJSON), config)
	if err != nil {
		// 阶段 4 到阶段 8 的增量策略都不允许因配置脏数据拖垮主链路。
		log.Errorf("loadSceneStrategyContext %v", err)
		config = &recommendDomain.StrategyVersionConfig{}
	}

	effectiveVersion := version
	if config.Publish != nil {
		effectiveVersion = recommendCache.NormalizeVersion(config.Publish.ResolveEffectiveVersion(version))
	}
	return &recommendDomain.SceneStrategyContext{
		Scene:              scene,
		Version:            version,
		EffectiveVersion:   effectiveVersion,
		VersionPublishedAt: versionPublishedAt,
		Config:             config,
	}, nil
}

// loadRecommendRecallProbeConfig 查询当前场景启用的召回探针配置。
func (c *RecommendModelVersionCase) loadRecommendRecallProbeConfig(ctx context.Context, scene int32) (string, time.Time, *recommendDomain.RecallProbeStrategy, error) {
	sceneStrategyContext, err := c.loadSceneStrategyContext(ctx, scene)
	if err != nil {
		return "", time.Time{}, nil, err
	}
	// 当前版本没有扩展探针配置时，直接回退到空探针配置。
	if sceneStrategyContext.Config == nil || sceneStrategyContext.Config.RecallProbe == nil {
		return sceneStrategyContext.EffectiveVersion, sceneStrategyContext.VersionPublishedAt, &recommendDomain.RecallProbeStrategy{}, nil
	}
	return sceneStrategyContext.EffectiveVersion, sceneStrategyContext.VersionPublishedAt, sceneStrategyContext.Config.RecallProbe, nil
}
