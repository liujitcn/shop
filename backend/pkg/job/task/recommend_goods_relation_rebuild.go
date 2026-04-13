package task

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	appBiz "shop/service/app/biz"
)

// RecommendGoodsRelationRebuild 推荐商品关联重建任务。
type RecommendGoodsRelationRebuild struct {
	recommendActorBindLogCase *appBiz.RecommendActorBindLogCase
	ctx                       context.Context
}

// NewRecommendGoodsRelationRebuild 创建推荐商品关联重建任务实例。
func NewRecommendGoodsRelationRebuild(recommendActorBindLogCase *appBiz.RecommendActorBindLogCase) *RecommendGoodsRelationRebuild {
	return &RecommendGoodsRelationRebuild{
		recommendActorBindLogCase: recommendActorBindLogCase,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐商品关联重建。
func (t *RecommendGoodsRelationRebuild) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendGoodsRelationRebuild Exec %+v", args)

	windowDays, err := parseRecommendWindowDaysArg(args["windowDays"])
	if err != nil {
		return []string{err.Error()}, err
	}

	err = t.recommendActorBindLogCase.RebuildRecommendGoodsRelation(t.ctx, nil, windowDays)
	if err != nil {
		return []string{err.Error()}, err
	}
	return []string{fmt.Sprintf("推荐商品关联重建完成: %d 天窗口", windowDays)}, nil
}
