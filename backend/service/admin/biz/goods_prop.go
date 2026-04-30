package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsPropCase 商品属性业务实例
type GoodsPropCase struct {
	*biz.BaseCase
	*data.GoodsPropRepository
	mapper *mapper.CopierMapper[adminv1.GoodsProp, models.GoodsProp]
}

// NewGoodsPropCase 创建商品属性业务实例
func NewGoodsPropCase(baseCase *biz.BaseCase, goodsPropRepo *data.GoodsPropRepository) *GoodsPropCase {
	return &GoodsPropCase{
		BaseCase:            baseCase,
		GoodsPropRepository: goodsPropRepo,
		mapper:              mapper.NewCopierMapper[adminv1.GoodsProp, models.GoodsProp](),
	}
}

// PageGoodsProps 查询商品属性列表
func (c *GoodsPropCase) PageGoodsProps(ctx context.Context, req *adminv1.PageGoodsPropsRequest) (*adminv1.PageGoodsPropsResponse, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	// 传入商品编号时，仅查询对应商品的属性。
	if req.GetGoodsId() > 0 {
		opts = append(opts, repository.Where(query.GoodsID.Eq(req.GetGoodsId())))
	}
	// 传入属性名时，按属性名模糊匹配。
	if req.GetLabel() != "" {
		opts = append(opts, repository.Where(query.Label.Like("%"+req.GetLabel()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.GoodsProp, 0, len(list))
	for _, item := range list {
		goodsProp := c.mapper.ToDTO(item)
		resList = append(resList, goodsProp)
	}
	return &adminv1.PageGoodsPropsResponse{GoodsProps: resList, Total: int32(total)}, nil
}

// GetGoodsProp 获取商品属性
func (c *GoodsPropCase) GetGoodsProp(ctx context.Context, id int64) (*adminv1.GoodsProp, error) {
	goodsProp, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.mapper.ToDTO(goodsProp)
	return res, nil
}

// CreateGoodsProp 创建商品属性
func (c *GoodsPropCase) CreateGoodsProp(ctx context.Context, req *adminv1.GoodsProp) error {
	goodsProp := c.mapper.ToEntity(req)
	err := c.Create(ctx, goodsProp)
	if err != nil {
		// 命中商品属性唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("商品属性重复", "goods_prop", "label", "unique_goods_prop").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateGoodsProp 更新商品属性
func (c *GoodsPropCase) UpdateGoodsProp(ctx context.Context, req *adminv1.GoodsProp) error {
	goodsProp := c.mapper.ToEntity(req)
	err := c.UpdateByID(ctx, goodsProp)
	if err != nil {
		// 命中商品属性唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("商品属性重复", "goods_prop", "label", "unique_goods_prop").WithCause(err)
		}
		return err
	}
	return nil
}

// ListGoodsPropByGoodsID 按商品查询属性列表
func (c *GoodsPropCase) ListGoodsPropByGoodsID(ctx context.Context, goodsID int64) ([]*adminv1.GoodsProp, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.GoodsProp, 0, len(list))
	for _, item := range list {
		goodsProp := c.mapper.ToDTO(item)
		resList = append(resList, goodsProp)
	}
	return resList, nil
}

// DeleteGoodsProp 删除商品属性
func (c *GoodsPropCase) DeleteGoodsProp(ctx context.Context, id string) error {
	goodsIDs := _string.ConvertStringToInt64Array(id)
	return c.DeleteByIDs(ctx, goodsIDs)
}
