package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// ShopBannerCase 商城轮播图业务实例
type ShopBannerCase struct {
	*biz.BaseCase
	*data.ShopBannerRepo
	formMapper *mapper.CopierMapper[admin.ShopBannerForm, models.ShopBanner]
	mapper     *mapper.CopierMapper[admin.ShopBanner, models.ShopBanner]
}

// NewShopBannerCase 创建商城轮播图业务实例
func NewShopBannerCase(baseCase *biz.BaseCase, shopBannerRepo *data.ShopBannerRepo) *ShopBannerCase {
	return &ShopBannerCase{
		BaseCase:       baseCase,
		ShopBannerRepo: shopBannerRepo,
		formMapper:     mapper.NewCopierMapper[admin.ShopBannerForm, models.ShopBanner](),
		mapper:         mapper.NewCopierMapper[admin.ShopBanner, models.ShopBanner](),
	}
}

// PageShopBanner 分页查询商城轮播图
func (c *ShopBannerCase) PageShopBanner(ctx context.Context, req *admin.PageShopBannerRequest) (*admin.PageShopBannerResponse, error) {
	query := c.Query(ctx).ShopBanner
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	if req.Site != nil {
		opts = append(opts, repo.Where(query.Site.Eq(int32(req.GetSite()))))
	}
	if req.Type != nil {
		opts = append(opts, repo.Where(query.Type.Eq(int32(req.GetType()))))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.ShopBanner, 0, len(list))
	for _, item := range list {
		shopBanner := c.mapper.ToDTO(item)
		resList = append(resList, shopBanner)
	}
	return &admin.PageShopBannerResponse{List: resList, Total: int32(total)}, nil
}

// GetShopBanner 获取商城轮播图
func (c *ShopBannerCase) GetShopBanner(ctx context.Context, id int64) (*admin.ShopBannerForm, error) {
	shopBanner, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(shopBanner)
	return res, nil
}

// CreateShopBanner 创建商城轮播图
func (c *ShopBannerCase) CreateShopBanner(ctx context.Context, req *admin.ShopBannerForm) error {
	shopBanner := c.formMapper.ToEntity(req)
	return c.Create(ctx, shopBanner)
}

// UpdateShopBanner 更新商城轮播图
func (c *ShopBannerCase) UpdateShopBanner(ctx context.Context, req *admin.ShopBannerForm) error {
	shopBanner := c.formMapper.ToEntity(req)
	return c.UpdateById(ctx, shopBanner)
}

// DeleteShopBanner 删除商城轮播图
func (c *ShopBannerCase) DeleteShopBanner(ctx context.Context, id string) error {
	return c.DeleteByIds(ctx, _string.ConvertStringToInt64Array(id))
}

// SetShopBannerStatus 设置商城轮播图状态
func (c *ShopBannerCase) SetShopBannerStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.ShopBanner{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}
