package biz

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// GoodsSKUCase 商品规格库存业务处理对象
type GoodsSKUCase struct {
	*biz.BaseCase
	*data.GoodsSKURepository
	mapper *mapper.CopierMapper[appv1.GoodsInfoResponse_Sku, models.GoodsSKU]
}

// NewGoodsSKUCase 创建商品规格库存业务处理对象
func NewGoodsSKUCase(baseCase *biz.BaseCase, goodsSKURepo *data.GoodsSKURepository) *GoodsSKUCase {
	return &GoodsSKUCase{
		BaseCase:           baseCase,
		GoodsSKURepository: goodsSKURepo,
		mapper: func() *mapper.CopierMapper[appv1.GoodsInfoResponse_Sku, models.GoodsSKU] {
			m := mapper.NewCopierMapper[appv1.GoodsInfoResponse_Sku, models.GoodsSKU]()
			m.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
			return m
		}(),
	}
}

// 按规格编码批量查询并返回映射
func (c *GoodsSKUCase) mapBySKUCodes(ctx context.Context, skuCodes []string) (map[string]*models.GoodsSKU, error) {
	res := make(map[string]*models.GoodsSKU)
	// 仅在存在 SKU 编码时，才访问数据库构建映射。
	if len(skuCodes) > 0 {
		query := c.Query(ctx).GoodsSKU
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.SKUCode.In(skuCodes...)))
		all, err := c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range all {
			res[item.SKUCode] = item
		}
	}
	return res, nil
}

// 查询商品下的全部规格库存列表
func (c *GoodsSKUCase) listByGoodsID(ctx context.Context, goodsID int64, member bool) ([]*appv1.GoodsInfoResponse_Sku, error) {
	query := c.Query(ctx).GoodsSKU
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*appv1.GoodsInfoResponse_Sku, 0)
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

// findByGoodsIDAndSKUCode 按商品编号和规格编码查询规格。
func (c *GoodsSKUCase) findByGoodsIDAndSKUCode(ctx context.Context, goodsID int64, skuCode string) (*models.GoodsSKU, error) {
	query := c.Query(ctx).GoodsSKU
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.SKUCode.Eq(skuCode)))
	goodsSKU, err := c.Find(ctx, opts...)
	if err != nil {
		// 规格不存在时，在 biz 层返回稳定可判断的资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("商品规格不存在").WithCause(err)
		}
		return nil, err
	}
	return goodsSKU, nil
}

// ensureEnoughInventory 校验规格库存是否满足目标数量。
func (c *GoodsSKUCase) ensureEnoughInventory(goodsSKU *models.GoodsSKU, num int64) error {
	// 规格记录缺失时，直接返回资源不存在。
	if goodsSKU == nil {
		return errorsx.ResourceNotFound("商品规格不存在")
	}
	// 购买数量非法时，直接拦截当前请求。
	if num <= 0 {
		return errorsx.InvalidArgument("商品购买数量必须大于0")
	}
	// 当前规格库存不足时，返回结构化冲突错误，便于前端稳定提示。
	if goodsSKU.Inventory < num {
		return errorsx.StateConflict(
			"商品库存不足",
			"goods_sku",
			strconv.FormatInt(goodsSKU.Inventory, 10),
			strconv.FormatInt(num, 10),
		)
	}
	return nil
}

// 增加规格销量并扣减库存
func (c *GoodsSKUCase) addSaleNum(ctx context.Context, skuCode string, num int64) error {
	query := c.Query(ctx).GoodsSKU
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Add(num),
		"inventory":     query.Inventory.Sub(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.SKUCode.Eq(skuCode), query.Inventory.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 未命中更新时，需要把“规格不存在”和“库存不足”区分成可判断的业务错误。
	if res.RowsAffected == 0 {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.SKUCode.Eq(skuCode)))
		goodsSKU, findErr := c.Find(ctx, opts...)
		// 规格已经不存在时，当前下单请求不应继续执行。
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errorsx.ResourceNotFound("商品规格不存在").WithCause(findErr)
			}
			return findErr
		}
		return errorsx.StateConflict(
			"商品库存不足",
			"goods_sku",
			strconv.FormatInt(goodsSKU.Inventory, 10),
			strconv.FormatInt(num, 10),
		)
	}
	return res.Error
}

// 回退规格销量并恢复库存
func (c *GoodsSKUCase) subSaleNum(ctx context.Context, skuCode string, num int64) error {
	query := c.Query(ctx).GoodsSKU
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Sub(num),
		"inventory":     query.Inventory.Add(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.SKUCode.Eq(skuCode), query.RealSaleNum.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 回退未命中时，说明规格已不存在或销量数据已经异常。
	if res.RowsAffected == 0 {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.SKUCode.Eq(skuCode)))
		goodsSKU, findErr := c.Find(ctx, opts...)
		// 规格记录缺失时，当前库存回退已经无法可靠执行。
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errorsx.Internal("商品库存回退失败，商品规格不存在").WithCause(findErr)
			}
			return findErr
		}
		return errorsx.Internal(
			fmt.Sprintf(
				"商品库存回退失败，规格销量数据异常：skuCode=%s，当前销量=%d，回退数量=%d",
				skuCode,
				goodsSKU.RealSaleNum,
				num,
			),
		)
	}
	return res.Error
}
