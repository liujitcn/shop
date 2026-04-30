package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// ShopBannerCase 商城轮播图业务实例
type ShopBannerCase struct {
	*biz.BaseCase
	*data.ShopBannerRepository
	formMapper *mapper.CopierMapper[adminv1.ShopBannerForm, models.ShopBanner]
	mapper     *mapper.CopierMapper[adminv1.ShopBanner, models.ShopBanner]
}

// NewShopBannerCase 创建商城轮播图业务实例
func NewShopBannerCase(baseCase *biz.BaseCase, shopBannerRepo *data.ShopBannerRepository) *ShopBannerCase {
	return &ShopBannerCase{
		BaseCase:             baseCase,
		ShopBannerRepository: shopBannerRepo,
		formMapper:           mapper.NewCopierMapper[adminv1.ShopBannerForm, models.ShopBanner](),
		mapper:               mapper.NewCopierMapper[adminv1.ShopBanner, models.ShopBanner](),
	}
}

// PageShopBanners 查询商城轮播图列表
func (c *ShopBannerCase) PageShopBanners(ctx context.Context, req *adminv1.PageShopBannersRequest) (*adminv1.PageShopBannersResponse, error) {
	query := c.Query(ctx).ShopBanner
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Site != nil {
		opts = append(opts, repository.Where(query.Site.Eq(int32(req.GetSite()))))
	}
	if req.Type != nil {
		opts = append(opts, repository.Where(query.Type.Eq(int32(req.GetType()))))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.ShopBanner, 0, len(list))
	for _, item := range list {
		shopBanner := c.mapper.ToDTO(item)
		resList = append(resList, shopBanner)
	}
	return &adminv1.PageShopBannersResponse{ShopBanners: resList, Total: int32(total)}, nil
}

// GetShopBanner 获取商城轮播图
func (c *ShopBannerCase) GetShopBanner(ctx context.Context, id int64) (*adminv1.ShopBannerForm, error) {
	shopBanner, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(shopBanner)
	return res, nil
}

// CreateShopBanner 创建商城轮播图
func (c *ShopBannerCase) CreateShopBanner(ctx context.Context, req *adminv1.ShopBannerForm) error {
	shopBanner := c.formMapper.ToEntity(req)
	return c.Create(ctx, shopBanner)
}

// UpdateShopBanner 更新商城轮播图
func (c *ShopBannerCase) UpdateShopBanner(ctx context.Context, req *adminv1.ShopBannerForm) error {
	shopBanner := c.formMapper.ToEntity(req)
	return c.UpdateByID(ctx, shopBanner)
}

// DeleteShopBanner 删除商城轮播图
func (c *ShopBannerCase) DeleteShopBanner(ctx context.Context, id string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(id))
}

// SetShopBannerStatus 设置商城轮播图状态
func (c *ShopBannerCase) SetShopBannerStatus(ctx context.Context, req *adminv1.SetShopBannerStatusRequest) error {
	return c.UpdateByID(ctx, &models.ShopBanner{
		ID:     req.GetId(),
		Status: int32(req.GetStatus()),
	})
}
