package biz

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"regexp"
	"strings"
	"time"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/models"
	baseBiz "shop/service/base/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/crypto"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/gorm"
)

const updatePhoneCodeCachePrefix = "admin:update:phone:code:"

var phoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// AuthCase 认证业务实例
type AuthCase struct {
	*biz.BaseCase
	baseUserCase *BaseUserCase
	baseRoleCase *BaseRoleCase
	baseDeptCase *BaseDeptCase
	baseMenuCase *BaseMenuCase
	fileCase     *baseBiz.FileCase
}

// NewAuthCase 创建认证业务实例
func NewAuthCase(
	baseCase *biz.BaseCase,
	baseUserCase *BaseUserCase,
	baseRoleCase *BaseRoleCase,
	baseDeptCase *BaseDeptCase,
	baseMenuCase *BaseMenuCase,
	fileCase *baseBiz.FileCase,
) *AuthCase {
	return &AuthCase{
		BaseCase:     baseCase,
		baseUserCase: baseUserCase,
		baseRoleCase: baseRoleCase,
		baseDeptCase: baseDeptCase,
		baseMenuCase: baseMenuCase,
		fileCase:     fileCase,
	}
}

// GetUserInfo 获取用户信息
func (c *AuthCase) GetUserInfo(ctx context.Context) (*admin.UserInfo, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	// 用户被停用时，不允许继续访问后台信息。
	if baseUser.Status != int32(common.Status_ENABLE) {
		return nil, errors.New("用户状态错误")
	}

	// 查询角色信息
	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindById(ctx, baseUser.RoleID)
	if err != nil {
		return nil, errors.New("角色不存在")
	}

	// 查询部门信息
	var baseDept *models.BaseDept
	baseDept, err = c.baseDeptCase.FindById(ctx, baseUser.DeptID)
	if err != nil {
		return nil, errors.New("部门不存在")
	}

	return &admin.UserInfo{
		UserName: baseUser.UserName,
		NickName: baseUser.NickName,
		Phone:    baseUser.Phone,
		Avatar:   baseUser.Avatar,
		RoleCode: baseRole.Code,
		RoleName: baseRole.Name,
		DeptName: baseDept.Name,
	}, nil
}

// GetUserMenu 获取用户菜单
func (c *AuthCase) GetUserMenu(ctx context.Context) (*admin.TreeRouteResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindById(ctx, authInfo.RoleId)
	if err != nil {
		return nil, errors.New("角色不存在")
	}
	// 角色被停用时，不允许继续获取菜单。
	if baseRole.Status != int32(common.Status_ENABLE) {
		return nil, errors.New("角色状态错误")
	}

	query := c.baseMenuCase.Query(ctx).BaseMenu
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Where(query.Type.In(
		int32(common.BaseMenuType_FOLDER),
		int32(common.BaseMenuType_MENU),
		int32(common.BaseMenuType_EXT_LINK),
	)))
	// 非超级管理员仅允许查看角色菜单里配置过的菜单。
	if baseRole.Code != _const.BaseRoleCode_Super {
		ids := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
		// 角色未配置任何菜单时，直接返回空菜单树。
		if len(ids) == 0 {
			return &admin.TreeRouteResponse{List: []*admin.RouteItem{}}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(ids...)))
	}

	var menuList []*models.BaseMenu
	menuList, err = c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := c.baseMenuCase.buildRouteTree(menuList, 0)
	return &admin.TreeRouteResponse{List: list}, nil
}

