package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsSpecCase 商品规格业务实例
type GoodsSpecCase struct {
	*biz.BaseCase
	*data.GoodsSpecRepo
	mapper *mapper.CopierMapper[admin.GoodsSpec, models.GoodsSpec]
}

// NewGoodsSpecCase 创建商品规格业务实例
func NewGoodsSpecCase(baseCase *biz.BaseCase, goodsSpecRepo *data.GoodsSpecRepo) *GoodsSpecCase {
	goodsSpecMapper := mapper.NewCopierMapper[admin.GoodsSpec, models.GoodsSpec]()
	goodsSpecMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &GoodsSpecCase{BaseCase: baseCase, GoodsSpecRepo: goodsSpecRepo, mapper: goodsSpecMapper}
}

// ListGoodsSpec 查询商品规格列表
func (c *GoodsSpecCase) ListGoodsSpec(ctx context.Context, req *admin.ListGoodsSpecRequest) (*admin.ListGoodsSpecResponse, error) {
	query := c.Query(ctx).GoodsSpec
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Where(query.GoodsID.Eq(req.GetGoodsId())))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsSpec, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.mapper.ToDTO(item))
	}
	return &admin.ListGoodsSpecResponse{List: resList}, nil
}
