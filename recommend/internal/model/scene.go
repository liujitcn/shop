package model

import "recommend/internal/core"

// Scene 表示推荐链路内部使用的场景标识。
type Scene string

const (
	// SceneHome 表示首页推荐场景。
	SceneHome Scene = "home"
	// SceneGoodsDetail 表示商品详情推荐场景。
	SceneGoodsDetail Scene = "goods_detail"
	// SceneCart 表示购物车推荐场景。
	SceneCart Scene = "cart"
	// SceneProfile 表示个人中心推荐场景。
	SceneProfile Scene = "profile"
	// SceneOrderDetail 表示订单详情推荐场景。
	SceneOrderDetail Scene = "order_detail"
	// SceneOrderPaid 表示订单支付完成推荐场景。
	SceneOrderPaid Scene = "order_paid"
)

// ResolveScene 将公开场景转换为内部场景。
func ResolveScene(scene core.Scene) Scene {
	// 公开场景已经收敛为固定枚举时，优先保留内部约定的常量值。
	switch scene {
	case core.SceneHome:
		return SceneHome
	case core.SceneGoodsDetail:
		return SceneGoodsDetail
	case core.SceneCart:
		return SceneCart
	case core.SceneProfile:
		return SceneProfile
	case core.SceneOrderDetail:
		return SceneOrderDetail
	case core.SceneOrderPaid:
		return SceneOrderPaid
	default:
		return Scene(scene)
	}
}

// String 返回场景的字符串值。
func (s Scene) String() string {
	return string(s)
}
