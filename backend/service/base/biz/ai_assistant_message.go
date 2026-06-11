package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/agent/assistant"
	assistantflow "shop/pkg/agent/assistant/flow"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/base/dto"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/sdk"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const aiAssistantHistorySize = 12

// AiAssistantMessageCase 管理 AI 助手消息数据。
type AiAssistantMessageCase struct {
	*biz.BaseCase
	tx                     data.Transaction
	aiAssistantMessageRepo *data.AiAssistantMessageRepository
	aiAssistantSessionCase *AiAssistantSessionCase
	baseAPIRepo            *data.BaseAPIRepository
	baseUserCase           *BaseUserCase
	assistantRuntime       *assistant.Runtime
}

// NewAiAssistantMessageCase 创建 AI 助手消息业务实例。
func NewAiAssistantMessageCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiAssistantMessageRepo *data.AiAssistantMessageRepository,
	aiAssistantSessionCase *AiAssistantSessionCase,
	baseAPIRepo *data.BaseAPIRepository,
	baseUserCase *BaseUserCase,
	assistantRuntime *assistant.Runtime,
) *AiAssistantMessageCase {
	c := &AiAssistantMessageCase{
		BaseCase:               baseCase,
		tx:                     tx,
		aiAssistantMessageRepo: aiAssistantMessageRepo,
		aiAssistantSessionCase: aiAssistantSessionCase,
		baseAPIRepo:            baseAPIRepo,
		baseUserCase:           baseUserCase,
		assistantRuntime:       assistantRuntime,
	}
	if assistantRuntime != nil {
		assistantRuntime.SetToolAccessChecker(c)
	}
	return c
}

// SendAiAssistantMessage 发送用户消息并生成 AI 助手回复。
func (c *AiAssistantMessageCase) SendAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	session, message, content, attachments, assistantAttachments, history, userName, err := c.prepareNewAiAssistantMessage(ctx, req)
	if err != nil {
		return nil, err
	}

	startAt := time.Now()
	var reply *assistant.Response
	reply, err = c.generateAiAssistantReply(ctx, session, userName, content, req.GetAction(), attachments, assistantAttachments, history, nil)
	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	firstTokenMs := durationMs
	if err != nil {
		failedReply := c.buildAiAssistantFailedReply(reply, err)
		err = c.finishAiAssistantMessage(ctx, session, message, failedReply, finishAt, firstTokenMs, durationMs, int32(commonv1.AiAssistantMessageStatus_FAILED_AAMS))
		if err != nil {
			return nil, err
		}
		return &basev1.SendAiAssistantMessageResponse{
			Messages: []*basev1.AiAssistantMessage{c.ToDTO(message)},
			Session:  c.aiAssistantSessionCase.ToDTO(session),
		}, nil
	}

	if err = c.finishAiAssistantMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, int32(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS)); err != nil {
		return nil, err
	}
	return &basev1.SendAiAssistantMessageResponse{
		Messages: []*basev1.AiAssistantMessage{c.ToDTO(message)},
		Session:  c.aiAssistantSessionCase.ToDTO(session),
	}, nil
}

