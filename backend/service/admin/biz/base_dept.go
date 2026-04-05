package biz

import (
	"context"
	"errors"
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

// BaseDeptCase 部门业务实例
type BaseDeptCase struct {
	*biz.BaseCase
	*data.BaseDeptRepo
	formMapper *mapper.CopierMapper[admin.BaseDeptForm, models.BaseDept]
	mapper     *mapper.CopierMapper[admin.BaseDept, models.BaseDept]
}

// NewBaseDeptCase 创建部门业务实例
func NewBaseDeptCase(
	baseCase *biz.BaseCase,
	baseDeptRepo *data.BaseDeptRepo,
) *BaseDeptCase {
	return &BaseDeptCase{
		BaseCase:     baseCase,
		BaseDeptRepo: baseDeptRepo,
		formMapper:   mapper.NewCopierMapper[admin.BaseDeptForm, models.BaseDept](),
		mapper:       mapper.NewCopierMapper[admin.BaseDept, models.BaseDept](),
	}
}

// TreeBaseDept 查询部门树
func (c *BaseDeptCase) TreeBaseDept(ctx context.Context) (*admin.TreeBaseDeptResponse, error) {
	query := c.Query(ctx).BaseDept
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &admin.TreeBaseDeptResponse{List: c.buildBaseDeptTree(list, 0)}, nil
}

// OptionBaseDept 查询部门选项
func (c *BaseDeptCase) OptionBaseDept(ctx context.Context, req *admin.OptionBaseDeptRequest) (*common.TreeOptionResponse, error) {
	query := c.Query(ctx).BaseDept
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.Sort.Asc()))
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &common.TreeOptionResponse{List: c.buildBaseDeptOption(list, req.GetParentId())}, nil
}

// GetBaseDept 获取部门
func (c *BaseDeptCase) GetBaseDept(ctx context.Context, id int64) (*admin.BaseDeptForm, error) {
	baseDept, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseDept)
	return res, nil
}

// CreateBaseDept 创建部门
func (c *BaseDeptCase) CreateBaseDept(ctx context.Context, req *admin.BaseDeptForm) error {
	baseDept := c.formMapper.ToEntity(req)

	err := c.Create(ctx, baseDept)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/0/%d", baseDept.ID)
	parentId := req.GetParentId()
	if parentId != 0 {
		var parentDept *models.BaseDept
		parentDept, err = c.FindById(ctx, parentId)
		if err != nil {
			return errors.New("创建部门失败，更新路径错误")
		}
		path = fmt.Sprintf("%s/%d", parentDept.Path, baseDept.ID)
	}

	baseDept.Path = path
	return c.UpdateById(ctx, baseDept)
}

// UpdateBaseDept 更新部门
func (c *BaseDeptCase) UpdateBaseDept(ctx context.Context, req *admin.BaseDeptForm) error {
	baseDept := c.formMapper.ToEntity(req)
	return c.UpdateById(ctx, baseDept)
}

// DeleteBaseDept 删除部门
func (c *BaseDeptCase) DeleteBaseDept(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseDept

	for _, deptId := range ids {
		count, err := c.Count(ctx, repo.Where(query.ParentID.Eq(deptId)))
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("删除部门失败,下面有部门")
		}
	}
	return c.DeleteByIds(ctx, ids)
}

// SetBaseDeptStatus 设置部门状态
func (c *BaseDeptCase) SetBaseDeptStatus(ctx context.Context, req *common.SetStatusRequest) error {
	query := c.Query(ctx).BaseDept

	count, err := c.Count(ctx, repo.Where(query.ParentID.Eq(req.GetId())))
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("设置状态失败,下面有部门")
	}

	return c.UpdateById(ctx, &models.BaseDept{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// buildBaseDeptTree 构建部门树
func (c *BaseDeptCase) buildBaseDeptTree(list []*models.BaseDept, parentId int64) []*admin.BaseDept {
	res := make([]*admin.BaseDept, 0)
	for _, item := range list {
		if item.ParentID != parentId {
			continue
		}
		baseDept := c.mapper.ToDTO(item)
		baseDept.Children = c.buildBaseDeptTree(list, item.ID)
		res = append(res, baseDept)
	}
	return res
}

// buildBaseDeptOption 构建部门选项树
func (c *BaseDeptCase) buildBaseDeptOption(list []*models.BaseDept, parentId int64) []*common.TreeOptionResponse_Option {
	res := make([]*common.TreeOptionResponse_Option, 0)
	for _, item := range list {
		if item.ParentID != parentId {
			continue
		}
		option := &common.TreeOptionResponse_Option{
			Label: item.Name,
			Value: item.ID,
		}
		option.Children = c.buildBaseDeptOption(list, item.ID)
		res = append(res, option)
	}
	return res
}
