package biz

import (
	"context"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/agent/assistant"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/base/dto"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const aiAssistantHistorySize = 12

// AiAssistantMessageCase 管理 AI 助手消息数据。
type AiAssistantMessageCase struct {
	*biz.BaseCase
	tx                     data.Transaction
	aiAssistantMessageRepo *data.AiAssistantMessageRepository
	aiAssistantSessionCase *AiAssistantSessionCase
	baseUserCase           *BaseUserCase
	assistantRuntime       *assistant.Runtime
	mapper                 *mapper.CopierMapper[basev1.AiAssistantMessage, models.AiAssistantMessage]
}

// NewAiAssistantMessageCase 创建 AI 助手消息业务实例。
func NewAiAssistantMessageCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiAssistantMessageRepo *data.AiAssistantMessageRepository,
	aiAssistantSessionCase *AiAssistantSessionCase,
	baseUserCase *BaseUserCase,
	assistantRuntime *assistant.Runtime,
) *AiAssistantMessageCase {
	return &AiAssistantMessageCase{
		BaseCase:               baseCase,
		tx:                     tx,
		aiAssistantMessageRepo: aiAssistantMessageRepo,
		aiAssistantSessionCase: aiAssistantSessionCase,
		baseUserCase:           baseUserCase,
		assistantRuntime:       assistantRuntime,
		mapper:                 mapper.NewCopierMapper[basev1.AiAssistantMessage, models.AiAssistantMessage](),
	}
}

// ListAiAssistantMessages 查询指定会话的消息列表。
func (c *AiAssistantMessageCase) ListAiAssistantMessages(ctx context.Context, req *basev1.ListAiAssistantMessagesRequest) (*basev1.ListAiAssistantMessagesResponse, error) {
	session, err := c.aiAssistantSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.SessionID.Eq(session.ID)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc(), query.ID.Asc()))
	list, err := c.aiAssistantMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	messages := make([]*basev1.AiAssistantMessage, 0, len(list))
	for _, item := range list {
		messages = append(messages, c.ToDTO(item))
	}
	return &basev1.ListAiAssistantMessagesResponse{Messages: messages}, nil
}

// SendAiAssistantMessage 发送用户消息并生成 AI 助手回复。
func (c *AiAssistantMessageCase) SendAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	session, err := c.aiAssistantSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.GetContent())
	attachments := assistant.NormalizeAttachments(req.GetAttachments())
	if content == "" && len(attachments) == 0 {
		return nil, errorsx.InvalidArgument("消息内容不能为空")
	}
	var assistantAttachments []assistant.Attachment
	assistantAttachments, err = c.buildAiAssistantAttachments(ctx, attachments)
	if err != nil {
		return nil, err
	}

	userName, err := c.baseUserCase.FindDisplayNameByID(ctx, session.UserID)
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

	var history []assistant.Message
	history, err = c.buildHistory(ctx, session.ID, aiAssistantHistorySize)
	if err != nil {
		return nil, err
	}
	var reply *assistant.Response
	reply, err = c.generateAiAssistantReply(ctx, session, userName, content, attachments, assistantAttachments, history, nil)
	if err != nil {
		reply = c.buildAiAssistantFallbackResponse(content, attachments, err)
	}

	responseMessages := make([]*basev1.AiAssistantMessage, 0, 3)
	var assistantMessage *models.AiAssistantMessage
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.aiAssistantMessageRepo.Create(txCtx, userMessage)
		if err != nil {
			return err
		}
		assistantMessage, err = c.createAiAssistantReplyMessage(txCtx, session, reply, now)
		if err != nil {
			return err
		}
		summary := assistant.BuildDynamicSummary(content, attachments)
		return c.aiAssistantSessionCase.TouchSession(txCtx, session, summary, 0, now)
	})
	if err != nil {
		return nil, err
	}

	responseMessages = append(responseMessages, c.ToDTO(userMessage))
	responseMessages = append(responseMessages, c.ToDTO(assistantMessage))

	return &basev1.SendAiAssistantMessageResponse{
		Messages: responseMessages,
		Session:  c.aiAssistantSessionCase.ToDTO(session),
	}, nil
}

