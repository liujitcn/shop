package biz

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"

	_const "shop/service/shop/consts"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	corequeue "shop/pkg/queue"
	"shop/service/shop/app/agent/comment"
	appDto "shop/service/shop/app/dto"
	"shop/service/shop/queue"
	"shop/service/shop/workspaceevent"
	systemappbiz "shop/service/system/app/biz"

	"github.com/go-kratos/kratos/v3/log"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

const (
	// commentSummarySourceLimit 表示单次评价摘要刷新最多读取的评价数量。
	commentSummarySourceLimit = 50
	// commentConsumerTimeout 表示单条评价异步任务允许占用的最长时间。
	commentConsumerTimeout = 60 * time.Second
)

// CommentCase 评价业务编排对象。
type CommentCase struct {
	*biz.BaseCase
	tx                    data.Transaction
	commentInfoCase       *CommentInfoCase
	commentSummaryCase    *CommentSummaryCase
	commentTagCase        *CommentTagCase
	commentReviewCase     *CommentReviewCase
	commentDiscussionCase *CommentDiscussionCase
	commentReactionCase   *CommentReactionCase
	orderInfoCase         *OrderInfoCase
	orderGoodsCase        *OrderGoodsCase
	baseUserCase          *systemappbiz.BaseUserCase
	commentRuntime        *comment.Runtime
	appInfo               *bootstrapConfigv1.AppInfo
}

// NewCommentCase 创建评价业务编排对象。
func NewCommentCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	commentInfoCase *CommentInfoCase,
	commentSummaryCase *CommentSummaryCase,
	commentTagCase *CommentTagCase,
	commentReviewCase *CommentReviewCase,
	commentDiscussionCase *CommentDiscussionCase,
	commentReactionCase *CommentReactionCase,
	orderInfoCase *OrderInfoCase,
	orderGoodsCase *OrderGoodsCase,
	baseUserCase *systemappbiz.BaseUserCase,
	commentRuntime *comment.Runtime,
	appInfo *bootstrapConfigv1.AppInfo,
) *CommentCase {
	c := &CommentCase{
		BaseCase:              baseCase,
		tx:                    tx,
		commentInfoCase:       commentInfoCase,
		commentSummaryCase:    commentSummaryCase,
		commentTagCase:        commentTagCase,
		commentReviewCase:     commentReviewCase,
		commentDiscussionCase: commentDiscussionCase,
		commentReactionCase:   commentReactionCase,
		orderInfoCase:         orderInfoCase,
		orderGoodsCase:        orderGoodsCase,
		baseUserCase:          baseUserCase,
		commentRuntime:        commentRuntime,
		appInfo:               appInfo,
	}
	// 注册评价审核与 评价摘要刷新异步消费者，避免提交评价时阻塞用户主流程。
	c.RegisterQueueConsumer(_const.COMMENT_AUDIT, c.consumeCommentAudit)
	c.RegisterQueueConsumer(_const.COMMENT_SUMMARY_REFRESH, c.consumeCommentSummaryRefresh)
	return c
}

// PageCommentDiscussion 查询评价讨论分页列表。
func (c *CommentCase) PageCommentDiscussion(ctx context.Context, req *shopappv1.PageCommentDiscussionRequest) (*shopappv1.PageCommentDiscussionResponse, error) {
	_, err := c.commentInfoCase.FindByID(ctx, req.GetCommentId())
	if err != nil {
		return nil, err
	}

	userID := int64(0)
	authInfo, authErr := c.GetAuthInfo(ctx)
	// 当前请求带有登录信息时，补齐当前用户编号用于点赞状态回显。
	if authErr == nil && authInfo != nil && authInfo.UserId > 0 {
		userID = authInfo.UserId
	}

	var list []*shopappv1.CommentDiscussionItem
	total := int32(0)
	list, total, err = c.commentDiscussionCase.PageCommentDiscussion(ctx, req.GetCommentId(), userID, req)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	return &shopappv1.PageCommentDiscussionResponse{
		CommentId:          req.GetCommentId(),
		CommentDiscussions: list,
		Total:              total,
		PageNum:            pageNum,
		PageSize:           pageSize,
		HasMore:            pageNum*pageSize < int64(total),
	}, nil
}

