package biz

import (
	"context"

	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/kratos-kit/auth"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
)

// CasbinRuleCase 权限规则业务实例
type CasbinRuleCase struct {
	*data.CasbinRuleRepo
	baseApiCase *BaseApiCase
	authzEngine authzEngine.Engine
}

// NewCasbinRuleCase 创建权限规则业务实例
func NewCasbinRuleCase(
	casbinRuleRepo *data.CasbinRuleRepo,
	baseApiCase *BaseApiCase,
	authzEngine authzEngine.Engine,
) (*CasbinRuleCase, error) {
	return &CasbinRuleCase{
		CasbinRuleRepo: casbinRuleRepo,
		baseApiCase:    baseApiCase,
		authzEngine:    authzEngine,
	}, nil
}

// RebuildPolicyRule 重建内存权限策略。
func (c *CasbinRuleCase) rebuildPolicyRule(ctx context.Context) error {
	policyRule := make([]casbin.PolicyRule, 0)
	// 查询全部api，默认给super 配置
	baseApiList, err := c.baseApiCase.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range baseApiList {
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: "p",
			V0:    _const.BaseRoleCode_Super,
			V1:    item.Operation,
			V2:    string(auth.Action),
			V3:    "*",
		})
	}
	// 查询casbin
	casbinRuleList := make([]*models.CasbinRule, 0)
	casbinRuleList, err = c.List(ctx)
	for _, item := range casbinRuleList {
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: item.Ptype,
			V0:    item.V0,
			V1:    item.V1,
			V2:    item.V2,
			V3:    item.V3,
			V4:    item.V4,
			V5:    item.V5,
		})
	}
	policyMap := make(authzEngine.PolicyMap)
	policyMap["policies"] = policyRule
	roleMap := make(authzEngine.RoleMap)
	return c.authzEngine.SetPolicies(ctx, policyMap, roleMap)
}
