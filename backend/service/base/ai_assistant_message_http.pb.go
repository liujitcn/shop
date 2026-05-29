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

const OperationAiAssistantMessageServiceDeleteAiAssistantMessage = "/base.v1.AiAssistantMessageService/DeleteAiAssistantMessage"
const OperationAiAssistantMessageServiceRegenerateAiAssistantMessage = "/base.v1.AiAssistantMessageService/RegenerateAiAssistantMessage"
const OperationAiAssistantMessageServiceRetryAiAssistantUserMessage = "/base.v1.AiAssistantMessageService/RetryAiAssistantUserMessage"
const OperationAiAssistantMessageServiceSendAiAssistantMessage = "/base.v1.AiAssistantMessageService/SendAiAssistantMessage"

// AiAssistantMessageServiceHTTPServer 定义 AI 助手消息发送 HTTP 服务。
type AiAssistantMessageServiceHTTPServer interface {
	// DeleteAiAssistantMessage 删除 AI 助手消息。
	DeleteAiAssistantMessage(context.Context, *basev1.DeleteAiAssistantMessageRequest) (*basev1.DeleteAiAssistantMessageResponse, error)
	// RegenerateAiAssistantMessage 重新生成助手回复。
	RegenerateAiAssistantMessage(context.Context, *basev1.RegenerateAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error)
	// RetryAiAssistantUserMessage 重试失败的用户消息。
	RetryAiAssistantUserMessage(context.Context, *basev1.RetryAiAssistantUserMessageRequest) (*basev1.SendAiAssistantMessageResponse, error)
	// StreamAiAssistantMessage 流式发送 AI 助手消息。
	StreamAiAssistantMessage(context.Context, *basev1.SendAiAssistantMessageRequest, dto.AiAssistantStreamEmitter) error
}

// RegisterAiAssistantMessageServiceHTTPServer 注册 AI 助手消息发送 HTTP 接口。
func RegisterAiAssistantMessageServiceHTTPServer(s *kratosHTTP.Server, srv AiAssistantMessageServiceHTTPServer) {
	r := s.Route("/")
	r.POST("/api/v1/base/ai/assistant/session/{session_id}/message", _AiAssistantMessageService_SendAiAssistantMessage0_HTTP_Handler(srv))
	r.DELETE("/api/v1/base/ai/assistant/session/{session_id}/message/{message_id}", _AiAssistantMessageService_DeleteAiAssistantMessage0_HTTP_Handler(srv))
	r.POST("/api/v1/base/ai/assistant/session/{session_id}/message/{message_id}/retry", _AiAssistantMessageService_RetryAiAssistantUserMessage0_HTTP_Handler(srv))
	r.POST("/api/v1/base/ai/assistant/session/{session_id}/message/{message_id}/regeneration", _AiAssistantMessageService_RegenerateAiAssistantMessage0_HTTP_Handler(srv))
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

// _AiAssistantMessageService_DeleteAiAssistantMessage0_HTTP_Handler 处理 AI 助手消息删除请求。
func _AiAssistantMessageService_DeleteAiAssistantMessage0_HTTP_Handler(srv AiAssistantMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.DeleteAiAssistantMessageRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiAssistantMessageServiceDeleteAiAssistantMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DeleteAiAssistantMessage(ctx, req.(*basev1.DeleteAiAssistantMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.DeleteAiAssistantMessageResponse))
	}
}

// _AiAssistantMessageService_RetryAiAssistantUserMessage0_HTTP_Handler 处理失败用户消息重试请求。
func _AiAssistantMessageService_RetryAiAssistantUserMessage0_HTTP_Handler(srv AiAssistantMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.RetryAiAssistantUserMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiAssistantMessageServiceRetryAiAssistantUserMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.RetryAiAssistantUserMessage(ctx, req.(*basev1.RetryAiAssistantUserMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.SendAiAssistantMessageResponse))
	}
}

// _AiAssistantMessageService_RegenerateAiAssistantMessage0_HTTP_Handler 处理助手回复重新生成请求。
func _AiAssistantMessageService_RegenerateAiAssistantMessage0_HTTP_Handler(srv AiAssistantMessageServiceHTTPServer) func(ctx kratosHTTP.Context) error {
	return func(ctx kratosHTTP.Context) error {
		var in basev1.RegenerateAiAssistantMessageRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		kratosHTTP.SetOperation(ctx, OperationAiAssistantMessageServiceRegenerateAiAssistantMessage)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.RegenerateAiAssistantMessage(ctx, req.(*basev1.RegenerateAiAssistantMessageRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		return ctx.Result(200, out.(*basev1.SendAiAssistantMessageResponse))
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
