package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsSkuCase 商品规格库存业务处理对象
type GoodsSkuCase struct {
	*biz.BaseCase
	*data.GoodsSkuRepo
	mapper *mapper.CopierMapper[app.GoodsInfoResponse_Sku, models.GoodsSku]
}

// NewGoodsSkuCase 创建商品规格库存业务处理对象
func NewGoodsSkuCase(baseCase *biz.BaseCase, goodsSkuRepo *data.GoodsSkuRepo) *GoodsSkuCase {
	return &GoodsSkuCase{
		BaseCase:     baseCase,
		GoodsSkuRepo: goodsSkuRepo,
		mapper: func() *mapper.CopierMapper[app.GoodsInfoResponse_Sku, models.GoodsSku] {
			m := mapper.NewCopierMapper[app.GoodsInfoResponse_Sku, models.GoodsSku]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// 按规格编码批量查询并返回映射
func (c *GoodsSkuCase) mapBySkuCodes(ctx context.Context, skuCodes []string) (map[string]*models.GoodsSku, error) {
	res := make(map[string]*models.GoodsSku)
	// 仅在存在 SKU 编码时，才访问数据库构建映射。
	if len(skuCodes) > 0 {
		query := c.Query(ctx).GoodsSku
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.SkuCode.In(skuCodes...)))
		all, err := c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range all {
			res[item.SkuCode] = item
		}
	}
	return res, nil
}

// 查询商品下的全部规格库存列表
func (c *GoodsSkuCase) listByGoodsId(ctx context.Context, goodsId int64, member bool) ([]*app.GoodsInfoResponse_Sku, error) {
	query := c.Query(ctx).GoodsSku
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.GoodsInfoResponse_Sku, 0)
	for _, item := range all {
		price := item.Price
		// 会员用户优先使用会员价展示 SKU。
		if member {
			price = item.DiscountPrice
		}
		sku := c.mapper.ToDTO(item)
		sku.Price = price
		sku.SaleNum = item.InitSaleNum + item.RealSaleNum
		list = append(list, sku)
	}
	return list, nil
}

// 增加规格销量并扣减库存
func (c *GoodsSkuCase) addSaleNum(ctx context.Context, skuCode string, num int64) error {
	query := c.Query(ctx).GoodsSku
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Add(num),
		"inventory":     query.Inventory.Sub(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.SkuCode.Eq(skuCode), query.Inventory.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 未命中任何规格时，视为无需更新直接返回。
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}

// 回退规格销量并恢复库存
func (c *GoodsSkuCase) subSaleNum(ctx context.Context, skuCode string, num int64) error {
	query := c.Query(ctx).GoodsSku
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Sub(num),
		"inventory":     query.Inventory.Add(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.SkuCode.Eq(skuCode), query.RealSaleNum.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 未命中任何规格时，视为无需回退直接返回。
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}
