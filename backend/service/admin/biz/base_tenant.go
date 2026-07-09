package biz

import (
	"context"
	"fmt"
	"strconv"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/utils"
)

const (
	baseTenantAdminUserName   = "admin"
	baseTenantAdminNickName   = "管理员"
	baseTenantDefaultDeptName = "默认部门"
	baseTenantDefaultDeptPath = "/0/%d"
	baseTenantDefaultDeptSort = int32(0)
	baseTenantInitialCode     = int64(1000)
	baseTenantMaxCode         = int64(9999)
	baseTenantNumericCodeExpr = "^[0-9]+$"
)

// BaseTenantCase 租户业务实例。
type BaseTenantCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseTenantRepository
	baseDeptRepo   *data.BaseDeptRepository
	baseRoleRepo   *data.BaseRoleRepository
	baseUserRepo   *data.BaseUserRepository
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
	casbinRuleCase *CasbinRuleCase,
) *BaseTenantCase {
	return &BaseTenantCase{
		BaseCase:             baseCase,
		tx:                   tx,
		BaseTenantRepository: baseTenantRepo,
		baseDeptRepo:         baseDeptRepo,
		baseRoleRepo:         baseRoleRepo,
		baseUserRepo:         baseUserRepo,
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

// CreateBaseTenant 创建租户。
func (c *BaseTenantCase) CreateBaseTenant(ctx context.Context, req *adminv1.BaseTenantForm) error {
	baseTenant := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		code, err := c.getNextBaseTenantCode(ctx)
		if err != nil {
			return err
		}

		// 租户编码只允许后端生成，避免客户端绕过前端禁用态传入自定义编码。
		baseTenant.Code = code
		// 未指定状态时，新租户默认启用，避免初始化完成后仍无法登录。
		if baseTenant.Status == 0 {
			baseTenant.Status = _const.STATUS_ENABLE
		}
		err = c.Create(ctx, baseTenant)
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

	baseTenant := c.formMapper.ToEntity(req)
	// 更新租户时沿用数据库中的原始编码，忽略客户端传入的 code。
	baseTenant.Code = oldBaseTenant.Code
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

// getNextBaseTenantCode 获取下一个可用租户编码。
func (c *BaseTenantCase) getNextBaseTenantCode(ctx context.Context) (string, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Unscoped())
	opts = append(opts, repository.Where(query.Code.Regexp(baseTenantNumericCodeExpr)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return "", err
	}

	maxCode := baseTenantInitialCode - 1
	for _, item := range list {
		var code int64
		code, err = strconv.ParseInt(item.Code, 10, 64)
		if err != nil {
			return "", errorsx.Internal("解析租户编码失败").WithCause(err)
		}
		if code > maxCode {
			maxCode = code
		}
	}
	if maxCode >= baseTenantMaxCode {
		return "", errorsx.StateConflict("租户编码已用完", "base_tenant", strconv.FormatInt(maxCode, 10), strconv.FormatInt(baseTenantMaxCode, 10))
	}
	return fmt.Sprintf("%04d", maxCode+1), nil
}

// initTenantDefaults 初始化租户默认组织、角色和管理员账号。
func (c *BaseTenantCase) initTenantDefaults(ctx context.Context, baseTenant *models.BaseTenant) error {
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
		return errorsx.Internal("初始化租户默认部门失败").WithCause(err)
	}

	baseDept.Path = fmt.Sprintf(baseTenantDefaultDeptPath, baseDept.ID)
	err = c.baseDeptRepo.UpdateByID(ctx, baseDept)
	if err != nil {
		return errorsx.Internal("初始化租户默认部门失败").WithCause(err)
	}

	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(databaseGorm.DefaultTenantCode)))
	var defaultTenant *models.BaseTenant
	defaultTenant, err = c.Find(ctx, opts...)
	if err != nil {
		return errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}

	roleQuery := c.baseRoleRepo.Query(ctx).BaseRole
	opts = make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(roleQuery.TenantID.Eq(defaultTenant.ID)))
	opts = append(opts, repository.Where(roleQuery.Code.Eq(_const.BASE_ROLE_CODE_TENANT)))
	var defaultRole *models.BaseRole
	defaultRole, err = c.baseRoleRepo.Find(ctx, opts...)
	if err != nil {
		return errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}

	baseRole := &models.BaseRole{
		TenantID:  baseTenant.ID,
		Name:      defaultRole.Name,
		Code:      defaultRole.Code,
		DataScope: defaultRole.DataScope,
		Menus:     defaultRole.Menus,
		Status:    defaultRole.Status,
		Remark:    defaultRole.Remark,
	}
	err = c.baseRoleRepo.Create(ctx, baseRole)
	if err != nil {
		// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("角色编码重复", "base_role", "code", "unique_base_role").WithCause(err)
		}
		return errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}

	var password string
	password, err = crypto.Encrypt(utils.GetDefaultPassword(baseTenantAdminUserName, baseTenant.ContactPhone))
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
	err = c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	if err != nil {
		return errorsx.Internal("初始化租户管理员角色权限失败").WithCause(err)
	}
	return nil
}
