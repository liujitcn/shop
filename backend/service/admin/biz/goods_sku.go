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

// GoodsSkuCase 商品规格项业务实例
type GoodsSkuCase struct {
	*biz.BaseCase
	*data.GoodsSkuRepo
	mapper *mapper.CopierMapper[admin.GoodsSku, models.GoodsSku]
}

// NewGoodsSkuCase 创建商品规格项业务实例
func NewGoodsSkuCase(baseCase *biz.BaseCase, goodsSkuRepo *data.GoodsSkuRepo) *GoodsSkuCase {
	return &GoodsSkuCase{
		BaseCase:     baseCase,
		GoodsSkuRepo: goodsSkuRepo,
		mapper:       mapper.NewCopierMapper[admin.GoodsSku, models.GoodsSku](),
	}
}

// PageGoodsSku 分页查询商品规格项
func (c *GoodsSkuCase) PageGoodsSku(ctx context.Context, req *admin.PageGoodsSkuRequest) (*admin.PageGoodsSkuResponse, error) {
	query := c.Query(ctx).GoodsSku
	opts := make([]repo.QueryOption, 0, 2)
	if req.GetGoodsId() > 0 {
		opts = append(opts, repo.Where(query.GoodsID.Eq(req.GetGoodsId())))
	}
	if req.GetSkuCode() != "" {
		opts = append(opts, repo.Where(query.SkuCode.Like("%"+req.GetSkuCode()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsSku, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toGoodsSku(item))
	}
	return &admin.PageGoodsSkuResponse{List: resList, Total: int32(total)}, nil
}

// GetGoodsSku 获取商品规格项
func (c *GoodsSkuCase) GetGoodsSku(ctx context.Context, id int64) (*admin.GoodsSku, error) {
	goodsSku, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toGoodsSku(goodsSku), nil
}

// UpdateGoodsSku 更新商品规格项
func (c *GoodsSkuCase) UpdateGoodsSku(ctx context.Context, req *admin.GoodsSku) error {
	return c.UpdateById(ctx, c.toGoodsSkuModel(req))
}

// ListGoodsSkuByGoodsId 按商品查询规格项列表
func (c *GoodsSkuCase) ListGoodsSkuByGoodsId(ctx context.Context, goodsId int64) ([]*admin.GoodsSku, error) {
	query := c.Query(ctx).GoodsSku
	list, err := c.List(ctx, repo.Where(query.GoodsID.Eq(goodsId)))
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsSku, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toGoodsSku(item))
	}
	return resList, nil
}

// toGoodsSku 转换商品规格项响应数据
func (c *GoodsSkuCase) toGoodsSku(item *models.GoodsSku) *admin.GoodsSku {
	goodsSku := c.mapper.ToDTO(item)
	goodsSku.SpecItem = _string.ConvertJsonStringToStringArray(item.SpecItem)
	return goodsSku
}

// toGoodsSkuModel 转换商品规格项模型数据
func (c *GoodsSkuCase) toGoodsSkuModel(item *admin.GoodsSku) *models.GoodsSku {
	goodsSku := c.mapper.ToEntity(item)
	goodsSku.SpecItem = _string.ConvertStringArrayToString(item.GetSpecItem())
	return goodsSku
}
