package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"

	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

const adminAssistantStreamPrefix = "assistant_admin_"

// Payload 表示 AI 助手流式事件负载。
type Payload struct {
	Event           commonv1.SseEvent            `json:"event"`
	SessionID       string                       `json:"session_id"`
	ClientMessageID string                       `json:"client_message_id"`
	Delta           string                       `json:"delta,omitempty"`
	Messages        []*basev1.AiAssistantMessage `json:"messages,omitempty"`
	Session         *basev1.AiAssistantSession   `json:"session,omitempty"`
	ErrorMessage    string                       `json:"error_message,omitempty"`
	OccurredAt      string                       `json:"occurred_at"`
}

// Publisher 负责发布 AI 助手流式 SSE 事件。
type Publisher struct {
	sse *sseServer.Server
}

// NewPublisher 创建 AI 助手流式事件发布器。
func NewPublisher(sse *sseServer.Server) *Publisher {
	return &Publisher{sse: sse}
}

// AdminAssistantStreamID 返回当前后台用户的 AI 助手专属流标识。
func AdminAssistantStreamID(userID int64) string {
	return fmt.Sprintf("%s%d", adminAssistantStreamPrefix, userID)
}

// ResolveAdminStreamID 将 SSE 流枚举转换为当前用户实际订阅的 stream ID。
func ResolveAdminStreamID(stream commonv1.SseStream, userID int64) string {
	switch stream {
	case commonv1.SseStream_SSE_STREAM_ADMIN_AI_ASSISTANT:
		return AdminAssistantStreamID(userID)
	case commonv1.SseStream_SSE_STREAM_ADMIN_WORKSPACE:
		return strconv.FormatInt(int64(stream), 10)
	default:
		return ""
	}
}

// ParseAdminStream 解析直接传入的 stream 查询参数。
func ParseAdminStream(raw string, userID int64) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == AdminAssistantStreamID(userID) {
		return raw
	}
	streamValue, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return raw
	}
	return ResolveAdminStreamID(commonv1.SseStream(streamValue), userID)
}

// PublishDelta 发布 AI 助手流式增量文本。
func (p *Publisher) PublishDelta(ctx context.Context, userID int64, sessionID string, clientMessageID string, delta string) error {
	if p == nil || p.sse == nil || strings.TrimSpace(delta) == "" {
		return nil
	}
	return p.publish(ctx, userID, commonv1.SseEvent_SSE_EVENT_AI_ASSISTANT_DELTA, Payload{
		SessionID:       strings.TrimSpace(sessionID),
		ClientMessageID: strings.TrimSpace(clientMessageID),
		Delta:           delta,
	})
}

// PublishFinish 发布 AI 助手流式完成事件。
func (p *Publisher) PublishFinish(
	ctx context.Context,
	userID int64,
	sessionID string,
	clientMessageID string,
	messages []*basev1.AiAssistantMessage,
	session *basev1.AiAssistantSession,
) error {
	if p == nil || p.sse == nil {
		return nil
	}
	return p.publish(ctx, userID, commonv1.SseEvent_SSE_EVENT_AI_ASSISTANT_FINISH, Payload{
		SessionID:       strings.TrimSpace(sessionID),
		ClientMessageID: strings.TrimSpace(clientMessageID),
		Messages:        messages,
		Session:         session,
	})
}

// PublishError 发布 AI 助手流式异常事件。
func (p *Publisher) PublishError(ctx context.Context, userID int64, sessionID string, clientMessageID string, err error) error {
	if p == nil || p.sse == nil || err == nil {
		return nil
	}
	return p.publish(ctx, userID, commonv1.SseEvent_SSE_EVENT_AI_ASSISTANT_ERROR, Payload{
		SessionID:       strings.TrimSpace(sessionID),
		ClientMessageID: strings.TrimSpace(clientMessageID),
		ErrorMessage:    err.Error(),
	})
}

// EnsureAdminAssistantStream 确保后台用户的 AI 助手流已创建。
func (p *Publisher) EnsureAdminAssistantStream(userID int64) {
	if p == nil || p.sse == nil || userID <= 0 {
		return
	}
	p.sse.CreateStream(sseServer.StreamID(AdminAssistantStreamID(userID)))
}

func (p *Publisher) publish(ctx context.Context, userID int64, event commonv1.SseEvent, payload Payload) error {
	if userID <= 0 {
		return nil
	}
	payload.Event = event
	payload.OccurredAt = time.Now().Format(time.RFC3339)
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	streamID := sseServer.StreamID(AdminAssistantStreamID(userID))
	p.sse.CreateStream(streamID)
	p.sse.Publish(ctx, streamID, &sseServer.Event{
		Event: []byte(strconv.FormatInt(int64(event), 10)),
		Data:  data,
	})
	return nil
}
