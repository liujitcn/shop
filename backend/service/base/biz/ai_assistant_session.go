package biz

import (
	"context"
	"errors"
	"strconv"
	"time"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
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
	tx                     data.Transaction
	aiAssistantMessageRepo *data.AiAssistantMessageRepository
	mapper                 *mapper.CopierMapper[basev1.AiAssistantSession, models.AiAssistantSession]
}

// NewAiAssistantSessionCase 创建 AI 助手会话业务实例。
func NewAiAssistantSessionCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	aiAssistantSessionRepo *data.AiAssistantSessionRepository,
	aiAssistantMessageRepo *data.AiAssistantMessageRepository,
) *AiAssistantSessionCase {
	return &AiAssistantSessionCase{
		BaseCase:                     baseCase,
		AiAssistantSessionRepository: aiAssistantSessionRepo,
		tx:                           tx,
		aiAssistantMessageRepo:       aiAssistantMessageRepo,
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
	opts = append(opts, repository.Order(query.UpdatedAt.Desc(), query.ID.Desc()))
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
		UserID:    authInfo.UserId,
		Terminal:  assistant.NormalizeTerminal(req.GetTerminal()),
		Title:     title,
		Summary:   assistant.BuildDefaultSummary(),
		CreatedAt: now,
		UpdatedAt: now,
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

// CreateAiAssistantSessionBranch 从指定消息创建当前用户的新分支会话。
func (c *AiAssistantSessionCase) CreateAiAssistantSessionBranch(ctx context.Context, req *basev1.CreateAiAssistantSessionBranchRequest) (*basev1.CreateAiAssistantSessionBranchResponse, error) {
	sourceSession, err := c.FindCurrentUserSessionByRawID(ctx, req.GetSourceSessionId())
	if err != nil {
		return nil, err
	}
	var anchorMessageID int64
	anchorMessageID, err = strconv.ParseInt(req.GetAnchorMessageId(), 10, 64)
	if err != nil || anchorMessageID <= 0 {
		return nil, errorsx.InvalidArgument("分支锚点消息编号不合法")
	}

	query := c.aiAssistantMessageRepo.Query(ctx).AiAssistantMessage
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.ID.Eq(anchorMessageID)))
	opts = append(opts, repository.Where(query.SessionID.Eq(sourceSession.ID)))
	opts = append(opts, repository.Where(query.UserID.Eq(sourceSession.UserID)))
	var anchorMessage *models.AiAssistantMessage
	anchorMessage, err = c.aiAssistantMessageRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("分支锚点消息不存在")
		}
		return nil, err
	}

	messageOpts := make([]repository.QueryOption, 0, 6)
	messageOpts = append(messageOpts, repository.Where(query.SessionID.Eq(sourceSession.ID)))
	messageOpts = append(messageOpts, repository.Where(query.UserID.Eq(sourceSession.UserID)))
	messageOpts = append(messageOpts, repository.Where(query.Status.Eq(int32(commonv1.AiAssistantMessageStatus_SUCCESS_AAMS))))
	messageOpts = append(messageOpts, repository.Where(query.CreatedAt.Lte(anchorMessage.CreatedAt)))
	messageOpts = append(messageOpts, repository.Order(query.CreatedAt.Asc(), query.ID.Asc()))
	var sourceMessages []*models.AiAssistantMessage
	sourceMessages, err = c.aiAssistantMessageRepo.List(ctx, messageOpts...)
	if err != nil {
		return nil, err
	}
	if len(sourceMessages) == 0 {
		return nil, errorsx.ResourceNotFound("分支消息不存在")
	}

	now := time.Now()
	title := req.GetTitle()
	if title == "" {
		title = "分支会话"
	}
	branchSession := &models.AiAssistantSession{
		UserID:    sourceSession.UserID,
		Terminal:  assistant.NormalizeTerminal(req.GetTerminal()),
		Title:     title,
		Summary:   sourceSession.Summary,
		CreatedAt: now,
		UpdatedAt: now,
	}
	branchMessages := make([]*models.AiAssistantMessage, 0, len(sourceMessages))
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if createErr := c.Create(txCtx, branchSession); createErr != nil {
			return createErr
		}
		for _, item := range sourceMessages {
			branchMessage := &models.AiAssistantMessage{
				SessionID:     branchSession.ID,
				UserID:        branchSession.UserID,
				InputContent:  item.InputContent,
				OutputContent: item.OutputContent,
				Attachments:   item.Attachments,
				Tools:         item.Tools,
				Token:         item.Token,
				FirstTokenMs:  item.FirstTokenMs,
				DurationMs:    item.DurationMs,
				Status:        item.Status,
				CreatedAt:     item.CreatedAt,
				UpdatedAt:     now,
			}
			if createErr := c.aiAssistantMessageRepo.Create(txCtx, branchMessage); createErr != nil {
				return createErr
			}
			branchMessages = append(branchMessages, branchMessage)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	messages := make([]*basev1.AiAssistantMessage, 0, len(branchMessages))
	for _, item := range branchMessages {
		messages = append(messages, toAiAssistantMessageDTO(item))
	}
	return &basev1.CreateAiAssistantSessionBranchResponse{
		Session:  c.ToDTO(branchSession),
		Messages: messages,
	}, nil
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

// UpdateSessionSummary 更新会话摘要与更新时间。
func (c *AiAssistantSessionCase) UpdateSessionSummary(ctx context.Context, session *models.AiAssistantSession, summary string, now time.Time) error {
	query := c.Query(ctx).AiAssistantSession
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
		UpdateSimple(
			query.Summary.Value(summary),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	session.Summary = summary
	session.UpdatedAt = now
	return nil
}

// RefreshSessionUpdatedAt 更新会话更新时间。
func (c *AiAssistantSessionCase) RefreshSessionUpdatedAt(ctx context.Context, session *models.AiAssistantSession, now time.Time) error {
	query := c.Query(ctx).AiAssistantSession
	_, err := query.WithContext(ctx).
		Where(query.ID.Eq(session.ID)).
		UpdateSimple(
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
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
