package biz

import (
	"context"
	"errors"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/util"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// UserCollectCase 用户收藏业务处理对象
type UserCollectCase struct {
	*biz.BaseCase
	*data.UserCollectRepo
	goodsCase    *GoodsCase
	goodsSkuCase *GoodsSkuCase
}

// NewUserCollectCase 创建用户收藏业务处理对象
func NewUserCollectCase(
	baseCase *biz.BaseCase,
	userCollectRepo *data.UserCollectRepo,
	goodsInfoCase *GoodsCase,
	goodsSkuCase *GoodsSkuCase,
) *UserCollectCase {
	return &UserCollectCase{
		BaseCase:        baseCase,
		UserCollectRepo: userCollectRepo,
		goodsCase:       goodsInfoCase,
		goodsSkuCase:    goodsSkuCase,
	}
}

// PageUserCollect 查询用户收藏列表
func (c *UserCollectCase) PageUserCollect(ctx context.Context, req *app.PageUserCollectRequest) (*app.PageUserCollectResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := util.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCollect
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.UserID.Eq(authInfo.UserId)))
	page, count, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0)
	for _, info := range page {
		goodsIds = append(goodsIds, info.GoodsID)
	}

	var goodsInfoMap map[int64]*models.Goods
	goodsInfoMap, err = c.goodsCase.mapByGoodsIds(ctx, goodsIds)
	if err != nil {
		return nil, err
	}

	list := make([]*app.UserCollect, 0)
	for _, item := range page {
		goods, ok := goodsInfoMap[item.GoodsID]
		if !ok {
			goods = &models.Goods{}
		}

		price := goods.Price
		if member {
			price = goods.DiscountPrice
		}

		collect := &app.UserCollect{
			Id:        item.ID,
			GoodsId:   item.GoodsID,
			Name:      goods.Name,
			Desc:      goods.Desc,
			Picture:   goods.Picture,
			SaleNum:   goods.InitSaleNum + goods.RealSaleNum,
			Price:     price,
			JoinPrice: item.Price,
		}
		list = append(list, collect)
	}
	return &app.PageUserCollectResponse{
		List:  list,
		Total: int32(count),
	}, nil
}

// GetIsCollect 查询用户是否收藏
func (c *UserCollectCase) GetIsCollect(ctx context.Context, req *app.IsCollectRequest) (bool, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return false, err
	}
	return c.findByUserIdAndGoodsId(ctx, authInfo.UserId, req.GetGoodsId())
}

// CreateUserCollect 创建用户收藏
func (c *UserCollectCase) CreateUserCollect(ctx context.Context, userCollect *app.UserCollectForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	member := util.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCollect
	// 已存在则执行取消收藏，不存在则创建收藏记录
	isCollect, err := c.findByUserIdAndGoodsId(ctx, authInfo.UserId, userCollect.GetGoodsId())
	if err != nil {
		return err
	}
	if !isCollect {
		var goods *models.Goods
		goods, err = c.goodsCase.GoodsRepo.FindById(ctx, userCollect.GetGoodsId())
		if err != nil {
			return err
		}
		price := goods.Price
		if member {
			price = goods.DiscountPrice
		}

		return c.Create(ctx, &models.UserCollect{
			UserID:  authInfo.UserId,
			GoodsID: userCollect.GetGoodsId(),
			Price:   price,
		})
	}

	// 删除
	return c.Delete(ctx,
		repo.Where(query.UserID.Eq(authInfo.UserId)),
		repo.Where(query.GoodsID.Eq(userCollect.GetGoodsId())),
	)
}

// DeleteUserCollect 删除用户收藏
func (c *UserCollectCase) DeleteUserCollect(ctx context.Context, ids string) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCollect
	return c.Delete(ctx,
		repo.Where(query.UserID.Eq(authInfo.UserId)),
		repo.Where(query.ID.In(_string.ConvertStringToInt64Array(ids)...)),
	)
}

// 按用户编号和商品编号判断是否已收藏
func (c *UserCollectCase) findByUserIdAndGoodsId(ctx context.Context, userId, goodsId int64) (bool, error) {
	query := c.Query(ctx).UserCollect
	find, err := c.Find(ctx,
		repo.Where(query.UserID.Eq(userId)),
		repo.Where(query.GoodsID.Eq(goodsId)),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return find != nil && find.ID > 0, nil
}
