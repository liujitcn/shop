package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/models"
	"shop/pkg/llm"
	baseDTO "shop/service/base/dto"

	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	aiAssistantSceneWorkspace = "workspace"
	aiAssistantSceneRecommend = "recommend"
	aiAssistantSceneComment   = "comment"

	aiAssistantTerminalAdmin = "admin"
	aiAssistantTerminalApp   = "app"

	aiAssistantRoleUser      = "user"
	aiAssistantRoleAssistant = "assistant"

	aiAssistantKindText    = "text"
	aiAssistantKindTool    = "tool"
	aiAssistantKindConfirm = "confirm"
	aiAssistantPreviewSize = 18
	aiAssistantHistorySize = 12
)

type aiAssistantReplyMeta struct {
	ReplySource    string                           `json:"reply_source"`
	Model          string                           `json:"model"`
	Fallback       bool                             `json:"fallback"`
	FallbackReason string                           `json:"fallback_reason"`
	Confirm        *baseDTO.AiAssistantConfirmState `json:"confirm,omitempty"`
}

// AiAssistantCase 管理当前系统 AI 助手会话与消息。
type AiAssistantCase struct {
	*biz.BaseCase
	aiAssistantSessionCase *AiAssistantSessionCase
	aiAssistantMessageCase *AiAssistantMessageCase
	baseUserCase           *BaseUserCase
	llmClient              *llm.Client
	toolRuntime            AiAssistantToolRuntime
}

// NewAiAssistantCase 创建 AI 助手业务实例。
func NewAiAssistantCase(
	baseCase *biz.BaseCase,
	aiAssistantSessionCase *AiAssistantSessionCase,
	aiAssistantMessageCase *AiAssistantMessageCase,
	baseUserCase *BaseUserCase,
	llmClient *llm.Client,
	toolRuntime AiAssistantToolRuntime,
) *AiAssistantCase {
	if toolRuntime == nil {
		toolRuntime = NewNoopAiAssistantToolRuntime()
	}
	return &AiAssistantCase{
		BaseCase:               baseCase,
		aiAssistantSessionCase: aiAssistantSessionCase,
		aiAssistantMessageCase: aiAssistantMessageCase,
		baseUserCase:           baseUserCase,
		llmClient:              llmClient,
		toolRuntime:            toolRuntime,
	}
}

// ListAiAssistantSessions 查询当前用户的 AI 助手会话列表。
func (c *AiAssistantCase) ListAiAssistantSessions(ctx context.Context, req *basev1.ListAiAssistantSessionsRequest) (*basev1.ListAiAssistantSessionsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal := normalizeAiAssistantTerminal(req.GetTerminal())
	list, err := c.aiAssistantSessionCase.ListByUserAndTerminal(ctx, authInfo.UserId, terminal)
	if err != nil {
		return nil, err
	}

	sessions := make([]*basev1.AiAssistantSession, 0, len(list))
	for _, item := range list {
		sessions = append(sessions, c.aiAssistantSessionCase.ToDTO(item))
	}
	return &basev1.ListAiAssistantSessionsResponse{Sessions: sessions}, nil
}

// CreateAiAssistantSession 创建当前用户的新会话。
func (c *AiAssistantCase) CreateAiAssistantSession(ctx context.Context, req *basev1.CreateAiAssistantSessionRequest) (*basev1.AiAssistantSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		title = "新对话"
	}
	scene := normalizeAiAssistantScene(req.GetScene())
	now := time.Now()
	model := &models.AiAssistantSession{
		UserID:        authInfo.UserId,
		Terminal:      normalizeAiAssistantTerminal(req.GetTerminal()),
		Title:         title,
		Scene:         scene,
		Summary:       buildAiAssistantDefaultSummary(scene),
		ToolCount:     0,
		LastMessageAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = c.aiAssistantSessionCase.CreateSession(ctx, model)
	if err != nil {
		return nil, err
	}
	return c.aiAssistantSessionCase.ToDTO(model), nil
}

// UpdateAiAssistantSession 更新当前用户的会话标题。
func (c *AiAssistantCase) UpdateAiAssistantSession(ctx context.Context, req *basev1.UpdateAiAssistantSessionRequest) (*basev1.AiAssistantSession, error) {
	title := strings.TrimSpace(req.GetTitle())
	if title == "" {
		return nil, errorsx.InvalidArgument("会话标题不能为空")
	}

	session, err := c.findAiAssistantSessionByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	now := time.Now()
	err = c.aiAssistantSessionCase.UpdateTitle(ctx, session, title, now)
	if err != nil {
		return nil, err
	}
	return c.aiAssistantSessionCase.ToDTO(session), nil
}

