package biz

import (
	"shop/pkg/biz"
	"shop/pkg/gen/data"
)

// BaseDeptCase 基础部门业务处理对象
type BaseDeptCase struct {
	*biz.BaseCase
	*data.BaseDeptRepo
}

// NewBaseDeptCase 创建基础部门业务处理对象
func NewBaseDeptCase(baseCase *biz.BaseCase, baseDeptRepo *data.BaseDeptRepo) *BaseDeptCase {
	return &BaseDeptCase{
		BaseCase:     baseCase,
		BaseDeptRepo: baseDeptRepo,
	}
}
