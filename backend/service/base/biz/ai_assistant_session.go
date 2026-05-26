package biz

import (
	"context"
	"errors"
	"strconv"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	"shop/pkg/agent/assistant"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/emptypb"
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

// ListAiAssistantSessions 查询当前用户的 AI 助手会话列表。
func (c *AiAssistantSessionCase) ListAiAssistantSessions(ctx context.Context, req *basev1.ListAiAssistantSessionsRequest) (*basev1.ListAiAssistantSessionsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal := assistant.NormalizeTerminal(req.GetTerminal())
	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	opts = append(opts, repository.Where(query.Terminal.Eq(terminal)))
	opts = append(opts, repository.Order(query.LastMessageAt.Desc(), query.ID.Desc()))
	var list []*models.AiAssistantSession
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	sessions := make([]*basev1.AiAssistantSession, 0, len(list))
	for _, item := range list {
		sessions = append(sessions, c.ToDTO(item))
	}
	return &basev1.ListAiAssistantSessionsResponse{Sessions: sessions}, nil
}

// CreateAiAssistantSession 创建当前用户的新会话。
func (c *AiAssistantSessionCase) CreateAiAssistantSession(ctx context.Context, req *basev1.CreateAiAssistantSessionRequest) (*basev1.AiAssistantSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	title := req.GetTitle()
	if title == "" {
		title = "新对话"
	}
	now := time.Now()
	model := &models.AiAssistantSession{
		UserID:        authInfo.UserId,
		Terminal:      assistant.NormalizeTerminal(req.GetTerminal()),
		Title:         title,
		Summary:       assistant.BuildDefaultSummary(),
		ToolCount:     0,
		LastMessageAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = c.Create(ctx, model); err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return nil, errorsx.UniqueConflict("AI助手会话创建失败", "ai_assistant_session", "id", "")
		}
		return nil, err
	}
	return c.ToDTO(model), nil
}

// UpdateAiAssistantSession 更新当前用户的会话标题。
func (c *AiAssistantSessionCase) UpdateAiAssistantSession(ctx context.Context, req *basev1.UpdateAiAssistantSessionRequest) (*basev1.AiAssistantSession, error) {
	title := req.GetTitle()
	if title == "" {
		return nil, errorsx.InvalidArgument("会话标题不能为空")
	}

	session, err := c.FindCurrentUserSessionByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	now := time.Now()
	query := c.Query(ctx).AiAssistantSession
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
	return c.ToDTO(session), nil
}

// DeleteAiAssistantSession 删除当前用户的会话。
func (c *AiAssistantSessionCase) DeleteAiAssistantSession(ctx context.Context, req *basev1.DeleteAiAssistantSessionRequest) (*emptypb.Empty, error) {
	session, err := c.FindCurrentUserSessionByRawID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(session.ID)))
	if err = c.Delete(ctx, opts...); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// FindCurrentUserSessionByRawID 按当前用户与字符串会话编号查询会话。
func (c *AiAssistantSessionCase) FindCurrentUserSessionByRawID(ctx context.Context, rawID string) (*models.AiAssistantSession, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var sessionID int64
	sessionID, err = strconv.ParseInt(rawID, 10, 64)
	if err != nil || sessionID <= 0 {
		return nil, errorsx.InvalidArgument("会话编号不合法")
	}

	query := c.Query(ctx).AiAssistantSession
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(sessionID)))
	opts = append(opts, repository.Where(query.UserID.Eq(authInfo.UserId)))
	var session *models.AiAssistantSession
	session, err = c.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("会话不存在")
		}
		return nil, err
	}
	return session, nil
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
	session.Terminal = assistant.NormalizeTerminalEnum(model.Terminal)
	return session
}
