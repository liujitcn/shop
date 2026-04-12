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
	recommendEvent "shop/pkg/recommend/event"
	"shop/pkg/utils"
	appDto "shop/service/app/dto"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// RecommendGoodsActionCase 推荐商品行为业务处理对象。
type RecommendGoodsActionCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.RecommendGoodsActionRepo
	recommendRequestItemCase         *RecommendRequestItemCase
	recommendUserPreferenceCase      *RecommendUserPreferenceCase
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase
	recommendGoodsRelationCase       *RecommendGoodsRelationCase
	goodsInfoCase                    *GoodsInfoCase
}

// NewRecommendGoodsActionCase 创建推荐商品行为业务处理对象。
func NewRecommendGoodsActionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	recommendGoodsActionRepo *data.RecommendGoodsActionRepo,
	recommendRequestItemCase *RecommendRequestItemCase,
	recommendUserPreferenceCase *RecommendUserPreferenceCase,
	recommendUserGoodsPreferenceCase *RecommendUserGoodsPreferenceCase,
	recommendGoodsRelationCase *RecommendGoodsRelationCase,
	goodsInfoCase *GoodsInfoCase,
) *RecommendGoodsActionCase {
	recommendGoodsActionCase := &RecommendGoodsActionCase{
		BaseCase:                         baseCase,
		tx:                               tx,
		RecommendGoodsActionRepo:         recommendGoodsActionRepo,
		recommendRequestItemCase:         recommendRequestItemCase,
		recommendUserPreferenceCase:      recommendUserPreferenceCase,
		recommendUserGoodsPreferenceCase: recommendUserGoodsPreferenceCase,
		recommendGoodsRelationCase:       recommendGoodsRelationCase,
		goodsInfoCase:                    goodsInfoCase,
	}
	recommendGoodsActionCase.RegisterQueueConsumer(_const.RecommendGoodsActionEvent, recommendGoodsActionCase.saveRecommendGoodsActionEvent)
	return recommendGoodsActionCase
}

// saveRecommendGoodsActionEvent 消费推荐商品行为事件。
func (c *RecommendGoodsActionCase) saveRecommendGoodsActionEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	// 队列消息转 JSON 失败时，无法继续解析业务体。
	if err != nil {
		return err
	}

	payload := make(map[string]*utils.RecommendGoodsActionEvent)
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

	list := event.GoodsItems
	// 队列消息没有商品行为明细时，不再继续落库和聚合。
	if len(list) == 0 {
		return nil
	}

	return c.tx.Transaction(context.TODO(), func(ctx context.Context) error {
		err = c.BatchCreate(ctx, list)
		if err != nil {
			return err
		}

		actor := event.RecommendActor
		// 匿名主体不沉淀画像，只保留行为明细。
		if actor == nil || actor.ActorType != recommendEvent.ActorTypeUser || actor.ActorId <= 0 {
			return nil
		}

		eventType := event.EventType
		// 无法识别的行为类型不参与后续聚合。
		if eventType == common.RecommendGoodsActionType_UNKNOWN_RGAT {
			return nil
		}

		isSingleGoodsEvent := recommendEvent.IsSingleGoodsEvent(eventType)
		isOrderGoodsEvent := recommendEvent.IsOrderGoodsEvent(eventType)
		// 非单商品且非订单级行为不参与后续聚合。
		if !isSingleGoodsEvent && !isOrderGoodsEvent {
			return nil
		}

		userId := actor.ActorId
		for _, item := range list {
			eventTime := item.CreatedAt
			var goodsInfo *models.GoodsInfo
			goodsInfo, err = c.goodsInfoCase.GoodsInfoRepo.FindById(ctx, item.GoodsID)
			if err != nil {
				return err
			}
			err = c.recommendUserGoodsPreferenceCase.upsertUserGoodsPreference(ctx, userId, item.GoodsID, eventType, eventTime, item.GoodsNum)
			if err != nil {
				return err
			}
			err = c.recommendUserPreferenceCase.upsertUserCategoryPreference(ctx, userId, goodsInfo.CategoryID, eventType, eventTime, item.GoodsNum)
			if err != nil {
				return err
			}
			// 单商品行为逐条沉淀商品关联。
			if isSingleGoodsEvent {
				err = c.upsertGoodsRelation(ctx, eventType, item.RequestID, item.GoodsID, item.GoodsNum, eventTime)
				if err != nil {
					return err
				}
			}
		}
		if isOrderGoodsEvent {
			return c.recommendGoodsRelationCase.upsertOrderGoodsRelations(ctx, list, eventType, event.EventTime)
		}
		return nil
	})
}

// bindRecommendGoodsActionActor 将匿名行为主体绑定为登录主体。
func (c *RecommendGoodsActionCase) bindRecommendGoodsActionActor(ctx context.Context, anonymousId, userId int64) error {
	recommendGoodsActionQuery := c.RecommendGoodsActionRepo.Data.Query(ctx).RecommendGoodsAction
	_, err := recommendGoodsActionQuery.WithContext(ctx).
		Where(
			recommendGoodsActionQuery.ActorType.Eq(recommendEvent.ActorTypeAnonymous),
			recommendGoodsActionQuery.ActorID.Eq(anonymousId),
		).
		Updates(map[string]interface{}{
			"actor_type": recommendEvent.ActorTypeUser,
			"actor_id":   userId,
		})
	return err
}

// publishRecommendGoodsActionEvent 投递单商品埋点事件。
func (c *RecommendGoodsActionCase) publishRecommendGoodsActionEvent(actor *appDto.RecommendActor, req *app.RecommendGoodsActionReportRequest) {
	// 空请求直接忽略，避免埋点接口影响主流程。
	if req == nil {
		return
	}

	utils.DispatchRecommendGoodsActionEvent(actor, req, time.Now())
}

// upsertGoodsRelation 按同一次推荐请求的共同出现结果累计商品关联度。
func (c *RecommendGoodsActionCase) upsertGoodsRelation(ctx context.Context, eventType common.RecommendGoodsActionType, requestId string, goodsId, goodsNum int64, eventTime time.Time) error {
	if requestId == "" {
		return nil
	}

	relatedGoodsIds, err := c.recommendRequestItemCase.listRelatedGoodsIdsByRequestId(ctx, requestId, goodsId)
	if err != nil {
		return err
	}
	for _, relatedGoodsId := range relatedGoodsIds {
		err = c.recommendGoodsRelationCase.upsertSingleGoodsRelation(ctx, relatedGoodsId, goodsId, eventType, eventTime, recommendEvent.NormalizeGoodsNum(goodsNum))
		if err != nil {
			return err
		}
		err = c.recommendGoodsRelationCase.upsertSingleGoodsRelation(ctx, goodsId, relatedGoodsId, eventType, eventTime, recommendEvent.NormalizeGoodsNum(goodsNum))
		if err != nil {
			return err
		}
	}
	return nil
}
