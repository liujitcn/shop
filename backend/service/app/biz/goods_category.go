package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/service/app/util"

	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsCategoryCase 商品分类业务处理对象
type GoodsCategoryCase struct {
	*biz.BaseCase
	*data.GoodsCategoryRepo
	goodsInfoRepo *data.GoodsRepo
}

// NewGoodsCategoryCase 创建商品分类业务处理对象
func NewGoodsCategoryCase(baseCase *biz.BaseCase, goodsCategoryRepo *data.GoodsCategoryRepo, goodsInfoRepo *data.GoodsRepo) *GoodsCategoryCase {
	return &GoodsCategoryCase{
		BaseCase:          baseCase,
		GoodsCategoryRepo: goodsCategoryRepo,
		goodsInfoRepo:     goodsInfoRepo,
	}
}

// ListGoodsCategory 查询分类列表
func (c *GoodsCategoryCase) ListGoodsCategory(ctx context.Context, req *app.ListGoodsCategoryRequest) (*app.ListGoodsCategoryResponse, error) {
	query := c.Query(ctx).GoodsCategory
	all, err := c.List(ctx,
		repo.Where(query.ParentID.Eq(req.GetParentId())),
		repo.Where(query.Status.Eq(int32(common.Status_ENABLE))),
	)
	if err != nil {
		return nil, err
	}

	member := util.IsMember(ctx)
	list := make([]*app.GoodsCategory, 0, len(all))
	for _, item := range all {
		category := c.convertToProto(item)
		// 二级分类需要同时返回分类下的推荐商品
		if category.ParentId > 0 {
			goodsQuery := c.goodsInfoRepo.Query(ctx).Goods
			var goodsList []*models.Goods
			goodsList, _, err = c.goodsInfoRepo.Page(ctx, 1, 9,
				repo.Where(goodsQuery.CategoryID.Eq(category.Id)),
				repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))),
			)
			if err != nil {
				return nil, err
			}
			category.Goods = make([]*app.Goods, 0, len(goodsList))
			for _, goods := range goodsList {
				price := goods.Price
				if member {
					price = goods.DiscountPrice
				}
				category.Goods = append(category.Goods, &app.Goods{
					Id:      goods.ID,
					Name:    goods.Name,
					Desc:    goods.Desc,
					Picture: goods.Picture,
					SaleNum: goods.InitSaleNum + goods.RealSaleNum,
					Price:   price,
				})
			}
		}
		list = append(list, category)
	}

	return &app.ListGoodsCategoryResponse{
		List: list,
	}, nil
}

// 将商品分类模型转换为接口响应
func (c *GoodsCategoryCase) convertToProto(item *models.GoodsCategory) *app.GoodsCategory {
	res := &app.GoodsCategory{
		Id:       item.ID,
		ParentId: item.ParentID,
		Name:     item.Name,
		Picture:  item.Picture,
		Goods:    nil,
	}
	return res
}