// PageGoodsComment 查询商品评价分页列表。
func (c *CommentCase) PageGoodsComment(ctx context.Context, req *shopappv1.PageGoodsCommentRequest) (*shopappv1.PageGoodsCommentResponse, error) {
	userID := int64(0)
	authInfo, authErr := c.GetAuthInfo(ctx)
	// 当前请求带有登录信息时，补齐当前用户编号用于互动状态回显。
	if authErr == nil && authInfo != nil && authInfo.UserId > 0 {
		userID = authInfo.UserId
	}

	summary, err := c.commentInfoCase.BuildOverviewSummary(ctx, req.GetGoodsId())
	if err != nil {
		return nil, err
	}
	var filterStats *appDto.CommentFilterStats
	filterStats, err = c.commentInfoCase.BuildFilterStats(ctx, req.GetGoodsId())
	if err != nil {
		return nil, err
	}
	var commentSummary *shopappv1.CommentSummary
	commentSummary, err = c.commentSummaryCase.buildCardByGoodsIDAndScene(ctx, req.GetGoodsId(), _const.COMMENT_SUMMARY_SCENE_LIST, userID)
	if err != nil {
		return nil, err
	}
	var list []*shopappv1.CommentItem
	total := int32(0)
	list, total, err = c.commentInfoCase.PageGoodsComment(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	filterList := make([]*shopappv1.CommentFilterItem, 0, 5)
	filterList = append(filterList, &shopappv1.CommentFilterItem{
		FilterType: shopcommonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_ALL),
		TagId:      0,
		Label:      "全部",
		Value:      strconv.FormatInt(int64(summary.RecentGoodRate), 10) + "%好评",
	})
	filterList = append(filterList, &shopappv1.CommentFilterItem{
		FilterType: shopcommonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_MEDIA),
		TagId:      0,
		Label:      "有图",
		Value:      strconv.FormatInt(int64(filterStats.MediaCount), 10),
	})
	filterList = append(filterList, &shopappv1.CommentFilterItem{
		FilterType: shopcommonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_GOOD),
		TagId:      0,
		Label:      "好评",
		Value:      strconv.FormatInt(int64(filterStats.GoodCount), 10),
	})
	filterList = append(filterList, &shopappv1.CommentFilterItem{
		FilterType: shopcommonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_MIDDLE),
		TagId:      0,
		Label:      "中评",
		Value:      strconv.FormatInt(int64(filterStats.MiddleCount), 10),
	})
	filterList = append(filterList, &shopappv1.CommentFilterItem{
		FilterType: shopcommonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_BAD),
		TagId:      0,
		Label:      "差评",
		Value:      strconv.FormatInt(int64(filterStats.BadCount), 10),
	})

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	return &shopappv1.PageGoodsCommentResponse{
		CommentFilters: filterList,
		CommentSummary: commentSummary,
		Comments:       list,
		Total:          total,
		PageNum:        pageNum,
		PageSize:       pageSize,
		HasMore:        pageNum*pageSize < int64(total),
	}, nil
}

// PageMyComment 查询我的评价分页列表。
func (c *CommentCase) PageMyComment(ctx context.Context, req *shopappv1.PageMyCommentRequest) (*shopappv1.PageMyCommentResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var list []*shopappv1.CommentItem
	total := int32(0)
	list, total, err = c.commentInfoCase.PageMyComment(ctx, authInfo.UserId, req)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	return &shopappv1.PageMyCommentResponse{
		Comments: list,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		HasMore:  pageNum*pageSize < int64(total),
	}, nil
}

// PagePendingCommentGoods 查询待评价商品分页列表。
func (c *CommentCase) PagePendingCommentGoods(ctx context.Context, req *shopappv1.PagePendingCommentGoodsRequest) (*shopappv1.PagePendingCommentGoodsResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())
	query := c.orderGoodsCase.Query(ctx)
	orderGoodsQuery := query.OrderGoods
	orderInfoQuery := query.OrderInfo
	commentInfoQuery := query.CommentInfo
	dao := orderGoodsQuery.WithContext(ctx).
		Select(orderGoodsQuery.ALL).
		Join(orderInfoQuery, orderGoodsQuery.OrderID.EqCol(orderInfoQuery.ID)).
		LeftJoin(
			commentInfoQuery,
			orderGoodsQuery.OrderID.EqCol(commentInfoQuery.OrderID),
			orderGoodsQuery.GoodsID.EqCol(commentInfoQuery.GoodsID),
			orderGoodsQuery.SKUCode.EqCol(commentInfoQuery.SKUCode),
			commentInfoQuery.UserID.Eq(authInfo.UserId),
		).
		Where(
			orderInfoQuery.UserID.Eq(authInfo.UserId),
			orderInfoQuery.Status.Eq(_const.ORDER_INFO_STATUS_WAIT_REVIEW),
			orderInfoQuery.DeletedAt.Eq(sql.NullInt64{Valid: true}),
			orderGoodsQuery.DeletedAt.Eq(sql.NullInt64{Valid: true}),
			commentInfoQuery.ID.IsNull(),
		).
		Order(orderInfoQuery.CreatedAt.Desc(), orderGoodsQuery.ID.Desc())
	if req.GetOrderId() > 0 {
		dao = dao.Where(orderInfoQuery.ID.Eq(req.GetOrderId()))
	}
	var orderGoodsList []*models.OrderGoods
	var total int64
	orderGoodsList, total, err = dao.FindByPage(int((pageNum-1)*pageSize), int(pageSize))
	if err != nil {
		return nil, err
	}

	pendingList := make([]*shopappv1.PendingCommentGoodsItem, 0, len(orderGoodsList))
	for _, orderGoods := range orderGoodsList {
		pendingList = append(pendingList, &shopappv1.PendingCommentGoodsItem{
			OrderId:      orderGoods.OrderID,
			GoodsId:      orderGoods.GoodsID,
			GoodsName:    orderGoods.Name,
			GoodsPicture: orderGoods.Picture,
			SkuCode:      orderGoods.SKUCode,
			SkuDesc:      strings.Join(_string.ConvertJsonStringToStringArray(orderGoods.SpecItem), " / "),
			Desc:         "分享你的使用体验，帮助其他买家更好选择",
		})
	}

	return &shopappv1.PagePendingCommentGoodsResponse{
		PendingCommentGoods: pendingList,
		Total:               int32(total),
		PageNum:             pageNum,
		PageSize:            pageSize,
		HasMore:             pageNum*pageSize < total,
	}, nil
}

