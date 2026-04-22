package biz

import (
	"context"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// ShopHotItemCase 热门推荐项业务处理对象
type ShopHotItemCase struct {
	*biz.BaseCase
	*data.ShopHotItemRepo
	shopHotRepo      *data.ShopHotRepo
	shopHotGoodsRepo *data.ShopHotGoodsRepo
	goodsInfoRepo    *data.GoodsInfoRepo
	mapper           *mapper.CopierMapper[app.ShopHotItem, models.ShopHotItem]
	goodsMapper      *mapper.CopierMapper[app.GoodsInfo, models.GoodsInfo]
}

// NewShopHotItemCase 创建热门推荐项业务处理对象
func NewShopHotItemCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepo, shopHotItemRepo *data.ShopHotItemRepo, shopHotGoodsRepo *data.ShopHotGoodsRepo, goodsInfoRepo *data.GoodsInfoRepo) *ShopHotItemCase {
	return &ShopHotItemCase{
		BaseCase:         baseCase,
		ShopHotItemRepo:  shopHotItemRepo,
		shopHotRepo:      shopHotRepo,
		shopHotGoodsRepo: shopHotGoodsRepo,
		goodsInfoRepo:    goodsInfoRepo,
		mapper:           mapper.NewCopierMapper[app.ShopHotItem, models.ShopHotItem](),
		goodsMapper:      mapper.NewCopierMapper[app.GoodsInfo, models.GoodsInfo](),
	}
}

// ListShopHotItem 查询热门推荐选项
func (c *ShopHotItemCase) ListShopHotItem(ctx context.Context, id int64) (*app.ListShopHotItemResponse, error) {
	shopHotQuery := c.shopHotRepo.Query(ctx).ShopHot
	shopHotOpts := make([]repo.QueryOption, 0, 2)
	shopHotOpts = append(shopHotOpts, repo.Where(shopHotQuery.ID.Eq(id)))
	shopHotOpts = append(shopHotOpts, repo.Where(shopHotQuery.Status.Eq(int32(common.Status_ENABLE))))
	shopHot, err := c.shopHotRepo.Find(ctx, shopHotOpts...)
	if err != nil {
		return nil, err
	}

	var all []*models.ShopHotItem

	shopHotItemQuery := c.Query(ctx).ShopHotItem
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Order(shopHotItemQuery.Sort.Asc()))
	opts = append(opts, repo.Order(shopHotItemQuery.CreatedAt.Desc()))
	opts = append(opts, repo.Where(shopHotItemQuery.HotID.Eq(shopHot.ID)))
	all, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*app.ShopHotItem, 0, len(all))
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}

	return &app.ListShopHotItemResponse{
		Id:     shopHot.ID,
		Title:  shopHot.Title,
		Banner: shopHot.Banner,
		List:   list,
	}, nil
}

// PageShopHotGoods 查询热门推荐商品
func (c *ShopHotItemCase) PageShopHotGoods(ctx context.Context, req *app.PageShopHotGoodsRequest) (*app.PageShopHotGoodsResponse, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	list := make([]*app.GoodsInfo, 0)
	offset, limit := repo.PageOffsetLimit(req.GetPageNum(), req.GetPageSize())
	baseDB := c.shopHotGoodsRepo.Query(ctx).ShopHotGoods.WithContext(ctx).UnderlyingDB().
		Table(models.TableNameShopHotGoods+" AS hot_goods").
		Joins("JOIN "+models.TableNameGoodsInfo+" ON "+models.TableNameGoodsInfo+".id = hot_goods.goods_id").
		Where("hot_goods.hot_item_id = ?", req.GetHotItemId()).
		Where(models.TableNameGoodsInfo+".deleted_at IS NULL").
		Where(models.TableNameGoodsInfo+".status = ?", int32(common.GoodsStatus_PUT_ON))

	count := int64(0)
	// 热门推荐总数必须只统计当前仍可展示的上架商品，避免把已下架商品继续算进分页。
	err := baseDB.Count(&count).Error
	if err != nil {
		return nil, err
	}
	// 当前分组没有可展示商品时，直接返回空分页结果。
	if count == 0 {
		return &app.PageShopHotGoodsResponse{
			List:  list,
			Total: 0,
		}, nil
	}

	all := make([]*models.GoodsInfo, 0)
	// 直接按推荐位顺序查询当前仍可展示的商品，避免二次回表后把下架商品重新掺进结果。
	err = baseDB.
		Select(models.TableNameGoodsInfo + ".*").
		Order("hot_goods.sort ASC").
		Order(models.TableNameGoodsInfo + ".created_at DESC").
		Offset(int(offset)).
		Limit(int(limit)).
		Scan(&all).Error
	if err != nil {
		return nil, err
	}
	for _, item := range all {
		price := item.Price
		// 会员访问时，优先展示会员价。
		if member {
			price = item.DiscountPrice
		}
		goodsInfo := c.goodsMapper.ToDTO(item)
		goodsInfo.SaleNum = item.InitSaleNum + item.RealSaleNum
		goodsInfo.Price = price
		list = append(list, goodsInfo)
	}
	return &app.PageShopHotGoodsResponse{
		List:  list,
		Total: int32(count),
	}, nil
}
