package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	_const "shop/pkg/const"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	appv1 "shop/api/gen/go/app/v1"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// GoodsInfoCase 商品业务处理对象
type GoodsInfoCase struct {
	*biz.BaseCase
	*data.GoodsInfoRepository
	goodsCategoryRepo *data.GoodsCategoryRepository
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSKUCase      *GoodsSKUCase
	responseMapper    *mapper.CopierMapper[appv1.GoodsInfoResponse, models.GoodsInfo]
	listMapper        *mapper.CopierMapper[appv1.GoodsInfo, models.GoodsInfo]
}

// NewGoodsInfoCase 创建商品业务处理对象
func NewGoodsInfoCase(
	baseCase *biz.BaseCase,
	goodsInfoRepo *data.GoodsInfoRepository,
	goodsCategoryRepo *data.GoodsCategoryRepository,
	goodsPropCase *GoodsPropCase,
	goodsSpecCase *GoodsSpecCase,
	goodsSKUCase *GoodsSKUCase,
) *GoodsInfoCase {
	responseMapper := mapper.NewCopierMapper[appv1.GoodsInfoResponse, models.GoodsInfo]()
	responseMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	responseMapper.AppendConverters(mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	listMapper := mapper.NewCopierMapper[appv1.GoodsInfo, models.GoodsInfo]()
	listMapper.AppendConverters(mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	return &GoodsInfoCase{
		BaseCase:            baseCase,
		GoodsInfoRepository: goodsInfoRepo,
		goodsCategoryRepo:   goodsCategoryRepo,
		goodsPropCase:       goodsPropCase,
		goodsSpecCase:       goodsSpecCase,
		goodsSKUCase:        goodsSKUCase,
		responseMapper:      responseMapper,
		listMapper:          listMapper,
	}
}

// GetGoodsInfo 查询商品详情
func (c *GoodsInfoCase) GetGoodsInfo(ctx context.Context, id int64) (*appv1.GoodsInfoResponse, error) {
	// 是否会员
	member := utils.IsMember(ctx)

	query := c.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
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
	goodsInfo.PropList, err = c.goodsPropCase.listByGoodsID(ctx, goodsInfo.Id)
	if err != nil {
		return nil, err
	}
	// 规格
	goodsInfo.SpecList, err = c.goodsSpecCase.listByGoodsID(ctx, goodsInfo.Id)
	if err != nil {
		return nil, err
	}
	// 规格库存
	goodsInfo.SkuList, err = c.goodsSKUCase.listByGoodsID(ctx, goodsInfo.Id, member)
	if err != nil {
		return nil, err
	}
	return goodsInfo, nil
}

// PageGoodsInfo 查询商品分页列表。
func (c *GoodsInfoCase) PageGoodsInfo(ctx context.Context, req *appv1.PageGoodsInfoRequest, extraOpts ...repository.QueryOption) (*appv1.PageGoodsInfoResponse, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	query := c.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 5+len(extraOpts))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))

	// 传入商品名称时，按名称模糊匹配商品。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}

	// 传入分类时，按分类或分类树范围过滤商品。
	if req.GetCategoryId() > 0 {
		categoryIDs, categoryErr := c.buildCategoryFilterIDs(ctx, req.GetCategoryId())
		if categoryErr != nil {
			return nil, categoryErr
		}
		goodsIDs := make([]int64, 0)
		goodsIDs, categoryErr = c.findGoodsIDsByCategoryIDs(ctx, categoryIDs)
		if categoryErr != nil {
			return nil, categoryErr
		}
		// 分类条件无命中商品时，直接返回空分页结果。
		if len(goodsIDs) == 0 {
			return &appv1.PageGoodsInfoResponse{GoodsInfos: []*appv1.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	}
	opts = append(opts, extraOpts...)
	page, count, err := c.GoodsInfoRepository.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*appv1.GoodsInfo, 0)
	for _, item := range page {
		goodsInfo := c.convertToProto(item, member)
		list = append(list, goodsInfo)
	}

	return &appv1.PageGoodsInfoResponse{
		GoodsInfos: list,
		Total:      int32(count),
	}, nil
}

