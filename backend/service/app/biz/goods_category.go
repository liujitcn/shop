package biz

import (
	"context"
	"encoding/json"

	_const "shop/pkg/const"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsCategoryCase 商品分类业务处理对象
type GoodsCategoryCase struct {
	*biz.BaseCase
	*data.GoodsCategoryRepository
	goodsInfoRepo *data.GoodsInfoRepository
	mapper        *mapper.CopierMapper[appv1.GoodsCategory, models.GoodsCategory]
	goodsMapper   *mapper.CopierMapper[appv1.GoodsInfo, models.GoodsInfo]
}

// NewGoodsCategoryCase 创建商品分类业务处理对象
func NewGoodsCategoryCase(baseCase *biz.BaseCase, goodsCategoryRepo *data.GoodsCategoryRepository, goodsInfoRepo *data.GoodsInfoRepository) *GoodsCategoryCase {
	goodsMapper := mapper.NewCopierMapper[appv1.GoodsInfo, models.GoodsInfo]()
	goodsMapper.AppendConverters(mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	return &GoodsCategoryCase{
		BaseCase:                baseCase,
		GoodsCategoryRepository: goodsCategoryRepo,
		goodsInfoRepo:           goodsInfoRepo,
		mapper:                  mapper.NewCopierMapper[appv1.GoodsCategory, models.GoodsCategory](),
		goodsMapper:             goodsMapper,
	}
}

// ListGoodsCategories 查询分类列表
func (c *GoodsCategoryCase) ListGoodsCategories(ctx context.Context, req *appv1.ListGoodsCategoriesRequest) (*appv1.ListGoodsCategoriesResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	member := utils.IsMember(ctx)
	list := make([]*appv1.GoodsCategory, 0, len(all))
	categoryGoodsList := make([]*models.GoodsInfo, 0)
	categoryGoodsLoaded := false
	for _, item := range all {
		category := c.mapper.ToDTO(item)
		// 二级分类需要同时返回分类下的推荐商品
		if category.ParentId > 0 {
			// 首次遇到二级分类时，一次性读取可展示商品，后续按分类在内存中筛选。
			if !categoryGoodsLoaded {
				goodsQuery := c.goodsInfoRepo.Query(ctx).GoodsInfo
				goodsOpts := make([]repository.QueryOption, 0, 3)
				goodsOpts = append(goodsOpts, repository.Where(goodsQuery.DeletedAt.IsNull()))
				goodsOpts = append(goodsOpts, repository.Where(goodsQuery.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
				goodsOpts = append(goodsOpts, repository.Order(goodsQuery.CreatedAt.Desc()))
				categoryGoodsList, err = c.goodsInfoRepo.List(ctx, goodsOpts...)
				if err != nil {
					return nil, err
				}
				categoryGoodsLoaded = true
			}
			goodsInfoList := make([]*models.GoodsInfo, 0, 9)
			for _, goodsInfo := range categoryGoodsList {
				categoryIDs := make([]int64, 0)
				parseErr := json.Unmarshal([]byte(goodsInfo.CategoryID), &categoryIDs)
				// 分类 JSON 异常时，跳过当前商品，避免脏数据影响分类页整体展示。
				if parseErr != nil {
					continue
				}
				matchedCategory := false
				for _, item := range categoryIDs {
					// 命中当前分类时，商品可在该分类下展示。
					if item == category.Id {
						matchedCategory = true
						break
					}
				}
				// 商品不属于当前分类时，继续检查下一件商品。
				if !matchedCategory {
					continue
				}
				goodsInfoList = append(goodsInfoList, goodsInfo)
				// 达到分类展示数量上限后，立即停止筛选。
				if len(goodsInfoList) >= 9 {
					break
				}
			}
			category.Goods = make([]*appv1.GoodsInfo, 0, len(goodsInfoList))
			for _, goodsInfo := range goodsInfoList {
				price := goodsInfo.Price
				// 会员用户访问分类时，优先展示会员价。
				if member {
					price = goodsInfo.DiscountPrice
				}
				goods := c.goodsMapper.ToDTO(goodsInfo)
				goods.SaleNum = goodsInfo.InitSaleNum + goodsInfo.RealSaleNum
				goods.Price = price
				category.Goods = append(category.Goods, goods)
			}
		}
		list = append(list, category)
	}

	return &appv1.ListGoodsCategoriesResponse{
		GoodsCategories: list,
	}, nil
}
