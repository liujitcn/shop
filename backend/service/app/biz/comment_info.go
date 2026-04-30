package biz

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	appDto "shop/service/app/dto"
	"shop/service/app/utils"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

const (
	ANONYMOUS_USER_NAME = "匿名用户"
)

// CommentInfoCase 评价主表业务处理对象。
type CommentInfoCase struct {
	*biz.BaseCase
	*data.CommentInfoRepository
	mapper *mapper.CopierMapper[appv1.CommentItem, models.CommentInfo]
}

// NewCommentInfoCase 创建评价主表业务处理对象。
func NewCommentInfoCase(baseCase *biz.BaseCase, commentInfoRepo *data.CommentInfoRepository) *CommentInfoCase {
	return &CommentInfoCase{
		BaseCase:              baseCase,
		CommentInfoRepository: commentInfoRepo,
		mapper:                mapper.NewCopierMapper[appv1.CommentItem, models.CommentInfo](),
	}
}

// BuildOverviewSummary 查询商品评价摘要统计。
func (c *CommentInfoCase) BuildOverviewSummary(ctx context.Context, goodsID int64) (*appDto.CommentSummary, error) {
	recordList, err := c.listByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	return c.buildOverviewSummary(recordList), nil
}

// buildOverviewSummary 基于已查询的评价记录构建摘要统计。
func (c *CommentInfoCase) buildOverviewSummary(recordList []*models.CommentInfo) *appDto.CommentSummary {
	totalCount := int32(len(recordList))
	// 当前商品还没有可展示评价时，直接返回空摘要。
	if totalCount == 0 {
		return &appDto.CommentSummary{}
	}

	recentTime := time.Now().AddDate(0, 0, -90)
	recentCount := int32(0)
	recentGoodCount := int32(0)
	for _, record := range recordList {
		// 超出近 90 天窗口的评价不参与近期好评率统计。
		if record.CreatedAt.Before(recentTime) {
			continue
		}
		recentCount++
		// 商品评分大于等于 4 时，计入近期好评数量。
		if record.GoodsScore >= 4 {
			recentGoodCount++
		}
	}
	// 近 90 天没有评价时，仅返回总评价数。
	if recentCount == 0 {
		return &appDto.CommentSummary{
			TotalCount: totalCount,
		}
	}

	return &appDto.CommentSummary{
		TotalCount:     totalCount,
		RecentGoodRate: recentGoodCount * 100 / recentCount,
	}
}

// BuildFilterStats 统计商品评价筛选项数量。
func (c *CommentInfoCase) BuildFilterStats(ctx context.Context, goodsID int64) (*appDto.CommentFilterStats, error) {
	recordList, err := c.listByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}

	stats := &appDto.CommentFilterStats{}
	for _, record := range recordList {
		imageList := _string.ConvertJsonStringToStringArray(record.Img)
		// 存在评价图片时，计入“有图”筛选统计。
		if len(imageList) > 0 {
			stats.MediaCount++
		}
		// 商品评分大于等于 4 时，归入好评统计。
		if record.GoodsScore >= 4 {
			stats.GoodCount++
			continue
		}
		// 商品评分等于 3 时，归入中评统计。
		if record.GoodsScore == 3 {
			stats.MiddleCount++
			continue
		}
		// 剩余评价统一归入差评统计。
		stats.BadCount++
	}
	return stats, nil
}

// ListPreviewByGoodsID 查询商品评价预览列表。
func (c *CommentInfoCase) ListPreviewByGoodsID(ctx context.Context, goodsID int64, previewLimit int32, userID int64) ([]*appv1.CommentItem, error) {
	recordList, err := c.listByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	return c.listPreviewByRecordList(ctx, recordList, previewLimit, userID)
}

