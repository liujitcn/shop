package biz

import (
	"context"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/agent/assistant"
	"shop/pkg/agent/stream"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/emptypb"
)

const aiAssistantHistorySize = 12

// AiAssistantCase 管理当前系统 AI 助手会话与消息。
type AiAssistantCase struct {
	*biz.BaseCase
	tx                     data.Transaction
	aiAssistantSessionCase *AiAssistantSessionCase
	aiAssistantMessageCase *AiAssistantMessageCase
	baseUserCase           *BaseUserCase
	assistantRuntime       *assistant.Runtime
	streamPublisher        *stream.Publisher
}

// NewAiAssistantCase 创建 AI 助手业务实例。
func NewAiAssistantCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiAssistantSessionCase *AiAssistantSessionCase,
	aiAssistantMessageCase *AiAssistantMessageCase,
	baseUserCase *BaseUserCase,
	assistantRuntime *assistant.Runtime,
	streamPublisher *stream.Publisher,
) *AiAssistantCase {
	return &AiAssistantCase{
		BaseCase:               baseCase,
		tx:                     tx,
		aiAssistantSessionCase: aiAssistantSessionCase,
		aiAssistantMessageCase: aiAssistantMessageCase,
		baseUserCase:           baseUserCase,
		assistantRuntime:       assistantRuntime,
		streamPublisher:        streamPublisher,
	}
}

