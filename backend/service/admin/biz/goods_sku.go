package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsSKUCase 商品规格项业务实例
type GoodsSKUCase struct {
	*biz.BaseCase
	*data.GoodsSKURepository
	mapper *mapper.CopierMapper[adminv1.GoodsSku, models.GoodsSKU]
}

// NewGoodsSKUCase 创建商品规格项业务实例
func NewGoodsSKUCase(baseCase *biz.BaseCase, goodsSKURepo *data.GoodsSKURepository) *GoodsSKUCase {
	goodsSKUMapper := mapper.NewCopierMapper[adminv1.GoodsSku, models.GoodsSKU]()
	goodsSKUMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &GoodsSKUCase{
		BaseCase:           baseCase,
		GoodsSKURepository: goodsSKURepo,
		mapper:             goodsSKUMapper,
	}
}

// ListGoodsSKUs 查询商品规格项列表
func (c *GoodsSKUCase) ListGoodsSKUs(ctx context.Context, req *adminv1.PageGoodsSkusRequest) (*adminv1.PageGoodsSkusResponse, error) {
	query := c.Query(ctx).GoodsSKU
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.SKUCode.Asc()))
	// 传入商品编号时，仅查询对应商品的 SKU。
	if req.GetGoodsId() > 0 {
		opts = append(opts, repository.Where(query.GoodsID.Eq(req.GetGoodsId())))
	}
	// 传入 SKU 编码时，按编码模糊匹配 SKU。
	if req.GetSkuCode() != "" {
		opts = append(opts, repository.Where(query.SKUCode.Like("%"+req.GetSkuCode()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.GoodsSku, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toGoodsSKU(item))
	}
	return &adminv1.PageGoodsSkusResponse{GoodsSkus: resList, Total: int32(total)}, nil
}

// GetGoodsSKU 获取商品规格项
func (c *GoodsSKUCase) GetGoodsSKU(ctx context.Context, id int64) (*adminv1.GoodsSku, error) {
	goodsSKU, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toGoodsSKU(goodsSKU), nil
}

// UpdateGoodsSKU 更新商品规格项
func (c *GoodsSKUCase) UpdateGoodsSKU(ctx context.Context, req *adminv1.GoodsSku) error {
	err := c.UpdateByID(ctx, c.toGoodsSKUModel(req))
	if err != nil {
		// 命中 SKU 编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("SKU编码重复", "goods_sku", "sku_code", "unique_goods_sku").WithCause(err)
		}
		return err
	}
	return nil
}

// ListGoodsSKUsByGoodsID 按商品查询规格项列表
func (c *GoodsSKUCase) ListGoodsSKUsByGoodsID(ctx context.Context, goodsID int64) ([]*adminv1.GoodsSku, error) {
	query := c.Query(ctx).GoodsSKU
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.GoodsSku, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toGoodsSKU(item))
	}
	return resList, nil
}

// toGoodsSKU 转换商品规格项响应数据
func (c *GoodsSKUCase) toGoodsSKU(item *models.GoodsSKU) *adminv1.GoodsSku {
	return c.mapper.ToDTO(item)
}

// toGoodsSKUModel 转换商品规格项模型数据
func (c *GoodsSKUCase) toGoodsSKUModel(item *adminv1.GoodsSku) *models.GoodsSKU {
	// 商品规格项为空时返回零值模型，保持批量转换流程兼容。
	if item == nil {
		return &models.GoodsSKU{}
	}
	return c.mapper.ToEntity(item)
}