// CreateComment 发布商品评价。
func (c *CommentCase) CreateComment(ctx context.Context, req *shopappv1.CreateCommentRequest) (*shopappv1.CreateCommentResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var orderInfo *models.OrderInfo
	orderInfo, err = c.orderInfoCase.findByUserIDAndID(ctx, authInfo.UserId, req.GetOrderId())
	if err != nil {
		return nil, errorsx.ResourceNotFound("订单不存在").WithCause(err)
	}
	// 当前订单不处于待评价状态时，不允许继续创建评价。
	if orderInfo.Status != _const.ORDER_INFO_STATUS_WAIT_REVIEW {
		return nil, errorsx.InvalidArgument("当前订单不可评价")
	}

	query := c.orderGoodsCase.Query(ctx).OrderGoods
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.OrderID.Eq(req.GetOrderId())))
	opts = append(opts, repository.Where(query.GoodsID.Eq(req.GetGoodsId())))
	opts = append(opts, repository.Where(query.SKUCode.Eq(req.GetSkuCode())))
	var orderGoods *models.OrderGoods
	orderGoods, err = c.orderGoodsCase.Find(ctx, opts...)
	if err != nil {
		return nil, errorsx.ResourceNotFound("订单商品不存在").WithCause(err)
	}
	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}

	opts = make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.OrderID.Eq(req.GetOrderId())))
	var allOrderGoodsList []*models.OrderGoods
	allOrderGoodsList, err = c.orderGoodsCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var record *models.CommentInfo
	orderCompleted := false
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		record, err = c.commentInfoCase.CreateComment(txCtx, orderInfo.TenantID, orderInfo.TenantStoreID, user, req, orderGoods)
		if err != nil {
			return err
		}

		orderCompleted, err = c.commentInfoCase.AreAllOrderGoodsCommented(txCtx, authInfo.UserId, allOrderGoodsList)
		if err != nil {
			return err
		}
		// 当前订单下全部商品都已评价时，将订单状态流转到已完成。
		if orderCompleted {
			return c.orderInfoCase.updateByIDs(txCtx, authInfo.UserId, []int64{req.GetOrderId()}, &models.OrderInfo{
				Status: _const.ORDER_INFO_STATUS_COMPLETED,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	queue.DispatchCommentAudit(_const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID)
	workspaceevent.Publish(
		ctx,
		workspaceevent.ReasonCommentChanged,
		workspaceevent.AreaMetrics,
		workspaceevent.AreaTodo,
		workspaceevent.AreaRisk,
		workspaceevent.AreaPendingComments,
	)

	return &shopappv1.CreateCommentResponse{
		CommentId:      record.ID,
		OrderId:        req.GetOrderId(),
		OrderCompleted: orderCompleted,
	}, nil
}

// CreateCommentDiscussion 发布评价讨论。
func (c *CommentCase) CreateCommentDiscussion(ctx context.Context, req *shopappv1.CreateCommentDiscussionRequest) (*shopappv1.CreateCommentDiscussionResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var commentInfo *models.CommentInfo
	commentInfo, err = c.commentInfoCase.FindByID(ctx, req.GetCommentId())
	if err != nil {
		return nil, err
	}

	var user *models.BaseUser
	user, err = c.baseUserCase.FindByID(ctx, authInfo.UserId)
	if err != nil {
		return nil, errorsx.ResourceNotFound("用户不存在").WithCause(err)
	}

	var record *models.CommentDiscussion
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		record, err = c.commentDiscussionCase.CreateDiscussion(txCtx, commentInfo, user, req)
		if err != nil {
			return err
		}
		return c.commentInfoCase.changeDiscussionCount(txCtx, req.GetCommentId(), _const.COMMENT_STATUS_PENDING_REVIEW, 1)
	})
	if err != nil {
		return nil, err
	}
	queue.DispatchCommentAudit(_const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID)
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo)

	response := &shopappv1.CreateCommentDiscussionResponse{
		DiscussionCount: commentInfo.DiscussionCount,
	}
	// 讨论默认待审核，未审核通过前不返回到公开讨论列表。
	if record.Status == _const.COMMENT_STATUS_APPROVED {
		response.DiscussionCount = commentInfo.DiscussionCount + 1
		response.Item = c.commentDiscussionCase.buildDiscussionItem(record, map[int64]int32{})
	}
	return response, nil
}

