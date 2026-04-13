package biz

import (
	"context"
	"errors"
	recommendCore "shop/pkg/recommend/core"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// OrderGoodsCase 订单商品明细业务处理对象
type OrderGoodsCase struct {
	*biz.BaseCase
	*data.OrderGoodsRepo
	goodsInfoCase *GoodsInfoCase
	goodsSkuCase  *GoodsSkuCase
	mapper        *mapper.CopierMapper[app.OrderGoods, models.OrderGoods]
}

// NewOrderGoodsCase 创建订单商品明细业务处理对象
func NewOrderGoodsCase(baseCase *biz.BaseCase, orderGoodsRepo *data.OrderGoodsRepo,
	goodsInfoCase *GoodsInfoCase,
	goodsSkuCase *GoodsSkuCase,
) *OrderGoodsCase {
	orderGoodsMapper := mapper.NewCopierMapper[app.OrderGoods, models.OrderGoods]()
	orderGoodsMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &OrderGoodsCase{
		BaseCase:       baseCase,
		OrderGoodsRepo: orderGoodsRepo,
		goodsInfoCase:  goodsInfoCase,
		goodsSkuCase:   goodsSkuCase,
		mapper:         orderGoodsMapper,
	}
}

// mapByOrderIds 按订单编号批量查询商品明细映射
func (c *OrderGoodsCase) mapByOrderIds(ctx context.Context, orderIds []int64) (map[int64][]*app.OrderGoods, error) {
	res := make(map[int64][]*app.OrderGoods)
	// 存在订单编号时，批量查询对应的商品明细映射。
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
			// 当前订单首次命中时，先初始化切片容器。
			if !ok {
				v = make([]*app.OrderGoods, 0)
			}
			v = append(v, c.mapper.ToDTO(item))

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
		list = append(list, c.mapper.ToDTO(item))
	}
	return list, nil
}

// createByOrder 批量创建订单商品记录
func (c *OrderGoodsCase) createByOrder(ctx context.Context, orderId int64, goods []*models.OrderGoods) error {
	// 订单商品为空时，禁止继续创建订单明细。
	if len(goods) == 0 {
		return errors.New("订单商品信息不能为空")
	}
	for _, item := range goods {
		item.OrderID = orderId
	}
	return c.BatchCreate(ctx, goods)
}

// listGoodsIdsByOrderId 查询订单中的商品 ID 列表。
func (c *OrderGoodsCase) listGoodsIdsByOrderId(ctx context.Context, orderId int64) ([]int64, error) {
	query := c.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.OrderID.Eq(orderId)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return recommendCore.DedupeInt64s(goodsIds), nil
}

// convertToModelList 将下单商品列表转换为模型列表
func (c *OrderGoodsCase) convertToModelList(ctx context.Context, goods []*app.CreateOrderInfoGoods) ([]*models.OrderGoods, error) {
	// 根据登录信息判断当前下单用户是否享受会员价
	member := utils.IsMember(ctx)

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

// convertToProtoByCreateOrderInfoGoods 预览下单商品信息
func (c *OrderGoodsCase) convertToProtoByCreateOrderInfoGoods(ctx context.Context, member bool, item *app.CreateOrderInfoGoods) (*app.OrderGoods, error) {
	model, err := c.convertToModel(ctx, member, item)
	if err != nil {
		return nil, err
	}
	return c.mapper.ToDTO(model), nil
}

// 将下单商品请求转换为订单商品模型
func (c *OrderGoodsCase) convertToModel(ctx context.Context, member bool, goods *app.CreateOrderInfoGoods) (*models.OrderGoods, error) {
	// 查询商品信息和规格信息
	goodsQuery := c.goodsInfoCase.Query(ctx).GoodsInfo
	goodsOpts := make([]repo.QueryOption, 0, 2)
	goodsOpts = append(goodsOpts, repo.Where(goodsQuery.ID.Eq(goods.GoodsId)))
	goodsOpts = append(goodsOpts, repo.Where(goodsQuery.Status.Eq(int32(common.Status_ENABLE))))
	goodsInfo, err := c.goodsInfoCase.Find(ctx, goodsOpts...)
	if err != nil {
		return nil, err
	}
	skuQuery := c.goodsSkuCase.Query(ctx).GoodsSku
	var goodsSku *models.GoodsSku
	skuOpts := make([]repo.QueryOption, 0, 2)
	skuOpts = append(skuOpts, repo.Where(skuQuery.SkuCode.Eq(goods.SkuCode)))
	skuOpts = append(skuOpts, repo.Where(skuQuery.GoodsID.Eq(goods.GoodsId)))
	goodsSku, err = c.goodsSkuCase.Find(ctx, skuOpts...)
	if err != nil {
		return nil, err
	}
	picture := goodsInfo.Picture
	// 规格图存在时，优先使用规格图作为订单商品展示图。
	if len(goodsSku.Picture) > 0 {
		// 规格图片优先级高于商品主图
		picture = goodsSku.Picture
	}

	// 支付价格
	payPrice := goodsSku.Price
	// 会员下单时，优先使用会员价计算支付金额。
	if member {
		payPrice = goodsSku.DiscountPrice
	}
	recommendContext := goods.GetRecommendContext()

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
		Scene:         int32(recommendContext.GetScene()),
		RequestID:     recommendContext.GetRequestId(),
		Position:      recommendContext.GetPosition(),
	}
	return res, nil
}
