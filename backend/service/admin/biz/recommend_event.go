package biz

import (
	"context"
	"fmt"
	"strconv"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	adminDto "shop/service/admin/dto"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// RecommendEventCase 推荐事件管理业务实例。
type RecommendEventCase struct {
	*biz.BaseCase
	*data.RecommendEventRepository
}

// NewRecommendEventCase 创建推荐事件管理业务实例。
func NewRecommendEventCase(
	baseCase *biz.BaseCase,
	recommendEventRepo *data.RecommendEventRepository,
) *RecommendEventCase {
	return &RecommendEventCase{
		BaseCase:                 baseCase,
		RecommendEventRepository: recommendEventRepo,
	}
}

// ListRecommendRequestEvents 查询推荐请求商品关联事件列表。
func (c *RecommendEventCase) ListRecommendRequestEvents(
	ctx context.Context,
	requestID int64,
	goodsID int64,
	position int32,
) (*adminv1.ListRecommendRequestEventsResponse, error) {
	// 请求编号非法时，无法继续查询推荐事件明细。
	if requestID <= 0 {
		return nil, errorsx.InvalidArgument("推荐请求编号不能为空")
	}
	// 商品编号非法时，无法继续查询推荐事件明细。
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

	query := c.RecommendEventRepository.Query(ctx).RecommendEvent
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.EventAt.Desc()))
	opts = append(opts, repository.Order(query.ID.Desc()))
	opts = append(opts, repository.Where(query.RequestID.Eq(requestID)))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Position.Eq(position)))
	list, err := c.RecommendEventRepository.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.RecommendEvent, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toRecommendEvent(item))
	}
	return &adminv1.ListRecommendRequestEventsResponse{
		RecommendEvents: resList,
		Total:           int32(len(resList)),
	}, nil
}

// toRecommendEvent 转换推荐事件响应数据。
func (c *RecommendEventCase) toRecommendEvent(item *models.RecommendEvent) *adminv1.RecommendEvent {
	// 事件实体为空时，回退到空响应结构，避免事件列表渲染空指针。
	if item == nil {
		return &adminv1.RecommendEvent{}
	}

	return &adminv1.RecommendEvent{
		Id:        item.ID,
		ActorType: commonv1.RecommendActorType(item.ActorType),
		ActorId:   item.ActorID,
		Scene:     commonv1.RecommendScene(item.Scene),
		EventType: commonv1.RecommendEventType(item.EventType),
		GoodsId:   item.GoodsID,
		GoodsNum:  item.GoodsNum,
		RequestId: strconv.FormatInt(item.RequestID, 10),
		Position:  item.Position,
		EventAt:   _time.TimeToTimeString(item.EventAt),
	}
}

// getRecommendEventCountMap 构建推荐商品事件数量映射。
func (c *RecommendEventCase) getRecommendEventCountMap(
	ctx context.Context,
	requestID int64,
	goodsIDs []int64,
	positionList []int32,
) (map[string]int64, error) {
	eventCountMap := make(map[string]int64)
	// 请求编号非法时，不存在可归属的推荐事件。
	if requestID <= 0 {
		return eventCountMap, nil
	}
	// 商品编号列表为空时，无需继续统计事件数量。
	if len(goodsIDs) == 0 {
		return eventCountMap, nil
	}
	// 结果位置列表为空时，无需继续统计事件数量。
	if len(positionList) == 0 {
		return eventCountMap, nil
	}

	rowList := make([]adminDto.RecommendRequestEventCountRow, 0)
	query := c.RecommendEventRepository.Query(ctx).RecommendEvent
	err := query.WithContext(ctx).
		Select(
			query.GoodsID,
			query.Position,
			query.ID.Count().As("event_count"),
		).
		Where(
			query.RequestID.Eq(requestID),
			query.GoodsID.In(goodsIDs...),
			query.Position.In(positionList...),
		).
		Group(query.GoodsID, query.Position).
		Scan(&rowList)
	if err != nil {
		return nil, err
	}

	for _, item := range rowList {
		eventCountMap[c.buildRecommendItemEventKey(item.GoodsID, item.Position)] = item.EventCount
	}
	return eventCountMap, nil
}

// buildRecommendItemEventKey 构建推荐商品事件映射键。
func (c *RecommendEventCase) buildRecommendItemEventKey(goodsID int64, position int32) string {
	return fmt.Sprintf("%d#%d", goodsID, position)
}