// GetUserButton 获取用户按钮
func (c *AuthCase) GetUserButton(ctx context.Context) (*common.StringValues, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	// 用户被停用时，不允许继续获取按钮权限。
	if baseUser.Status != int32(common.Status_ENABLE) {
		return nil, errors.New("用户状态错误")
	}

	// 查询角色信息
	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindById(ctx, baseUser.RoleID)
	if err != nil {
		return nil, errors.New("角色不存在")
	}

	query := c.baseMenuCase.Query(ctx).BaseMenu

	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.Status_ENABLE))))
	opts = append(opts, repo.Where(query.Type.In(int32(common.BaseMenuType_BUTTON))))
	// 非超级管理员仅允许查看角色菜单里配置过的按钮。
	if baseRole.Code != _const.BaseRoleCode_Super {
		ids := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
		// 角色未配置任何按钮时，直接返回空按钮集。
		if len(ids) == 0 {
			return &common.StringValues{}, nil
		}
		opts = append(opts, repo.Where(query.ID.In(ids...)))
	}

	var baseMenus []*models.BaseMenu
	baseMenus, err = c.baseMenuCase.List(ctx, opts...)
	if err != nil {
		return nil, common.ErrorAccessForbidden("用户权限不存在")
	}

	permission := make([]string, 0, len(baseMenus))
	for _, item := range baseMenus {
		permission = append(permission, item.Path)
	}

	return &common.StringValues{
		Value: permission,
	}, nil
}

// GetUserProfile 获取用户资料
func (c *AuthCase) GetUserProfile(ctx context.Context) (*admin.UserProfileForm, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	// 用户被停用时，不允许继续获取个人资料。
	if baseUser.Status != int32(common.Status_ENABLE) {
		return nil, errors.New("用户状态错误")
	}

	var baseRole *models.BaseRole
	baseRole, err = c.baseRoleCase.FindById(ctx, baseUser.RoleID)
	if err != nil {
		return nil, errors.New("角色不存在")
	}

	var baseDept *models.BaseDept
	baseDept, err = c.baseDeptCase.FindById(ctx, baseUser.DeptID)
	if err != nil {
		return nil, errors.New("部门不存在")
	}

	return &admin.UserProfileForm{
		UserName:  baseUser.UserName,
		NickName:  baseUser.NickName,
		Avatar:    baseUser.Avatar,
		Gender:    baseUser.Gender,
		Phone:     baseUser.Phone,
		RoleName:  baseRole.Name,
		DeptName:  baseDept.Name,
		CreatedAt: _time.TimeToTimeString(baseUser.CreatedAt),
	}, nil
}

// UpdateUserProfile 更新用户资料
func (c *AuthCase) UpdateUserProfile(ctx context.Context, req *admin.UserProfileForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var oldBaseUser *models.BaseUser
	oldBaseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return err
	}
	baseUser := models.BaseUser{
		ID:       authInfo.UserId,
		UserName: req.GetUserName(),
		NickName: req.GetNickName(),
		Avatar:   req.GetAvatar(),
		Gender:   req.GetGender(),
	}
	err = c.baseUserCase.UpdateById(ctx, &baseUser)
	if err != nil {
		return err
	}
	// 删除图片
	c.fileCase.DeleteFile(oldBaseUser.Avatar, baseUser.Avatar)

	return nil
}

// SendUpdatePhoneCode 发送更新手机号验证码
func (c *AuthCase) SendUpdatePhoneCode(ctx context.Context, req *admin.SendUpdatePhoneCodeForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// 手机号格式非法时，不继续发送验证码。
	if !phoneRegexp.MatchString(req.GetPhone()) {
		return errors.New("手机号格式错误")
	}

	var userId int64
	userId, err = c.findUserIdByPhone(ctx, req.GetPhone())
	// 手机号占用查询异常时，统一返回发送失败。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("发送验证码失败")
	}
	// 手机号已绑定其他用户时，不允许继续发送验证码。
	if userId > 0 && userId != authInfo.UserId {
		return errors.New("手机号已被占用")
	}

	code := fmt.Sprintf("%06d", rand.IntN(1000000))
	err = sdk.Runtime.GetCache().Set(c.makeUpdatePhoneCodeCacheKey(authInfo.UserId, req.GetPhone()), code, 5*time.Minute)
	if err != nil {
		return errors.New("发送验证码失败")
	}

	// 当前先将验证码写入日志，后续接入短信渠道时替换这里
	log.Infof("send update phone code userId=%d phone=%s code=%s", authInfo.UserId, req.GetPhone(), code)
	return nil
}

