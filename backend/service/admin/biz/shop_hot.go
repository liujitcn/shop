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

// ShopHotCase 热门专区业务实例
type ShopHotCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.ShopHotRepo
	shopHotItemCase *ShopHotItemCase
	formMapper      *mapper.CopierMapper[admin.ShopHotForm, models.ShopHot]
	mapper          *mapper.CopierMapper[admin.ShopHot, models.ShopHot]
}

// NewShopHotCase 创建热门专区业务实例
func NewShopHotCase(baseCase *biz.BaseCase, tx data.Transaction, shopHotRepo *data.ShopHotRepo, shopHotItemCase *ShopHotItemCase) *ShopHotCase {
	return &ShopHotCase{
		BaseCase:        baseCase,
		tx:              tx,
		ShopHotRepo:     shopHotRepo,
		shopHotItemCase: shopHotItemCase,
		formMapper:      mapper.NewCopierMapper[admin.ShopHotForm, models.ShopHot](),
		mapper:          mapper.NewCopierMapper[admin.ShopHot, models.ShopHot](),
	}
}

// PageShopHot 分页查询热门专区
func (c *ShopHotCase) PageShopHot(ctx context.Context, req *admin.PageShopHotRequest) (*admin.PageShopHotResponse, error) {
	baseQuery := c.Query(ctx).ShopHot
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(baseQuery.Sort.Asc()))
	opts = append(opts, repo.Order(baseQuery.CreatedAt.Desc()))
	if req.GetTitle() != "" {
		opts = append(opts, repo.Where(baseQuery.Title.Like("%"+req.GetTitle()+"%")))
	}
	if req.GetDesc() != "" {
		opts = append(opts, repo.Where(baseQuery.Desc.Like("%"+req.GetDesc()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(baseQuery.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.ShopHot, 0, len(list))
	for _, item := range list {
		shopHot := c.mapper.ToDTO(item)
		resList = append(resList, shopHot)
	}
	return &admin.PageShopHotResponse{List: resList, Total: int32(total)}, nil
}

// GetShopHot 获取热门专区
func (c *ShopHotCase) GetShopHot(ctx context.Context, id int64) (*admin.ShopHotForm, error) {
	var shopHot *models.ShopHot
	var err error
	shopHot, err = c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(shopHot)
	res.Picture = _string.ConvertJsonStringToStringArray(shopHot.Picture)
	return res, nil
}

// CreateShopHot 创建热门专区
func (c *ShopHotCase) CreateShopHot(ctx context.Context, req *admin.ShopHotForm) error {
	shopHot := c.formMapper.ToEntity(req)
	shopHot.Picture = _string.ConvertStringArrayToString(req.GetPicture())
	return c.Create(ctx, shopHot)
}

// UpdateShopHot 更新热门专区
func (c *ShopHotCase) UpdateShopHot(ctx context.Context, req *admin.ShopHotForm) error {
	shopHot := c.formMapper.ToEntity(req)
	shopHot.Picture = _string.ConvertStringArrayToString(req.GetPicture())
	return c.UpdateById(ctx, shopHot)
}

// DeleteShopHot 删除热门专区
func (c *ShopHotCase) DeleteShopHot(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.DeleteByIds(ctx, ids)
		if err != nil {
			return err
		}

		// 删除热门专区后需要同步删除下属项目，避免残留孤儿数据
		hotItemQuery := c.shopHotItemCase.Query(ctx).ShopHotItem
		var hotItemList []*models.ShopHotItem
		hotItemOpts := make([]repo.QueryOption, 0, 1)
		hotItemOpts = append(hotItemOpts, repo.Where(hotItemQuery.HotID.In(ids...)))
		hotItemList, err = c.shopHotItemCase.List(ctx, hotItemOpts...)
		if err != nil {
			return err
		}

		itemIds := make([]int64, 0, len(hotItemList))
		for _, item := range hotItemList {
			itemIds = append(itemIds, item.ID)
		}
		if len(itemIds) > 0 {
			err = c.shopHotItemCase.DeleteByIds(ctx, itemIds)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// SetShopHotStatus 设置热门专区状态
func (c *ShopHotCase) SetShopHotStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.ShopHot{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// PageShopHotItem 分页查询热门专区项目
func (c *ShopHotCase) PageShopHotItem(ctx context.Context, req *admin.PageShopHotItemRequest) (*admin.PageShopHotItemResponse, error) {
	return c.shopHotItemCase.PageShopHotItem(ctx, req)
}

// GetShopHotItem 获取热门专区项目
func (c *ShopHotCase) GetShopHotItem(ctx context.Context, id int64) (*admin.ShopHotItemForm, error) {
	return c.shopHotItemCase.GetShopHotItem(ctx, id)
}

// CreateShopHotItem 创建热门专区项目
func (c *ShopHotCase) CreateShopHotItem(ctx context.Context, req *admin.ShopHotItemForm) error {
	return c.shopHotItemCase.CreateShopHotItem(ctx, req)
}

// UpdateShopHotItem 更新热门专区项目
func (c *ShopHotCase) UpdateShopHotItem(ctx context.Context, req *admin.ShopHotItemForm) error {
	return c.shopHotItemCase.UpdateShopHotItem(ctx, req)
}

// DeleteShopHotItem 删除热门专区项目
func (c *ShopHotCase) DeleteShopHotItem(ctx context.Context, id string) error {
	return c.shopHotItemCase.DeleteShopHotItem(ctx, id)
}

// SetShopHotItemStatus 设置热门专区项目状态
func (c *ShopHotCase) SetShopHotItemStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.shopHotItemCase.SetShopHotItemStatus(ctx, req)
}
