package biz

import (
	"context"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
)

// ShopHotCase 热门推荐分组业务处理对象
type ShopHotCase struct {
	*biz.BaseCase
	*data.ShopHotRepo
}

// NewShopHotCase 创建热门推荐分组业务处理对象
func NewShopHotCase(baseCase *biz.BaseCase, shopHotRepo *data.ShopHotRepo) *ShopHotCase {
	return &ShopHotCase{
		BaseCase:    baseCase,
		ShopHotRepo: shopHotRepo,
	}
}

// ListShopHot 查询热门推荐列表
func (c *ShopHotCase) ListShopHot(ctx context.Context) (*app.ListShopHotResponse, error) {
	query := c.Query(ctx).ShopHot
	all, err := c.ShopHotRepo.List(ctx,
		repo.Where(query.Status.Eq(int32(common.Status_ENABLE))),
	)
	if err != nil {
		return nil, err
	}

	list := make([]*app.ShopHot, 0, len(all))
	for _, item := range all {
		list = append(list, c.convertToProto(item))
	}

	return &app.ListShopHotResponse{
		List: list,
	}, nil
}

// 将热门推荐分组模型转换为接口响应
func (c *ShopHotCase) convertToProto(item *models.ShopHot) *app.ShopHot {
	return &app.ShopHot{
		Id:      item.ID,
		Title:   item.Title,
		Desc:    item.Desc,
		Picture: _string.ConvertJsonStringToStringArray(item.Picture),
	}
}

// 按编号查询启用中的热门推荐分组
func (c *ShopHotCase) findById(ctx context.Context, id int64) (*models.ShopHot, error) {
	query := c.Query(ctx).ShopHot
	return c.Find(ctx,
		repo.Where(query.ID.Eq(id)),
		repo.Where(query.Status.Eq(int32(common.Status_ENABLE))),
	)
}
