package biz

import (
	"context"
	"slices"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	"github.com/liujitcn/kratos-kit/auth"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
)

// CasbinRuleCase 权限规则业务实例
type CasbinRuleCase struct {
	*biz.BaseCase
	*data.CasbinRuleRepo
	baseMenuRepo *data.BaseMenuRepo
	baseRoleRepo *data.BaseRoleRepo
	baseApiCase  *BaseApiCase
	authzEngine  authzEngine.Engine
}

// NewCasbinRuleCase 创建权限规则业务实例
func NewCasbinRuleCase(baseCase *biz.BaseCase, casbinRuleRepo *data.CasbinRuleRepo, baseMenuRepo *data.BaseMenuRepo, baseRoleRepo *data.BaseRoleRepo, baseApiCase *BaseApiCase, authzEngine authzEngine.Engine) (*CasbinRuleCase, error) {
	return &CasbinRuleCase{
		BaseCase:       baseCase,
		CasbinRuleRepo: casbinRuleRepo,
		baseMenuRepo:   baseMenuRepo,
		baseRoleRepo:   baseRoleRepo,
		baseApiCase:    baseApiCase,
		authzEngine:    authzEngine,
	}, nil
}

// RebuildCasbinRuleByMenuId 按菜单重建角色权限
func (c *CasbinRuleCase) RebuildCasbinRuleByMenuId(ctx context.Context, menuId int64) error {
	baseRoleList, err := c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, item := range baseRoleList {
		menus := _string.ConvertJsonStringToInt64Array(item.Menus)
		// 当前角色未配置目标菜单时，无需重建该角色权限。
		if !slices.Contains(menus, menuId) {
			continue
		}
		err = c.RebuildCasbinRuleByRole(ctx, item)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// DeleteCasbinRuleByMenuIds 按菜单批量删除角色权限
func (c *CasbinRuleCase) DeleteCasbinRuleByMenuIds(ctx context.Context, menuIds []int64) error {
	baseRoleList, err := c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, item := range baseRoleList {
		oldMenus := _string.ConvertJsonStringToInt64Array(item.Menus)
		newMenus := make([]int64, 0, len(oldMenus))
		for _, menuId := range oldMenus {
			// 命中待删除菜单时，从新的菜单集合中剔除该菜单。
			if slices.Contains(menuIds, menuId) {
				continue
			}
			newMenus = append(newMenus, menuId)
		}
		// 菜单集合未发生变化时，无需重建该角色权限。
		if len(oldMenus) == len(newMenus) {
			continue
		}
		err = c.RebuildCasbinRuleByRole(ctx, item)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// RebuildCasbinRuleByRole 按角色重建权限规则
func (c *CasbinRuleCase) RebuildCasbinRuleByRole(ctx context.Context, baseRole *models.BaseRole) error {
	query := c.Query(ctx).CasbinRule
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.V0.Eq(baseRole.Code)))
	rebuildErr := c.Delete(ctx, opts...)
	if rebuildErr != nil {
		return rebuildErr
	}

	menuIds := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
	// 角色未配置菜单时，只需要刷新内存权限策略。
	if len(menuIds) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	var baseMenuList []*models.BaseMenu
	baseMenuList, rebuildErr = c.baseMenuRepo.ListByIds(ctx, menuIds)
	if rebuildErr != nil {
		return rebuildErr
	}

	operations := make([]string, 0)
	for _, item := range baseMenuList {
		operations = append(operations, _string.ConvertJsonStringToStringArray(item.API)...)
	}
	// 菜单未配置接口权限时，只需要刷新内存权限策略。
	if len(operations) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	var allApiList []*models.BaseApi
	allApiList, rebuildErr = c.baseApiCase.List(ctx)
	if rebuildErr != nil {
		return rebuildErr
	}

	casbinRuleList := make([]*models.CasbinRule, 0)
	for _, item := range allApiList {
		// 非当前角色菜单命中的接口不参与规则生成。
		if !slices.Contains(operations, item.Operation) {
			continue
		}
		casbinRuleList = append(casbinRuleList, &models.CasbinRule{
			Ptype: "p",
			V0:    baseRole.Code,
			V1:    item.Operation,
			V2:    string(auth.Action),
			V3:    "*",
		})
	}
	// 命中接口规则时，批量写入角色权限规则。
	if len(casbinRuleList) > 0 {
		rebuildErr = c.BatchCreate(ctx, casbinRuleList)
		if rebuildErr != nil {
			return rebuildErr
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// DeleteCasbinRuleByRoleIds 按角色批量删除权限规则
func (c *CasbinRuleCase) DeleteCasbinRuleByRoleIds(ctx context.Context, roleIds []int64) error {
	baseRoleList, err := c.baseRoleRepo.ListByIds(ctx, roleIds)
	if err != nil {
		return err
	}

	roleKeys := make([]string, 0, len(baseRoleList))
	for _, item := range baseRoleList {
		roleKeys = append(roleKeys, item.Code)
	}
	// 角色集合为空时，只需要刷新内存权限策略。
	if len(roleKeys) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	query := c.Query(ctx).CasbinRule
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.V0.In(roleKeys...)))
	err = c.Delete(ctx, opts...)
	if err != nil {
		return err
	}
	return c.RebuildPolicyRule(ctx)
}
