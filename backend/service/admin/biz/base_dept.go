package biz

import (
	"context"
	"fmt"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseDeptCase 部门业务实例
type BaseDeptCase struct {
	*biz.BaseCase
	*data.BaseDeptRepository
	formMapper *mapper.CopierMapper[adminv1.BaseDeptForm, models.BaseDept]
	mapper     *mapper.CopierMapper[adminv1.BaseDept, models.BaseDept]
}

// NewBaseDeptCase 创建部门业务实例
func NewBaseDeptCase(
	baseCase *biz.BaseCase,
	baseDeptRepo *data.BaseDeptRepository,
) *BaseDeptCase {
	return &BaseDeptCase{
		BaseCase:           baseCase,
		BaseDeptRepository: baseDeptRepo,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseDeptForm, models.BaseDept](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseDept, models.BaseDept](),
	}
}

// TreeBaseDepts 查询部门树
func (c *BaseDeptCase) TreeBaseDepts(ctx context.Context) (*adminv1.TreeBaseDeptsResponse, error) {
	query := c.Query(ctx).BaseDept
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &adminv1.TreeBaseDeptsResponse{BaseDepts: c.buildBaseDeptTree(list, 0)}, nil
}

// OptionBaseDepts 查询部门选项
func (c *BaseDeptCase) OptionBaseDepts(ctx context.Context, req *adminv1.OptionBaseDeptsRequest) (*commonv1.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseDept
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &commonv1.TreeOptionResponse{List: c.buildBaseDeptOption(list, req.GetParentId())}, nil
}

// GetBaseDept 获取部门
func (c *BaseDeptCase) GetBaseDept(ctx context.Context, id int64) (*adminv1.BaseDeptForm, error) {
	baseDept, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDept)
	return res, nil
}

// CreateBaseDept 创建部门
func (c *BaseDeptCase) CreateBaseDept(ctx context.Context, req *adminv1.BaseDeptForm) error {
	baseDept := c.formMapper.ToEntity(req)

	err := c.Create(ctx, baseDept)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/0/%d", baseDept.ID)
	parentID := req.GetParentId()
	// 存在父部门时，继承父部门路径并追加当前节点。
	if parentID != 0 {
		var parentDept *models.BaseDept
		parentDept, err = c.FindByID(ctx, parentID)
		if err != nil {
			return errorsx.Internal("创建部门失败，更新路径错误").WithCause(err)
		}
		path = fmt.Sprintf("%s/%d", parentDept.Path, baseDept.ID)
	}

	baseDept.Path = path
	return c.UpdateByID(ctx, baseDept)
}

// UpdateBaseDept 更新部门
func (c *BaseDeptCase) UpdateBaseDept(ctx context.Context, req *adminv1.BaseDeptForm) error {
	baseDept := c.formMapper.ToEntity(req)
	return c.UpdateByID(ctx, baseDept)
}

// DeleteBaseDept 删除部门
func (c *BaseDeptCase) DeleteBaseDept(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseDept

	for _, deptID := range ids {
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ParentID.Eq(deptID)))
		count, err := c.Count(ctx, opts...)
		if err != nil {
			return err
		}
		// 仍然存在子部门时，禁止删除当前部门。
		if count > 0 {
			return errorsx.HasChildrenConflict("删除部门失败，下面有部门", "base_dept", "base_dept")
		}
	}
	return c.DeleteByIDs(ctx, ids)
}

// SetBaseDeptStatus 设置部门状态
func (c *BaseDeptCase) SetBaseDeptStatus(ctx context.Context, req *adminv1.SetBaseDeptStatusRequest) error {
	query := c.Query(ctx).BaseDept

	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ParentID.Eq(req.GetId())))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return err
	}
	// 存在子部门时，不允许直接调整当前部门状态。
	if count > 0 {
		return errorsx.HasChildrenConflict("设置状态失败，下面有部门", "base_dept", "base_dept")
	}

	return c.UpdateByID(ctx, &models.BaseDept{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// buildBaseDeptTree 构建部门树
func (c *BaseDeptCase) buildBaseDeptTree(list []*models.BaseDept, parentID int64) []*adminv1.BaseDept {
	res := make([]*adminv1.BaseDept, 0)
	for _, item := range list {
		// 非当前父节点的部门不参与当前层级构建。
		if item.ParentID != parentID {
			continue
		}
		baseDept := c.mapper.ToDTO(item)
		baseDept.Children = c.buildBaseDeptTree(list, item.ID)
		res = append(res, baseDept)
	}
	return res
}

// buildBaseDeptOption 构建部门选项树
func (c *BaseDeptCase) buildBaseDeptOption(list []*models.BaseDept, parentID int64) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range list {
		// 非当前父节点的部门不参与当前层级选项构建。
		if item.ParentID != parentID {
			continue
		}
		option := &commonv1.TreeOptionResponse_Option{
			Label: item.Name,
			Value: item.ID,
		}
		option.Children = c.buildBaseDeptOption(list, item.ID)
		res = append(res, option)
	}
	return res
}
