package task

import (
	"context"
	"fmt"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendAggregate "shop/pkg/recommend/offline/aggregate"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendGoodsStatDay 推荐商品日统计任务。
type RecommendGoodsStatDay struct {
	tx                        data.Transaction
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo
	recommendRequestRepo      *data.RecommendRequestRepo
	recommendRequestItemRepo  *data.RecommendRequestItemRepo
	recommendExposureRepo     *data.RecommendExposureRepo
	recommendExposureItemRepo *data.RecommendExposureItemRepo
	recommendGoodsActionRepo  *data.RecommendGoodsActionRepo
	orderGoodsRepo            *data.OrderGoodsRepo
	ctx                       context.Context
}

// NewRecommendGoodsStatDay 创建推荐商品日统计任务实例。
func NewRecommendGoodsStatDay(
	tx data.Transaction,
	recommendGoodsStatDayRepo *data.RecommendGoodsStatDayRepo,
	recommendRequestRepo *data.RecommendRequestRepo,
	recommendRequestItemRepo *data.RecommendRequestItemRepo,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendExposureItemRepo *data.RecommendExposureItemRepo,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	orderGoodsRepo *data.OrderGoodsRepo,
) *RecommendGoodsStatDay {
	return &RecommendGoodsStatDay{
		tx:                        tx,
		recommendGoodsStatDayRepo: recommendGoodsStatDayRepo,
		recommendRequestRepo:      recommendRequestRepo,
		recommendRequestItemRepo:  recommendRequestItemRepo,
		recommendExposureRepo:     recommendExposureRepo,
		recommendExposureItemRepo: recommendExposureItemRepo,
		recommendGoodsActionRepo:  recommendGoodsActionRepo,
		orderGoodsRepo:            orderGoodsRepo,
		ctx:                       context.Background(),
	}
}

// Exec 执行推荐商品日统计。
func (t *RecommendGoodsStatDay) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job RecommendGoodsStatDay Exec %+v", args)

	statTime, err := parseStatDateArg(args["statDate"])
	if err != nil {
		return []string{err.Error()}, err
	}

	statDate := time.Date(statTime.Year(), statTime.Month(), statTime.Day(), 0, 0, 0, 0, statTime.Location())
	startAt := statDate
	endAt := statDate.AddDate(0, 0, 1)

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		statQuery := t.recommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
		// 统计任务按天全量重算，先清掉当天旧数据再回写。
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(statQuery.StatDate.Eq(statDate)))
		err := t.recommendGoodsStatDayRepo.Delete(ctx, opts...)
		if err != nil {
			return err
		}

		requestList, err := t.loadRecommendRequestList(ctx, startAt, endAt)
		if err != nil {
			return err
		}
		requestItemList, err := t.loadRecommendRequestItemList(ctx, requestList)
		if err != nil {
			return err
		}
		exposureList, err := t.loadRecommendExposureList(ctx, startAt, endAt)
		if err != nil {
			return err
		}
		exposureItemList, err := t.loadRecommendExposureItemList(ctx, exposureList)
		if err != nil {
			return err
		}
		actionList, err := t.loadRecommendGoodsActionList(ctx, startAt, endAt)
		if err != nil {
			return err
		}
		orderGoodsList, err := t.loadRecommendOrderGoodsList(ctx, actionList)
		if err != nil {
			return err
		}

		list := recommendAggregate.BuildRecommendGoodsStatDays(
			statDate,
			requestList,
			requestItemList,
			exposureList,
			exposureItemList,
			actionList,
			orderGoodsList,
		)
		// 没有统计结果时只保留清理动作，不再写入空数据。
		if len(list) == 0 {
			return nil
		}
		return t.recommendGoodsStatDayRepo.BatchCreate(ctx, list)
	})
	if err != nil {
		return []string{err.Error()}, err
	}

	return []string{fmt.Sprintf("推荐商品日统计完成: %s", statDate.Format("2006-01-02"))}, nil
}

// loadRecommendRequestList 读取指定时间窗口内的推荐请求主表记录。
func (t *RecommendGoodsStatDay) loadRecommendRequestList(ctx context.Context, startAt, endAt time.Time) ([]*models.RecommendRequest, error) {
	query := t.recommendRequestRepo.Query(ctx).RecommendRequest
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
	return t.recommendRequestRepo.List(ctx, opts...)
}

