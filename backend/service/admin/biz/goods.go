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
func NewGoodsInfoCase(baseCase *biz.BaseCase, tx data.Transaction, goodsRepo *data.GoodsInfoRepo, goodsCategoryCase *GoodsCategoryCase, goodsPropCase *GoodsPropCase, goodsSpecCase *GoodsSpecCase, goodsSkuCase *GoodsSkuCase) *GoodsInfoCase {
	formMapper := _mapper.NewCopierMapper[admin.GoodsInfoForm, models.GoodsInfo]()
	formMapper.AppendConverters(_mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	mapper := _mapper.NewCopierMapper[admin.GoodsInfo, models.GoodsInfo]()
	return &GoodsInfoCase{
		BaseCase:          baseCase,
		tx:                tx,
		GoodsInfoRepo:     goodsRepo,
		goodsCategoryCase: goodsCategoryCase,
		goodsPropCase:     goodsPropCase,
		goodsSpecCase:     goodsSpecCase,
		goodsSkuCase:      goodsSkuCase,
		formMapper:        formMapper,
		mapper:            mapper,
	}
}

// ListGoodsInfo 查询商品列表
func (c *GoodsInfoCase) ListGoodsInfo(ctx context.Context, req *admin.ListGoodsInfoRequest) (*admin.ListGoodsInfoResponse, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var categoryNames map[int64]string
	categoryNames, err = c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.ListGoodsInfoResponse_GoodsInfo, 0, len(list))
	for _, item := range list {
		resList = append(resList, &admin.ListGoodsInfoResponse_GoodsInfo{
			Id:           item.ID,
			Name:         item.Name,
			Price:        item.Price,
			CategoryName: categoryNames[item.CategoryID],
		})
	}
	return &admin.ListGoodsInfoResponse{List: resList}, nil
}

