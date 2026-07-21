package biz

import (
	"context"
	"errors"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	_const "shop/service/shop/consts"
	"shop/service/shop/queue"
	"shop/service/shop/workspaceevent"
	systemadminbiz "shop/service/system/admin/biz"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// CommentDiscussionCase 评论讨论业务实例。
type CommentDiscussionCase struct {
	*biz.BaseCase
	*data.CommentDiscussionRepository
	tx                data.Transaction
	commentInfoRepo   *data.CommentInfoRepository
	commentReviewCase *CommentReviewCase
	baseUserCase      *systemadminbiz.BaseUserCase
	discussionMapper  *mapper.CopierMapper[shopadminv1.CommentDiscussion, models.CommentDiscussion]
}

// NewCommentDiscussionCase 创建评论讨论业务实例。
func NewCommentDiscussionCase(
	baseCase *biz.BaseCase,
	commentDiscussionRepo *data.CommentDiscussionRepository,
	tx data.Transaction,
	commentInfoRepo *data.CommentInfoRepository,
	commentReviewCase *CommentReviewCase,
	baseUserCase *systemadminbiz.BaseUserCase,
) *CommentDiscussionCase {
	return &CommentDiscussionCase{
		BaseCase:                    baseCase,
		CommentDiscussionRepository: commentDiscussionRepo,
		tx:                          tx,
		commentInfoRepo:             commentInfoRepo,
		commentReviewCase:           commentReviewCase,
		baseUserCase:                baseUserCase,
		discussionMapper:            mapper.NewCopierMapper[shopadminv1.CommentDiscussion, models.CommentDiscussion](),
	}
}

// PageCommentDiscussion 分页查询评论讨论审核列表。
func (c *CommentDiscussionCase) PageCommentDiscussion(ctx context.Context, req *shopadminv1.PageCommentDiscussionRequest) (*shopadminv1.PageCommentDiscussionResponse, error) {
	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Where(query.CommentID.Eq(req.GetCommentId())))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	// 传入用户昵称关键字时，按讨论用户昵称快照模糊匹配。
	if req.GetUserName() != "" {
		opts = append(opts, repository.Where(query.UserNameSnapshot.Like("%"+req.GetUserName()+"%")))
	}
	// 传入讨论内容关键字时，按讨论正文模糊匹配。
	if req.GetContent() != "" {
		opts = append(opts, repository.Where(query.Content.Like("%"+req.GetContent()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*shopadminv1.CommentDiscussion, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.discussionMapper.ToDTO(item))
	}
	return &shopadminv1.PageCommentDiscussionResponse{CommentDiscussions: resList, Total: int32(total)}, nil
}

// ListByCommentIDs 查询评论讨论列表。
func (c *CommentDiscussionCase) ListByCommentIDs(ctx context.Context, commentIDs []int64) ([]*shopadminv1.CommentDiscussion, error) {
	resList := make([]*shopadminv1.CommentDiscussion, 0)
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

// SetCommentDiscussionStatus 设置评论讨论审核状态。
func (c *CommentDiscussionCase) SetCommentDiscussionStatus(ctx context.Context, req *shopadminv1.SetCommentDiscussionStatusRequest) error {
	err := validateCommentStatus(int32(req.GetStatus()))
	if err != nil {
		return err
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var discussion *models.CommentDiscussion
	discussion, err = c.findAnyByID(ctx, req.GetId())
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
			TenantID:      discussion.TenantID,
			TenantStoreID: discussion.TenantStoreID,
			TargetType:    _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION,
			TargetID:      discussion.ID,
			Type:          _const.COMMENT_REVIEW_TYPE_MANUAL,
			Status:        commentReviewStatusByCommentStatus(int32(req.GetStatus())),
			Tags:          _string.ConvertAnyToJsonString([]string{}),
			OperatorID:    authInfo.UserId,
			OperatorName:  operatorName,
			Reason:        req.GetReason(),
		})
	})
	if err != nil {
		return err
	}
	// 讨论人工审核通过后，刷新所属商品 评价摘要。
	if int32(req.GetStatus()) == _const.COMMENT_STATUS_APPROVED {
		var commentInfo *models.CommentInfo
		commentInfo, err = c.findCommentInfoByID(ctx, discussion.CommentID)
		if err == nil {
			queue.DispatchCommentSummaryRefresh(commentInfo.GoodsID)
		}
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaReputation)
	return nil
}

// operatorName 查询后台操作人展示名称。
func (c *CommentDiscussionCase) operatorName(ctx context.Context, userID int64) string {
	baseUser, err := c.baseUserCase.FindByID(ctx, userID)
	if err != nil || baseUser == nil {
		return "管理员"
	}
	operatorName := baseUser.NickName
	// 后台用户昵称为空时，回退到账号名作为操作人名称。
	if operatorName == "" {
		operatorName = baseUser.UserName
	}
	if operatorName == "" {
		return "管理员"
	}
	return operatorName
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
	var update field.AssignExpr
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