// StreamAiAssistantMessage 发送用户消息并流式返回单助手回复。
func (c *AiAssistantMessageCase) StreamAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest, emitter dto.AiAssistantStreamEmitter) error {
	if emitter == nil {
		return errorsx.Internal("AI助手流式响应未初始化")
	}
	session, message, content, attachments, assistantAttachments, history, userName, err := c.prepareNewAiAssistantMessage(ctx, req)
	if err != nil {
		return err
	}

	messageID := strconv.FormatInt(message.ID, 10)
	startAt := time.Now()
	var firstTokenMs int32
	reply, runErr := c.generateAiAssistantReply(ctx, session, userName, content, req.GetAction(), attachments, assistantAttachments, history, func(delta string) {
		if delta == "" {
			return
		}
		if firstTokenMs == 0 {
			firstTokenMs = durationMilliseconds(startAt, time.Now())
		}
		emitErr := emitter.EmitAiAssistantStream(dto.AiAssistantStreamEventDelta, dto.AiAssistantStreamPayload{
			SessionID: req.GetSessionId(),
			MessageID: messageID,
			Delta:     delta,
		})
		if emitErr != nil {
			log.Errorf("StreamAiAssistantMessage EmitDelta %v", emitErr)
		}
	})

	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	if firstTokenMs == 0 && durationMs > 0 {
		firstTokenMs = durationMs
	}
	status := int32(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS)
	if runErr != nil {
		log.Errorf("StreamAiAssistantMessage RunStream %v", runErr)
		reply = c.buildAiAssistantFailedReply(reply, runErr)
		status = int32(commonv1.AiAssistantMessageStatus_FAILED_AAMS)
	}

	saveErr := c.finishAiAssistantMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, status)
	if saveErr != nil {
		log.Errorf("StreamAiAssistantMessage SaveReply %v", saveErr)
		_ = emitter.EmitAiAssistantStream(dto.AiAssistantStreamEventError, dto.AiAssistantStreamPayload{
			SessionID: req.GetSessionId(),
			MessageID: messageID,
		})
		return nil
	}

	emitErr := emitter.EmitAiAssistantStream(dto.AiAssistantStreamEventFinish, dto.AiAssistantStreamPayload{
		SessionID: req.GetSessionId(),
		MessageID: messageID,
		Messages:  []*basev1.AiAssistantMessage{c.ToDTO(message)},
		Session:   c.aiAssistantSessionCase.ToDTO(session),
	})
	if emitErr != nil {
		log.Errorf("StreamAiAssistantMessage EmitFinish %v", emitErr)
	}
	return nil
}

// DeleteAiAssistantMessage 删除当前用户当前会话下的单轮消息。
func (c *AiAssistantMessageCase) DeleteAiAssistantMessage(ctx context.Context, req *basev1.DeleteAiAssistantMessageRequest) error {
	message, _, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return err
	}
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(message.ID)))
	return c.aiAssistantMessageRepo.Delete(ctx, opts...)
}

// UpdateAiAssistantMessage 更新当前用户消息文本并重新生成同一轮助手输出。
func (c *AiAssistantMessageCase) UpdateAiAssistantMessage(ctx context.Context, req *basev1.UpdateAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	content := req.GetContent()
	if content == "" {
		return nil, errorsx.InvalidArgument("消息内容不能为空")
	}
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	err = c.ensureLastAiAssistantMessage(ctx, session.ID, message.ID)
	if err != nil {
		return nil, err
	}
	if message.Status == int32(commonv1.AiAssistantMessageStatus_GENERATING_AAMS) {
		return nil, errorsx.StateConflict("助手回复仍在生成中", "ai_assistant_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS)))
	}
	return c.regenerateAiAssistantMessageWithContent(ctx, session, message, content)
}

// RetryAiAssistantUserMessage 重试失败的 AI 助手消息。
func (c *AiAssistantMessageCase) RetryAiAssistantUserMessage(ctx context.Context, req *basev1.RetryAiAssistantUserMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	if message.Status != int32(commonv1.AiAssistantMessageStatus_FAILED_AAMS) {
		return nil, errorsx.StateConflict("只能重试失败的消息", "ai_assistant_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(commonv1.AiAssistantMessageStatus_FAILED_AAMS)))
	}
	return c.regenerateAiAssistantMessage(ctx, session, message)
}

// RegenerateAiAssistantMessage 重新生成指定 AI 助手消息。
func (c *AiAssistantMessageCase) RegenerateAiAssistantMessage(ctx context.Context, req *basev1.RegenerateAiAssistantMessageRequest) (*basev1.SendAiAssistantMessageResponse, error) {
	message, session, err := c.findCurrentUserMessage(ctx, req.GetSessionId(), req.GetMessageId())
	if err != nil {
		return nil, err
	}
	if message.Status == int32(commonv1.AiAssistantMessageStatus_GENERATING_AAMS) {
		return nil, errorsx.StateConflict("助手回复仍在生成中", "ai_assistant_message", strconv.Itoa(int(message.Status)), strconv.Itoa(int(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS)))
	}
	return c.regenerateAiAssistantMessage(ctx, session, message)
}

// ListAiAssistantMessages 查询指定会话的消息列表。
func (c *AiAssistantMessageCase) ListAiAssistantMessages(ctx context.Context, req *basev1.ListAiAssistantMessagesRequest) (*basev1.ListAiAssistantMessagesResponse, error) {
	session, err := c.aiAssistantSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, err
	}

	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.SessionID.Eq(session.ID)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc(), query.ID.Asc()))
	var list []*models.AiAssistantMessage
	list, err = c.aiAssistantMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	messages := make([]*basev1.AiAssistantMessage, 0, len(list))
	for _, item := range list {
		messages = append(messages, c.ToDTO(item))
	}
	return &basev1.ListAiAssistantMessagesResponse{Messages: messages}, nil
}

