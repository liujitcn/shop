package biz

import (
	"context"
	"strings"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/queue"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
)

// CommentInfoCase 评论管理业务实例。
type CommentInfoCase struct {
	*biz.BaseCase
	*data.CommentInfoRepository
	tx                    data.Transaction
	commentTagCase        *CommentTagCase
	commentDiscussionCase *CommentDiscussionCase
	commentAiCase         *CommentAiCase
	commentReviewCase     *CommentReviewCase
	baseUserCase          *BaseUserCase
	commentInfoMapper     *mapper.CopierMapper[adminv1.CommentInfo, models.CommentInfo]
}

// NewCommentInfoCase 创建评论管理业务实例。
func NewCommentInfoCase(
	baseCase *biz.BaseCase,
	commentInfoRepo *data.CommentInfoRepository,
	tx data.Transaction,
	commentTagCase *CommentTagCase,
	commentDiscussionCase *CommentDiscussionCase,
	commentAiCase *CommentAiCase,
	commentReviewCase *CommentReviewCase,
	baseUserCase *BaseUserCase,
) *CommentInfoCase {
	commentInfoMapper := mapper.NewCopierMapper[adminv1.CommentInfo, models.CommentInfo]()
	commentInfoMapper.AppendConverters(mapper.NewJSONTypeConverter[[]int64]().NewConverterPair())
	return &CommentInfoCase{
		BaseCase:              baseCase,
		CommentInfoRepository: commentInfoRepo,
		tx:                    tx,
		commentTagCase:        commentTagCase,
		commentDiscussionCase: commentDiscussionCase,
		commentAiCase:         commentAiCase,
		commentReviewCase:     commentReviewCase,
		baseUserCase:          baseUserCase,
		commentInfoMapper:     commentInfoMapper,
	}
}

// PageCommentInfos 分页查询评论审核列表。
func (c *CommentInfoCase) PageCommentInfos(ctx context.Context, req *adminv1.PageCommentInfosRequest) (*adminv1.PageCommentInfosResponse, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GoodsId != nil && req.GetGoodsId() > 0 {
		opts = append(opts, repository.Where(query.GoodsID.Eq(req.GetGoodsId())))
	}
	// 传入商品名关键字时，按商品名称快照模糊匹配。
	if strings.TrimSpace(req.GetGoodsName()) != "" {
		opts = append(opts, repository.Where(query.GoodsNameSnapshot.Like("%"+strings.TrimSpace(req.GetGoodsName())+"%")))
	}
	// 传入用户昵称关键字时，按用户昵称快照模糊匹配。
	if strings.TrimSpace(req.GetUserName()) != "" {
		opts = append(opts, repository.Where(query.UserNameSnapshot.Like("%"+strings.TrimSpace(req.GetUserName())+"%")))
	}
	if req.GoodsScore != nil && req.GetGoodsScore() > 0 {
		opts = append(opts, repository.Where(query.GoodsScore.Eq(req.GetGoodsScore())))
	}
	if req.MinGoodsScore != nil && req.GetMinGoodsScore() > 0 {
		opts = append(opts, repository.Where(query.GoodsScore.Gte(req.GetMinGoodsScore())))
	}
	if req.MaxGoodsScore != nil && req.GetMaxGoodsScore() > 0 {
		opts = append(opts, repository.Where(query.GoodsScore.Lte(req.GetMaxGoodsScore())))
	}
	if req.HasPendingDiscussion != nil && req.GetHasPendingDiscussion() {
		opts = append(opts, repository.Where(query.PendingDiscussionCount.Gt(0)))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.CommentInfo, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.commentInfoMapper.ToDTO(item))
	}
	return &adminv1.PageCommentInfosResponse{CommentInfos: resList, Total: int32(total)}, nil
}

// GetGoodsCommentInfo 按商品查询评论聚合信息。
func (c *CommentInfoCase) GetGoodsCommentInfo(ctx context.Context, goodsID int64) (*adminv1.GoodsCommentInfoResponse, error) {
	// 商品编号为空时，无法定位评论聚合范围。
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品ID不能为空")
	}

	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	commentList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	commentIDs := make([]int64, 0, len(commentList))
	resCommentList := make([]*adminv1.CommentInfo, 0, len(commentList))
	for _, item := range commentList {
		commentIDs = append(commentIDs, item.ID)
		resCommentList = append(resCommentList, c.commentInfoMapper.ToDTO(item))
	}

	var tagList []*adminv1.CommentTag
	tagList, err = c.commentTagCase.ListByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}

	var discussionList []*adminv1.CommentDiscussion
	discussionList, err = c.commentDiscussionCase.ListByCommentIDs(ctx, commentIDs)
	if err != nil {
		return nil, err
	}

	var aiList []*adminv1.CommentAi
	aiList, err = c.commentAiCase.ListByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}

	return &adminv1.GoodsCommentInfoResponse{
		CommentInfos:       resCommentList,
		CommentTags:        tagList,
		CommentDiscussions: discussionList,
		CommentAis:         aiList,
	}, nil
}

