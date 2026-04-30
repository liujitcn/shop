package biz

import (
	"context"
	"errors"
	"strings"

	_const "shop/pkg/const"
	appDto "shop/service/app/dto"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
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
	mapper              *mapper.CopierMapper[appv1.CommentDiscussionItem, models.CommentDiscussion]
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
		mapper:                      mapper.NewCopierMapper[appv1.CommentDiscussionItem, models.CommentDiscussion](),
	}
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

// PageCommentDiscussion 查询评价讨论分页列表。
func (c *CommentDiscussionCase) PageCommentDiscussion(
	ctx context.Context,
	commentID int64,
	userID int64,
	req *appv1.PageCommentDiscussionRequest,
) ([]*appv1.CommentDiscussionItem, int32, error) {
	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	query := c.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.CommentID.Eq(commentID)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	recordList, total, err := c.Page(ctx, pageNum, pageSize, opts...)
	if err != nil {
		return nil, 0, err
	}

	var userReactionTypeMap map[int64]int32
	userReactionTypeMap, err = c.buildDiscussionUserReactionTypeMap(ctx, recordList, userID)
	if err != nil {
		return nil, 0, err
	}

	list := make([]*appv1.CommentDiscussionItem, 0, len(recordList))
	for _, record := range recordList {
		list = append(list, c.buildDiscussionItem(record, userReactionTypeMap))
	}
	return list, int32(total), nil
}

// CreateDiscussion 创建评价讨论。
func (c *CommentDiscussionCase) CreateDiscussion(
	ctx context.Context,
	user *models.BaseUser,
	req *appv1.CreateCommentDiscussionRequest,
) (*models.CommentDiscussion, error) {
	content := strings.TrimSpace(req.GetContent())
	// 讨论内容为空时，不允许创建空讨论。
	if content == "" {
		return nil, errorsx.InvalidArgument("讨论内容不能为空")
	}

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
	// 传入父级讨论编号时，要求父级讨论归属于当前评价。
	if req.GetParentId() > 0 {
		parentRecord, findErr := c.FindByID(ctx, req.GetParentId())
		if findErr != nil {
			return nil, findErr
		}
		// 父级讨论归属评价不一致时，拒绝当前讨论创建请求。
		if parentRecord.CommentID != req.GetCommentId() {
			return nil, errorsx.InvalidArgument("父级讨论不存在")
		}
	}
	// 传入回复目标讨论编号时，要求回复目标归属于当前评价。
	if req.GetReplyToDiscussionId() > 0 {
		replyRecord, findErr := c.FindByID(ctx, req.GetReplyToDiscussionId())
		if findErr != nil {
			return nil, findErr
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
		Content:             content,
		Status:              _const.COMMENT_STATUS_PENDING_REVIEW,
	}

	err := c.Create(ctx, record)
	if err != nil {
		return nil, err
	}
	return record, nil
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

// buildDiscussionItem 构造讨论展示项。
func (c *CommentDiscussionCase) buildDiscussionItem(
	record *models.CommentDiscussion,
	userReactionTypeMap map[int64]int32,
) *appv1.CommentDiscussionItem {
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
	item.User = &appv1.CommentUserView{
		UserName:    userName,
		Avatar:      userAvatar,
		UserTagText: userTagText,
		Anonymous:   record.IsAnonymous,
	}
	item.ReplyToDisplayName = record.ReplyToDisplayName
	item.DateText = record.CreatedAt.Format("01-02 15:04")
	item.LikeCount = record.LikeCount
	reactionType := userReactionTypeMap[record.ID]
	item.ReactionType = commonv1.CommentReactionType(reactionType)
	return item
}
