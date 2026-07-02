package biz

import (
	"context"
	"fmt"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

const (
	baseTenantAdminRoleCode    = _const.BASE_ROLE_CODE_TENANT
	baseTenantAdminRoleName    = "租户管理员"
	baseTenantAdminUserName    = "admin"
	baseTenantAdminNickName    = "管理员"
	baseTenantAdminPassword    = "112233"
	baseTenantDefaultDeptName  = "默认部门"
	baseTenantDefaultDeptPath  = "/0/%d"
	baseTenantDefaultDeptSort  = int32(0)
	baseTenantDefaultRoleScope = _const.BASE_ROLE_DATA_SCOPE_ALL
)

// baseTenantAdminMenuIDs 表示普通租户管理员默认可见的后台菜单，明确排除平台公共管理、账单、商城服务和推荐等能力。
var baseTenantAdminMenuIDs = []int64{
	10, 20, 90,
	50, 53,
	200, 2000, 2001, 2002, 2003, 2004, 2005,
	2100, 2101, 2102, 2103, 2104, 2105,
	2200, 2201, 2202, 2203, 2204,
	300, 3000, 3001, 3002, 3003, 3004,
	3100, 3101, 3102, 3103, 3104,
	3200, 3201, 3202,
	3300, 3301,
	3400, 3401, 3402, 3403,
	3500, 3501,
	3600, 3601, 3602, 3603,
	400, 4000, 4001, 4002, 4003, 4100, 4200,
}

// BaseTenantCase 租户业务实例。
type BaseTenantCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseTenantRepository
	baseDeptRepo   *data.BaseDeptRepository
	baseRoleRepo   *data.BaseRoleRepository
	baseUserRepo   *data.BaseUserRepository
	baseMenuRepo   *data.BaseMenuRepository
	casbinRuleCase *CasbinRuleCase
	formMapper     *mapper.CopierMapper[adminv1.BaseTenantForm, models.BaseTenant]
	mapper         *mapper.CopierMapper[adminv1.BaseTenant, models.BaseTenant]
}

// NewBaseTenantCase 创建租户业务实例。
func NewBaseTenantCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseTenantRepo *data.BaseTenantRepository,
	baseDeptRepo *data.BaseDeptRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseUserRepo *data.BaseUserRepository,
	baseMenuRepo *data.BaseMenuRepository,
	casbinRuleCase *CasbinRuleCase,
) *BaseTenantCase {
	return &BaseTenantCase{
		BaseCase:             baseCase,
		tx:                   tx,
		BaseTenantRepository: baseTenantRepo,
		baseDeptRepo:         baseDeptRepo,
		baseRoleRepo:         baseRoleRepo,
		baseUserRepo:         baseUserRepo,
		baseMenuRepo:         baseMenuRepo,
		casbinRuleCase:       casbinRuleCase,
		formMapper:           mapper.NewCopierMapper[adminv1.BaseTenantForm, models.BaseTenant](),
		mapper:               mapper.NewCopierMapper[adminv1.BaseTenant, models.BaseTenant](),
	}
}

// OptionBaseTenants 查询租户选项。
func (c *BaseTenantCase) OptionBaseTenants(ctx context.Context, req *adminv1.OptionBaseTenantsRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.STATUS_ENABLE)))
	// 默认租户是系统内置租户，不在租户选择中展示。
	opts = append(opts, repository.Where(query.Code.Neq(databaseGorm.DefaultTenantCode)))
	if req.GetKeyword() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetKeyword()+"%")))
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

// PageBaseTenants 分页查询租户。
func (c *BaseTenantCase) PageBaseTenants(ctx context.Context, req *adminv1.PageBaseTenantsRequest) (*adminv1.PageBaseTenantsResponse, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 默认租户是系统内置租户，不在租户管理列表中展示。
	opts = append(opts, repository.Where(query.Code.Neq(databaseGorm.DefaultTenantCode)))
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseTenant, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.mapper.ToDTO(item))
	}
	return &adminv1.PageBaseTenantsResponse{BaseTenants: resList, Total: int32(total)}, nil
}

