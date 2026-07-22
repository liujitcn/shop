package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	_const "shop/service/shop/consts"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/service/shop/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	_set "github.com/liujitcn/go-utils/set"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// GoodsInfoCase 商品业务处理对象
type GoodsInfoCase struct {
	*biz.BaseCase
	*data.GoodsInfoRepository
	goodsCategoryCase *GoodsCategoryCase
	tenantStoreCase   *TenantStoreCase
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSKUCase      *GoodsSKUCase
	responseMapper    *mapper.CopierMapper[shopappv1.GoodsInfoResponse, models.GoodsInfo]
	listMapper        *mapper.CopierMapper[shopappv1.GoodsInfo, models.GoodsInfo]
}

// NewGoodsInfoCase 创建商品业务处理对象
func NewGoodsInfoCase(
	baseCase *biz.BaseCase,
	goodsInfoRepo *data.GoodsInfoRepository,
	goodsCategoryCase *GoodsCategoryCase,
	tenantStoreCase *TenantStoreCase,
	goodsPropCase *GoodsPropCase,
	goodsSpecCase *GoodsSpecCase,
	goodsSKUCase *GoodsSKUCase,
) *GoodsInfoCase {
	responseMapper := mapper.NewCopierMapper[shopappv1.GoodsInfoResponse, models.GoodsInfo]()
	responseMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	responseMapper.AppendConverters(mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	listMapper := mapper.NewCopierMapper[shopappv1.GoodsInfo, models.GoodsInfo]()
	listMapper.AppendConverters(mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	return &GoodsInfoCase{
		BaseCase:            baseCase,
		GoodsInfoRepository: goodsInfoRepo,
		goodsCategoryCase:   goodsCategoryCase,
		tenantStoreCase:     tenantStoreCase,
		goodsPropCase:       goodsPropCase,
		goodsSpecCase:       goodsSpecCase,
		goodsSKUCase:        goodsSKUCase,
		responseMapper:      responseMapper,
		listMapper:          listMapper,
	}
}

// PageGoodsInfo 查询商品分页列表。
func (c *GoodsInfoCase) PageGoodsInfo(ctx context.Context, req *shopappv1.PageGoodsInfoRequest, extraOpts ...repository.QueryOption) (*shopappv1.PageGoodsInfoResponse, error) {
	var err error
	// 是否会员
	member := utils.IsMember(ctx)
	query := c.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 5+len(extraOpts))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
	if req.GetTenantStoreId() > 0 {
		opts = append(opts, repository.Where(query.TenantStoreID.Eq(req.GetTenantStoreId())))
	}

	// 传入商品名称时，按名称模糊匹配商品。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}

	// 传入分类时，按分类或分类树范围过滤商品。
	if req.GetCategoryId() > 0 {
		var categoryIDs []int64
		categoryIDs, err = c.buildCategoryFilterIDs(ctx, req.GetCategoryId())
		if err != nil {
			return nil, err
		}
		var goodsIDs []int64
		goodsIDs, err = c.findGoodsIDsByCategoryIDs(ctx, categoryIDs)
		if err != nil {
			return nil, err
		}
		// 分类条件无命中商品时，直接返回空分页结果。
		if len(goodsIDs) == 0 {
			return &shopappv1.PageGoodsInfoResponse{GoodsInfos: []*shopappv1.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	}
	opts = append(opts, extraOpts...)
	var page []*models.GoodsInfo
	var count int64
	page, count, err = c.GoodsInfoRepository.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	list := make([]*shopappv1.GoodsInfo, 0)
	tenantStoreIds := _set.NewSet[int64]()
	for _, goodsInfo := range page {
		tenantStoreIds.Add(goodsInfo.TenantStoreID)
	}

	var tenantStoreMap map[int64]*models.TenantStore
	tenantStoreMap, err = c.tenantStoreCase.GetTenantStoreMapByIDs(ctx, tenantStoreIds.ToSlice())
	if err != nil {
		return nil, err
	}
	for _, item := range page {
		goodsInfo := c.convertToProto(item, member)
		if tenantStore, ok := tenantStoreMap[item.TenantStoreID]; ok {
			goodsInfo.TenantStoreName = tenantStore.Name
		}
		list = append(list, goodsInfo)
	}

	return &shopappv1.PageGoodsInfoResponse{
		GoodsInfos: list,
		Total:      int32(count),
	}, nil
}

// GetGoodsInfo 查询商品详情
func (c *GoodsInfoCase) GetGoodsInfo(ctx context.Context, id int64) (*shopappv1.GoodsInfoResponse, error) {
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
	var tenantStore *models.TenantStore
	tenantStore, err = c.tenantStoreCase.FindByID(ctx, info.TenantStoreID)
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
	goodsInfo.TenantStoreName = tenantStore.Name
	goodsInfo.TenantStoreLogo = tenantStore.Logo
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

// listByGoodsIDs 按商品 ID 顺序查询商品信息。
func (c *GoodsInfoCase) listByGoodsIDs(ctx context.Context, goodsIDs []int64) ([]*shopappv1.GoodsInfo, error) {
	// 是否会员
	member := utils.IsMember(ctx)
	result := make([]*shopappv1.GoodsInfo, 0, len(goodsIDs))
	// 没有候选商品时，无需访问数据库。
	if len(goodsIDs) == 0 {
		return result, nil
	}

	query := c.GoodsInfoRepository.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 4)
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

// convertToProto 转换单个商品为接口返回结构
func (c *GoodsInfoCase) convertToProto(item *models.GoodsInfo, member bool) *shopappv1.GoodsInfo {
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

// mapByGoodsIDs 按商品编号批量查询可展示商品并组装映射。
func (c *GoodsInfoCase) mapByGoodsIDs(ctx context.Context, goodsIDs []int64) (map[int64]*models.GoodsInfo, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	res := make(map[int64]*models.GoodsInfo, len(all))
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
		var goodsInfo *models.GoodsInfo
		goodsInfo, err = c.FindByID(ctx, goodsID)
		// 商品已经不存在时，当前下单请求不应继续执行。
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorsx.ResourceNotFound("商品不存在").WithCause(err)
			}
			return err
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
		var goodsInfo *models.GoodsInfo
		goodsInfo, err = c.FindByID(ctx, goodsID)
		// 商品记录缺失时，当前库存回退已经无法可靠执行。
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errorsx.Internal("商品库存回退失败，商品不存在").WithCause(err)
			}
			return err
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

// buildCategoryFilterIDs 构建分类筛选范围。
func (c *GoodsInfoCase) buildCategoryFilterIDs(ctx context.Context, categoryID int64) ([]int64, error) {
	// 先校验分类存在，避免按无效分类编号继续查询商品。
	_, err := c.goodsCategoryCase.FindByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	var categoryList []*models.GoodsCategory
	categoryList, err = c.goodsCategoryCase.List(ctx)
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
	if rawCategoryIDs == "" {
		return []int64{}
	}

	categoryIDs := make([]int64, 0)
	// 分类 JSON 解析失败时，回退为空列表，避免单条脏数据影响推荐与分类查询。
	if err := json.Unmarshal([]byte(rawCategoryIDs), &categoryIDs); err != nil {
		return []int64{}
	}
	return categoryIDs
}
