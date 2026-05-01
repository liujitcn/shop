package biz

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/llm"
	"shop/pkg/queue"
	"shop/pkg/workspaceevent"
	appDto "shop/service/app/dto"
	"shop/service/app/utils"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"gorm.io/gen"
)

// CommentCase 评价业务编排对象。
type CommentCase struct {
	*biz.BaseCase
	tx                    data.Transaction
	commentInfoCase       *CommentInfoCase
	commentAiCase         *CommentAiCase
	commentTagCase        *CommentTagCase
	commentReviewCase     *CommentReviewCase
	commentDiscussionCase *CommentDiscussionCase
	commentReactionCase   *CommentReactionCase
	orderInfoCase         *OrderInfoCase
	orderGoodsCase        *OrderGoodsCase
	baseUserCase          *BaseUserCase
	llmClient             *llm.Client
}

// NewCommentCase 创建评价业务编排对象。
func NewCommentCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	commentInfoCase *CommentInfoCase,
	commentAiCase *CommentAiCase,
	commentTagCase *CommentTagCase,
	commentReviewCase *CommentReviewCase,
	commentDiscussionCase *CommentDiscussionCase,
	commentReactionCase *CommentReactionCase,
	orderInfoCase *OrderInfoCase,
	orderGoodsCase *OrderGoodsCase,
	baseUserCase *BaseUserCase,
	llmClient *llm.Client,
) *CommentCase {
	c := &CommentCase{
		BaseCase:              baseCase,
		tx:                    tx,
		commentInfoCase:       commentInfoCase,
		commentAiCase:         commentAiCase,
		commentTagCase:        commentTagCase,
		commentReviewCase:     commentReviewCase,
		commentDiscussionCase: commentDiscussionCase,
		commentReactionCase:   commentReactionCase,
		orderInfoCase:         orderInfoCase,
		orderGoodsCase:        orderGoodsCase,
		baseUserCase:          baseUserCase,
		llmClient:             llmClient,
	}
	// 注册评价审核与 AI 摘要刷新异步消费者，避免提交评价时阻塞用户主流程。
	c.RegisterQueueConsumer(_const.COMMENT_AUDIT, c.consumeCommentAudit)
	c.RegisterQueueConsumer(_const.COMMENT_AI_REFRESH, c.consumeCommentAiRefresh)
	return c
}

// GoodsCommentOverview 查询商品评价摘要。
func (c *CommentCase) GoodsCommentOverview(ctx context.Context, req *appv1.GoodsCommentOverviewRequest) (*appv1.GoodsCommentOverviewResponse, error) {
	// 请求参数为空时，无法继续查询商品评价摘要。
	if req == nil {
		return nil, errorsx.InvalidArgument("查询条件不能为空")
	}
	// 商品编号非法时，无法继续查询商品评价摘要。
	if req.GetGoodsId() <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

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
		return &appv1.GoodsCommentOverviewResponse{
			TotalCount:     summary.TotalCount,
			RecentDays:     90,
			RecentGoodRate: summary.RecentGoodRate,
			AiSummary:      &appv1.CommentAi{},
		}, nil
	}

	var aiSummary *appv1.CommentAi
	aiSummary, err = c.commentAiCase.GoodsCommentOverview(ctx, req.GetGoodsId(), userID)
	if err != nil {
		return nil, err
	}
	var tagList []*appv1.CommentFilterItem
	tagList, err = c.commentTagCase.ListOverviewTags(ctx, req.GetGoodsId())
	if err != nil {
		return nil, err
	}
	var previewList []*appv1.CommentItem
	previewList, err = c.commentInfoCase.listPreviewByRecordList(ctx, recordList, req.GetPreviewLimit(), userID)
	if err != nil {
		return nil, err
	}

	return &appv1.GoodsCommentOverviewResponse{
		TotalCount:      summary.TotalCount,
		RecentDays:      90,
		RecentGoodRate:  summary.RecentGoodRate,
		AiSummary:       aiSummary,
		CommentFilters:  tagList,
		PreviewComments: previewList,
	}, nil
}