// DeleteComment 删除商品评价。
func (c *CommentCase) DeleteComment(ctx context.Context, commentID int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}

	var record *models.CommentInfo
	record, err = c.commentInfoCase.FindOwnerByID(ctx, commentID, authInfo.UserId)
	if err != nil {
		return err
	}

	tagIDs := _string.ConvertJsonStringToInt64Array(record.TagID)
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.commentInfoCase.DeleteOwnerComment(txCtx, commentID, authInfo.UserId)
		if err != nil {
			return err
		}
		return c.commentTagCase.DecreaseMentionCount(txCtx, tagIDs)
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(
		ctx,
		workspaceevent.ReasonCommentChanged,
		workspaceevent.AreaMetrics,
		workspaceevent.AreaTodo,
		workspaceevent.AreaRisk,
		workspaceevent.AreaReputation,
		workspaceevent.AreaPendingComments,
	)
	return nil
}

// GoodsCommentOverview 查询商品评价摘要。
func (c *CommentCase) GoodsCommentOverview(ctx context.Context, req *shopappv1.GoodsCommentOverviewRequest) (*shopappv1.GoodsCommentOverviewResponse, error) {
	userID := int64(0)
	authInfo, authErr := c.GetAuthInfo(ctx)
	// 当前请求带有登录信息时，补齐当前用户编号用于互动状态回显。
	if authErr == nil && authInfo != nil && authInfo.UserId > 0 {
		userID = authInfo.UserId
	}

	recordList, err := c.commentInfoCase.listByGoodsID(ctx, req.GetGoodsId())
	if err != nil {
		return nil, err
	}
	summary := c.commentInfoCase.buildOverviewSummary(recordList)
	// 当前商品没有审核通过评价时，直接返回空摘要，避免继续查询 AI、标签和预览列表。
	if summary.TotalCount == 0 {
		return &shopappv1.GoodsCommentOverviewResponse{
			TotalCount:     summary.TotalCount,
			RecentDays:     90,
			RecentGoodRate: summary.RecentGoodRate,
			CommentSummary: &shopappv1.CommentSummary{},
		}, nil
	}

	var commentSummary *shopappv1.CommentSummary
	commentSummary, err = c.commentSummaryCase.buildCardByGoodsIDAndScene(ctx, req.GetGoodsId(), _const.COMMENT_SUMMARY_SCENE_OVERVIEW, userID)
	if err != nil {
		return nil, err
	}
	var previewList []*shopappv1.CommentItem
	previewList, err = c.commentInfoCase.listPreviewByRecordList(ctx, recordList, req.GetPreviewLimit(), userID)
	if err != nil {
		return nil, err
	}

	return &shopappv1.GoodsCommentOverviewResponse{
		TotalCount:      summary.TotalCount,
		RecentDays:      90,
		RecentGoodRate:  summary.RecentGoodRate,
		CommentSummary:  commentSummary,
		PreviewComments: previewList,
	}, nil
}

// GoodsCommentTag 查询商品评价标签列表。
func (c *CommentCase) GoodsCommentTag(ctx context.Context, req *shopappv1.GoodsCommentTagRequest) (*shopappv1.GoodsCommentTagResponse, error) {
	commentTags, err := c.commentTagCase.ListTags(ctx, req.GetGoodsId(), req.GetLimit())
	if err != nil {
		return nil, err
	}
	return &shopappv1.GoodsCommentTagResponse{
		CommentTags: commentTags,
	}, nil
}

