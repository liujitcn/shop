package biz

import (
	"shop/pkg/gen/data"
)

type BaseDeptCase struct {
	*data.BaseDeptRepository
}

// NewBaseDeptCase 创建基础部门业务实例。
func NewBaseDeptCase(baseDeptRepo *data.BaseDeptRepository) *BaseDeptCase {
	return &BaseDeptCase{
		BaseDeptRepository: baseDeptRepo,
	}
}