// listPreviewByRecordList 基于已查询的评价记录构建商品评价预览列表。
func (c *CommentInfoCase) listPreviewByRecordList(ctx context.Context, recordList []*models.CommentInfo, previewLimit int32, userID int64) ([]*appv1.CommentItem, error) {
	c.sortByDefault(recordList)

	// 未传预览条数时，默认返回 2 条摘要预览。
	if previewLimit <= 0 {
		previewLimit = 2
	}

	pageList := make([]*models.CommentInfo, 0)
	for index, record := range recordList {
		// 已达到预览条数上限时，停止继续组装列表。
		if int32(index) >= previewLimit {
			break
		}
		pageList = append(pageList, record)
	}

	list, err := c.buildCommentItems(ctx, pageList, false, userID)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// PageGoodsComment 查询商品评价分页列表。
func (c *CommentInfoCase) PageGoodsComment(ctx context.Context, req *appv1.PageGoodsCommentRequest, userID int64) ([]*appv1.CommentItem, int32, error) {
	recordList, err := c.listByGoodsID(ctx, req.GetGoodsId())
	if err != nil {
		return nil, 0, err
	}

	filteredList := make([]*models.CommentInfo, 0, len(recordList))
	for _, record := range recordList {
		// 开启“当前商品”且指定 SKU 时，仅保留匹配当前 SKU 的评价。
		if req.GetCurrentGoodsOnly() && req.GetSkuCode() != "" && record.SKUCode != req.GetSkuCode() {
			continue
		}

		imageList := _string.ConvertJsonStringToStringArray(record.Img)
		// 仅筛选有图评价时，要求图片列表非空。
		if req.GetFilterType() == commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_MEDIA) && len(imageList) == 0 {
			continue
		}
		// 好评筛选要求商品评分大于等于 4。
		if req.GetFilterType() == commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_GOOD) && record.GoodsScore < 4 {
			continue
		}
		// 中评筛选要求商品评分等于 3。
		if req.GetFilterType() == commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_MIDDLE) && record.GoodsScore != 3 {
			continue
		}
		// 差评筛选要求商品评分小于等于 2。
		if req.GetFilterType() == commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_BAD) && record.GoodsScore > 2 {
			continue
		}
		// 标签筛选要求传入有效标签并命中评价标签列表。
		if req.GetFilterType() == commonv1.CommentFilterType(_const.COMMENT_FILTER_TYPE_TAG) {
			hasTag := false
			for _, tagID := range _string.ConvertJsonStringToInt64Array(record.TagID) {
				// 命中目标标签时，标记当前评价可进入结果集。
				if tagID == req.GetTagId() && req.GetTagId() > 0 {
					hasTag = true
					break
				}
			}
			// 当前评价未命中目标标签时，跳过当前记录。
			if !hasTag {
				continue
			}
		}
		filteredList = append(filteredList, record)
	}

	// 显式切换到“最新”排序时，按时间倒序返回评价。
	if req.GetSortType() == commonv1.CommentSortType(_const.COMMENT_SORT_TYPE_LATEST) {
		c.sortByLatest(filteredList)
	} else {
		// 其余排序统一回退到推荐排序。
		c.sortByDefault(filteredList)
	}

	total := int32(len(filteredList))
	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())
	pageList := c.paginateRecordList(filteredList, pageNum, pageSize)

	var list []*appv1.CommentItem
	list, err = c.buildCommentItems(ctx, pageList, false, userID)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// PageMyComment 查询我的评价分页列表。
func (c *CommentInfoCase) PageMyComment(ctx context.Context, userID int64, req *appv1.PageMyCommentRequest) ([]*appv1.CommentItem, int32, error) {
	recordList, err := c.listByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	c.sortByLatest(recordList)

	total := int32(len(recordList))
	pageNum, pageSize := repository.PageDefault(req.GetPageNum(), req.GetPageSize())
	pageList := c.paginateRecordList(recordList, pageNum, pageSize)

	var list []*appv1.CommentItem
	list, err = c.buildCommentItems(ctx, pageList, true, userID)
	if err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// FindByID 按评价编号查询审核通过的评价主记录。
func (c *CommentInfoCase) FindByID(ctx context.Context, commentID int64) (*models.CommentInfo, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(commentID)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		// 目标评价不存在时，明确返回资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("评价不存在")
		}
		return nil, err
	}
	return record, nil
}