// listAvailableGoodsIDs 按商品 ID 顺序过滤出当前可展示的商品编号。
func (c *GoodsInfoCase) listAvailableGoodsIDs(ctx context.Context, goodsIDs []int64) ([]int64, error) {
	result := make([]int64, 0, len(goodsIDs))
	// 没有候选商品时，无需访问数据库。
	if len(goodsIDs) == 0 {
		return result, nil
	}

	query := c.GoodsInfoRepository.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIDSet := make(map[int64]struct{}, len(list))
	for _, item := range list {
		goodsIDSet[item.ID] = struct{}{}
	}
	for _, goodsID := range goodsIDs {
		// 推荐结果里的商品已下架或不存在时，直接跳过当前无效商品。
		if _, ok := goodsIDSet[goodsID]; !ok {
			continue
		}
		result = append(result, goodsID)
	}
	return result, nil
}

// listByGoodsIDs 按商品 ID 顺序查询商品信息。
func (c *GoodsInfoCase) listByGoodsIDs(ctx context.Context, goodsIDs []int64) ([]*appv1.GoodsInfo, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	result := make([]*appv1.GoodsInfo, 0, len(goodsIDs))
	// 没有候选商品时，无需访问数据库。
	if len(goodsIDs) == 0 {
		return result, nil
	}

	query := c.GoodsInfoRepository.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsMap := make(map[int64]*models.GoodsInfo, len(list))
	for _, item := range list {
		goodsMap[item.ID] = item
	}
	for _, goodsID := range goodsIDs {
		item, ok := goodsMap[goodsID]
		// 查询结果缺少对应商品时，直接跳过无效 ID。
		if !ok {
			continue
		}

		goodsInfo := c.convertToProto(item, member)
		result = append(result, goodsInfo)
	}
	return result, nil
}

// listCategoryIDsByGoodsIDs 根据商品 ID 列表查询分类 ID 列表。
func (c *GoodsInfoCase) listCategoryIDsByGoodsIDs(ctx context.Context, goodsIDs []int64) ([]int64, error) {
	// 没有商品上下文时无需访问数据库查询类目。
	if len(goodsIDs) == 0 {
		return []int64{}, nil
	}

	query := c.GoodsInfoRepository.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	list, err := c.GoodsInfoRepository.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	categoryIDs := make([]int64, 0, len(list))

	for _, item := range list {
		categoryIDs = append(categoryIDs, c.parseCategoryIDs(item.CategoryID)...)
	}
	return _slice.Unique(categoryIDs), nil
}

// buildCategoryFilterIDs 构建分类筛选范围。
func (c *GoodsInfoCase) buildCategoryFilterIDs(ctx context.Context, categoryID int64) ([]int64, error) {
	// 先校验分类存在，避免按无效分类编号继续查询商品。
	_, err := c.goodsCategoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	var categoryList []*models.GoodsCategory
	categoryList, err = c.goodsCategoryRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	childMap := make(map[int64][]int64, len(categoryList))
	for _, item := range categoryList {
		childMap[item.ParentID] = append(childMap[item.ParentID], item.ID)
	}

	categoryIDs := []int64{categoryID}
	queue := []int64{categoryID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		childIDs := childMap[currentID]
		// 当前分类没有子分类时，继续展开下一项待处理分类。
		if len(childIDs) == 0 {
			continue
		}
		categoryIDs = append(categoryIDs, childIDs...)
		queue = append(queue, childIDs...)
	}
	return categoryIDs, nil
}

