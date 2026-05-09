package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/llm"

	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
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
	aiAssistantPreviewSize = 18
	aiAssistantHistorySize = 12
)

// AiAssistantCase 管理当前系统 AI 助手会话与消息。
type AiAssistantCase struct {
	*biz.BaseCase
	*data.AiAssistantSessionRepository
	aiAssistantMessageRepo *data.AiAssistantMessageRepository
	baseUserRepo           *data.BaseUserRepository
	llmClient              *llm.Client
}

// NewAiAssistantCase 创建 AI 助手业务实例。
func NewAiAssistantCase(
	baseCase *biz.BaseCase,
	aiAssistantSessionRepo *data.AiAssistantSessionRepository,
	aiAssistantMessageRepo *data.AiAssistantMessageRepository,
	baseUserRepo *data.BaseUserRepository,
	llmClient *llm.Client,
) *AiAssistantCase {
	return &AiAssistantCase{
		BaseCase:                     baseCase,
		AiAssistantSessionRepository: aiAssistantSessionRepo,
		aiAssistantMessageRepo:       aiAssistantMessageRepo,
		baseUserRepo:                 baseUserRepo,
		llmClient:                    llmClient,
	}
}

// ListAiAssistantSessions 查询当前用户的 AI 助手会话列表。
func (c *AiAssistantCase) ListAiAssistantSessions(ctx context.Context, req *basev1.ListAiAssistantSessionsRequest) (*basev1.ListAiAssistantSessionsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal := normalizeAiAssistantTerminal(req.GetTerminal())
	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.Terminal.Eq(terminal)))
	opts = append(opts, repository.Order(query.LastMessageAt.Desc(), query.ID.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	sessions := make([]*basev1.AiAssistantSession, 0, len(list))
	for _, item := range list {
		sessions = append(sessions, c.toAiAssistantSessionDTO(item))
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
	err = c.Create(ctx, model)
	if err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return nil, errorsx.UniqueConflict("AI助手会话创建失败", "ai_assistant_session", "id", "")
		}
		return nil, err
	}
	return c.toAiAssistantSessionDTO(model), nil
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
	query := c.Query(ctx).AiAssistantSession
	now := time.Now()
	_, err = query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
		UpdateSimple(
			query.Title.Value(title),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return nil, err
	}
	session.Title = title
	session.UpdatedAt = now
	return c.toAiAssistantSessionDTO(session), nil
}

// DeleteAiAssistantSession 删除当前用户的会话。
func (c *AiAssistantCase) DeleteAiAssistantSession(ctx context.Context, req *basev1.DeleteAiAssistantSessionRequest) (*emptypb.Empty, error) {
	session, err := c.findAiAssistantSessionByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(session.ID)))
	err = c.AiAssistantSessionRepository.Delete(ctx, opts...)
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
		messages = append(messages, c.toAiAssistantMessageDTO(item))
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
	err = c.aiAssistantMessageRepo.Create(ctx, userMessage)
	if err != nil {
		return nil, err
	}

	history, err := c.buildAiAssistantHistory(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	replyText, tokenUsage, err := c.generateAiAssistantReply(ctx, session, userName, content, attachments, history)
	if err != nil {
		replyText = buildAiAssistantFallbackReply(session.Scene, content, attachments)
		tokenUsage = 0
	}

	assistantMessage := &models.AiAssistantMessage{
		SessionID:        session.ID,
		UserID:           session.UserID,
		Role:             aiAssistantRoleAssistant,
		Kind:             aiAssistantKindText,
		Content:          replyText,
		AttachmentsJSON:  mustMarshalAiAssistantAttachments(nil),
		ToolsJSON:        marshalAiAssistantTools(nil),
		ConfirmLinesJSON: marshalAiAssistantConfirmLines(nil),
		TokenUsage:       int32(tokenUsage),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = c.aiAssistantMessageRepo.Create(ctx, assistantMessage)
	if err != nil {
		return nil, err
	}

	summary := buildAiAssistantDynamicSummary(session.Scene, content, attachments)
	_, err = c.Query(ctx).AiAssistantSession.WithContext(ctx).
		Where(c.Query(ctx).AiAssistantSession.ID.Eq(session.ID)).
		UpdateSimple(
			c.Query(ctx).AiAssistantSession.Summary.Value(summary),
			c.Query(ctx).AiAssistantSession.ToolCount.Value(int32(0)),
			c.Query(ctx).AiAssistantSession.LastMessageAt.Value(now),
			c.Query(ctx).AiAssistantSession.UpdatedAt.Value(now),
		)
	if err != nil {
		return nil, err
	}
	session.Summary = summary
	session.ToolCount = 0
	session.LastMessageAt = now
	session.UpdatedAt = now

	return &basev1.SendAiAssistantMessageResponse{
		Messages: []*basev1.AiAssistantMessage{
			c.toAiAssistantMessageDTO(userMessage),
			c.toAiAssistantMessageDTO(assistantMessage),
		},
		Session: c.toAiAssistantSessionDTO(session),
	}, nil
}

// findAiAssistantSessionByID 按会话编号查询当前用户的会话。
func (c *AiAssistantCase) findAiAssistantSessionByID(ctx context.Context, rawID string) (*models.AiAssistantSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	sessionID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || sessionID <= 0 {
		return nil, errorsx.InvalidArgument("会话编号不合法")
	}

	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	session, err := c.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("会话不存在")
		}
		return nil, err
	}
	return session, nil
}

