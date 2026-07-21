package biz

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"

	_const "shop/service/shop/consts"

	shopappv1 "shop/api/gen/go/shop/app/v1"
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
	mapper            *mapper.CopierMapper[shopappv1.ShopBanner, models.ShopBanner]
}

// NewShopBannerCase 创建商城轮播图业务处理对象
func NewShopBannerCase(baseCase *biz.BaseCase, shopBannerRepo *data.ShopBannerRepository, goodsCategoryCase *GoodsCategoryCase) *ShopBannerCase {
	return &ShopBannerCase{
		BaseCase:             baseCase,
		ShopBannerRepository: shopBannerRepo,
		goodsCategoryCase:    goodsCategoryCase,
		mapper:               mapper.NewCopierMapper[shopappv1.ShopBanner, models.ShopBanner](),
	}
}

// ListShopBanner 查询商城轮播图列表
func (c *ShopBannerCase) ListShopBanner(ctx context.Context, req *shopappv1.ListShopBannerRequest) (*shopappv1.ListShopBannerResponse, error) {
	all, err := c.listBySite(ctx, int32(req.GetSite()))
	if err != nil {
		return nil, err
	}

	list := make([]*shopappv1.ShopBanner, 0, len(all))
	for _, item := range all {
		list = append(list, c.convertToProto(ctx, item))
	}

	return &shopappv1.ListShopBannerResponse{
		ShopBanners: list,
	}, nil
}

// 将商城轮播图模型转换为接口响应
func (c *ShopBannerCase) convertToProto(ctx context.Context, item *models.ShopBanner) *shopappv1.ShopBanner {
	res := c.mapper.ToDTO(item)
	var href string
	// 按轮播图类型把后台配置值转换成前端可直接消费的跳转参数。
	switch shopcommonv1.ShopBannerType(item.Type) {
	case shopcommonv1.ShopBannerType(_const.SHOP_BANNER_TYPE_BANNER_GOODS_DETAIL):
		id := parseBannerTargetID(item.Href, []string{"id", "goods_id"}, "goods")
		// 商品目标有效时，统一输出商品详情页使用的参数名。
		if id > 0 {
			href = fmt.Sprintf("id=%d", id)
		}
	case shopcommonv1.ShopBannerType(_const.SHOP_BANNER_TYPE_CATEGORY_DETAIL):
		id := parseBannerTargetID(item.Href, []string{"category_id", "categoryId"}, "category")
		// 分类编号解析成功时，再确认目标分类存在。
		if id > 0 {
			var find *models.GoodsCategory
			query := c.goodsCategoryCase.Query(ctx).GoodsCategory
			opts := make([]repository.QueryOption, 0, 1)
			opts = append(opts, repository.Where(query.ID.Eq(id)))
			var err error
			find, err = c.goodsCategoryCase.Find(ctx, opts...)
			// 分类存在且查询成功时，使用分类参数覆盖原始链接。
			if err == nil && find != nil {
				href = fmt.Sprintf("category_id=%d", find.ID)
			}
		}
	default:
		href = item.Href
	}
	res.Href = href
	return res
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

// parseBannerTargetID 兼容纯编号、结构化参数和历史路径格式的轮播目标。
func parseBannerTargetID(href string, queryKeys []string, legacySegment string) int64 {
	normalizedHref := strings.TrimSpace(href)
	id, err := strconv.ParseInt(normalizedHref, 10, 64)
	// 初始化数据只保存目标编号时可直接返回。
	if err == nil && id > 0 {
		return id
	}

	var parsedURL *url.URL
	parsedURL, err = url.Parse(normalizedHref)
	if err != nil {
		return 0
	}

	values := parsedURL.Query()
	// 兼容未带问号、只保存 query 内容的结构化配置。
	if parsedURL.RawQuery == "" && strings.Contains(normalizedHref, "=") {
		values, err = url.ParseQuery(strings.TrimLeft(normalizedHref, "?#&"))
		if err != nil {
			return 0
		}
	}
	for _, key := range queryKeys {
		id, err = strconv.ParseInt(values.Get(key), 10, 64)
		// 找到首个有效参数后即可确定跳转目标。
		if err == nil && id > 0 {
			return id
		}
	}

	segments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	// 兼容历史的 /goods/{id} 与 /category/{id} 路径配置。
	for index := 0; index+1 < len(segments); index++ {
		if segments[index] != legacySegment {
			continue
		}
		id, err = strconv.ParseInt(segments[index+1], 10, 64)
		if err == nil && id > 0 {
			return id
		}
	}
	return 0
}