// FindOwnerByID 按评价编号查询当前用户自己的评价主记录。
func (c *CommentInfoCase) FindOwnerByID(ctx context.Context, commentID int64, userID int64) (*models.CommentInfo, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.ID.Eq(commentID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Status.In(_const.COMMENT_STATUS_PENDING_REVIEW, _const.COMMENT_STATUS_APPROVED, _const.COMMENT_STATUS_REJECTED)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		// 当前用户评价不存在时，统一返回资源不存在。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("评价不存在")
		}
		return nil, err
	}
	return record, nil
}

// FindAnyByID 按评价编号查询未删除评价主记录。
func (c *CommentInfoCase) FindAnyByID(ctx context.Context, commentID int64) (*models.CommentInfo, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(commentID)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		// 目标评价不存在时，明确返回资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("评价不存在")
		}
		return nil, err
	}
	return record, nil
}

// CreateComment 创建商品评价。
func (c *CommentInfoCase) CreateComment(ctx context.Context, user *models.BaseUser, req *appv1.CreateCommentRequest, orderGoods *models.OrderGoods) (*models.CommentInfo, error) {
	userID := int64(0)
	userName := ANONYMOUS_USER_NAME
	userAvatar := ""
	userTagText := "买家"
	// 查询到了当前用户快照时，优先使用真实用户信息构造评价归属。
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

	record := &models.CommentInfo{
		OrderID:              req.GetOrderId(),
		GoodsID:              orderGoods.GoodsID,
		GoodsNameSnapshot:    orderGoods.Name,
		GoodsPictureSnapshot: orderGoods.Picture,
		SKUCode:              orderGoods.SKUCode,
		SKUDescSnapshot:      strings.Join(_string.ConvertJsonStringToStringArray(orderGoods.SpecItem), " / "),
		UserID:               userID,
		UserNameSnapshot:     userName,
		UserAvatarSnapshot:   userAvatar,
		UserTagText:          userTagText,
		IsAnonymous:          req.GetIsAnonymous(),
		GoodsScore:           req.GetGoodsScore(),
		PackageScore:         req.GetPackageScore(),
		DeliveryScore:        req.GetDeliveryScore(),
		Content:              strings.TrimSpace(req.GetContent()),
		TagID:                _string.ConvertAnyToJsonString([]int64{}),
		Img:                  _string.ConvertAnyToJsonString(req.GetImg()),
		Status:               _const.COMMENT_STATUS_PENDING_REVIEW,
	}

	err := c.Create(ctx, record)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// DeleteOwnerComment 逻辑删除当前用户自己的商品评价。
func (c *CommentInfoCase) DeleteOwnerComment(ctx context.Context, commentID int64, userID int64) error {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(commentID)))
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	return c.Delete(ctx, opts...)
}

// UpdateTagIDs 更新评价命中的标签编号列表。
func (c *CommentInfoCase) UpdateTagIDs(ctx context.Context, commentID int64, tagIDs []int64) error {
	query := c.Query(ctx).CommentInfo
	// 标签回写需要判断 RowsAffected，避免把不存在的评价当作更新成功。
	result, err := query.WithContext(ctx).
		Where(query.ID.Eq(commentID)).
		Update(query.TagID, _string.ConvertAnyToJsonString(tagIDs))
	if err != nil {
		return err
	}
	// 目标评价不存在时，无法继续回写标签命中结果。
	if result.RowsAffected == 0 {
		return errorsx.ResourceNotFound("评价不存在")
	}
	return nil
}

