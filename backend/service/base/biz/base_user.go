package biz

import (
	"context"

	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

type BaseUserCase struct {
	*data.BaseUserRepository
}

// NewBaseUserCase 创建基础用户业务实例。
func NewBaseUserCase(baseUserRepo *data.BaseUserRepository) *BaseUserCase {
	return &BaseUserCase{
		BaseUserRepository: baseUserRepo,
	}
}

// FindByUserName 按用户名查询基础用户。
func (c *BaseUserCase) FindByUserName(ctx context.Context, userName string) (*models.BaseUser, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserName.Eq(userName)))
	return c.Find(ctx, opts...)
}

// FindDisplayNameByID 按用户编号查询展示名称。
func (c *BaseUserCase) FindDisplayNameByID(ctx context.Context, userID int64) (string, error) {
	user, err := c.FindByID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errorsx.ResourceNotFound("用户不存在")
		}
		return "", err
	}
	if user.NickName != "" {
		return user.NickName, nil
	}
	return user.UserName, nil
}