// findAiAssistantUserName 查询当前会话所属用户名称。
func (c *AiAssistantCase) findAiAssistantUserName(ctx context.Context, userID int64) (string, error) {
	query := c.baseUserRepo.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(userID)))
	opts = append(opts, repository.Limit(1))
	list, err := c.baseUserRepo.List(ctx, opts...)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", errorsx.ResourceNotFound("用户不存在")
	}
	if strings.TrimSpace(list[0].NickName) != "" {
		return list[0].NickName, nil
	}
	return list[0].UserName, nil
}

// buildAiAssistantHistory 构造本次问答的历史消息上下文。
func (c *AiAssistantCase) buildAiAssistantHistory(ctx context.Context, sessionID int64) ([]llm.AiAssistantMessage, error) {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(aiAssistantHistorySize))
	list, err := c.aiAssistantMessageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	history := make([]llm.AiAssistantMessage, 0, len(list))
	for index := len(list) - 1; index >= 0; index-- {
		item := list[index]
		if strings.TrimSpace(item.Content) == "" {
			continue
		}
		history = append(history, llm.AiAssistantMessage{
			Role:    item.Role,
			Content: item.Content,
		})
	}
	return history, nil
}

// generateAiAssistantReply 生成当前消息的 AI 助手回复。
func (c *AiAssistantCase) generateAiAssistantReply(
	ctx context.Context,
	session *models.AiAssistantSession,
	userName string,
	content string,
	attachments []*basev1.AiAssistantAttachment,
	history []llm.AiAssistantMessage,
) (string, int64, error) {
	llmAttachments := make([]llm.AiAssistantAttachment, 0, len(attachments))
	for _, item := range attachments {
		llmAttachments = append(llmAttachments, llm.AiAssistantAttachment{
			Name: item.GetName(),
			Size: item.GetSize(),
		})
	}

	reply, tokenUsage, err := c.llmClient.GenerateAiAssistantReply(ctx, llm.AiAssistantRequest{
		Terminal:     session.Terminal,
		Scene:        session.Scene,
		UserName:     userName,
		SessionTitle: session.Title,
		Content:      strings.TrimSpace(content),
		History:      history,
		Attachments:  llmAttachments,
	})
	if err == nil {
		return reply, tokenUsage, nil
	}

	// 当前系统先以“可用优先”为主，只要模型调用链异常，就统一降级到本地兜底回复，避免聊天接口直接失败。
	return buildAiAssistantFallbackReply(session.Scene, content, attachments), 0, nil
}

// toAiAssistantSessionDTO 转换会话模型到接口对象。
func (c *AiAssistantCase) toAiAssistantSessionDTO(model *models.AiAssistantSession) *basev1.AiAssistantSession {
	if model == nil {
		return nil
	}
	return &basev1.AiAssistantSession{
		Id:        strconv.FormatInt(model.ID, 10),
		Title:     model.Title,
		Scene:     model.Scene,
		Summary:   model.Summary,
		ToolCount: model.ToolCount,
		UpdatedAt: timestamppb.New(model.UpdatedAt),
		Terminal:  model.Terminal,
	}
}

// toAiAssistantMessageDTO 转换消息模型到接口对象。
func (c *AiAssistantCase) toAiAssistantMessageDTO(model *models.AiAssistantMessage) *basev1.AiAssistantMessage {
	if model == nil {
		return nil
	}
	return &basev1.AiAssistantMessage{
		Id:           strconv.FormatInt(model.ID, 10),
		Role:         model.Role,
		Kind:         model.Kind,
		Content:      model.Content,
		Attachments:  parseAiAssistantAttachments(model.AttachmentsJSON),
		Tools:        parseAiAssistantTools(model.ToolsJSON),
		ConfirmTitle: model.ConfirmTitle,
		ConfirmLines: parseAiAssistantConfirmLines(model.ConfirmLinesJSON),
		CreatedAt:    timestamppb.New(model.CreatedAt),
	}
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
			Id:   strings.TrimSpace(item.GetId()),
			Name: name,
			Size: item.GetSize(),
		})
	}
	return result
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
			"id":   item.GetId(),
			"name": item.GetName(),
			"size": item.GetSize(),
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
			"name":    item.GetName(),
			"elapsed": item.GetElapsed(),
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
		ID   string `json:"id"`
		Name string `json:"name"`
		Size int64  `json:"size"`
	}
	values := make([]attachmentPayload, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []*basev1.AiAssistantAttachment{}
	}
	result := make([]*basev1.AiAssistantAttachment, 0, len(values))
	for _, item := range values {
		result = append(result, &basev1.AiAssistantAttachment{
			Id:   item.ID,
			Name: item.Name,
			Size: item.Size,
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
		Name    string `json:"name"`
		Elapsed string `json:"elapsed"`
	}
	values := make([]toolPayload, 0)
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []*basev1.AiAssistantTool{}
	}
	result := make([]*basev1.AiAssistantTool, 0, len(values))
	for _, item := range values {
		result = append(result, &basev1.AiAssistantTool{
			Name:    item.Name,
			Elapsed: item.Elapsed,
		})
	}
	return result
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