// findGoodsIDsByCategoryIDs 查询命中分类集合的商品编号。
func (c *GoodsInfoCase) findGoodsIDsByCategoryIDs(ctx context.Context, categoryIDs []int64) ([]int64, error) {
	// 没有分类编号时，不需要继续访问数据库查询商品。
	if len(categoryIDs) == 0 {
		return []int64{}, nil
	}

	type goodsCategoryRow struct {
		ID         int64  `gorm:"column:id"`
		CategoryID string `gorm:"column:category_id"`
	}

	query := c.Query(ctx).GoodsInfo
	rows := make([]*goodsCategoryRow, 0)
	err := query.WithContext(ctx).
		Select(query.ID, query.CategoryID).
		Where(query.DeletedAt.IsNull()).
		Scan(&rows)
	if err != nil {
		return nil, err
	}

	categoryIDSet := make(map[int64]struct{}, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		categoryIDSet[categoryID] = struct{}{}
	}

	goodsIDs := make([]int64, 0)
	for _, row := range rows {
		matchedCategory := false
		for _, categoryID := range c.parseCategoryIDs(row.CategoryID) {
			// 命中任一分类时即可认为商品满足分类筛选。
			if _, ok := categoryIDSet[categoryID]; ok {
				matchedCategory = true
				break
			}
		}
		// 商品分类列表命中任一筛选分类时，加入候选商品集合。
		if matchedCategory {
			goodsIDs = append(goodsIDs, row.ID)
		}
	}
	return goodsIDs, nil
}

// parseCategoryIDs 解析商品分类编号列表。
func (c *GoodsInfoCase) parseCategoryIDs(rawCategoryIDs string) []int64 {
	// 分类字段为空时，直接返回空分类列表。
	if strings.TrimSpace(rawCategoryIDs) == "" {
		return []int64{}
	}

	categoryIDs := make([]int64, 0)
	// 分类 JSON 解析失败时，回退为空列表，避免单条脏数据影响推荐与分类查询。
	if err := json.Unmarshal([]byte(rawCategoryIDs), &categoryIDs); err != nil {
		return []int64{}
	}
	return categoryIDs
}

// 按商品编号批量查询并组装映射
func (c *GoodsInfoCase) mapByGoodsIDs(ctx context.Context, goodsIDs []int64) (map[int64]*models.GoodsInfo, error) {
	all, err := c.ListByIDs(ctx, goodsIDs)
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
func (c *GoodsInfoCase) addSaleNum(ctx context.Context, goodsID, num int64) error {
	query := c.Query(ctx).GoodsInfo
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Add(num),
		"inventory":     query.Inventory.Sub(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.ID.Eq(goodsID), query.Inventory.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 未命中更新时，需要把“商品不存在”和“库存不足”区分成可判断的业务错误。
	if res.RowsAffected == 0 {
		goodsInfo, findErr := c.FindByID(ctx, goodsID)
		// 商品已经不存在时，当前下单请求不应继续执行。
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errorsx.ResourceNotFound("商品不存在").WithCause(findErr)
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
func (c *GoodsInfoCase) subSaleNum(ctx context.Context, goodsID, num int64) error {
	query := c.Query(ctx).GoodsInfo
	updates := map[string]interface{}{
		"real_sale_num": query.RealSaleNum.Sub(num),
		"inventory":     query.Inventory.Add(num),
	}
	res, err := query.WithContext(ctx).
		Where(query.ID.Eq(goodsID), query.RealSaleNum.Gte(num)).
		Updates(updates)
	if err != nil {
		return err
	}
	// 回退未命中时，说明商品已不存在或销量聚合数据已经异常。
	if res.RowsAffected == 0 {
		goodsInfo, findErr := c.FindByID(ctx, goodsID)
		// 商品记录缺失时，当前库存回退已经无法可靠执行。
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				return errorsx.Internal("商品库存回退失败，商品不存在").WithCause(findErr)
			}
			return findErr
		}
		return errorsx.Internal(
			fmt.Sprintf(
				"商品库存回退失败，商品销量数据异常：goodsID=%d，当前销量=%d，回退数量=%d",
				goodsID,
				goodsInfo.RealSaleNum,
				num,
			),
		)
	}
	return res.Error

}

// convertToProto 转换单个商品为接口返回结构
func (c *GoodsInfoCase) convertToProto(item *models.GoodsInfo, member bool) *appv1.GoodsInfo {
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
