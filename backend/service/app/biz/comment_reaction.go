package biz

import (
	"context"
	"errors"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
	"gorm.io/gorm"
)

// CommentReactionCase 评价互动业务处理对象。
type CommentReactionCase struct {
	*biz.BaseCase
	*data.CommentReactionRepository
	commentInfoRepo       *data.CommentInfoRepository
	commentAiRepo         *data.CommentAiRepository
	commentDiscussionRepo *data.CommentDiscussionRepository
	reactionMapper        *mapper.CopierMapper[appv1.SaveCommentReactionRequest, models.CommentReaction]
}

// NewCommentReactionCase 创建评价互动业务处理对象。
func NewCommentReactionCase(
	baseCase *biz.BaseCase,
	commentReactionRepo *data.CommentReactionRepository,
	commentInfoRepo *data.CommentInfoRepository,
	commentAiRepo *data.CommentAiRepository,
	commentDiscussionRepo *data.CommentDiscussionRepository,
) *CommentReactionCase {
	return &CommentReactionCase{
		BaseCase:                  baseCase,
		CommentReactionRepository: commentReactionRepo,
		commentInfoRepo:           commentInfoRepo,
		commentAiRepo:             commentAiRepo,
		commentDiscussionRepo:     commentDiscussionRepo,
		reactionMapper:            mapper.NewCopierMapper[appv1.SaveCommentReactionRequest, models.CommentReaction](),
	}
}

