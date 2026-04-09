package biz

import (
	"context"
	"fmt"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/service/app/util"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsInfoCase 商品业务处理对象
type GoodsInfoCase struct {
	*biz.BaseCase
	*data.GoodsInfoRepo
	goodsCategoryRepo *data.GoodsCategoryRepo
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSkuCase      *GoodsSkuCase
	responseMapper    *mapper.CopierMapper[app.GoodsInfoResponse, models.GoodsInfo]
	listMapper        *mapper.CopierMapper[app.GoodsInfo, models.GoodsInfo]
}

// NewGoodsInfoCase 创建商品业务处理对象
func NewGoodsInfoCase(
	baseCase *biz.BaseCase,
	goodsInfoRepo *data.GoodsInfoRepo,
	goodsCategoryRepo *data.GoodsCategoryRepo,
	goodsPropCase *GoodsPropCase,
	goodsSpecCase *GoodsSpecCase,
	goodsSkuCase *GoodsSkuCase,
) *GoodsInfoCase {
	responseMapper := mapper.NewCopierMapper[app.GoodsInfoResponse, models.GoodsInfo]()
	responseMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	listMapper := mapper.NewCopierMapper[app.GoodsInfo, models.GoodsInfo]()
	return &GoodsInfoCase{
		BaseCase:          baseCase,
		GoodsInfoRepo:     goodsInfoRepo,
		goodsCategoryRepo: goodsCategoryRepo,
		goodsPropCase:     goodsPropCase,
		goodsSpecCase:     goodsSpecCase,
		goodsSkuCase:      goodsSkuCase,
		responseMapper:    responseMapper,
		listMapper:        listMapper,
	}
}

// GetGoodsInfo 查询商品详情
func (c *GoodsInfoCase) GetGoodsInfo(ctx context.Context, id int64) (*app.GoodsInfoResponse, error) {
	// 是否会员
	member := util.IsMember(ctx)

	query := c.Query(ctx).GoodsInfo

	info, err := c.Find(ctx,
		repo.Where(query.ID.Eq(id)),
		repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))),
	)
	if err != nil {
		return nil, err
	}
	price := info.Price
	if member {
		price = info.DiscountPrice
	}

	goodsInfo := c.responseMapper.ToDTO(info)
	goodsInfo.Price = price
	goodsInfo.SaleNum = info.InitSaleNum + info.RealSaleNum
	// 属性
	goodsInfo.PropList, err = c.goodsPropCase.listByGoodsId(ctx, goodsInfo.Id)
	if err != nil {
		return nil, err
	}
	// 规格
	goodsInfo.SpecList, err = c.goodsSpecCase.listByGoodsId(ctx, goodsInfo.Id)
	if err != nil {
		return nil, err
	}
	// 规格库存
	goodsInfo.SkuList, err = c.goodsSkuCase.listByGoodsId(ctx, goodsInfo.Id, member)
	if err != nil {
		return nil, err
	}
	return goodsInfo, nil
}

// PageGoodsInfo 查询商品分页列表
func (c *GoodsInfoCase) PageGoodsInfo(ctx context.Context, req *app.PageGoodsInfoRequest) (*app.PageGoodsInfoResponse, error) {
	// 是否会员
	member := util.IsMember(ctx)
	query := c.Query(ctx)
	goodsQuery := query.GoodsInfo
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(goodsQuery.CreatedAt.Desc()))
	opts = append(opts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))

	if req.GetName() != "" {
		opts = append(opts, repo.Where(goodsQuery.Name.Like("%"+req.GetName()+"%")))
	}

	if req.GetCategoryId() > 0 {
		// 顶级分类需要展开为其子分类后再查询商品分类 ID
		category, err := c.goodsCategoryRepo.FindById(ctx, req.GetCategoryId())
		if err != nil {
			return nil, err
		}

		if category.ParentID == 0 {
			categoryQuery := query.GoodsCategory
			opts = append(opts, repo.Join(
				categoryQuery,
				categoryQuery.ID.EqCol(goodsQuery.CategoryID),
				categoryQuery.Path.Like(fmt.Sprintf("%s%%", category.Path+"/")),
			))
		} else {
			opts = append(opts, repo.Where(goodsQuery.CategoryID.Eq(req.GetCategoryId())))
		}
	}
	page, count, err := c.GoodsInfoRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.GoodsInfo, 0)
	for _, item := range page {
		price := item.Price
		if member {
			price = item.DiscountPrice
		}
		goodsInfo := c.listMapper.ToDTO(item)
		goodsInfo.SaleNum = item.InitSaleNum + item.RealSaleNum
		goodsInfo.Price = price
		list = append(list, goodsInfo)
	}

	return &app.PageGoodsInfoResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// 按商品编号批量查询并组装映射
func (c *GoodsInfoCase) mapByGoodsIds(ctx context.Context, goodsIds []int64) (map[int64]*models.GoodsInfo, error) {
	all, err := c.ListByIds(ctx, goodsIds)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*models.GoodsInfo)
	for _, item := range all {
		res[item.ID] = item
	}
	return res, nil
}

// 增加商品销量
func (c *GoodsInfoCase) addSaleNum(ctx context.Context, goodsId, num int64) error {
	query := c.Query(ctx).GoodsInfo
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Add(num),
		"inventory":     query.Inventory.Sub(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.ID.Eq(goodsId)).
		Updates(updates)
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}

// 回退商品销量
func (c *GoodsInfoCase) subSaleNum(ctx context.Context, goodsId, num int64) error {
	query := c.Query(ctx).GoodsInfo
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Sub(num),
		"inventory":     query.Inventory.Add(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.ID.Eq(goodsId), query.RealSaleNum.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error

}