// ToDTO 转换消息模型到接口对象。
func (c *AiAssistantMessageCase) ToDTO(model *models.AiAssistantMessage) *basev1.AiAssistantMessage {
	if model == nil {
		return nil
	}

	return toAiAssistantMessageDTO(model)
}

// ToolConfigs 查询当前终端允许暴露给 Agent 的工具配置。
func (c *AiAssistantMessageCase) ToolConfigs(ctx context.Context, terminal string, names []string) (map[string]assistant.ToolConfig, error) {
	result := make(map[string]assistant.ToolConfig)
	if c == nil || c.baseAPIRepo == nil || len(names) == 0 {
		return result, nil
	}
	filteredNames := make([]string, 0, len(names))
	for _, name := range names {
		if !matchAgentToolPrefix(terminal, name) {
			continue
		}
		filteredNames = append(filteredNames, name)
	}
	if len(filteredNames) == 0 {
		return result, nil
	}
	query := c.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.ToolName.In(filteredNames...)))
	list, err := c.baseAPIRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	totalByName := make(map[string]int, len(filteredNames))
	enabledByName := make(map[string]int, len(filteredNames))
	descByName := make(map[string]string, len(filteredNames))
	for _, item := range list {
		totalByName[item.ToolName]++
		if item.AgentEnabled {
			enabledByName[item.ToolName]++
		}
		if descByName[item.ToolName] == "" {
			descByName[item.ToolName] = item.ToolDesc
		}
	}
	for _, name := range filteredNames {
		if totalByName[name] == 0 {
			continue
		}
		result[name] = assistant.ToolConfig{
			Enabled: totalByName[name] == enabledByName[name],
			Desc:    descByName[name],
		}
	}
	return result, nil
}

// toAiAssistantMessageDTO 转换消息模型到接口对象。
func toAiAssistantMessageDTO(model *models.AiAssistantMessage) *basev1.AiAssistantMessage {
	if model == nil {
		return nil
	}
	inputContent := assistant.ParseInputContent(model.InputContent)
	outputContent := assistant.ParseOutputContent(model.OutputContent)
	token := assistant.ParseTokenUsage(model.Token)
	return &basev1.AiAssistantMessage{
		Id:            strconv.FormatInt(model.ID, 10),
		InputContent:  toAiAssistantInputContent(inputContent),
		OutputContent: toAiAssistantOutputContent(outputContent),
		Attachments:   assistant.ParseAttachments(model.Attachments),
		CreatedAt:     timestamppb.New(model.CreatedAt),
		Status:        commonv1.AiAssistantMessageStatus(model.Status),
		Token:         toAiAssistantToken(token),
		Tools:         toAiAssistantTools(assistant.ParseTools(model.Tools)),
		FirstTokenMs:  model.FirstTokenMs,
		DurationMs:    model.DurationMs,
	}
}

