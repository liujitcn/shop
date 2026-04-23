package biz

import (
	"context"
	"fmt"
	"strconv"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	adminDto "shop/service/admin/dto"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendEventCase 推荐事件管理业务实例。
type RecommendEventCase struct {
	*biz.BaseCase
	*data.RecommendEventRepo
}

// NewRecommendEventCase 创建推荐事件管理业务实例。
func NewRecommendEventCase(
	baseCase *biz.BaseCase,
	recommendEventRepo *data.RecommendEventRepo,
) *RecommendEventCase {
	return &RecommendEventCase{
		BaseCase:           baseCase,
		RecommendEventRepo: recommendEventRepo,
	}
}

// GetRecommendRequestEvent 查询推荐请求商品关联事件。
func (c *RecommendEventCase) GetRecommendRequestEvent(
	ctx context.Context,
	requestId int64,
	goodsId int64,
	position int32,
) (*admin.GetRecommendRequestEventResponse, error) {
	// 请求编号非法时，无法继续查询推荐事件明细。
	if requestId <= 0 {
		return nil, errorsx.InvalidArgument("推荐请求编号不能为空")
	}
	// 商品编号非法时，无法继续查询推荐事件明细。
	if goodsId <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

	query := c.RecommendEventRepo.Query(ctx).RecommendEvent
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Order(query.EventAt.Desc()))
	opts = append(opts, repo.Order(query.ID.Desc()))
	opts = append(opts, repo.Where(query.RequestID.Eq(requestId)))
	opts = append(opts, repo.Where(query.GoodsID.Eq(goodsId)))
	opts = append(opts, repo.Where(query.Position.Eq(position)))
	list, err := c.RecommendEventRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.RecommendEvent, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toRecommendEvent(item))
	}
	return &admin.GetRecommendRequestEventResponse{
		List:  resList,
		Total: int32(len(resList)),
	}, nil
}

// toRecommendEvent 转换推荐事件响应数据。
func (c *RecommendEventCase) toRecommendEvent(item *models.RecommendEvent) *admin.RecommendEvent {
	// 事件实体为空时，回退到空响应结构，避免事件列表渲染空指针。
	if item == nil {
		return &admin.RecommendEvent{}
	}

	return &admin.RecommendEvent{
		Id:        item.ID,
		ActorType: item.ActorType,
		ActorId:   item.ActorID,
		Scene:     common.RecommendScene(item.Scene),
		EventType: common.RecommendEventType(item.EventType),
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
	requestId int64,
	goodsIds []int64,
	positionList []int32,
) (map[string]int64, error) {
	eventCountMap := make(map[string]int64)
	// 请求编号非法时，不存在可归属的推荐事件。
	if requestId <= 0 {
		return eventCountMap, nil
	}
	// 商品编号列表为空时，无需继续统计事件数量。
	if len(goodsIds) == 0 {
		return eventCountMap, nil
	}
	// 结果位置列表为空时，无需继续统计事件数量。
	if len(positionList) == 0 {
		return eventCountMap, nil
	}

	rowList := make([]adminDto.RecommendRequestEventCountRow, 0)
	err := c.RecommendEventRepo.Query(ctx).RecommendEvent.WithContext(ctx).UnderlyingDB().
		Model(&models.RecommendEvent{}).
		Select("goods_id, position, COUNT(*) AS event_count").
		Where("request_id = ?", requestId).
		Where("goods_id IN ?", goodsIds).
		Where("position IN ?", positionList).
		Group("goods_id, position").
		Scan(&rowList).Error
	if err != nil {
		return nil, err
	}

	for _, item := range rowList {
		eventCountMap[c.buildRecommendItemEventKey(item.GoodsId, item.Position)] = item.EventCount
	}
	return eventCountMap, nil
}

// buildRecommendItemEventKey 构建推荐商品事件映射键。
func (c *RecommendEventCase) buildRecommendItemEventKey(goodsId int64, position int32) string {
	return fmt.Sprintf("%d#%d", goodsId, position)
}
