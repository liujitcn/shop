package biz

import (
	"context"
	"encoding/json"
	"time"

	"shop/api/gen/go/app"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendevent "shop/pkg/recommend/event"
	"shop/pkg/utils"
	appdto "shop/service/app/dto"

	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// recommendExposureEvent 推荐曝光队列事件。
type recommendExposureEvent struct {
	Exposure *models.RecommendExposure `json:"exposure"`
	GoodsIds []int64                   `json:"goodsIds"`
}

// RecommendExposureCase 推荐曝光业务处理对象。
type RecommendExposureCase struct {
	*biz.BaseCase
	*data.RecommendExposureRepo
	recommendExposureItemCase *RecommendExposureItemCase
	recommendGoodsActionRepo  *data.RecommendGoodsActionRepo
}

// NewRecommendExposureCase 创建推荐曝光业务处理对象。
func NewRecommendExposureCase(
	baseCase *biz.BaseCase,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendExposureItemCase *RecommendExposureItemCase,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
) *RecommendExposureCase {
	recommendExposureCase := &RecommendExposureCase{
		BaseCase:                  baseCase,
		RecommendExposureRepo:     recommendExposureRepo,
		recommendExposureItemCase: recommendExposureItemCase,
		recommendGoodsActionRepo:  recommendGoodsActionRepo,
	}
	recommendExposureCase.RegisterQueueConsumer(_const.RecommendExposureEvent, recommendExposureCase.saveRecommendExposureEvent)
	return recommendExposureCase
}

// saveRecommendExposureEvent 消费推荐曝光事件。
func (c *RecommendExposureCase) saveRecommendExposureEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	// 队列消息转 JSON 失败时，无法继续解析业务体。
	if err != nil {
		return err
	}

	payload := make(map[string]*recommendExposureEvent)
	err = json.Unmarshal(rawBody, &payload)
	// 队列消息反序列化失败时，直接返回错误交由上层处理。
	if err != nil {
		return err
	}

	event, ok := payload["data"]
	// 队列体或主表数据缺失时，不再继续落库。
	if !ok || event == nil || event.Exposure == nil {
		return nil
	}

	return c.RecommendExposureRepo.Data.Transaction(context.TODO(), func(ctx context.Context) error {
		err := c.RecommendExposureRepo.Create(ctx, event.Exposure)
		if err != nil {
			return err
		}
		return c.recommendExposureItemCase.batchCreateByRecommendExposure(ctx, event.Exposure.ID, event.Exposure.RequestID, event.GoodsIds)
	})
}

// publishRecommendExposureEvent 投递推荐曝光事件。
func (c *RecommendExposureCase) publishRecommendExposureEvent(actor *appdto.RecommendActor, req *app.RecommendExposureReportRequest) {
	// 空请求直接忽略，避免埋点接口影响主流程。
	if req == nil {
		return
	}

	utils.AddQueue(_const.RecommendExposureEvent, &recommendExposureEvent{
		Exposure: &models.RecommendExposure{
			RequestID: req.GetRequestId(),
			ActorType: actor.ActorType,
			ActorID:   actor.ActorId,
			Scene:     int32(req.GetScene()),
			CreatedAt: time.Now(),
		},
		GoodsIds: req.GetGoodsIds(),
	})
}

// loadActorExposurePenalties 加载当前主体的曝光惩罚分。
func (c *RecommendExposureCase) loadActorExposurePenalties(ctx context.Context, actor *appdto.RecommendActor, scene int32, goodsIds []int64) (map[int64]float64, error) {
	// 主体、场景或候选商品缺失时，不计算曝光惩罚。
	if actor == nil || actor.ActorId <= 0 || scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}

	cutoff := time.Now().AddDate(0, 0, -recommendCandidate.ActorExposureLookbackDays)

	exposureCountMap, err := c.recommendExposureItemCase.loadRecommendExposureCountMap(ctx, actor, scene, cutoff, goodsIds)
	if err != nil {
		return nil, err
	}

	clickCountMap := make(map[int64]int64, len(goodsIds))
	clickCountMap, err = c.loadRecommendClickCountMap(ctx, actor, scene, cutoff, goodsIds)
	if err != nil {
		return nil, err
	}

	penalties := make(map[int64]float64, len(goodsIds))
	for _, goodsId := range goodsIds {
		exposureCount := exposureCountMap[goodsId]
		clickCount := clickCountMap[goodsId]
		// 曝光明显偏高且没有点击时，直接下调该商品权重。
		if exposureCount >= 3 && clickCount == 0 {
			penalties[goodsId] = 0.6
			continue
		}
		// 曝光很高但点击率极低时，施加更强的曝光惩罚。
		if exposureCount >= 5 && clickCount*20 < exposureCount {
			penalties[goodsId] = 0.3
		}
	}
	return penalties, nil
}

// bindRecommendExposureActor 将匿名曝光主体绑定为登录主体。
func (c *RecommendExposureCase) bindRecommendExposureActor(ctx context.Context, anonymousId, userId int64) error {
	recommendExposureQuery := c.RecommendExposureRepo.Data.Query(ctx).RecommendExposure
	_, err := recommendExposureQuery.WithContext(ctx).
		Where(
			recommendExposureQuery.ActorType.Eq(recommendevent.ActorTypeAnonymous),
			recommendExposureQuery.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendevent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// loadRecommendClickCountMap 查询当前主体在指定场景下的点击次数。
func (c *RecommendExposureCase) loadRecommendClickCountMap(ctx context.Context, actor *appdto.RecommendActor, scene int32, cutoff time.Time, goodsIds []int64) (map[int64]int64, error) {
	query := c.recommendGoodsActionRepo.Query(ctx).RecommendGoodsAction
	opts := make([]repo.QueryOption, 0, 6)
	opts = append(opts, repo.Where(query.ActorType.Eq(actor.ActorType)))
	opts = append(opts, repo.Where(query.ActorID.Eq(actor.ActorId)))
	opts = append(opts, repo.Where(query.Scene.Eq(scene)))
	opts = append(opts, repo.Where(query.EventType.Eq(int32(common.RecommendGoodsActionType_CLICK))))
	opts = append(opts, repo.Where(query.CreatedAt.Gte(cutoff)))
	opts = append(opts, repo.Where(query.GoodsID.In(goodsIds...)))

	list, err := c.recommendGoodsActionRepo.List(ctx, opts...)
	// 查询点击行为失败时，直接返回错误交由调用方处理。
	if err != nil {
		return nil, err
	}

	countMap := make(map[int64]int64, len(list))
	for _, item := range list {
		countMap[item.GoodsID]++
	}
	return countMap, nil
}
