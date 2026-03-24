package biz

import (
	"context"

	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/gorm-kit/repo"
)

// BaseUserCase 基础用户业务处理对象
type BaseUserCase struct {
	*biz.BaseCase
	*data.BaseUserRepo
}

// NewBaseUserCase 创建基础用户业务处理对象
func NewBaseUserCase(baseCase *biz.BaseCase, baseUserRepo *data.BaseUserRepo) *BaseUserCase {
	return &BaseUserCase{
		BaseCase:     baseCase,
		BaseUserRepo: baseUserRepo,
	}
}

// 按微信唯一标识查询用户
func (c *BaseUserCase) findByOpenid(ctx context.Context, openid string) (*models.BaseUser, error) {
	query := c.Query(ctx).BaseUser
	return c.Find(ctx,
		repo.Where(query.Openid.Eq(openid)),
	)
}

// 按手机号查询用户
func (c *BaseUserCase) findByPhone(ctx context.Context, phone string) (*models.BaseUser, error) {
	query := c.Query(ctx).BaseUser
	return c.Find(ctx,
		repo.Where(query.Phone.Eq(phone)),
	)
}
