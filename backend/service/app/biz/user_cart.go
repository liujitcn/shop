package biz

import (
	"context"
	"errors"
	"time"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/recommend/dto"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// UserCartCase 用户购物车业务处理对象
type UserCartCase struct {
	*biz.BaseCase
	*data.UserCartRepository
	goodsInfoCase *GoodsInfoCase
	goodsSKUCase  *GoodsSKUCase
	mapper        *mapper.CopierMapper[appv1.UserCart, models.UserCart]
}

// NewUserCartCase 创建用户购物车业务处理对象
func NewUserCartCase(
	baseCase *biz.BaseCase,
	userCartRepo *data.UserCartRepository,
	goodsInfoCase *GoodsInfoCase,
	goodsSKUCase *GoodsSKUCase,
) *UserCartCase {
	userCartMapper := mapper.NewCopierMapper[appv1.UserCart, models.UserCart]()
	return &UserCartCase{
		BaseCase:           baseCase,
		UserCartRepository: userCartRepo,
		goodsInfoCase:      goodsInfoCase,
		goodsSKUCase:       goodsSKUCase,
		mapper:             userCartMapper,
	}
}

// CountUserCart 查询用户购物车数量
func (c *UserCartCase) CountUserCart(ctx context.Context) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	return c.Count(ctx, opts...)
}

// ListUserCarts 查询用户购物车列表
func (c *UserCartCase) ListUserCarts(ctx context.Context) (*appv1.ListUserCartsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var all []*models.UserCart
	all, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIDs := make([]int64, 0)
	skuCodes := make([]string, 0)
	for _, info := range all {
		goodsIDs = append(goodsIDs, info.GoodsID)
		skuCodes = append(skuCodes, info.SKUCode)
	}

	var goodsInfoMap map[int64]*models.GoodsInfo
	goodsInfoMap, err = c.goodsInfoCase.mapByGoodsIDs(ctx, goodsIDs)
	if err != nil {
		return nil, err
	}
	var goodsSKUMap map[string]*models.GoodsSKU
	goodsSKUMap, err = c.goodsSKUCase.mapBySKUCodes(ctx, skuCodes)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.UserCart, 0)
	for _, item := range all {
		sku, ok1 := goodsSKUMap[item.SKUCode]
		// 购物车引用的 SKU 已失效时，使用空 SKU 兜底避免列表组装失败。
		if !ok1 {
			sku = &models.GoodsSKU{}
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
		cart.RecommendContext = &appv1.RecommendContext{
			Scene:     commonv1.RecommendScene(item.Scene),
			RequestId: item.RequestID,
			Position:  item.Position,
		}
		list = append(list, cart)
	}
	return &appv1.ListUserCartsResponse{
		UserCarts: list,
	}, nil
}

// CreateUserCart 创建用户购物车
func (c *UserCartCase) CreateUserCart(ctx context.Context, userCart *appv1.CreateUserCartRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	// 购物车数量非法时，直接拦截当前请求。
	if userCart.GetNum() <= 0 {
		return errorsx.InvalidArgument("商品购买数量必须大于0")
	}
	recommendContext := userCart.GetRecommendContext()
	// 加购请求未携带推荐上下文时，统一回退到空上下文，避免空指针并保持事件结构稳定。
	if recommendContext == nil {
		recommendContext = &appv1.RecommendContext{}
	}
	query := c.Query(ctx).UserCart
	// 先查同一商品同一规格是否已在购物车中，存在则直接累加数量
	var find *models.UserCart
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.GoodsID.Eq(userCart.GetGoodsId())))
	opts = append(opts, repository.Where(query.SKUCode.Eq(userCart.GetSkuCode())))
	find, err = c.Find(ctx, opts...)
	// 同规格购物车记录查询失败时，仅对“未找到”场景继续新增。
	if err != nil {
		// 购物车记录不存在时，按新增购物车逻辑处理。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var sku *models.GoodsSKU
			sku, err = c.goodsSKUCase.findByGoodsIDAndSKUCode(ctx, userCart.GetGoodsId(), userCart.GetSkuCode())
			if err != nil {
				return err
			}
			// 新增购物车前，先校验本次数量不超过当前规格库存。
			err = c.goodsSKUCase.ensureEnoughInventory(sku, userCart.GetNum())
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
				SKUCode:   userCart.GetSkuCode(),
				Num:       userCart.GetNum(),
				Price:     price,
				IsChecked: true,
				Scene:     int32(recommendContext.GetScene()),
				RequestID: recommendContext.GetRequestId(),
				Position:  recommendContext.GetPosition(),
			}
			err = c.UserCartRepository.Create(ctx, userCartModel)
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
	var sku *models.GoodsSKU
	sku, err = c.goodsSKUCase.findByGoodsIDAndSKUCode(ctx, userCart.GetGoodsId(), userCart.GetSkuCode())
	if err != nil {
		return err
	}
	nextNum := find.Num + userCart.GetNum()
	// 已有购物车累加数量前，先校验累加后的总数量不超过库存。
	err = c.goodsSKUCase.ensureEnoughInventory(sku, nextNum)
	if err != nil {
		return err
	}
	find.Num = nextNum
	// 购物车已有推荐上下文时优先保留，仅在缺失字段上补齐本次上下文。
	if find.Scene == 0 {
		find.Scene = int32(recommendContext.GetScene())
	}
	// 历史购物车未记录请求编号时，补齐本次请求编号。
	if find.RequestID == 0 {
		find.RequestID = recommendContext.GetRequestId()
	}
	// 历史购物车未记录位置信息时，补齐本次位置信息。
	if find.Position == 0 {
		find.Position = recommendContext.GetPosition()
	}
	err = c.UserCartRepository.UpdateByID(ctx, find)
	if err != nil {
		return err
	}
	// 已有购物车累加成功后，仍按本次新增数量回写推荐加购事件。
	c.dispatchRecommendAddCartEvent(authInfo.UserId, userCart.GetGoodsId(), userCart.GetNum(), recommendContext)
	return nil
}