// prepareNewAiAssistantMessage 校验请求并创建生成中的消息记录。
func (c *AiAssistantMessageCase) prepareNewAiAssistantMessage(ctx context.Context, req *basev1.SendAiAssistantMessageRequest) (*models.AiAssistantSession, *models.AiAssistantMessage, string, []*basev1.AiAssistantAttachment, []assistant.Attachment, []assistant.Message, string, error) {
	session, err := c.aiAssistantSessionCase.FindCurrentUserSessionByRawID(ctx, req.GetSessionId())
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}

	content := req.GetContent()
	attachments := assistant.NormalizeAttachments(req.GetAttachments())
	if content == "" && len(attachments) == 0 && req.GetAction() == nil {
		return nil, nil, "", nil, nil, nil, "", errorsx.InvalidArgument("消息内容不能为空")
	}
	var assistantAttachments []assistant.Attachment
	assistantAttachments, err = c.buildAiAssistantAttachments(ctx, attachments)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var userName string
	userName, err = c.baseUserCase.FindDisplayNameByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	var history []assistant.Message
	history, err = c.buildHistory(ctx, session.ID, aiAssistantHistorySize)
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}

	now := time.Now()
	message := &models.AiAssistantMessage{
		SessionID:     session.ID,
		UserID:        session.UserID,
		InputContent:  assistant.MarshalInputContent(content, attachments),
		OutputContent: assistant.MarshalEmptyOutputContent(),
		Attachments:   assistant.MarshalAttachments(attachments),
		Tools:         "[]",
		Token:         assistant.MarshalTokenUsage(assistant.TokenUsage{}),
		FirstTokenMs:  0,
		DurationMs:    0,
		Status:        int32(commonv1.AiAssistantMessageStatus_GENERATING_AAMS),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if createErr := c.aiAssistantMessageRepo.Create(txCtx, message); createErr != nil {
			return createErr
		}
		summary := assistant.BuildDynamicSummary(content, attachments)
		return c.aiAssistantSessionCase.UpdateSessionSummary(txCtx, session, summary, now)
	})
	if err != nil {
		return nil, nil, "", nil, nil, nil, "", err
	}
	return session, message, content, attachments, assistantAttachments, history, userName, nil
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
		if next.URL != "" {
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

// buildHistory 构造问答历史上下文。
func (c *AiAssistantMessageCase) buildHistory(ctx context.Context, sessionID int64, historySize int) ([]assistant.Message, error) {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS))))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiAssistantMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return buildHistoryFromMessages(list), nil
}

