package biz

import (
	"context"
	"fmt"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/utils"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseUserCase 用户业务实例
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepository
	baseDeptRepo *data.BaseDeptRepository
	baseRoleCase *BaseRoleCase
	baseDeptCase *BaseDeptCase
	baseMenuCase *BaseMenuCase
	formMapper   *mapper.CopierMapper[adminv1.BaseUserForm, models.BaseUser]
	mapper       *mapper.CopierMapper[adminv1.BaseUser, models.BaseUser]
}

// NewBaseUserCase 创建用户业务实例
func NewBaseUserCase(baseCase *biz.BaseCase, baseUserRepo *data.BaseUserRepository, baseDeptRepo *data.BaseDeptRepository, baseRoleCase *BaseRoleCase, baseDeptCase *BaseDeptCase, baseMenuCase *BaseMenuCase,
) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:           baseCase,
		BaseUserRepository: baseUserRepo,
		baseDeptRepo:       baseDeptRepo,
		baseRoleCase:       baseRoleCase,
		baseDeptCase:       baseDeptCase,
		baseMenuCase:       baseMenuCase,
		formMapper:         mapper.NewCopierMapper[adminv1.BaseUserForm, models.BaseUser](),
		mapper:             mapper.NewCopierMapper[adminv1.BaseUser, models.BaseUser](),
	}
}

// OptionBaseUsers 查询用户选项
func (c *BaseUserCase) OptionBaseUsers(ctx context.Context, req *adminv1.OptionBaseUsersRequest) (*commonv1.SelectOptionResponse, error) {
	keyword := strings.TrimSpace(req.GetKeyword())
	// 未传关键字时，直接返回空选项集。
	if keyword == "" {
		return &commonv1.SelectOptionResponse{List: []*commonv1.SelectOptionResponse_Option{}}, nil
	}

	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.NickName.Like("%"+keyword+"%")))

	list, _, err := c.Page(ctx, 1, 100, opts...)
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
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 指定部门时，按部门及其子部门范围筛选用户。
	if req.DeptId != nil && req.GetDeptId() > 0 {
		dept, deptErr := c.baseDeptRepo.FindByID(ctx, req.GetDeptId())
		if deptErr != nil {
			return nil, deptErr
		}

		deptQuery := c.baseDeptRepo.Query(ctx).BaseDept
		deptOpts := make([]repository.QueryOption, 0, 1)
		deptOpts = append(deptOpts, repository.Where(deptQuery.Path.Like(dept.Path+"%")))
		var deptList []*models.BaseDept
		deptList, deptErr = c.baseDeptRepo.List(ctx, deptOpts...)
		if deptErr != nil {
			return nil, deptErr
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
	res := c.formMapper.ToDTO(baseUser)
	return res, nil
}

// CreateBaseUser 创建用户
func (c *BaseUserCase) CreateBaseUser(ctx context.Context, req *adminv1.BaseUserForm) error {
	var passwordStr string
	var err error
	// 未显式传入密码时，回退到系统默认密码规则。
	if req.GetPwd() == nil {
		passwordStr = c.getDefaultPassword(req.GetUserName(), req.GetPhone())
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
	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = oldBaseUser.Password
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
	for _, baseUser := range baseUserList {
		// 超级管理员账号不允许被删除。
		if baseUser.UserName == _const.BASE_USER_NAME_SUPER {
			return errorsx.PermissionDenied("删除用户失败，不能操作超级管理员")
		}
	}
	err = c.DeleteByIDs(ctx, ids)
	if err != nil {
		return err
	}
	// 用户删除成功后，再异步清理推荐系统中的用户主体。
	queue.DispatchRecommendDeleteBaseUser(ids)
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
		passwordStr = c.getDefaultPassword(baseUser.UserName, baseUser.Phone)
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

// getDefaultPassword 生成默认密码
func (c *BaseUserCase) getDefaultPassword(userName, phone string) string {
	prefix := phone
	// 手机号长度充足时，仅截取前 4 位作为默认密码前缀。
	if len(phone) > 4 {
		prefix = phone[:4]
	}
	prefix = fmt.Sprintf("%-4s", prefix)
	return fmt.Sprintf("%s@%s", userName, prefix)
}