// PageGoodsComment 查询商品评价分页列表。
func (c *CommentCase) PageGoodsComment(ctx context.Context, req *appv1.PageGoodsCommentRequest) (*appv1.PageGoodsCommentResponse, error) {
	// 请求参数为空时，无法继续查询商品评价列表。
	if req == nil {
		return nil, errorsx.InvalidArgument("查询条件不能为空")
	}
	// 商品编号非法时，无法继续查询商品评价列表。
	if req.GetGoodsId() <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

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
	var tagList []*appv1.CommentFilterItem
	tagList, err = c.commentTagCase.ListFilterTags(ctx, req.GetGoodsId())
	if err != nil {
		return nil, err
	}
	var aiSummary *appv1.CommentAi
	aiSummary, err = c.commentAiCase.PageGoodsComment(ctx, req.GetGoodsId(), userID)
	if err != nil {
		return nil, err
	}
	var list []*appv1.CommentItem
	total := int32(0)
	list, total, err = c.commentInfoCase.PageGoodsComment(ctx, req, userID)
	if err != nil {
		return nil, err
	}

	filterList := make([]*appv1.CommentFilterItem, 0, len(tagList)+5)
	filterList = append(filterList, &appv1.CommentFilterItem{
		FilterType: commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_ALL),
		TagId:      0,
		Label:      "全部",
		Value:      strconv.FormatInt(int64(summary.RecentGoodRate), 10) + "%好评",
	})
	filterList = append(filterList, &appv1.CommentFilterItem{
		FilterType: commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_MEDIA),
		TagId:      0,
		Label:      "有图",
		Value:      strconv.FormatInt(int64(filterStats.MediaCount), 10),
	})
	filterList = append(filterList, tagList...)
	filterList = append(filterList, &appv1.CommentFilterItem{
		FilterType: commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_GOOD),
		TagId:      0,
		Label:      "好评",
		Value:      strconv.FormatInt(int64(filterStats.GoodCount), 10),
	})
	filterList = append(filterList, &appv1.CommentFilterItem{
		FilterType: commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_MIDDLE),
		TagId:      0,
		Label:      "中评",
		Value:      strconv.FormatInt(int64(filterStats.MiddleCount), 10),
	})
	filterList = append(filterList, &appv1.CommentFilterItem{
		FilterType: commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_BAD),
		TagId:      0,
		Label:      "差评",
		Value:      strconv.FormatInt(int64(filterStats.BadCount), 10),
	})

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	return &appv1.PageGoodsCommentResponse{
		CommentFilters: filterList,
		AiSummary:      aiSummary,
		Comments:       list,
		Total:          total,
		PageNum:        pageNum,
		PageSize:       pageSize,
		HasMore:        pageNum*pageSize < int64(total),
	}, nil
}

// PageCommentDiscussion 查询评价讨论分页列表。
func (c *CommentCase) PageCommentDiscussion(ctx context.Context, req *appv1.PageCommentDiscussionRequest) (*appv1.PageCommentDiscussionResponse, error) {
	// 请求参数为空时，无法继续查询评价讨论列表。
	if req == nil {
		return nil, errorsx.InvalidArgument("查询条件不能为空")
	}
	// 评价编号非法时，无法继续查询评价讨论列表。
	if req.GetCommentId() <= 0 {
		return nil, errorsx.InvalidArgument("评价编号不能为空")
	}

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

	var list []*appv1.CommentDiscussionItem
	total := int32(0)
	list, total, err = c.commentDiscussionCase.PageCommentDiscussion(ctx, req.GetCommentId(), userID, req)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	return &appv1.PageCommentDiscussionResponse{
		CommentId:          req.GetCommentId(),
		CommentDiscussions: list,
		Total:              total,
		PageNum:            pageNum,
		PageSize:           pageSize,
		HasMore:            pageNum*pageSize < int64(total),
	}, nil
}

