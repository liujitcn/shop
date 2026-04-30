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

// ShopServiceCase 服务保障业务实例
type ShopServiceCase struct {
	*biz.BaseCase
	*data.ShopServiceRepository
	formMapper *mapper.CopierMapper[adminv1.ShopServiceForm, models.ShopService]
	mapper     *mapper.CopierMapper[adminv1.ShopService, models.ShopService]
}

// NewShopServiceCase 创建服务保障业务实例
func NewShopServiceCase(baseCase *biz.BaseCase, shopServiceRepo *data.ShopServiceRepository) *ShopServiceCase {
	return &ShopServiceCase{
		BaseCase:              baseCase,
		ShopServiceRepository: shopServiceRepo,
		formMapper:            mapper.NewCopierMapper[adminv1.ShopServiceForm, models.ShopService](),
		mapper:                mapper.NewCopierMapper[adminv1.ShopService, models.ShopService](),
	}
}

// PageShopServices 查询服务保障列表
func (c *ShopServiceCase) PageShopServices(ctx context.Context, req *adminv1.PageShopServicesRequest) (*adminv1.PageShopServicesResponse, error) {
	query := c.Query(ctx).ShopService
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入服务名称时，按名称模糊匹配服务保障项。
	if req.GetLabel() != "" {
		opts = append(opts, repository.Where(query.Label.Like("%"+req.GetLabel()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.ShopService, 0, len(list))
	for _, item := range list {
		shopService := c.mapper.ToDTO(item)
		resList = append(resList, shopService)
	}
	return &adminv1.PageShopServicesResponse{ShopServices: resList, Total: int32(total)}, nil
}

// GetShopService 获取服务保障
func (c *ShopServiceCase) GetShopService(ctx context.Context, id int64) (*adminv1.ShopServiceForm, error) {
	shopService, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(shopService)
	return res, nil
}

// CreateShopService 创建服务保障
func (c *ShopServiceCase) CreateShopService(ctx context.Context, req *adminv1.ShopServiceForm) error {
	shopService := c.formMapper.ToEntity(req)
	return c.Create(ctx, shopService)
}

// UpdateShopService 更新服务保障
func (c *ShopServiceCase) UpdateShopService(ctx context.Context, req *adminv1.ShopServiceForm) error {
	shopService := c.formMapper.ToEntity(req)
	return c.UpdateByID(ctx, shopService)
}

// DeleteShopService 删除服务保障
func (c *ShopServiceCase) DeleteShopService(ctx context.Context, id string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(id))
}

// SetShopServiceStatus 设置服务保障状态
func (c *ShopServiceCase) SetShopServiceStatus(ctx context.Context, req *adminv1.SetShopServiceStatusRequest) error {
	return c.UpdateByID(ctx, &models.ShopService{
		ID:     req.GetId(),
		Status: int32(req.GetStatus()),
	})
}
