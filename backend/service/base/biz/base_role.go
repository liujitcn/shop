package biz

import (
	"shop/pkg/gen/data"
)

type BaseRoleCase struct {
	*data.BaseRoleRepo
}

// NewBaseRoleCase 创建基础角色业务实例。
func NewBaseRoleCase(
	baseRoleRepo *data.BaseRoleRepo,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseRoleRepo: baseRoleRepo,
	}
}
