package biz

import (
	"context"
	"fmt"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"

	"strconv"
)

// ShopBannerCase 轮播图业务处理对象
type ShopBannerCase struct {
	*biz.BaseCase
	*data.ShopBannerRepo
	goodsCategoryCase *GoodsCategoryCase
}

// NewShopBannerCase 创建轮播图业务处理对象
func NewShopBannerCase(baseCase *biz.BaseCase, shopBannerRepo *data.ShopBannerRepo, goodsCategoryCase *GoodsCategoryCase) *ShopBannerCase {
	return &ShopBannerCase{
		BaseCase:          baseCase,
		ShopBannerRepo:    shopBannerRepo,
		goodsCategoryCase: goodsCategoryCase,
	}
}

// ListShopBanner 查询轮播图列表
func (c *ShopBannerCase) ListShopBanner(ctx context.Context, req *app.ListShopBannerRequest) (*app.ListShopBannerResponse, error) {
	all, err := c.listBySite(ctx, req.GetSite())
	if err != nil {
		return nil, err
	}

	list := make([]*app.ShopBanner, 0, len(all))
	for _, item := range all {
		list = append(list, c.convertToProto(ctx, item))
	}

	return &app.ListShopBannerResponse{
		List: list,
	}, nil
}

// 查询指定站点下启用中的轮播图
func (c *ShopBannerCase) listBySite(ctx context.Context, site int32) ([]*models.ShopBanner, error) {
	query := c.Query(ctx).ShopBanner
	return c.List(ctx,
		repo.Where(query.Site.Eq(site)),
		repo.Where(query.Status.Eq(int32(common.Status_ENABLE))),
	)
}

// 将轮播图模型转换为接口响应
func (c *ShopBannerCase) convertToProto(ctx context.Context, item *models.ShopBanner) *app.ShopBanner {
	var href string
	switch common.ShopBannerType(item.Type) {
	case common.ShopBannerType_GOODS_DETAIL:
		href = fmt.Sprintf("id=%s", item.Href)
	case common.ShopBannerType_CATEGORY_DETAIL:
		// 分类轮播图需要把分类 ID 转成前端可直接使用的跳转参数
		id, err := strconv.ParseInt(item.Href, 10, 64)
		if err == nil {
			var find *models.GoodsCategory
			query := c.goodsCategoryCase.Query(ctx).GoodsCategory
			find, err = c.goodsCategoryCase.Find(ctx,
				repo.Where(query.ID.Eq(id)),
			)
			if err == nil && find != nil {
				href = fmt.Sprintf("categoryId=%d&categoryName=%s", find.ID, find.Name)
			}
		}
	default:
		href = item.Href
	}
	res := &app.ShopBanner{
		Site:    common.ShopBannerSite(item.Site),
		Picture: item.Picture,
		Type:    common.ShopBannerType(item.Type),
		Href:    href,
	}
	return res
}
