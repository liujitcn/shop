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

// NewBaseUserCase new a BaseUser use case.
func NewBaseUserCase(baseUserRepo *data.BaseUserRepo) *BaseUserCase {
	return &BaseUserCase{
		BaseUserRepo: baseUserRepo,
	}
}

func (c *BaseUserCase) FindByUserName(ctx context.Context, userName string) (*models.BaseUser, error) {
	return c.Find(ctx,
		repo.Where(c.Query(ctx).BaseUser.UserName.Eq(userName)),
	)
}
