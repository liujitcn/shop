package biz

import (
	"context"
	"encoding/json"
	"errors"
	"shop/pkg/errorsx"
	"shop/pkg/queue"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-sql-driver/mysql"
	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

// GoodsInfoCase 商品业务实例
type GoodsInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.GoodsInfoRepository
	goodsCategoryCase *GoodsCategoryCase
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSKUCase      *GoodsSKUCase
	formMapper        *_mapper.CopierMapper[adminv1.GoodsInfoForm, models.GoodsInfo]
	mapper            *_mapper.CopierMapper[adminv1.GoodsInfo, models.GoodsInfo]
}

const (
	GOODS_INVENTORY_ALERT_LOW  int32 = 1
	GOODS_INVENTORY_ALERT_ZERO int32 = 2
	GOODS_PRICE_ALERT_ABNORMAL int32 = 1
)

// NewGoodsInfoCase 创建商品业务实例
func NewGoodsInfoCase(baseCase *biz.BaseCase, tx data.Transaction, goodsInfoRepo *data.GoodsInfoRepository, goodsCategoryCase *GoodsCategoryCase, goodsPropCase *GoodsPropCase, goodsSpecCase *GoodsSpecCase, goodsSKUCase *GoodsSKUCase,
) *GoodsInfoCase {
	formMapper := _mapper.NewCopierMapper[adminv1.GoodsInfoForm, models.GoodsInfo]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[adminv1.GoodsInfo, models.GoodsInfo]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	return &GoodsInfoCase{
		BaseCase:            baseCase,
		tx:                  tx,
		GoodsInfoRepository: goodsInfoRepo,
		goodsCategoryCase:   goodsCategoryCase,
		goodsPropCase:       goodsPropCase,
		goodsSpecCase:       goodsSpecCase,
		goodsSKUCase:        goodsSKUCase,
		formMapper:          formMapper,
		mapper:              mapper,
	}
}

// OptionGoodsInfos 查询商品下拉选择
func (c *GoodsInfoCase) OptionGoodsInfos(ctx context.Context, req *adminv1.OptionGoodsInfosRequest) (*adminv1.OptionGoodsInfosResponse, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入商品名称关键字时，按名称模糊过滤。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	categoryNames := make(map[int64]string)
	categoryNames, err = c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.OptionGoodsInfosResponse_GoodsInfo, 0, len(list))
	for _, item := range list {
		resList = append(resList, &adminv1.OptionGoodsInfosResponse_GoodsInfo{
			Id:           item.ID,
			Name:         item.Name,
			Price:        item.Price,
			CategoryName: c.buildCategoryNameText(c.parseCategoryIDs(item.CategoryID), categoryNames),
		})
	}
	return &adminv1.OptionGoodsInfosResponse{GoodsInfos: resList}, nil
}

