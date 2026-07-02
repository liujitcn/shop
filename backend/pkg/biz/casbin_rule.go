package biz

import (
	"context"

	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	"github.com/liujitcn/kratos-kit/auth/authz/engine/casbin"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

// CasbinRuleCase 权限规则业务实例
type CasbinRuleCase struct {
	*data.CasbinRuleRepository
	baseAPICase *BaseAPICase
	authzEngine authzEngine.Engine
}

// NewCasbinRuleCase 创建权限规则业务实例
func NewCasbinRuleCase(
	casbinRuleRepo *data.CasbinRuleRepository,
	baseAPICase *BaseAPICase,
	authzEngine authzEngine.Engine,
) (*CasbinRuleCase, error) {
	return &CasbinRuleCase{
		CasbinRuleRepository: casbinRuleRepo,
		baseAPICase:          baseAPICase,
		authzEngine:          authzEngine,
	}, nil
}

// RebuildPolicyRule 重建内存权限策略。
func (c *CasbinRuleCase) rebuildPolicyRule(ctx context.Context) error {
	policyRule := make([]casbin.PolicyRule, 0)
	// 查询全部 API，默认给 super 配置
	baseAPIList, err := c.baseAPICase.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range baseAPIList {
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: "p",
			V0:    databaseGorm.DefaultTenantCode,
			V1:    _const.BASE_ROLE_CODE_SUPER,
			V2:    item.Operation,
			V3:    item.Method,
			V4:    "*",
		})
	}
	// 查询 casbin
	var casbinRuleList []*models.CasbinRule
	casbinRuleList, err = c.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range casbinRuleList {
		// 旧版本策略缺少租户或项目占位字段时会被 Casbin 识别为 4 段规则，启动阶段直接跳过等待角色权限重建修复。
		if item.Ptype == "" || item.V0 == "" || item.V1 == "" || item.V2 == "" || item.V3 == "" || item.V4 == "" {
			continue
		}
		policyRule = append(policyRule, casbin.PolicyRule{
			PType: item.Ptype,
			V0:    item.V0,
			V1:    item.V1,
			V2:    item.V2,
			V3:    item.V3,
			V4:    item.V4,
		})
	}
	policyMap := make(authzEngine.PolicyMap)
	policyMap["policies"] = policyRule
	roleMap := make(authzEngine.RoleMap)
	return c.authzEngine.SetPolicies(ctx, policyMap, roleMap)
}
