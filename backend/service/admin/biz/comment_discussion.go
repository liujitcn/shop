package biz

import (
	"context"
	"errors"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"
	"shop/pkg/workspaceevent"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
	"gorm.io/gorm"
)

// CommentDiscussionCase 评论讨论业务实例。
type CommentDiscussionCase struct {
	*biz.BaseCase
	*data.CommentDiscussionRepository
	tx                data.Transaction
	commentInfoRepo   *data.CommentInfoRepository
	commentReviewCase *CommentReviewCase
	baseUserCase      *BaseUserCase
	discussionMapper  *mapper.CopierMapper[adminv1.CommentDiscussion, models.CommentDiscussion]
}

// NewCommentDiscussionCase 创建评论讨论业务实例。
func NewCommentDiscussionCase(
	baseCase *biz.BaseCase,
	commentDiscussionRepo *data.CommentDiscussionRepository,
	tx data.Transaction,
	commentInfoRepo *data.CommentInfoRepository,
	commentReviewCase *CommentReviewCase,
	baseUserCase *BaseUserCase,
) *CommentDiscussionCase {
	return &CommentDiscussionCase{
		BaseCase:                    baseCase,
		CommentDiscussionRepository: commentDiscussionRepo,
		tx:                          tx,
		commentInfoRepo:             commentInfoRepo,
		commentReviewCase:           commentReviewCase,
		baseUserCase:                baseUserCase,
		discussionMapper:            mapper.NewCopierMapper[adminv1.CommentDiscussion, models.CommentDiscussion](),
	}
}

// PageCommentDiscussions 分页查询评论讨论审核列表。
func (c *CommentDiscussionCase) PageCommentDiscussions(ctx context.Context, req *adminv1.PageCommentDiscussionsRequest) (*adminv1.PageCommentDiscussionsResponse, error) {
	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Where(query.CommentID.Eq(req.GetCommentId())))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入用户昵称关键字时，按讨论用户昵称快照模糊匹配。
	if strings.TrimSpace(req.GetUserName()) != "" {
		opts = append(opts, repository.Where(query.UserNameSnapshot.Like("%"+strings.TrimSpace(req.GetUserName())+"%")))
	}
	// 传入讨论内容关键字时，按讨论正文模糊匹配。
	if strings.TrimSpace(req.GetContent()) != "" {
		opts = append(opts, repository.Where(query.Content.Like("%"+strings.TrimSpace(req.GetContent())+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.CommentDiscussion, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.discussionMapper.ToDTO(item))
	}
	return &adminv1.PageCommentDiscussionsResponse{CommentDiscussions: resList, Total: int32(total)}, nil
}

// SetCommentDiscussionStatus 设置评论讨论审核状态。
func (c *CommentDiscussionCase) SetCommentDiscussionStatus(ctx context.Context, req *adminv1.SetCommentDiscussionStatusRequest) error {
	err := validateCommentStatus(int32(req.GetStatus()))
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	discussion, err := c.findAnyByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	operatorName := c.operatorName(ctx, authInfo.UserId)
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		updateErr := c.updateStatus(txCtx, discussion.ID, int32(req.GetStatus()))
		if updateErr != nil {
			return updateErr
		}
		updateErr = c.applyDiscussionCountChange(txCtx, discussion.CommentID, discussion.Status, int32(req.GetStatus()))
		if updateErr != nil {
			return updateErr
		}
		return c.commentReviewCase.CreateReview(txCtx, &models.CommentReview{
			TargetType:   _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION,
			TargetID:     discussion.ID,
			Type:         _const.COMMENT_REVIEW_TYPE_MANUAL,
			Status:       commentReviewStatusByCommentStatus(int32(req.GetStatus())),
			Tags:         _string.ConvertAnyToJsonString([]string{}),
			OperatorID:   authInfo.UserId,
			OperatorName: operatorName,
			Reason:       strings.TrimSpace(req.GetReason()),
		})
	})
	if err != nil {
		return err
	}
	// 讨论人工审核通过后，刷新所属商品 AI 摘要。
	if int32(req.GetStatus()) == _const.COMMENT_STATUS_APPROVED {
		commentInfo, findErr := c.findCommentInfoByID(ctx, discussion.CommentID)
		if findErr == nil {
			queue.DispatchCommentAiRefresh(commentInfo.GoodsID)
		}
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaReputation)
	return nil
}

