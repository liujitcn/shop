package task

import (
	"fmt"

	"shop/api/gen/go/common"
)

const (
	recommendGoodsActionTypeView    = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_VIEW
	recommendGoodsActionTypeCollect = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_COLLECT
	recommendGoodsActionTypeCart    = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CART
	recommendGoodsActionTypeOrder   = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_CREATE
	recommendGoodsActionTypePay     = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_ORDER_PAY
	recommendGoodsActionTypeClick   = common.RecommendGoodsActionType_RECOMMEND_GOODS_ACTION_CLICK
)

// buildRecommendGoodsActionTypeNameExpr 构建行为枚举到字符串事件名的 SQL 表达式。
func buildRecommendGoodsActionTypeNameExpr(column string) string {
	return fmt.Sprintf(
		"CASE %s WHEN %d THEN '%s' WHEN %d THEN '%s' WHEN %d THEN '%s' WHEN %d THEN '%s' WHEN %d THEN '%s' WHEN %d THEN '%s' ELSE '' END",
		column,
		recommendGoodsActionTypeClick, "recommend_click",
		recommendGoodsActionTypeView, "goods_view",
		recommendGoodsActionTypeCollect, "goods_collect",
		recommendGoodsActionTypeCart, "goods_cart",
		recommendGoodsActionTypeOrder, "order_create",
		recommendGoodsActionTypePay, "order_pay",
	)
}
