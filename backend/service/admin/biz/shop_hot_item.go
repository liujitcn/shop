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

// ShopHotItemCase 热门专区项目业务实例
type ShopHotItemCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.ShopHotItemRepo
	shopHotGoodsRepo *data.ShopHotGoodsRepo
	formMapper       *mapper.CopierMapper[admin.ShopHotItemForm, models.ShopHotItem]
	mapper           *mapper.CopierMapper[admin.ShopHotItem, models.ShopHotItem]
}

// NewShopHotItemCase 创建热门专区项目业务实例
func NewShopHotItemCase(baseCase *biz.BaseCase, tx data.Transaction, shopHotRepo *data.ShopHotRepo, shopHotItemRepo *data.ShopHotItemRepo, shopHotGoodsRepo *data.ShopHotGoodsRepo) *ShopHotItemCase {
	return &ShopHotItemCase{
		BaseCase:         baseCase,
		tx:               tx,
		ShopHotItemRepo:  shopHotItemRepo,
		shopHotGoodsRepo: shopHotGoodsRepo,
		formMapper:       mapper.NewCopierMapper[admin.ShopHotItemForm, models.ShopHotItem](),
		mapper:           mapper.NewCopierMapper[admin.ShopHotItem, models.ShopHotItem](),
	}
}

// PageShopHotItem 分页查询热门专区项目
func (c *ShopHotItemCase) PageShopHotItem(ctx context.Context, req *admin.PageShopHotItemRequest) (*admin.PageShopHotItemResponse, error) {
	query := c.Query(ctx).ShopHotItem
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 传入专区编号时，仅查询指定热门专区下的项目。
	if req.GetHotId() > 0 {
		opts = append(opts, repo.Where(query.HotID.Eq(req.GetHotId())))
	}
	// 传入标题关键字时，按标题模糊匹配热门专区项目。
	if req.GetTitle() != "" {
		opts = append(opts, repo.Where(query.Title.Like("%"+req.GetTitle()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.ShopHotItem, 0, len(list))
	for _, item := range list {
		shopHotItem := c.mapper.ToDTO(item)
		resList = append(resList, shopHotItem)
	}
	return &admin.PageShopHotItemResponse{List: resList, Total: int32(total)}, nil
}

// GetShopHotItem 获取热门专区项目
func (c *ShopHotItemCase) GetShopHotItem(ctx context.Context, id int64) (*admin.ShopHotItemForm, error) {
	shopHotItem, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	res := c.formMapper.ToDTO(shopHotItem)
	query := c.shopHotGoodsRepo.Query(ctx).ShopHotGoods
	var hotGoodsList []*models.ShopHotGoods
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Where(query.HotItemID.Eq(shopHotItem.ID)))
	hotGoodsList, err = c.shopHotGoodsRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(hotGoodsList))
	for _, item := range hotGoodsList {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	res.GoodsIds = goodsIds
	return res, nil
}

// CreateShopHotItem 创建热门专区项目
func (c *ShopHotItemCase) CreateShopHotItem(ctx context.Context, req *admin.ShopHotItemForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		shopHotItem := c.formMapper.ToEntity(req)
		err := c.Create(ctx, shopHotItem)
		if err != nil {
			return err
		}
		return c.replaceShopHotGoods(ctx, shopHotItem.ID, req.GetGoodsIds())
	})
}

// UpdateShopHotItem 更新热门专区项目
func (c *ShopHotItemCase) UpdateShopHotItem(ctx context.Context, req *admin.ShopHotItemForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		shopHotItem := c.formMapper.ToEntity(req)
		err := c.UpdateById(ctx, shopHotItem)
		if err != nil {
			return err
		}
		return c.replaceShopHotGoods(ctx, req.GetId(), req.GetGoodsIds())
	})
}

// DeleteShopHotItem 删除热门专区项目
func (c *ShopHotItemCase) DeleteShopHotItem(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.DeleteByIds(ctx, ids)
		if err != nil {
			return err
		}
		query := c.shopHotGoodsRepo.Query(ctx).ShopHotGoods
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(query.HotItemID.In(ids...)))
		return c.shopHotGoodsRepo.Delete(ctx, opts...)
	})
}

// SetShopHotItemStatus 设置热门专区项目状态
func (c *ShopHotItemCase) SetShopHotItemStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.ShopHotItem{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// replaceShopHotGoods 替换热门选项商品
func (c *ShopHotItemCase) replaceShopHotGoods(ctx context.Context, hotItemId int64, goodsIds []int64) error {
	query := c.shopHotGoodsRepo.Query(ctx).ShopHotGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.HotItemID.Eq(hotItemId)))
	err := c.shopHotGoodsRepo.Delete(ctx, opts...)
	if err != nil {
		return err
	}

	list := make([]*models.ShopHotGoods, 0, len(goodsIds))
	for idx, goodsId := range goodsIds {
		list = append(list, &models.ShopHotGoods{
			HotItemID: hotItemId,
			GoodsID:   goodsId,
			Sort:      int64(idx + 1),
		})
	}
	return c.shopHotGoodsRepo.BatchCreate(ctx, list)
}
