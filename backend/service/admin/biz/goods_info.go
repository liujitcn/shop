package biz

import (
	"context"
	"encoding/json"
	"errors"
	"shop/pkg/errorsx"
	pkgQueue "shop/pkg/queue"
	"strings"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-sql-driver/mysql"
	_mapper "github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsInfoCase 商品业务实例
type GoodsInfoCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.GoodsInfoRepo
	goodsCategoryCase *GoodsCategoryCase
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSkuCase      *GoodsSkuCase
	formMapper        *_mapper.CopierMapper[admin.GoodsInfoForm, models.GoodsInfo]
	mapper            *_mapper.CopierMapper[admin.GoodsInfo, models.GoodsInfo]
}

const (
	goodsInventoryAlertLow  int32 = 1
	goodsInventoryAlertZero int32 = 2
	goodsPriceAlertAbnormal int32 = 1
)

// NewGoodsInfoCase 创建商品业务实例
func NewGoodsInfoCase(baseCase *biz.BaseCase, tx data.Transaction, goodsInfoRepo *data.GoodsInfoRepo, goodsCategoryCase *GoodsCategoryCase, goodsPropCase *GoodsPropCase, goodsSpecCase *GoodsSpecCase, goodsSkuCase *GoodsSkuCase,
) *GoodsInfoCase {
	formMapper := _mapper.NewCopierMapper[admin.GoodsInfoForm, models.GoodsInfo]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[admin.GoodsInfo, models.GoodsInfo]()
	mapper.AppendConverters(_mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	return &GoodsInfoCase{
		BaseCase:          baseCase,
		tx:                tx,
		GoodsInfoRepo:     goodsInfoRepo,
		goodsCategoryCase: goodsCategoryCase,
		goodsPropCase:     goodsPropCase,
		goodsSpecCase:     goodsSpecCase,
		goodsSkuCase:      goodsSkuCase,
		formMapper:        formMapper,
		mapper:            mapper,
	}
}

// OptionGoodsInfo 查询商品下拉选择
func (c *GoodsInfoCase) OptionGoodsInfo(ctx context.Context, req *admin.OptionGoodsInfoRequest) (*admin.OptionGoodsInfoResponse, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 传入商品名称关键字时，按名称模糊过滤。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
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

	resList := make([]*admin.OptionGoodsInfoResponse_GoodsInfo, 0, len(list))
	for _, item := range list {
		resList = append(resList, &admin.OptionGoodsInfoResponse_GoodsInfo{
			Id:           item.ID,
			Name:         item.Name,
			Price:        item.Price,
			CategoryName: c.buildCategoryNameText(c.parseCategoryIds(item.CategoryID), categoryNames),
		})
	}
	return &admin.OptionGoodsInfoResponse{List: resList}, nil
}

// PageGoodsInfo 分页查询商品
func (c *GoodsInfoCase) PageGoodsInfo(ctx context.Context, req *admin.PageGoodsInfoRequest) (*admin.PageGoodsInfoResponse, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 商品名称存在时，按名称模糊过滤。
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 指定分类时，按分类层级筛选商品。
	if req.CategoryId != nil && req.GetCategoryId() > 0 {
		categoryIdList, categoryErr := c.buildCategoryFilterIds(ctx, req.GetCategoryId())
		if categoryErr != nil {
			return nil, categoryErr
		}
		goodsIdList := make([]int64, 0)
		goodsIdList, categoryErr = c.findGoodsIdsByCategoryIds(ctx, categoryIdList)
		if categoryErr != nil {
			return nil, categoryErr
		}
		// 分类条件无命中商品时，直接返回空分页结果。
		if len(goodsIdList) == 0 {
			return &admin.PageGoodsInfoResponse{List: []*admin.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(goodsIdList...)))
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 指定库存预警时，先筛出符合预警条件的商品集合。
	if req.InventoryAlert != nil {
		goodsIdList, inventoryErr := c.findGoodsIdsByInventoryAlert(ctx, req.GetInventoryAlert())
		if inventoryErr != nil {
			return nil, inventoryErr
		}
		// 预警条件无命中商品时，直接返回空分页结果。
		if len(goodsIdList) == 0 {
			return &admin.PageGoodsInfoResponse{List: []*admin.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(goodsIdList...)))
	}
	// 异常价格预警只在指定预警类型时生效。
	if req.PriceAlert != nil && req.GetPriceAlert() == goodsPriceAlertAbnormal {
		goodsIdList, priceErr := c.findGoodsIdsByAbnormalPrice(ctx)
		if priceErr != nil {
			return nil, priceErr
		}
		// 异常价格条件无命中商品时，直接返回空分页结果。
		if len(goodsIdList) == 0 {
			return &admin.PageGoodsInfoResponse{List: []*admin.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(goodsIdList...)))
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

	resList := make([]*admin.GoodsInfo, 0, len(list))
	for _, item := range list {
		goodsInfo := c.mapper.ToDTO(item)
		goodsInfo.CategoryName = c.buildCategoryNameText(c.parseCategoryIds(item.CategoryID), categoryNames)
		resList = append(resList, goodsInfo)
	}
	return &admin.PageGoodsInfoResponse{List: resList, Total: int32(total)}, nil
}

// findGoodsIdsByInventoryAlert 查询命中库存预警的商品标识。
func (c *GoodsInfoCase) findGoodsIdsByInventoryAlert(ctx context.Context, inventoryAlert int32) ([]int64, error) {
	// 非支持的库存预警类型不返回商品集合。
	if inventoryAlert != goodsInventoryAlertLow && inventoryAlert != goodsInventoryAlertZero {
		return nil, nil
	}

	db := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Select("DISTINCT goods_sku.goods_id").
		Joins("JOIN " + models.TableNameGoodsInfo + " ON " + models.TableNameGoodsInfo + ".id = goods_sku.goods_id").
		Where(models.TableNameGoodsInfo + ".deleted_at IS NULL").
		Where("goods_sku.deleted_at IS NULL")

	// 根据不同预警类型追加库存过滤条件。
	if inventoryAlert == goodsInventoryAlertLow {
		db = db.Where("goods_sku.inventory > 0 AND goods_sku.inventory <= ?", lowInventoryThreshold)
	} else {
		db = db.Where("goods_sku.inventory = 0")
	}

	goodsIdList := make([]int64, 0)
	err := db.Pluck("goods_sku.goods_id", &goodsIdList).Error
	return goodsIdList, err
}

// findGoodsIdsByAbnormalPrice 查询命中异常价格预警的商品标识。
func (c *GoodsInfoCase) findGoodsIdsByAbnormalPrice(ctx context.Context) ([]int64, error) {
	goodsIdList := make([]int64, 0)
	err := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Select("DISTINCT goods_sku.goods_id").
		Joins("JOIN "+models.TableNameGoodsInfo+" ON "+models.TableNameGoodsInfo+".id = goods_sku.goods_id").
		Where(models.TableNameGoodsInfo+".deleted_at IS NULL").
		Where("goods_sku.deleted_at IS NULL").
		Where("goods_sku.price <= 0 OR goods_sku.discount_price < 0 OR (goods_sku.discount_price > 0 AND goods_sku.discount_price > goods_sku.price)").
		Pluck("goods_sku.goods_id", &goodsIdList).Error
	return goodsIdList, err
}

// GetGoodsInfo 获取商品
func (c *GoodsInfoCase) GetGoodsInfo(ctx context.Context, id int64) (*admin.GoodsInfoForm, error) {
	goodsInfo, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	goodsForm := c.formMapper.ToDTO(goodsInfo)
	categoryNames := make(map[int64]string)
	categoryNames, err = c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}
	goodsForm.CategoryName = c.buildCategoryNameText(c.parseCategoryIds(goodsInfo.CategoryID), categoryNames)

	goodsForm.PropList, err = c.GoodsPropCaseList(ctx, goodsForm.Id)
	if err != nil {
		return nil, err
	}
	var specList *admin.ListGoodsSpecResponse
	specList, err = c.goodsSpecCase.ListGoodsSpec(ctx, &admin.ListGoodsSpecRequest{GoodsId: goodsForm.Id})
	if err != nil {
		return nil, err
	}
	goodsForm.SpecList = specList.GetList()
	goodsForm.SkuList, err = c.GoodsSkuCaseList(ctx, goodsForm.Id)
	if err != nil {
		return nil, err
	}
	return goodsForm, nil
}

// CreateGoodsInfo 创建商品
func (c *GoodsInfoCase) CreateGoodsInfo(ctx context.Context, req *admin.GoodsInfoForm) error {
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
		return c.batchCreateGoodsSku(ctx, goodsInfo.ID, skuList)
	})
	if err != nil {
		return c.wrapGoodsInfoDuplicateConflict(err)
	}
	// 商品创建成功后，再异步同步最新商品快照到推荐系统。
	pkgQueue.DispatchRecommendSyncGoodsInfo(goodsInfo.ID)
	return nil
}

