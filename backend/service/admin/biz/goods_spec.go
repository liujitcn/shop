package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsSpecCase 商品规格业务实例
type GoodsSpecCase struct {
	*biz.BaseCase
	*data.GoodsSpecRepository
	mapper *mapper.CopierMapper[adminv1.GoodsSpec, models.GoodsSpec]
}

// NewGoodsSpecCase 创建商品规格业务实例
func NewGoodsSpecCase(baseCase *biz.BaseCase, goodsSpecRepo *data.GoodsSpecRepository) *GoodsSpecCase {
	goodsSpecMapper := mapper.NewCopierMapper[adminv1.GoodsSpec, models.GoodsSpec]()
	goodsSpecMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &GoodsSpecCase{BaseCase: baseCase, GoodsSpecRepository: goodsSpecRepo, mapper: goodsSpecMapper}
}

// ListGoodsSpecs 查询商品规格列表
func (c *GoodsSpecCase) ListGoodsSpecs(ctx context.Context, req *adminv1.ListGoodsSpecsRequest) (*adminv1.ListGoodsSpecsResponse, error) {
	query := c.Query(ctx).GoodsSpec
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Where(query.GoodsID.Eq(req.GetGoodsId())))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.GoodsSpec, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.mapper.ToDTO(item))
	}
	return &adminv1.ListGoodsSpecsResponse{GoodsSpecs: resList}, nil
}