// StreamAiAssistantMessage 发送用户消息并流式返回单助手回复。
func (c *AiAssistantMessageCase) StreamAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest, emitter dto.AiAssistantStreamEmitter) error {
	if emitter == nil {
		return errorsx.Internal("AI助手流式响应未初始化")
	}
	session, err := c.aiAssistantSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return err
	}

	content := strings.TrimSpace(req.GetContent())
	attachments := assistant.NormalizeAttachments(req.GetAttachments())
	if content == "" && len(attachments) == 0 {
		return errorsx.InvalidArgument("消息内容不能为空")
	}
	var assistantAttachments []assistant.Attachment
	assistantAttachments, err = c.buildAiAssistantAttachments(ctx, attachments)
	if err != nil {
		return err
	}
	userName, err := c.baseUserCase.FindDisplayNameByID(ctx, session.UserID)
	if err != nil {
		return err
	}
	var history []assistant.Message
	history, err = c.buildHistory(ctx, session.ID, aiAssistantHistorySize)
	if err != nil {
		return err
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
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if createErr := c.aiAssistantMessageRepo.Create(txCtx, userMessage); createErr != nil {
			return createErr
		}
		summary := assistant.BuildDynamicSummary(content, attachments)
		return c.aiAssistantSessionCase.TouchSession(txCtx, session, summary, 0, now)
	})
	if err != nil {
		return err
	}

	clientMessageID := strings.TrimSpace(req.GetClientMessageId())
	reply, runErr := c.generateAiAssistantReply(ctx, session, userName, content, attachments, assistantAttachments, history, func(delta string) {
		if strings.TrimSpace(delta) == "" {
			return
		}
		emitErr := emitter.EmitAiAssistantStream(dto.AiAssistantStreamEventDelta, dto.AiAssistantStreamPayload{
			SessionID:       req.GetSessionId(),
			ClientMessageID: clientMessageID,
			Delta:           delta,
		})
		if emitErr != nil {
			log.Errorf("StreamAiAssistantMessage EmitDelta %v", emitErr)
		}
	})
	if runErr != nil {
		log.Errorf("StreamAiAssistantMessage RunStream %v", runErr)
		// 模型调用异常时继续落一条降级回复，并通过 finish 替换前端占位消息，避免同一轮同时出现失败卡片和兜底回复。
	}

	finishTime := time.Now()
	var assistantMessage *models.AiAssistantMessage
	saveErr := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var createErr error
		assistantMessage, createErr = c.createAiAssistantReplyMessage(txCtx, session, reply, finishTime)
		if createErr != nil {
			return createErr
		}
		return c.aiAssistantSessionCase.RefreshLastMessageAt(txCtx, session, finishTime)
	})
	if saveErr != nil {
		log.Errorf("StreamAiAssistantMessage SaveReply %v", saveErr)
		_ = emitter.EmitAiAssistantStream(dto.AiAssistantStreamEventError, dto.AiAssistantStreamPayload{
			SessionID:       req.GetSessionId(),
			ClientMessageID: clientMessageID,
			ErrorMessage:    saveErr.Error(),
		})
		return nil
	}

	emitErr := emitter.EmitAiAssistantStream(dto.AiAssistantStreamEventFinish, dto.AiAssistantStreamPayload{
		SessionID:       req.GetSessionId(),
		ClientMessageID: clientMessageID,
		Messages:        []*basev1.AiAssistantMessage{c.ToDTO(assistantMessage)},
		Session:         c.aiAssistantSessionCase.ToDTO(session),
	})
	if emitErr != nil {
		log.Errorf("StreamAiAssistantMessage EmitFinish %v", emitErr)
	}
	return nil
}

