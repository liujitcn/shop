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

// ShopServiceCase 服务保障业务实例
type ShopServiceCase struct {
	*biz.BaseCase
	*data.ShopServiceRepo
	formMapper *mapper.CopierMapper[admin.ShopServiceForm, models.ShopService]
	mapper     *mapper.CopierMapper[admin.ShopService, models.ShopService]
}

// NewShopServiceCase 创建服务保障业务实例
func NewShopServiceCase(baseCase *biz.BaseCase, shopServiceRepo *data.ShopServiceRepo) *ShopServiceCase {
	return &ShopServiceCase{
		BaseCase:        baseCase,
		ShopServiceRepo: shopServiceRepo,
		formMapper:      mapper.NewCopierMapper[admin.ShopServiceForm, models.ShopService](),
		mapper:          mapper.NewCopierMapper[admin.ShopService, models.ShopService](),
	}
}

// PageShopService 分页查询服务保障
func (c *ShopServiceCase) PageShopService(ctx context.Context, req *admin.PageShopServiceRequest) (*admin.PageShopServiceResponse, error) {
	query := c.Query(ctx).ShopService
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 传入服务名称时，按名称模糊匹配服务保障项。
	if req.GetLabel() != "" {
		opts = append(opts, repo.Where(query.Label.Like("%"+req.GetLabel()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.ShopService, 0, len(list))
	for _, item := range list {
		shopService := c.mapper.ToDTO(item)
		resList = append(resList, shopService)
	}
	return &admin.PageShopServiceResponse{List: resList, Total: int32(total)}, nil
}

// GetShopService 获取服务保障
func (c *ShopServiceCase) GetShopService(ctx context.Context, id int64) (*admin.ShopServiceForm, error) {
	shopService, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(shopService)
	return res, nil
}

// CreateShopService 创建服务保障
func (c *ShopServiceCase) CreateShopService(ctx context.Context, req *admin.ShopServiceForm) error {
	shopService := c.formMapper.ToEntity(req)
	return c.Create(ctx, shopService)
}

// UpdateShopService 更新服务保障
func (c *ShopServiceCase) UpdateShopService(ctx context.Context, req *admin.ShopServiceForm) error {
	shopService := c.formMapper.ToEntity(req)
	return c.UpdateById(ctx, shopService)
}

// DeleteShopService 删除服务保障
func (c *ShopServiceCase) DeleteShopService(ctx context.Context, id string) error {
	return c.DeleteByIds(ctx, _string.ConvertStringToInt64Array(id))
}

// SetShopServiceStatus 设置服务保障状态
func (c *ShopServiceCase) SetShopServiceStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.ShopService{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}
