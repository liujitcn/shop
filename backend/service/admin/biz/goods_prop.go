package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
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
	opts := make([]repo.QueryOption, 0, 2)
	if req.GetGoodsId() > 0 {
		opts = append(opts, repo.Where(query.GoodsID.Eq(req.GetGoodsId())))
	}
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
	return c.Create(ctx, goodsProp)
}

// UpdateGoodsProp 更新商品属性
func (c *GoodsPropCase) UpdateGoodsProp(ctx context.Context, req *admin.GoodsProp) error {
	goodsProp := c.mapper.ToEntity(req)
	return c.UpdateById(ctx, goodsProp)
}

// ListGoodsPropByGoodsId 按商品查询属性列表
func (c *GoodsPropCase) ListGoodsPropByGoodsId(ctx context.Context, goodsId int64) ([]*admin.GoodsProp, error) {
	query := c.Query(ctx).GoodsProp
	list, err := c.List(ctx, repo.Where(query.GoodsID.Eq(goodsId)))
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
