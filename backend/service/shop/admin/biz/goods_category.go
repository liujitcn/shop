package biz

import (
	"context"
	"strconv"
	"strings"

	commonv1 "shop/api/gen/go/common/v1"
	shopadminv1 "shop/api/gen/go/shop/admin/v1"
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
	formMapper *mapper.CopierMapper[shopadminv1.GoodsCategoryForm, models.GoodsCategory]
	mapper     *mapper.CopierMapper[shopadminv1.GoodsCategory, models.GoodsCategory]
}

// NewGoodsCategoryCase 创建商品分类业务实例
func NewGoodsCategoryCase(baseCase *biz.BaseCase, goodsCategoryRepo *data.GoodsCategoryRepository) *GoodsCategoryCase {
	return &GoodsCategoryCase{
		BaseCase:                baseCase,
		GoodsCategoryRepository: goodsCategoryRepo,
		formMapper:              mapper.NewCopierMapper[shopadminv1.GoodsCategoryForm, models.GoodsCategory](),
		mapper:                  mapper.NewCopierMapper[shopadminv1.GoodsCategory, models.GoodsCategory](),
	}
}

// OptionGoodsCategory 查询分类选项
func (c *GoodsCategoryCase) OptionGoodsCategory(ctx context.Context, req *shopadminv1.OptionGoodsCategoryRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listGoodsCategoryParentIDsWithChildren(ctx, list)
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &commonv1.TreeOptionResponse{
		List: c.buildOption(list, parentID, req.ParentId == nil, req.GetLazy(), hasChildren),
	}, nil
}

// TreeGoodsCategory 查询分类树
func (c *GoodsCategoryCase) TreeGoodsCategory(ctx context.Context, req *shopadminv1.TreeGoodsCategoryRequest) (*shopadminv1.TreeGoodsCategoryResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetLazy() {
		opts = append(opts, repository.Where(query.ParentID.Eq(req.GetParentId())))
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.listGoodsCategoryParentIDsWithChildren(ctx, list)
	if err != nil {
		return nil, err
	}
	parentID := int64(0)
	if req.GetLazy() {
		parentID = req.GetParentId()
	}
	return &shopadminv1.TreeGoodsCategoryResponse{
		GoodsCategories: c.buildTree(list, parentID, req.GetLazy(), hasChildren),
	}, nil
}

// GetGoodsCategory 获取分类
func (c *GoodsCategoryCase) GetGoodsCategory(ctx context.Context, id int64) (*shopadminv1.GoodsCategoryForm, error) {
	goodsCategory, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(goodsCategory)
	return res, nil
}

// CreateGoodsCategory 创建分类
func (c *GoodsCategoryCase) CreateGoodsCategory(ctx context.Context, req *shopadminv1.GoodsCategoryForm) error {
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
func (c *GoodsCategoryCase) UpdateGoodsCategory(ctx context.Context, req *shopadminv1.GoodsCategoryForm) error {
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
func (c *GoodsCategoryCase) SetGoodsCategoryStatus(ctx context.Context, req *shopadminv1.SetGoodsCategoryStatusRequest) error {
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
func (c *GoodsCategoryCase) buildTree(
	categoryList []*models.GoodsCategory,
	parentID int64,
	lazy bool,
	hasChildren map[int64]struct{},
) []*shopadminv1.GoodsCategory {
	res := make([]*shopadminv1.GoodsCategory, 0)
	for _, item := range categoryList {
		// 仅处理当前父节点下的直接子分类。
		if item.ParentID != parentID {
			continue
		}
		category := c.mapper.ToDTO(item)
		category.CreatedAt = _time.TimeToTimeString(item.CreatedAt)
		category.UpdatedAt = _time.TimeToTimeString(item.UpdatedAt)
		_, category.HasChildren = hasChildren[item.ID]
		if !lazy {
			category.Children = c.buildTree(categoryList, item.ID, false, hasChildren)
		}
		res = append(res, category)
	}
	return res
}

// buildOption 构建分类选项树
func (c *GoodsCategoryCase) buildOption(
	categoryList []*models.GoodsCategory,
	parentID int64,
	disabled bool,
	lazy bool,
	hasChildren map[int64]struct{},
) []*commonv1.TreeOptionResponse_Option {
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
		_, category.HasChildren = hasChildren[item.ID]
		if !lazy {
			category.Children = c.buildOption(categoryList, item.ID, disabled, false, hasChildren)
		}
		res = append(res, category)
	}
	return res
}

// listGoodsCategoryParentIDsWithChildren 查询存在子节点的商品分类父级编号。
func (c *GoodsCategoryCase) listGoodsCategoryParentIDsWithChildren(
	ctx context.Context,
	list []*models.GoodsCategory,
) (map[int64]struct{}, error) {
	parentIDs := make([]int64, 0, len(list))
	for _, item := range list {
		parentIDs = append(parentIDs, item.ID)
	}
	hasChildren := make(map[int64]struct{}, len(parentIDs))
	if len(parentIDs) == 0 {
		return hasChildren, nil
	}

	query := c.Query(ctx).GoodsCategory
	children, err := c.List(ctx, repository.Where(query.ParentID.In(parentIDs...)))
	if err != nil {
		return nil, err
	}
	for _, item := range children {
		hasChildren[item.ParentID] = struct{}{}
	}
	return hasChildren, nil
}
