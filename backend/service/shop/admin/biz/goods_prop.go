package biz

import (
	"context"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
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
	goodsInfoRepo *data.GoodsInfoRepository
	mapper        *mapper.CopierMapper[shopadminv1.GoodsProp, models.GoodsProp]
}

// NewGoodsPropCase 创建商品属性业务实例
func NewGoodsPropCase(baseCase *biz.BaseCase, goodsPropRepo *data.GoodsPropRepository, goodsInfoRepo *data.GoodsInfoRepository) *GoodsPropCase {
	return &GoodsPropCase{
		BaseCase:            baseCase,
		GoodsPropRepository: goodsPropRepo,
		goodsInfoRepo:       goodsInfoRepo,
		mapper:              mapper.NewCopierMapper[shopadminv1.GoodsProp, models.GoodsProp](),
	}
}

// PageGoodsProp 查询商品属性列表
func (c *GoodsPropCase) PageGoodsProp(ctx context.Context, req *shopadminv1.PageGoodsPropRequest) (*shopadminv1.PageGoodsPropResponse, error) {
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

	resList := make([]*shopadminv1.GoodsProp, 0, len(list))
	for _, item := range list {
		goodsProp := c.mapper.ToDTO(item)
		resList = append(resList, goodsProp)
	}
	return &shopadminv1.PageGoodsPropResponse{GoodsProps: resList, Total: int32(total)}, nil
}

// GetGoodsProp 获取商品属性
func (c *GoodsPropCase) GetGoodsProp(ctx context.Context, id int64) (*shopadminv1.GoodsProp, error) {
	goodsProp, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.mapper.ToDTO(goodsProp)
	return res, nil
}

// CreateGoodsProp 创建商品属性
func (c *GoodsPropCase) CreateGoodsProp(ctx context.Context, req *shopadminv1.GoodsProp) error {
	if req.GetGoodsId() <= 0 {
		return errorsx.InvalidArgument("商品属性参数不合法")
	}
	goodsProp := c.mapper.ToEntity(req)
	goodsInfo, err := c.goodsInfoRepo.FindByID(ctx, req.GetGoodsId())
	if err != nil {
		return errorsx.ResourceNotFound("商品不存在").WithCause(err)
	}
	goodsProp.TenantID = goodsInfo.TenantID
	goodsProp.TenantStoreID = goodsInfo.TenantStoreID
	err = c.Create(ctx, goodsProp)
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
func (c *GoodsPropCase) UpdateGoodsProp(ctx context.Context, req *shopadminv1.GoodsProp) error {
	if req.GetId() <= 0 || req.GetGoodsId() <= 0 {
		return errorsx.InvalidArgument("商品属性参数不合法")
	}
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

// DeleteGoodsProp 删除商品属性
func (c *GoodsPropCase) DeleteGoodsProp(ctx context.Context, id string) error {
	goodsIDs := _string.ConvertStringToInt64Array(id)
	return c.DeleteByIDs(ctx, goodsIDs)
}

// ListGoodsPropByGoodsID 按商品查询属性列表
func (c *GoodsPropCase) ListGoodsPropByGoodsID(ctx context.Context, goodsID int64) ([]*shopadminv1.GoodsProp, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*shopadminv1.GoodsProp, 0, len(list))
	for _, item := range list {
		goodsProp := c.mapper.ToDTO(item)
		resList = append(resList, goodsProp)
	}
	return resList, nil
}
