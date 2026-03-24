package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsSkuCase 商品规格库存业务处理对象
type GoodsSkuCase struct {
	*biz.BaseCase
	*data.GoodsSkuRepo
}

// NewGoodsSkuCase 创建商品规格库存业务处理对象
func NewGoodsSkuCase(baseCase *biz.BaseCase, goodsSkuRepo *data.GoodsSkuRepo) *GoodsSkuCase {
	return &GoodsSkuCase{
		BaseCase:     baseCase,
		GoodsSkuRepo: goodsSkuRepo,
	}
}

// 按规格编码批量查询并返回映射
func (c *GoodsSkuCase) mapBySkuCodes(ctx context.Context, skuCodes []string) (map[string]*models.GoodsSku, error) {
	res := make(map[string]*models.GoodsSku)
	if len(skuCodes) > 0 {
		query := c.Query(ctx).GoodsSku
		all, err := c.List(ctx,
			repo.Where(query.SkuCode.In(skuCodes...)),
		)
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
func (c *GoodsSkuCase) listByGoodsId(ctx context.Context, goodsId int64, member bool) ([]*app.GoodsResponse_Sku, error) {
	query := c.Query(ctx).GoodsSku
	all, err := c.List(ctx,
		repo.Where(query.GoodsID.Eq(goodsId)),
	)
	if err != nil {
		return nil, err
	}
	list := make([]*app.GoodsResponse_Sku, 0)
	for _, item := range all {
		list = append(list, c.convertToProto(item, member))
	}
	return list, nil
}

// 将规格库存模型转换为接口响应
func (c *GoodsSkuCase) convertToProto(item *models.GoodsSku, member bool) *app.GoodsResponse_Sku {
	price := item.Price
	if member {
		price = item.DiscountPrice
	}
	res := &app.GoodsResponse_Sku{
		Picture:   item.Picture,
		SpecItem:  _string.ConvertJsonStringToStringArray(item.SpecItem),
		SkuCode:   item.SkuCode,
		Price:     price,
		SaleNum:   item.InitSaleNum + item.RealSaleNum,
		Inventory: item.Inventory,
	}
	return res
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
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}
