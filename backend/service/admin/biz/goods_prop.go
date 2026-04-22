package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsPropCase 商品属性业务实例
type GoodsPropCase struct {
	*biz.BaseCase
	*data.GoodsPropRepo
	mapper *mapper.CopierMapper[admin.GoodsProp, models.GoodsProp]
}

// NewGoodsPropCase 创建商品属性业务实例
func NewGoodsPropCase(baseCase *biz.BaseCase, goodsPropRepo *data.GoodsPropRepo) *GoodsPropCase {
	return &GoodsPropCase{
		BaseCase:      baseCase,
		GoodsPropRepo: goodsPropRepo,
		mapper:        mapper.NewCopierMapper[admin.GoodsProp, models.GoodsProp](),
	}
}

// PageGoodsProp 分页查询商品属性
func (c *GoodsPropCase) PageGoodsProp(ctx context.Context, req *admin.PageGoodsPropRequest) (*admin.PageGoodsPropResponse, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	// 传入商品编号时，仅查询对应商品的属性。
	if req.GetGoodsId() > 0 {
		opts = append(opts, repo.Where(query.GoodsID.Eq(req.GetGoodsId())))
	}
	// 传入属性名时，按属性名模糊匹配。
	if req.GetLabel() != "" {
		opts = append(opts, repo.Where(query.Label.Like("%"+req.GetLabel()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsProp, 0, len(list))
	for _, item := range list {
		goodsProp := c.mapper.ToDTO(item)
		resList = append(resList, goodsProp)
	}
	return &admin.PageGoodsPropResponse{List: resList, Total: int32(total)}, nil
}

// GetGoodsProp 获取商品属性
func (c *GoodsPropCase) GetGoodsProp(ctx context.Context, id int64) (*admin.GoodsProp, error) {
	goodsProp, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.mapper.ToDTO(goodsProp)
	return res, nil
}

// CreateGoodsProp 创建商品属性
func (c *GoodsPropCase) CreateGoodsProp(ctx context.Context, req *admin.GoodsProp) error {
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
func (c *GoodsPropCase) UpdateGoodsProp(ctx context.Context, req *admin.GoodsProp) error {
	goodsProp := c.mapper.ToEntity(req)
	err := c.UpdateById(ctx, goodsProp)
	if err != nil {
		// 命中商品属性唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("商品属性重复", "goods_prop", "label", "unique_goods_prop").WithCause(err)
		}
		return err
	}
	return nil
}

// ListGoodsPropByGoodsId 按商品查询属性列表
func (c *GoodsPropCase) ListGoodsPropByGoodsId(ctx context.Context, goodsId int64) ([]*admin.GoodsProp, error) {
	query := c.Query(ctx).GoodsProp
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsProp, 0, len(list))
	for _, item := range list {
		goodsProp := c.mapper.ToDTO(item)
		resList = append(resList, goodsProp)
	}
	return resList, nil
}

// DeleteGoodsProp 删除商品属性
func (c *GoodsPropCase) DeleteGoodsProp(ctx context.Context, id string) error {
	goodsIds := _string.ConvertStringToInt64Array(id)
	return c.DeleteByIds(ctx, goodsIds)
}
