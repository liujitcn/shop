package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/llm"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// AiAssistantMessageCase 管理 AI 助手消息数据。
type AiAssistantMessageCase struct {
	aiAssistantMessageRepo *data.AiAssistantMessageRepository
	mapper                 *mapper.CopierMapper[basev1.AiAssistantMessage, models.AiAssistantMessage]
}

// NewAiAssistantMessageCase 创建 AI 助手消息业务实例。
func NewAiAssistantMessageCase(aiAssistantMessageRepo *data.AiAssistantMessageRepository) *AiAssistantMessageCase {
	return &AiAssistantMessageCase{
		aiAssistantMessageRepo: aiAssistantMessageRepo,
		mapper:                 mapper.NewCopierMapper[basev1.AiAssistantMessage, models.AiAssistantMessage](),
	}
}

// ListBySessionID 按会话查询消息列表。
func (c *AiAssistantMessageCase) ListBySessionID(ctx context.Context, sessionID int64) ([]*models.AiAssistantMessage, error) {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc(), query.ID.Asc()))
	return c.aiAssistantMessageRepo.List(ctx, opts...)
}

// FindBySessionAndRawID 按会话与字符串消息编号查询消息。
func (c *AiAssistantMessageCase) FindBySessionAndRawID(ctx context.Context, sessionID int64, rawID string) (*models.AiAssistantMessage, error) {
	messageID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || messageID <= 0 {
		return nil, errorsx.InvalidArgument("消息编号不合法")
	}

	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.ID.Eq(messageID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Limit(1))
	message, err := c.aiAssistantMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("消息不存在")
		}
		return nil, err
	}
	return message, nil
}

// CreateMessage 创建消息。
func (c *AiAssistantMessageCase) CreateMessage(ctx context.Context, model *models.AiAssistantMessage) error {
	return c.aiAssistantMessageRepo.Create(ctx, model)
}

// UpdateReplyMeta 更新消息回复元信息。
func (c *AiAssistantMessageCase) UpdateReplyMeta(ctx context.Context, message *models.AiAssistantMessage) error {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(message.ID)).
		UpdateSimple(
			query.Content.Value(message.Content),
			query.ToolsJSON.Value(message.ToolsJSON),
			query.UpdatedAt.Value(message.UpdatedAt),
		)
	return err
}

// BuildHistory 构造问答历史上下文。
func (c *AiAssistantMessageCase) BuildHistory(ctx context.Context, sessionID int64, historySize int) ([]llm.AiAssistantMessage, error) {
	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.SessionID.Eq(sessionID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc(), query.ID.Desc()))
	opts = append(opts, repository.Limit(historySize))
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
			Content: parseAiAssistantReplyContent(item.Content),
		})
	}
	return history, nil
}

// ToDTO 转换消息模型到接口对象。
func (c *AiAssistantMessageCase) ToDTO(model *models.AiAssistantMessage) *basev1.AiAssistantMessage {
	if model == nil {
		return nil
	}

	meta := parseAiAssistantReplyMeta(model.Content)
	message := c.mapper.ToDTO(model)
	message.Id = strconv.FormatInt(model.ID, 10)
	message.Content = parseAiAssistantReplyContent(model.Content)
	message.Attachments = parseAiAssistantAttachments(model.AttachmentsJSON)
	message.Tools = parseAiAssistantTools(model.ToolsJSON)
	message.ConfirmLines = parseAiAssistantConfirmLines(model.ConfirmLinesJSON)
	message.CreatedAt = timestamppb.New(model.CreatedAt)
	message.ReplySource = meta.ReplySource
	message.Model = meta.Model
	message.Fallback = meta.Fallback
	message.FallbackReason = meta.FallbackReason
	message.ConfirmAction = aiAssistantConfirmAction(model.Content)
	message.ConfirmStatus = aiAssistantConfirmStatus(model.Content)
	message.ConfirmSummary = aiAssistantConfirmSummary(model.Content)
	return message
}
