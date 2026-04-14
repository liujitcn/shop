package recommend

import "context"

// BuildNonPersonalized 构建最新商品、场景热销和全站热销候选池。
func BuildNonPersonalized(_ context.Context, _ Dependencies, _ BuildNonPersonalizedRequest) (*BuildResult, error) {
	return nil, ErrNotImplemented
}

// BuildUserCandidate 构建用户商品偏好和类目偏好候选池。
func BuildUserCandidate(_ context.Context, _ Dependencies, _ BuildUserCandidateRequest) (*BuildResult, error) {
	return nil, ErrNotImplemented
}

// BuildGoodsRelation 构建商品关联候选池。
func BuildGoodsRelation(_ context.Context, _ Dependencies, _ BuildGoodsRelationRequest) (*BuildResult, error) {
	return nil, ErrNotImplemented
}

// BuildUserToUser 构建相似用户召回所需的邻居用户池。
func BuildUserToUser(_ context.Context, _ Dependencies, _ BuildUserToUserRequest) (*BuildResult, error) {
	return nil, ErrNotImplemented
}

// BuildCollaborative 构建协同过滤候选池。
func BuildCollaborative(_ context.Context, _ Dependencies, _ BuildCollaborativeRequest) (*BuildResult, error) {
	return nil, ErrNotImplemented
}

// BuildExternal 构建活动池、营销池、人工池等外部推荐池。
func BuildExternal(_ context.Context, _ Dependencies, _ BuildExternalRequest) (*BuildResult, error) {
	return nil, ErrNotImplemented
}