// PageGoodsInfo 分页查询商品
func (c *GoodsInfoCase) PageGoodsInfo(ctx context.Context, req *admin.PageGoodsInfoRequest) (*admin.PageGoodsInfoResponse, error) {
	query := c.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Order(query.UpdatedAt.Desc()))
	var err error
	if req.GetName() != "" {
		opts = append(opts, repo.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.CategoryId != nil && req.GetCategoryId() > 0 {
		var category *models.GoodsCategory
		category, err = c.goodsCategoryCase.FindById(ctx, req.GetCategoryId())
		if err != nil {
			return nil, err
		}
		if category.ParentID == 0 {
			categoryQuery := c.goodsCategoryCase.Query(ctx).GoodsCategory
			var categoryList []*models.GoodsCategory
			categoryOpts := make([]repo.QueryOption, 0, 1)
			categoryOpts = append(categoryOpts, repo.Where(categoryQuery.Path.Like(category.Path+"/"+fmt.Sprintf("%d", category.ID)+"%")))
			categoryList, err = c.goodsCategoryCase.List(ctx, categoryOpts...)
			if err != nil {
				return nil, err
			}
			categoryIds := []int64{category.ID}
			for _, item := range categoryList {
				categoryIds = append(categoryIds, item.ID)
			}
			opts = append(opts, repo.Where(query.CategoryID.In(categoryIds...)))
		} else {
			opts = append(opts, repo.Where(query.CategoryID.Eq(req.GetCategoryId())))
		}
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.InventoryAlert != nil {
		var goodsIDs []int64
		goodsIDs, err = c.findGoodsIDsByInventoryAlert(ctx, req.GetInventoryAlert())
		if err != nil {
			return nil, err
		}
		if len(goodsIDs) == 0 {
			return &admin.PageGoodsInfoResponse{List: []*admin.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(goodsIDs...)))
	}
	if req.PriceAlert != nil && req.GetPriceAlert() == goodsPriceAlertAbnormal {
		var goodsIDs []int64
		goodsIDs, err = c.findGoodsIDsByAbnormalPrice(ctx)
		if err != nil {
			return nil, err
		}
		if len(goodsIDs) == 0 {
			return &admin.PageGoodsInfoResponse{List: []*admin.GoodsInfo{}, Total: 0}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(goodsIDs...)))
	}

	var list []*models.GoodsInfo
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	var categoryNames map[int64]string
	categoryNames, err = c.getCategoryNameMap(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.GoodsInfo, 0, len(list))
	for _, item := range list {
		goods := c.mapper.ToDTO(item)
		goods.CategoryName = categoryNames[item.CategoryID]
		resList = append(resList, goods)
	}
	return &admin.PageGoodsInfoResponse{List: resList, Total: int32(total)}, nil
}

func (c *GoodsInfoCase) findGoodsIDsByInventoryAlert(ctx context.Context, inventoryAlert int32) ([]int64, error) {
	if inventoryAlert != goodsInventoryAlertLow && inventoryAlert != goodsInventoryAlertZero {
		return nil, nil
	}

	db := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Select("DISTINCT goods_sku.goods_id").
		Joins("JOIN " + models.TableNameGoodsInfo + " ON " + models.TableNameGoodsInfo + ".id = goods_sku.goods_id").
		Where(models.TableNameGoodsInfo + ".deleted_at IS NULL").
		Where("goods_sku.deleted_at IS NULL")

	if inventoryAlert == goodsInventoryAlertLow {
		db = db.Where("goods_sku.inventory > 0 AND goods_sku.inventory <= ?", lowInventoryThreshold)
	} else {
		db = db.Where("goods_sku.inventory = 0")
	}

	goodsIDs := make([]int64, 0)
	err := db.Pluck("goods_sku.goods_id", &goodsIDs).Error
	return goodsIDs, err
}

func (c *GoodsInfoCase) findGoodsIDsByAbnormalPrice(ctx context.Context) ([]int64, error) {
	goodsIDs := make([]int64, 0)
	err := c.goodsSkuCase.Query(ctx).GoodsSku.WithContext(ctx).UnderlyingDB().
		Model(&models.GoodsSku{}).
		Select("DISTINCT goods_sku.goods_id").
		Joins("JOIN "+models.TableNameGoodsInfo+" ON "+models.TableNameGoodsInfo+".id = goods_sku.goods_id").
		Where(models.TableNameGoodsInfo+".deleted_at IS NULL").
		Where("goods_sku.deleted_at IS NULL").
		Where("goods_sku.price <= 0 OR goods_sku.discount_price < 0 OR (goods_sku.discount_price > 0 AND goods_sku.discount_price > goods_sku.price)").
		Pluck("goods_sku.goods_id", &goodsIDs).Error
	return goodsIDs, err
}

// GetGoodsInfo 获取商品
func (c *GoodsInfoCase) GetGoodsInfo(ctx context.Context, id int64) (*admin.GoodsInfoForm, error) {
	goods, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	goodsForm := c.formMapper.ToDTO(goods)

	var category *models.GoodsCategory
	category, err = c.goodsCategoryCase.FindById(ctx, goods.CategoryID)
	if err == nil {
		goodsForm.CategoryName = category.Name
		if category.ParentID > 0 {
			var parentCategory *models.GoodsCategory
			parentCategory, err = c.goodsCategoryCase.FindById(ctx, category.ParentID)
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
		goods := c.formMapper.ToEntity(req)
		skuList := req.GetSkuList()
		for idx, sku := range skuList {
			if idx == 0 {
				goods.Price = sku.Price
				goods.DiscountPrice = sku.DiscountPrice
			}
			goods.InitSaleNum += sku.InitSaleNum
			goods.RealSaleNum += sku.RealSaleNum
			goods.Inventory += sku.Inventory
		}

		err := c.Create(ctx, goods)
		if err != nil {
			return err
		}

		err = c.batchCreateGoodsProp(ctx, goods.ID, req.GetPropList())
		if err != nil {
			return err
		}
		err = c.batchCreateGoodsSpec(ctx, goods.ID, req.GetSpecList())
		if err != nil {
			return err
		}
		return c.batchCreateGoodsSku(ctx, goods.ID, skuList)
	})
}

// UpdateGoodsInfo 更新商品
func (c *GoodsInfoCase) UpdateGoodsInfo(ctx context.Context, req *admin.GoodsInfoForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		goods := c.formMapper.ToEntity(req)
		skuList := req.GetSkuList()
		for idx, sku := range skuList {
			if idx == 0 {
				goods.Price = sku.Price
				goods.DiscountPrice = sku.DiscountPrice
			}
			goods.InitSaleNum += sku.InitSaleNum
			goods.RealSaleNum += sku.RealSaleNum
			goods.Inventory += sku.Inventory
		}

		err := c.UpdateById(ctx, goods)
		if err != nil {
			return err
		}

		err = c.deleteGoodsChildren(ctx, []int64{goods.ID})
		if err != nil {
			return err
		}
		err = c.batchCreateGoodsProp(ctx, goods.ID, req.GetPropList())
		if err != nil {
			return err
		}
		err = c.batchCreateGoodsSpec(ctx, goods.ID, req.GetSpecList())
		if err != nil {
			return err
		}
		return c.batchCreateGoodsSku(ctx, goods.ID, skuList)
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
	categoryQuery := c.goodsCategoryCase.Query(ctx).GoodsCategory
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(categoryQuery.Sort.Asc()))
	opts = append(opts, repo.Order(categoryQuery.UpdatedAt.Desc()))
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
		if parentId > 0 {
			if parentName, ok := nameMap[parentId]; ok {
				res[id] = parentName + "/" + name
				continue
			}
		}
		res[id] = name
	}
	return res, nil
}