// DeleteAiAssistantSession 删除当前用户的会话。
func (c *AiAssistantCase) DeleteAiAssistantSession(ctx context.Context, req *basev1.DeleteAiAssistantSessionRequest) (*emptypb.Empty, error) {
	session, err := c.findAiAssistantSessionByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	err = c.aiAssistantSessionCase.DeleteSession(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ListAiAssistantMessages 查询指定会话的消息列表。
func (c *AiAssistantCase) ListAiAssistantMessages(ctx context.Context, req *basev1.ListAiAssistantMessagesRequest) (*basev1.ListAiAssistantMessagesResponse, error) {
	session, err := c.findAiAssistantSessionByID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	list, err := c.aiAssistantMessageCase.ListBySessionID(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	messages := make([]*basev1.AiAssistantMessage, 0, len(list))
	for _, item := range list {
		messages = append(messages, c.aiAssistantMessageCase.ToDTO(item))
	}
	return &basev1.ListAiAssistantMessagesResponse{Messages: messages}, nil
}

// SendAiAssistantMessage 发送用户消息并生成 AI 助手回复。
func (c *AiAssistantCase) SendAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	session, err := c.findAiAssistantSessionByID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.GetContent())
	attachments := normalizeAiAssistantAttachments(req.GetAttachments())
	if content == "" && len(attachments) == 0 {
		return nil, errorsx.InvalidArgument("消息内容不能为空")
	}
	llmAttachments, err := c.buildAiAssistantLLMAttachments(ctx, attachments)
	if err != nil {
		return nil, err
	}

	userName, err := c.findAiAssistantUserName(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	userMessage := &models.AiAssistantMessage{
		SessionID:        session.ID,
		UserID:           session.UserID,
		Role:             aiAssistantRoleUser,
		Kind:             aiAssistantKindText,
		Content:          buildAiAssistantUserContent(content, attachments),
		AttachmentsJSON:  mustMarshalAiAssistantAttachments(attachments),
		ToolsJSON:        marshalAiAssistantTools(nil),
		ConfirmLinesJSON: marshalAiAssistantConfirmLines(nil),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = c.aiAssistantMessageCase.CreateMessage(ctx, userMessage)
	if err != nil {
		return nil, err
	}

	history, err := c.aiAssistantMessageCase.BuildHistory(ctx, session.ID, aiAssistantHistorySize)
	if err != nil {
		return nil, err
	}
	reply, err := c.generateAiAssistantReply(ctx, session, userName, content, attachments, llmAttachments, history)
	if err != nil {
		reply = &llm.AiAssistantResponse{
			Content:        buildAiAssistantFallbackReply(session.Scene, content, attachments),
			TokenUsage:     0,
			Source:         "fallback",
			Model:          c.llmClient.Model(),
			Fallback:       true,
			FallbackReason: err.Error(),
			Tools:          []llm.AiAssistantToolCall{},
		}
	}

	responseMessages := make([]*basev1.AiAssistantMessage, 0, 3)
	responseMessages = append(responseMessages, c.aiAssistantMessageCase.ToDTO(userMessage))
	assistantMessage, confirmMessage, err := c.createAiAssistantReplyMessages(ctx, session, reply, now)
	if err != nil {
		return nil, err
	}
	responseMessages = append(responseMessages, c.aiAssistantMessageCase.ToDTO(assistantMessage))
	if confirmMessage != nil {
		responseMessages = append(responseMessages, c.aiAssistantMessageCase.ToDTO(confirmMessage))
	}

	summary := buildAiAssistantDynamicSummary(session.Scene, content, attachments)
	err = c.aiAssistantSessionCase.TouchSession(ctx, session, summary, int32(len(reply.Tools)), now)
	if err != nil {
		return nil, err
	}

	return &basev1.SendAiAssistantMessageResponse{
		Messages: responseMessages,
		Session:  c.aiAssistantSessionCase.ToDTO(session),
	}, nil
}

// OperateAiAssistantConfirm 处理确认卡的确认或拒绝动作。
func (c *AiAssistantCase) OperateAiAssistantConfirm(ctx context.Context, req *basev1.OperateAiAssistantConfirmRequest) (*basev1.OperateAiAssistantConfirmResponse, error) {
	session, err := c.findAiAssistantSessionByID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	message, err := c.findAiAssistantMessageByID(ctx, session.ID, req.GetMessageId())
	if err != nil {
		return nil, err
	}
	confirmState := parseAiAssistantReplyMeta(message.Content).Confirm
	if confirmState == nil {
		return nil, errorsx.InvalidArgument("当前消息不支持确认操作")
	}
	if confirmState.Status != baseDTO.AiAssistantConfirmStatusPending {
		return nil, errorsx.StateConflict("确认卡已处理，无需重复操作", "ai_assistant_confirm", confirmState.Status, baseDTO.AiAssistantConfirmStatusPending)
	}

	action := normalizeAiAssistantConfirmAction(req.GetAction())
	confirmResult, err := c.toolRuntime.ExecuteConfirm(ctx, AiAssistantConfirmRuntimeInput{
		SessionID: req.GetSessionId(),
		MessageID: req.GetMessageId(),
		Action:    action,
		Confirm:   confirmState,
		FormJSON:  strings.TrimSpace(req.GetFormJson()),
	})
	if err != nil {
		return nil, err
	}

	if confirmResult == nil {
		confirmResult = &AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusFailed,
			Summary: "确认动作未返回结果",
			Reply:   "确认动作暂未完成，请稍后重试。",
		}
	}

	confirmState.Status = normalizeAiAssistantConfirmStatus(confirmResult.Status)
	if strings.TrimSpace(confirmResult.Summary) != "" {
		confirmState.Summary = strings.TrimSpace(confirmResult.Summary)
	}

	now := time.Now()
	messageMeta := parseAiAssistantReplyMeta(message.Content)
	message.Content = marshalAiAssistantReplyContent(&llm.AiAssistantResponse{
		Content:        parseAiAssistantReplyContent(message.Content),
		Source:         messageMeta.ReplySource,
		Model:          messageMeta.Model,
		Fallback:       messageMeta.Fallback,
		FallbackReason: messageMeta.FallbackReason,
		Confirm:        toLLMConfirmRequest(confirmState, message.ConfirmTitle, parseAiAssistantConfirmLines(message.ConfirmLinesJSON)),
	})
	message.ToolsJSON = marshalAiAssistantTools(toAiAssistantToolsDTO(confirmResult.Tools))
	message.UpdatedAt = now

	err = c.aiAssistantMessageCase.UpdateReplyMeta(ctx, message)
	if err != nil {
		return nil, err
	}

	assistantReplyMessage := &models.AiAssistantMessage{
		SessionID:        session.ID,
		UserID:           session.UserID,
		Role:             aiAssistantRoleAssistant,
		Kind:             resolveAiAssistantReplyKind(confirmResult.Tools),
		Content:          marshalAiAssistantReplyContent(&llm.AiAssistantResponse{Content: strings.TrimSpace(confirmResult.Reply), Source: "tool", Model: c.llmClient.Model()}),
		AttachmentsJSON:  mustMarshalAiAssistantAttachments(nil),
		ToolsJSON:        marshalAiAssistantTools(toAiAssistantToolsDTO(confirmResult.Tools)),
		ConfirmLinesJSON: marshalAiAssistantConfirmLines(nil),
		TokenUsage:       0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = c.aiAssistantMessageCase.CreateMessage(ctx, assistantReplyMessage)
	if err != nil {
		return nil, err
	}

	err = c.aiAssistantSessionCase.RefreshLastMessageAt(ctx, session, now)
	if err != nil {
		return nil, err
	}

	return &basev1.OperateAiAssistantConfirmResponse{
		Messages: []*basev1.AiAssistantMessage{
			c.aiAssistantMessageCase.ToDTO(message),
			c.aiAssistantMessageCase.ToDTO(assistantReplyMessage),
		},
		Session: c.aiAssistantSessionCase.ToDTO(session),
	}, nil
}

// findAiAssistantSessionByID 按会话编号查询当前用户的会话。
func (c *AiAssistantCase) findAiAssistantSessionByID(ctx context.Context, rawID string) (*models.AiAssistantSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	return c.aiAssistantSessionCase.FindByUserAndRawID(ctx, authInfo.UserId, rawID)
}

// findAiAssistantMessageByID 查询指定会话下的消息。
func (c *AiAssistantCase) findAiAssistantMessageByID(ctx context.Context, sessionID int64, rawID string) (*models.AiAssistantMessage, error) {
	return c.aiAssistantMessageCase.FindBySessionAndRawID(ctx, sessionID, rawID)
}

// findAiAssistantUserName 查询当前会话所属用户名称。
func (c *AiAssistantCase) findAiAssistantUserName(ctx context.Context, userID int64) (string, error) {
	return c.baseUserCase.FindDisplayNameByID(ctx, userID)
}

// generateAiAssistantReply 生成当前消息的 AI 助手回复。
func (c *AiAssistantCase) generateAiAssistantReply(
	ctx context.Context,
	session *models.AiAssistantSession,
	userName string,
	content string,
	attachments []*basev1.AiAssistantAttachment,
	llmAttachments []llm.AiAssistantAttachment,
	history []llm.AiAssistantMessage,
) (*llm.AiAssistantResponse, error) {
	toolResult, err := c.toolRuntime.RunToolCalls(ctx, AiAssistantToolRuntimeInput{
		Scene:        session.Scene,
		Terminal:     session.Terminal,
		SessionTitle: session.Title,
		Content:      content,
	})
	if err != nil {
		return nil, err
	}

	request := llm.AiAssistantRequest{
		Terminal:     session.Terminal,
		Scene:        session.Scene,
		UserName:     userName,
		SessionTitle: session.Title,
		Content:      strings.TrimSpace(content),
		History:      history,
		Attachments:  llmAttachments,
	}
	if toolResult != nil && strings.TrimSpace(toolResult.PromptAugment) != "" {
		request.Content = strings.TrimSpace(content) + "\n\n系统工具结果：\n" + strings.TrimSpace(toolResult.PromptAugment)
	}
	response, err := c.llmClient.GenerateAiAssistantResponse(ctx, request)
	if err == nil {
		if toolResult != nil && len(toolResult.Tools) > 0 {
			response.Source = "tool"
			response.Tools = append([]llm.AiAssistantToolCall(nil), toolResult.Tools...)
		}
		if toolResult != nil && toolResult.Confirm != nil {
			response.Confirm = toLLMConfirmRequestFromDTO(toolResult.Confirm)
		}
		return response, nil
	}

	// 当前系统先以“可用优先”为主，只要模型调用链异常，就统一降级到本地兜底回复，避免聊天接口直接失败。
	fallback := &llm.AiAssistantResponse{
		Content:        buildAiAssistantFallbackReply(session.Scene, content, attachments),
		TokenUsage:     0,
		Source:         "fallback",
		Model:          c.llmClient.Model(),
		Fallback:       true,
		FallbackReason: err.Error(),
	}
	if toolResult != nil && len(toolResult.Tools) > 0 {
		fallback.Tools = append([]llm.AiAssistantToolCall(nil), toolResult.Tools...)
	}
	if toolResult != nil && toolResult.Confirm != nil {
		fallback.Confirm = toLLMConfirmRequestFromDTO(toolResult.Confirm)
	}
	return fallback, nil
}

// createAiAssistantReplyMessages 按回复内容落库助手消息与确认卡消息。
func (c *AiAssistantCase) createAiAssistantReplyMessages(ctx context.Context, session *models.AiAssistantSession, reply *llm.AiAssistantResponse, now time.Time) (*models.AiAssistantMessage, *models.AiAssistantMessage, error) {
	assistantMessage := &models.AiAssistantMessage{
		SessionID:        session.ID,
		UserID:           session.UserID,
		Role:             aiAssistantRoleAssistant,
		Kind:             resolveAiAssistantReplyKind(reply.Tools),
		Content:          marshalAiAssistantReplyContent(reply),
		AttachmentsJSON:  mustMarshalAiAssistantAttachments(nil),
		ToolsJSON:        marshalAiAssistantTools(toAiAssistantToolsDTO(reply.Tools)),
		ConfirmLinesJSON: marshalAiAssistantConfirmLines(nil),
		TokenUsage:       int32(reply.TokenUsage),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err := c.aiAssistantMessageCase.CreateMessage(ctx, assistantMessage)
	if err != nil {
		return nil, nil, err
	}

	var confirmMessage *models.AiAssistantMessage
	if reply != nil && reply.Confirm != nil {
		confirmReply := &llm.AiAssistantResponse{
			Content: "",
			Source:  "tool",
			Model:   c.llmClient.Model(),
			Confirm: reply.Confirm,
		}
		confirmMessage = &models.AiAssistantMessage{
			SessionID:        session.ID,
			UserID:           session.UserID,
			Role:             aiAssistantRoleAssistant,
			Kind:             aiAssistantKindConfirm,
			Content:          marshalAiAssistantReplyContent(confirmReply),
			AttachmentsJSON:  mustMarshalAiAssistantAttachments(nil),
			ToolsJSON:        marshalAiAssistantTools(nil),
			ConfirmTitle:     strings.TrimSpace(reply.Confirm.Title),
			ConfirmLinesJSON: marshalAiAssistantConfirmLines(reply.Confirm.Lines),
			TokenUsage:       0,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		err = c.aiAssistantMessageCase.CreateMessage(ctx, confirmMessage)
		if err != nil {
			return nil, nil, err
		}
	}
	return assistantMessage, confirmMessage, nil
}

// normalizeAiAssistantScene 规范化会话场景。
func normalizeAiAssistantScene(scene string) string {
	switch strings.TrimSpace(scene) {
	case aiAssistantSceneRecommend:
		return aiAssistantSceneRecommend
	case aiAssistantSceneComment:
		return aiAssistantSceneComment
	default:
		return aiAssistantSceneWorkspace
	}
}

// normalizeAiAssistantTerminal 规范化终端类型。
func normalizeAiAssistantTerminal(terminal string) string {
	switch strings.TrimSpace(strings.ToLower(terminal)) {
	case aiAssistantTerminalApp:
		return aiAssistantTerminalApp
	default:
		return aiAssistantTerminalAdmin
	}
}

// normalizeAiAssistantAttachments 清理附件列表。
func normalizeAiAssistantAttachments(values []*basev1.AiAssistantAttachment) []*basev1.AiAssistantAttachment {
	result := make([]*basev1.AiAssistantAttachment, 0, len(values))
	for _, item := range values {
		if item == nil {
			continue
		}
		name := strings.TrimSpace(item.GetName())
		if name == "" {
			name = "未命名附件"
		}
		result = append(result, &basev1.AiAssistantAttachment{
			Id:       strings.TrimSpace(item.GetId()),
			Name:     name,
			Size:     item.GetSize(),
			Url:      strings.TrimSpace(item.GetUrl()),
			MimeType: strings.TrimSpace(item.GetMimeType()),
		})
	}
	return result
}

// buildAiAssistantLLMAttachments 读取附件内容，构造模型输入附件。
func (c *AiAssistantCase) buildAiAssistantLLMAttachments(ctx context.Context, attachments []*basev1.AiAssistantAttachment) ([]llm.AiAssistantAttachment, error) {
	if len(attachments) == 0 {
		return []llm.AiAssistantAttachment{}, nil
	}
	ossClient := sdk.Runtime.GetOSS()
	result := make([]llm.AiAssistantAttachment, 0, len(attachments))
	if ossClient == nil {
		for _, item := range attachments {
			if item == nil {
				continue
			}
			result = append(result, llm.AiAssistantAttachment{
				Name:     item.GetName(),
				Size:     item.GetSize(),
				URL:      item.GetUrl(),
				MIMEType: detectAiAssistantAttachmentMIME(item.GetName(), item.GetMimeType()),
			})
		}
		return result, nil
	}
	for _, item := range attachments {
		if item == nil {
			continue
		}
		next := llm.AiAssistantAttachment{
			Name:     item.GetName(),
			Size:     item.GetSize(),
			URL:      item.GetUrl(),
			MIMEType: detectAiAssistantAttachmentMIME(item.GetName(), item.GetMimeType()),
		}
		if strings.TrimSpace(next.URL) != "" {
			fileBytes, err := ossClient.GetFileByte(next.URL)
			if err != nil {
				return nil, errorsx.Internal("读取 AI 助手附件失败").WithCause(err)
			}
			next.Content = extractAttachmentText(fileBytes, next.MIMEType)
			next.Bytes = fileBytes
		}
		result = append(result, next)
	}
	return result, nil
}

// resolveAiAssistantReplyKind 根据工具调用记录判断消息类型。
func resolveAiAssistantReplyKind(toolCalls []llm.AiAssistantToolCall) string {
	if len(toolCalls) > 0 {
		return aiAssistantKindTool
	}
	return aiAssistantKindText
}

// marshalAiAssistantReplyContent 序列化助手回复内容与元信息。
func marshalAiAssistantReplyContent(response *llm.AiAssistantResponse) string {
	if response == nil {
		return ""
	}
	payload := map[string]any{
		"content":         strings.TrimSpace(response.Content),
		"reply_source":    strings.TrimSpace(response.Source),
		"model":           strings.TrimSpace(response.Model),
		"fallback":        response.Fallback,
		"fallback_reason": strings.TrimSpace(response.FallbackReason),
	}
	if response.Confirm != nil {
		payload["confirm"] = map[string]any{
			"status":      normalizeAiAssistantConfirmStatus(response.Confirm.Status),
			"action":      strings.TrimSpace(response.Confirm.Action),
			"summary":     strings.TrimSpace(response.Confirm.Summary),
			"payload":     response.Confirm.Payload,
			"form_schema": response.Confirm.FormSchema,
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return strings.TrimSpace(response.Content)
	}
	return string(raw)
}

// parseAiAssistantReplyContent 解析助手回复正文。
func parseAiAssistantReplyContent(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	payload := make(map[string]any)
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}
	return strings.TrimSpace(fmt.Sprint(payload["content"]))
}

// parseAiAssistantReplyMeta 解析助手回复元信息。
func parseAiAssistantReplyMeta(raw string) aiAssistantReplyMeta {
	meta := aiAssistantReplyMeta{}
	if strings.TrimSpace(raw) == "" {
		return meta
	}
	payload := make(map[string]any)
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return meta
	}
	meta.ReplySource = strings.TrimSpace(fmt.Sprint(payload["reply_source"]))
	meta.Model = strings.TrimSpace(fmt.Sprint(payload["model"]))
	meta.Fallback = payload["fallback"] == true
	meta.FallbackReason = strings.TrimSpace(fmt.Sprint(payload["fallback_reason"]))
	meta.Confirm = parseAiAssistantConfirmPayload(payload["confirm"])
	return meta
}

// parseAiAssistantConfirmState 解析确认状态。
func parseAiAssistantConfirmState(raw string) *baseDTO.AiAssistantConfirmState {
	return parseAiAssistantReplyMeta(raw).Confirm
}

func aiAssistantConfirmAction(raw string) string {
	confirm := parseAiAssistantConfirmState(raw)
	if confirm == nil {
		return ""
	}
	return confirm.Action
}

func aiAssistantConfirmStatus(raw string) string {
	confirm := parseAiAssistantConfirmState(raw)
	if confirm == nil {
		return ""
	}
	return confirm.Status
}

func aiAssistantConfirmSummary(raw string) string {
	confirm := parseAiAssistantConfirmState(raw)
	if confirm == nil {
		return ""
	}
	return confirm.Summary
}

// toAiAssistantToolsDTO 转换工具调用记录为协议结构。
func toAiAssistantToolsDTO(toolCalls []llm.AiAssistantToolCall) []*basev1.AiAssistantTool {
	result := make([]*basev1.AiAssistantTool, 0, len(toolCalls))
	for _, item := range toolCalls {
		result = append(result, &basev1.AiAssistantTool{
			Name:         item.Name,
			Elapsed:      item.Elapsed,
			Summary:      item.Summary,
			Status:       item.Status,
			ErrorMessage: item.ErrorMessage,
			Input:        item.Input,
		})
	}
	return result
}

// toAiAssistantToolCalls 转换工具展示对象为内部记录。
func toAiAssistantToolCalls(tools []*basev1.AiAssistantTool) []llm.AiAssistantToolCall {
	result := make([]llm.AiAssistantToolCall, 0, len(tools))
	for _, item := range tools {
		if item == nil {
			continue
		}
		result = append(result, llm.AiAssistantToolCall{
			Name:         item.GetName(),
			Status:       item.GetStatus(),
			Elapsed:      item.GetElapsed(),
			Input:        item.GetInput(),
			Summary:      item.GetSummary(),
			ErrorMessage: item.GetErrorMessage(),
		})
	}
	return result
}

func toLLMConfirmRequestFromDTO(confirm *baseDTO.AiAssistantConfirmRequest) *llm.AiAssistantConfirmRequest {
	if confirm == nil {
		return nil
	}
	return &llm.AiAssistantConfirmRequest{
		Title:      confirm.Title,
		Lines:      append([]string(nil), confirm.Lines...),
		Status:     baseDTO.AiAssistantConfirmStatusPending,
		Action:     confirm.Action,
		Summary:    confirm.Summary,
		Payload:    append([]byte(nil), confirm.Payload...),
		FormSchema: toLLMConfirmFormSchema(confirm.FormSchema),
	}
}

func toLLMConfirmRequest(state *baseDTO.AiAssistantConfirmState, title string, lines []string) *llm.AiAssistantConfirmRequest {
	if state == nil {
		return nil
	}
	return &llm.AiAssistantConfirmRequest{
		Title:      title,
		Lines:      append([]string(nil), lines...),
		Status:     normalizeAiAssistantConfirmStatus(state.Status),
		Action:     state.Action,
		Summary:    state.Summary,
		Payload:    append([]byte(nil), state.Payload...),
		FormSchema: toLLMConfirmFormSchema(state.FormSchema),
	}
}

func toLLMConfirmFormSchema(fields []baseDTO.AiAssistantConfirmFormField) []map[string]any {
	result := make([]map[string]any, 0, len(fields))
	for _, field := range fields {
		result = append(result, map[string]any{
			"prop":        field.Prop,
			"label":       field.Label,
			"placeholder": field.Placeholder,
			"required":    field.Required,
		})
	}
	return result
}

// detectAiAssistantAttachmentMIME 规范化附件 MIME 类型。
func detectAiAssistantAttachmentMIME(fileName string, rawMIMEType string) string {
	if strings.TrimSpace(rawMIMEType) != "" {
		return strings.TrimSpace(rawMIMEType)
	}
	return mime.TypeByExtension(strings.ToLower(pathExt(fileName)))
}

// pathExt 提取文件扩展名。
func pathExt(fileName string) string {
	index := strings.LastIndex(strings.TrimSpace(fileName), ".")
	if index < 0 {
		return ""
	}
	return fileName[index:]
}

// extractAttachmentText 提取文本类附件内容。
func extractAttachmentText(fileBytes []byte, mimeType string) string {
	if len(fileBytes) == 0 {
		return ""
	}
	cleanMIMEType := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(cleanMIMEType, "text/") || strings.Contains(cleanMIMEType, "json") || strings.Contains(cleanMIMEType, "xml") || strings.Contains(cleanMIMEType, "csv") {
		text := strings.TrimSpace(string(bytes.TrimSpace(fileBytes)))
		if len([]rune(text)) > 4000 {
			return string([]rune(text)[:4000])
		}
		return text
	}
	return ""
}

// buildAiAssistantDefaultSummary 生成默认场景摘要。
func buildAiAssistantDefaultSummary(scene string) string {
	switch normalizeAiAssistantScene(scene) {
	case aiAssistantSceneRecommend:
		return "推荐分析 · 新会话"
	case aiAssistantSceneComment:
		return "评价中心 · 新会话"
	default:
		return "工作台 · 新会话"
	}
}

// buildAiAssistantDynamicSummary 根据当前问题更新会话摘要。
func buildAiAssistantDynamicSummary(scene string, content string, attachments []*basev1.AiAssistantAttachment) string {
	preview := normalizeAiAssistantPreview(content)
	if preview == "" && len(attachments) > 0 {
		preview = fmt.Sprintf("%d 个附件", len(attachments))
	}
	if preview == "" {
		return buildAiAssistantDefaultSummary(scene)
	}

	switch normalizeAiAssistantScene(scene) {
	case aiAssistantSceneRecommend:
		return "推荐分析 · " + preview
	case aiAssistantSceneComment:
		return "评价中心 · " + preview
	default:
		return "工作台 · " + preview
	}
}

// buildAiAssistantUserContent 在只有附件时补一条可读提示。
func buildAiAssistantUserContent(content string, attachments []*basev1.AiAssistantAttachment) string {
	if strings.TrimSpace(content) != "" {
		return strings.TrimSpace(content)
	}
	if len(attachments) == 0 {
		return ""
	}
	return "请结合附件内容继续分析"
}

// buildAiAssistantFallbackReply 在未启用大模型时返回本地兜底文本。
func buildAiAssistantFallbackReply(scene string, content string, attachments []*basev1.AiAssistantAttachment) string {
	if len(attachments) > 0 {
		return fmt.Sprintf("已收到你的问题和 %d 个附件。我会结合当前系统场景继续整理分析结果。", len(attachments))
	}
	switch normalizeAiAssistantScene(scene) {
	case aiAssistantSceneRecommend:
		return fmt.Sprintf("已收到推荐分析请求：%s。我会围绕推荐链路、热门兜底和曝光波动继续整理建议。", normalizeAiAssistantPreview(content))
	case aiAssistantSceneComment:
		return fmt.Sprintf("已收到评价分析请求：%s。我会优先关注待审核评价、讨论内容和异常风险。", normalizeAiAssistantPreview(content))
	default:
		return fmt.Sprintf("已收到工作台分析请求：%s。我会围绕订单、评价和经营风险继续整理结果。", normalizeAiAssistantPreview(content))
	}
}

// normalizeAiAssistantPreview 截断摘要预览文本。
func normalizeAiAssistantPreview(content string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= aiAssistantPreviewSize {
		return trimmed
	}
	return string(runes[:aiAssistantPreviewSize]) + "..."
}

// mustMarshalAiAssistantAttachments 序列化附件 JSON。
func mustMarshalAiAssistantAttachments(attachments []*basev1.AiAssistantAttachment) string {
	payload := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		payload = append(payload, map[string]any{
			"id":        item.GetId(),
			"name":      item.GetName(),
			"size":      item.GetSize(),
			"url":       item.GetUrl(),
			"mime_type": item.GetMimeType(),
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// marshalAiAssistantTools 序列化工具列表 JSON。
func marshalAiAssistantTools(tools []*basev1.AiAssistantTool) string {
	payload := make([]map[string]any, 0, len(tools))
	for _, item := range tools {
		payload = append(payload, map[string]any{
			"name":          item.GetName(),
			"elapsed":       item.GetElapsed(),
			"summary":       item.GetSummary(),
			"status":        item.GetStatus(),
			"error_message": item.GetErrorMessage(),
			"input":         item.GetInput(),
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// marshalAiAssistantConfirmLines 序列化确认信息 JSON。
func marshalAiAssistantConfirmLines(lines []string) string {
	payload := make([]string, 0, len(lines))
	payload = append(payload, lines...)
	raw, err := json.Marshal(payload)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// parseAiAssistantAttachments 反序列化附件列表。
func parseAiAssistantAttachments(raw string) []*basev1.AiAssistantAttachment {
	if strings.TrimSpace(raw) == "" {
		return []*basev1.AiAssistantAttachment{}
	}
	type attachmentPayload struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Size     int64  `json:"size"`
		URL      string `json:"url"`
		MIMEType string `json:"mime_type"`
	}
	values := make([]attachmentPayload, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []*basev1.AiAssistantAttachment{}
	}
	result := make([]*basev1.AiAssistantAttachment, 0, len(values))
	for _, item := range values {
		result = append(result, &basev1.AiAssistantAttachment{
			Id:       item.ID,
			Name:     item.Name,
			Size:     item.Size,
			Url:      item.URL,
			MimeType: item.MIMEType,
		})
	}
	return result
}

// parseAiAssistantTools 反序列化工具列表。
func parseAiAssistantTools(raw string) []*basev1.AiAssistantTool {
	if strings.TrimSpace(raw) == "" {
		return []*basev1.AiAssistantTool{}
	}
	type toolPayload struct {
		Name         string `json:"name"`
		Elapsed      string `json:"elapsed"`
		Summary      string `json:"summary"`
		Status       string `json:"status"`
		ErrorMessage string `json:"error_message"`
		Input        string `json:"input"`
	}
	values := make([]toolPayload, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []*basev1.AiAssistantTool{}
	}
	result := make([]*basev1.AiAssistantTool, 0, len(values))
	for _, item := range values {
		result = append(result, &basev1.AiAssistantTool{
			Name:         item.Name,
			Elapsed:      item.Elapsed,
			Summary:      item.Summary,
			Status:       item.Status,
			ErrorMessage: item.ErrorMessage,
			Input:        item.Input,
		})
	}
	return result
}

func parseAiAssistantConfirmPayload(value any) *baseDTO.AiAssistantConfirmState {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	confirm := &baseDTO.AiAssistantConfirmState{}
	if err = json.Unmarshal(raw, confirm); err != nil {
		return nil
	}
	confirm.Status = normalizeAiAssistantConfirmStatus(confirm.Status)
	confirm.Action = strings.TrimSpace(confirm.Action)
	confirm.Summary = strings.TrimSpace(confirm.Summary)
	return confirm
}

func normalizeAiAssistantConfirmAction(action string) string {
	switch strings.TrimSpace(strings.ToLower(action)) {
	case "reject":
		return "reject"
	default:
		return "approve"
	}
}

func normalizeAiAssistantConfirmStatus(status string) string {
	switch strings.TrimSpace(status) {
	case baseDTO.AiAssistantConfirmStatusApproved:
		return baseDTO.AiAssistantConfirmStatusApproved
	case baseDTO.AiAssistantConfirmStatusRejected:
		return baseDTO.AiAssistantConfirmStatusRejected
	case baseDTO.AiAssistantConfirmStatusFailed:
		return baseDTO.AiAssistantConfirmStatusFailed
	default:
		return baseDTO.AiAssistantConfirmStatusPending
	}
}

// parseAiAssistantConfirmLines 反序列化确认信息。
func parseAiAssistantConfirmLines(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	values := make([]string, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}
