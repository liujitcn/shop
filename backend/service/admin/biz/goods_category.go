package biz

import (
	"context"
	"strconv"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// GoodsCategoryCase 商品分类业务实例
type GoodsCategoryCase struct {
	*biz.BaseCase
	*data.GoodsCategoryRepository
	formMapper *mapper.CopierMapper[adminv1.GoodsCategoryForm, models.GoodsCategory]
	mapper     *mapper.CopierMapper[adminv1.GoodsCategory, models.GoodsCategory]
}

// NewGoodsCategoryCase 创建商品分类业务实例
func NewGoodsCategoryCase(baseCase *biz.BaseCase, goodsCategoryRepo *data.GoodsCategoryRepository) *GoodsCategoryCase {
	return &GoodsCategoryCase{
		BaseCase:                baseCase,
		GoodsCategoryRepository: goodsCategoryRepo,
		formMapper:              mapper.NewCopierMapper[adminv1.GoodsCategoryForm, models.GoodsCategory](),
		mapper:                  mapper.NewCopierMapper[adminv1.GoodsCategory, models.GoodsCategory](),
	}
}

// TreeGoodsCategories 查询分类树
func (c *GoodsCategoryCase) TreeGoodsCategories(ctx context.Context, _ *adminv1.TreeGoodsCategoriesRequest) (*adminv1.TreeGoodsCategoriesResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &adminv1.TreeGoodsCategoriesResponse{
		GoodsCategories: c.buildTree(list, 0),
	}, nil
}

// OptionGoodsCategories 查询分类选项
func (c *GoodsCategoryCase) OptionGoodsCategories(ctx context.Context, req *adminv1.OptionGoodsCategoriesRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &commonv1.TreeOptionResponse{
		List: c.buildOption(list, 0, req.ParentId == nil),
	}, nil
}

// GetGoodsCategory 获取分类
func (c *GoodsCategoryCase) GetGoodsCategory(ctx context.Context, id int64) (*adminv1.GoodsCategoryForm, error) {
	goodsCategory, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(goodsCategory)
	return res, nil
}

// CreateGoodsCategory 创建分类
func (c *GoodsCategoryCase) CreateGoodsCategory(ctx context.Context, req *adminv1.GoodsCategoryForm) error {
	goodsCategory := c.formMapper.ToEntity(req)
	// 根分类直接挂在虚拟根节点下。
	if goodsCategory.ParentID == 0 {
		goodsCategory.Path = "0"
	} else {
		parentGoodsCategory, err := c.FindByID(ctx, goodsCategory.ParentID)
		if err != nil {
			return err
		}
		goodsCategory.Path = parentGoodsCategory.Path + "/" + strconv.FormatInt(parentGoodsCategory.ID, 10)
	}
	return c.Create(ctx, goodsCategory)
}

// UpdateGoodsCategory 更新分类
func (c *GoodsCategoryCase) UpdateGoodsCategory(ctx context.Context, req *adminv1.GoodsCategoryForm) error {
	goodsCategory, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}

	goodsCategory.ParentID = req.GetParentId()
	goodsCategory.Picture = req.GetPicture()
	goodsCategory.Name = req.GetName()
	goodsCategory.Sort = req.GetSort()
	goodsCategory.Status = int32(req.GetStatus())
	// 根分类直接挂在虚拟根节点下。
	if goodsCategory.ParentID == 0 {
		goodsCategory.Path = "0"
	} else {
		var parentGoodsCategory *models.GoodsCategory
		parentGoodsCategory, err = c.FindByID(ctx, goodsCategory.ParentID)
		if err != nil {
			return err
		}
		goodsCategory.Path = parentGoodsCategory.Path + "/" + strconv.FormatInt(parentGoodsCategory.ID, 10)
	}
	return c.UpdateByID(ctx, goodsCategory)
}

// DeleteGoodsCategory 删除分类
func (c *GoodsCategoryCase) DeleteGoodsCategory(ctx context.Context, id string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(id))
}

// SetGoodsCategoryStatus 设置分类状态
func (c *GoodsCategoryCase) SetGoodsCategoryStatus(ctx context.Context, req *adminv1.SetGoodsCategoryStatusRequest) error {
	return c.UpdateByID(ctx, &models.GoodsCategory{
		ID:     req.GetId(),
		Status: int32(req.GetStatus()),
	})
}

// NameMap 查询分类路径名称映射
func (c *GoodsCategoryCase) NameMap(ctx context.Context, parentID *int64) map[int64]string {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 3)
	// 指定父分类时，仅返回该父分类下的直接子分类。
	if parentID != nil {
		opts = append(opts, repository.Where(query.ParentID.Eq(*parentID)))
	}
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))

	categoryList, err := c.List(ctx, opts...)
	if err != nil {
		return map[int64]string{}
	}

	categoryNameMap := make(map[int64]string, len(categoryList))
	categoryPathMap := make(map[int64]string, len(categoryList))
	for _, category := range categoryList {
		categoryNameMap[category.ID] = category.Name
		categoryPathMap[category.ID] = category.Path
	}

	for categoryID, path := range categoryPathMap {
		paths := strings.Split(path, "/")
		pathName := make([]string, 0, len(paths))
		for _, item := range paths {
			var value int64
			value, err = strconv.ParseInt(item, 10, 64)
			// 非法路径片段直接跳过，避免影响剩余路径解析。
			if err != nil {
				continue
			}
			// 命中分类名称时，按路径顺序拼接展示名称。
			if name, ok := categoryNameMap[value]; ok {
				pathName = append(pathName, name)
			}
		}
		categoryPathMap[categoryID] = strings.Join(pathName, "/")
	}
	return categoryPathMap
}

// buildTree 构建分类树
func (c *GoodsCategoryCase) buildTree(categoryList []*models.GoodsCategory, parentID int64) []*adminv1.GoodsCategory {
	res := make([]*adminv1.GoodsCategory, 0)
	for _, item := range categoryList {
		// 仅处理当前父节点下的直接子分类。
		if item.ParentID != parentID {
			continue
		}
		category := c.mapper.ToDTO(item)
		category.CreatedAt = _time.TimeToTimeString(item.CreatedAt)
		category.UpdatedAt = _time.TimeToTimeString(item.UpdatedAt)
		category.Children = c.buildTree(categoryList, item.ID)
		res = append(res, category)
	}
	return res
}

// buildOption 构建分类选项树
func (c *GoodsCategoryCase) buildOption(categoryList []*models.GoodsCategory, parentID int64, disabled bool) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range categoryList {
		// 仅处理当前父节点下的直接子分类。
		if item.ParentID != parentID {
			continue
		}
		category := &commonv1.TreeOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: disabled && item.ParentID == 0,
		}
		category.Children = c.buildOption(categoryList, item.ID, disabled)
		res = append(res, category)
	}
	return res
}