// UpdateStatus 更新评价审核状态。
func (c *CommentInfoCase) UpdateStatus(ctx context.Context, commentID int64, status int32) error {
	query := c.Query(ctx).CommentInfo
	result, err := query.WithContext(ctx).
		Where(query.ID.Eq(commentID)).
		Update(query.Status, status)
	if err != nil {
		return err
	}
	// 目标评价不存在时，无法更新审核状态。
	if result.RowsAffected == 0 {
		return errorsx.ResourceNotFound("评价不存在")
	}
	return nil
}

// BuildCommentedOrderGoodsMap 按当前用户的订单商品关联键构建已评价集合。
func (c *CommentInfoCase) BuildCommentedOrderGoodsMap(ctx context.Context, userID int64, orderIDs []int64) (map[string]bool, error) {
	commentedOrderGoodsMap := make(map[string]bool)
	// 用户编号非法或订单编号列表为空时，直接返回空集合。
	if userID <= 0 || len(orderIDs) == 0 {
		return commentedOrderGoodsMap, nil
	}

	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Unscoped())
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.OrderID.In(orderIDs...)))
	commentList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range commentList {
		commentedOrderGoodsMap[utils.BuildOrderGoodsCommentKey(item.OrderID, item.GoodsID, item.SKUCode)] = true
	}
	return commentedOrderGoodsMap, nil
}

// IsOrderGoodsCommented 判断当前用户订单商品是否已经评价。
func (c *CommentInfoCase) IsOrderGoodsCommented(ctx context.Context, userID int64, orderID int64, goodsID int64, skuCode string) (bool, error) {
	// 用户编号非法时，不命中任何已评价记录。
	if userID <= 0 {
		return false, nil
	}

	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Unscoped())
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.OrderID.Eq(orderID)))
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.SKUCode.Eq(strings.TrimSpace(skuCode))))
	count, err := c.Count(ctx, opts...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AreAllOrderGoodsCommented 判断当前用户订单下商品是否全部完成评价。
func (c *CommentInfoCase) AreAllOrderGoodsCommented(ctx context.Context, userID int64, orderGoodsList []*models.OrderGoods) (bool, error) {
	// 订单商品列表为空时，不视为已全部评价。
	if userID <= 0 || len(orderGoodsList) == 0 {
		return false, nil
	}

	expectedCommentMap := make(map[string]bool)
	for _, item := range orderGoodsList {
		expectedCommentMap[utils.BuildOrderGoodsCommentKey(item.OrderID, item.GoodsID, item.SKUCode)] = true
	}

	commentedOrderGoodsMap, err := c.BuildCommentedOrderGoodsMap(ctx, userID, []int64{orderGoodsList[0].OrderID})
	if err != nil {
		return false, err
	}
	for key := range expectedCommentMap {
		// 订单内仍存在未评价商品时，不允许流转到已完成。
		if !commentedOrderGoodsMap[key] {
			return false, nil
		}
	}
	return true, nil
}

// listApprovedByGoodsID 查询商品下全部审核通过评价记录。
func (c *CommentInfoCase) listApprovedByGoodsID(ctx context.Context, goodsID int64) ([]*models.CommentInfo, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
	return c.List(ctx, opts...)
}

// listByGoodsID 查询商品下的全部可展示评价记录。
func (c *CommentInfoCase) listByGoodsID(ctx context.Context, goodsID int64) ([]*models.CommentInfo, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_APPROVED)))
	// 商品详情和评价列表只展示有正文的评价，一键好评等空正文评价仅保留在“我的评价”。
	opts = append(opts, repository.Where(query.Content.Neq("")))
	return c.List(ctx, opts...)
}

// listByUserID 查询用户的全部评价记录。
func (c *CommentInfoCase) listByUserID(ctx context.Context, userID int64) ([]*models.CommentInfo, error) {
	query := c.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.UserID.Eq(userID)))
	opts = append(opts, repository.Where(query.Status.In(_const.COMMENT_STATUS_PENDING_REVIEW, _const.COMMENT_STATUS_APPROVED, _const.COMMENT_STATUS_REJECTED)))
	return c.List(ctx, opts...)
}