// SaveCommentReaction 保存评价互动状态。
func (c *CommentReactionCase) SaveCommentReaction(
	ctx context.Context,
	userID int64,
	req *appv1.SaveCommentReactionRequest,
) (*appv1.SaveCommentReactionResponse, error) {
	reactionQuery := c.Query(ctx).CommentReaction
	reactionOpts := make([]repository.QueryOption, 0, 3)
	reactionOpts = append(reactionOpts, repository.Where(reactionQuery.TargetType.Eq(int32(req.GetTargetType()))))
	reactionOpts = append(reactionOpts, repository.Where(reactionQuery.TargetID.Eq(req.GetTargetId())))
	reactionOpts = append(reactionOpts, repository.Where(reactionQuery.UserID.Eq(userID)))
	reaction, err := c.Find(ctx, reactionOpts...)
	hasOldReaction := err == nil
	// 查询失败且不是未互动记录时，说明底层查询异常，需要中断本次互动保存。
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	response := &appv1.SaveCommentReactionResponse{
		TargetType: req.GetTargetType(),
		TargetId:   req.GetTargetId(),
	}
	currentReactionType := int32(0)
	hasCurrentReaction := false
	oldReactionType := int32(0)
	if hasOldReaction {
		oldReactionType = reaction.ReactionType
	}

	// 不同互动目标使用各自的计数回写和状态切换逻辑。
	switch req.GetTargetType() {
	case commonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_AI):
		aiQuery := c.commentAiRepo.Query(ctx).CommentAi
		aiOpts := make([]repository.QueryOption, 0, 1)
		aiOpts = append(aiOpts, repository.Where(aiQuery.ID.Eq(req.GetTargetId())))
		_, err = c.commentAiRepo.Find(ctx, aiOpts...)
		if err != nil {
			// 互动目标 AI 摘要不存在时，拒绝保存当前互动状态。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errorsx.ResourceNotFound("评价摘要不存在")
			}
			return nil, err
		}

		// 当前请求需要开启某个互动状态时，按最新状态覆盖旧值。
		if req.GetActive() {
			if !hasOldReaction {
				reaction = c.reactionMapper.ToEntity(req)
				reaction.UserID = userID
				err = c.Create(ctx, reaction)
				if err != nil {
					return nil, err
				}
			} else if reaction.ReactionType != int32(req.GetReactionType()) {
				// 已有互动状态与本次不一致时，覆盖为最新互动类型。
				_, err = reactionQuery.WithContext(ctx).
					Where(reactionQuery.ID.Eq(reaction.ID)).
					Update(reactionQuery.ReactionType, int32(req.GetReactionType()))
				if err != nil {
					return nil, err
				}
				reaction.ReactionType = int32(req.GetReactionType())
			}
			hasCurrentReaction = true
			currentReactionType = int32(req.GetReactionType())
		} else if hasOldReaction && reaction.ReactionType == int32(req.GetReactionType()) {
			// 当前关闭的是已存在互动状态时，删除互动明细。
			deleteOpts := make([]repository.QueryOption, 0, 1)
			deleteOpts = append(deleteOpts, repository.Where(reactionQuery.ID.Eq(reaction.ID)))
			err = c.Delete(ctx, deleteOpts...)
			if err != nil {
				return nil, err
			}
		} else if hasOldReaction {
			// 关闭的互动类型与历史状态不一致时，保留历史互动状态不变。
			hasCurrentReaction = true
			currentReactionType = reaction.ReactionType
		}
	case commonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION):
		discussionQuery := c.commentDiscussionRepo.Query(ctx).CommentDiscussion
		discussionOpts := make([]repository.QueryOption, 0, 1)
		discussionOpts = append(discussionOpts, repository.Where(discussionQuery.ID.Eq(req.GetTargetId())))
		_, err = c.commentDiscussionRepo.Find(ctx, discussionOpts...)
		if err != nil {
			// 互动目标讨论不存在时，拒绝保存当前互动状态。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errorsx.ResourceNotFound("讨论不存在")
			}
			return nil, err
		}

		// 当前请求需要开启讨论点赞状态时，仅在状态首次命中时增加计数。
		if req.GetActive() {
			if !hasOldReaction {
				reaction = c.reactionMapper.ToEntity(req)
				reaction.UserID = userID
				err = c.Create(ctx, reaction)
				if err != nil {
					return nil, err
				}
			} else if reaction.ReactionType != _const.COMMENT_REACTION_TYPE_LIKE {
				// 历史状态异常不是点赞时，切换为讨论点赞状态。
				_, err = reactionQuery.WithContext(ctx).
					Where(reactionQuery.ID.Eq(reaction.ID)).
					Update(reactionQuery.ReactionType, _const.COMMENT_REACTION_TYPE_LIKE)
				if err != nil {
					return nil, err
				}
				reaction.ReactionType = _const.COMMENT_REACTION_TYPE_LIKE
			}
			hasCurrentReaction = true
			currentReactionType = _const.COMMENT_REACTION_TYPE_LIKE
		} else if hasOldReaction && reaction.ReactionType == int32(req.GetReactionType()) {
			// 关闭讨论点赞时，删除互动明细并同步扣减讨论点赞缓存。
			deleteOpts := make([]repository.QueryOption, 0, 1)
			deleteOpts = append(deleteOpts, repository.Where(reactionQuery.ID.Eq(reaction.ID)))
			err = c.Delete(ctx, deleteOpts...)
			if err != nil {
				return nil, err
			}
		} else if hasOldReaction {
			// 关闭的互动类型与历史讨论点赞状态不一致时，保留历史状态不变。
			hasCurrentReaction = true
			currentReactionType = reaction.ReactionType
		}
	case commonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_COMMENT):
		commentQuery := c.Query(ctx).CommentInfo
		// 评价互动需要校验目标评价已审核通过，避免对待审核或不存在评价写入互动状态。
		_, err = commentQuery.WithContext(ctx).
			Where(commentQuery.ID.Eq(req.GetTargetId()), commentQuery.Status.Eq(_const.COMMENT_STATUS_APPROVED)).
			First()
		if err != nil {
			// 互动目标评价不存在或未审核通过时，拒绝保存当前互动状态。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errorsx.ResourceNotFound("评价不存在")
			}
			return nil, err
		}

		// 当前请求需要开启评价互动状态时，首次互动创建记录，已有状态按本次互动类型覆盖。
		if req.GetActive() {
			if !hasOldReaction {
				reaction = c.reactionMapper.ToEntity(req)
				reaction.UserID = userID
				err = c.Create(ctx, reaction)
				if err != nil {
					return nil, err
				}
			} else if reaction.ReactionType != int32(req.GetReactionType()) {
				// 已有评价互动状态与本次不一致时，覆盖为最新点赞或点踩类型。
				_, err = reactionQuery.WithContext(ctx).
					Where(reactionQuery.ID.Eq(reaction.ID)).
					Update(reactionQuery.ReactionType, int32(req.GetReactionType()))
				if err != nil {
					return nil, err
				}
				reaction.ReactionType = int32(req.GetReactionType())
			}
			hasCurrentReaction = true
			currentReactionType = int32(req.GetReactionType())
		} else if hasOldReaction && reaction.ReactionType == int32(req.GetReactionType()) {
			// 当前关闭的是已存在评价互动状态时，删除互动明细。
			deleteOpts := make([]repository.QueryOption, 0, 1)
			deleteOpts = append(deleteOpts, repository.Where(reactionQuery.ID.Eq(reaction.ID)))
			err = c.Delete(ctx, deleteOpts...)
			if err != nil {
				return nil, err
			}
		} else if hasOldReaction {
			// 关闭的互动类型与历史评价互动状态不一致时，保留历史状态不变。
			hasCurrentReaction = true
			currentReactionType = reaction.ReactionType
		}
	default:
		return nil, errorsx.InvalidArgument("互动目标类型不支持")
	}

	newReactionType := int32(0)
	if hasCurrentReaction {
		newReactionType = currentReactionType
	}
	err = c.applyReactionCounterChange(ctx, int32(req.GetTargetType()), req.GetTargetId(), oldReactionType, newReactionType)
	if err != nil {
		return nil, err
	}

	likeCount, dislikeCount, err := c.getCachedReactionCounts(ctx, int32(req.GetTargetType()), req.GetTargetId())
	if err != nil {
		return nil, err
	}
	response.LikeCount = likeCount
	response.DislikeCount = dislikeCount

	// 当前用户仍保留互动状态时，回填点赞 / 点踩展示状态。
	if hasCurrentReaction {
		response.ReactionType = commonv1.CommentReactionType(currentReactionType)
	}
	return response, nil
}

