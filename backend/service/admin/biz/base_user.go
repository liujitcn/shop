package biz

import (
	"context"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/utils"
)

// BaseUserCase 用户业务实例
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepository
	baseDeptRepo  *data.BaseDeptRepository
	orderInfoRepo *data.OrderInfoRepository
	baseRoleCase  *BaseRoleCase
	baseDeptCase  *BaseDeptCase
	baseMenuCase  *BaseMenuCase
	formMapper    *mapper.CopierMapper[adminv1.BaseUserForm, models.BaseUser]
	mapper        *mapper.CopierMapper[adminv1.BaseUser, models.BaseUser]
}

// NewBaseUserCase 创建用户业务实例
func NewBaseUserCase(
	baseCase *biz.BaseCase,
	baseUserRepo *data.BaseUserRepository,
	baseDeptRepo *data.BaseDeptRepository,
	orderInfoRepo *data.OrderInfoRepository,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	baseMenuCase *BaseMenuCase,
) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:           baseCase,
		BaseUserRepository: baseUserRepo,
		baseDeptRepo:       baseDeptRepo,
		orderInfoRepo:      orderInfoRepo,
		baseRoleCase:       baseRoleCase,
		baseDeptCase:       baseDeptCase,
		baseMenuCase:       baseMenuCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseUserForm, models.BaseUser](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseUser, models.BaseUser](),
	}
}