// buildHistoryBeforeMessage 构造指定消息之前的上下文。
func (c *AiAssistantMessageCase) buildHistoryBeforeMessage(ctx context.Context, sessionID int64, message *models.AiAssistantMessage, historySize int) ([]assistant.Message, error) {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS))))
	opts = append(opts, repository.Where(query.CreatedAt.Lt(message.CreatedAt)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
	list, err := c.aiAssistantMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return buildHistoryFromMessages(list), nil
}

// generateAiAssistantReply 生成当前消息的 AI 助手回复。
func (c *AiAssistantMessageCase) generateAiAssistantReply(
	ctx context.Context,
	session *models.AiAssistantSession,
	userName string,
	content string,
	action *basev1.AiAssistantAction,
	attachments []*basev1.AiAssistantAttachment,
	assistantAttachments []assistant.Attachment,
	history []assistant.Message,
	onDelta func(string),
) (*assistant.Response, error) {
	var err error
	var handled bool
	var flowReply *assistant.Response
	flowReply, handled, err = assistantflow.GenerateReply(ctx, c.assistantRuntime, session.Terminal, content, action)
	if handled {
		// 移动端闭环流程由本地 flow 直接生成结构化回复，先透出正文让前端有流式反馈。
		if err == nil && flowReply != nil && flowReply.Content != "" && onDelta != nil {
			onDelta(flowReply.Content)
		}
		return flowReply, err
	}
	if c.assistantRuntime != nil {
		input := assistant.RuntimeInput{
			Terminal:     assistant.NormalizeTerminalString(session.Terminal),
			UserName:     userName,
			SessionTitle: session.Title,
			SessionID:    strconv.FormatInt(session.ID, 10),
			Summary:      session.Summary,
			Content:      content,
			History:      history,
			Attachments:  assistantAttachments,
		}
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

	err = errorsx.Internal("AI助手运行时未初始化")
	return c.buildAiAssistantFallbackResponse(content, attachments, err), err
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
		Token:          assistant.TokenUsage{},
		Tools:          []assistant.ToolUsage{},
		Source:         "fallback",
		Model:          model,
		Fallback:       true,
		FallbackReason: fallbackReason,
	}
}

// buildAiAssistantFailedReply 构造可展示和可排障的助手异常回复。
func (c *AiAssistantMessageCase) buildAiAssistantFailedReply(reply *assistant.Response, cause error) *assistant.Response {
	failedReply := reply
	if failedReply == nil {
		failedReply = c.buildAiAssistantFallbackResponse("", nil, cause)
	}
	reason := failedReply.FallbackReason
	if reason == "" && cause != nil {
		reason = cause.Error()
	}
	return &assistant.Response{
		Content:        failedReply.Content,
		Token:          failedReply.Token,
		Tools:          failedReply.Tools,
		Source:         "fallback",
		Model:          failedReply.Model,
		Fallback:       true,
		FallbackReason: reason,
	}
}

// regenerateAiAssistantMessage 使用已有输入重新生成当前轮次输出。
func (c *AiAssistantMessageCase) regenerateAiAssistantMessage(ctx context.Context, session *models.AiAssistantSession, message *models.AiAssistantMessage) (*basev1.SendAiAssistantMessageResponse, error) {
	input := assistant.ParseInputContent(message.InputContent)
	content := input.Content
	return c.regenerateAiAssistantMessageWithContent(ctx, session, message, content)
}

// regenerateAiAssistantMessageWithContent 使用指定输入内容重新生成当前轮次输出。
func (c *AiAssistantMessageCase) regenerateAiAssistantMessageWithContent(ctx context.Context, session *models.AiAssistantSession, message *models.AiAssistantMessage, content string) (*basev1.SendAiAssistantMessageResponse, error) {
	attachments := assistant.ParseAttachments(message.Attachments)

	var err error
	var assistantAttachments []assistant.Attachment
	assistantAttachments, err = c.buildAiAssistantAttachments(ctx, attachments)
	if err != nil {
		return nil, err
	}
	var userName string
	userName, err = c.baseUserCase.FindDisplayNameByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	var history []assistant.Message
	history, err = c.buildHistoryBeforeMessage(ctx, session.ID, message, aiAssistantHistorySize)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err = c.markAiAssistantMessageGenerating(ctx, message, content, attachments, now); err != nil {
		return nil, err
	}
	startAt := time.Now()
	var reply *assistant.Response
	reply, err = c.generateAiAssistantReply(ctx, session, userName, content, nil, attachments, assistantAttachments, history, nil)
	finishAt := time.Now()
	durationMs := durationMilliseconds(startAt, finishAt)
	firstTokenMs := durationMs
	status := int32(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS)
	if err != nil {
		reply = c.buildAiAssistantFailedReply(reply, err)
		status = int32(commonv1.AiAssistantMessageStatus_FAILED_AAMS)
	}
	err = c.finishAiAssistantMessage(ctx, session, message, reply, finishAt, firstTokenMs, durationMs, status)
	if err != nil {
		return nil, err
	}
	return &basev1.SendAiAssistantMessageResponse{
		Messages: []*basev1.AiAssistantMessage{c.ToDTO(message)},
		Session:  c.aiAssistantSessionCase.ToDTO(session),
	}, nil
}

// finishAiAssistantMessage 回填当前轮次输出、工具、token 与耗时。
func (c *AiAssistantMessageCase) finishAiAssistantMessage(
	ctx context.Context,
	session *models.AiAssistantSession,
	message *models.AiAssistantMessage,
	reply *assistant.Response,
	now time.Time,
	firstTokenMs int32,
	durationMs int32,
	status int32,
) error {
	outputContent := assistant.MarshalReplyContent(reply)
	tools := assistant.MarshalTools(nil)
	token := assistant.MarshalTokenUsage(assistant.TokenUsage{})
	if reply != nil {
		tools = assistant.MarshalTools(reply.Tools)
		token = assistant.MarshalTokenUsage(reply.Token)
	}
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		query := c.aiAssistantMessageRepo.Query(txCtx).AiAssistantMessage
		_, updateErr := query.WithContext(txCtx).
			Where(query.ID.Eq(message.ID)).
			UpdateSimple(
				query.OutputContent.Value(outputContent),
				query.Tools.Value(tools),
				query.Token.Value(token),
				query.FirstTokenMs.Value(firstTokenMs),
				query.DurationMs.Value(durationMs),
				query.Status.Value(status),
				query.UpdatedAt.Value(now),
			)
		if updateErr != nil {
			return updateErr
		}
		return c.aiAssistantSessionCase.RefreshSessionUpdatedAt(txCtx, session, now)
	})
	if err != nil {
		return err
	}
	message.OutputContent = outputContent
	message.Tools = tools
	message.Token = token
	message.FirstTokenMs = firstTokenMs
	message.DurationMs = durationMs
	message.Status = status
	message.UpdatedAt = now
	return nil
}

