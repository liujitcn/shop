package biz

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// BaseUserCase 用户业务实例
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepo
	baseDeptRepo *data.BaseDeptRepo
	baseRoleCase *BaseRoleCase
	baseDeptCase *BaseDeptCase
	baseMenuCase *BaseMenuCase
	formMapper   *mapper.CopierMapper[admin.BaseUserForm, models.BaseUser]
	mapper       *mapper.CopierMapper[admin.BaseUser, models.BaseUser]
}

// NewBaseUserCase 创建用户业务实例
func NewBaseUserCase(baseCase *biz.BaseCase, baseUserRepo *data.BaseUserRepo, baseDeptRepo *data.BaseDeptRepo, baseRoleCase *BaseRoleCase, baseDeptCase *BaseDeptCase, baseMenuCase *BaseMenuCase) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:     baseCase,
		BaseUserRepo: baseUserRepo,
		baseDeptRepo: baseDeptRepo,
		baseRoleCase: baseRoleCase,
		baseDeptCase: baseDeptCase,
		baseMenuCase: baseMenuCase,
		formMapper:   mapper.NewCopierMapper[admin.BaseUserForm, models.BaseUser](),
		mapper:       mapper.NewCopierMapper[admin.BaseUser, models.BaseUser](),
	}
}

// OptionBaseUser 查询用户选项
func (c *BaseUserCase) OptionBaseUser(ctx context.Context, req *admin.OptionBaseUserRequest) (*common.SelectOptionResponse, error) {
	keyword := strings.TrimSpace(req.GetKeyword())
	// 未传关键字时，直接返回空选项集。
	if keyword == "" {
		return &common.SelectOptionResponse{List: []*common.SelectOptionResponse_Option{}}, nil
	}

	query := c.Query(ctx).BaseUser
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.NickName.Like("%"+keyword+"%")))

	list, _, err := c.Page(ctx, 1, 100, opts...)
	if err != nil {
		return nil, err
	}

	options := make([]*common.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &common.SelectOptionResponse_Option{
			Label: item.NickName,
			Value: item.ID,
		})
	}
	return &common.SelectOptionResponse{List: options}, nil
}

// PageBaseUser 分页查询用户
func (c *BaseUserCase) PageBaseUser(ctx context.Context, req *admin.PageBaseUserRequest) (*admin.PageBaseUserResponse, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repo.QueryOption, 0, 6)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	// 指定部门时，按部门及其子部门范围筛选用户。
	if req.DeptId != nil && req.GetDeptId() > 0 {
		dept, err := c.baseDeptRepo.FindById(ctx, req.GetDeptId())
		if err != nil {
			return nil, err
		}

		deptQuery := c.baseDeptRepo.Query(ctx).BaseDept
		deptOpts := make([]repo.QueryOption, 0, 1)
		deptOpts = append(deptOpts, repo.Where(deptQuery.Path.Like(dept.Path+"%")))
		deptList, err := c.baseDeptRepo.List(ctx, deptOpts...)
		if err != nil {
			return nil, err
		}

		deptIds := make([]int64, 0, len(deptList))
		for _, item := range deptList {
			deptIds = append(deptIds, item.ID)
		}
		// 命中部门集合时，按部门编号范围过滤用户。
		if len(deptIds) > 0 {
			opts = append(opts, repo.Where(query.DeptID.In(deptIds...)))
		}
	}
	if req.Status != nil {
		opts = append(opts, repo.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	// 传入用户名关键字时，按用户名模糊匹配。
	if req.GetUserName() != "" {
		opts = append(opts, repo.Where(query.UserName.Like("%"+req.GetUserName()+"%")))
	}
	// 传入昵称关键字时，按昵称模糊匹配。
	if req.GetNickName() != "" {
		opts = append(opts, repo.Where(query.NickName.Like("%"+req.GetNickName()+"%")))
	}
	// 传入手机号关键字时，按手机号模糊匹配。
	if req.GetPhone() != "" {
		opts = append(opts, repo.Where(query.Phone.Like("%"+req.GetPhone()+"%")))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseUser, 0, len(list))
	for _, item := range list {
		baseUser := c.mapper.ToDTO(item)
		resList = append(resList, baseUser)
	}
	return &admin.PageBaseUserResponse{List: resList, Total: int32(total)}, nil
}

// GetBaseUser 获取用户
func (c *BaseUserCase) GetBaseUser(ctx context.Context, id int64) (*admin.BaseUserForm, error) {
	baseUser, err := c.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	res := c.formMapper.ToDTO(baseUser)
	return res, nil
}

// CreateBaseUser 创建用户
func (c *BaseUserCase) CreateBaseUser(ctx context.Context, req *admin.BaseUserForm) error {
	passwordStr := req.GetPwd()
	// 未显式传入密码时，回退到系统默认密码规则。
	if passwordStr == "" {
		passwordStr = c.getDefaultPassword(req.GetUserName(), req.GetPhone())
	}

	password, err := crypto.Encrypt(passwordStr)
	if err != nil {
		return err
	}
	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = password
	return c.Create(ctx, baseUser)
}

// UpdateBaseUser 更新用户
func (c *BaseUserCase) UpdateBaseUser(ctx context.Context, req *admin.BaseUserForm) error {
	oldBaseUser, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return common.ErrorUserNotFound("更新用户失败, 用户信息不存在")
	}
	// 超级管理员账号不允许被修改。
	if oldBaseUser.UserName == _const.BaseUserName_Super {
		return errors.New("更新用户失败，不能操作超级管理员")
	}
	baseUser := c.formMapper.ToEntity(req)
	baseUser.Password = oldBaseUser.Password
	return c.UpdateById(ctx, baseUser)
}

// DeleteBaseUser 删除用户
func (c *BaseUserCase) DeleteBaseUser(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	baseUserList, err := c.ListByIds(ctx, ids)
	if err != nil {
		return err
	}
	for _, baseUser := range baseUserList {
		// 超级管理员账号不允许被删除。
		if baseUser.UserName == _const.BaseUserName_Super {
			return errors.New("删除用户失败，不能操作超级管理员")
		}
	}
	return c.DeleteByIds(ctx, ids)
}

// SetBaseUserStatus 设置用户状态
func (c *BaseUserCase) SetBaseUserStatus(ctx context.Context, req *common.SetStatusRequest) error {
	baseUser, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return common.ErrorUserNotFound("设置状态失败, 用户信息不存在")
	}
	// 超级管理员账号不允许被停用或启用。
	if baseUser.UserName == _const.BaseUserName_Super {
		return errors.New("设置状态失败，不能操作超级管理员")
	}
	return c.UpdateById(ctx, &models.BaseUser{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// ResetBaseUserPwd 重置用户密码
func (c *BaseUserCase) ResetBaseUserPwd(ctx context.Context, req *admin.ResetBaseUserPwdRequest) error {
	baseUser, err := c.FindById(ctx, req.GetId())
	if err != nil {
		return common.ErrorUserNotFound("重置密码失败, 用户信息不存在")
	}
	// 超级管理员账号不允许被重置密码。
	if baseUser.UserName == _const.BaseUserName_Super {
		return errors.New("重置密码失败，不能操作超级管理员")
	}

	passwordStr := req.GetPwd()
	// 未显式传入密码时，回退到系统默认密码规则。
	if passwordStr == "" {
		passwordStr = c.getDefaultPassword(baseUser.UserName, baseUser.Phone)
	}

	password, err := crypto.Encrypt(passwordStr)
	if err != nil {
		return err
	}
	return c.UpdateById(ctx, &models.BaseUser{
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
