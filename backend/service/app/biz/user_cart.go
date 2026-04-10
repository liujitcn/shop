package biz

import (
	"context"
	"errors"
	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/app/util"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// UserCartCase 用户购物车业务处理对象
type UserCartCase struct {
	*biz.BaseCase
	*data.UserCartRepo
	goodsInfoCase *GoodsInfoCase
	goodsSkuCase  *GoodsSkuCase
	mapper        *mapper.CopierMapper[app.UserCart, models.UserCart]
}

// NewUserCartCase 创建用户购物车业务处理对象
func NewUserCartCase(
	baseCase *biz.BaseCase,
	userCartRepo *data.UserCartRepo,
	goodsInfoCase *GoodsInfoCase,
	goodsSkuCase *GoodsSkuCase,
) *UserCartCase {
	userCartMapper := mapper.NewCopierMapper[app.UserCart, models.UserCart]()
	return &UserCartCase{
		BaseCase:      baseCase,
		UserCartRepo:  userCartRepo,
		goodsInfoCase: goodsInfoCase,
		goodsSkuCase:  goodsSkuCase,
		mapper:        userCartMapper,
	}
}

// CountUserCart 查询用户购物车数量
func (c *UserCartCase) CountUserCart(ctx context.Context) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	query := c.Query(ctx).UserCart
	return c.Count(ctx,
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
}

// ListUserCart 查询用户购物车列表
func (c *UserCartCase) ListUserCart(ctx context.Context) (*app.ListUserCartResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := util.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCart
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Order(query.CreatedAt.Desc()))
	opts = append(opts, repo.Where(query.UserID.Eq(authInfo.UserId)))
	all, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0)
	skuCodes := make([]string, 0)
	for _, info := range all {
		goodsIds = append(goodsIds, info.GoodsID)
		skuCodes = append(skuCodes, info.SkuCode)
	}

	var goodsInfoMap map[int64]*models.GoodsInfo
	goodsInfoMap, err = c.goodsInfoCase.mapByGoodsIds(ctx, goodsIds)
	if err != nil {
		return nil, err
	}
	var goodsSkuMap map[string]*models.GoodsSku
	goodsSkuMap, err = c.goodsSkuCase.mapBySkuCodes(ctx, skuCodes)
	if err != nil {
		return nil, err
	}

	list := make([]*app.UserCart, 0)
	for _, item := range all {
		sku, ok1 := goodsSkuMap[item.SkuCode]
		if !ok1 {
			sku = &models.GoodsSku{}
		}
		goods, ok2 := goodsInfoMap[item.GoodsID]
		if !ok2 {
			goods = &models.GoodsInfo{}
		}

		picture := goods.Picture
		if len(sku.Picture) > 0 {
			picture = sku.Picture
		}

		price := sku.Price
		if member {
			price = sku.DiscountPrice
		}

		cart := c.mapper.ToDTO(item)
		cart.Picture = picture
		cart.Name = goods.Name
		cart.SpecItem = _string.ConvertJsonStringToStringArray(sku.SpecItem)
		cart.Inventory = sku.Inventory
		cart.Price = price
		cart.JoinPrice = item.Price
		cart.RecommendContext = &app.RecommendContext{
			Scene:     common.RecommendScene(item.Scene),
			RequestId: item.RequestID,
			Position:  item.Position,
		}
		list = append(list, cart)
	}
	return &app.ListUserCartResponse{
		List: list,
	}, nil
}

// CreateUserCart 创建用户购物车
func (c *UserCartCase) CreateUserCart(ctx context.Context, userCart *app.CreateUserCartRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	member := util.IsMemberByAuthInfo(authInfo)
	cartQuery := c.Query(ctx).UserCart
	// 先查同一商品同一规格是否已在购物车中，存在则直接累加数量
	var find *models.UserCart
	find, err = c.Find(ctx,
		repo.Where(cartQuery.UserID.Eq(authInfo.UserId)),
		repo.Where(cartQuery.GoodsID.Eq(userCart.GetGoodsId())),
		repo.Where(cartQuery.SkuCode.Eq(userCart.GetSkuCode())),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			skuQuery := c.goodsSkuCase.Query(ctx).GoodsSku
			var sku *models.GoodsSku
			sku, err = c.goodsSkuCase.Find(ctx,
				repo.Where(skuQuery.SkuCode.Eq(userCart.GetSkuCode())),
			)
			if err != nil {
				return err
			}
			price := sku.Price
			if member {
				price = sku.DiscountPrice
			}

			userCartModel := &models.UserCart{
				UserID:    authInfo.UserId,
				GoodsID:   userCart.GetGoodsId(),
				SkuCode:   userCart.GetSkuCode(),
				Num:       userCart.GetNum(),
				Price:     price,
				IsChecked: true,
			}
			recommendContext := userCart.GetRecommendContext()
			if recommendContext != nil {
				userCartModel.Scene = int32(recommendContext.Scene)
				userCartModel.RequestID = recommendContext.RequestId
				userCartModel.Position = recommendContext.Position
			}
			return c.UserCartRepo.Create(ctx, userCartModel)
		}
		return err
	}

	// 更新
	find.Num += userCart.GetNum()
	recommendContext := userCart.GetRecommendContext()
	if recommendContext != nil {
		if find.Scene == 0 {
			find.Scene = int32(recommendContext.Scene)
		}
		if find.GoodsID == 0 {
			find.RequestID = recommendContext.RequestId
		}
		if find.Position == 0 {
			find.Position = recommendContext.Position
		}
	}
	return c.UserCartRepo.UpdateById(ctx, find)
}

// UpdateUserCart 更新用户购物车
func (c *UserCartCase) UpdateUserCart(ctx context.Context, req *app.UpdateUserCartRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	return c.Update(ctx, &models.UserCart{
		ID:  req.GetId(),
		Num: req.GetNum(),
	},
		repo.Where(query.ID.Eq(req.GetId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
}

// DeleteUserCart 删除用户购物车
func (c *UserCartCase) DeleteUserCart(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	return c.Delete(ctx,
		repo.Where(query.ID.Eq(id)),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
}

// SetUserCartStatus 设置购物车选中状态
func (c *UserCartCase) SetUserCartStatus(ctx context.Context, req *app.SetUserCartStatusRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	return c.Update(ctx, &models.UserCart{
		ID:        req.GetId(),
		IsChecked: req.GetIsChecked(),
	},
		repo.Where(query.ID.Eq(req.GetId())),
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
}

// SelectedUserCart 设置购物车全选状态
func (c *UserCartCase) SelectedUserCart(ctx context.Context, isChecked bool) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	return c.Update(ctx, &models.UserCart{
		IsChecked: isChecked,
	},
		repo.Where(query.UserID.Eq(authInfo.UserId)),
	)
}

// 按用户编号、商品编号和规格编码删除购物车商品
func (c *UserCartCase) deleteByUserIdAndGoodsIdAndSkuCode(ctx context.Context, userId, goodsId int64, skuCode string) error {
	query := c.Query(ctx).UserCart
	return c.Delete(ctx,
		repo.Where(query.UserID.Eq(userId)),
		repo.Where(query.GoodsID.Eq(goodsId)),
		repo.Where(query.SkuCode.Eq(skuCode)),
	)
}
