package biz

import (
	"context"
	"strconv"
	"strings"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// GoodsCategoryCase 商品分类业务实例
type GoodsCategoryCase struct {
	*biz.BaseCase
	*data.GoodsCategoryRepo
	formMapper *mapper.CopierMapper[admin.GoodsCategoryForm, models.GoodsCategory]
}

// NewGoodsCategoryCase 创建商品分类业务实例
func NewGoodsCategoryCase(baseCase *biz.BaseCase, goodsCategoryRepo *data.GoodsCategoryRepo) *GoodsCategoryCase {
	return &GoodsCategoryCase{
		BaseCase:          baseCase,
		GoodsCategoryRepo: goodsCategoryRepo,
		formMapper:        mapper.NewCopierMapper[admin.GoodsCategoryForm, models.GoodsCategory](),
	}
}

// TreeGoodsCategory 查询分类树
func (c *GoodsCategoryCase) TreeGoodsCategory(ctx context.Context) (*admin.TreeGoodsCategoryResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &admin.TreeGoodsCategoryResponse{
		List: c.buildTree(list, 0),
	}, nil
}

// OptionGoodsCategory 查询分类选项
func (c *GoodsCategoryCase) OptionGoodsCategory(ctx context.Context, req *admin.OptionGoodsCategoryRequest) (*common.TreeOptionResponse, error) {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &common.TreeOptionResponse{
		List: c.buildOption(list, 0, req.ParentId == nil),
	}, nil
}

// GetGoodsCategory 获取分类
func (c *GoodsCategoryCase) GetGoodsCategory(ctx context.Context, id int64) (*admin.GoodsCategoryForm, error) {
	goodsCategory, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(goodsCategory)
	return res, nil
}

// CreateGoodsCategory 创建分类
func (c *GoodsCategoryCase) CreateGoodsCategory(ctx context.Context, req *admin.GoodsCategoryForm) error {
	goodsCategory := c.formMapper.ToEntity(req)
	if goodsCategory.ParentID == 0 {
		goodsCategory.Path = "0"
	} else {
		parentGoodsCategory, err := c.FindById(ctx, goodsCategory.ParentID)
		if err != nil {
			return err
		}
		goodsCategory.Path = parentGoodsCategory.Path + "/" + strconv.FormatInt(parentGoodsCategory.ID, 10)
	}
	return c.Create(ctx, goodsCategory)
}

// UpdateGoodsCategory 更新分类
func (c *GoodsCategoryCase) UpdateGoodsCategory(ctx context.Context, req *admin.GoodsCategoryForm) error {
	goodsCategory, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return err
	}

	goodsCategory.ParentID = req.GetParentId()
	goodsCategory.Picture = req.GetPicture()
	goodsCategory.Name = req.GetName()
	goodsCategory.Sort = req.GetSort()
	goodsCategory.Status = int32(req.GetStatus())
	if goodsCategory.ParentID == 0 {
		goodsCategory.Path = "0"
	} else {
		var parentGoodsCategory *models.GoodsCategory
		parentGoodsCategory, err = c.FindById(ctx, goodsCategory.ParentID)
		if err != nil {
			return err
		}
		goodsCategory.Path = parentGoodsCategory.Path + "/" + strconv.FormatInt(parentGoodsCategory.ID, 10)
	}
	return c.UpdateById(ctx, goodsCategory)
}

// DeleteGoodsCategory 删除分类
func (c *GoodsCategoryCase) DeleteGoodsCategory(ctx context.Context, id string) error {
	return c.DeleteByIds(ctx, _string.ConvertStringToInt64Array(id))
}

// SetGoodsCategoryStatus 设置分类状态
func (c *GoodsCategoryCase) SetGoodsCategoryStatus(ctx context.Context, req *common.SetStatusRequest) error {
	return c.UpdateById(ctx, &models.GoodsCategory{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// NameMap 查询分类路径名称映射
func (c *GoodsCategoryCase) NameMap(ctx context.Context, parentId *int64) map[int64]string {
	query := c.Query(ctx).GoodsCategory
	opts := make([]repo.QueryOption, 0, 3)
	if parentId != nil {
		opts = append(opts, repo.Where(query.ParentID.Eq(*parentId)))
	}
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))

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

	for categoryId, path := range categoryPathMap {
		paths := strings.Split(path, "/")
		pathName := make([]string, 0, len(paths))
		for _, item := range paths {
			value, convErr := strconv.ParseInt(item, 10, 64)
			if convErr != nil {
				continue
			}
			if name, ok := categoryNameMap[value]; ok {
				pathName = append(pathName, name)
			}
		}
		categoryPathMap[categoryId] = strings.Join(pathName, "/")
	}
	return categoryPathMap
}

// buildTree 构建分类树
func (c *GoodsCategoryCase) buildTree(categoryList []*models.GoodsCategory, parentId int64) []*admin.GoodsCategory {
	res := make([]*admin.GoodsCategory, 0)
	for _, item := range categoryList {
		if item.ParentID != parentId {
			continue
		}
		category := &admin.GoodsCategory{
			Id:        item.ID,
			ParentId:  item.ParentID,
			Name:      item.Name,
			Picture:   item.Picture,
			Sort:      item.Sort,
			Status:    common.Status(item.Status),
			CreatedAt: _time.TimeToTimeString(item.CreatedAt),
			UpdatedAt: _time.TimeToTimeString(item.UpdatedAt),
		}
		category.Children = c.buildTree(categoryList, item.ID)
		res = append(res, category)
	}
	return res
}

// buildOption 构建分类选项树
func (c *GoodsCategoryCase) buildOption(categoryList []*models.GoodsCategory, parentId int64, disabled bool) []*common.TreeOptionResponse_Option {
	res := make([]*common.TreeOptionResponse_Option, 0)
	for _, item := range categoryList {
		if item.ParentID != parentId {
			continue
		}
		category := &common.TreeOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: disabled && item.ParentID == 0,
		}
		category.Children = c.buildOption(categoryList, item.ID, disabled)
		res = append(res, category)
	}
	return res
}