// PageGoodsInfos 查询商品列表
func (c *GoodsInfoCase) PageGoodsInfos(ctx context.Context, req *adminv1.PageGoodsInfosRequest) (*adminv1.PageGoodsInfosResponse, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 商品名称存在时，按名称模糊过滤。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 指定分类时，按分类层级筛选商品。
	if req.CategoryId != nil && req.GetCategoryId() > 0 {
		categoryIDList, categoryErr := c.buildCategoryFilterIDs(ctx, req.GetCategoryId())
		if categoryErr != nil {
			return nil, categoryErr
		}
		goodsIDList := make([]int64, 0)
		goodsIDList, categoryErr = c.findGoodsIDsByCategoryIDs(ctx, categoryIDList)
		if categoryErr != nil {
			return nil, categoryErr
		}
		// 分类条件无命中商品时，直接返回空分页结果。
		if len(goodsIDList) == 0 {
			return &adminv1.PageGoodsInfosResponse{GoodsInfos: []*adminv1.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(goodsIDList...)))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 指定库存预警时，先筛出符合预警条件的商品集合。
	if req.InventoryAlert != nil {
		goodsIDList, inventoryErr := c.findGoodsIDsByInventoryAlert(ctx, req.GetInventoryAlert())
		if inventoryErr != nil {
			return nil, inventoryErr
		}
		// 预警条件无命中商品时，直接返回空分页结果。
		if len(goodsIDList) == 0 {
			return &adminv1.PageGoodsInfosResponse{GoodsInfos: []*adminv1.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(goodsIDList...)))
	}
	// 异常价格预警只在指定预警类型时生效。
	if req.PriceAlert != nil && req.GetPriceAlert() == GOODS_PRICE_ALERT_ABNORMAL {
		goodsIDList, priceErr := c.findGoodsIDsByAbnormalPrice(ctx)
		if priceErr != nil {
			return nil, priceErr
		}
		// 异常价格条件无命中商品时，直接返回空分页结果。
		if len(goodsIDList) == 0 {
			return &adminv1.PageGoodsInfosResponse{GoodsInfos: []*adminv1.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(goodsIDList...)))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	categoryNames := make(map[int64]string)
	categoryNames, err = c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.GoodsInfo, 0, len(list))
	for _, item := range list {
		goodsInfo := c.mapper.ToDTO(item)
		goodsInfo.CategoryName = c.buildCategoryNameText(c.parseCategoryIDs(item.CategoryID), categoryNames)
		resList = append(resList, goodsInfo)
	}
	return &adminv1.PageGoodsInfosResponse{GoodsInfos: resList, Total: int32(total)}, nil
}

// findGoodsIDsByInventoryAlert 查询命中库存预警的商品标识。
func (c *GoodsInfoCase) findGoodsIDsByInventoryAlert(ctx context.Context, inventoryAlert int32) ([]int64, error) {
	// 非支持的库存预警类型不返回商品集合。
	if inventoryAlert != GOODS_INVENTORY_ALERT_LOW && inventoryAlert != GOODS_INVENTORY_ALERT_ZERO {
		return nil, nil
	}

	availableGoodsIDs, err := c.listNonDeletedGoodsIDs(ctx)
	if err != nil {
		return nil, err
	}
	// 没有可用商品时，库存预警筛选直接返回空结果。
	if len(availableGoodsIDs) == 0 {
		return []int64{}, nil
	}

	query := c.goodsSKUCase.Query(ctx).GoodsSKU
	dao := query.WithContext(ctx).
		Where(
			query.DeletedAt.IsNull(),
			query.GoodsID.In(availableGoodsIDs...),
		)
	// 根据不同预警类型追加库存过滤条件。
	if inventoryAlert == GOODS_INVENTORY_ALERT_LOW {
		dao = dao.Where(
			query.Inventory.Gt(0),
			query.Inventory.Lte(LOW_INVENTORY_THRESHOLD),
		)
	} else {
		dao = dao.Where(query.Inventory.Eq(0))
	}

	goodsIDList := make([]int64, 0)
	err = dao.Distinct(query.GoodsID).Pluck(query.GoodsID, &goodsIDList)
	return goodsIDList, err
}

// findGoodsIDsByAbnormalPrice 查询命中异常价格预警的商品标识。
func (c *GoodsInfoCase) findGoodsIDsByAbnormalPrice(ctx context.Context) ([]int64, error) {
	availableGoodsIDs, err := c.listNonDeletedGoodsIDs(ctx)
	if err != nil {
		return nil, err
	}
	// 没有可用商品时，异常价格筛选直接返回空结果。
	if len(availableGoodsIDs) == 0 {
		return []int64{}, nil
	}

	goodsIDList := make([]int64, 0)
	query := c.goodsSKUCase.Query(ctx).GoodsSKU
	err = query.WithContext(ctx).
		Where(
			query.DeletedAt.IsNull(),
			query.GoodsID.In(availableGoodsIDs...),
			field.Or(
				query.Price.Lte(0),
				query.DiscountPrice.Lt(0),
				field.And(
					query.DiscountPrice.Gt(0),
					query.DiscountPrice.GtCol(query.Price),
				),
			),
		).
		Distinct(query.GoodsID).
		Pluck(query.GoodsID, &goodsIDList)
	return goodsIDList, err
}

// GetGoodsInfo 获取商品
func (c *GoodsInfoCase) GetGoodsInfo(ctx context.Context, id int64) (*adminv1.GoodsInfoForm, error) {
	goodsInfo, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	goodsForm := c.formMapper.ToDTO(goodsInfo)
	categoryNames := make(map[int64]string)
	categoryNames, err = c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}
	goodsForm.CategoryName = c.buildCategoryNameText(c.parseCategoryIDs(goodsInfo.CategoryID), categoryNames)

	goodsForm.PropList, err = c.GoodsPropCaseList(ctx, goodsForm.Id)
	if err != nil {
		return nil, err
	}
	var specList *adminv1.ListGoodsSpecsResponse
	specList, err = c.goodsSpecCase.ListGoodsSpecs(ctx, &adminv1.ListGoodsSpecsRequest{GoodsId: goodsForm.Id})
	if err != nil {
		return nil, err
	}
	goodsForm.SpecList = specList.GetGoodsSpecs()
	goodsForm.SkuList, err = c.GoodsSKUCaseList(ctx, goodsForm.Id)
	if err != nil {
		return nil, err
	}
	return goodsForm, nil
}

