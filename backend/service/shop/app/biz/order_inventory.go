package biz

import (
	"context"
	"fmt"

	"shop/pkg/errorsx"
	"shop/pkg/gen/data"

	"github.com/liujitcn/gorm-kit/repository"
)

// OrderInventoryCase 处理订单商品库存与销量回退。
type OrderInventoryCase struct {
	orderGoodsRepo *data.OrderGoodsRepository
	goodsInfoRepo  *data.GoodsInfoRepository
	goodsSKURepo   *data.GoodsSKURepository
}

// NewOrderInventoryCase 创建订单库存业务处理对象。
func NewOrderInventoryCase(
	orderGoodsRepo *data.OrderGoodsRepository,
	goodsInfoRepo *data.GoodsInfoRepository,
	goodsSKURepo *data.GoodsSKURepository,
) *OrderInventoryCase {
	return &OrderInventoryCase{
		orderGoodsRepo: orderGoodsRepo,
		goodsInfoRepo:  goodsInfoRepo,
		goodsSKURepo:   goodsSKURepo,
	}
}

// RestoreOrder 恢复指定门店订单的商品库存并回退销量，调用方负责在同一事务内完成状态抢占。
func (c *OrderInventoryCase) RestoreOrder(ctx context.Context, orderID int64) error {
	query := c.orderGoodsRepo.Query(ctx).OrderGoods
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	orderGoods, err := c.orderGoodsRepo.List(ctx, opts...)
	if err != nil {
		return err
	}
	if len(orderGoods) == 0 {
		return errorsx.Internal(fmt.Sprintf("订单库存回退失败，订单商品不存在：orderID=%d", orderID))
	}

	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	skuQuery := c.goodsSKURepo.Query(ctx).GoodsSKU
	for _, item := range orderGoods {
		result, updateErr := goodsQuery.WithContext(ctx).
			Where(goodsQuery.ID.Eq(item.GoodsID), goodsQuery.RealSaleNum.Gte(item.Num)).
			Updates(map[string]interface{}{
				"real_sale_num": goodsQuery.RealSaleNum.Sub(item.Num),
				"inventory":     goodsQuery.Inventory.Add(item.Num),
			})
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected == 0 {
			return errorsx.Internal(fmt.Sprintf("订单库存回退失败，商品销量数据异常：goodsID=%d，num=%d", item.GoodsID, item.Num))
		}

		result, updateErr = skuQuery.WithContext(ctx).
			Where(skuQuery.SKUCode.Eq(item.SKUCode), skuQuery.RealSaleNum.Gte(item.Num)).
			Updates(map[string]interface{}{
				"real_sale_num": skuQuery.RealSaleNum.Sub(item.Num),
				"inventory":     skuQuery.Inventory.Add(item.Num),
			})
		if updateErr != nil {
			return updateErr
		}
		if result.RowsAffected == 0 {
			return errorsx.Internal(fmt.Sprintf("订单库存回退失败，规格销量数据异常：skuCode=%s，num=%d", item.SKUCode, item.Num))
		}
	}
	return nil
}