// markAiAssistantMessageGenerating 标记消息进入生成中。
func (c *AiAssistantMessageCase) markAiAssistantMessageGenerating(ctx context.Context, message *models.AiAssistantMessage, content string, attachments []*basev1.AiAssistantAttachment, now time.Time) error {
	inputContent := assistant.MarshalInputContent(content, attachments)
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(message.ID)).
		UpdateSimple(
			query.InputContent.Value(inputContent),
			query.OutputContent.Value(assistant.MarshalEmptyOutputContent()),
			query.Tools.Value("[]"),
			query.Token.Value(assistant.MarshalTokenUsage(assistant.TokenUsage{})),
			query.FirstTokenMs.Value(0),
			query.DurationMs.Value(0),
			query.Status.Value(int32(commonv1.AiAssistantMessageStatus_GENERATING_AAMS)),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	message.InputContent = inputContent
	message.OutputContent = assistant.MarshalEmptyOutputContent()
	message.Tools = "[]"
	message.Token = assistant.MarshalTokenUsage(assistant.TokenUsage{})
	message.FirstTokenMs = 0
	message.DurationMs = 0
	message.Status = int32(commonv1.AiAssistantMessageStatus_GENERATING_AAMS)
	message.UpdatedAt = now
	return nil
}

// findCurrentUserMessage 查询当前用户当前会话下的消息。
func (c *AiAssistantMessageCase) findCurrentUserMessage(ctx context.Context, rawSessionID string, rawMessageID string) (*models.AiAssistantMessage, *models.AiAssistantSession, error) {
	session, err := c.aiAssistantSessionCase.FindCurrentUserSessionByRawID(ctx, rawSessionID)
	if err != nil {
		return nil, nil, err
	}
	var messageID int64
	messageID, err = strconv.ParseInt(rawMessageID, 10, 64)
	if err != nil || messageID <= 0 {
		return nil, nil, errorsx.InvalidArgument("消息编号不合法")
	}

	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.ID.Eq(messageID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(session.ID)))
	opts = append(opts, repository.Where(query.UserID.Eq(session.UserID)))
	var message *models.AiAssistantMessage
	message, err = c.aiAssistantMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errorsx.ResourceNotFound("消息不存在")
		}
		return nil, nil, err
	}
	return message, session, nil
}

