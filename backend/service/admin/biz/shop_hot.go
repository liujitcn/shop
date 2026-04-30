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

// ShopHotCase 热门专区业务实例
type ShopHotCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.ShopHotRepository
	shopHotItemCase *ShopHotItemCase
	formMapper      *mapper.CopierMapper[adminv1.ShopHotForm, models.ShopHot]
	mapper          *mapper.CopierMapper[adminv1.ShopHot, models.ShopHot]
}

// NewShopHotCase 创建热门专区业务实例
func NewShopHotCase(baseCase *biz.BaseCase, tx data.Transaction, shopHotRepo *data.ShopHotRepository, shopHotItemCase *ShopHotItemCase) *ShopHotCase {
	return &ShopHotCase{
		BaseCase:          baseCase,
		tx:                tx,
		ShopHotRepository: shopHotRepo,
		shopHotItemCase:   shopHotItemCase,
		formMapper:        mapper.NewCopierMapper[adminv1.ShopHotForm, models.ShopHot](),
		mapper:            mapper.NewCopierMapper[adminv1.ShopHot, models.ShopHot](),
	}
}

// PageShopHots 查询热门专区列表
func (c *ShopHotCase) PageShopHots(ctx context.Context, req *adminv1.PageShopHotsRequest) (*adminv1.PageShopHotsResponse, error) {
	query := c.Query(ctx).ShopHot
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入标题关键字时，按标题模糊匹配热门专区。
	if req.GetTitle() != "" {
		opts = append(opts, repository.Where(query.Title.Like("%"+req.GetTitle()+"%")))
	}
	// 传入描述关键字时，按描述模糊匹配热门专区。
	if req.GetDesc() != "" {
		opts = append(opts, repository.Where(query.Desc.Like("%"+req.GetDesc()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.ShopHot, 0, len(list))
	for _, item := range list {
		shopHot := c.mapper.ToDTO(item)
		resList = append(resList, shopHot)
	}
	return &adminv1.PageShopHotsResponse{ShopHots: resList, Total: int32(total)}, nil
}

// GetShopHot 获取热门专区
func (c *ShopHotCase) GetShopHot(ctx context.Context, id int64) (*adminv1.ShopHotForm, error) {
	shopHot, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(shopHot)
	res.Picture = _string.ConvertJsonStringToStringArray(shopHot.Picture)
	return res, nil
}

// CreateShopHot 创建热门专区
func (c *ShopHotCase) CreateShopHot(ctx context.Context, req *adminv1.ShopHotForm) error {
	shopHot := c.formMapper.ToEntity(req)
	shopHot.Picture = _string.ConvertStringArrayToString(req.GetPicture())
	return c.Create(ctx, shopHot)
}

// UpdateShopHot 更新热门专区
func (c *ShopHotCase) UpdateShopHot(ctx context.Context, req *adminv1.ShopHotForm) error {
	shopHot := c.formMapper.ToEntity(req)
	shopHot.Picture = _string.ConvertStringArrayToString(req.GetPicture())
	return c.UpdateByID(ctx, shopHot)
}

// DeleteShopHot 删除热门专区
func (c *ShopHotCase) DeleteShopHot(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}

		// 删除热门专区后需要同步删除下属项目，避免残留孤儿数据
		query := c.shopHotItemCase.Query(ctx).ShopHotItem
		var hotItemList []*models.ShopHotItem
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.HotID.In(ids...)))
		hotItemList, err = c.shopHotItemCase.List(ctx, opts...)
		if err != nil {
			return err
		}

		itemIDs := make([]int64, 0, len(hotItemList))
		for _, item := range hotItemList {
			itemIDs = append(itemIDs, item.ID)
		}
		// 命中下属项目时，同步清理项目数据避免孤儿记录。
		if len(itemIDs) > 0 {
			err = c.shopHotItemCase.DeleteByIDs(ctx, itemIDs)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// SetShopHotStatus 设置热门专区状态
func (c *ShopHotCase) SetShopHotStatus(ctx context.Context, req *adminv1.SetShopHotStatusRequest) error {
	return c.UpdateByID(ctx, &models.ShopHot{
		ID:     req.GetId(),
		Status: int32(req.GetStatus()),
	})
}

// PageShopHotItems 查询热门专区项目列表
func (c *ShopHotCase) PageShopHotItems(ctx context.Context, req *adminv1.PageShopHotItemsRequest) (*adminv1.PageShopHotItemsResponse, error) {
	return c.shopHotItemCase.PageShopHotItems(ctx, req)
}

// GetShopHotItem 获取热门专区项目
func (c *ShopHotCase) GetShopHotItem(ctx context.Context, id int64) (*adminv1.ShopHotItemForm, error) {
	return c.shopHotItemCase.GetShopHotItem(ctx, id)
}

// CreateShopHotItem 创建热门专区项目
func (c *ShopHotCase) CreateShopHotItem(ctx context.Context, req *adminv1.ShopHotItemForm) error {
	return c.shopHotItemCase.CreateShopHotItem(ctx, req)
}

// UpdateShopHotItem 更新热门专区项目
func (c *ShopHotCase) UpdateShopHotItem(ctx context.Context, req *adminv1.ShopHotItemForm) error {
	return c.shopHotItemCase.UpdateShopHotItem(ctx, req)
}

// DeleteShopHotItem 删除热门专区项目
func (c *ShopHotCase) DeleteShopHotItem(ctx context.Context, id string) error {
	return c.shopHotItemCase.DeleteShopHotItem(ctx, id)
}

// SetShopHotItemStatus 设置热门专区项目状态
func (c *ShopHotCase) SetShopHotItemStatus(ctx context.Context, req *adminv1.SetShopHotItemStatusRequest) error {
	return c.shopHotItemCase.SetShopHotItemStatus(ctx, req)
}
