package biz

import (
	"context"
	"strconv"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/agent/assistant"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
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

// CreateMessage 创建消息。
func (c *AiAssistantMessageCase) CreateMessage(ctx context.Context, model *models.AiAssistantMessage) error {
	return c.aiAssistantMessageRepo.Create(ctx, model)
}

// BuildHistory 构造问答历史上下文。
func (c *AiAssistantMessageCase) BuildHistory(ctx context.Context, sessionID int64, historySize int) ([]assistant.Message, error) {
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