// ListAiAssistantSessions 查询当前用户的 AI 助手会话列表。
func (c *AiAssistantCase) ListAiAssistantSessions(ctx context.Context, req *basev1.ListAiAssistantSessionsRequest) (*basev1.ListAiAssistantSessionsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal := assistant.NormalizeTerminal(req.GetTerminal())
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
	scene := assistant.NormalizeScene(req.GetScene())
	now := time.Now()
	model := &models.AiAssistantSession{
		UserID:        authInfo.UserId,
		Terminal:      assistant.NormalizeTerminal(req.GetTerminal()),
		Title:         title,
		Scene:         scene,
		Summary:       assistant.BuildDefaultSummary(scene),
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
	attachments := assistant.NormalizeAttachments(req.GetAttachments())
	if content == "" && len(attachments) == 0 {
		return nil, errorsx.InvalidArgument("消息内容不能为空")
	}
	assistantAttachments, err := c.buildAiAssistantAttachments(ctx, attachments)
	if err != nil {
		return nil, err
	}

	userName, err := c.findAiAssistantUserName(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	userMessage := &models.AiAssistantMessage{
		SessionID:       session.ID,
		UserID:          session.UserID,
		Role:            assistant.RoleUser,
		Kind:            assistant.KindText,
		Content:         assistant.BuildUserContent(content, attachments),
		AttachmentsJSON: assistant.MarshalAttachments(attachments),
		ToolsJSON:       "[]",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	history, err := c.aiAssistantMessageCase.BuildHistory(ctx, session.ID, aiAssistantHistorySize)
	if err != nil {
		return nil, err
	}
	clientMessageID := strings.TrimSpace(req.GetClientMessageId())
	if c.streamPublisher != nil {
		c.streamPublisher.EnsureAdminAssistantStream(session.UserID)
	}
	reply, err := c.generateAiAssistantReply(ctx, session, userName, content, attachments, assistantAttachments, history, func(delta string) {
		if c.streamPublisher == nil {
			return
		}
		_ = c.streamPublisher.PublishDelta(ctx, session.UserID, req.GetSessionId(), clientMessageID, delta)
	})
	if err != nil {
		reply = &assistant.Response{
			Content:        assistant.BuildFallbackReply(session.Scene, content, attachments),
			TokenUsage:     0,
			Source:         "fallback",
			Model:          c.resolveAssistantModel(),
			Fallback:       true,
			FallbackReason: err.Error(),
		}
	}

	responseMessages := make([]*basev1.AiAssistantMessage, 0, 3)
	var assistantMessage *models.AiAssistantMessage
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.aiAssistantMessageCase.CreateMessage(txCtx, userMessage)
		if err != nil {
			return err
		}
		assistantMessage, _, err = c.createAiAssistantReplyMessages(txCtx, session, reply, now)
		if err != nil {
			return err
		}
		summary := assistant.BuildDynamicSummary(session.Scene, content, attachments)
		return c.aiAssistantSessionCase.TouchSession(txCtx, session, summary, 0, now)
	})
	if err != nil {
		return nil, err
	}

	responseMessages = append(responseMessages, c.aiAssistantMessageCase.ToDTO(userMessage))
	responseMessages = append(responseMessages, c.aiAssistantMessageCase.ToDTO(assistantMessage))
	if c.streamPublisher != nil {
		_ = c.streamPublisher.PublishFinish(ctx, session.UserID, req.GetSessionId(), clientMessageID, responseMessages, c.aiAssistantSessionCase.ToDTO(session))
	}

	return &basev1.SendAiAssistantMessageResponse{
		Messages: responseMessages,
		Session:  c.aiAssistantSessionCase.ToDTO(session),
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
	assistantAttachments []assistant.Attachment,
	history []assistant.Message,
	onDelta func(string),
) (*assistant.Response, error) {
	if c.assistantRuntime != nil {
		input := assistant.RuntimeInput{
			Terminal:     assistant.NormalizeTerminalString(session.Terminal),
			Scene:        session.Scene,
			UserName:     userName,
			SessionTitle: session.Title,
			SessionID:    strconv.FormatInt(session.ID, 10),
			Summary:      session.Summary,
			Content:      strings.TrimSpace(content),
			History:      history,
			Attachments:  assistantAttachments,
		}
		if onDelta != nil {
			response, err := c.assistantRuntime.RunStream(ctx, input, onDelta)
			if err == nil {
				return response, nil
			}
			return c.buildAiAssistantFallbackResponse(session, content, attachments, err), nil
		}
		response, err := c.assistantRuntime.Run(ctx, input)
		if err == nil {
			return response, nil
		}
		return c.buildAiAssistantFallbackResponse(session, content, attachments, err), nil
	}

	return c.buildAiAssistantFallbackResponse(session, content, attachments, errorsx.Internal("AI助手运行时未初始化")), nil
}

// createAiAssistantReplyMessages 按回复内容落库助手消息。
func (c *AiAssistantCase) createAiAssistantReplyMessages(ctx context.Context, session *models.AiAssistantSession, reply *assistant.Response, now time.Time) (*models.AiAssistantMessage, *models.AiAssistantMessage, error) {
	assistantMessage := &models.AiAssistantMessage{
		SessionID:       session.ID,
		UserID:          session.UserID,
		Role:            assistant.RoleAssistant,
		Kind:            assistant.KindText,
		Content:         assistant.MarshalReplyContent(reply),
		AttachmentsJSON: assistant.MarshalAttachments(nil),
		ToolsJSON:       "[]",
		TokenUsage:      int32(reply.TokenUsage),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	err := c.aiAssistantMessageCase.CreateMessage(ctx, assistantMessage)
	if err != nil {
		return nil, nil, err
	}
	return assistantMessage, nil, nil
}

// buildAiAssistantAttachments 读取附件内容，构造智能体输入附件。
func (c *AiAssistantCase) buildAiAssistantAttachments(ctx context.Context, attachments []*basev1.AiAssistantAttachment) ([]assistant.Attachment, error) {
	if len(attachments) == 0 {
		return []assistant.Attachment{}, nil
	}
	ossClient := sdk.Runtime.GetOSS()
	result := make([]assistant.Attachment, 0, len(attachments))
	if ossClient == nil {
		for _, item := range attachments {
			if item == nil {
				continue
			}
			result = append(result, assistant.Attachment{
				Name:     item.GetName(),
				Size:     item.GetSize(),
				URL:      item.GetUrl(),
				MIMEType: assistant.DetectAttachmentMIME(item.GetName(), item.GetMimeType()),
			})
		}
		return result, nil
	}
	for _, item := range attachments {
		if item == nil {
			continue
		}
		next := assistant.Attachment{
			Name:     item.GetName(),
			Size:     item.GetSize(),
			URL:      item.GetUrl(),
			MIMEType: assistant.DetectAttachmentMIME(item.GetName(), item.GetMimeType()),
		}
		if strings.TrimSpace(next.URL) != "" {
			fileBytes, err := ossClient.GetFileByte(next.URL)
			if err != nil {
				return nil, errorsx.Internal("读取 AI 助手附件失败").WithCause(err)
			}
			next.Content = assistant.ExtractAttachmentText(fileBytes, next.MIMEType)
			next.Bytes = fileBytes
		}
		result = append(result, next)
	}
	return result, nil
}

// resolveAssistantModel 返回当前 AI 助手使用的模型名称。
func (c *AiAssistantCase) resolveAssistantModel() string {
	if c == nil || c.assistantRuntime == nil {
		return ""
	}
	return c.assistantRuntime.Model()
}

func (c *AiAssistantCase) buildAiAssistantFallbackResponse(
	session *models.AiAssistantSession,
	content string,
	attachments []*basev1.AiAssistantAttachment,
	err error,
) *assistant.Response {
	fallbackReason := ""
	if err != nil {
		fallbackReason = err.Error()
	}
	return &assistant.Response{
		Content:        assistant.BuildFallbackReply(session.Scene, content, attachments),
		TokenUsage:     0,
		Source:         "fallback",
		Model:          c.resolveAssistantModel(),
		Fallback:       true,
		FallbackReason: fallbackReason,
	}
}
