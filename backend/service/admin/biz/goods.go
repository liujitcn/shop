package biz

import (
	"context"
	"fmt"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsCase 商品业务实例
type GoodsCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.GoodsRepo
	goodsCategoryCase *GoodsCategoryCase
	goodsPropCase     *GoodsPropCase
	goodsSpecCase     *GoodsSpecCase
	goodsSkuCase      *GoodsSkuCase
	formMapper        *mapper.CopierMapper[admin.GoodsForm, models.Goods]
	mapper            *mapper.CopierMapper[admin.Goods, models.Goods]
}

// NewGoodsCase 创建商品业务实例
func NewGoodsCase(baseCase *biz.BaseCase, tx data.Transaction, goodsRepo *data.GoodsRepo, goodsCategoryCase *GoodsCategoryCase, goodsPropCase *GoodsPropCase, goodsSpecCase *GoodsSpecCase, goodsSkuCase *GoodsSkuCase) *GoodsCase {
	return &GoodsCase{
		BaseCase:          baseCase,
		tx:                tx,
		GoodsRepo:         goodsRepo,
		goodsCategoryCase: goodsCategoryCase,
		goodsPropCase:     goodsPropCase,
		goodsSpecCase:     goodsSpecCase,
		goodsSkuCase:      goodsSkuCase,
		formMapper:        mapper.NewCopierMapper[admin.GoodsForm, models.Goods](),
		mapper:            mapper.NewCopierMapper[admin.Goods, models.Goods](),
	}
}

// ListGoods 查询商品列表
func (c *GoodsCase) ListGoods(ctx context.Context, req *admin.ListGoodsRequest) (*admin.ListGoodsResponse, error) {
	query := c.Query(ctx).Goods
	opts := make([]repo.QueryOption, 0, 1)
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

	resList := make([]*admin.ListGoodsResponse_Goods, 0, len(list))
	for _, item := range list {
		resList = append(resList, &admin.ListGoodsResponse_Goods{
			Id:           item.ID,
			Name:         item.Name,
			Price:        item.Price,
			CategoryName: categoryNames[item.CategoryID],
		})
	}
	return &admin.ListGoodsResponse{List: resList}, nil
}

// PageGoods 分页查询商品
func (c *GoodsCase) PageGoods(ctx context.Context, req *admin.PageGoodsRequest) (*admin.PageGoodsResponse, error) {
	query := c.Query(ctx).Goods
	opts := make([]repo.QueryOption, 0, 3)
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
			categoryList, err = c.goodsCategoryCase.List(ctx, repo.Where(categoryQuery.Path.Like(category.Path+"/"+fmt.Sprintf("%d", category.ID)+"%")))
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

	var list []*models.Goods
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

	resList := make([]*admin.Goods, 0, len(list))
	for _, item := range list {
		goods := c.mapper.ToDTO(item)
		goods.CategoryName = categoryNames[item.CategoryID]
		resList = append(resList, goods)
	}
	return &admin.PageGoodsResponse{List: resList, Total: int32(total)}, nil
}

// GetGoods 获取商品
func (c *GoodsCase) GetGoods(ctx context.Context, id int64) (*admin.GoodsForm, error) {
	goods, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	goodsForm := c.formMapper.ToDTO(goods)
	goodsForm.Banner = _string.ConvertJsonStringToStringArray(goods.Banner)
	goodsForm.Detail = _string.ConvertJsonStringToStringArray(goods.Detail)

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

// CreateGoods 创建商品
func (c *GoodsCase) CreateGoods(ctx context.Context, req *admin.GoodsForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		goods := c.formMapper.ToEntity(req)
		goods.Banner = _string.ConvertStringArrayToString(req.GetBanner())
		goods.Detail = _string.ConvertStringArrayToString(req.GetDetail())
		skuList := req.GetSkuList()
		for idx, sku := range skuList {
			if idx == 0 {
				goods.Price = sku.Price
				goods.DiscountPrice = sku.DiscountPrice
			}
			goods.InitSaleNum += sku.InitSaleNum
			goods.RealSaleNum += sku.RealSaleNum
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

// UpdateGoods 更新商品
func (c *GoodsCase) UpdateGoods(ctx context.Context, req *admin.GoodsForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		goods := c.formMapper.ToEntity(req)
		goods.Banner = _string.ConvertStringArrayToString(req.GetBanner())
		goods.Detail = _string.ConvertStringArrayToString(req.GetDetail())
		skuList := req.GetSkuList()
		for idx, sku := range skuList {
			if idx == 0 {
				goods.Price = sku.Price
				goods.DiscountPrice = sku.DiscountPrice
			}
			goods.InitSaleNum += sku.InitSaleNum
			goods.RealSaleNum += sku.RealSaleNum
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

// DeleteGoods 删除商品
func (c *GoodsCase) DeleteGoods(ctx context.Context, id string) error {
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

// SetGoodsStatus 设置商品状态
func (c *GoodsCase) SetGoodsStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.Goods{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// GoodsPropCaseList 查询商品属性列表
func (c *GoodsCase) GoodsPropCaseList(ctx context.Context, goodsId int64) ([]*admin.GoodsProp, error) {
	res, err := c.goodsPropCase.ListGoodsPropByGoodsId(ctx, goodsId)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GoodsSkuCaseList 查询商品规格项列表
func (c *GoodsCase) GoodsSkuCaseList(ctx context.Context, goodsId int64) ([]*admin.GoodsSku, error) {
	return c.goodsSkuCase.ListGoodsSkuByGoodsId(ctx, goodsId)
}

// deleteGoodsChildren 删除商品子表数据
func (c *GoodsCase) deleteGoodsChildren(ctx context.Context, ids []int64) error {
	for _, goodsId := range ids {
		propQuery := c.goodsPropCase.Query(ctx).GoodsProp
		err := c.goodsPropCase.Delete(ctx, repo.Where(propQuery.GoodsID.Eq(goodsId)))
		if err != nil {
			return err
		}

		specQuery := c.goodsSpecCase.Query(ctx).GoodsSpec
		err = c.goodsSpecCase.Delete(ctx, repo.Where(specQuery.GoodsID.Eq(goodsId)))
		if err != nil {
			return err
		}

		skuQuery := c.goodsSkuCase.Query(ctx).GoodsSku
		err = c.goodsSkuCase.Delete(ctx, repo.Where(skuQuery.GoodsID.Eq(goodsId)))
		if err != nil {
			return err
		}
	}
	return nil
}

// batchCreateGoodsProp 批量创建商品属性
func (c *GoodsCase) batchCreateGoodsProp(ctx context.Context, goodsId int64, list []*admin.GoodsProp) error {
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
func (c *GoodsCase) batchCreateGoodsSpec(ctx context.Context, goodsId int64, list []*admin.GoodsSpec) error {
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
func (c *GoodsCase) batchCreateGoodsSku(ctx context.Context, goodsId int64, list []*admin.GoodsSku) error {
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
func (c *GoodsCase) getCategoryNameMap(ctx context.Context) (map[int64]string, error) {
	categoryList, err := c.goodsCategoryCase.List(ctx)
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
