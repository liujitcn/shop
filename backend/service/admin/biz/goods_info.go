package biz

import (
	"context"
	"fmt"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

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
func NewGoodsInfoCase(baseCase *biz.BaseCase, tx data.Transaction, goodsInfoRepo *data.GoodsInfoRepo, goodsCategoryCase *GoodsCategoryCase, goodsPropCase *GoodsPropCase, goodsSpecCase *GoodsSpecCase, goodsSkuCase *GoodsSkuCase) *GoodsInfoCase {
	formMapper := _mapper.NewCopierMapper[admin.GoodsInfoForm, models.GoodsInfo]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[admin.GoodsInfo, models.GoodsInfo]()
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

	categoryNames, err := c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.OptionGoodsInfoResponse_GoodsInfo, 0, len(list))
	for _, item := range list {
		resList = append(resList, &admin.OptionGoodsInfoResponse_GoodsInfo{
			Id:           item.ID,
			Name:         item.Name,
			Price:        item.Price,
			CategoryName: categoryNames[item.CategoryID],
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
		category, err := c.goodsCategoryCase.FindById(ctx, req.GetCategoryId())
		if err != nil {
			return nil, err
		}
		// 一级分类需要同时包含全部子分类商品。
		if category.ParentID == 0 {
			categoryQuery := c.goodsCategoryCase.Query(ctx).GoodsCategory
			var categoryList []*models.GoodsCategory
			categoryOpts := make([]repo.QueryOption, 0, 1)
			categoryOpts = append(categoryOpts, repo.Where(categoryQuery.Path.Like(category.Path+"/"+fmt.Sprintf("%d", category.ID)+"%")))
			categoryList, err := c.goodsCategoryCase.List(ctx, categoryOpts...)
			if err != nil {
				return nil, err
			}
			categoryIdList := []int64{category.ID}
			for _, item := range categoryList {
				categoryIdList = append(categoryIdList, item.ID)
			}
			opts = append(opts, repo.Where(query.CategoryID.In(categoryIdList...)))
		} else {
			opts = append(opts, repo.Where(query.CategoryID.Eq(req.GetCategoryId())))
		}
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 指定库存预警时，先筛出符合预警条件的商品集合。
	if req.InventoryAlert != nil {
		goodsIdList, err := c.findGoodsIdsByInventoryAlert(ctx, req.GetInventoryAlert())
		if err != nil {
			return nil, err
		}
		// 预警条件无命中商品时，直接返回空分页结果。
		if len(goodsIdList) == 0 {
			return &admin.PageGoodsInfoResponse{List: []*admin.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(goodsIdList...)))
	}
	// 异常价格预警只在指定预警类型时生效。
	if req.PriceAlert != nil && req.GetPriceAlert() == goodsPriceAlertAbnormal {
		goodsIdList, err := c.findGoodsIdsByAbnormalPrice(ctx)
		if err != nil {
			return nil, err
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

	categoryNames, err := c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsInfo, 0, len(list))
	for _, item := range list {
		goodsInfo := c.mapper.ToDTO(item)
		goodsInfo.CategoryName = categoryNames[item.CategoryID]
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

	category, err := c.goodsCategoryCase.FindById(ctx, goodsInfo.CategoryID)
	// 分类存在时，补齐商品所属分类名称。
	if err == nil {
		goodsForm.CategoryName = category.Name
		// 二级分类需要同时拼接父分类名称。
		if category.ParentID > 0 {
			parentCategory, err := c.goodsCategoryCase.FindById(ctx, category.ParentID)
			// 父分类存在时，返回“父/子”形式的完整分类名称。
			if err == nil {
				goodsForm.CategoryName = fmt.Sprintf("%s/%s", parentCategory.Name, category.Name)
			}
		}
	}

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
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		goodsInfo := c.formMapper.ToEntity(req)
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

		err := c.Create(ctx, goodsInfo)
		if err != nil {
			return err
		}

		err = c.batchCreateGoodsProp(ctx, goodsInfo.ID, req.GetPropList())
		if err != nil {
			return err
		}
		err = c.batchCreateGoodsSpec(ctx, goodsInfo.ID, req.GetSpecList())
		if err != nil {
			return err
		}
		return c.batchCreateGoodsSku(ctx, goodsInfo.ID, skuList)
	})
}

// UpdateGoodsInfo 更新商品
func (c *GoodsInfoCase) UpdateGoodsInfo(ctx context.Context, req *admin.GoodsInfoForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		goodsInfo := c.formMapper.ToEntity(req)
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

		err := c.UpdateById(ctx, goodsInfo)
		if err != nil {
			return err
		}

		err = c.deleteGoodsChildren(ctx, []int64{goodsInfo.ID})
		if err != nil {
			return err
		}
		err = c.batchCreateGoodsProp(ctx, goodsInfo.ID, req.GetPropList())
		if err != nil {
			return err
		}
		err = c.batchCreateGoodsSpec(ctx, goodsInfo.ID, req.GetSpecList())
		if err != nil {
			return err
		}
		return c.batchCreateGoodsSku(ctx, goodsInfo.ID, skuList)
	})
}

// DeleteGoodsInfo 删除商品
func (c *GoodsInfoCase) DeleteGoodsInfo(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.DeleteByIds(ctx, ids)
		if err != nil {
			return err
		}

		// 删除商品后需要同步清理属性、规格和 SKU 等子数据
		return c.deleteGoodsChildren(ctx, ids)
	})
}

// SetGoodsInfoStatus 设置商品状态
func (c *GoodsInfoCase) SetGoodsInfoStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.GoodsInfo{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
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