// UpdateUserPhone 更新用户手机号
func (c *AuthCase) UpdateUserPhone(ctx context.Context, req *admin.UpdatePhoneForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// 手机号格式非法时，不允许继续修改。
	if !phoneRegexp.MatchString(req.GetPhone()) {
		return errors.New("手机号格式错误")
	}
	// 验证码为空时，不允许继续修改。
	if strings.TrimSpace(req.GetCode()) == "" {
		return errors.New("验证码不能为空")
	}

	cacheKey := c.makeUpdatePhoneCodeCacheKey(authInfo.UserId, req.GetPhone())
	var cacheCode string
	cacheCode, err = sdk.Runtime.GetCache().Get(cacheKey)
	// 验证码不存在或已过期时，直接返回业务错误。
	if err != nil || cacheCode == "" {
		return errors.New("验证码已过期")
	}
	// 验证码不匹配时，直接返回业务错误。
	if cacheCode != req.GetCode() {
		return errors.New("验证码错误")
	}

	var userId int64
	userId, err = c.findUserIdByPhone(ctx, req.GetPhone())
	// 手机号占用查询异常时，统一返回修改失败。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("修改手机号失败")
	}
	// 手机号已绑定其他用户时，不允许继续修改。
	if userId > 0 && userId != authInfo.UserId {
		return errors.New("手机号已被占用")
	}

	err = c.baseUserCase.UpdateById(ctx, &models.BaseUser{
		ID:    authInfo.UserId,
		Phone: req.GetPhone(),
	})
	if err != nil {
		return errors.New("修改手机号失败")
	}

	err = sdk.Runtime.GetCache().Del(cacheKey)
	// 验证码缓存删除失败时，只记录日志不影响主流程。
	if err != nil {
		log.Errorf("删除修改手机号验证码缓存失败")
	}
	return nil
}

// UpdateUserPwd 更新用户密码
func (c *AuthCase) UpdateUserPwd(ctx context.Context, req *admin.UpdatePwdForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// 原密码为空时，不允许继续修改。
	if strings.TrimSpace(req.GetOldPwd()) == "" {
		return errors.New("原密码不能为空")
	}
	// 新密码为空时，不允许继续修改。
	if strings.TrimSpace(req.GetNewPwd()) == "" {
		return errors.New("新密码不能为空")
	}
	// 两次输入密码不一致时，不允许继续修改。
	if req.GetNewPwd() != req.GetConfirmPwd() {
		return errors.New("两次输入的密码不一致")
	}

	var baseUser *models.BaseUser
	baseUser, err = c.baseUserCase.FindById(ctx, authInfo.UserId)
	if err != nil {
		return errors.New("用户不存在")
	}

	err = crypto.Verify(req.GetOldPwd(), baseUser.Password)
	if err != nil {
		return errors.New("原密码错误")
	}

	var encrypted string
	encrypted, err = crypto.Encrypt(req.GetNewPwd())
	if err != nil {
		return errors.New("密码加密失败")
	}

	return c.baseUserCase.UpdateById(ctx, &models.BaseUser{
		ID:       authInfo.UserId,
		Password: encrypted,
	})
}

// makeUpdatePhoneCodeCacheKey 生成更新手机号验证码缓存键
func (c *AuthCase) makeUpdatePhoneCodeCacheKey(userId int64, phone string) string {
	return fmt.Sprintf("%s%d:%s", updatePhoneCodeCachePrefix, userId, phone)
}

// findUserIdByPhone 根据手机号查询用户ID
func (c *AuthCase) findUserIdByPhone(ctx context.Context, phone string) (int64, error) {
	query := c.baseUserCase.Query(ctx).BaseUser
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.Phone.Eq(phone)))
	baseUser, err := c.baseUserCase.Find(ctx, opts...)
	if err != nil {
		return 0, err
	}
	return baseUser.ID, nil
}
