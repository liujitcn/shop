package biz

import (
	"context"
	"encoding/json"
	"shop/api/gen/go/app"
	"shop/pkg/utils"
	"time"

	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendCandidate "shop/pkg/recommend/candidate"
	recommendevent "shop/pkg/recommend/event"
	appdto "shop/service/app/dto"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repo"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// RecommendExposureCase 推荐曝光业务处理对象。
type RecommendExposureCase struct {
	*biz.BaseCase
	*data.RecommendExposureRepo
	recommendGoodsActionCase *RecommendGoodsActionCase
}

// NewRecommendExposureCase 创建推荐曝光业务处理对象。
func NewRecommendExposureCase(
	baseCase *biz.BaseCase,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendGoodsActionCase *RecommendGoodsActionCase,
) *RecommendExposureCase {
	recommendExposureCase := &RecommendExposureCase{
		BaseCase:                 baseCase,
		RecommendExposureRepo:    recommendExposureRepo,
		recommendGoodsActionCase: recommendGoodsActionCase,
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

	payload := make(map[string]*models.RecommendExposure)
	err = json.Unmarshal(rawBody, &payload)
	// 队列消息反序列化失败时，直接返回错误交由上层处理。
	if err != nil {
		return err
	}

	event, ok := payload["data"]
	// 队列消息缺少业务体时直接丢弃，避免消费者重复报错。
	if !ok || event == nil {
		return nil
	}
	return c.Create(context.TODO(), event)
}

// publishRecommendExposureEvent 投递推荐曝光事件。
func (c *RecommendExposureCase) publishRecommendExposureEvent(actor *appdto.RecommendActor, req *app.RecommendExposureReportRequest) {
	utils.AddQueue(_const.RecommendExposureEvent, &models.RecommendExposure{
		RequestID: req.GetRequestId(),
		ActorType: actor.ActorType,
		ActorID:   actor.ActorId,
		Scene:     int32(req.GetScene()),
		GoodsIds:  _string.ConvertInt64ArrayToString(req.GetGoodsIds()),
		CreatedAt: time.Now(),
	})
}

// loadActorExposurePenalties 加载当前主体的曝光惩罚分。
func (c *RecommendExposureCase) loadActorExposurePenalties(ctx context.Context, actor *appdto.RecommendActor, scene int32, goodsIds []int64) (map[int64]float64, error) {
	// 主体、场景或候选商品缺失时，不计算曝光惩罚。
	if actor == nil || actor.ActorId <= 0 || scene == 0 || len(goodsIds) == 0 {
		return map[int64]float64{}, nil
	}

	exposureQuery := c.RecommendExposureRepo.Query(ctx).RecommendExposure
	exposureOpts := make([]repo.QueryOption, 0, 4)
	exposureOpts = append(exposureOpts, repo.Where(exposureQuery.ActorType.Eq(actor.ActorType)))
	exposureOpts = append(exposureOpts, repo.Where(exposureQuery.ActorID.Eq(actor.ActorId)))
	exposureOpts = append(exposureOpts, repo.Where(exposureQuery.Scene.Eq(scene)))

	cutoff := time.Now().AddDate(0, 0, -recommendCandidate.ActorExposureLookbackDays)
	exposureOpts = append(exposureOpts, repo.Where(exposureQuery.CreatedAt.Gte(cutoff)))
	exposureList, err := c.RecommendExposureRepo.List(ctx, exposureOpts...)
	// 查询曝光批次失败时，无法继续计算惩罚分。
	if err != nil {
		return nil, err
	}

	exposureCountMap := make(map[int64]int64, len(goodsIds))
	for _, item := range exposureList {
		ids := make([]int64, 0)
		// 曝光商品列表反序列化失败时，直接跳过当前批次。
		if err = json.Unmarshal([]byte(item.GoodsIds), &ids); err != nil {
			continue
		}
		for _, goodsID := range ids {
			exposureCountMap[goodsID]++
		}
	}

	clickCountMap := make(map[int64]int64, len(goodsIds))
	clickCountMap, err = c.recommendGoodsActionCase.loadRecommendClickCountMap(ctx, actor, scene, cutoff, goodsIds)
	// 查询点击行为失败时，无法继续计算曝光点击比。
	if err != nil {
		return nil, err
	}

	penalties := make(map[int64]float64, len(goodsIds))
	for _, goodsID := range goodsIds {
		exposureCount := exposureCountMap[goodsID]
		clickCount := clickCountMap[goodsID]
		// 曝光明显偏高且没有点击时，直接下调该商品权重。
		if exposureCount >= 3 && clickCount == 0 {
			penalties[goodsID] = 0.6
			continue
		}
		// 曝光很高但点击率极低时，施加更强的曝光惩罚。
		if exposureCount >= 5 && clickCount*20 < exposureCount {
			penalties[goodsID] = 0.3
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
