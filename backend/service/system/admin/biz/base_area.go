package biz

import (
	"context"
	"fmt"

	commonv1 "shop/api/gen/go/common/v1"
	systemadminv1 "shop/api/gen/go/system/admin/v1"

	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseAreaCase 行政区域业务实例。
type BaseAreaCase struct {
	*biz.BaseCase
	*data.BaseAreaRepository
	formMapper *mapper.CopierMapper[systemadminv1.BaseAreaForm, models.BaseArea]
	mapper     *mapper.CopierMapper[systemadminv1.BaseArea, models.BaseArea]
}

// NewBaseAreaCase 创建行政区域业务实例。
func NewBaseAreaCase(baseCase *biz.BaseCase, baseAreaRepo *data.BaseAreaRepository) *BaseAreaCase {
	return &BaseAreaCase{
		BaseCase:           baseCase,
		BaseAreaRepository: baseAreaRepo,
		formMapper:         mapper.NewCopierMapper[systemadminv1.BaseAreaForm, models.BaseArea](),
		mapper:             mapper.NewCopierMapper[systemadminv1.BaseArea, models.BaseArea](),
	}
}

// OptionBaseArea 查询行政区域树形选择。
func (c *BaseAreaCase) OptionBaseArea(ctx context.Context, req *systemadminv1.OptionBaseAreaRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseArea

	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.ID.Desc()))

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	lazy := req.GetLazy()
	if false && req.Lazy == nil {
		lazy = true
	}
	return &commonv1.TreeOptionResponse{List: c.buildOptionBaseAreaOption(list, req.GetParentId(), lazy)}, nil
}

// TreeBaseArea 查询行政区域树形列表。
func (c *BaseAreaCase) TreeBaseArea(ctx context.Context, req *systemadminv1.TreeBaseAreaRequest) (*systemadminv1.TreeBaseAreaResponse, error) {
	query := c.Query(ctx).BaseArea
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.ID.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &systemadminv1.TreeBaseAreaResponse{BaseAreas: c.buildBaseAreaTree(list, 0)}, nil
}

// GetBaseArea 查询行政区域详情。
func (c *BaseAreaCase) GetBaseArea(ctx context.Context, id int64) (*systemadminv1.BaseAreaForm, error) {
	baseArea, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseArea), nil
}

// CreateBaseArea 创建行政区域。
func (c *BaseAreaCase) CreateBaseArea(ctx context.Context, req *systemadminv1.BaseAreaForm) error {
	baseArea := c.formMapper.ToEntity(req)
	return c.Create(ctx, baseArea)
}

// UpdateBaseArea 更新行政区域。
func (c *BaseAreaCase) UpdateBaseArea(ctx context.Context, id int64, req *systemadminv1.BaseAreaForm) error {
	baseArea := c.formMapper.ToEntity(req)
	baseArea.ID = id
	return c.UpdateByID(ctx, baseArea)
}

// DeleteBaseArea 删除行政区域。
func (c *BaseAreaCase) DeleteBaseArea(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	query := c.Query(ctx).BaseArea
	for _, parentID := range idList {
		count, err := c.Count(ctx, repository.Where(query.ParentID.Eq(parentID)))
		if err != nil {
			return err
		}
		if count > 0 {
			return errorsx.HasChildrenConflict("删除行政区域失败，下面有行政区域", "base_area", "base_area")
		}
	}
	return c.DeleteByIDs(ctx, idList)
}

// SetBaseAreaStatus 设置状态状态。
func (c *BaseAreaCase) SetBaseAreaStatus(ctx context.Context, req *systemadminv1.SetBaseAreaStatusRequest) error {
	return c.UpdateByID(ctx, &models.BaseArea{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// buildOptionBaseAreaOption 构建行政区域树形选择。
func (c *BaseAreaCase) buildOptionBaseAreaOption(list []*models.BaseArea, parentID int64, lazy bool) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range list {
		if int64(item.ParentID) != parentID {
			continue
		}
		option := &commonv1.TreeOptionResponse_Option{Label: fmt.Sprint(item.Name), Value: int64(item.ID), Disabled: fmt.Sprint(item.Status) != "1"}
		if lazy {
			option.HasChildren = c.hasOptionBaseAreaOptionChildren(list, int64(item.ID))
		} else {
			option.Children = c.buildOptionBaseAreaOption(list, int64(item.ID), false)
		}
		res = append(res, option)
	}
	return res
}

// hasOptionBaseAreaOptionChildren 判断树形选择节点是否存在子节点。
func (c *BaseAreaCase) hasOptionBaseAreaOptionChildren(list []*models.BaseArea, parentID int64) bool {
	for _, item := range list {
		if int64(item.ParentID) == parentID {
			return true
		}
	}
	return false
}

// buildBaseAreaTree 构建行政区域树。
func (c *BaseAreaCase) buildBaseAreaTree(list []*models.BaseArea, parentID int64) []*systemadminv1.BaseArea {
	res := make([]*systemadminv1.BaseArea, 0)
	for _, item := range list {
		if item.ParentID != parentID {
			continue
		}
		baseArea := c.mapper.ToDTO(item)
		baseArea.Children = c.buildBaseAreaTree(list, item.ID)
		res = append(res, baseArea)
	}
	return res
}