// SaveCommentReaction 保存评价互动状态。
func (c *CommentCase) SaveCommentReaction(ctx context.Context, req *shopappv1.SaveCommentReactionRequest) (*shopappv1.SaveCommentReactionResponse, error) {
	// 互动类型非法时，无法继续保存互动状态。
	if req.GetReactionType() != shopcommonv1.CommentReactionType(_const.COMMENT_REACTION_TYPE_LIKE) &&
		req.GetReactionType() != shopcommonv1.CommentReactionType(_const.COMMENT_REACTION_TYPE_DISLIKE) {
		return nil, errorsx.InvalidArgument("互动类型不支持")
	}

	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	// 不同互动目标使用各自的存在性校验和行为限制。
	switch req.GetTargetType() {
	case shopcommonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_SUMMARY):
		_, err = c.commentSummaryCase.FindByID(ctx, req.GetTargetId())
		if err != nil {
			return nil, err
		}
	case shopcommonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION):
		// 讨论互动当前只支持点赞，不支持点踩。
		if req.GetReactionType() != shopcommonv1.CommentReactionType(_const.COMMENT_REACTION_TYPE_LIKE) {
			return nil, errorsx.InvalidArgument("讨论仅支持点赞")
		}
		_, err = c.commentDiscussionCase.FindByID(ctx, req.GetTargetId())
		if err != nil {
			return nil, err
		}
	case shopcommonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_COMMENT):
		// 评价互动支持点赞和点踩，但只允许对审核通过的评价操作。
		_, err = c.commentInfoCase.FindByID(ctx, req.GetTargetId())
		if err != nil {
			return nil, err
		}
	default:
		return nil, errorsx.InvalidArgument("互动目标类型不支持")
	}

	var response *shopappv1.SaveCommentReactionResponse
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		response, err = c.commentReactionCase.SaveCommentReaction(txCtx, authInfo.UserId, req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// AuditComment 根据评价对象执行 AI 审核流程。
func (c *CommentCase) AuditComment(ctx context.Context, record *models.CommentInfo) error {
	// 仅待审核评价进入 AI 审核，避免人工已处理后被异步消息覆盖。
	if record.Status != _const.COMMENT_STATUS_PENDING_REVIEW {
		return nil
	}
	if !c.commentRuntime.Enabled() {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM客户端未配置", c.commentRuntime.Model())
	}

	var err error
	var imageURLs []string
	var imageData []comment.ReviewImageData
	imageURLs, imageData, err = c.buildCommentReviewImages(record.Img)
	if err != nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error(), c.commentRuntime.Model())
	}

	var existingTags []string
	existingTags, err = c.commentTagCase.ExistingTagNames(ctx, record.GoodsID)
	if err != nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error(), c.commentRuntime.Model())
	}

	var result *comment.ReviewResult
	result, err = c.commentRuntime.ReviewComment(ctx, comment.ReviewRequest{
		GoodsName:    record.GoodsNameSnapshot,
		SKUDesc:      record.SKUDescSnapshot,
		Content:      record.Content,
		ExistingTags: existingTags,
		ImageURLs:    imageURLs,
		ImageData:    imageData,
	})
	if err != nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error(), c.commentRuntime.Model())
	}

	if result == nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM审核结果为空", c.commentRuntime.Model())
	}
	if !result.Approved {
		if !comment.HasConcreteReviewReason(result.RiskReason) {
			return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, result.Tags, commentReviewMissingReason(result), c.commentRuntime.Model())
		}
		return c.rejectCommentByAI(ctx, record, result)
	}
	return c.approveCommentByAI(ctx, record, result)
}

// AuditDiscussion 根据讨论对象执行 AI 审核流程。
func (c *CommentCase) AuditDiscussion(ctx context.Context, record *models.CommentDiscussion) error {
	// 仅待审核讨论进入 AI 审核，避免人工已处理后被异步消息覆盖。
	if record.Status != _const.COMMENT_STATUS_PENDING_REVIEW {
		return nil
	}
	var commentInfo *models.CommentInfo
	var err error
	commentInfo, err = c.commentInfoCase.FindAnyByID(ctx, record.CommentID)
	if err != nil {
		return err
	}
	if !c.commentRuntime.Enabled() {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM客户端未配置", c.commentRuntime.Model())
	}

	var existingTags []string
	existingTags, err = c.commentTagCase.ExistingTagNames(ctx, commentInfo.GoodsID)
	if err != nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error(), c.commentRuntime.Model())
	}

	var result *comment.ReviewResult
	result, err = c.commentRuntime.ReviewComment(ctx, comment.ReviewRequest{
		GoodsName:    commentInfo.GoodsNameSnapshot,
		SKUDesc:      commentInfo.SKUDescSnapshot,
		Content:      record.Content,
		ExistingTags: existingTags,
	})
	if err != nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error(), c.commentRuntime.Model())
	}
	if result == nil {
		return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM审核结果为空", c.commentRuntime.Model())
	}
	if !result.Approved {
		if !comment.HasConcreteReviewReason(result.RiskReason) {
			return c.commentReviewCase.createAIReview(ctx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, result.Tags, commentReviewMissingReason(result), c.commentRuntime.Model())
		}
		return c.rejectDiscussionByAI(ctx, record, result)
	}
	return c.approveDiscussionByAI(ctx, record, result)
}

// consumeCommentAudit 消费评价与讨论审核队列。
func (c *CommentCase) consumeCommentAudit(message queueData.Message) error {
	event, err := corequeue.DecodeQueueData[queue.CommentAuditEvent](message)
	if err != nil {
		return err
	}
	// 队列消息缺失目标时直接忽略，避免无效消息反复重试。
	if event == nil || event.TargetID <= 0 {
		return nil
	}

	ctx, cancel := newCommentConsumerContext()
	defer cancel()
	switch event.TargetType {
	case _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT:
		return c.auditComment(ctx, event.TargetID)
	case _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION:
		return c.auditDiscussion(ctx, event.TargetID)
	default:
		return nil
	}
}

