package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repository"
)

// BaseUserCase 基础用户业务处理对象
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepository
}

// NewBaseUserCase 创建基础用户业务处理对象
func NewBaseUserCase(baseCase *biz.BaseCase, baseUserRepo *data.BaseUserRepository) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:           baseCase,
		BaseUserRepository: baseUserRepo,
	}
}

// 按微信唯一标识查询用户
func (c *BaseUserCase) findByOpenID(ctx context.Context, openID string) (*models.BaseUser, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Openid.Eq(openID)))
	return c.Find(ctx, opts...)
}

// 按手机号查询用户
func (c *BaseUserCase) findByPhone(ctx context.Context, phone string) (*models.BaseUser, error) {
	query := c.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Phone.Eq(phone)))
	return c.Find(ctx, opts...)
}
