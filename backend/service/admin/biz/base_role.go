package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseRoleCase 角色业务实例
type BaseRoleCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseRoleRepository
	casbinRuleCase *CasbinRuleCase
	formMapper     *mapper.CopierMapper[adminv1.BaseRoleForm, models.BaseRole]
	mapper         *mapper.CopierMapper[adminv1.BaseRole, models.BaseRole]
}

// NewBaseRoleCase 创建角色业务实例
func NewBaseRoleCase(baseCase *biz.BaseCase, tx data.Transaction, baseRoleRepo *data.BaseRoleRepository, casbinRuleCase *CasbinRuleCase) *BaseRoleCase {
	return &BaseRoleCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseRoleRepository: baseRoleRepo,
		casbinRuleCase:     casbinRuleCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseRoleForm, models.BaseRole](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseRole, models.BaseRole](),
	}
}

// OptionBaseRoles 查询角色选项
func (c *BaseRoleCase) OptionBaseRoles(ctx context.Context) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label: item.Name,
			Value: item.ID,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseRoles 分页查询角色
func (c *BaseRoleCase) PageBaseRoles(ctx context.Context, req *adminv1.PageBaseRolesRequest) (*adminv1.PageBaseRolesResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入名称关键字时，按名称模糊匹配角色。
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	// 传入编码关键字时，按编码模糊匹配角色。
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseRole, 0, len(list))
	for _, item := range list {
		baseRole := c.mapper.ToDTO(item)
		baseRole.Menus = _string.ConvertJsonStringToInt64Array(item.Menus)
		resList = append(resList, baseRole)
	}
	return &adminv1.PageBaseRolesResponse{BaseRoles: resList, Total: int32(total)}, nil
}

// GetBaseRole 获取角色
func (c *BaseRoleCase) GetBaseRole(ctx context.Context, id int64) (*adminv1.BaseRoleForm, error) {
	baseRole, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseRole)
	res.Menus = _string.ConvertJsonStringToInt64Array(baseRole.Menus)
	return res, nil
}

// CreateBaseRole 创建角色
func (c *BaseRoleCase) CreateBaseRole(ctx context.Context, req *adminv1.BaseRoleForm) error {
	baseRole := c.formMapper.ToEntity(req)
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.Create(ctx, baseRole)
		if err != nil {
			// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("角色编码重复", "base_role", "code", "unique_base_role").WithCause(err)
			}
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// UpdateBaseRole 更新角色
func (c *BaseRoleCase) UpdateBaseRole(ctx context.Context, req *adminv1.BaseRoleForm) error {
	oldBaseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 超级管理员角色不允许被修改。
	if oldBaseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return errorsx.PermissionDenied("更新角色失败，不能操作超级管理员角色")
	}

	baseRole := c.formMapper.ToEntity(req)
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, baseRole)
		if err != nil {
			// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("角色编码重复", "base_role", "code", "unique_base_role").WithCause(err)
			}
			return err
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// DeleteBaseRole 删除角色
func (c *BaseRoleCase) DeleteBaseRole(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseRole

	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.In(ids...)))
	opts = append(opts, repository.Where(query.Code.Eq(_const.BASE_ROLE_CODE_SUPER)))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return errorsx.Internal("删除角色失败").WithCause(err)
	}
	// 命中超级管理员角色时，禁止继续删除。
	if count > 0 {
		return errorsx.PermissionDenied("删除角色失败，不能操作超级管理员角色")
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.DeleteByIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.casbinRuleCase.DeleteCasbinRuleByRoleIDs(ctx, ids)
	})
}

// SetBaseRoleStatus 设置角色状态
func (c *BaseRoleCase) SetBaseRoleStatus(ctx context.Context, req *adminv1.SetBaseRoleStatusRequest) error {
	baseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 超级管理员角色不允许修改状态。
	if baseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return errorsx.PermissionDenied("设置状态失败，不能操作超级管理员角色")
	}
	return c.UpdateByID(ctx, &models.BaseRole{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// SetBaseRoleMenu 设置角色菜单
func (c *BaseRoleCase) SetBaseRoleMenu(ctx context.Context, req *adminv1.SetBaseRoleMenuRequest) error {
	oldBaseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 超级管理员角色不允许调整菜单权限。
	if oldBaseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return errorsx.PermissionDenied("更新角色失败，不能操作超级管理员角色")
	}

	baseRole := &models.BaseRole{
		ID:    req.GetId(),
		Menus: _string.ConvertInt64ArrayToString(req.GetMenus()),
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, baseRole)
		if err != nil {
			return err
		}
		baseRole.Code = oldBaseRole.Code
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}
