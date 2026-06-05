package biz

import (
	"shop/pkg/gen/data"
)

// BaseRoleCase 处理基础角色业务。
type BaseRoleCase struct {
	*data.BaseRoleRepository
}

// NewBaseRoleCase 创建基础角色业务实例。
func NewBaseRoleCase(
	baseRoleRepo *data.BaseRoleRepository,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseRoleRepository: baseRoleRepo,
	}
}
