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

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsCase 商品业务处理对象
type GoodsCase struct {
	*biz.BaseCase
	*data.GoodsRepo
	goodsCategoryRepo *data.GoodsCategoryRepo
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSkuCase      *GoodsSkuCase
}

// NewGoodsCase 创建商品业务处理对象
func NewGoodsCase(
	baseCase *biz.BaseCase,
	goodsInfoRepo *data.GoodsRepo,
	goodsCategoryRepo *data.GoodsCategoryRepo,
	goodsPropCase *GoodsPropCase,
	goodsSpecCase *GoodsSpecCase,
	goodsSkuCase *GoodsSkuCase,
) *GoodsCase {
	return &GoodsCase{
		BaseCase:          baseCase,
		GoodsRepo:         goodsInfoRepo,
		goodsCategoryRepo: goodsCategoryRepo,
		goodsPropCase:     goodsPropCase,
		goodsSpecCase:     goodsSpecCase,
		goodsSkuCase:      goodsSkuCase,
	}
}

// GetGoods 查询商品详情
func (c *GoodsCase) GetGoods(ctx context.Context, id int64) (*app.GoodsResponse, error) {
	// 是否会员
	member := util.IsMember(ctx)

	query := c.Query(ctx).Goods

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

	goodsInfo := &app.GoodsResponse{
		Id:         info.ID,
		CategoryId: info.CategoryID,
		Name:       info.Name,
		Desc:       info.Desc,
		Price:      price,
		SaleNum:    info.InitSaleNum + info.RealSaleNum,
		Picture:    info.Picture,
		Banner:     _string.ConvertJsonStringToStringArray(info.Banner),
		Detail:     _string.ConvertJsonStringToStringArray(info.Detail),
	}
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

// PageGoods 查询商品分页列表
func (c *GoodsCase) PageGoods(ctx context.Context, req *app.PageGoodsRequest) (*app.PageGoodsResponse, error) {
	// 是否会员
	member := util.IsMember(ctx)
	query := c.Query(ctx)
	goodsQuery := query.Goods
	opts := make([]repo.QueryOption, 0, 4)
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

	page, count, err := c.GoodsRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.Goods, 0)
	for _, item := range page {
		price := item.Price
		if member {
			price = item.DiscountPrice
		}
		list = append(list, &app.Goods{
			Id:      item.ID,
			Name:    item.Name,
			Desc:    item.Desc,
			Picture: item.Picture,
			SaleNum: item.InitSaleNum + item.RealSaleNum,
			Price:   price,
		})
	}

	return &app.PageGoodsResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// 按商品编号批量查询并组装映射
func (c *GoodsCase) mapByGoodsIds(ctx context.Context, goodsIds []int64) (map[int64]*models.Goods, error) {
	all, err := c.ListByIds(ctx, goodsIds)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*models.Goods)
	for _, item := range all {
		res[item.ID] = item
	}
	return res, nil
}

// 增加商品销量
func (c *GoodsCase) addSaleNum(ctx context.Context, goodsId, num int64) error {
	query := c.Query(ctx).Goods
	res, err := query.WithContext(ctx).
		Where(query.ID.Eq(goodsId)).
		Update(query.RealSaleNum, query.RealSaleNum.Add(num))
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error
}

// 回退商品销量
func (c *GoodsCase) subSaleNum(ctx context.Context, goodsId, num int64) error {
	query := c.Query(ctx).Goods
	res, err := query.WithContext(ctx).
		Where(query.ID.Eq(goodsId), query.RealSaleNum.Gte(num)).
		Update(query.RealSaleNum, query.RealSaleNum.Sub(num))
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return res.Error

}
