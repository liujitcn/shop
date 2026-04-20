package biz

import (
	"context"
	"errors"
	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	pkgUtils "shop/pkg/utils"
	"shop/service/app/utils"
	"time"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/gorm"
)

// UserCollectCase 用户收藏业务处理对象
type UserCollectCase struct {
	*biz.BaseCase
	*data.UserCollectRepo
	goodsInfoCase *GoodsInfoCase
	goodsSkuCase  *GoodsSkuCase
	mapper        *mapper.CopierMapper[app.UserCollect, models.UserCollect]
	goodsMapper   *mapper.CopierMapper[app.UserCollect, models.GoodsInfo]
}

// NewUserCollectCase 创建用户收藏业务处理对象
func NewUserCollectCase(
	baseCase *biz.BaseCase,
	userCollectRepo *data.UserCollectRepo,
	goodsInfoCase *GoodsInfoCase,
	goodsSkuCase *GoodsSkuCase,
) *UserCollectCase {
	return &UserCollectCase{
		BaseCase:        baseCase,
		UserCollectRepo: userCollectRepo,
		goodsInfoCase:   goodsInfoCase,
		goodsSkuCase:    goodsSkuCase,
		mapper:          mapper.NewCopierMapper[app.UserCollect, models.UserCollect](),
		goodsMapper:     mapper.NewCopierMapper[app.UserCollect, models.GoodsInfo](),
	}
}

// PageUserCollect 查询用户收藏列表
func (c *UserCollectCase) PageUserCollect(ctx context.Context, req *app.PageUserCollectRequest) (*app.PageUserCollectResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
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

	var goodsInfoMap map[int64]*models.GoodsInfo
	goodsInfoMap, err = c.goodsInfoCase.mapByGoodsIds(ctx, goodsIds)
	if err != nil {
		return nil, err
	}

	list := make([]*app.UserCollect, 0)
	for _, item := range page {
		goodsInfo, ok := goodsInfoMap[item.GoodsID]
		// 收藏商品已失效时，使用空商品信息兜底避免列表组装失败。
		if !ok {
			goodsInfo = &models.GoodsInfo{}
		}

		price := goodsInfo.Price
		// 会员用户优先展示会员价。
		if member {
			price = goodsInfo.DiscountPrice
		}

		collect := c.mapper.ToDTO(item)
		goodsCollect := c.goodsMapper.ToDTO(goodsInfo)
		collect.Name = goodsCollect.Name
		collect.Desc = goodsCollect.Desc
		collect.Picture = goodsCollect.Picture
		collect.SaleNum = goodsInfo.InitSaleNum + goodsInfo.RealSaleNum
		collect.Price = price
		collect.JoinPrice = item.Price
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
	member := utils.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCollect
	// 已存在则执行取消收藏，不存在则创建收藏记录
	var isCollect bool
	isCollect, err = c.findByUserIdAndGoodsId(ctx, authInfo.UserId, userCollect.GetGoodsId())
	if err != nil {
		return err
	}
	// 当前未收藏时，按新增收藏路径写入记录。
	if !isCollect {
		recommendContext := userCollect.GetRecommendContext()
		// 推荐上下文缺失时，回退到空上下文，避免收藏接口出现空指针。
		if recommendContext == nil {
			recommendContext = &app.RecommendContext{}
		}
		var goodsInfo *models.GoodsInfo
		goodsInfo, err = c.goodsInfoCase.GoodsInfoRepo.FindById(ctx, userCollect.GetGoodsId())
		if err != nil {
			return err
		}
		price := goodsInfo.Price
		// 会员用户收藏商品时，优先记录会员价快照。
		if member {
			price = goodsInfo.DiscountPrice
		}

		err = c.Create(ctx, &models.UserCollect{
			UserID:    authInfo.UserId,
			GoodsID:   userCollect.GetGoodsId(),
			Price:     price,
			Scene:     int32(recommendContext.GetScene()),
			RequestID: recommendContext.GetRequestId(),
			Position:  recommendContext.GetPosition(),
		})
		if err != nil {
			return err
		}
		// 收藏成功后，按后端事实回写推荐收藏行为。
		c.dispatchRecommendGoodsActionEvent(authInfo.UserId, userCollect.GetGoodsId(), recommendContext)
		return nil
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
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.UserID.Eq(userId)))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	find, err := c.Find(ctx, opts...)
	if err != nil {
		// 记录不存在时，明确返回“未收藏”而不是错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return find != nil && find.ID > 0, nil
}

// dispatchRecommendGoodsActionEvent 根据收藏落库事实回写推荐收藏行为。
func (c *UserCollectCase) dispatchRecommendGoodsActionEvent(userId, goodsId int64, recommendContext *app.RecommendContext) {
	// 用户编号或商品编号非法时，无法构建可归因的推荐收藏行为。
	if userId <= 0 || goodsId <= 0 {
		return
	}
	// 收藏请求未携带推荐上下文时，统一回退到空上下文，避免空指针并保持事件结构稳定。
	if recommendContext == nil {
		recommendContext = &app.RecommendContext{}
	}

	// 只在收藏记录写库成功后回写推荐收藏行为，确保推荐链路与后端事实一致。
	pkgUtils.DispatchRecommendGoodsActionEvent(&app.RecommendActor{
		ActorType: common.RecommendActorType_USER,
		ActorId:   userId,
	}, &app.RecommendGoodsActionReportRequest{
		EventType: common.RecommendGoodsActionType_COLLECT,
		GoodsItems: []*app.RecommendGoodsActionItem{
			{
				GoodsId:          goodsId,
				GoodsNum:         1,
				RecommendContext: recommendContext,
			},
		},
	}, time.Time{})
}
