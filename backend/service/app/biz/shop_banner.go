package biz

import (
	"context"
	"fmt"
	"strconv"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// ShopBannerCase 商城轮播图业务处理对象
type ShopBannerCase struct {
	*biz.BaseCase
	*data.ShopBannerRepository
	goodsCategoryCase *GoodsCategoryCase
	mapper            *mapper.CopierMapper[appv1.ShopBanner, models.ShopBanner]
}

// NewShopBannerCase 创建商城轮播图业务处理对象
func NewShopBannerCase(baseCase *biz.BaseCase, shopBannerRepo *data.ShopBannerRepository, goodsCategoryCase *GoodsCategoryCase) *ShopBannerCase {
	return &ShopBannerCase{
		BaseCase:             baseCase,
		ShopBannerRepository: shopBannerRepo,
		goodsCategoryCase:    goodsCategoryCase,
		mapper:               mapper.NewCopierMapper[appv1.ShopBanner, models.ShopBanner](),
	}
}

// ListShopBanners 查询商城轮播图列表
func (c *ShopBannerCase) ListShopBanners(ctx context.Context, req *appv1.ListShopBannersRequest) (*appv1.ListShopBannersResponse, error) {
	all, err := c.listBySite(ctx, int32(req.GetSite()))
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.ShopBanner, 0, len(all))
	for _, item := range all {
		list = append(list, c.convertToProto(ctx, item))
	}

	return &appv1.ListShopBannersResponse{
		ShopBanners: list,
	}, nil
}

// 查询指定站点下启用中的商城轮播图
func (c *ShopBannerCase) listBySite(ctx context.Context, site int32) ([]*models.ShopBanner, error) {
	query := c.Query(ctx).ShopBanner
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Site.Eq(site)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	return c.List(ctx, opts...)
}

// 将商城轮播图模型转换为接口响应
func (c *ShopBannerCase) convertToProto(ctx context.Context, item *models.ShopBanner) *appv1.ShopBanner {
	res := c.mapper.ToDTO(item)
	var href string
	// 按轮播图类型把后台配置值转换成前端可直接消费的跳转参数。
	switch commonv1.ShopBannerType(item.Type) {
	case commonv1.ShopBannerType(_const.SHOP_BANNER_TYPE_BANNER_GOODS_DETAIL):
		href = fmt.Sprintf("id=%s", item.Href)
	case commonv1.ShopBannerType(_const.SHOP_BANNER_TYPE_CATEGORY_DETAIL):
		// 商城轮播图分类需要把分类 ID 转成前端可直接使用的跳转参数
		id, err := strconv.ParseInt(item.Href, 10, 64)
		// 分类编号解析成功时，再继续查询分类名称拼装跳转参数。
		if err == nil {
			var find *models.GoodsCategory
			query := c.goodsCategoryCase.Query(ctx).GoodsCategory
			opts := make([]repository.QueryOption, 0, 1)
			opts = append(opts, repository.Where(query.ID.Eq(id)))
			find, err = c.goodsCategoryCase.Find(ctx, opts...)
			// 分类存在且查询成功时，使用分类参数覆盖原始链接。
			if err == nil && find != nil {
				href = fmt.Sprintf("categoryId=%d&categoryName=%s", find.ID, find.Name)
			}
		}
	default:
		href = item.Href
	}
	res.Href = href
	return res
}
