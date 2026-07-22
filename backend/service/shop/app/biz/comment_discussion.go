package biz

import (
	"context"
	"errors"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"

	appDto "shop/service/shop/app/dto"
	_const "shop/service/shop/consts"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// CommentDiscussionCase 评价讨论业务处理对象。
type CommentDiscussionCase struct {
	*biz.BaseCase
	*data.CommentDiscussionRepository
	commentReactionRepo *data.CommentReactionRepository
	mapper              *mapper.CopierMapper[shopappv1.CommentDiscussionItem, models.CommentDiscussion]
}

// NewCommentDiscussionCase 创建评价讨论业务处理对象。
func NewCommentDiscussionCase(
	baseCase *biz.BaseCase,
	commentDiscussionRepo *data.CommentDiscussionRepository,
	commentReactionRepo *data.CommentReactionRepository,
) *CommentDiscussionCase {
	return &CommentDiscussionCase{
		BaseCase:                    baseCase,
		CommentDiscussionRepository: commentDiscussionRepo,
		commentReactionRepo:         commentReactionRepo,
		mapper:                      mapper.NewCopierMapper[shopappv1.CommentDiscussionItem, models.CommentDiscussion](),
	}
}

// PageCommentDiscussion 查询评价讨论分页列表。
func (c *CommentDiscussionCase) PageCommentDiscussion(
	ctx context.Context,
	commentID int64,
	userID int64,
	req *shopappv1.PageCommentDiscussionRequest,
) ([]*shopappv1.CommentDiscussionItem, int32, error) {
	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.CommentID.Eq(commentID)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
	opts = append(opts, repository.Where(query.ParentID.Eq(0)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	rootRecordList, total, err := c.Page(ctx, pageNum, pageSize, opts...)
	if err != nil {
		return nil, 0, err
	}

	recordList := rootRecordList
	if len(rootRecordList) > 0 {
		rootDiscussionIDs := make([]int64, 0, len(rootRecordList))
		for _, record := range rootRecordList {
			rootDiscussionIDs = append(rootDiscussionIDs, record.ID)
		}

		replyQuery := c.Query(ctx).CommentDiscussion
		replyOpts := make([]repository.QueryOption, 0, 4)
		replyOpts = append(replyOpts, repository.Where(replyQuery.CommentID.Eq(commentID)))
		replyOpts = append(replyOpts, repository.Where(replyQuery.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
		replyOpts = append(replyOpts, repository.Where(replyQuery.ParentID.In(rootDiscussionIDs...)))
		replyOpts = append(replyOpts, repository.Order(replyQuery.CreatedAt.Asc()))

		var replyRecordList []*models.CommentDiscussion
		replyRecordList, err = c.List(ctx, replyOpts...)
		if err != nil {
			return nil, 0, err
		}
		recordList = append(recordList, replyRecordList...)
	}

	var userReactionTypeMap map[int64]int32
	userReactionTypeMap, err = c.buildDiscussionUserReactionTypeMap(ctx, recordList, userID)
	if err != nil {
		return nil, 0, err
	}

	list := make([]*shopappv1.CommentDiscussionItem, 0, len(recordList))
	for _, record := range recordList {
		list = append(list, c.buildDiscussionItem(record, userReactionTypeMap))
	}
	return list, int32(total), nil
}

// CreateDiscussion 创建评价讨论。
func (c *CommentDiscussionCase) CreateDiscussion(
	ctx context.Context,
	commentInfo *models.CommentInfo,
	user *models.BaseUser,
	req *shopappv1.CreateCommentDiscussionRequest,
) (*models.CommentDiscussion, error) {
	userID := int64(0)
	userName := ANONYMOUS_USER_NAME
	userAvatar := ""
	userTagText := "买家"
	// 查询到了当前用户快照时，优先使用真实用户信息构造讨论归属。
	if user != nil {
		userID = user.ID
		userAvatar = user.Avatar
		// 用户昵称存在时，优先展示昵称。
		if user.NickName != "" {
			userName = user.NickName
		} else if user.UserName != "" {
			// 用户昵称为空时，回退到账号名作为展示昵称。
			userName = user.UserName
		}
	}

	replyToDiscussionID := int64(0)
	replyToUserID := int64(0)
	replyToDisplayName := ""
	var err error
	// 传入父级讨论编号时，要求父级讨论归属于当前评价。
	if req.GetParentId() > 0 {
		var parentRecord *models.CommentDiscussion
		parentRecord, err = c.FindByID(ctx, req.GetParentId())
		if err != nil {
			return nil, err
		}
		// 父级讨论归属评价不一致时，拒绝当前讨论创建请求。
		if parentRecord.CommentID != req.GetCommentId() {
			return nil, errorsx.InvalidArgument("父级讨论不存在")
		}
	}
	// 传入回复目标讨论编号时，要求回复目标归属于当前评价。
	if req.GetReplyToDiscussionId() > 0 {
		var replyRecord *models.CommentDiscussion
		replyRecord, err = c.FindByID(ctx, req.GetReplyToDiscussionId())
		if err != nil {
			return nil, err
		}
		// 回复目标归属评价不一致时，拒绝当前讨论创建请求。
		if replyRecord.CommentID != req.GetCommentId() {
			return nil, errorsx.InvalidArgument("回复目标不存在")
		}
		replyToDiscussionID = replyRecord.ID
		replyToUserID = replyRecord.UserID
		replyToDisplayName = replyRecord.UserNameSnapshot
		// 回复匿名讨论或缺少昵称时，统一展示为匿名用户。
		if replyRecord.IsAnonymous || replyToDisplayName == "" {
			replyToDisplayName = ANONYMOUS_USER_NAME
		}
	}

	record := &models.CommentDiscussion{
		TenantID:            commentInfo.TenantID,
		TenantStoreID:       commentInfo.TenantStoreID,
		CommentID:           req.GetCommentId(),
		UserID:              userID,
		UserNameSnapshot:    userName,
		UserAvatarSnapshot:  userAvatar,
		UserTagText:         userTagText,
		IsAnonymous:         req.GetIsAnonymous(),
		ParentID:            req.GetParentId(),
		ReplyToDiscussionID: replyToDiscussionID,
		ReplyToUserID:       replyToUserID,
		ReplyToDisplayName:  replyToDisplayName,
		Content:             req.GetContent(),
		Status:              _const.COMMENT_STATUS_PENDING_REVIEW,
	}

	err = c.Create(ctx, record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// FindByID 按编号查询审核通过的评价讨论。
func (c *CommentDiscussionCase) FindByID(ctx context.Context, discussionID int64) (*models.CommentDiscussion, error) {
	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(discussionID)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		// 当前讨论不存在时，明确返回资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("讨论不存在")
		}
		return nil, err
	}
	return record, nil
}

// buildDiscussionItem 构造讨论展示项。
func (c *CommentDiscussionCase) buildDiscussionItem(
	record *models.CommentDiscussion,
	userReactionTypeMap map[int64]int32,
) *shopappv1.CommentDiscussionItem {
	item := c.mapper.ToDTO(record)
	userName := record.UserNameSnapshot
	userAvatar := record.UserAvatarSnapshot
	userTagText := record.UserTagText
	// 匿名讨论在前台统一隐藏真实昵称、头像和用户标签。
	if record.IsAnonymous {
		userName = ANONYMOUS_USER_NAME
		userAvatar = ""
		userTagText = ""
	}
	// 未提供展示昵称时，回退到匿名用户文案兜底。
	if userName == "" {
		userName = ANONYMOUS_USER_NAME
	}

	item.Id = record.ID
	item.CommentId = record.CommentID
	item.User = &shopappv1.CommentUserView{
		UserName:    userName,
		Avatar:      userAvatar,
		UserTagText: userTagText,
		Anonymous:   record.IsAnonymous,
	}
	item.ReplyToDisplayName = record.ReplyToDisplayName
	item.DateText = record.CreatedAt.Format("01-02 15:04")
	item.LikeCount = record.LikeCount
	item.ParentId = record.ParentID
	item.ReplyToDiscussionId = record.ReplyToDiscussionID
	reactionType := userReactionTypeMap[record.ID]
	item.ReactionType = shopcommonv1.CommentReactionType(reactionType)
	return item
}

// findAnyByID 按编号查询未删除讨论记录。
func (c *CommentDiscussionCase) findAnyByID(ctx context.Context, discussionID int64) (*models.CommentDiscussion, error) {
	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(discussionID)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// updatePendingStatus 将待审核讨论更新为目标审核状态。
func (c *CommentDiscussionCase) updatePendingStatus(ctx context.Context, discussionID int64, status int32) (bool, error) {
	query := c.Query(ctx).CommentDiscussion
	result, err := query.WithContext(ctx).
		Where(
			query.ID.Eq(discussionID),
			query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW),
		).
		Update(query.Status, status)
	if err != nil {
		return false, err
	}
	return result.RowsAffected > 0, nil
}

// buildDiscussionUserReactionTypeMap 查询当前用户对讨论的互动状态。
func (c *CommentDiscussionCase) buildDiscussionUserReactionTypeMap(ctx context.Context, recordList []*models.CommentDiscussion, userID int64) (map[int64]int32, error) {
	reactionTypeMap := make(map[int64]int32)
	// 未登录或讨论列表为空时，无需查询当前用户互动状态。
	if userID <= 0 || len(recordList) == 0 {
		return reactionTypeMap, nil
	}

	discussionIDs := make([]int64, 0, len(recordList))
	for _, record := range recordList {
		discussionIDs = append(discussionIDs, record.ID)
	}

	query := c.commentReactionRepo.Query(ctx).CommentReaction
	rows := make([]*appDto.CommentTargetReactionRow, 0)
	err := query.WithContext(ctx).
		Select(query.TargetID, query.ReactionType).
		Where(
			query.TargetType.Eq(_const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION),
			query.TargetID.In(discussionIDs...),
			query.UserID.Eq(userID),
		).
		Scan(&rows)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		reactionTypeMap[row.TargetID] = row.ReactionType
	}
	return reactionTypeMap, nil
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
	// 目标讨论不存在时，无法继续回写状态。
	if result.RowsAffected == 0 {
		return errorsx.ResourceNotFound("讨论不存在")
	}
	return nil
}
