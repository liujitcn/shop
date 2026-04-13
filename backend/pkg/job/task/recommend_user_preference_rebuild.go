package task

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"

	appBiz "shop/service/app/biz"
)

// RecommendUserPreferenceRebuild 推荐用户偏好重建任务。
type RecommendUserPreferenceRebuild struct {
	recommendActorBindLogCase *appBiz.RecommendActorBindLogCase
	ctx                       context.Context
}

// NewRecommendUserPreferenceRebuild 创建推荐用户偏好重建任务实例。
func NewRecommendUserPreferenceRebuild(recommendActorBindLogCase *appBiz.RecommendActorBindLogCase) *RecommendUserPreferenceRebuild {
	return &RecommendUserPreferenceRebuild{
		recommendActorBindLogCase: recommendActorBindLogCase,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐用户偏好重建。
func (t *RecommendUserPreferenceRebuild) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendUserPreferenceRebuild Exec %+v", args)

	windowDays, err := parseRecommendWindowDaysArg(args["windowDays"])
	if err != nil {
		return []string{err.Error()}, err
	}

	err = t.recommendActorBindLogCase.RebuildRecommendUserPreference(t.ctx, nil, windowDays)
	if err != nil {
		return []string{err.Error()}, err
	}
	return []string{fmt.Sprintf("推荐用户偏好重建完成: %d 天窗口", windowDays)}, nil
}
