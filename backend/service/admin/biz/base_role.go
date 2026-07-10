package biz

import (
	"context"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
)

// BaseRoleCase 角色业务实例
type BaseRoleCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseRoleRepository
	baseTenantRepo *data.BaseTenantRepository
	casbinRuleCase *CasbinRuleCase
	formMapper     *mapper.CopierMapper[adminv1.BaseRoleForm, models.BaseRole]
	mapper         *mapper.CopierMapper[adminv1.BaseRole, models.BaseRole]
}

// NewBaseRoleCase 创建角色业务实例
func NewBaseRoleCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseRoleRepo *data.BaseRoleRepository,
	baseTenantRepo *data.BaseTenantRepository,
	casbinRuleCase *CasbinRuleCase,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseCase:           baseCase,
		tx:                 tx,
		BaseRoleRepository: baseRoleRepo,
		baseTenantRepo:     baseTenantRepo,
		casbinRuleCase:     casbinRuleCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseRoleForm, models.BaseRole](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseRole, models.BaseRole](),
	}
}

// OptionBaseRoles 查询角色选项
func (c *BaseRoleCase) OptionBaseRoles(ctx context.Context, req *adminv1.OptionBaseRolesRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Code.NotIn(_const.BASE_ROLE_CODE_SUPER, _const.BASE_ROLE_CODE_TENANT)))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
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
	defaultTenant, err := c.getDefaultTenant(ctx)
	if err != nil {
		return nil, err
	}

	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Code.Neq(_const.BASE_ROLE_CODE_SUPER)))
	// 租户内置管理员角色只展示默认租户模板，其他租户副本仅用于登录绑定与权限同步。
	opts = append(opts, repository.Where(field.Or(
		query.Code.Neq(_const.BASE_ROLE_CODE_TENANT),
		field.And(query.Code.Eq(_const.BASE_ROLE_CODE_TENANT), query.TenantID.Eq(defaultTenant.ID)),
	)))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
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
	// 内置角色编码只允许系统初始化数据维护，避免租户侧创建同名角色覆盖固定权限边界。
	if _const.IsDefaultBaseRole(baseRole.Code) {
		return errorsx.ProtectedResourceConflict("创建角色失败，不能使用内置角色编码", "base_role")
	}
	err := c.validateAssignableMenus(ctx, req.GetMenus())
	if err != nil {
		return err
	}
	baseRole.Menus = _string.ConvertInt64ArrayToString(req.GetMenus())
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, baseRole)
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
	// 内置角色不允许被修改，避免破坏平台和租户的固定权限边界。
	if _const.IsDefaultBaseRole(oldBaseRole.Code) {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能操作内置角色", "base_role")
	}
	if _const.IsDefaultBaseRole(req.GetCode()) {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能使用内置角色编码", "base_role")
	}
	err = c.validateAssignableMenus(ctx, req.GetMenus())
	if err != nil {
		return err
	}

	baseRole := c.formMapper.ToEntity(req)
	baseRole.TenantID = oldBaseRole.TenantID
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
	opts = append(opts, repository.Where(query.Code.In(_const.BASE_ROLE_CODE_SUPER, _const.BASE_ROLE_CODE_TENANT)))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return errorsx.Internal("删除角色失败").WithCause(err)
	}
	// 命中内置角色时，禁止继续删除。
	if count > 0 {
		return errorsx.ProtectedResourceConflict("删除角色失败，不能操作内置角色", "base_role")
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.casbinRuleCase.DeleteCasbinRuleByRoleIDs(ctx, ids)
		if err != nil {
			return err
		}
		return c.DeleteByIDs(ctx, ids)
	})
}