// CreateGoodsInfo 创建商品
func (c *GoodsInfoCase) CreateGoodsInfo(ctx context.Context, req *adminv1.GoodsInfoForm) error {
	goodsInfo := c.formMapper.ToEntity(req)
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		skuList := req.GetSkuList()
		for idx, sku := range skuList {
			// 首个 SKU 的价格作为商品主价格展示。
			if idx == 0 {
				goodsInfo.Price = sku.Price
				goodsInfo.DiscountPrice = sku.DiscountPrice
			}
			goodsInfo.InitSaleNum += sku.InitSaleNum
			goodsInfo.RealSaleNum += sku.RealSaleNum
			goodsInfo.Inventory += sku.Inventory
		}

		txErr := c.Create(ctx, goodsInfo)
		if txErr != nil {
			return txErr
		}
		txErr = c.batchCreateGoodsProp(ctx, goodsInfo.ID, req.GetPropList())
		if txErr != nil {
			return txErr
		}
		txErr = c.batchCreateGoodsSpec(ctx, goodsInfo.ID, req.GetSpecList())
		if txErr != nil {
			return txErr
		}
		return c.batchCreateGoodsSKU(ctx, goodsInfo.ID, skuList)
	})
	if err != nil {
		return c.wrapGoodsInfoDuplicateConflict(err)
	}
	// 商品创建成功后，再异步同步最新商品快照到推荐系统。
	queue.DispatchRecommendSyncGoodsInfo(goodsInfo.ID)
	return nil
}

// UpdateGoodsInfo 更新商品
func (c *GoodsInfoCase) UpdateGoodsInfo(ctx context.Context, req *adminv1.GoodsInfoForm) error {
	goodsInfo := c.formMapper.ToEntity(req)
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		skuList := req.GetSkuList()
		for idx, sku := range skuList {
			// 首个 SKU 的价格作为商品主价格展示。
			if idx == 0 {
				goodsInfo.Price = sku.Price
				goodsInfo.DiscountPrice = sku.DiscountPrice
			}
			goodsInfo.InitSaleNum += sku.InitSaleNum
			goodsInfo.RealSaleNum += sku.RealSaleNum
			goodsInfo.Inventory += sku.Inventory
		}

		txErr := c.UpdateByID(ctx, goodsInfo)
		if txErr != nil {
			return txErr
		}

		txErr = c.deleteGoodsChildren(ctx, []int64{goodsInfo.ID})
		if txErr != nil {
			return txErr
		}
		txErr = c.batchCreateGoodsProp(ctx, goodsInfo.ID, req.GetPropList())
		if txErr != nil {
			return txErr
		}
		txErr = c.batchCreateGoodsSpec(ctx, goodsInfo.ID, req.GetSpecList())
		if txErr != nil {
			return txErr
		}
		return c.batchCreateGoodsSKU(ctx, goodsInfo.ID, skuList)
	})
	if err != nil {
		return c.wrapGoodsInfoDuplicateConflict(err)
	}
	// 商品更新成功后，再异步同步最新商品快照到推荐系统。
	queue.DispatchRecommendSyncGoodsInfo(goodsInfo.ID)
	return nil
}

// DeleteGoodsInfo 删除商品
func (c *GoodsInfoCase) DeleteGoodsInfo(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		txErr := c.DeleteByIDs(ctx, ids)
		if txErr != nil {
			return txErr
		}

		// 删除商品后需要同步清理属性、规格和 SKU 等子数据
		return c.deleteGoodsChildren(ctx, ids)
	})
	if err != nil {
		return err
	}
	// 商品删除成功后，再异步清理推荐系统中的商品主体。
	queue.DispatchRecommendDeleteGoodsInfo(ids)
	return nil
}