// loadRecommendRequestItemList 按推荐请求主表批量读取逐商品明细。
func (t *RecommendGoodsStatDay) loadRecommendRequestItemList(ctx context.Context, requestList []*models.RecommendRequest) ([]*models.RecommendRequestItem, error) {
	requestRecordIds := make([]int64, 0, len(requestList))
	for _, item := range requestList {
		// 非法请求主表记录直接跳过，避免污染 item 明细查询条件。
		if item == nil || item.ID <= 0 {
			continue
		}
		requestRecordIds = append(requestRecordIds, item.ID)
	}
	// 当前没有有效请求主记录时，不需要继续查询逐商品明细。
	if len(requestRecordIds) == 0 {
		return []*models.RecommendRequestItem{}, nil
	}

	query := t.recommendRequestItemRepo.Query(ctx).RecommendRequestItem
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.RecommendRequestID.In(requestRecordIds...)))
	return t.recommendRequestItemRepo.List(ctx, opts...)
}

// loadRecommendExposureList 读取指定时间窗口内的推荐曝光主表记录。
func (t *RecommendGoodsStatDay) loadRecommendExposureList(ctx context.Context, startAt, endAt time.Time) ([]*models.RecommendExposure, error) {
	query := t.recommendExposureRepo.Query(ctx).RecommendExposure
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
	return t.recommendExposureRepo.List(ctx, opts...)
}

// loadRecommendExposureItemList 按推荐曝光主表批量读取逐商品明细。
func (t *RecommendGoodsStatDay) loadRecommendExposureItemList(ctx context.Context, exposureList []*models.RecommendExposure) ([]*models.RecommendExposureItem, error) {
	exposureIds := make([]int64, 0, len(exposureList))
	for _, item := range exposureList {
		// 非法曝光主表记录直接跳过，避免污染 item 明细查询条件。
		if item == nil || item.ID <= 0 {
			continue
		}
		exposureIds = append(exposureIds, item.ID)
	}
	// 当前没有有效曝光主记录时，不需要继续查询逐商品明细。
	if len(exposureIds) == 0 {
		return []*models.RecommendExposureItem{}, nil
	}

	query := t.recommendExposureItemRepo.Query(ctx).RecommendExposureItem
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.RecommendExposureID.In(exposureIds...)))
	return t.recommendExposureItemRepo.List(ctx, opts...)
}

// loadRecommendGoodsActionList 读取指定时间窗口内的推荐商品行为事实。
func (t *RecommendGoodsStatDay) loadRecommendGoodsActionList(ctx context.Context, startAt, endAt time.Time) ([]*models.RecommendGoodsAction, error) {
	query := t.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.CreatedAt.Gte(startAt)))
	opts = append(opts, repo.Where(query.CreatedAt.Lt(endAt)))
	return t.recommendGoodsActionRepo.List(ctx, opts...)
}

// loadRecommendOrderGoodsList 按支付行为回查订单商品金额。
func (t *RecommendGoodsStatDay) loadRecommendOrderGoodsList(ctx context.Context, actionList []*models.RecommendGoodsAction) ([]*models.OrderGoods, error) {
	requestIdSet := make(map[string]struct{}, len(actionList))
	for _, item := range actionList {
		// 只有支付事件需要回查订单商品金额。
		if item == nil || item.EventType != int32(common.RecommendGoodsActionType_ORDER_PAY) || item.RequestID == "" {
			continue
		}
		requestIdSet[item.RequestID] = struct{}{}
	}

	requestIds := make([]string, 0, len(requestIdSet))
	for requestId := range requestIdSet {
		requestIds = append(requestIds, requestId)
	}
	// 没有支付请求时，不需要继续查询订单商品金额。
	if len(requestIds) == 0 {
		return []*models.OrderGoods{}, nil
	}

	query := t.orderGoodsRepo.Query(ctx).OrderGoods
	opts := make([]repo.QueryOption, 0, 1)
	opts = append(opts, repo.Where(query.RequestID.In(requestIds...)))
	return t.orderGoodsRepo.List(ctx, opts...)
}
