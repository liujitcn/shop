package biz

import (
	"shop/pkg/gen/data"
)

type BaseRoleCase struct {
	*data.BaseRoleRepo
}

// NewBaseRoleCase new a BaseRole use case.
func NewBaseRoleCase(
	baseRoleRepo *data.BaseRoleRepo,
) *BaseRoleCase {
	return &BaseRoleCase{
		BaseRoleRepo: baseRoleRepo,
	}
}
