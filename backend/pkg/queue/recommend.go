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
func DispatchRecommendSyncBaseUser(userID int64) {
	// 用户编号非法时，当前同步请求无效。
	if userID <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RECOMMEND_SYNC_BASE_USER, userID)
}

// DispatchRecommendDeleteBaseUser 将用户删除事件投递到推荐系统异步同步链路。
func DispatchRecommendDeleteBaseUser(userIDs []int64) {
	// 用户编号非法时，不再继续投递删除消息。
	if len(userIDs) <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RECOMMEND_DELETE_BASE_USER, userIDs)
}

// DispatchRecommendSyncGoodsInfo 将商品主键投递到推荐系统异步同步链路。
func DispatchRecommendSyncGoodsInfo(goodsID int64) {
	// 商品编号非法时，当前同步请求无效。
	if goodsID <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RECOMMEND_SYNC_GOODS_INFO, goodsID)
}

// DispatchRecommendDeleteGoodsInfo 将商品删除事件投递到推荐系统异步同步链路。
func DispatchRecommendDeleteGoodsInfo(goodsIDs []int64) {
	// 商品编号非法时，不再继续投递删除消息。
	if len(goodsIDs) <= 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RECOMMEND_DELETE_GOODS_INFO, goodsIDs)
}

// DispatchRecommendEventList 将历史推荐事件投递到推荐系统回放链路。
func DispatchRecommendEventList(eventList []*models.RecommendEvent) {
	// 历史事件为空时，不再继续投递回放消息。
	if len(eventList) == 0 || !recommendEnabled.Load() {
		return
	}
	AddQueue(_const.RECOMMEND_EVENT, eventList)
}