// buildHistory 构造问答历史上下文。
func (c *AiAssistantMessageCase) buildHistory(ctx context.Context, sessionID int64, historySize int) ([]assistant.Message, error) {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiAssistantMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	history := make([]assistant.Message, 0, len(list))
	for index := len(list) - 1; index >= 0; index-- {
		item := list[index]
		if strings.TrimSpace(item.Content) == "" {
			continue
		}
		history = append(history, assistant.Message{
			Role:    item.Role,
			Content: assistant.ParseReplyContent(item.Content),
		})
	}
	return history, nil
}

// generateAiAssistantReply 生成当前消息的 AI 助手回复。
func (c *AiAssistantMessageCase) generateAiAssistantReply(
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
			UserName:     userName,
			SessionTitle: session.Title,
			SessionID:    strconv.FormatInt(session.ID, 10),
			Summary:      session.Summary,
			Content:      strings.TrimSpace(content),
			History:      history,
			Attachments:  assistantAttachments,
		}
		var err error
		var response *assistant.Response
		if onDelta != nil {
			response, err = c.assistantRuntime.RunStream(ctx, input, onDelta)
			if err == nil {
				return response, nil
			}
			return c.buildAiAssistantFallbackResponse(content, attachments, err), err
		}
		response, err = c.assistantRuntime.Run(ctx, input)
		if err == nil {
			return response, nil
		}
		return c.buildAiAssistantFallbackResponse(content, attachments, err), nil
	}

	err := errorsx.Internal("AI助手运行时未初始化")
	return c.buildAiAssistantFallbackResponse(content, attachments, err), err
}

// createAiAssistantReplyMessage 按回复内容落库助手消息。
func (c *AiAssistantMessageCase) createAiAssistantReplyMessage(ctx context.Context, session *models.AiAssistantSession, reply *assistant.Response, now time.Time) (*models.AiAssistantMessage, error) {
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
	err := c.aiAssistantMessageRepo.Create(ctx, assistantMessage)
	if err != nil {
		return nil, err
	}
	return assistantMessage, nil
}

// buildAiAssistantAttachments 读取附件内容，构造 AI 助手输入附件。
func (c *AiAssistantMessageCase) buildAiAssistantAttachments(ctx context.Context, attachments []*basev1.AiAssistantAttachment) ([]assistant.Attachment, error) {
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

// buildAiAssistantFallbackResponse 构造 AI 助手降级回复。
func (c *AiAssistantMessageCase) buildAiAssistantFallbackResponse(
	content string,
	attachments []*basev1.AiAssistantAttachment,
	err error,
) *assistant.Response {
	fallbackReason := ""
	if err != nil {
		fallbackReason = err.Error()
	}
	model := ""
	if c != nil && c.assistantRuntime != nil {
		model = c.assistantRuntime.Model()
	}
	return &assistant.Response{
		Content:        assistant.BuildFallbackReply(content, attachments),
		TokenUsage:     0,
		Source:         "fallback",
		Model:          model,
		Fallback:       true,
		FallbackReason: fallbackReason,
	}
}

// ToDTO 转换消息模型到接口对象。
func (c *AiAssistantMessageCase) ToDTO(model *models.AiAssistantMessage) *basev1.AiAssistantMessage {
	if model == nil {
		return nil
	}

	meta := assistant.ParseReplyMeta(model.Content)
	message := c.mapper.ToDTO(model)
	message.Id = strconv.FormatInt(model.ID, 10)
	message.Content = assistant.ParseReplyContent(model.Content)
	message.Attachments = assistant.ParseAttachments(model.AttachmentsJSON)
	message.CreatedAt = timestamppb.New(model.CreatedAt)
	message.ReplySource = meta.ReplySource
	message.Model = meta.Model
	message.Fallback = meta.Fallback
	message.FallbackReason = meta.FallbackReason
	return message
}