// UpdateGoodsInfo 更新商品
func (c *GoodsInfoCase) UpdateGoodsInfo(ctx context.Context, req *admin.GoodsInfoForm) error {
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

		txErr := c.UpdateById(ctx, goodsInfo)
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
		return c.batchCreateGoodsSku(ctx, goodsInfo.ID, skuList)
	})
	if err != nil {
		return c.wrapGoodsInfoDuplicateConflict(err)
	}
	// 商品更新成功后，再异步同步最新商品快照到推荐系统。
	pkgQueue.DispatchRecommendSyncGoodsInfo(goodsInfo.ID)
	return nil
}

// DeleteGoodsInfo 删除商品
func (c *GoodsInfoCase) DeleteGoodsInfo(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	err := c.tx.Transaction(ctx, func(ctx context.Context) error {
		txErr := c.DeleteByIds(ctx, ids)
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
	pkgQueue.DispatchRecommendDeleteGoodsInfo(ids)
	return nil
}

// SetGoodsInfoStatus 设置商品状态
func (c *GoodsInfoCase) SetGoodsInfoStatus(ctx context.Context, req *common.SetStatusRequest) error {
	err := c.UpdateById(ctx, &models.GoodsInfo{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
	if err != nil {
		return err
	}
	// 商品状态变更成功后，再异步同步最新状态到推荐系统。
	pkgQueue.DispatchRecommendSyncGoodsInfo(req.GetId())
	return nil
}

// GoodsPropCaseList 查询商品属性列表
func (c *GoodsInfoCase) GoodsPropCaseList(ctx context.Context, goodsId int64) ([]*admin.GoodsProp, error) {
	res, err := c.goodsPropCase.ListGoodsPropByGoodsId(ctx, goodsId)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GoodsSkuCaseList 查询商品规格项列表
func (c *GoodsInfoCase) GoodsSkuCaseList(ctx context.Context, goodsId int64) ([]*admin.GoodsSku, error) {
	return c.goodsSkuCase.ListGoodsSkuByGoodsId(ctx, goodsId)
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
	case strings.Contains(mysqlErr.Message, models.TableNameGoodsSku):
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
	for _, goodsId := range ids {
		// 商品编辑会按“删旧建新”重建子表数据，这里必须物理删除，否则唯一索引会与软删除数据冲突。
		propQuery := query.GoodsProp
		_, err := propQuery.WithContext(ctx).Unscoped().Where(propQuery.GoodsID.Eq(goodsId)).Delete()
		if err != nil {
			return err
		}

		specQuery := query.GoodsSpec
		_, err = specQuery.WithContext(ctx).Unscoped().Where(specQuery.GoodsID.Eq(goodsId)).Delete()
		if err != nil {
			return err
		}

		skuQuery := query.GoodsSku
		_, err = skuQuery.WithContext(ctx).Unscoped().Where(skuQuery.GoodsID.Eq(goodsId)).Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

// batchCreateGoodsProp 批量创建商品属性
func (c *GoodsInfoCase) batchCreateGoodsProp(ctx context.Context, goodsId int64, list []*admin.GoodsProp) error {
	entities := make([]*models.GoodsProp, 0, len(list))
	for _, item := range list {
		entities = append(entities, &models.GoodsProp{
			GoodsID: goodsId,
			Label:   item.GetLabel(),
			Value:   item.GetValue(),
			Sort:    item.GetSort(),
		})
	}
	return c.goodsPropCase.BatchCreate(ctx, entities)
}

// batchCreateGoodsSpec 批量创建商品规格
func (c *GoodsInfoCase) batchCreateGoodsSpec(ctx context.Context, goodsId int64, list []*admin.GoodsSpec) error {
	entities := make([]*models.GoodsSpec, 0, len(list))
	for _, item := range list {
		entities = append(entities, &models.GoodsSpec{
			GoodsID: goodsId,
			Name:    item.GetName(),
			Item:    _string.ConvertStringArrayToString(item.GetItem()),
			Sort:    item.GetSort(),
		})
	}
	return c.goodsSpecCase.BatchCreate(ctx, entities)
}

// batchCreateGoodsSku 批量创建商品规格项
func (c *GoodsInfoCase) batchCreateGoodsSku(ctx context.Context, goodsId int64, list []*admin.GoodsSku) error {
	entities := make([]*models.GoodsSku, 0, len(list))
	for _, item := range list {
		entities = append(entities, &models.GoodsSku{
			GoodsID:       goodsId,
			Picture:       item.GetPicture(),
			SpecItem:      _string.ConvertStringArrayToString(item.GetSpecItem()),
			SkuCode:       item.GetSkuCode(),
			Price:         item.GetPrice(),
			DiscountPrice: item.GetDiscountPrice(),
			InitSaleNum:   item.GetInitSaleNum(),
			RealSaleNum:   item.GetRealSaleNum(),
			Inventory:     item.GetInventory(),
		})
	}
	return c.goodsSkuCase.BatchCreate(ctx, entities)
}

// buildCategoryFilterIds 构建分类筛选范围。
func (c *GoodsInfoCase) buildCategoryFilterIds(ctx context.Context, categoryId int64) ([]int64, error) {
	// 先校验分类存在，避免后续按无效分类编号继续查询商品。
	_, err := c.goodsCategoryCase.FindById(ctx, categoryId)
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

	categoryIdList := []int64{categoryId}
	queue := []int64{categoryId}
	for len(queue) > 0 {
		currentId := queue[0]
		queue = queue[1:]

		childIds := childMap[currentId]
		// 当前分类没有子分类时，继续展开下一项待处理分类。
		if len(childIds) == 0 {
			continue
		}
		categoryIdList = append(categoryIdList, childIds...)
		queue = append(queue, childIds...)
	}
	return categoryIdList, nil
}

// findGoodsIdsByCategoryIds 查询命中分类集合的商品编号。
func (c *GoodsInfoCase) findGoodsIdsByCategoryIds(ctx context.Context, categoryIds []int64) ([]int64, error) {
	// 没有分类编号时，不需要继续访问数据库查询商品。
	if len(categoryIds) == 0 {
		return []int64{}, nil
	}

	categoryIdsJSON, err := json.Marshal(categoryIds)
	if err != nil {
		return nil, err
	}

	goodsIdList := make([]int64, 0)
	err = c.Query(ctx).GoodsInfo.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsInfo{}).
		Where(models.TableNameGoodsInfo+".deleted_at IS NULL").
		Where("JSON_OVERLAPS("+models.TableNameGoodsInfo+".category_id, CAST(? AS JSON))", string(categoryIdsJSON)).
		Pluck(models.TableNameGoodsInfo+".id", &goodsIdList).Error
	return goodsIdList, err
}

// parseCategoryIds 解析商品分类编号列表。
func (c *GoodsInfoCase) parseCategoryIds(rawCategoryIds string) []int64 {
	// 分类字段为空时，直接返回空分类列表。
	if strings.TrimSpace(rawCategoryIds) == "" {
		return []int64{}
	}

	categoryIds := make([]int64, 0)
	// 分类 JSON 解析失败时，回退为空列表，避免后台列表因单条脏数据整体失败。
	if err := json.Unmarshal([]byte(rawCategoryIds), &categoryIds); err != nil {
		return []int64{}
	}
	return categoryIds
}

// buildCategoryNameText 构建商品分类展示文本。
func (c *GoodsInfoCase) buildCategoryNameText(categoryIds []int64, categoryNameMap map[int64]string) string {
	// 商品没有配置分类时，直接返回空展示文本。
	if len(categoryIds) == 0 {
		return ""
	}

	nameList := make([]string, 0, len(categoryIds))
	for _, categoryId := range categoryIds {
		// 分类名称存在时，按商品配置顺序拼接展示文本。
		if name, ok := categoryNameMap[categoryId]; ok && name != "" {
			nameList = append(nameList, name)
		}
	}
	return strings.Join(nameList, "、")
}

// getCategoryNameMap 查询分类名称映射
func (c *GoodsInfoCase) getCategoryNameMap(ctx context.Context) (map[int64]string, error) {
	query := c.goodsCategoryCase.Query(ctx).GoodsCategory
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
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
		parentId := parentMap[id]
		// 存在父分类时，优先拼接父子层级名称。
		if parentId > 0 {
			// 父分类名称存在时，返回完整的“父/子”展示名称。
			if parentName, ok := nameMap[parentId]; ok {
				res[id] = parentName + "/" + name
				continue
			}
		}
		res[id] = name
	}
	return res, nil
}