// sortByDefault 按推荐排序对评价记录重新排序。
func (c *CommentInfoCase) sortByDefault(recordList []*models.CommentInfo) {
	sort.SliceStable(recordList, func(leftIndex, rightIndex int) bool {
		leftScore := 0
		leftImageList := _string.ConvertJsonStringToStringArray(recordList[leftIndex].Img)
		// 有图评价在默认排序中优先级更高。
		if len(leftImageList) > 0 {
			leftScore += 100
		}
		leftScore += len([]rune(recordList[leftIndex].Content)) / 10
		leftScore += len(leftImageList) * 3

		rightScore := 0
		rightImageList := _string.ConvertJsonStringToStringArray(recordList[rightIndex].Img)
		// 有图评价在默认排序中优先级更高。
		if len(rightImageList) > 0 {
			rightScore += 100
		}
		rightScore += len([]rune(recordList[rightIndex].Content)) / 10
		rightScore += len(rightImageList) * 3

		// 推荐分存在差异时，优先返回分值更高的评价。
		if leftScore != rightScore {
			return leftScore > rightScore
		}
		return recordList[leftIndex].CreatedAt.After(recordList[rightIndex].CreatedAt)
	})
}

// sortByLatest 按创建时间倒序排序评价记录。
func (c *CommentInfoCase) sortByLatest(recordList []*models.CommentInfo) {
	sort.SliceStable(recordList, func(leftIndex, rightIndex int) bool {
		return recordList[leftIndex].CreatedAt.After(recordList[rightIndex].CreatedAt)
	})
}

// buildCommentItems 批量将评价记录转换为接口响应结构。
func (c *CommentInfoCase) buildCommentItems(ctx context.Context, recordList []*models.CommentInfo, ownerView bool, userID int64) ([]*appv1.CommentItem, error) {
	userReactionTypeMap, err := c.buildCommentUserReactionTypeMap(ctx, recordList, userID)
	if err != nil {
		return nil, err
	}

	list := make([]*appv1.CommentItem, 0, len(recordList))
	for _, record := range recordList {
		list = append(list, c.buildCommentItem(record, ownerView, userReactionTypeMap))
	}
	return list, nil
}

