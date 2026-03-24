package biz

import (
	"shop/pkg/gen/data"
)

type BaseDeptCase struct {
	*data.BaseDeptRepo
}

// NewBaseDeptCase new a BaseDept use case.
func NewBaseDeptCase(baseDeptRepo *data.BaseDeptRepo) *BaseDeptCase {
	return &BaseDeptCase{
		BaseDeptRepo: baseDeptRepo,
	}
}
