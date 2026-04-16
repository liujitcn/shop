package task

import (
	"context"
	"fmt"
	"time"

	"shop/pkg/gen/data"
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

	err = t.tx.Transaction(t.ctx, func(ctx context.Context) error {
		statQuery := t.recommendGoodsStatDayRepo.Query(ctx).RecommendGoodsStatDay
		// 统计任务按天全量重算，先清掉当天旧数据再回写。
		opts := make([]repo.QueryOption, 0, 1)
		opts = append(opts, repo.Where(statQuery.StatDate.Eq(statDate)))
		err := t.recommendGoodsStatDayRepo.Delete(ctx, opts...)
		if err != nil {
			return err
		}

		list, err := recommendAggregate.BuildRecommendGoodsStatDays(
			ctx,
			statDate,
			t.recommendRequestRepo,
			t.recommendRequestItemRepo,
			t.recommendExposureRepo,
			t.recommendExposureItemRepo,
			t.recommendGoodsActionRepo,
			t.orderGoodsRepo,
		)
		if err != nil {
			return err
		}
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