// auditComment 执行单条评价的 AI 审核流程。
func (c *CommentCase) auditComment(ctx context.Context, commentID int64) error {
	record, err := c.commentInfoCase.FindAnyByID(ctx, commentID)
	if err != nil {
		return err
	}
	return c.AuditComment(ctx, record)
}

// buildCommentReviewImages 构建评价审核使用的图片输入。
func (c *CommentCase) buildCommentReviewImages(rawImages string) ([]string, []comment.ReviewImageData, error) {
	images := _string.ConvertJsonStringToStringArray(rawImages)
	imageURLs := make([]string, 0, len(images))
	imageData := make([]comment.ReviewImageData, 0, len(images))
	for _, image := range images {
		// 图片地址为空时跳过无效项。
		if image == "" {
			continue
		}
		// 绝对地址和 data URL 可直接交给模型服务识别。
		if c.isLLMImageURL(image) {
			imageURLs = append(imageURLs, image)
			continue
		}
		bytes, err := c.readCommentReviewImage(commentReviewImagePath(image))
		if err != nil {
			return nil, nil, err
		}
		imageData = append(imageData, comment.ReviewImageData{
			Name:     path.Base(image),
			Bytes:    bytes,
			MIMEType: commentReviewImageMIMEType(image),
		})
	}
	return imageURLs, imageData, nil
}

// readCommentReviewImage 读取评价审核本地或对象存储图片内容。
func (c *CommentCase) readCommentReviewImage(imagePath string) ([]byte, error) {
	oss := sdk.Runtime.GetOSS()
	if oss == nil {
		return nil, errorsx.Internal("读取评价图片失败").WithCause(fmt.Errorf("OSS未初始化: %s", imagePath))
	}
	bytes, err := oss.GetFileByte(imagePath)
	if err != nil {
		return nil, errorsx.Internal("读取评价图片失败").WithCause(fmt.Errorf("%s: %w", imagePath, err))
	}
	if len(bytes) == 0 {
		return nil, errorsx.Internal("读取评价图片失败").WithCause(fmt.Errorf("评价图片内容为空: %s", imagePath))
	}
	return bytes, nil
}

// approveCommentByAI 将评价审核通过结果写入业务表和审核记录。
func (c *CommentCase) approveCommentByAI(ctx context.Context, record *models.CommentInfo, result *comment.ReviewResult) error {
	cleanTags := cleanCommentTagNames(result.Tags)
	updated := false
	err := c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var updateErr error
		updated, updateErr = c.commentInfoCase.UpdatePendingStatus(txCtx, record.ID, _const.COMMENT_STATUS_APPROVED)
		if updateErr != nil {
			return updateErr
		}
		// 审核执行期间评价已被其他流程处理时，当前 AI 结果不再覆盖业务状态。
		if !updated {
			return nil
		}
		tagIDs, tagNames, upsertErr := c.commentTagCase.UpsertTagsByNames(txCtx, record.TenantID, record.TenantStoreID, record.GoodsID, cleanTags)
		if upsertErr != nil {
			return upsertErr
		}
		updateErr = c.commentInfoCase.UpdateTagIDs(txCtx, record.ID, tagIDs)
		if updateErr != nil {
			return updateErr
		}
		return c.commentReviewCase.createAIReview(txCtx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_APPROVED, tagNames, "", c.commentRuntime.Model())
	})
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	queue.DispatchCommentSummaryRefresh(record.GoodsID)
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaRisk, workspaceevent.AreaReputation, workspaceevent.AreaPendingComments)
	return nil
}

// rejectCommentByAI 将评价审核不通过结果写入业务表和审核记录。
func (c *CommentCase) rejectCommentByAI(ctx context.Context, record *models.CommentInfo, result *comment.ReviewResult) error {
	reason := commentReviewRejectReason(result)
	updated := false
	var err error
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		updated, err = c.commentInfoCase.UpdatePendingStatus(txCtx, record.ID, _const.COMMENT_STATUS_REJECTED)
		if err != nil {
			return err
		}
		// 审核执行期间评价已被其他流程处理时，当前 AI 结果不再覆盖业务状态。
		if !updated {
			return nil
		}
		return c.commentReviewCase.createAIReview(txCtx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_REJECTED, result.Tags, reason, c.commentRuntime.Model())
	})
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaRisk, workspaceevent.AreaPendingComments)
	return nil
}

// auditDiscussion 执行单条讨论的 AI 审核流程。
func (c *CommentCase) auditDiscussion(ctx context.Context, discussionID int64) error {
	record, err := c.commentDiscussionCase.findAnyByID(ctx, discussionID)
	if err != nil {
		return err
	}
	return c.AuditDiscussion(ctx, record)
}

