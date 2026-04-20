package biz

import (
	"context"
	"encoding/json"
	"time"

	"shop/api/gen/go/app"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	appDto "shop/service/app/dto"

	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRequestCase 推荐请求业务处理对象。
type RecommendRequestCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendRequestRepo
	*data.RecommendRequestItemRepo
}

// NewRecommendRequestCase 创建推荐请求业务处理对象。
func NewRecommendRequestCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
) *RecommendRequestCase {
	return &RecommendRequestCase{
		BaseCase:                 baseCase,
		tx:                       tx,
		RecommendRequestRepo:     recommendRequestRepo,
		RecommendRequestItemRepo: recommendRequestItemRepo,
	}
}

// resolveRecommendRequestId 解析本次推荐请求应使用的请求编号。
func (c *RecommendRequestCase) resolveRecommendRequestId(ctx context.Context, actor *app.RecommendActor, req *app.RecommendGoodsRequest) (int64, error) {
	requestId := req.GetRequestId()
	// 首次请求未携带请求编号时，直接生成新的推荐会话编号。
	if requestId <= 0 {
		return id.GenSnowflakeID(), nil
	}

	query := c.RecommendRequestRepo.Query(ctx).RecommendRequest
	opts := make([]repo.QueryOption, 0, 4)
	opts = append(opts, repo.Where(query.RequestID.Eq(requestId)))
	// 同一个推荐会话允许生成多条请求日志，这里只取最近一条做翻页复用校验。
	opts = append(opts, repo.Order(query.RequestAt.Desc()))
	opts = append(opts, repo.Order(query.ID.Desc()))
	opts = append(opts, repo.Limit(1))
	requestList, err := c.RecommendRequestRepo.List(ctx, opts...)
	if err != nil {
		return 0, errorsx.Internal("查询推荐请求失败").WithCause(err)
	}
	// 历史请求不存在时，不复用客户端传入值，直接生成新的推荐会话编号。
	if len(requestList) == 0 {
		return id.GenSnowflakeID(), nil
	}
	requestModel := requestList[0]

	// 推荐请求主体或场景发生变化时，不允许继续复用旧的推荐会话。
	if requestModel.ActorType != int32(actor.GetActorType()) || requestModel.ActorID != actor.GetActorId() || requestModel.Scene != int32(req.GetScene()) {
		return id.GenSnowflakeID(), nil
	}

	contextRecord := &appDto.RecommendRequestContextRecord{}
	// 历史请求上下文无法解析时，回退为新的推荐会话，避免错误串联翻页请求。
	if requestModel.ContextJSON != "" && json.Unmarshal([]byte(requestModel.ContextJSON), contextRecord) != nil {
		return id.GenSnowflakeID(), nil
	}
	// 锚点商品或订单变化时，当前请求已不属于同一推荐会话。
	if contextRecord.GoodsId != req.GetGoodsId() || contextRecord.OrderId != req.GetOrderId() {
		return id.GenSnowflakeID(), nil
	}
	return requestId, nil
}

// saveRecommendRequest 保存推荐请求主记录与结果明细。
func (c *RecommendRequestCase) saveRecommendRequest(
	ctx context.Context,
	actor *app.RecommendActor,
	requestId int64,
	req *app.RecommendGoodsRequest,
	contextRecord *appDto.RecommendRequestContextRecord,
	goodsList []*app.GoodsInfo,
	total int64,
	pageNum, pageSize int64,
) error {
	contextBytes, err := json.Marshal(contextRecord)
	if err != nil {
		return errorsx.Internal("序列化推荐上下文失败").WithCause(err)
	}

	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		requestAt := time.Now()
		// 无论是否复用同一个 request_id，每次请求都新增一条推荐日志，保留真实分页轨迹。
		createErr := c.RecommendRequestRepo.Create(ctx, &models.RecommendRequest{
			RequestID:   requestId,
			ActorType:   int32(actor.GetActorType()),
			ActorID:     actor.GetActorId(),
			Scene:       int32(req.GetScene()),
			PageNum:     int32(pageNum),
			PageSize:    int32(pageSize),
			Total:       int32(total),
			ContextJSON: string(contextBytes),
			RequestAt:   requestAt,
		})
		if createErr != nil {
			return errorsx.Internal("保存推荐请求失败").WithCause(createErr)
		}

		startPosition := (pageNum - 1) * pageSize
		itemQuery := c.RecommendRequestItemRepo.Query(ctx).RecommendRequestItem
		itemClearOpts := make([]repo.QueryOption, 0, 3)
		itemClearOpts = append(itemClearOpts, repo.Where(itemQuery.RequestID.Eq(requestId)))
		itemClearOpts = append(itemClearOpts, repo.Where(itemQuery.Position.Gte(int32(startPosition))))
		itemClearOpts = append(itemClearOpts, repo.Where(itemQuery.Position.Lt(int32(startPosition+pageSize))))
		clearErr := c.RecommendRequestItemRepo.Delete(ctx, itemClearOpts...)
		if clearErr != nil {
			return errorsx.Internal("清理推荐请求结果失败").WithCause(clearErr)
		}

		itemList := make([]*models.RecommendRequestItem, 0, len(goodsList))
		for idx, item := range goodsList {
			// 推荐结果里商品为空或商品编号非法时，直接跳过无效明细。
			if item == nil || item.GetId() <= 0 {
				continue
			}
			itemList = append(itemList, &models.RecommendRequestItem{
				RequestID: requestId,
				GoodsID:   item.GetId(),
				Position:  int32(startPosition + int64(idx)),
			})
		}
		// 当前页没有有效推荐商品时，无需写入空明细列表。
		if len(itemList) == 0 {
			return nil
		}
		batchErr := c.RecommendRequestItemRepo.BatchCreate(ctx, itemList)
		if batchErr != nil {
			return errorsx.Internal("保存推荐请求结果失败").WithCause(batchErr)
		}
		return nil
	})
}

// listRecommendRequestPositionMap 查询请求明细中的推荐位序号映射。
func (c *RecommendRequestCase) listRecommendRequestPositionMap(ctx context.Context, requestId int64, goodsIds []int64) (map[int64]int32, error) {
	positionMap := make(map[int64]int32)
	// 推荐请求编号为空或商品列表为空时，不存在可回查的位置信息。
	if requestId <= 0 || len(goodsIds) == 0 {
		return positionMap, nil
	}

	query := c.RecommendRequestItemRepo.Query(ctx).RecommendRequestItem
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.RequestID.Eq(requestId)))
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))
	list, err := c.RecommendRequestItemRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	for _, item := range list {
		positionMap[item.GoodsID] = item.Position
	}
	return positionMap, nil
}