// SetGoodsInfoStatus 设置商品状态
func (c *GoodsInfoCase) SetGoodsInfoStatus(ctx context.Context, req *adminv1.SetGoodsInfoStatusRequest) error {
	err := c.UpdateByID(ctx, &models.GoodsInfo{
		ID:     req.GetId(),
		Status: int32(req.GetStatus()),
	})
	if err != nil {
		return err
	}
	// 商品状态变更成功后，再异步同步最新状态到推荐系统。
	queue.DispatchRecommendSyncGoodsInfo(req.GetId())
	return nil
}

// GoodsPropCaseList 查询商品属性列表
func (c *GoodsInfoCase) GoodsPropCaseList(ctx context.Context, goodsID int64) ([]*adminv1.GoodsProp, error) {
	res, err := c.goodsPropCase.ListGoodsPropByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GoodsSKUCaseList 查询商品规格项列表
func (c *GoodsInfoCase) GoodsSKUCaseList(ctx context.Context, goodsID int64) ([]*adminv1.GoodsSku, error) {
	return c.goodsSKUCase.ListGoodsSKUsByGoodsID(ctx, goodsID)
}

// wrapGoodsInfoDuplicateConflict 把商品及其子表的唯一索引冲突转换成稳定业务错误。
func (c *GoodsInfoCase) wrapGoodsInfoDuplicateConflict(err error) error {
	// 非唯一索引冲突时，直接透传原始错误给上层统一处理。
	if !errorsx.IsMySQLDuplicateKey(err) {
		return err
	}

	var mysqlErr *mysql.MySQLError
	// 无法提取底层 MySQL 错误明细时，退化为通用唯一约束冲突。
	if !errors.As(err, &mysqlErr) {
		return errorsx.UniqueConflict("商品信息存在重复数据", "goods_info", "", "").WithCause(err)
	}

	// 根据唯一索引落在哪张子表上，返回对应的业务冲突错误。
	switch {
	case strings.Contains(mysqlErr.Message, models.TableNameGoodsProp):
		return errorsx.UniqueConflict("商品属性重复", "goods_prop", "label", "unique_goods_prop").WithCause(err)
	case strings.Contains(mysqlErr.Message, models.TableNameGoodsSKU):
		return errorsx.UniqueConflict("SKU编码重复", "goods_sku", "sku_code", "unique_goods_sku").WithCause(err)
	case strings.Contains(mysqlErr.Message, models.TableNameGoodsSpec):
		return errorsx.UniqueConflict("商品规格重复", "goods_spec", "name", "unique_goods_spec").WithCause(err)
	default:
		return errorsx.UniqueConflict("商品信息存在重复数据", "goods_info", "", "").WithCause(err)
	}
}

// deleteGoodsChildren 删除商品子表数据
func (c *GoodsInfoCase) deleteGoodsChildren(ctx context.Context, ids []int64) error {
	query := c.Query(ctx)
	for _, goodsID := range ids {
		// 商品编辑会按“删旧建新”重建子表数据，这里必须物理删除，否则唯一索引会与软删除数据冲突。
		propQuery := query.GoodsProp
		_, err := propQuery.WithContext(ctx).Unscoped().Where(propQuery.GoodsID.Eq(goodsID)).Delete()
		if err != nil {
			return err
		}

		specQuery := query.GoodsSpec
		_, err = specQuery.WithContext(ctx).Unscoped().Where(specQuery.GoodsID.Eq(goodsID)).Delete()
		if err != nil {
			return err
		}

		skuQuery := query.GoodsSKU
		_, err = skuQuery.WithContext(ctx).Unscoped().Where(skuQuery.GoodsID.Eq(goodsID)).Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

// batchCreateGoodsProp 批量创建商品属性
func (c *GoodsInfoCase) batchCreateGoodsProp(ctx context.Context, goodsID int64, list []*adminv1.GoodsProp) error {
	entities := make([]*models.GoodsProp, 0, len(list))
	for _, item := range list {
		entity := c.goodsPropCase.mapper.ToEntity(item)
		// 商品属性明细为空时，保持旧逻辑的零值入库行为。
		if entity == nil {
			entity = &models.GoodsProp{}
		}
		entity.GoodsID = goodsID
		entities = append(entities, entity)
	}
	return c.goodsPropCase.BatchCreate(ctx, entities)
}

// batchCreateGoodsSpec 批量创建商品规格
func (c *GoodsInfoCase) batchCreateGoodsSpec(ctx context.Context, goodsID int64, list []*adminv1.GoodsSpec) error {
	entities := make([]*models.GoodsSpec, 0, len(list))
	for _, item := range list {
		entity := c.goodsSpecCase.mapper.ToEntity(item)
		// 商品规格明细为空时，保持旧逻辑的零值入库行为。
		if entity == nil {
			entity = &models.GoodsSpec{}
		}
		entity.GoodsID = goodsID
		entities = append(entities, entity)
	}
	return c.goodsSpecCase.BatchCreate(ctx, entities)
}

// batchCreateGoodsSKU 批量创建商品规格项
func (c *GoodsInfoCase) batchCreateGoodsSKU(ctx context.Context, goodsID int64, list []*adminv1.GoodsSku) error {
	entities := make([]*models.GoodsSKU, 0, len(list))
	for _, item := range list {
		entity := c.goodsSKUCase.toGoodsSKUModel(item)
		// 商品 SKU 明细为空时，保持旧逻辑的零值入库行为。
		if entity == nil {
			entity = &models.GoodsSKU{}
		}
		entity.GoodsID = goodsID
		entities = append(entities, entity)
	}
	return c.goodsSKUCase.BatchCreate(ctx, entities)
}

// buildCategoryFilterIDs 构建分类筛选范围。
func (c *GoodsInfoCase) buildCategoryFilterIDs(ctx context.Context, categoryID int64) ([]int64, error) {
	// 先校验分类存在，避免后续按无效分类编号继续查询商品。
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

	categoryIDList := []int64{categoryID}
	queue := []int64{categoryID}
	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		childIDs := childMap[currentID]
		// 当前分类没有子分类时，继续展开下一项待处理分类。
		if len(childIDs) == 0 {
			continue
		}
		categoryIDList = append(categoryIDList, childIDs...)
		queue = append(queue, childIDs...)
	}
	return categoryIDList, nil
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

	goodsIDList := make([]int64, 0)
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
			goodsIDList = append(goodsIDList, row.ID)
		}
	}
	return goodsIDList, nil
}

