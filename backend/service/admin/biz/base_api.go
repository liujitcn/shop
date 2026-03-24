package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
)

// BaseApiCase 接口业务实例
type BaseApiCase struct {
	*biz.BaseCase
	*data.BaseApiRepo
	mapper *mapper.CopierMapper[admin.BaseApi, models.BaseApi]
}

// NewBaseApiCase 创建接口业务实例
func NewBaseApiCase(baseCase *biz.BaseCase, baseApiRepo *data.BaseApiRepo) *BaseApiCase {
	return &BaseApiCase{
		BaseCase:    baseCase,
		BaseApiRepo: baseApiRepo,
		mapper:      mapper.NewCopierMapper[admin.BaseApi, models.BaseApi](),
	}
}

// ListBaseApi 查询接口列表
func (c *BaseApiCase) ListBaseApi(ctx context.Context) (*admin.ListBaseApiResponse, error) {
	list, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.BaseApi, 0, len(list))
	for _, item := range list {
		baseApi := c.mapper.ToDTO(item)
		resList = append(resList, baseApi)
	}

	return &admin.ListBaseApiResponse{List: resList}, nil
}