// CreateCommentDiscussion 发布评价讨论。
func (c *CommentCase) CreateCommentDiscussion(ctx context.Context, req *appv1.CreateCommentDiscussionRequest) (*appv1.CreateCommentDiscussionResponse, error) {
	// 请求参数为空时，无法继续创建评价讨论。
	if req == nil {
		return nil, errorsx.InvalidArgument("请求参数不能为空")
	}
	// 评价编号非法时，无法继续创建评价讨论。
	if req.GetCommentId() <= 0 {
		return nil, errorsx.InvalidArgument("评价编号不能为空")
	}

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
	err = c.transaction(ctx, func(txCtx context.Context) error {
		record, err = c.commentDiscussionCase.CreateDiscussion(txCtx, user, req)
		if err != nil {
			return err
		}
		return c.changeCommentDiscussionCount(txCtx, req.GetCommentId(), _const.COMMENT_STATUS_PENDING_REVIEW, 1)
	})
	if err != nil {
		return nil, err
	}
	queue.DispatchCommentAudit(_const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID)
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo)

	response := &appv1.CreateCommentDiscussionResponse{
		DiscussionCount: commentInfo.DiscussionCount,
	}
	// 讨论默认待审核，未审核通过前不返回到公开讨论列表。
	if record.Status == _const.COMMENT_STATUS_APPROVED {
		response.DiscussionCount = commentInfo.DiscussionCount + 1
		response.Item = c.commentDiscussionCase.buildDiscussionItem(record, map[int64]int32{})
	}
	return response, nil
}

