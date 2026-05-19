package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/errorsx"
	"shop/service/base/dto"

	kratosHTTP "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
)

var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = kratosHTTP.SupportPackageIsVersion1

const OperationAiAssistantMessageServiceSendAiAssistantMessage = "/base.v1.AiAssistantMessageService/SendAiAssistantMessage"

// AiAssistantMessageServiceHTTPServer 定义 AI 助手消息发送 HTTP 服务。
type AiAssistantMessageServiceHTTPServer interface {
	// StreamAiAssistantMessage 流式发送 AI 助手消息。
	StreamAiAssistantMessage(context.Context, *basev1.SendAiAssistantMessageRequest, dto.AiAssistantStreamEmitter) error
}

// RegisterAiAssistantMessageServiceHTTPServer 注册 AI 助手消息发送 HTTP 接口。
func RegisterAiAssistantMessageServiceHTTPServer(s *kratosHTTP.Server, srv AiAssistantMessageServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/api/v1/base/ai/assistant/session/{session_id}/message", _AiAssistantMessageService_SendAiAssistantMessage0_HTTP_Handler(srv))
}

// _AiAssistantMessageService_SendAiAssistantMessage0_HTTP_Handler 处理 AI 助手消息发送流式请求。
func _AiAssistantMessageService_SendAiAssistantMessage0_HTTP_Handler(srv AiAssistantMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.SendAiAssistantMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiAssistantMessageServiceSendAiAssistantMessage)
		response := ctx.Response()
		flusher, ok := response.(http.Flusher)
		if !ok {
			return errorsx.Internal("流式响应不支持")
		}
		emitter := &aiAssistantStreamEmitter{
			writer:  response,
			flusher: flusher,
		}
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			header := response.Header()
			header.Set("Content-Type", "text/event-stream; charset=utf-8")
			header.Set("Cache-Control", "no-cache")
			header.Set("Connection", "keep-alive")
			header.Set("X-Accel-Buffering", "no")
			return nil, srv.StreamAiAssistantMessage(ctx, req.(*basev1.SendAiAssistantMessageRequest), emitter)
		})
		_, err := h(ctx, &in)
		return err
	}
}

type aiAssistantStreamEmitter struct {
	writer  http.ResponseWriter
	flusher http.Flusher
	mutex   sync.Mutex
}

// EmitAiAssistantStream 写入单条 AI 助手 SSE 事件。
func (e *aiAssistantStreamEmitter) EmitAiAssistantStream(event dto.AiAssistantStreamEvent, payload dto.AiAssistantStreamPayload) error {
	if e == nil || e.writer == nil {
		return fmt.Errorf("AI助手流式响应写入器未初始化")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if _, err = fmt.Fprintf(e.writer, "event: %s\n", event); err != nil {
		return err
	}
	if _, err = fmt.Fprintf(e.writer, "data: %s\n\n", data); err != nil {
		return err
	}
	if e.flusher != nil {
		e.flusher.Flush()
	}
	return nil
}
