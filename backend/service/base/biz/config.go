package biz

import (
	"context"
	"errors"

	"shop/api/gen/go/base"
	"shop/pkg/gen/data"

	"github.com/liujitcn/gorm-kit/repo"
)

type ConfigCase struct {
	*data.BaseConfigRepo
}

// NewConfigCase new a Config use case.
func NewConfigCase(baseConfigRepo *data.BaseConfigRepo) *ConfigCase {
	return &ConfigCase{
		BaseConfigRepo: baseConfigRepo,
	}
}

func (c *ConfigCase) GetConfig(ctx context.Context, req *base.ConfigRequest) (*base.ConfigResponse, error) {
	if req.GetSite() == 0 {
		return nil, errors.New("位置不能为空")
	}
	query := c.Query(ctx).BaseConfig
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.Site.Eq(int32(req.GetSite()))))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	resData := make([]*base.ConfigResponse_Data, 0)
	for _, item := range list {
		resData = append(resData, &base.ConfigResponse_Data{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	return &base.ConfigResponse{
		Data: resData,
	}, nil
}
