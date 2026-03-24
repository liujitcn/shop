package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsSpecCase 商品规格业务实例
type GoodsSpecCase struct {
	*biz.BaseCase
	*data.GoodsSpecRepo
}

// NewGoodsSpecCase 创建商品规格业务实例
func NewGoodsSpecCase(baseCase *biz.BaseCase, goodsSpecRepo *data.GoodsSpecRepo) *GoodsSpecCase {
	return &GoodsSpecCase{BaseCase: baseCase, GoodsSpecRepo: goodsSpecRepo}
}

// ListGoodsSpec 查询商品规格列表
func (c *GoodsSpecCase) ListGoodsSpec(ctx context.Context, req *admin.ListGoodsSpecRequest) (*admin.ListGoodsSpecResponse, error) {
	query := c.Query(ctx).GoodsSpec
	list, err := c.List(ctx, repo.Where(query.GoodsID.Eq(req.GetGoodsId())))
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsSpec, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toGoodsSpec(item))
	}
	return &admin.ListGoodsSpecResponse{List: resList}, nil
}

// toGoodsSpec 转换商品规格响应数据
func (c *GoodsSpecCase) toGoodsSpec(item *models.GoodsSpec) *admin.GoodsSpec {
	return &admin.GoodsSpec{
		Id:      item.ID,
		GoodsId: item.GoodsID,
		Name:    item.Name,
		Item:    _string.ConvertJsonStringToStringArray(item.Item),
		Sort:    item.Sort,
	}
}
