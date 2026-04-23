package queue

import (
	_const "shop/pkg/const"
	"shop/pkg/gen/models"
	"sync/atomic"
)

var recommendEnabled atomic.Bool

// SetRecommendEnabled 设置推荐系统异步投递链路是否启用。
func SetRecommendEnabled(enabled bool) {
	recommendEnabled.Store(enabled)
}

// DispatchRecommendSyncBaseUser 将用户主键投递到推荐系统异步同步链路。
func DispatchRecommendSyncBaseUser(userId int64) {
	// 用户编号非法时，当前同步请求无效。
	if userId <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RecommendSyncBaseUser, userId)
}

// DispatchRecommendDeleteBaseUser 将用户删除事件投递到推荐系统异步同步链路。
func DispatchRecommendDeleteBaseUser(userIds []int64) {
	// 用户编号非法时，不再继续投递删除消息。
	if len(userIds) <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RecommendDeleteBaseUser, userIds)
}

// DispatchRecommendSyncGoodsInfo 将商品主键投递到推荐系统异步同步链路。
func DispatchRecommendSyncGoodsInfo(goodsId int64) {
	// 商品编号非法时，当前同步请求无效。
	if goodsId <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RecommendSyncGoodsInfo, goodsId)
}

// DispatchRecommendDeleteGoodsInfo 将商品删除事件投递到推荐系统异步同步链路。
func DispatchRecommendDeleteGoodsInfo(goodsIds []int64) {
	// 商品编号非法时，不再继续投递删除消息。
	if len(goodsIds) <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RecommendDeleteGoodsInfo, goodsIds)
}

// DispatchRecommendEventList 将历史推荐事件投递到推荐系统回放链路。
func DispatchRecommendEventList(eventList []*models.RecommendEvent) {
	// 历史事件为空时，不再继续投递回放消息。
	if len(eventList) == 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RecommendEvent, eventList)
}
