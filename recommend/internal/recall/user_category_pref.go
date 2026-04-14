package recall

import (
	"context"
	"recommend/internal/model"
)

// RecallUserCategoryPreference 召回用户类目偏好结果。
func RecallUserCategoryPreference(ctx context.Context, request Request) ([]*model.Candidate, error) {
	// 登录用户缺失时，用户类目偏好召回无法执行。
	if !request.Actor.IsUser() {
		return []*model.Candidate{}, nil
	}

	rows, err := request.Dependencies.Recommend.ListUserCategoryPreference(ctx, request.Actor.Id, ResolveLimit(request.Limit))
	if err != nil {
		return nil, err
	}
	return buildCategoryCandidates(ctx, request.Dependencies.Goods, rows, ResolveLimit(request.Limit))
}
