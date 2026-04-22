package biz

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
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
	member := utils.IsMember(ctx)

	query := c.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.Eq(id)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	info, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	price := info.Price
	// 会员访问时，详情页优先展示会员价。
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

// PageGoodsInfo 查询商品分页列表。
func (c *GoodsInfoCase) PageGoodsInfo(ctx context.Context, req *app.PageGoodsInfoRequest, extraOpts ...repo.QueryOption) (*app.PageGoodsInfoResponse, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	query := c.Query(ctx)
	goodsQuery := query.GoodsInfo
	opts := make([]repo.QueryOption, 0, 5+len(extraOpts))
	opts = append(opts, repo.Order(goodsQuery.CreatedAt.Desc()))
	opts = append(opts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))

	// 传入商品名称时，按名称模糊匹配商品。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(goodsQuery.Name.Like("%"+req.GetName()+"%")))
	}

	// 传入分类时，按分类或分类树范围过滤商品。
	if req.GetCategoryId() > 0 {
		// 顶级分类需要展开为其子分类后再查询商品分类 ID
		category, err := c.goodsCategoryRepo.FindById(ctx, req.GetCategoryId())
		if err != nil {
			return nil, err
		}

		// 顶级分类需要展开到全部子分类一起查询。
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
	opts = append(opts, extraOpts...)
	page, count, err := c.GoodsInfoRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*app.GoodsInfo, 0)
	for _, item := range page {
		goodsInfo := c.convertToProto(item, member)
		list = append(list, goodsInfo)
	}

	return &app.PageGoodsInfoResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// listAvailableGoodsIds 按商品 ID 顺序过滤出当前可展示的商品编号。
func (c *GoodsInfoCase) listAvailableGoodsIds(ctx context.Context, goodsIds []int64) ([]int64, error) {
	result := make([]int64, 0, len(goodsIds))
	// 没有候选商品时，无需访问数据库。
	if len(goodsIds) == 0 {
		return result, nil
	}

	query := c.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.ID.In(goodsIds...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIdSet := make(map[int64]struct{}, len(list))
	for _, item := range list {
		goodsIdSet[item.ID] = struct{}{}
	}
	for _, goodsId := range goodsIds {
		// 推荐结果里的商品已下架或不存在时，直接跳过当前无效商品。
		if _, ok := goodsIdSet[goodsId]; !ok {
			continue
		}
		result = append(result, goodsId)
	}
	return result, nil
}

// listByGoodsIds 按商品 ID 顺序查询商品信息。
func (c *GoodsInfoCase) listByGoodsIds(ctx context.Context, goodsIds []int64) ([]*app.GoodsInfo, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	result := make([]*app.GoodsInfo, 0, len(goodsIds))
	// 没有候选商品时，无需访问数据库。
	if len(goodsIds) == 0 {
		return result, nil
	}

	query := c.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(query.ID.In(goodsIds...)))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsMap := make(map[int64]*models.GoodsInfo, len(list))
	for _, item := range list {
		goodsMap[item.ID] = item
	}
	for _, goodsId := range goodsIds {
		item, ok := goodsMap[goodsId]
		// 查询结果缺少对应商品时，直接跳过无效 ID。
		if !ok {
			continue
		}

		goodsInfo := c.convertToProto(item, member)
		result = append(result, goodsInfo)
	}
	return result, nil
}

// listCategoryIdsByGoodsIds 根据商品 ID 列表查询分类 ID 列表。
func (c *GoodsInfoCase) listCategoryIdsByGoodsIds(ctx context.Context, goodsIds []int64) ([]int64, error) {
	// 没有商品上下文时无需访问数据库查询类目。
	if len(goodsIds) == 0 {
		return []int64{}, nil
	}

	query := c.GoodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.ID.In(goodsIds...)))
	list, err := c.GoodsInfoRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0, len(list))

	for _, item := range list {
		categoryIds = append(categoryIds, item.CategoryID)
	}
	return _slice.Unique(categoryIds), nil
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
		Where(query.ID.Eq(goodsId), query.Inventory.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 未命中更新时，需要把“商品不存在”和“库存不足”区分成可判断的业务错误。
	if res.RowsAffected == 0 {
		goodsInfo, findErr := c.FindById(ctx, goodsId)
		// 商品已经不存在时，当前下单请求不应继续执行。
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errorsx.ResourceNotFound("商品不存在")
			}
			return findErr
		}
		return errorsx.StateConflict(
			"商品库存不足",
			"goods_info",
			strconv.FormatInt(goodsInfo.Inventory, 10),
			strconv.FormatInt(num, 10),
		)
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
	// 回退未命中时，说明商品已不存在或销量聚合数据已经异常。
	if res.RowsAffected == 0 {
		goodsInfo, findErr := c.FindById(ctx, goodsId)
		// 商品记录缺失时，当前库存回退已经无法可靠执行。
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errorsx.Internal("商品库存回退失败，商品不存在")
			}
			return findErr
		}
		return errorsx.Internal(
			fmt.Sprintf(
				"商品库存回退失败，商品销量数据异常：goodsId=%d，当前销量=%d，回退数量=%d",
				goodsId,
				goodsInfo.RealSaleNum,
				num,
			),
		)
	}
	return res.Error

}

// convertToProto 转换单个商品为接口返回结构
func (c *GoodsInfoCase) convertToProto(item *models.GoodsInfo, member bool) *app.GoodsInfo {
	goodsInfo := c.listMapper.ToDTO(item)
	// 会员使用会员价，普通用户返回标准售价。
	// 会员访问时，优先返回会员价。
	if member {
		goodsInfo.Price = item.DiscountPrice
	} else {
		goodsInfo.Price = item.Price
	}
	goodsInfo.SaleNum = item.InitSaleNum + item.RealSaleNum
	return goodsInfo
}
