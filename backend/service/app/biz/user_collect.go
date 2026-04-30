package biz

import (
	"context"
	"time"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/recommend/dto"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	_slice "github.com/liujitcn/go-utils/slice"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// UserCollectCase 用户收藏业务处理对象
type UserCollectCase struct {
	*biz.BaseCase
	*data.UserCollectRepository
	goodsInfoCase *GoodsInfoCase
	goodsSKUCase  *GoodsSKUCase
	mapper        *mapper.CopierMapper[appv1.UserCollect, models.UserCollect]
	goodsMapper   *mapper.CopierMapper[appv1.UserCollect, models.GoodsInfo]
}

// NewUserCollectCase 创建用户收藏业务处理对象
func NewUserCollectCase(
	baseCase *biz.BaseCase,
	userCollectRepo *data.UserCollectRepository,
	goodsInfoCase *GoodsInfoCase,
	goodsSKUCase *GoodsSKUCase,
) *UserCollectCase {
	return &UserCollectCase{
		BaseCase:              baseCase,
		UserCollectRepository: userCollectRepo,
		goodsInfoCase:         goodsInfoCase,
		goodsSKUCase:          goodsSKUCase,
		mapper:                mapper.NewCopierMapper[appv1.UserCollect, models.UserCollect](),
		goodsMapper:           mapper.NewCopierMapper[appv1.UserCollect, models.GoodsInfo](),
	}
}

// PageUserCollects 查询用户收藏列表
func (c *UserCollectCase) PageUserCollects(ctx context.Context, req *appv1.PageUserCollectsRequest) (*appv1.PageUserCollectsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCollect
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var page []*models.UserCollect
	var count int64
	page, count, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	goodsIDs := make([]int64, 0)
	for _, info := range page {
		goodsIDs = append(goodsIDs, info.GoodsID)
	}

	var goodsInfoMap map[int64]*models.GoodsInfo
	goodsInfoMap, err = c.goodsInfoCase.mapByGoodsIDs(ctx, goodsIDs)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.UserCollect, 0)
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
	return &appv1.PageUserCollectsResponse{
		UserCollects: list,
		Total:        int32(count),
	}, nil
}

// GetIsCollect 查询用户是否收藏
func (c *UserCollectCase) GetIsCollect(ctx context.Context, req *appv1.GetIsCollectRequest) (bool, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return false, err
	}
	return c.findByUserIDAndGoodsID(ctx, authInfo.UserId, req.GetGoodsId())
}

// CreateUserCollect 创建用户收藏
func (c *UserCollectCase) CreateUserCollect(ctx context.Context, userCollect *appv1.UserCollectForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	member := utils.IsMemberByAuthInfo(authInfo)
	query := c.Query(ctx).UserCollect
	// 已存在则执行取消收藏，不存在则创建收藏记录
	var isCollect bool
	isCollect, err = c.findByUserIDAndGoodsID(ctx, authInfo.UserId, userCollect.GetGoodsId())
	if err != nil {
		return err
	}
	// 当前未收藏时，按新增收藏路径写入记录。
	if !isCollect {
		recommendContext := userCollect.GetRecommendContext()
		// 推荐上下文缺失时，回退到空上下文，避免收藏接口出现空指针。
		if recommendContext == nil {
			recommendContext = &appv1.RecommendContext{}
		}
		var goodsInfo *models.GoodsInfo
		goodsInfo, err = c.goodsInfoCase.GoodsInfoRepository.FindByID(ctx, userCollect.GetGoodsId())
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
		// 收藏成功后，按后端事实回写推荐收藏事件。
		c.dispatchRecommendCollectEvent(authInfo.UserId, userCollect.GetGoodsId(), recommendContext)
		return nil
	}

	// 删除
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.GoodsID.Eq(userCollect.GetGoodsId())))
	return c.Delete(ctx, opts...)
}

// DeleteUserCollect 删除用户收藏
func (c *UserCollectCase) DeleteUserCollect(ctx context.Context, ids string) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	query := c.Query(ctx).UserCollect
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.ID.In(_string.ConvertStringToInt64Array(ids)...)))
	return c.Delete(ctx, opts...)
}

// 按用户编号和商品编号判断是否已收藏
func (c *UserCollectCase) findByUserIDAndGoodsID(ctx context.Context, userID, goodsID int64) (bool, error) {
	query := c.Query(ctx).UserCollect
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// listGoodsIDsByUserID 查询用户收藏商品 ID 列表。
func (c *UserCollectCase) listGoodsIDsByUserID(ctx context.Context, userID int64, limit int) ([]int64, error) {
	// 用户编号非法时，不存在可用收藏上下文。
	if userID <= 0 {
		return []int64{}, nil
	}

	query := c.Query(ctx).UserCollect
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	// 仅在传入正数限制时，按条数裁剪最近收藏上下文。
	if limit > 0 {
		opts = append(opts, repository.Limit(limit))
	}
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

// dispatchRecommendCollectEvent 根据收藏落库事实回写推荐收藏事件。
func (c *UserCollectCase) dispatchRecommendCollectEvent(userID, goodsID int64, recommendContext *appv1.RecommendContext) {
	// 用户编号或商品编号非法时，无法构建可归因的推荐收藏事件。
	if userID <= 0 || goodsID <= 0 {
		return
	}
	// 收藏请求未携带推荐上下文时，统一回退到空上下文，避免空指针并保持事件结构稳定。
	if recommendContext == nil {
		recommendContext = &appv1.RecommendContext{}
	}

	// 只在收藏记录写库成功后回写推荐收藏事件，确保推荐链路与后端事实一致。
	queue.DispatchRecommendEvent(&dto.RecommendActor{
		ActorType: commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_USER),
		ActorID:   userID,
	}, &appv1.RecommendEventReportRequest{
		EventType: commonv1.RecommendEventType(_const.RECOMMEND_EVENT_TYPE_COLLECT),
		RecommendContext: &appv1.RecommendEventContext{
			Scene:     recommendContext.GetScene(),
			RequestId: recommendContext.GetRequestId(),
		},
		Items: []*appv1.RecommendEventItem{
			{
				GoodsId:  goodsID,
				GoodsNum: 1,
				Position: recommendContext.GetPosition(),
			},
		},
	}, time.Time{})
}
