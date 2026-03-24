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
	var baseRoleList []*models.BaseRole
	var err error
	baseRoleList, err = c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, item := range baseRoleList {
		menus := _string.ConvertJsonStringToInt64Array(item.Menus)
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
	var baseRoleList []*models.BaseRole
	var err error
	baseRoleList, err = c.baseRoleRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, item := range baseRoleList {
		oldMenus := _string.ConvertJsonStringToInt64Array(item.Menus)
		newMenus := make([]int64, 0, len(oldMenus))
		for _, menuId := range oldMenus {
			if slices.Contains(menuIds, menuId) {
				continue
			}
			newMenus = append(newMenus, menuId)
		}
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
	baseQuery := c.Query(ctx)

	err := c.Delete(ctx, repo.Where(baseQuery.CasbinRule.V0.Eq(baseRole.Code)))
	if err != nil {
		return err
	}

	menuIds := _string.ConvertJsonStringToInt64Array(baseRole.Menus)
	if len(menuIds) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	var baseMenuList []*models.BaseMenu
	baseMenuList, err = c.baseMenuRepo.ListByIds(ctx, menuIds)
	if err != nil {
		return err
	}

	operations := make([]string, 0)
	for _, item := range baseMenuList {
		operations = append(operations, _string.ConvertJsonStringToStringArray(item.Apis)...)
	}
	if len(operations) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	var allApiList []*models.BaseApi
	allApiList, err = c.baseApiCase.List(ctx)
	if err != nil {
		return err
	}

	casbinRuleList := make([]*models.CasbinRule, 0)
	for _, item := range allApiList {
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
	if len(casbinRuleList) > 0 {
		err = c.BatchCreate(ctx, casbinRuleList)
		if err != nil {
			return err
		}
	}
	return c.RebuildPolicyRule(ctx)
}

// DeleteCasbinRuleByRoleIds 按角色批量删除权限规则
func (c *CasbinRuleCase) DeleteCasbinRuleByRoleIds(ctx context.Context, roleIds []int64) error {
	var baseRoleList []*models.BaseRole
	var err error
	baseRoleList, err = c.baseRoleRepo.ListByIds(ctx, roleIds)
	if err != nil {
		return err
	}

	roleKeys := make([]string, 0, len(baseRoleList))
	for _, item := range baseRoleList {
		roleKeys = append(roleKeys, item.Code)
	}
	if len(roleKeys) == 0 {
		return c.RebuildPolicyRule(ctx)
	}

	baseQuery := c.Query(ctx).CasbinRule
	err = c.Delete(ctx, repo.Where(baseQuery.V0.In(roleKeys...)))
	if err != nil {
		return err
	}
	return c.RebuildPolicyRule(ctx)
}
