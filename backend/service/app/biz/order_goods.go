package biz

import (
	"context"
	"errors"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/util"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderGoodsCase 订单商品明细业务处理对象
type OrderGoodsCase struct {
	*biz.BaseCase
	*data.OrderGoodsRepo
	goodsInfoCase *GoodsCase
	goodsSkuCase  *GoodsSkuCase
}

// NewOrderGoodsCase 创建订单商品明细业务处理对象
func NewOrderGoodsCase(baseCase *biz.BaseCase, orderGoodsRepo *data.OrderGoodsRepo,
	goodsInfoCase *GoodsCase,
	goodsSkuCase *GoodsSkuCase,
) *OrderGoodsCase {
	return &OrderGoodsCase{
		BaseCase:       baseCase,
		OrderGoodsRepo: orderGoodsRepo,
		goodsInfoCase:  goodsInfoCase,
		goodsSkuCase:   goodsSkuCase,
	}
}

// mapByOrderIds 按订单编号批量查询商品明细映射
func (c *OrderGoodsCase) mapByOrderIds(ctx context.Context, orderIds []int64) (map[int64][]*app.OrderGoods, error) {
	res := make(map[int64][]*app.OrderGoods)
	if len(orderIds) > 0 {
		query := c.Query(ctx).OrderGoods
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.OrderID.In(orderIds...)))
		all, err := c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range all {
			v, ok := res[item.OrderID]
			if !ok {
				v = make([]*app.OrderGoods, 0)
			}
			v = append(v, c.convertToProto(item))

			res[item.OrderID] = v
		}
	}
	return res, nil
}

// listByOrderId 查询单个订单的商品明细
func (c *OrderGoodsCase) listByOrderId(ctx context.Context, orderId int64) ([]*app.OrderGoods, error) {
	query := c.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.OrderGoods, 0)
	for _, item := range all {
		list = append(list, c.convertToProto(item))
	}
	return list, nil
}

// createByOrder 批量创建订单商品记录
func (c *OrderGoodsCase) createByOrder(ctx context.Context, orderId int64, goods []*models.OrderGoods) error {
	if len(goods) == 0 {
		return errors.New("订单商品信息不能为空")
	}
	for _, item := range goods {
		item.OrderID = orderId
	}
	return c.BatchCreate(ctx, goods)
}

// convertToModelList 将下单商品列表转换为模型列表
func (c *OrderGoodsCase) convertToModelList(ctx context.Context, goods []*app.CreateOrderGoods) ([]*models.OrderGoods, error) {
	// 根据登录信息判断当前下单用户是否享受会员价
	member := util.IsMember(ctx)

	orderGoodsList := make([]*models.OrderGoods, 0)
	for _, item := range goods {
		orderGoods, err := c.convertToModel(ctx, member, item)
		if err != nil {
			return nil, err
		}
		orderGoodsList = append(orderGoodsList, orderGoods)
	}
	return orderGoodsList, nil
}

// convertToProtoByCreateOrderGoods 预览下单商品信息
func (c *OrderGoodsCase) convertToProtoByCreateOrderGoods(ctx context.Context, member bool, item *app.CreateOrderGoods) (*app.OrderGoods, error) {
	// 查询商品信息和规格信息
	goodsQuery := c.goodsInfoCase.Query(ctx).Goods
	goodsInfo, err := c.goodsInfoCase.Find(ctx,
		repo.Where(goodsQuery.ID.Eq(item.GetGoodsId())),
		repo.Where(goodsQuery.Status.Eq(int32(common.Status_ENABLE))),
	)
	if err != nil {
		return nil, err
	}
	skuQuery := c.goodsSkuCase.Query(ctx).GoodsSku
	var goodsSku *models.GoodsSku
	goodsSku, err = c.goodsSkuCase.Find(ctx,
		repo.Where(skuQuery.SkuCode.Eq(item.GetSkuCode())),
		repo.Where(skuQuery.GoodsID.Eq(item.GetGoodsId())),
	)
	if err != nil {
		return nil, err
	}
	picture := goodsInfo.Picture
	if len(goodsSku.Picture) > 0 {
		// 规格图片优先级高于商品主图
		picture = goodsSku.Picture
	}

	// 支付价格
	payPrice := goodsSku.Price
	if member {
		payPrice = goodsSku.DiscountPrice
	}
	res := &app.OrderGoods{
		GoodsId:       goodsInfo.ID,
		SkuCode:       goodsSku.SkuCode,
		Picture:       picture,
		Name:          goodsInfo.Name,
		Num:           item.GetNum(),
		SpecItem:      _string.ConvertJsonStringToStringArray(goodsSku.SpecItem),
		Price:         goodsSku.Price,
		PayPrice:      payPrice,
		TotalPrice:    goodsSku.Price * item.GetNum(),
		TotalPayPrice: payPrice * item.GetNum(),
	}
	return res, nil
}

// 将订单商品模型转换为接口响应
func (c *OrderGoodsCase) convertToProto(item *models.OrderGoods) *app.OrderGoods {
	res := &app.OrderGoods{
		GoodsId:       item.GoodsID,
		SkuCode:       item.SkuCode,
		Picture:       item.Picture,
		Name:          item.Name,
		Num:           item.Num,
		SpecItem:      _string.ConvertJsonStringToStringArray(item.SpecItem),
		Price:         item.Price,
		PayPrice:      item.PayPrice,
		TotalPrice:    item.TotalPrice,
		TotalPayPrice: item.TotalPayPrice,
	}
	return res
}

// 将下单商品请求转换为订单商品模型
func (c *OrderGoodsCase) convertToModel(ctx context.Context, member bool, goods *app.CreateOrderGoods) (*models.OrderGoods, error) {
	// 查询商品信息和规格信息
	goodsQuery := c.goodsInfoCase.Query(ctx).Goods
	goodsInfo, err := c.goodsInfoCase.Find(ctx,
		repo.Where(goodsQuery.ID.Eq(goods.GoodsId)),
		repo.Where(goodsQuery.Status.Eq(int32(common.Status_ENABLE))),
	)
	if err != nil {
		return nil, err
	}
	skuQuery := c.goodsSkuCase.Query(ctx).GoodsSku
	var goodsSku *models.GoodsSku
	goodsSku, err = c.goodsSkuCase.Find(ctx,
		repo.Where(skuQuery.SkuCode.Eq(goods.SkuCode)),
		repo.Where(skuQuery.GoodsID.Eq(goods.GoodsId)),
	)
	if err != nil {
		return nil, err
	}
	picture := goodsInfo.Picture
	if len(goodsSku.Picture) > 0 {
		// 规格图片优先级高于商品主图
		picture = goodsSku.Picture
	}

	// 支付价格
	payPrice := goodsSku.Price
	if member {
		payPrice = goodsSku.DiscountPrice
	}

	res := &models.OrderGoods{
		GoodsID:       goodsInfo.ID,
		SkuCode:       goodsSku.SkuCode,
		Picture:       picture,
		Name:          goodsInfo.Name,
		Num:           goods.Num,
		SpecItem:      goodsSku.SpecItem,
		Price:         goodsSku.Price,
		PayPrice:      payPrice,
		TotalPrice:    goodsSku.Price * goods.Num,
		TotalPayPrice: payPrice * goods.Num,
	}
	return res, nil
}