// GetBaseTenant 获取租户。
func (c *BaseTenantCase) GetBaseTenant(ctx context.Context, id int64) (*adminv1.BaseTenantForm, error) {
	baseTenant, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseTenant), nil
}

// FindByCode 按编码查询租户。
func (c *BaseTenantCase) FindByCode(ctx context.Context, code string) (*models.BaseTenant, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(code)))
	return c.Find(ctx, opts...)
}

// CreateBaseTenant 创建租户。
func (c *BaseTenantCase) CreateBaseTenant(ctx context.Context, req *adminv1.BaseTenantForm) error {
	baseTenant := c.formMapper.ToEntity(req)
	// 未指定状态时，新租户默认启用，便于创建后直接登录验证。
	if baseTenant.Status == 0 {
		baseTenant.Status = _const.STATUS_ENABLE
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.Create(ctx, baseTenant)
		if err != nil {
			// 命中租户编码唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsMySQLDuplicateKey(err) {
				return errorsx.UniqueConflict("租户编码重复", "base_tenant", "code", "unique_base_tenant").WithCause(err)
			}
			return err
		}
		return c.initTenantDefaults(ctx, baseTenant)
	})
}

// UpdateBaseTenant 更新租户。
func (c *BaseTenantCase) UpdateBaseTenant(ctx context.Context, req *adminv1.BaseTenantForm) error {
	oldBaseTenant, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if oldBaseTenant.Code == databaseGorm.DefaultTenantCode && req.GetCode() != databaseGorm.DefaultTenantCode {
		return errorsx.ProtectedResourceConflict("默认租户编码不能修改", "base_tenant")
	}

	baseTenant := c.formMapper.ToEntity(req)
	err = c.UpdateByID(ctx, baseTenant)
	if err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("租户编码重复", "base_tenant", "code", "unique_base_tenant").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteBaseTenant 删除租户。
func (c *BaseTenantCase) DeleteBaseTenant(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	baseTenants, err := c.List(ctx, repository.Where(c.Query(ctx).BaseTenant.ID.In(ids...)))
	if err != nil {
		return err
	}
	for _, item := range baseTenants {
		if item.Code == databaseGorm.DefaultTenantCode {
			return errorsx.ProtectedResourceConflict("默认租户不能删除", "base_tenant")
		}
	}
	return c.DeleteByIDs(ctx, ids)
}

// SetBaseTenantStatus 设置租户状态。
func (c *BaseTenantCase) SetBaseTenantStatus(ctx context.Context, req *adminv1.SetBaseTenantStatusRequest) error {
	baseTenant, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if baseTenant.Code == databaseGorm.DefaultTenantCode && req.GetStatus() != _const.STATUS_ENABLE {
		return errorsx.ProtectedResourceConflict("默认租户不能禁用", "base_tenant")
	}
	return c.UpdateByID(ctx, &models.BaseTenant{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// initTenantDefaults 初始化租户默认组织、角色和管理员账号。
func (c *BaseTenantCase) initTenantDefaults(ctx context.Context, baseTenant *models.BaseTenant) error {
	baseDept, err := c.createTenantDefaultDept(ctx, baseTenant)
	if err != nil {
		return err
	}

	var baseRole *models.BaseRole
	baseRole, err = c.createTenantAdminRole(ctx, baseTenant)
	if err != nil {
		return err
	}

	err = c.createTenantAdminUser(ctx, baseTenant, baseDept, baseRole)
	if err != nil {
		return err
	}
	err = c.casbinRuleCase.RebuildCasbinRuleByTenantRole(ctx, baseTenant.Code, baseRole)
	if err != nil {
		return errorsx.Internal("初始化租户管理员角色权限失败").WithCause(err)
	}
	return nil
}

// createTenantDefaultDept 创建租户默认部门。
func (c *BaseTenantCase) createTenantDefaultDept(ctx context.Context, baseTenant *models.BaseTenant) (*models.BaseDept, error) {
	baseDept := &models.BaseDept{
		TenantID: baseTenant.ID,
		ParentID: 0,
		Name:     baseTenantDefaultDeptName,
		Sort:     baseTenantDefaultDeptSort,
		Status:   _const.STATUS_ENABLE,
		Remark:   "租户默认部门",
	}

	err := c.baseDeptRepo.Create(ctx, baseDept)
	if err != nil {
		return nil, errorsx.Internal("初始化租户默认部门失败").WithCause(err)
	}

	baseDept.Path = fmt.Sprintf(baseTenantDefaultDeptPath, baseDept.ID)
	err = c.baseDeptRepo.UpdateByID(ctx, baseDept)
	if err != nil {
		return nil, errorsx.Internal("初始化租户默认部门失败").WithCause(err)
	}
	return baseDept, nil
}

// createTenantAdminRole 创建租户管理员角色。
func (c *BaseTenantCase) createTenantAdminRole(ctx context.Context, baseTenant *models.BaseTenant) (*models.BaseRole, error) {
	menuIDs, err := c.tenantAdminMenuIDs(ctx)
	if err != nil {
		return nil, errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}

	baseRole := &models.BaseRole{
		TenantID:  baseTenant.ID,
		Name:      baseTenantAdminRoleName,
		Code:      baseTenantAdminRoleCode,
		DataScope: baseTenantDefaultRoleScope,
		Menus:     _string.ConvertInt64ArrayToString(menuIDs),
		Status:    _const.STATUS_ENABLE,
		Remark:    "租户内置管理员角色，不允许修改",
	}

	err = c.baseRoleRepo.Create(ctx, baseRole)
	if err != nil {
		// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return nil, errorsx.UniqueConflict("角色编码重复", "base_role", "code", "unique_base_role").WithCause(err)
		}
		return nil, errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}
	return baseRole, nil
}

// createTenantAdminUser 创建租户管理员账号。
func (c *BaseTenantCase) createTenantAdminUser(ctx context.Context, baseTenant *models.BaseTenant, baseDept *models.BaseDept, baseRole *models.BaseRole) error {
	password, err := crypto.Encrypt(baseTenantAdminPassword)
	if err != nil {
		return errorsx.Internal("初始化租户管理员账号失败").WithCause(err)
	}

	baseUser := &models.BaseUser{
		TenantID: baseTenant.ID,
		UserName: baseTenantAdminUserName,
		NickName: baseTenantAdminNickName,
		RoleID:   baseRole.ID,
		DeptID:   baseDept.ID,
		Phone:    baseTenant.ContactPhone,
		Password: password,
		Gender:   _const.BASE_USER_GENDER_SECRET,
		Status:   _const.STATUS_ENABLE,
		Remark:   "租户默认管理员",
	}
	err = c.baseUserRepo.Create(ctx, baseUser)
	if err != nil {
		// 命中用户账号唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("用户账号重复", "base_user", "user_name", "unique_base_user").WithCause(err)
		}
		return errorsx.Internal("初始化租户管理员账号失败").WithCause(err)
	}
	return nil
}

// tenantAdminMenuIDs 查询租户管理员默认菜单，按固定白名单收敛可见能力。
func (c *BaseTenantCase) tenantAdminMenuIDs(ctx context.Context) ([]int64, error) {
	query := c.baseMenuRepo.Query(ctx).BaseMenu
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.In(baseTenantAdminMenuIDs...)))
	list, err := c.baseMenuRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	existingSet := make(map[int64]struct{}, len(list))
	for _, item := range list {
		existingSet[item.ID] = struct{}{}
	}

	menuIDs := make([]int64, 0, len(baseTenantAdminMenuIDs))
	for _, menuID := range baseTenantAdminMenuIDs {
		// 菜单脚本尚未初始化对应节点时跳过，避免租户创建被历史库数据阻断。
		if _, exists := existingSet[menuID]; !exists {
			continue
		}
		menuIDs = append(menuIDs, menuID)
	}
	return menuIDs, nil
}
