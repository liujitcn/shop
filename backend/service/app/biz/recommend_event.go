package biz

import (
	"context"
	"encoding/json"
	"shop/pkg/biz"
	"time"

	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/utils"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

const (
	recommendEventTypeExposure = "recommend_exposure"
	recommendEventTypeClick    = "recommend_click"
	recommendEventTypeView     = "goods_view"
)

// RecommendEvent 推荐行为异步事件。
type RecommendEvent struct {
	EventType  string  `json:"eventType"`
	UserID     int64   `json:"userId"`
	RequestID  string  `json:"requestId,omitempty"`
	Scene      string  `json:"scene,omitempty"`
	Source     string  `json:"source,omitempty"`
	GoodsID    int64   `json:"goodsId,omitempty"`
	GoodsIDs   []int64 `json:"goodsIds,omitempty"`
	Position   int32   `json:"position,omitempty"`
	ExposeMode string  `json:"exposeMode,omitempty"`
	ViewMode   string  `json:"viewMode,omitempty"`
	OccurredAt int64   `json:"occurredAt,omitempty"`
}

// RecommendEventCase 推荐行为事件消费者。
type RecommendEventCase struct {
	*biz.BaseCase
	*data.RecommendExposureRepo
	*data.RecommendClickRepo
	*data.RecommendGoodsViewRepo
}

// NewRecommendEventCase 创建推荐行为事件消费者并注册队列。
func NewRecommendEventCase(
	baseCase *biz.BaseCase,
	recommendExposureRepo *data.RecommendExposureRepo,
	recommendClickRepo *data.RecommendClickRepo,
	recommendGoodsViewRepo *data.RecommendGoodsViewRepo,
) *RecommendEventCase {
	c := &RecommendEventCase{
		BaseCase:               baseCase,
		RecommendExposureRepo:  recommendExposureRepo,
		RecommendClickRepo:     recommendClickRepo,
		RecommendGoodsViewRepo: recommendGoodsViewRepo,
	}

	c.RegisterQueueConsumer(_const.RecommendEvent, c.SaveRecommendEvent)
	return c
}

// SaveRecommendEvent 消费推荐行为事件。
func (c *RecommendEventCase) SaveRecommendEvent(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		return err
	}
	var payload map[string]*RecommendEvent
	if err = json.Unmarshal(rawBody, &payload); err != nil {
		return err
	}
	event, ok := payload["data"]
	if !ok || event == nil {
		return nil
	}
	return c.consume(context.TODO(), event)
}

func publishRecommendExposureEvent(userID int64, requestID, scene string, goodsIDs []int64) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeExposure,
		UserID:     userID,
		RequestID:  requestID,
		Scene:      scene,
		GoodsIDs:   goodsIDs,
		ExposeMode: "viewport_once",
		OccurredAt: time.Now().Unix(),
	})
}

func publishRecommendClickEvent(userID, goodsID int64, requestID, scene, source string, position int32) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeClick,
		UserID:     userID,
		RequestID:  requestID,
		Scene:      scene,
		Source:     source,
		GoodsID:    goodsID,
		Position:   position,
		OccurredAt: time.Now().Unix(),
	})
}

func publishGoodsViewEvent(userID, goodsID int64, position int32, requestID, source, scene string) {
	utils.AddQueue(_const.RecommendEvent, &RecommendEvent{
		EventType:  recommendEventTypeView,
		UserID:     userID,
		RequestID:  requestID,
		Scene:      scene,
		Source:     source,
		GoodsID:    goodsID,
		Position:   position,
		ViewMode:   "detail_open",
		OccurredAt: time.Now().Unix(),
	})
}

func (c *RecommendEventCase) consume(ctx context.Context, event *RecommendEvent) error {
	switch event.EventType {
	case recommendEventTypeExposure:
		goodsIDsJSON, err := json.Marshal(event.GoodsIDs)
		if err != nil {
			return err
		}
		return c.RecommendExposureRepo.Create(ctx, &models.RecommendExposure{
			RequestID:    event.RequestID,
			UserID:       event.UserID,
			Scene:        event.Scene,
			GoodsIdsJSON: string(goodsIDsJSON),
			ExposeMode:   defaultString(event.ExposeMode, "viewport_once"),
		})
	case recommendEventTypeClick:
		return c.RecommendClickRepo.Create(ctx, &models.RecommendClick{
			RequestID: event.RequestID,
			UserID:    event.UserID,
			Scene:     event.Scene,
			GoodsID:   event.GoodsID,
			Position:  event.Position,
			Source:    defaultString(event.Source, "recommend"),
		})
	case recommendEventTypeView:
		return c.RecommendGoodsViewRepo.Create(ctx, &models.RecommendGoodsView{
			UserID:    event.UserID,
			GoodsID:   event.GoodsID,
			Source:    defaultString(event.Source, "direct"),
			Scene:     event.Scene,
			RequestID: event.RequestID,
			Position:  event.Position,
			ViewMode:  defaultString(event.ViewMode, "detail_open"),
		})
	default:
		return nil
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