// OptionBaseUsers 查询用户选项
func (c *BaseUserCase) OptionBaseUsers(ctx context.Context, req *adminv1.OptionBaseUsersRequest) (*commonv1.SelectOptionResponse, error) {
	keyword := req.GetKeyword()
	// 未传关键字时，直接返回空选项集。
	if keyword == "" {
		return &commonv1.SelectOptionResponse{List: []*commonv1.SelectOptionResponse_Option{}}, nil
	}

	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	queryCtx := ctx
	isOrderUserSearch := req.GetTenantId() == 0 && authInfo.TenantCode != databaseGorm.DefaultTenantCode
	// 普通租户需要跨租户读取订单中的商城用户，订单范围在联表条件中单独收敛。
	if isOrderUserSearch {
		unscopedAuthInfo := *authInfo
		unscopedAuthInfo.TenantCode = databaseGorm.DefaultTenantCode
		queryCtx = authnEngine.ContextWithAuthClaims(ctx, unscopedAuthInfo.MakeAuthClaims())
	}

	query := c.Query(queryCtx).BaseUser
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.NickName.Like("%"+keyword+"%")))
	if isOrderUserSearch {
		orderQuery := c.orderInfoRepo.Query(ctx).OrderInfo
		opts = append(opts, repository.Join(orderQuery, query.ID.EqCol(orderQuery.UserID)))
		opts = append(opts, repository.Where(orderQuery.TenantID.Eq(authInfo.TenantId)))
		opts = append(opts, repository.Where(orderQuery.DeletedAt.IsNull()))
		opts = append(opts, repository.Distinct(query.ALL))
	}
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	opts = append(opts, repository.Limit(100))

	var list []*models.BaseUser
	list, err = c.List(queryCtx, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label: item.NickName,
			Value: item.ID,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseUsers 分页查询用户
func (c *BaseUserCase) PageBaseUsers(ctx context.Context, req *adminv1.PageBaseUsersRequest) (*adminv1.PageBaseUsersResponse, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	var err error
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	// 指定部门时，按部门及其子部门范围筛选用户。
	if req.DeptId != nil && req.GetDeptId() > 0 {
		var dept *models.BaseDept
		dept, err = c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
		if err != nil {
			return nil, err
		}
		if req.GetTenantId() > 0 && dept.TenantID != req.GetTenantId() {
			return &adminv1.PageBaseUsersResponse{BaseUsers: []*adminv1.BaseUser{}, Total: 0}, nil
		}

		deptQuery := c.baseDeptRepo.Query(ctx).BaseDept
		deptOpts := make([]repository.QueryOption, 0, 2)
		deptOpts = append(deptOpts, repository.Where(deptQuery.Path.Like(dept.Path+"%")))
		deptOpts = append(deptOpts, repository.Where(deptQuery.TenantID.Eq(dept.TenantID)))
		var deptList []*models.BaseDept
		deptList, err = c.baseDeptRepo.List(ctx, deptOpts...)
		if err != nil {
			return nil, err
		}

		deptIDs := make([]int64, 0, len(deptList))
		for _, item := range deptList {
			deptIDs = append(deptIDs, item.ID)
		}
		// 命中部门集合时，按部门编号范围过滤用户。
		if len(deptIDs) > 0 {
			opts = append(opts, repository.Where(query.DeptID.In(deptIDs...)))
		}
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.Gender != nil {
		opts = append(opts, repository.Where(query.Gender.Eq(int32(req.GetGender()))))
	}
	// 传入用户名关键字时，按用户名模糊匹配。
	if req.GetUserName() != "" {
		opts = append(opts, repository.Where(query.UserName.Like("%"+req.GetUserName()+"%")))
	}
	// 传入昵称关键字时，按昵称模糊匹配。
	if req.GetNickName() != "" {
		opts = append(opts, repository.Where(query.NickName.Like("%"+req.GetNickName()+"%")))
	}
	// 传入手机号关键字时，按手机号模糊匹配。
	if req.GetPhone() != "" {
		opts = append(opts, repository.Where(query.Phone.Like("%"+req.GetPhone()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.BaseUser, 0, len(list))
	for _, item := range list {
		baseUser := c.mapper.ToDTO(item)
		resList = append(resList, baseUser)
	}
	return &adminv1.PageBaseUsersResponse{BaseUsers: resList, Total: int32(total)}, nil
}

// GetBaseUser 获取用户
func (c *BaseUserCase) GetBaseUser(ctx context.Context, id int64) (*adminv1.BaseUserForm, error) {
	baseUser, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseUser), nil
}

// CreateBaseUser 创建用户
func (c *BaseUserCase) CreateBaseUser(ctx context.Context, req *adminv1.BaseUserForm) error {
	baseRole, err := c.baseRoleCase.FindByID(ctx, req.GetRoleId())
	if err != nil {
		return errorsx.Internal("校验用户角色失败").WithCause(err)
	}
	if _const.IsDefaultBaseRole(baseRole.Code) {
		return errorsx.ProtectedResourceConflict("创建用户失败，不能选择内置角色", "base_user")
	}
	var baseDept *models.BaseDept
	baseDept, err = c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
	if err != nil {
		return errorsx.Internal("校验用户部门失败").WithCause(err)
	}
	if baseRole.TenantID != baseDept.TenantID {
		return errorsx.InvalidArgument("用户角色与部门所属租户不一致")
	}
	if req.GetTenantId() > 0 && req.GetTenantId() != baseDept.TenantID {
		return errorsx.InvalidArgument("用户所属租户与部门不一致")
	}

	var passwordStr string
	// 未显式传入密码时，回退到系统默认密码规则。
	if req.GetPwd() == nil {
		passwordStr = utils.GetDefaultPassword(req.GetUserName(), req.GetPhone())
	} else {
		passwordStr, err = utils.DecryptPassword(req.GetPwd(), commonv1.PasswordCryptoScene_CREATE_BASE_USER)
		if err != nil {
			return err
		}
	}

	var password string
	password, err = crypto.Encrypt(passwordStr)
	if err != nil {
		return err
	}
	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = password
	baseUser.TenantID = baseDept.TenantID
	err = c.Create(ctx, baseUser)
	if err != nil {
		// 命中用户账号唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("用户账号重复", "base_user", "user_name", "unique_base_user").WithCause(err)
		}
		return err
	}
	// 用户写库成功后，再异步同步用户画像到推荐系统。
	queue.DispatchRecommendSyncBaseUser(baseUser.ID)
	return nil
}

// UpdateBaseUser 更新用户
func (c *BaseUserCase) UpdateBaseUser(ctx context.Context, req *adminv1.BaseUserForm) error {
	oldBaseUser, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("更新用户失败，用户信息不存在").WithCause(err)
	}
	// 超级管理员账号不允许被修改。
	if oldBaseUser.UserName == _const.BASE_USER_NAME_SUPER {
		return errorsx.PermissionDenied("更新用户失败，不能操作超级管理员")
	}
	var oldBaseRole *models.BaseRole
	oldBaseRole, err = c.baseRoleCase.FindByID(ctx, oldBaseUser.RoleID)
	if err != nil {
		return errorsx.Internal("校验用户角色失败").WithCause(err)
	}
	if !_const.IsDefaultBaseRole(oldBaseRole.Code) {
		var newBaseRole *models.BaseRole
		newBaseRole, err = c.baseRoleCase.FindByID(ctx, req.GetRoleId())
		if err != nil {
			return errorsx.Internal("校验用户角色失败").WithCause(err)
		}
		if _const.IsDefaultBaseRole(newBaseRole.Code) {
			return errorsx.ProtectedResourceConflict("更新用户失败，不能选择内置角色", "base_user")
		}
		if newBaseRole.TenantID != oldBaseUser.TenantID {
			return errorsx.InvalidArgument("用户角色与所属租户不一致")
		}
	}
	var newBaseDept *models.BaseDept
	newBaseDept, err = c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
	if err != nil {
		return errorsx.Internal("校验用户部门失败").WithCause(err)
	}
	if newBaseDept.TenantID != oldBaseUser.TenantID {
		return errorsx.InvalidArgument("用户部门与所属租户不一致")
	}

	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = oldBaseUser.Password
	baseUser.TenantID = oldBaseUser.TenantID
	// 内置角色用户只能改基础资料，角色保持不变。
	if _const.IsDefaultBaseRole(oldBaseRole.Code) {
		baseUser.RoleID = oldBaseUser.RoleID
	}
	err = c.UpdateByID(ctx, baseUser)
	if err != nil {
		// 命中用户账号唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("用户账号重复", "base_user", "user_name", "unique_base_user").WithCause(err)
		}
		return err
	}
	// 用户更新成功后，再按最新数据库快照同步到推荐系统。
	queue.DispatchRecommendSyncBaseUser(baseUser.ID)
	return nil
}

// DeleteBaseUser 删除用户
func (c *BaseUserCase) DeleteBaseUser(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	baseUserList, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	visibleIDs := make([]int64, 0, len(baseUserList))
	for _, baseUser := range baseUserList {
		visibleIDs = append(visibleIDs, baseUser.ID)
		// 超级管理员账号不允许被删除。
		if baseUser.UserName == _const.BASE_USER_NAME_SUPER {
			return errorsx.PermissionDenied("删除用户失败，不能操作超级管理员")
		}
	}
	err = c.DeleteByIDs(ctx, visibleIDs)
	if err != nil {
		return err
	}
	// 用户删除成功后，再异步清理推荐系统中的用户主体。
	queue.DispatchRecommendDeleteBaseUser(visibleIDs)
	return nil
}

// SetBaseUserStatus 设置用户状态
func (c *BaseUserCase) SetBaseUserStatus(ctx context.Context, req *adminv1.SetBaseUserStatusRequest) error {
	baseUser, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("设置状态失败，用户信息不存在").WithCause(err)
	}
	// 超级管理员账号不允许被停用或启用。
	if baseUser.UserName == _const.BASE_USER_NAME_SUPER {
		return errorsx.PermissionDenied("设置状态失败，不能操作超级管理员")
	}
	baseUser.Status = req.GetStatus()
	err = c.UpdateByID(ctx, baseUser)
	if err != nil {
		return err
	}
	// 用户状态变更成功后，再同步最新状态到推荐系统。
	queue.DispatchRecommendSyncBaseUser(baseUser.ID)
	return nil
}

// ResetBaseUserPassword 重置用户密码
func (c *BaseUserCase) ResetBaseUserPassword(ctx context.Context, req *adminv1.ResetBaseUserPasswordRequest) error {
	baseUser, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return errorsx.ResourceNotFound("重置密码失败，用户信息不存在").WithCause(err)
	}
	// 超级管理员账号不允许被重置密码。
	if baseUser.UserName == _const.BASE_USER_NAME_SUPER {
		return errorsx.PermissionDenied("重置密码失败，不能操作超级管理员")
	}

	var passwordStr string
	// 未显式传入密码时，回退到系统默认密码规则。
	if req.GetPwd() == nil {
		passwordStr = utils.GetDefaultPassword(baseUser.UserName, baseUser.Phone)
	} else {
		passwordStr, err = utils.DecryptPassword(req.GetPwd(), commonv1.PasswordCryptoScene_RESET_BASE_USER_PASSWORD)
		if err != nil {
			return err
		}
	}

	var password string
	password, err = crypto.Encrypt(passwordStr)
	if err != nil {
		return err
	}
	return c.UpdateByID(ctx, &models.BaseUser{
		ID:       req.GetId(),
		Password: password,
	})
}
