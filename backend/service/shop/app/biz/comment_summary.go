package biz

import (
	"context"
	"errors"
	"time"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"

	_const "shop/service/shop/consts"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/service/shop/app/agent/comment"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// CommentSummaryCase 评价摘要业务处理对象。
type CommentSummaryCase struct {
	*biz.BaseCase
	*data.CommentSummaryRepository
	commentReactionRepo  *data.CommentReactionRepository
	commentSummaryMapper *mapper.CopierMapper[shopappv1.CommentSummary, models.CommentSummary]
}

// NewCommentSummaryCase 创建评价摘要业务处理对象。
func NewCommentSummaryCase(
	baseCase *biz.BaseCase,
	commentSummaryRepo *data.CommentSummaryRepository,
	commentReactionRepo *data.CommentReactionRepository,
) *CommentSummaryCase {
	commentSummaryMapper := mapper.NewCopierMapper[shopappv1.CommentSummary, models.CommentSummary]()
	commentSummaryMapper.AppendConverters(mapper.NewJSONTypeConverter[[]*shopcommonv1.CommentSummaryContentItem]().NewConverterPair())
	return &CommentSummaryCase{
		BaseCase:                 baseCase,
		CommentSummaryRepository: commentSummaryRepo,
		commentReactionRepo:      commentReactionRepo,
		commentSummaryMapper:     commentSummaryMapper,
	}
}

// GoodsCommentOverview 查询商品详情评价摘要卡片。
func (c *CommentSummaryCase) GoodsCommentOverview(ctx context.Context, goodsID, userID int64) (*shopappv1.CommentSummary, error) {
	return c.buildCardByGoodsIDAndScene(ctx, goodsID, _const.COMMENT_SUMMARY_SCENE_OVERVIEW, userID)
}

// PageGoodsComment 查询评价列表摘要卡片。
func (c *CommentSummaryCase) PageGoodsComment(ctx context.Context, goodsID, userID int64) (*shopappv1.CommentSummary, error) {
	return c.buildCardByGoodsIDAndScene(ctx, goodsID, _const.COMMENT_SUMMARY_SCENE_LIST, userID)
}

// FindByID 按编号查询评价摘要记录。
func (c *CommentSummaryCase) FindByID(ctx context.Context, summaryID int64) (*models.CommentSummary, error) {
	query := c.Query(ctx).CommentSummary
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(summaryID)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		// 目标评价摘要不存在时，返回资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("评价摘要不存在")
		}
		return nil, err
	}
	return record, nil
}

// UpsertGoodsCommentSummary 保存商品两个场景的评价摘要。
func (c *CommentSummaryCase) UpsertGoodsCommentSummary(ctx context.Context, tenantID, tenantStoreID, goodsID int64, result *comment.SummaryResult) error {
	// 摘要结果为空时，不覆盖旧摘要，避免异常降级影响前台展示。
	if result == nil {
		return nil
	}
	err := c.upsertSceneContent(ctx, tenantID, tenantStoreID, goodsID, _const.COMMENT_SUMMARY_SCENE_OVERVIEW, result.Overview.Content)
	if err != nil {
		return err
	}
	return c.upsertSceneContent(ctx, tenantID, tenantStoreID, goodsID, _const.COMMENT_SUMMARY_SCENE_LIST, result.List.Content)
}

// buildCardByGoodsIDAndScene 按商品和场景查询评价摘要卡片。
func (c *CommentSummaryCase) buildCardByGoodsIDAndScene(ctx context.Context, goodsID int64, scene int32, userID int64) (*shopappv1.CommentSummary, error) {
	query := c.Query(ctx).CommentSummary
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Scene.Eq(scene)))
	opts = append(opts, repository.Order(query.ID))
	opts = append(opts, repository.Limit(1))
	recordList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// 当前商品场景未配置评价摘要时，返回空卡片结构兜底，并避免把正常空数据打成错误日志。
	if len(recordList) == 0 {
		return &shopappv1.CommentSummary{}, nil
	}
	record := recordList[0]

	card := c.commentSummaryMapper.ToDTO(record)
	card.LikeCount = record.LikeCount
	card.DislikeCount = record.DislikeCount
	// 当前请求带有登录用户时，再补充点赞 / 点踩状态。
	if userID > 0 {
		reactionQuery := c.commentReactionRepo.Query(ctx).CommentReaction
		reactionOpts := make([]repository.QueryOption, 0, 3)
		reactionOpts = append(reactionOpts, repository.Where(reactionQuery.TargetType.Eq(_const.COMMENT_REACTION_TARGET_TYPE_SUMMARY)))
		reactionOpts = append(reactionOpts, repository.Where(reactionQuery.TargetID.Eq(record.ID)))
		reactionOpts = append(reactionOpts, repository.Where(reactionQuery.UserID.Eq(userID)))
		reaction, reactionErr := c.commentReactionRepo.Find(ctx, reactionOpts...)
		// 当前用户已经对该评价摘要做过互动时，回填点赞 / 点踩展示状态。
		if reactionErr == nil {
			card.ReactionType = shopcommonv1.CommentReactionType(reaction.ReactionType)
		} else if !errors.Is(reactionErr, gorm.ErrRecordNotFound) {
			return nil, reactionErr
		}
	}
	return card, nil
}

// upsertSceneContent 保存单个场景的评价摘要内容。
func (c *CommentSummaryCase) upsertSceneContent(ctx context.Context, tenantID, tenantStoreID, goodsID int64, scene int32, content []comment.SummaryContentItem) error {
	contentList := make([]*shopcommonv1.CommentSummaryContentItem, 0, len(content))
	for _, item := range content {
		// 摘要内容为空时不进入最终展示内容。
		if item.Content == "" {
			continue
		}
		contentList = append(contentList, &shopcommonv1.CommentSummaryContentItem{
			Label:   item.Label,
			Content: item.Content,
		})
	}
	// 当前场景没有有效摘要内容时，不覆盖旧摘要。
	if len(contentList) == 0 {
		return nil
	}

	contentJSON := _string.ConvertAnyToJsonString(contentList)
	query := c.Query(ctx).CommentSummary
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Scene.Eq(scene)))
	record, err := c.Find(ctx, opts...)
	if err == nil {
		_, err = query.WithContext(ctx).
			Where(query.ID.Eq(record.ID)).
			UpdateSimple(
				query.Content.Value(contentJSON),
				query.UpdatedAt.Value(time.Now()),
			)
		return err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return c.Create(ctx, &models.CommentSummary{
		TenantID:      tenantID,
		TenantStoreID: tenantStoreID,
		GoodsID:       goodsID,
		Scene:         scene,
		Content:       contentJSON,
	})
}