// ListByCommentIDs 查询评论讨论列表。
func (c *CommentDiscussionCase) ListByCommentIDs(ctx context.Context, commentIDs []int64) ([]*adminv1.CommentDiscussion, error) {
	resList := make([]*adminv1.CommentDiscussion, 0)
	// 没有评论编号时，直接返回空讨论列表。
	if len(commentIDs) == 0 {
		return resList, nil
	}

	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.CommentID.In(commentIDs...)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	for _, item := range list {
		resList = append(resList, c.discussionMapper.ToDTO(item))
	}
	return resList, nil
}

// findAnyByID 按编号查询未删除讨论。
func (c *CommentDiscussionCase) findAnyByID(ctx context.Context, discussionID int64) (*models.CommentDiscussion, error) {
	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(discussionID)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("讨论不存在")
		}
		return nil, err
	}
	return record, nil
}

// findCommentInfoByID 按编号查询评价主记录。
func (c *CommentDiscussionCase) findCommentInfoByID(ctx context.Context, commentID int64) (*models.CommentInfo, error) {
	query := c.commentInfoRepo.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(commentID)))
	record, err := c.commentInfoRepo.Find(ctx, opts...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("评价不存在")
		}
		return nil, err
	}
	return record, nil
}

// updateStatus 更新讨论审核状态。
func (c *CommentDiscussionCase) updateStatus(ctx context.Context, discussionID int64, status int32) error {
	query := c.Query(ctx).CommentDiscussion
	result, err := query.WithContext(ctx).
		Where(query.ID.Eq(discussionID)).
		Update(query.Status, status)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errorsx.ResourceNotFound("讨论不存在")
	}
	return nil
}

// applyDiscussionCountChange 根据讨论状态变化同步评价主表缓存数量。
func (c *CommentDiscussionCase) applyDiscussionCountChange(ctx context.Context, commentID int64, oldStatus int32, newStatus int32) error {
	if oldStatus == newStatus {
		return nil
	}
	err := c.changeDiscussionCount(ctx, commentID, oldStatus, -1)
	if err != nil {
		return err
	}
	return c.changeDiscussionCount(ctx, commentID, newStatus, 1)
}

// changeDiscussionCount 调整评价主表中的讨论状态缓存数量。
func (c *CommentDiscussionCase) changeDiscussionCount(ctx context.Context, commentID int64, status int32, delta int32) error {
	if delta == 0 {
		return nil
	}
	query := c.commentInfoRepo.Query(ctx).CommentInfo
	update := query.DiscussionCount.Add(delta)
	conditions := []gen.Condition{query.ID.Eq(commentID)}
	switch status {
	case _const.COMMENT_STATUS_PENDING_REVIEW:
		update = query.PendingDiscussionCount.Add(delta)
		if delta < 0 {
			conditions = append(conditions, query.PendingDiscussionCount.Gt(0))
		}
	case _const.COMMENT_STATUS_APPROVED:
		update = query.DiscussionCount.Add(delta)
		if delta < 0 {
			conditions = append(conditions, query.DiscussionCount.Gt(0))
		}
	default:
		return nil
	}
	_, err := query.WithContext(ctx).Where(conditions...).UpdateSimple(update)
	return err
}

// operatorName 查询后台操作人展示名称。
func (c *CommentDiscussionCase) operatorName(ctx context.Context, userID int64) string {
	baseUser, err := c.baseUserCase.FindByID(ctx, userID)
	if err != nil || baseUser == nil {
		return "管理员"
	}
	operatorName := strings.TrimSpace(baseUser.NickName)
	// 后台用户昵称为空时，回退到账号名作为操作人名称。
	if operatorName == "" {
		operatorName = strings.TrimSpace(baseUser.UserName)
	}
	if operatorName == "" {
		return "管理员"
	}
	return operatorName
}