// SaveCommentReaction 保存评价互动状态。
func (c *CommentCase) SaveCommentReaction(ctx context.Context, req *appv1.SaveCommentReactionRequest) (*appv1.SaveCommentReactionResponse, error) {
	// 请求参数为空时，无法继续保存互动状态。
	if req == nil {
		return nil, errorsx.InvalidArgument("请求参数不能为空")
	}
	// 互动目标编号非法时，无法继续保存互动状态。
	if req.GetTargetId() <= 0 {
		return nil, errorsx.InvalidArgument("互动目标编号不能为空")
	}
	// 互动类型非法时，无法继续保存互动状态。
	if req.GetReactionType() != commonv1.CommentReactionType(_const.COMMENT_REACTION_TYPE_LIKE) &&
		req.GetReactionType() != commonv1.CommentReactionType(_const.COMMENT_REACTION_TYPE_DISLIKE) {
		return nil, errorsx.InvalidArgument("互动类型不支持")
	}

	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	// 不同互动目标使用各自的存在性校验和行为限制。
	switch req.GetTargetType() {
	case commonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_AI):
		_, err = c.commentAiCase.FindByID(ctx, req.GetTargetId())
		if err != nil {
			return nil, err
		}
	case commonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_DISCUSSION):
		// 讨论互动当前只支持点赞，不支持点踩。
		if req.GetReactionType() != commonv1.CommentReactionType(_const.COMMENT_REACTION_TYPE_LIKE) {
			return nil, errorsx.InvalidArgument("讨论仅支持点赞")
		}
		_, err = c.commentDiscussionCase.FindByID(ctx, req.GetTargetId())
		if err != nil {
			return nil, err
		}
	case commonv1.CommentReactionTargetType(_const.COMMENT_REACTION_TARGET_TYPE_COMMENT):
		// 评价互动支持点赞和点踩，但只允许对审核通过的评价操作。
		_, err = c.commentInfoCase.FindByID(ctx, req.GetTargetId())
		if err != nil {
			return nil, err
		}
	default:
		return nil, errorsx.InvalidArgument("互动目标类型不支持")
	}

	var response *appv1.SaveCommentReactionResponse
	err = c.transaction(ctx, func(txCtx context.Context) error {
		response, err = c.commentReactionCase.SaveCommentReaction(txCtx, authInfo.UserId, req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// PagePendingCommentGoods 查询待评价商品分页列表。
func (c *CommentCase) PagePendingCommentGoods(ctx context.Context, req *appv1.PagePendingCommentGoodsRequest) (*appv1.PagePendingCommentGoodsResponse, error) {
	// 请求参数为空时，无法继续查询待评价商品列表。
	if req == nil {
		return nil, errorsx.InvalidArgument("查询条件不能为空")
	}

	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	orderQuery := c.orderInfoCase.Query(ctx).OrderInfo
	orderOpts := make([]repository.QueryOption, 0, 3)
	orderOpts = append(orderOpts, repository.Where(orderQuery.UserID.Eq(authInfo.UserId)))
	orderOpts = append(orderOpts, repository.Where(orderQuery.Status.Eq(_const.ORDER_STATUS_WAIT_REVIEW)))
	orderOpts = append(orderOpts, repository.Order(orderQuery.CreatedAt.Desc()))
	var orderList []*models.OrderInfo
	orderList, err = c.orderInfoCase.List(ctx, orderOpts...)
	if err != nil {
		return nil, err
	}

	orderIDs := make([]int64, 0, len(orderList))
	for _, orderInfo := range orderList {
		orderIDs = append(orderIDs, orderInfo.ID)
	}
	// 当前用户不存在待评价订单时，直接返回空分页结果。
	if len(orderIDs) == 0 {
		return &appv1.PagePendingCommentGoodsResponse{
			PendingCommentGoods: []*appv1.PendingCommentGoodsItem{},
			Total:               0,
			PageNum:             pageNum,
			PageSize:            pageSize,
			HasMore:             false,
		}, nil
	}

	orderGoodsQuery := c.orderGoodsCase.Query(ctx).OrderGoods
	orderGoodsOpts := make([]repository.QueryOption, 0, 2)
	orderGoodsOpts = append(orderGoodsOpts, repository.Where(orderGoodsQuery.OrderID.In(orderIDs...)))
	orderGoodsOpts = append(orderGoodsOpts, repository.Order(orderGoodsQuery.ID.Desc()))
	var orderGoodsList []*models.OrderGoods
	orderGoodsList, err = c.orderGoodsCase.List(ctx, orderGoodsOpts...)
	if err != nil {
		return nil, err
	}

	commentedOrderGoodsMap := make(map[string]bool)
	// 当前批次存在待评价订单时，预先批量查询已评价商品关联键集合。
	if len(orderIDs) > 0 {
		commentedOrderGoodsMap, err = c.commentInfoCase.BuildCommentedOrderGoodsMap(ctx, authInfo.UserId, orderIDs)
		if err != nil {
			return nil, err
		}
	}

	orderGoodsMap := make(map[int64][]*models.OrderGoods)
	for _, orderGoods := range orderGoodsList {
		// 当前订单商品已经完成评价时，不再进入待评价列表。
		if commentedOrderGoodsMap[utils.BuildOrderGoodsCommentKey(orderGoods.OrderID, orderGoods.GoodsID, orderGoods.SKUCode)] {
			continue
		}
		orderGoodsMap[orderGoods.OrderID] = append(orderGoodsMap[orderGoods.OrderID], orderGoods)
	}

	pendingList := make([]*appv1.PendingCommentGoodsItem, 0)
	for _, orderInfo := range orderList {
		for _, orderGoods := range orderGoodsMap[orderInfo.ID] {
			pendingList = append(pendingList, &appv1.PendingCommentGoodsItem{
				OrderId:      orderInfo.ID,
				GoodsId:      orderGoods.GoodsID,
				GoodsName:    orderGoods.Name,
				GoodsPicture: orderGoods.Picture,
				SkuCode:      orderGoods.SKUCode,
				SkuDesc:      strings.Join(_string.ConvertJsonStringToStringArray(orderGoods.SpecItem), " / "),
				Desc:         "分享你的使用体验，帮助其他买家更好选择",
			})
		}
	}

	total := int32(len(pendingList))
	start := (pageNum - 1) * pageSize
	// 起始下标越界时，直接返回空分页结果。
	if start >= int64(len(pendingList)) {
		return &appv1.PagePendingCommentGoodsResponse{
			PendingCommentGoods: []*appv1.PendingCommentGoodsItem{},
			Total:               total,
			PageNum:             pageNum,
			PageSize:            pageSize,
			HasMore:             false,
		}, nil
	}
	end := start + pageSize
	// 结束下标超过列表长度时，回退到列表末尾。
	if end > int64(len(pendingList)) {
		end = int64(len(pendingList))
	}

	return &appv1.PagePendingCommentGoodsResponse{
		PendingCommentGoods: append([]*appv1.PendingCommentGoodsItem(nil), pendingList[start:end]...),
		Total:               total,
		PageNum:             pageNum,
		PageSize:            pageSize,
		HasMore:             end < int64(total),
	}, nil
}

// CreateComment 发布商品评价。
func (c *CommentCase) CreateComment(ctx context.Context, req *appv1.CreateCommentRequest) (*appv1.CreateCommentResponse, error) {
	// 请求参数为空时，无法继续创建评价。
	if req == nil {
		return nil, errorsx.InvalidArgument("请求参数不能为空")
	}
	// 订单编号非法时，无法继续创建评价。
	if req.GetOrderId() <= 0 {
		return nil, errorsx.InvalidArgument("订单编号不能为空")
	}
	// 商品编号非法时，无法继续创建评价。
	if req.GetGoodsId() <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}
	// SKU 编码为空时，无法继续创建评价。
	if strings.TrimSpace(req.GetSkuCode()) == "" {
		return nil, errorsx.InvalidArgument("SKU编码不能为空")
	}
	// 评价图片超过当前页面允许的最大数量时，拒绝当前发布请求。
	if len(req.GetImg()) > 6 {
		return nil, errorsx.InvalidArgument("评价图片最多上传6张")
	}
	content := strings.TrimSpace(req.GetContent())
	// 评价正文超过当前页面允许的最大长度时，拒绝当前发布请求。
	if len([]rune(content)) > 500 {
		return nil, errorsx.InvalidArgument("评价正文不能超过500字")
	}
	// 商品评分超出 1 到 5 分范围时，拒绝当前发布请求。
	if req.GetGoodsScore() < 1 || req.GetGoodsScore() > 5 {
		return nil, errorsx.InvalidArgument("商品评分范围必须在1到5之间")
	}
	// 包装评分超出 1 到 5 分范围时，拒绝当前发布请求。
	if req.GetPackageScore() < 1 || req.GetPackageScore() > 5 {
		return nil, errorsx.InvalidArgument("包装评分范围必须在1到5之间")
	}
	// 送货评分超出 1 到 5 分范围时，拒绝当前发布请求。
	if req.GetDeliveryScore() < 1 || req.GetDeliveryScore() > 5 {
		return nil, errorsx.InvalidArgument("送货评分范围必须在1到5之间")
	}

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
	if orderInfo.Status != _const.ORDER_STATUS_WAIT_REVIEW {
		return nil, errorsx.InvalidArgument("当前订单不可评价")
	}

	query := c.orderGoodsCase.Query(ctx).OrderGoods
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.OrderID.Eq(req.GetOrderId())))
	opts = append(opts, repository.Where(query.GoodsID.Eq(req.GetGoodsId())))
	opts = append(opts, repository.Where(query.SKUCode.Eq(strings.TrimSpace(req.GetSkuCode()))))
	var orderGoods *models.OrderGoods
	orderGoods, err = c.orderGoodsCase.Find(ctx, opts...)
	if err != nil {
		return nil, errorsx.ResourceNotFound("订单商品不存在").WithCause(err)
	}
	// 当前订单商品已经评价过时，不允许继续重复评价。
	isCommented := false
	isCommented, err = c.commentInfoCase.IsOrderGoodsCommented(ctx, authInfo.UserId, req.GetOrderId(), req.GetGoodsId(), req.GetSkuCode())
	if err != nil {
		return nil, err
	}
	if isCommented {
		return nil, errorsx.InvalidArgument("当前商品已评价")
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
	err = c.transaction(ctx, func(txCtx context.Context) error {
		record, err = c.commentInfoCase.CreateComment(txCtx, user, req, orderGoods)
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
				Status: _const.ORDER_STATUS_COMPLETED,
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

	return &appv1.CreateCommentResponse{
		CommentId:      record.ID,
		OrderId:        req.GetOrderId(),
		OrderCompleted: orderCompleted,
	}, nil
}

// DeleteComment 删除商品评价。
func (c *CommentCase) DeleteComment(ctx context.Context, commentID int64) error {
	// 评价编号非法时，无法继续删除评价。
	if commentID <= 0 {
		return errorsx.InvalidArgument("评价编号不能为空")
	}

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
	err = c.transaction(ctx, func(txCtx context.Context) error {
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

// PageMyComment 查询我的评价分页列表。
func (c *CommentCase) PageMyComment(ctx context.Context, req *appv1.PageMyCommentRequest) (*appv1.PageMyCommentResponse, error) {
	// 请求参数为空时，无法继续查询我的评价列表。
	if req == nil {
		return nil, errorsx.InvalidArgument("查询条件不能为空")
	}

	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}

	var list []*appv1.CommentItem
	total := int32(0)
	list, total, err = c.commentInfoCase.PageMyComment(ctx, authInfo.UserId, req)
	if err != nil {
		return nil, err
	}

	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())

	return &appv1.PageMyCommentResponse{
		Comments: list,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		HasMore:  pageNum*pageSize < int64(total),
	}, nil
}

// consumeCommentAudit 消费评价与讨论审核队列。
func (c *CommentCase) consumeCommentAudit(message queueData.Message) error {
	event, err := queue.DecodeQueueData[queue.CommentAuditEvent](message)
	if err != nil {
		return err
	}
	// 队列消息缺失目标时直接忽略，避免无效消息反复重试。
	if event == nil || event.TargetID <= 0 {
		return nil
	}

	ctx := context.TODO()
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
	// 仅待审核评价进入 AI 审核，避免人工已处理后被异步消息覆盖。
	if record.Status != _const.COMMENT_STATUS_PENDING_REVIEW {
		return nil
	}
	if !c.llmClient.Enabled() {
		return c.createAIReview(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, commentID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM客户端未配置")
	}

	result, err := c.llmClient.ReviewComment(ctx, llm.CommentReviewRequest{
		GoodsName: record.GoodsNameSnapshot,
		SKUDesc:   record.SKUDescSnapshot,
		Content:   record.Content,
		ImageURLs: _string.ConvertJsonStringToStringArray(record.Img),
	})
	if err != nil {
		return c.createAIReview(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, commentID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error())
	}

	if result == nil {
		return c.createAIReview(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, commentID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM审核结果为空")
	}
	if !result.Approved {
		return c.rejectCommentByAI(ctx, record, result)
	}
	return c.approveCommentByAI(ctx, record, result)
}

// approveCommentByAI 将评价审核通过结果写入业务表和审核记录。
func (c *CommentCase) approveCommentByAI(ctx context.Context, record *models.CommentInfo, result *llm.CommentReviewResult) error {
	cleanTags := cleanCommentTagNames(result.Tags)
	err := c.transaction(ctx, func(txCtx context.Context) error {
		tagIDs, tagNames, upsertErr := c.commentTagCase.UpsertTagsByNames(txCtx, record.GoodsID, cleanTags)
		if upsertErr != nil {
			return upsertErr
		}
		updateErr := c.commentInfoCase.UpdateTagIDs(txCtx, record.ID, tagIDs)
		if updateErr != nil {
			return updateErr
		}
		updateErr = c.commentInfoCase.UpdateStatus(txCtx, record.ID, _const.COMMENT_STATUS_APPROVED)
		if updateErr != nil {
			return updateErr
		}
		return c.createAIReview(txCtx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_APPROVED, tagNames, "")
	})
	if err != nil {
		return err
	}
	queue.DispatchCommentAiRefresh(record.GoodsID)
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaRisk, workspaceevent.AreaReputation, workspaceevent.AreaPendingComments)
	return nil
}

// rejectCommentByAI 将评价审核不通过结果写入业务表和审核记录。
func (c *CommentCase) rejectCommentByAI(ctx context.Context, record *models.CommentInfo, result *llm.CommentReviewResult) error {
	reason := strings.TrimSpace(result.RiskReason)
	// 模型没有给出明确原因时，使用统一兜底文案方便后台识别。
	if reason == "" {
		reason = "LLM审核不通过"
	}
	err := c.transaction(ctx, func(txCtx context.Context) error {
		err := c.commentInfoCase.UpdateStatus(txCtx, record.ID, _const.COMMENT_STATUS_REJECTED)
		if err != nil {
			return err
		}
		return c.createAIReview(txCtx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, record.ID, _const.COMMENT_REVIEW_STATUS_REJECTED, result.Tags, reason)
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaRisk, workspaceevent.AreaPendingComments)
	return nil
}

// auditDiscussion 执行单条讨论的 AI 审核流程。
func (c *CommentCase) auditDiscussion(ctx context.Context, discussionID int64) error {
	record, err := c.findAnyDiscussionByID(ctx, discussionID)
	if err != nil {
		return err
	}
	// 仅待审核讨论进入 AI 审核，避免人工已处理后被异步消息覆盖。
	if record.Status != _const.COMMENT_STATUS_PENDING_REVIEW {
		return nil
	}
	commentInfo, err := c.commentInfoCase.FindAnyByID(ctx, record.CommentID)
	if err != nil {
		return err
	}
	if !c.llmClient.Enabled() {
		return c.createAIReview(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, discussionID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM客户端未配置")
	}

	result, err := c.llmClient.ReviewComment(ctx, llm.CommentReviewRequest{
		GoodsName: commentInfo.GoodsNameSnapshot,
		SKUDesc:   commentInfo.SKUDescSnapshot,
		Content:   record.Content,
	})
	if err != nil {
		return c.createAIReview(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, discussionID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, err.Error())
	}
	if result == nil {
		return c.createAIReview(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, discussionID, _const.COMMENT_REVIEW_STATUS_EXCEPTION, nil, "LLM审核结果为空")
	}
	if !result.Approved {
		return c.rejectDiscussionByAI(ctx, record, result)
	}
	return c.approveDiscussionByAI(ctx, record, result)
}

// approveDiscussionByAI 将讨论审核通过结果写入业务表和审核记录。
func (c *CommentCase) approveDiscussionByAI(ctx context.Context, record *models.CommentDiscussion, result *llm.CommentReviewResult) error {
	err := c.transaction(ctx, func(txCtx context.Context) error {
		err := c.updateDiscussionStatus(txCtx, record.ID, _const.COMMENT_STATUS_APPROVED)
		if err != nil {
			return err
		}
		err = c.changeCommentDiscussionCount(txCtx, record.CommentID, _const.COMMENT_STATUS_PENDING_REVIEW, -1)
		if err != nil {
			return err
		}
		err = c.changeCommentDiscussionCount(txCtx, record.CommentID, _const.COMMENT_STATUS_APPROVED, 1)
		if err != nil {
			return err
		}
		return c.createAIReview(txCtx, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_APPROVED, result.Tags, "")
	})
	if err != nil {
		return err
	}
	commentInfo, findErr := c.commentInfoCase.FindAnyByID(ctx, record.CommentID)
	if findErr == nil {
		queue.DispatchCommentAiRefresh(commentInfo.GoodsID)
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo, workspaceevent.AreaReputation)
	return nil
}

// rejectDiscussionByAI 将讨论审核不通过结果写入业务表和审核记录。
func (c *CommentCase) rejectDiscussionByAI(ctx context.Context, record *models.CommentDiscussion, result *llm.CommentReviewResult) error {
	reason := strings.TrimSpace(result.RiskReason)
	// 模型没有给出明确原因时，使用统一兜底文案方便后台识别。
	if reason == "" {
		reason = "LLM审核不通过"
	}
	err := c.transaction(ctx, func(txCtx context.Context) error {
		err := c.updateDiscussionStatus(txCtx, record.ID, _const.COMMENT_STATUS_REJECTED)
		if err != nil {
			return err
		}
		err = c.changeCommentDiscussionCount(txCtx, record.CommentID, _const.COMMENT_STATUS_PENDING_REVIEW, -1)
		if err != nil {
			return err
		}
		return c.createAIReview(txCtx, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, record.ID, _const.COMMENT_REVIEW_STATUS_REJECTED, result.Tags, reason)
	})
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaTodo)
	return nil
}

// consumeCommentAiRefresh 消费商品评价 AI 摘要刷新队列。
func (c *CommentCase) consumeCommentAiRefresh(message queueData.Message) error {
	goodsID, err := queue.DecodeQueueData[int64](message)
	if err != nil {
		return err
	}
	// 商品编号缺失时直接忽略。
	if goodsID == nil || *goodsID <= 0 {
		return nil
	}
	return c.refreshGoodsCommentAi(context.TODO(), *goodsID)
}

// refreshGoodsCommentAi 基于审核通过评价刷新商品 AI 摘要。
func (c *CommentCase) refreshGoodsCommentAi(ctx context.Context, goodsID int64) error {
	// LLM 未配置时不刷新摘要，前台继续使用旧摘要或空摘要降级。
	if !c.llmClient.Enabled() {
		return nil
	}
	commentList, err := c.commentInfoCase.listApprovedByGoodsID(ctx, goodsID)
	if err != nil {
		return err
	}
	// 当前商品暂无通过评价时，不生成空摘要覆盖旧内容。
	if len(commentList) == 0 {
		return nil
	}

	goodsName := ""
	comments := make([]llm.CommentAiComment, 0, len(commentList))
	for _, item := range commentList {
		if goodsName == "" {
			goodsName = item.GoodsNameSnapshot
		}
		comments = append(comments, llm.CommentAiComment{
			Content:       item.Content,
			GoodsScore:    item.GoodsScore,
			PackageScore:  item.PackageScore,
			DeliveryScore: item.DeliveryScore,
			Tags:          c.tagNamesByIDs(ctx, item.GoodsID, _string.ConvertJsonStringToInt64Array(item.TagID)),
		})
	}
	result, err := c.llmClient.GenerateCommentAi(ctx, llm.CommentAiRequest{
		GoodsName: goodsName,
		Comments:  comments,
	})
	if err != nil {
		return err
	}
	err = c.commentAiCase.UpsertGoodsCommentAi(ctx, goodsID, result)
	if err != nil {
		return err
	}
	workspaceevent.Publish(ctx, workspaceevent.ReasonCommentChanged, workspaceevent.AreaReputation)
	return nil
}

// tagNamesByIDs 根据标签编号查询标签名称，失败时降级为空列表避免影响摘要主流程。
func (c *CommentCase) tagNamesByIDs(ctx context.Context, goodsID int64, tagIDs []int64) []string {
	if len(tagIDs) == 0 {
		return []string{}
	}
	query := c.commentTagCase.Query(ctx).CommentTag
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.ID.In(tagIDs...)))
	tagList, err := c.commentTagCase.List(ctx, opts...)
	if err != nil {
		return []string{}
	}
	tagNames := make([]string, 0, len(tagList))
	for _, tag := range tagList {
		tagNames = append(tagNames, tag.Name)
	}
	return tagNames
}

// createAIReview 创建 AI 审核记录。
func (c *CommentCase) createAIReview(ctx context.Context, targetType int32, targetID int64, status int32, tags []string, reason string) error {
	operatorName := c.llmClient.Model()
	// 模型名称为空时，使用统一名称区分 AI 审核来源。
	if operatorName == "" {
		operatorName = "LLM"
	}
	return c.commentReviewCase.CreateReview(ctx, &models.CommentReview{
		TargetType:   targetType,
		TargetID:     targetID,
		Type:         _const.COMMENT_REVIEW_TYPE_AI,
		Status:       status,
		Tags:         jsonStringTagNames(tags),
		OperatorID:   0,
		OperatorName: operatorName,
		Reason:       strings.TrimSpace(reason),
	})
}

// findAnyDiscussionByID 按编号查询未删除讨论记录。
func (c *CommentCase) findAnyDiscussionByID(ctx context.Context, discussionID int64) (*models.CommentDiscussion, error) {
	query := c.commentDiscussionCase.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(discussionID)))
	record, err := c.commentDiscussionCase.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// updateDiscussionStatus 更新讨论审核状态。
func (c *CommentCase) updateDiscussionStatus(ctx context.Context, discussionID int64, status int32) error {
	query := c.commentDiscussionCase.Query(ctx).CommentDiscussion
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

// changeCommentDiscussionCount 按审核状态调整评价讨论缓存数量。
func (c *CommentCase) changeCommentDiscussionCount(ctx context.Context, commentID int64, status int32, delta int32) error {
	if delta == 0 {
		return nil
	}
	query := c.commentInfoCase.Query(ctx).CommentInfo
	update := query.DiscussionCount.Add(delta)
	conditions := []gen.Condition{query.ID.Eq(commentID)}
	switch status {
	case _const.COMMENT_STATUS_PENDING_REVIEW:
		update = query.PendingDiscussionCount.Add(delta)
		// 递减待审数量时，增加大于 0 条件避免缓存数量出现负数。
		if delta < 0 {
			conditions = append(conditions, query.PendingDiscussionCount.Gt(0))
		}
	case _const.COMMENT_STATUS_APPROVED:
		update = query.DiscussionCount.Add(delta)
		// 递减通过数量时，增加大于 0 条件避免缓存数量出现负数。
		if delta < 0 {
			conditions = append(conditions, query.DiscussionCount.Gt(0))
		}
	default:
		return nil
	}
	_, err := query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(update)
	return err
}

// formatCommentStatus 将审核状态转为日志可读文案。
func formatCommentStatus(status int32) string {
	switch status {
	case _const.COMMENT_STATUS_PENDING_REVIEW:
		return "待审核"
	case _const.COMMENT_STATUS_APPROVED:
		return "审核通过"
	case _const.COMMENT_STATUS_REJECTED:
		return "审核不通过"
	default:
		return fmt.Sprintf("未知状态%d", status)
	}
}

// transaction 使用仓库统一事务执行评论写入逻辑。
func (c *CommentCase) transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		return fn(txCtx)
	})
}
