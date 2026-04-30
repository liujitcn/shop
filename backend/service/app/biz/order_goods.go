package biz

import (
	"context"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repository"
)

// OrderGoodsCase 订单商品明细业务处理对象
type OrderGoodsCase struct {
	*biz.BaseCase
	*data.OrderGoodsRepository
	goodsInfoCase *GoodsInfoCase
	goodsSKUCase  *GoodsSKUCase
	mapper        *mapper.CopierMapper[appv1.OrderGoods, models.OrderGoods]
}

// NewOrderGoodsCase 创建订单商品明细业务处理对象
func NewOrderGoodsCase(baseCase *biz.BaseCase, orderGoodsRepo *data.OrderGoodsRepository,
	goodsInfoCase *GoodsInfoCase,
	goodsSKUCase *GoodsSKUCase,
) *OrderGoodsCase {
	orderGoodsMapper := mapper.NewCopierMapper[appv1.OrderGoods, models.OrderGoods]()
	orderGoodsMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &OrderGoodsCase{
		BaseCase:             baseCase,
		OrderGoodsRepository: orderGoodsRepo,
		goodsInfoCase:        goodsInfoCase,
		goodsSKUCase:         goodsSKUCase,
		mapper:               orderGoodsMapper,
	}
}

// mapByOrderIDs 按订单编号批量查询商品明细映射
func (c *OrderGoodsCase) mapByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]*appv1.OrderGoods, error) {
	res := make(map[int64][]*appv1.OrderGoods)
	// 存在订单编号时，批量查询对应的商品明细映射。
	if len(orderIDs) > 0 {
		query := c.Query(ctx).OrderGoods
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.OrderID.In(orderIDs...)))
		all, err := c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range all {
			v, ok := res[item.OrderID]
			// 当前订单首次命中时，先初始化切片容器。
			if !ok {
				v = make([]*appv1.OrderGoods, 0)
			}
			v = append(v, c.toOrderGoods(item))

			res[item.OrderID] = v
		}
	}
	return res, nil
}

// listByOrderID 查询单个订单的商品明细
func (c *OrderGoodsCase) listByOrderID(ctx context.Context, orderID int64) ([]*appv1.OrderGoods, error) {
	query := c.Query(ctx).OrderGoods
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*appv1.OrderGoods, 0)
	for _, item := range all {
		list = append(list, c.toOrderGoods(item))
	}
	return list, nil
}

// createByOrder 批量创建订单商品记录
func (c *OrderGoodsCase) createByOrder(ctx context.Context, orderID int64, goods []*models.OrderGoods) error {
	// 订单商品为空时，禁止继续创建订单明细。
	if len(goods) == 0 {
		return errorsx.InvalidArgument("订单商品信息不能为空")
	}
	for _, item := range goods {
		item.OrderID = orderID
	}
	return c.BatchCreate(ctx, goods)
}

// listGoodsIDsByOrderID 查询订单中的商品 ID 列表。
func (c *OrderGoodsCase) listGoodsIDsByOrderID(ctx context.Context, orderID int64) ([]int64, error) {
	query := c.Query(ctx).OrderGoods
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIDs := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIDs = append(goodsIDs, item.GoodsID)
	}
	return _slice.Unique(goodsIDs), nil
}

// toOrderGoods 转换订单商品响应并补齐推荐上下文。
func (c *OrderGoodsCase) toOrderGoods(item *models.OrderGoods) *appv1.OrderGoods {
	orderGoods := c.mapper.ToDTO(item)
	orderGoods.RecommendContext = &appv1.RecommendContext{
		Scene:     commonv1.RecommendScene(item.Scene),
		RequestId: item.RequestID,
		Position:  item.Position,
	}
	return orderGoods
}

// 将下单商品请求转换为订单商品模型
func (c *OrderGoodsCase) convertToModel(ctx context.Context, member bool, goods *appv1.CreateOrderInfoGoods) (*models.OrderGoods, error) {
	// 下单商品明细为空时，无法继续生成订单快照。
	if goods == nil {
		return nil, errorsx.InvalidArgument("订单商品信息不能为空")
	}
	// 购买数量非法时，直接拦截当前下单请求。
	if goods.Num <= 0 {
		return nil, errorsx.InvalidArgument("商品购买数量必须大于0")
	}

	// 查询商品信息和规格信息
	goodsQuery := c.goodsInfoCase.Query(ctx).GoodsInfo
	goodsOpts := make([]repository.QueryOption, 0, 2)
	goodsOpts = append(goodsOpts, repository.Where(goodsQuery.ID.Eq(goods.GoodsId)))
	goodsOpts = append(goodsOpts, repository.Where(goodsQuery.Status.Eq(_const.STATUS_ENABLE)))
	goodsInfo, err := c.goodsInfoCase.Find(ctx, goodsOpts...)
	if err != nil {
		return nil, err
	}
	skuQuery := c.goodsSKUCase.Query(ctx).GoodsSKU
	var goodsSKU *models.GoodsSKU
	skuOpts := make([]repository.QueryOption, 0, 2)
	skuOpts = append(skuOpts, repository.Where(skuQuery.SKUCode.Eq(goods.SkuCode)))
	skuOpts = append(skuOpts, repository.Where(skuQuery.GoodsID.Eq(goods.GoodsId)))
	goodsSKU, err = c.goodsSKUCase.Find(ctx, skuOpts...)
	if err != nil {
		return nil, err
	}
	// 当前规格库存不足时，直接阻止继续创建订单。
	err = c.goodsSKUCase.ensureEnoughInventory(goodsSKU, goods.Num)
	if err != nil {
		return nil, err
	}
	picture := goodsInfo.Picture
	// 规格图存在时，优先使用规格图作为订单商品展示图。
	if len(goodsSKU.Picture) > 0 {
		// 规格图片优先级高于商品主图
		picture = goodsSKU.Picture
	}

	// 支付价格
	payPrice := goodsSKU.Price
	// 会员下单时，优先使用会员价计算支付金额。
	if member {
		payPrice = goodsSKU.DiscountPrice
	}
	recommendContext := goods.GetRecommendContext()
	// 下单商品未携带推荐上下文时，统一回退到空上下文，避免空指针并保持订单快照结构稳定。
	if recommendContext == nil {
		recommendContext = &appv1.RecommendContext{}
	}

	res := &models.OrderGoods{
		GoodsID:       goodsInfo.ID,
		SKUCode:       goodsSKU.SKUCode,
		Picture:       picture,
		Name:          goodsInfo.Name,
		Num:           goods.Num,
		SpecItem:      goodsSKU.SpecItem,
		Price:         goodsSKU.Price,
		PayPrice:      payPrice,
		TotalPrice:    goodsSKU.Price * goods.Num,
		TotalPayPrice: payPrice * goods.Num,
		Scene:         int32(recommendContext.GetScene()),
		RequestID:     recommendContext.GetRequestId(),
		Position:      recommendContext.GetPosition(),
	}
	return res, nil
}