// approveDiscussionByAI 将讨论审核通过结果写入业务表和审核记录。
func (c *CommentCase) approveDiscussionByAI(ctx context.Context, record *models.CommentDiscussion, result *comment.ReviewResult) error {
	updated := false
	var err error
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		updated, err = c.commentDiscussionCase.updatePendingStatus(txCtx, record.ID, _const.COMMENT_STATUS_APPROVED)
		if err != nil {
			return err
		}
		// 审核执行期间讨论已被其他流程处理时，当前 AI 结果不再覆盖业务状态和计数。
		if !updated {
			return nil
		}
		err = c.commentInfoCase.changeDiscussionCount(txCtx, record.CommentID, _const.COMMENT_STATUS_PENDING_REVIEW, -1)
		if err != nil {
			return err
		}
		err = c.commentInfoCase.changeDiscussionCount(txCtx, record.CommentID, _const.COMMENT_STATUS_APPROVED, 1)
		if err != nil {
			return err
		}
		return c.commentReviewCase.createAIReview(txCtx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_APPROVED, result.Tags, "", c.commentRuntime.Model())
	})
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	var commentInfo *models.CommentInfo
	commentInfo, err = c.commentInfoCase.FindAnyByID(ctx, record.CommentID)
	if err == nil {
		queue.DispatchCommentSummaryRefresh(commentInfo.GoodsID)
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaReputation)
	return nil
}

// rejectDiscussionByAI 将讨论审核不通过结果写入业务表和审核记录。
func (c *CommentCase) rejectDiscussionByAI(ctx context.Context, record *models.CommentDiscussion, result *comment.ReviewResult) error {
	reason := commentReviewRejectReason(result)
	updated := false
	var err error
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		updated, err = c.commentDiscussionCase.updatePendingStatus(txCtx, record.ID, _const.COMMENT_STATUS_REJECTED)
		if err != nil {
			return err
		}
		// 审核执行期间讨论已被其他流程处理时，当前 AI 结果不再覆盖业务状态和计数。
		if !updated {
			return nil
		}
		err = c.commentInfoCase.changeDiscussionCount(txCtx, record.CommentID, _const.COMMENT_STATUS_PENDING_REVIEW, -1)
		if err != nil {
			return err
		}
		return c.commentReviewCase.createAIReview(txCtx, record.TenantID, record.TenantStoreID, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_REJECTED, result.Tags, reason, c.commentRuntime.Model())
	})
	if err != nil {
		return err
	}
	if !updated {
		return nil
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo)
	return nil
}

// consumeCommentSummaryRefresh 消费商品评价摘要刷新队列。
func (c *CommentCase) consumeCommentSummaryRefresh(message queueData.Message) error {
	goodsID, err := corequeue.DecodeQueueData[int64](message)
	if err != nil {
		return err
	}
	// 商品编号缺失时直接忽略。
	if goodsID == nil || *goodsID <= 0 {
		return nil
	}
	ctx, cancel := newCommentConsumerContext()
	defer cancel()
	return c.refreshGoodsCommentSummary(ctx, *goodsID)
}

// refreshGoodsCommentSummary 基于审核通过评价刷新商品评价摘要。
func (c *CommentCase) refreshGoodsCommentSummary(ctx context.Context, goodsID int64) error {
	// LLM 未配置时不刷新摘要，前台继续使用旧摘要或空摘要降级。
	if !c.commentRuntime.Enabled() {
		log.Warn(fmt.Sprintf("refreshGoodsCommentSummary skip goodsID=%d: comment runtime disabled", goodsID))
		return nil
	}
	commentList, err := c.commentInfoCase.listSummarySourceByGoodsID(ctx, goodsID, commentSummarySourceLimit)
	if err != nil {
		return err
	}
	// 当前商品暂无通过评价时，不生成空摘要覆盖旧内容。
	if len(commentList) == 0 {
		log.Info(fmt.Sprintf("refreshGoodsCommentSummary skip goodsID=%d: no approved comment", goodsID))
		return nil
	}

	tagNameMap := make(map[int64]string)
	var tagList []*models.CommentTag
	tagList, err = c.commentTagCase.listVisibleByGoodsID(ctx, goodsID)
	if err != nil {
		log.Error(fmt.Sprintf("refreshGoodsCommentSummary load tags goodsID=%d err=%v", goodsID, err))
	} else {
		tagNameMap = make(map[int64]string, len(tagList))
		for _, tag := range tagList {
			// 标签名称为空时无法作为摘要输入。
			if tag.Name == "" {
				continue
			}
			tagNameMap[tag.ID] = tag.Name
		}
	}

	goodsName := ""
	comments := make([]comment.SummaryComment, 0, len(commentList))
	for _, item := range commentList {
		if goodsName == "" {
			goodsName = item.GoodsNameSnapshot
		}
		comments = append(comments, comment.SummaryComment{
			Content:       item.Content,
			GoodsScore:    item.GoodsScore,
			PackageScore:  item.PackageScore,
			DeliveryScore: item.DeliveryScore,
			Tags:          summaryTagNamesByIDs(_string.ConvertJsonStringToInt64Array(item.TagID), tagNameMap),
		})
	}
	var result *comment.SummaryResult
	result, err = c.commentRuntime.GenerateSummary(ctx, comment.SummaryRequest{
		GoodsName: goodsName,
		Comments:  comments,
	})
	if err != nil {
		return err
	}
	// 模型返回空摘要时不落库，避免空结果覆盖旧摘要。
	if result == nil || len(result.Overview.Content) == 0 && len(result.List.Content) == 0 {
		log.Warn(fmt.Sprintf("refreshGoodsCommentSummary skip goodsID=%d: summary result empty, sourceCount=%d", goodsID, len(commentList)))
		return nil
	}
	err = c.commentSummaryCase.UpsertGoodsCommentSummary(ctx, commentList[0].TenantID, commentList[0].TenantStoreID, goodsID, result)
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaReputation)
	return nil
}