// UpdateUserCart 更新用户购物车
func (c *UserCartCase) UpdateUserCart(ctx context.Context, req *appv1.UserCartForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	// 购物车数量非法时，直接拦截当前请求。
	if req.GetNum() <= 0 {
		return errorsx.InvalidArgument("商品购买数量必须大于0")
	}
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(req.GetId())))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var userCart *models.UserCart
	userCart, err = c.Find(ctx, opts...)
	if err != nil {
		return err
	}
	var goodsSKU *models.GoodsSKU
	goodsSKU, err = c.goodsSKUCase.findByGoodsIDAndSKUCode(ctx, userCart.GoodsID, userCart.SKUCode)
	if err != nil {
		return err
	}
	// 手动改数量前，先校验目标数量不超过当前库存。
	err = c.goodsSKUCase.ensureEnoughInventory(goodsSKU, req.GetNum())
	if err != nil {
		return err
	}
	return c.Update(ctx, &models.UserCart{
		ID:  req.GetId(),
		Num: req.GetNum(),
	}, opts...)
}

// DeleteUserCart 删除用户购物车
func (c *UserCartCase) DeleteUserCart(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	return c.Delete(ctx, opts...)
}

// SetUserCartStatus 设置购物车选中状态
func (c *UserCartCase) SetUserCartStatus(ctx context.Context, req *appv1.SetUserCartStatusRequest) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(req.GetId())))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	return c.Update(ctx, &models.UserCart{
		ID:        req.GetId(),
		IsChecked: req.GetIsChecked(),
	}, opts...)
}

// SetUserCartSelection 设置购物车全选状态
func (c *UserCartCase) SetUserCartSelection(ctx context.Context, isChecked bool) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	return c.Update(ctx, &models.UserCart{
		IsChecked: isChecked,
	}, opts...)
}

// 按用户编号、商品编号和规格编码删除购物车商品
func (c *UserCartCase) deleteByUserIDAndGoodsIDAndSKUCode(ctx context.Context, userID, goodsID int64, skuCode string) error {
	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.SKUCode.Eq(skuCode)))
	return c.Delete(ctx, opts...)
}

// listGoodsIDsByUserID 查询指定用户购物车中的商品 ID 列表。
func (c *UserCartCase) listGoodsIDsByUserID(ctx context.Context, userID int64) ([]int64, error) {
	// 未登录用户没有专属购物车，直接返回空集合。
	if userID == 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).UserCart
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	goodsIDs := make([]int64, 0, len(list))
	for _, item := range list {
		goodsIDs = append(goodsIDs, item.GoodsID)
	}
	return _slice.Unique(goodsIDs), nil
}

// dispatchRecommendAddCartEvent 根据购物车落库事实回写推荐加购事件。
func (c *UserCartCase) dispatchRecommendAddCartEvent(userID, goodsID, goodsNum int64, recommendContext *appv1.RecommendContext) {
	// 用户编号、商品编号或加购数量非法时，无法构建可归因的推荐加购事件。
	if userID <= 0 || goodsID <= 0 || goodsNum <= 0 {
		return
	}
	// 加购请求未携带推荐上下文时，统一回退到空上下文，避免空指针并保持事件结构稳定。
	if recommendContext == nil {
		recommendContext = &appv1.RecommendContext{}
	}

	// 只在购物车写库成功后回写推荐加购事件，确保推荐链路与后端事实一致。
	queue.DispatchRecommendEvent(&dto.RecommendActor{
		ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER),
		ActorID:   userID,
	}, &appv1.RecommendEventReportRequest{
		EventType: commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_ADD_CART),
		RecommendContext: &appv1.RecommendEventContext{
			Scene:     recommendContext.GetScene(),
			RequestId: recommendContext.GetRequestId(),
		},
		Items: []*appv1.RecommendEventItem{
			{
				GoodsId:  goodsID,
				GoodsNum: goodsNum,
				Position: recommendContext.GetPosition(),
			},
		},
	}, time.Time{})
}
