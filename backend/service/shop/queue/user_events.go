package queue

import "shop/pkg/event"

// RecommendUserEventSubscriber 将基础用户变更转换为商城推荐同步任务。
type RecommendUserEventSubscriber struct{}

// NewRecommendUserEventSubscriber 创建商城推荐用户事件订阅者。
func NewRecommendUserEventSubscriber() *RecommendUserEventSubscriber {
	return &RecommendUserEventSubscriber{}
}

// UserChanged 投递用户画像同步任务。
func (s *RecommendUserEventSubscriber) UserChanged(userID int64) {
	DispatchRecommendSyncBaseUser(userID)
}

// UsersDeleted 投递用户画像删除任务。
func (s *RecommendUserEventSubscriber) UsersDeleted(userIDs []int64) {
	DispatchRecommendDeleteBaseUser(userIDs)
}

var _ event.UserSubscriber = (*RecommendUserEventSubscriber)(nil)
