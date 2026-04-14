package model

import "recommend"

// Scene 表示推荐链路内部使用的场景标识。
type Scene string

const (
	SceneHome        Scene = "home"
	SceneGoodsDetail Scene = "goods_detail"
	SceneCart        Scene = "cart"
	SceneProfile     Scene = "profile"
	SceneOrderDetail Scene = "order_detail"
	SceneOrderPaid   Scene = "order_paid"
)

// ResolveScene 将公开场景转换为内部场景。
func ResolveScene(scene recommend.Scene) Scene {
	// 公开场景已经收敛为固定枚举时，优先保留内部约定的常量值。
	switch scene {
	case recommend.SceneHome:
		return SceneHome
	case recommend.SceneGoodsDetail:
		return SceneGoodsDetail
	case recommend.SceneCart:
		return SceneCart
	case recommend.SceneProfile:
		return SceneProfile
	case recommend.SceneOrderDetail:
		return SceneOrderDetail
	case recommend.SceneOrderPaid:
		return SceneOrderPaid
	default:
		return Scene(scene)
	}
}

// String 返回场景的字符串值。
func (s Scene) String() string {
	return string(s)
}