// isLLMImageURL 判断图片地址是否可直接作为多模态 URL 输入。
func (c *CommentCase) isLLMImageURL(imageURL string) bool {
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return false
	}
	// HTTP(S) 绝对地址和 data URL 是 OpenAI 兼容 image_url 接口可识别的格式。
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https":
		return parsedURL.Host != "" && !c.isLocalProjectImageURL(parsedURL)
	case "data":
		return strings.HasPrefix(strings.ToLower(imageURL), "data:image/")
	default:
		return false
	}
}

// isLocalProjectImageURL 判断图片是否为本服务托管的本地静态资源。
func (c *CommentCase) isLocalProjectImageURL(parsedURL *url.URL) bool {
	if parsedURL == nil || !strings.HasPrefix(parsedURL.Path, "/"+c.appInfo.GetProject()+"/") {
		return false
	}
	host := parsedURL.Hostname()
	ip := net.ParseIP(host)
	// 本机或内网地址无法被远端模型服务访问，需要转为图片字节输入。
	if ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}
	return strings.EqualFold(host, "localhost")
}

// commentReviewImagePath 提取评价图片在 OSS 中的对象路径。
func commentReviewImagePath(imagePath string) string {
	parsedURL, err := url.Parse(imagePath)
	if err != nil || parsedURL.Scheme == "" {
		return imagePath
	}
	return parsedURL.Path
}

// commentReviewImageMIMEType 按图片路径推断审核图片 MIME 类型。
func commentReviewImageMIMEType(imagePath string) string {
	lowerPath := strings.ToLower(imagePath)
	queryIndex := strings.Index(lowerPath, "?")
	// 图片路径携带查询参数时，先剔除查询部分再判断扩展名。
	if queryIndex >= 0 {
		lowerPath = lowerPath[:queryIndex]
	}
	// 按常见图片扩展名推断 MIME 类型，未知格式默认按 JPEG 处理。
	switch {
	case strings.HasSuffix(lowerPath, ".png"):
		return "image/png"
	case strings.HasSuffix(lowerPath, ".webp"):
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// summaryTagNamesByIDs 根据评价记录中的标签编号返回标签名称列表。
func summaryTagNamesByIDs(tagIDs []int64, tagNameMap map[int64]string) []string {
	if len(tagIDs) == 0 || len(tagNameMap) == 0 {
		return []string{}
	}

	tagNames := make([]string, 0, len(tagIDs))
	seen := make(map[int64]struct{}, len(tagIDs))
	for _, tagID := range tagIDs {
		// 单条评价命中重复标签时，只向模型传入一次。
		if _, ok := seen[tagID]; ok {
			continue
		}
		tagName := tagNameMap[tagID]
		// 评价引用的标签已不存在或名称为空时跳过。
		if tagName == "" {
			continue
		}
		seen[tagID] = struct{}{}
		tagNames = append(tagNames, tagName)
	}
	return tagNames
}

// newCommentConsumerContext 创建带统一超时的评价队列消费上下文。
func newCommentConsumerContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), commentConsumerTimeout)
}

// commentReviewRejectReason 根据审核结果生成不通过原因。
func commentReviewRejectReason(result *comment.ReviewResult) string {
	if result == nil {
		return "审核服务未返回结果，无法确认内容安全"
	}
	if result.RiskReason != "" {
		return result.RiskReason
	}
	return "LLM审核不通过但未返回具体违规原因"
}

// commentReviewMissingReason 生成缺少具体拒绝原因时的审核异常说明。
func commentReviewMissingReason(result *comment.ReviewResult) string {
	if result == nil {
		return "LLM审核不通过但未返回具体违规原因：审核结果为空"
	}
	rawReason := result.RiskReason
	if rawReason == "" {
		rawReason = "空"
	}
	tagText := "无"
	if len(result.Tags) > 0 {
		tagText = strings.Join(result.Tags, "、")
	}
	return fmt.Sprintf(
		"LLM审核不通过但未返回具体违规原因：approved=%t，textRisk=%t，imageRisk=%t，riskReason=%s，tags=%s",
		result.Approved,
		result.TextRisk,
		result.ImageRisk,
		rawReason,
		tagText,
	)
}
