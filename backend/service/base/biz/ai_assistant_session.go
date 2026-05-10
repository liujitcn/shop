package biz

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

// AiAssistantSessionCase 管理 AI 助手会话数据。
type AiAssistantSessionCase struct {
	*biz.BaseCase
	*data.AiAssistantSessionRepository
	mapper *mapper.CopierMapper[basev1.AiAssistantSession, models.AiAssistantSession]
}

// NewAiAssistantSessionCase 创建 AI 助手会话业务实例。
func NewAiAssistantSessionCase(baseCase *biz.BaseCase, aiAssistantSessionRepo *data.AiAssistantSessionRepository) *AiAssistantSessionCase {
	return &AiAssistantSessionCase{
		BaseCase:                     baseCase,
		AiAssistantSessionRepository: aiAssistantSessionRepo,
		mapper:                       mapper.NewCopierMapper[basev1.AiAssistantSession, models.AiAssistantSession](),
	}
}

// ListByUserAndTerminal 按用户与终端查询会话列表。
func (c *AiAssistantSessionCase) ListByUserAndTerminal(ctx context.Context, userID int64, terminal string) ([]*models.AiAssistantSession, error) {
	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Terminal.Eq(terminal)))
	opts = append(opts, repository.Order(query.LastMessageAt.Desc(), query.ID.Desc()))
	return c.List(ctx, opts...)
}

// CreateSession 创建 AI 助手会话。
func (c *AiAssistantSessionCase) CreateSession(ctx context.Context, model *models.AiAssistantSession) error {
	err := c.Create(ctx, model)
	if err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("AI助手会话创建失败", "ai_assistant_session", "id", "")
		}
		return err
	}
	return nil
}

// FindByUserAndRawID 按用户与字符串会话编号查询会话。
func (c *AiAssistantSessionCase) FindByUserAndRawID(ctx context.Context, userID int64, rawID string) (*models.AiAssistantSession, error) {
	sessionID, err := strconv.ParseInt(strings.TrimSpace(rawID), 10, 64)
	if err != nil || sessionID <= 0 {
		return nil, errorsx.InvalidArgument("会话编号不合法")
	}

	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	session, err := c.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("会话不存在")
		}
		return nil, err
	}
	return session, nil
}

// UpdateTitle 更新会话标题。
func (c *AiAssistantSessionCase) UpdateTitle(ctx context.Context, session *models.AiAssistantSession, title string, now time.Time) error {
	query := c.Query(ctx).AiAssistantSession
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
		UpdateSimple(
			query.Title.Value(title),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	session.Title = title
	session.UpdatedAt = now
	return nil
}

// DeleteSession 删除会话。
func (c *AiAssistantSessionCase) DeleteSession(ctx context.Context, sessionID int64) error {
	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(sessionID)))
	return c.Delete(ctx, opts...)
}

// TouchSession 更新会话摘要、工具数与最后消息时间。
func (c *AiAssistantSessionCase) TouchSession(ctx context.Context, session *models.AiAssistantSession, summary string, toolCount int32, now time.Time) error {
	query := c.Query(ctx).AiAssistantSession
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
		UpdateSimple(
			query.Summary.Value(summary),
			query.ToolCount.Value(toolCount),
			query.LastMessageAt.Value(now),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	session.Summary = summary
	session.ToolCount = toolCount
	session.LastMessageAt = now
	session.UpdatedAt = now
	return nil
}

// RefreshLastMessageAt 更新会话最后消息时间。
func (c *AiAssistantSessionCase) RefreshLastMessageAt(ctx context.Context, session *models.AiAssistantSession, now time.Time) error {
	query := c.Query(ctx).AiAssistantSession
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
		UpdateSimple(
			query.LastMessageAt.Value(now),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	session.LastMessageAt = now
	session.UpdatedAt = now
	return nil
}

// ToDTO 转换会话模型到接口对象。
func (c *AiAssistantSessionCase) ToDTO(model *models.AiAssistantSession) *basev1.AiAssistantSession {
	if model == nil {
		return nil
	}
	session := c.mapper.ToDTO(model)
	session.Id = strconv.FormatInt(model.ID, 10)
	session.UpdatedAt = timestamppb.New(model.UpdatedAt)
	return session
}