// GetCommentInfo 查询评论详情。
func (c *CommentInfoCase) GetCommentInfo(ctx context.Context, commentID int64) (*adminv1.CommentInfoDetail, error) {
	commentInfo, err := c.FindByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	var tagList []*adminv1.CommentTag
	tagList, err = c.commentTagCase.ListByGoodsID(ctx, commentInfo.GoodsID)
	if err != nil {
		return nil, err
	}

	var discussionList []*adminv1.CommentDiscussion
	discussionList, err = c.commentDiscussionCase.ListByCommentIDs(ctx, []int64{commentInfo.ID})
	if err != nil {
		return nil, err
	}

	var aiList []*adminv1.CommentAi
	aiList, err = c.commentAiCase.ListByGoodsID(ctx, commentInfo.GoodsID)
	if err != nil {
		return nil, err
	}

	var reviewList []*adminv1.CommentReview
	reviewList, err = c.commentReviewCase.ListByTarget(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, commentInfo.ID)
	if err != nil {
		return nil, err
	}

	return &adminv1.CommentInfoDetail{
		Comment:            c.commentInfoMapper.ToDTO(commentInfo),
		CommentTags:        tagList,
		CommentDiscussions: discussionList,
		CommentAis:         aiList,
		CommentReviews:     reviewList,
	}, nil
}

// SetCommentInfoStatus 设置评论审核状态。
func (c *CommentInfoCase) SetCommentInfoStatus(ctx context.Context, req *adminv1.SetCommentInfoStatusRequest) error {
	err := validateCommentStatus(int32(req.GetStatus()))
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	commentInfo, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	operatorName := c.operatorName(ctx, authInfo.UserId)
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		// 评论从通过改为不通过时，释放已计入筛选项的标签提及次数并清空命中标签。
		if commentInfo.Status == _const.COMMENT_STATUS_APPROVED && int32(req.GetStatus()) != _const.COMMENT_STATUS_APPROVED {
			tagIDs := _string.ConvertJsonStringToInt64Array(commentInfo.TagID)
			updateErr := c.commentTagCase.DecreaseMentionCount(txCtx, tagIDs)
			if updateErr != nil {
				return updateErr
			}
			updateErr = c.updateTagIDs(txCtx, req.GetId(), []int64{})
			if updateErr != nil {
				return updateErr
			}
		}
		updateErr := c.updateStatus(txCtx, req.GetId(), int32(req.GetStatus()))
		if updateErr != nil {
			return updateErr
		}
		return c.commentReviewCase.CreateReview(txCtx, &models.CommentReview{
			TargetType:   _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT,
			TargetID:     req.GetId(),
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
	// 评论进入或离开通过态后异步刷新商品评价摘要，不阻塞后台审核操作。
	if int32(req.GetStatus()) == _const.COMMENT_STATUS_APPROVED || commentInfo.Status == _const.COMMENT_STATUS_APPROVED {
		queue.DispatchCommentAiRefresh(commentInfo.GoodsID)
	}
	return nil
}

// updateTagIDs 更新评论命中的标签编号。
func (c *CommentInfoCase) updateTagIDs(ctx context.Context, commentID int64, tagIDs []int64) error {
	query := c.Query(ctx).CommentInfo
	result, err := query.WithContext(ctx).
		Where(query.ID.Eq(commentID)).
		Update(query.TagID, _string.ConvertAnyToJsonString(tagIDs))
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errorsx.ResourceNotFound("评论不存在")
	}
	return nil
}

// updateStatus 更新评论审核状态。
func (c *CommentInfoCase) updateStatus(ctx context.Context, commentID int64, status int32) error {
	query := c.Query(ctx).CommentInfo
	result, err := query.WithContext(ctx).
		Where(query.ID.Eq(commentID)).
		Update(query.Status, status)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errorsx.ResourceNotFound("评论不存在")
	}
	return nil
}

// changeDiscussionCount 调整评论讨论缓存数量。
func (c *CommentInfoCase) changeDiscussionCount(ctx context.Context, commentID int64, status int32, delta int32) error {
	if delta == 0 {
		return nil
	}
	query := c.Query(ctx).CommentInfo
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
func (c *CommentInfoCase) operatorName(ctx context.Context, userID int64) string {
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