// ensureLastAiAssistantMessage 确认当前消息是会话最后一轮消息。
func (c *AiAssistantMessageCase) ensureLastAiAssistantMessage(ctx context.Context, sessionID int64, messageID int64) error {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(1))
	lastMessage, err := c.aiAssistantMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("消息不存在")
		}
		return err
	}
	if lastMessage.ID != messageID {
		return errorsx.StateConflict("只能编辑最后一条消息", "ai_assistant_message", strconv.FormatInt(messageID, 10), strconv.FormatInt(lastMessage.ID, 10))
	}
	return nil
}

// matchAgentToolPrefix 判断工具名是否属于当前终端或公共 Base 工具。
func matchAgentToolPrefix(terminal, toolName string) bool {
	return toolName != "" && (terminal == "" || strings.HasPrefix(toolName, terminal+"_") || strings.HasPrefix(toolName, "base_"))
}

// toAiAssistantInputContent 转换输入内容 JSON 为接口对象。
func toAiAssistantInputContent(value assistant.InputContentPayload) *basev1.AiAssistantInputContent {
	return &basev1.AiAssistantInputContent{
		Kind:    value.Kind,
		Content: value.Content,
	}
}

// toAiAssistantOutputContent 转换输出内容 JSON 为接口对象。
func toAiAssistantOutputContent(value assistant.OutputContentPayload) *basev1.AiAssistantOutputContent {
	return &basev1.AiAssistantOutputContent{
		Kind:           value.Kind,
		Content:        value.Content,
		ReplySource:    value.ReplySource,
		Model:          value.Model,
		Fallback:       value.Fallback,
		FallbackReason: value.FallbackReason,
		Flow:           value.Flow,
		Step:           value.Step,
		BlocksJson:     value.BlocksJSON,
	}
}

// toAiAssistantToken 转换 token 统计为接口对象。
func toAiAssistantToken(value assistant.TokenUsage) *basev1.AiAssistantToken {
	return &basev1.AiAssistantToken{
		Input:  value.Input,
		Output: value.Output,
		Cache:  value.Cache,
		Total:  value.Total,
	}
}

// toAiAssistantTools 转换工具使用记录为接口对象。
func toAiAssistantTools(values []assistant.ToolUsage) []*basev1.AiAssistantTool {
	if len(values) == 0 {
		return []*basev1.AiAssistantTool{}
	}
	tools := make([]*basev1.AiAssistantTool, 0, len(values))
	for _, item := range values {
		tools = append(tools, &basev1.AiAssistantTool{
			Type:   item.Type,
			Name:   item.Name,
			Title:  item.Title,
			Status: item.Status,
			Input:  item.Input,
			Output: item.Output,
		})
	}
	return tools
}

// buildHistoryFromMessages 将一轮一行消息拆成模型需要的 user/assistant 上下文。
func buildHistoryFromMessages(list []*models.AiAssistantMessage) []assistant.Message {
	history := make([]assistant.Message, 0, len(list)*2)
	for index := len(list) - 1; index >= 0; index-- {
		item := list[index]
		input := assistant.ParseInputContent(item.InputContent)
		if input.Content != "" {
			history = append(history, assistant.Message{
				Role:    assistant.RoleUser,
				Content: input.Content,
			})
		}
		output := assistant.ParseOutputContent(item.OutputContent)
		if output.Content != "" {
			tools := assistant.ParseTools(item.Tools)
			history = append(history, assistant.Message{
				Role:    assistant.RoleAssistant,
				Content: output.Content,
				Tools:   tools,
			})
		}
	}
	return history
}

// durationMilliseconds 计算两个时间点之间的毫秒数。
func durationMilliseconds(start time.Time, end time.Time) int32 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	ms := end.Sub(start).Milliseconds()
	if ms <= 0 {
		return 0
	}
	if ms > int64(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(ms)
}
