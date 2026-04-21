package biz

import (
	"context"
	"errors"
	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	pkgQueue "shop/pkg/queue"
	"shop/service/app/utils"
	"time"

	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
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
	member := utils.IsMemberByAuthInfo(authInfo)
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
		// 购物车引用的 SKU 已失效时，使用空 SKU 兜底避免列表组装失败。
		if !ok1 {
			sku = &models.GoodsSku{}
		}
		goods, ok2 := goodsInfoMap[item.GoodsID]
		// 购物车引用的商品已失效时，使用空商品信息兜底避免列表组装失败。
		if !ok2 {
			goods = &models.GoodsInfo{}
		}

		picture := goods.Picture
		// SKU 自带图片时，优先使用 SKU 图片展示。
		if len(sku.Picture) > 0 {
			picture = sku.Picture
		}

		price := sku.Price
		// 会员用户优先展示会员价。
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
	member := utils.IsMemberByAuthInfo(authInfo)
	recommendContext := userCart.GetRecommendContext()
	// 加购请求未携带推荐上下文时，统一回退到空上下文，避免空指针并保持事件结构稳定。
	if recommendContext == nil {
		recommendContext = &app.RecommendContext{}
	}
	cartQuery := c.Query(ctx).UserCart
	// 先查同一商品同一规格是否已在购物车中，存在则直接累加数量
	var find *models.UserCart
	opts := make([]repo.QueryOption, 0, 3)
	opts = append(opts, repo.Where(cartQuery.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repo.Where(cartQuery.GoodsID.Eq(userCart.GetGoodsId())))
	opts = append(opts, repo.Where(cartQuery.SkuCode.Eq(userCart.GetSkuCode())))
	find, err = c.Find(ctx, opts...)
	// 同规格购物车记录查询失败时，仅对“未找到”场景继续新增。
	if err != nil {
		// 购物车记录不存在时，按新增购物车逻辑处理。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			skuQuery := c.goodsSkuCase.Query(ctx).GoodsSku
			var sku *models.GoodsSku
			skuOpts := make([]repo.QueryOption, 0, 1)
			skuOpts = append(skuOpts, repo.Where(skuQuery.SkuCode.Eq(userCart.GetSkuCode())))
			sku, err = c.goodsSkuCase.Find(ctx, skuOpts...)
			if err != nil {
				return err
			}
			price := sku.Price
			// 会员用户优先写入会员价。
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
				Scene:     int32(recommendContext.GetScene()),
				RequestID: recommendContext.GetRequestId(),
				Position:  recommendContext.GetPosition(),
			}
			err = c.UserCartRepo.Create(ctx, userCartModel)
			if err != nil {
				return err
			}
			// 新增购物车成功后，按本次加购数量回写推荐加购事件。
			c.dispatchRecommendAddCartEvent(authInfo.UserId, userCart.GetGoodsId(), userCart.GetNum(), recommendContext)
			return nil
		}
		return err
	}

	// 更新
	find.Num += userCart.GetNum()
	// 推荐位序号从 0 开始，不能把 position=0 误判成“缺失”，这里以 request_id 是否缺失作为补齐依据。
	shouldFillPosition := find.RequestID == 0 && recommendContext.GetRequestId() > 0
	// 购物车已有推荐上下文时优先保留，仅在缺失字段上补齐本次上下文。
	if find.Scene == 0 {
		find.Scene = int32(recommendContext.GetScene())
	}
	// 历史购物车未记录请求编号时，补齐本次请求编号。
	if find.RequestID == 0 {
		find.RequestID = recommendContext.GetRequestId()
	}
	// 只有历史记录未绑定过推荐请求时，才补齐本次位置信息，避免把首位推荐的 position=0 错当成空值覆盖掉。
	if shouldFillPosition {
		find.Position = recommendContext.GetPosition()
	}
	err = c.UserCartRepo.UpdateById(ctx, find)
	if err != nil {
		return err
	}
	// 已有购物车累加成功后，仍按本次新增数量回写推荐加购事件。
	c.dispatchRecommendAddCartEvent(authInfo.UserId, userCart.GetGoodsId(), userCart.GetNum(), recommendContext)
	return nil
}

// UpdateUserCart 更新用户购物车
func (c *UserCartCase) UpdateUserCart(ctx context.Context, req *app.UserCartForm) error {
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

// SetUserCartSelection 设置购物车全选状态
func (c *UserCartCase) SetUserCartSelection(ctx context.Context, isChecked bool) error {
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

// listGoodsIdsByUserId 查询指定用户购物车中的商品 ID 列表。
func (c *UserCartCase) listGoodsIdsByUserId(ctx context.Context, userId int64) ([]int64, error) {
	// 未登录用户没有专属购物车，直接返回空集合。
	if userId == 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).UserCart
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIds := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIds = append(goodsIds, item.GoodsID)
	}
	return _slice.Unique(goodsIds), nil
}

// dispatchRecommendAddCartEvent 根据购物车落库事实回写推荐加购事件。
func (c *UserCartCase) dispatchRecommendAddCartEvent(userId, goodsId, goodsNum int64, recommendContext *app.RecommendContext) {
	// 用户编号、商品编号或加购数量非法时，无法构建可归因的推荐加购事件。
	if userId <= 0 || goodsId <= 0 || goodsNum <= 0 {
		return
	}
	// 加购请求未携带推荐上下文时，统一回退到空上下文，避免空指针并保持事件结构稳定。
	if recommendContext == nil {
		recommendContext = &app.RecommendContext{}
	}

	// 只在购物车写库成功后回写推荐加购事件，确保推荐链路与后端事实一致。
	pkgQueue.DispatchRecommendEvent(&app.RecommendActor{
		ActorType: common.RecommendActorType_USER,
		ActorId:   userId,
	}, &app.RecommendEventReportRequest{
		EventType: common.RecommendEventType_ADD_CART,
		RecommendContext: &app.RecommendEventContext{
			Scene:     recommendContext.GetScene(),
			RequestId: recommendContext.GetRequestId(),
		},
		Items: []*app.RecommendEventItem{
			{
				GoodsId:  goodsId,
				GoodsNum: goodsNum,
				Position: recommendContext.GetPosition(),
			},
		},
	}, time.Time{})
}
