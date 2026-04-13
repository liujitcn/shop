package biz

import (
	"shop/pkg/gen/data"
)

type BaseDeptCase struct {
	*data.BaseDeptRepo
}

// NewBaseDeptCase 创建基础部门业务实例。
func NewBaseDeptCase(baseDeptRepo *data.BaseDeptRepo) *BaseDeptCase {
	return &BaseDeptCase{
		BaseDeptRepo: baseDeptRepo,
	}
}
