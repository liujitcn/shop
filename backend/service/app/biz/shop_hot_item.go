package biz

import (
	"context"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/util"

	"github.com/liujitcn/gorm-kit/repo"
)

// ShopHotItemCase 热门推荐项业务处理对象
type ShopHotItemCase struct {
	*biz.BaseCase
	*data.ShopHotItemRepo
	shopHotRepo      *data.ShopHotRepo
	shopHotGoodsRepo *data.ShopHotGoodsRepo
	goodsRepo        *data.GoodsRepo
}

// NewShopHotItemCase 创建热门推荐项业务处理对象
func NewShopHotItemCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepo, shopHotItemRepo *data.ShopHotItemRepo, shopHotGoodsRepo *data.ShopHotGoodsRepo, goodsInfoRepo *data.GoodsRepo) *ShopHotItemCase {
	return &ShopHotItemCase{
		BaseCase:         baseCase,
		ShopHotItemRepo:  shopHotItemRepo,
		shopHotRepo:      shopHotRepo,
		shopHotGoodsRepo: shopHotGoodsRepo,
		goodsRepo:        goodsInfoRepo,
	}
}

// ListShopHotItem 查询热门推荐选项
func (c *ShopHotItemCase) ListShopHotItem(ctx context.Context, id int64) (*app.ListShopHotItemResponse, error) {
	shopHotQuery := c.shopHotRepo.Query(ctx).ShopHot
	shopHot, err := c.shopHotRepo.Find(ctx,
		repo.Where(shopHotQuery.ID.Eq(id)),
		repo.Where(shopHotQuery.Status.Eq(int32(common.Status_ENABLE))),
	)
	if err != nil {
		return nil, err
	}

	var all []*models.ShopHotItem

	shopHotItemQuery := c.Query(ctx).ShopHotItem
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(shopHotItemQuery.Sort.Asc()))
	opts = append(opts, repo.Order(shopHotItemQuery.UpdatedAt.Desc()))
	opts = append(opts, repo.Where(shopHotItemQuery.HotID.Eq(shopHot.ID)))
	all, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*app.ShopHotItem, 0, len(all))
	for _, item := range all {
		list = append(list, c.convertToProto(item))
	}

	return &app.ListShopHotItemResponse{
		Id:     shopHot.ID,
		Title:  shopHot.Title,
		Banner: shopHot.Banner,
		List:   list,
	}, nil
}

// PageShopHotGoods 查询热门推荐商品
func (c *ShopHotItemCase) PageShopHotGoods(ctx context.Context, req *app.PageShopHotGoodsRequest) (*app.PageShopHotGoodsResponse, error) {
	// 是否会员
	member := util.IsMember(ctx)
	// 先查询专区和商品关系，再批量回表查询商品详情
	hotGoodsQuery := c.shopHotGoodsRepo.Query(ctx).ShopHotGoods
	hotGoodsOpts := make([]repo.QueryOption, 0, 2)
	hotGoodsOpts = append(hotGoodsOpts, repo.Order(hotGoodsQuery.Sort.Asc()))
	hotGoodsOpts = append(hotGoodsOpts, repo.Where(hotGoodsQuery.HotItemID.Eq(req.HotItemId)))
	hotGoodsList, count, err := c.shopHotGoodsRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), hotGoodsOpts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.Goods, 0)
	if count > 0 {
		goodsIds := make([]int64, 0, len(hotGoodsList))
		for _, item := range hotGoodsList {
			goodsIds = append(goodsIds, item.GoodsID)
		}
		var all []*models.Goods
		goodsQuery := c.goodsRepo.Query(ctx).Goods
		goodsOpts := make([]repo.QueryOption, 0, 3)
		goodsOpts = append(goodsOpts, repo.Order(goodsQuery.UpdatedAt.Desc()))
		goodsOpts = append(goodsOpts, repo.Where(goodsQuery.ID.In(goodsIds...)))
		goodsOpts = append(goodsOpts, repo.Where(goodsQuery.Status.Eq(int32(common.Status_ENABLE))))
		all, err = c.goodsRepo.List(ctx, goodsOpts...)
		if err != nil {
			return nil, err
		}
		for _, item := range all {
			price := item.Price
			if member {
				price = item.DiscountPrice
			}
			list = append(list, &app.Goods{
				Id:      item.ID,
				Name:    item.Name,
				Desc:    item.Desc,
				Picture: item.Picture,
				SaleNum: item.InitSaleNum + item.RealSaleNum,
				Price:   price,
			})
		}
	}
	return &app.PageShopHotGoodsResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// 将热门推荐项模型转换为接口响应
func (c *ShopHotItemCase) convertToProto(item *models.ShopHotItem) *app.ShopHotItem {
	res := &app.ShopHotItem{
		Id:    item.ID,
		Title: item.Title,
	}
	return res
}