// applyReactionCounterChange 根据互动状态变化同步目标表缓存数量。
func (c *CommentReactionCase) applyReactionCounterChange(ctx context.Context, targetType int32, targetID int64, oldReactionType int32, newReactionType int32) error {
	// 互动状态未变化时，无需调整缓存数量。
	if oldReactionType == newReactionType {
		return nil
	}
	if oldReactionType > 0 {
		err := c.changeReactionCounter(ctx, targetType, targetID, oldReactionType, -1)
		if err != nil {
			return err
		}
	}
	if newReactionType > 0 {
		err := c.changeReactionCounter(ctx, targetType, targetID, newReactionType, 1)
		if err != nil {
			return err
		}
	}
	return nil
}

// changeReactionCounter 原子调整互动目标的缓存数量。
func (c *CommentReactionCase) changeReactionCounter(ctx context.Context, targetType int32, targetID int64, reactionType int32, delta int32) error {
	// 讨论仅支持点赞，点踩缓存不存在时直接忽略异常历史状态。
	if targetType == _const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION && reactionType != _const.COMMENT_REACTION_TYPE_LIKE {
		return nil
	}

	switch targetType {
	case _const.COMMENT_REACTION_TARGET_TYPE_COMMENT:
		query := c.commentInfoRepo.Query(ctx).CommentInfo
		update := query.LikeCount.Add(delta)
		conditions := []gen.Condition{query.ID.Eq(targetID)}
		// 按互动类型选择评价点赞或点踩缓存字段。
		if reactionType == _const.COMMENT_REACTION_TYPE_DISLIKE {
			update = query.DislikeCount.Add(delta)
			if delta < 0 {
				conditions = append(conditions, query.DislikeCount.Gt(0))
			}
		} else if delta < 0 {
			conditions = append(conditions, query.LikeCount.Gt(0))
		}
		_, err := query.WithContext(ctx).Where(conditions...).UpdateSimple(update)
		return err
	case _const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION:
		query := c.commentDiscussionRepo.Query(ctx).CommentDiscussion
		conditions := []gen.Condition{query.ID.Eq(targetID)}
		if delta < 0 {
			conditions = append(conditions, query.LikeCount.Gt(0))
		}
		_, err := query.WithContext(ctx).Where(conditions...).UpdateSimple(query.LikeCount.Add(delta))
		return err
	case _const.COMMENT_REACTION_TARGET_TYPE_AI:
		query := c.commentAiRepo.Query(ctx).CommentAi
		update := query.LikeCount.Add(delta)
		conditions := []gen.Condition{query.ID.Eq(targetID)}
		// 按互动类型选择 AI 摘要点赞或点踩缓存字段。
		if reactionType == _const.COMMENT_REACTION_TYPE_DISLIKE {
			update = query.DislikeCount.Add(delta)
			if delta < 0 {
				conditions = append(conditions, query.DislikeCount.Gt(0))
			}
		} else if delta < 0 {
			conditions = append(conditions, query.LikeCount.Gt(0))
		}
		_, err := query.WithContext(ctx).Where(conditions...).UpdateSimple(update)
		return err
	default:
		return errorsx.InvalidArgument("互动目标类型不支持")
	}
}

// getCachedReactionCounts 读取互动目标当前缓存数量。
func (c *CommentReactionCase) getCachedReactionCounts(ctx context.Context, targetType int32, targetID int64) (int32, int32, error) {
	switch targetType {
	case _const.COMMENT_REACTION_TARGET_TYPE_COMMENT:
		query := c.commentInfoRepo.Query(ctx).CommentInfo
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ID.Eq(targetID)))
		record, err := c.commentInfoRepo.Find(ctx, opts...)
		if err != nil {
			return 0, 0, err
		}
		return record.LikeCount, record.DislikeCount, nil
	case _const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION:
		query := c.commentDiscussionRepo.Query(ctx).CommentDiscussion
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ID.Eq(targetID)))
		record, err := c.commentDiscussionRepo.Find(ctx, opts...)
		if err != nil {
			return 0, 0, err
		}
		return record.LikeCount, 0, nil
	case _const.COMMENT_REACTION_TARGET_TYPE_AI:
		query := c.commentAiRepo.Query(ctx).CommentAi
		opts := make([]repository.QueryOption, 0, 1)
		opts = append(opts, repository.Where(query.ID.Eq(targetID)))
		record, err := c.commentAiRepo.Find(ctx, opts...)
		if err != nil {
			return 0, 0, err
		}
		return record.LikeCount, record.DislikeCount, nil
	default:
		return 0, 0, errorsx.InvalidArgument("互动目标类型不支持")
	}
}
