package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsCategoryCase 商品分类业务处理对象
type GoodsCategoryCase struct {
	*biz.BaseCase
	*data.GoodsCategoryRepo
	goodsInfoRepo *data.GoodsInfoRepo
	mapper        *mapper.CopierMapper[app.GoodsCategory, models.GoodsCategory]
	goodsMapper   *mapper.CopierMapper[app.GoodsInfo, models.GoodsInfo]
}

// NewGoodsCategoryCase 创建商品分类业务处理对象
func NewGoodsCategoryCase(baseCase *biz.BaseCase, goodsCategoryRepo *data.GoodsCategoryRepo, goodsInfoRepo *data.GoodsInfoRepo) *GoodsCategoryCase {
	return &GoodsCategoryCase{
		BaseCase:          baseCase,
		GoodsCategoryRepo: goodsCategoryRepo,
		goodsInfoRepo:     goodsInfoRepo,
		mapper:            mapper.NewCopierMapper[app.GoodsCategory, models.GoodsCategory](),
		goodsMapper:       mapper.NewCopierMapper[app.GoodsInfo, models.GoodsInfo](),
	}
}

// ListGoodsCategory 查询分类列表
func (c *GoodsCategoryCase) ListGoodsCategory(ctx context.Context, req *app.ListGoodsCategoryRequest) (*app.ListGoodsCategoryResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.ParentID.Eq(req.GetParentId())))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	member := utils.IsMember(ctx)
	list := make([]*app.GoodsCategory, 0, len(all))
	for _, item := range all {
		category := c.mapper.ToDTO(item)
		// 二级分类需要同时返回分类下的推荐商品
		if category.ParentId > 0 {
			goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
			var goodsInfoList []*models.GoodsInfo
			goodsOpts := make([]repo.QueryOption, 0, 3)
			goodsOpts = append(goodsOpts, repo.Order(goodsQuery.CreatedAt.Desc()))
			goodsOpts = append(goodsOpts, repo.Where(goodsQuery.CategoryID.Eq(category.Id)))
			goodsOpts = append(goodsOpts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
			goodsInfoList, _, err = c.goodsInfoRepo.Page(ctx, 1, 9, goodsOpts...)
			if err != nil {
				return nil, err
			}
			category.Goods = make([]*app.GoodsInfo, 0, len(goodsInfoList))
			for _, goodsInfo := range goodsInfoList {
				price := goodsInfo.Price
				if member {
					price = goodsInfo.DiscountPrice
				}
				goods := c.goodsMapper.ToDTO(goodsInfo)
				goods.SaleNum = goodsInfo.InitSaleNum + goodsInfo.RealSaleNum
				goods.Price = price
				category.Goods = append(category.Goods, goods)
			}
		}
		list = append(list, category)
	}

	return &app.ListGoodsCategoryResponse{
		List: list,
	}, nil
}
