package biz

import (
	"context"
	"errors"
	"sort"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// ShopHotItemCase 热门推荐项业务处理对象
type ShopHotItemCase struct {
	*biz.BaseCase
	*data.ShopHotItemRepository
	shopHotRepo      *data.ShopHotRepository
	shopHotGoodsRepo *data.ShopHotGoodsRepository
	goodsInfoRepo    *data.GoodsInfoRepository
	mapper           *mapper.CopierMapper[appv1.ShopHotItem, models.ShopHotItem]
	goodsMapper      *mapper.CopierMapper[appv1.GoodsInfo, models.GoodsInfo]
}

// NewShopHotItemCase 创建热门推荐项业务处理对象
func NewShopHotItemCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepository, shopHotItemRepo *data.ShopHotItemRepository, shopHotGoodsRepo *data.ShopHotGoodsRepository, goodsInfoRepo *data.GoodsInfoRepository) *ShopHotItemCase {
	return &ShopHotItemCase{
		BaseCase:              baseCase,
		ShopHotItemRepository: shopHotItemRepo,
		shopHotRepo:           shopHotRepo,
		shopHotGoodsRepo:      shopHotGoodsRepo,
		goodsInfoRepo:         goodsInfoRepo,
		mapper:                mapper.NewCopierMapper[appv1.ShopHotItem, models.ShopHotItem](),
		goodsMapper:           mapper.NewCopierMapper[appv1.GoodsInfo, models.GoodsInfo](),
	}
}

// ListShopHotItems 查询热门推荐选项
func (c *ShopHotItemCase) ListShopHotItems(ctx context.Context, id int64) (*appv1.ListShopHotItemsResponse, error) {
	shopHotQuery := c.shopHotRepo.Query(ctx).ShopHot
	shopHotOpts := make([]repository.QueryOption, 0, 2)
	shopHotOpts = append(shopHotOpts, repository.Where(shopHotQuery.ID.Eq(id)))
	shopHotOpts = append(shopHotOpts, repository.Where(shopHotQuery.Status.Eq(_const.STATUS_ENABLE)))
	shopHot, err := c.shopHotRepo.Find(ctx, shopHotOpts...)
	if err != nil {
		// 推荐专区不存在或已禁用时，对外按资源不存在返回，避免公共页面记录 500。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("热门推荐不存在").WithCause(err)
		}
		return nil, err
	}

	var all []*models.ShopHotItem

	shopHotItemQuery := c.Query(ctx).ShopHotItem
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(shopHotItemQuery.Sort.Asc()))
	opts = append(opts, repository.Order(shopHotItemQuery.CreatedAt.Desc()))
	opts = append(opts, repository.Where(shopHotItemQuery.HotID.Eq(shopHot.ID)))
	all, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.ShopHotItem, 0, len(all))
	for _, item := range all {
		list = append(list, c.mapper.ToDTO(item))
	}

	return &appv1.ListShopHotItemsResponse{
		Id:           shopHot.ID,
		Title:        shopHot.Title,
		Banner:       shopHot.Banner,
		ShopHotItems: list,
	}, nil
}

// PageShopHotGoods 查询热门推荐商品
func (c *ShopHotItemCase) PageShopHotGoods(ctx context.Context, req *appv1.PageShopHotGoodsRequest) (*appv1.PageShopHotGoodsResponse, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	list := make([]*appv1.GoodsInfo, 0)
	offset, limit := repository.PageOffsetLimit(req.GetPageNum(), req.GetPageSize())

	hotGoodsQuery := c.shopHotGoodsRepo.Query(ctx).ShopHotGoods
	hotGoodsOpts := make([]repository.QueryOption, 0, 2)
	hotGoodsOpts = append(hotGoodsOpts, repository.Where(hotGoodsQuery.HotItemID.Eq(req.GetHotItemId())))
	hotGoodsOpts = append(hotGoodsOpts, repository.Order(hotGoodsQuery.Sort.Asc()))
	hotGoodsList, err := c.shopHotGoodsRepo.List(ctx, hotGoodsOpts...)
	if err != nil {
		return nil, err
	}
	// 当前分组没有可展示商品时，直接返回空分页结果。
	if len(hotGoodsList) == 0 {
		return &appv1.PageShopHotGoodsResponse{
			GoodsInfos: list,
			Total:      0,
		}, nil
	}

	goodsIDs := make([]int64, 0, len(hotGoodsList))
	for _, hotGoods := range hotGoodsList {
		goodsIDs = append(goodsIDs, hotGoods.GoodsID)
	}

	goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
	goodsOpts := make([]repository.QueryOption, 0, 3)
	goodsOpts = append(goodsOpts, repository.Where(goodsQuery.DeletedAt.IsNull()))
	goodsOpts = append(goodsOpts, repository.Where(goodsQuery.ID.In(goodsIDs...)))
	goodsOpts = append(goodsOpts, repository.Where(goodsQuery.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
	var goodsList []*models.GoodsInfo
	goodsList, err = c.goodsInfoRepo.List(ctx, goodsOpts...)
	if err != nil {
		return nil, err
	}
	goodsMap := make(map[int64]*models.GoodsInfo, len(goodsList))
	for _, goodsInfo := range goodsList {
		goodsMap[goodsInfo.ID] = goodsInfo
	}

	availableItems := make([]*shopHotGoodsItem, 0, len(hotGoodsList))
	for _, hotGoods := range hotGoodsList {
		goodsInfo, ok := goodsMap[hotGoods.GoodsID]
		// 关联商品已下架或不存在时，不计入当前推荐分页。
		if !ok {
			continue
		}
		availableItems = append(availableItems, &shopHotGoodsItem{
			hotGoods:  hotGoods,
			goodsInfo: goodsInfo,
		})
	}
	sort.SliceStable(availableItems, func(i, j int) bool {
		// 推荐位排序相同时，按商品创建时间倒序保持原有展示规则。
		if availableItems[i].hotGoods.Sort == availableItems[j].hotGoods.Sort {
			return availableItems[i].goodsInfo.CreatedAt.After(availableItems[j].goodsInfo.CreatedAt)
		}
		return availableItems[i].hotGoods.Sort < availableItems[j].hotGoods.Sort
	})

	count := int64(len(availableItems))
	start := int(offset)
	if start >= len(availableItems) {
		return &appv1.PageShopHotGoodsResponse{
			GoodsInfos: list,
			Total:      int32(count),
		}, nil
	}
	end := start + int(limit)
	if end > len(availableItems) {
		end = len(availableItems)
	}

	for _, item := range availableItems[start:end] {
		goodsInfo := item.goodsInfo
		price := goodsInfo.Price
		// 会员访问时，优先展示会员价。
		if member {
			price = goodsInfo.DiscountPrice
		}
		goods := c.goodsMapper.ToDTO(goodsInfo)
		goods.SaleNum = goodsInfo.InitSaleNum + goodsInfo.RealSaleNum
		goods.Price = price
		list = append(list, goods)
	}
	return &appv1.PageShopHotGoodsResponse{
		GoodsInfos: list,
		Total:      int32(count),
	}, nil
}

// shopHotGoodsItem 表示已过滤有效商品后的推荐关联项。
type shopHotGoodsItem struct {
	hotGoods  *models.ShopHotGoods
	goodsInfo *models.GoodsInfo
}