// buildCommentUserReactionTypeMap 查询当前用户对评价的互动状态。
func (c *CommentInfoCase) buildCommentUserReactionTypeMap(ctx context.Context, recordList []*models.CommentInfo, userID int64) (map[int64]int32, error) {
	reactionTypeMap := make(map[int64]int32)
	// 未登录或评价列表为空时，无需查询当前用户互动状态。
	if userID <= 0 || len(recordList) == 0 {
		return reactionTypeMap, nil
	}

	commentIDs := make([]int64, 0, len(recordList))
	for _, record := range recordList {
		commentIDs = append(commentIDs, record.ID)
	}

	query := c.Query(ctx).CommentReaction
	rows := make([]*appDto.CommentTargetReactionRow, 0)
	err := query.WithContext(ctx).
		Select(query.TargetID, query.ReactionType).
		Where(
			query.TargetType.Eq(_const.COMMENT_REACTION_TARGET_TYPE_COMMENT),
			query.TargetID.In(commentIDs...),
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

// buildCommentItem 将评价记录转换为接口响应结构。
func (c *CommentInfoCase) buildCommentItem(
	record *models.CommentInfo,
	ownerView bool,
	userReactionTypeMap map[int64]int32,
) *appv1.CommentItem {
	item := c.mapper.ToDTO(record)
	dateText := record.CreatedAt.Format("01-02")
	// “我的评价”需要展示完整年月日。
	if ownerView {
		dateText = record.CreatedAt.Format("2006-01-02")
	}

	userName := record.UserNameSnapshot
	userAvatar := record.UserAvatarSnapshot
	userTagText := record.UserTagText
	// 匿名评价在前台统一隐藏真实昵称、头像和用户标签。
	if record.IsAnonymous {
		userName = ANONYMOUS_USER_NAME
		userAvatar = ""
		userTagText = ""
	}
	// 未提供展示昵称时，回退到匿名用户文案兜底。
	if userName == "" {
		userName = ANONYMOUS_USER_NAME
	}

	imageList := _string.ConvertJsonStringToStringArray(record.Img)
	item.Id = record.ID
	item.GoodsId = record.GoodsID
	item.GoodsName = record.GoodsNameSnapshot
	item.GoodsPicture = record.GoodsPictureSnapshot
	item.SkuDesc = record.SKUDescSnapshot
	item.DateText = dateText
	item.User = &appv1.CommentUserView{
		UserName:    userName,
		Avatar:      userAvatar,
		UserTagText: userTagText,
		Anonymous:   record.IsAnonymous,
	}
	item.ContentSegments = c.buildContentSegments(record.Content, nil)
	item.Img = imageList
	item.ImageCount = int32(len(imageList))
	item.DiscussionCount = record.DiscussionCount
	item.AnonymousForOwner = ownerView && record.IsAnonymous
	item.GoodsScore = record.GoodsScore
	item.Status = commonv1.CommentStatus(record.Status)
	item.LikeCount = record.LikeCount
	item.DislikeCount = record.DislikeCount
	reactionType := userReactionTypeMap[record.ID]
	item.ReactionType = commonv1.CommentReactionType(reactionType)
	return item
}

// buildContentSegments 将正文和高亮词转换为文本片段列表。
func (c *CommentInfoCase) buildContentSegments(content string, highlightWords []string) []*appv1.CommentTextSegment {
	content = strings.TrimSpace(content)
	// 评价正文为空时，直接返回空片段列表。
	if content == "" {
		return []*appv1.CommentTextSegment{}
	}
	// 未配置高亮词时，整段正文按普通文本返回。
	if len(highlightWords) == 0 {
		return []*appv1.CommentTextSegment{{Text: content}}
	}

	result := make([]*appv1.CommentTextSegment, 0)
	remain := content
	for len(remain) > 0 {
		nextIndex := -1
		nextWord := ""
		for _, word := range highlightWords {
			// 空高亮词不参与正文切分。
			if strings.TrimSpace(word) == "" {
				continue
			}
			index := strings.Index(remain, word)
			// 当前高亮词未在剩余正文中出现时，继续尝试其他候选词。
			if index < 0 {
				continue
			}
			// 选择当前位置之后最先出现的高亮词作为下一个切分点。
			if nextIndex < 0 || index < nextIndex {
				nextIndex = index
				nextWord = word
			}
		}
		// 剩余正文里已没有高亮词时，整体作为普通片段返回。
		if nextIndex < 0 {
			result = append(result, &appv1.CommentTextSegment{Text: remain})
			break
		}
		// 高亮词前存在普通正文时，先输出普通片段。
		if nextIndex > 0 {
			result = append(result, &appv1.CommentTextSegment{Text: remain[:nextIndex]})
		}
		result = append(result, &appv1.CommentTextSegment{
			Text:      nextWord,
			Highlight: true,
		})
		remain = remain[nextIndex+len(nextWord):]
	}
	return result
}

// paginateRecordList 按页码和页大小切分评价记录。
func (c *CommentInfoCase) paginateRecordList(recordList []*models.CommentInfo, pageNum, pageSize int64) []*models.CommentInfo {
	start := (pageNum - 1) * pageSize
	// 起始下标越界时，返回空分页结果。
	if start >= int64(len(recordList)) {
		return []*models.CommentInfo{}
	}
	end := start + pageSize
	// 结束下标超过列表长度时，回退到列表末尾。
	if end > int64(len(recordList)) {
		end = int64(len(recordList))
	}
	return append([]*models.CommentInfo(nil), recordList[start:end]...)
}