// SetBaseRoleStatus 设置角色状态
func (c *BaseRoleCase) SetBaseRoleStatus(ctx context.Context, req *adminv1.SetBaseRoleStatusRequest) error {
	baseRole, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	// 内置角色不允许修改状态，避免管理员身份被禁用后无法维护租户。
	if _const.IsDefaultBaseRole(baseRole.Code) {
		return errorsx.ProtectedResourceConflict("设置状态失败，不能操作内置角色", "base_role")
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
	if oldBaseRole.Code == _const.BASE_ROLE_CODE_SUPER {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能操作内置角色", "base_role")
	}
	var defaultTenant *models.BaseTenant
	defaultTenant, err = c.getDefaultTenant(ctx)
	if err != nil {
		return err
	}
	// 普通租户的内置管理员角色只允许跟随默认租户模板，不允许单独改菜单。
	if oldBaseRole.Code == _const.BASE_ROLE_CODE_TENANT && oldBaseRole.TenantID != defaultTenant.ID {
		return errorsx.ProtectedResourceConflict("更新角色失败，不能操作租户内置角色", "base_role")
	}
	err = c.validateAssignableMenus(ctx, req.GetMenus())
	if err != nil {
		return err
	}

	baseRole := &models.BaseRole{
		ID:       req.GetId(),
		TenantID: oldBaseRole.TenantID,
		Menus:    _string.ConvertInt64ArrayToString(req.GetMenus()),
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.UpdateByID(ctx, baseRole)
		if err != nil {
			return err
		}
		baseRole.Code = oldBaseRole.Code
		// 默认租户的租户管理员角色是权限模板，变更后同步所有租户副本。
		if oldBaseRole.Code == _const.BASE_ROLE_CODE_TENANT {
			return c.syncTenantRoleMenus(ctx, baseRole)
		}
		return c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	})
}

// validateAssignableMenus 校验待保存菜单不超过当前登录角色的权限范围。
func (c *BaseRoleCase) validateAssignableMenus(ctx context.Context, menus []int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// 超级管理员拥有完整菜单分配权限，不需要做菜单上限校验。
	if authInfo.RoleCode == _const.BASE_ROLE_CODE_SUPER {
		return nil
	}

	var currentBaseRole *models.BaseRole
	currentBaseRole, err = c.FindByID(ctx, authInfo.RoleId)
	if err != nil {
		return errorsx.Internal("查询当前角色权限失败").WithCause(err)
	}
	// 当前角色已停用时，不允许继续分配其他角色权限。
	if currentBaseRole.Status != _const.STATUS_ENABLE {
		return errorsx.PermissionDenied("角色已被禁用")
	}

	allowedMenuIDs := _string.ConvertJsonStringToInt64Array(currentBaseRole.Menus)
	allowedMenuIDSet := make(map[int64]struct{}, len(allowedMenuIDs))
	for _, menuID := range allowedMenuIDs {
		allowedMenuIDSet[menuID] = struct{}{}
	}
	for _, menuID := range menus {
		// 提交菜单不在当前角色权限范围内时，直接拒绝保存。
		if _, ok := allowedMenuIDSet[menuID]; !ok {
			return errorsx.PermissionDenied("设置角色菜单权限失败，不能分配超出当前角色的菜单权限")
		}
	}
	return nil
}

// getDefaultTenant 查询系统默认租户。
func (c *BaseRoleCase) getDefaultTenant(ctx context.Context) (*models.BaseTenant, error) {
	query := c.baseTenantRepo.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(databaseGorm.DefaultTenantCode)))
	baseTenant, err := c.baseTenantRepo.Find(ctx, opts...)
	if err != nil {
		return nil, errorsx.Internal("查询默认租户失败").WithCause(err)
	}
	return baseTenant, nil
}

// syncTenantRoleMenus 同步默认租户管理员角色菜单到所有租户副本并重建权限。
func (c *BaseRoleCase) syncTenantRoleMenus(ctx context.Context, templateRole *models.BaseRole) error {
	query := c.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}

	for _, item := range list {
		if item.ID != templateRole.ID && item.Menus != templateRole.Menus {
			err = c.UpdateByID(ctx, &models.BaseRole{
				ID:       item.ID,
				TenantID: item.TenantID,
				Menus:    templateRole.Menus,
			})
			if err != nil {
				return err
			}
			item.Menus = templateRole.Menus
		}
		err = c.casbinRuleCase.rebuildCasbinRuleByRole(ctx, item)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}