// listNonDeletedGoodsIDs 查询未删除商品编号。
func (c *GoodsInfoCase) listNonDeletedGoodsIDs(ctx context.Context) ([]int64, error) {
	query := c.Query(ctx).GoodsInfo
	goodsIDs := make([]int64, 0)
	err := query.WithContext(ctx).
		Where(query.DeletedAt.IsNull()).
		Pluck(query.ID, &goodsIDs)
	return goodsIDs, err
}

// parseCategoryIDs 解析商品分类编号列表。
func (c *GoodsInfoCase) parseCategoryIDs(rawCategoryIDs string) []int64 {
	// 分类字段为空时，直接返回空分类列表。
	if strings.TrimSpace(rawCategoryIDs) == "" {
		return []int64{}
	}

	categoryIDs := make([]int64, 0)
	// 分类 JSON 解析失败时，回退为空列表，避免后台列表因单条脏数据整体失败。
	if err := json.Unmarshal([]byte(rawCategoryIDs), &categoryIDs); err != nil {
		return []int64{}
	}
	return categoryIDs
}

// buildCategoryNameText 构建商品分类展示文本。
func (c *GoodsInfoCase) buildCategoryNameText(categoryIDs []int64, categoryNameMap map[int64]string) string {
	// 商品没有配置分类时，直接返回空展示文本。
	if len(categoryIDs) == 0 {
		return ""
	}

	nameList := make([]string, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		// 分类名称存在时，按商品配置顺序拼接展示文本。
		if name, ok := categoryNameMap[categoryID]; ok && name != "" {
			nameList = append(nameList, name)
		}
	}
	return strings.Join(nameList, "、")
}

// getCategoryNameMap 查询分类名称映射
func (c *GoodsInfoCase) getCategoryNameMap(ctx context.Context) (map[int64]string, error) {
	query := c.goodsCategoryCase.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	categoryList, err := c.goodsCategoryCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	nameMap := make(map[int64]string, len(categoryList))
	parentMap := make(map[int64]int64, len(categoryList))
	for _, item := range categoryList {
		nameMap[item.ID] = item.Name
		parentMap[item.ID] = item.ParentID
	}

	res := make(map[int64]string, len(categoryList))
	for id, name := range nameMap {
		parentID := parentMap[id]
		// 存在父分类时，优先拼接父子层级名称。
		if parentID > 0 {
			// 父分类名称存在时，返回完整的“父/子”展示名称。
			if parentName, ok := nameMap[parentID]; ok {
				res[id] = parentName + "/" + name
				continue
			}
		}
		res[id] = name
	}
	return res, nil
}
