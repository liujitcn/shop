package biz

import (
	"context"

	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

type BaseUserCase struct {
	*data.BaseUserRepo
}

// NewBaseUserCase 创建基础用户业务实例。
func NewBaseUserCase(baseUserRepo *data.BaseUserRepo) *BaseUserCase {
	return &BaseUserCase{
		BaseUserRepo: baseUserRepo,
	}
}

// FindByUserName 按用户名查询基础用户。
func (c *BaseUserCase) FindByUserName(ctx context.Context, userName string) (*models.BaseUser, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.UserName.Eq(userName)))
	return c.Find(ctx, opts...)
}
