package biz

import (
	"context"
	"errors"
	"time"

	_const "shop/pkg/const"

	appv1 "shop/api/gen/go/app/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"shop/pkg/llm"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// CommentAiCase 评价 AI 摘要业务处理对象。
type CommentAiCase struct {
	*biz.BaseCase
	*data.CommentAiRepository
	commentReactionRepo *data.CommentReactionRepository
	commentAiMapper     *mapper.CopierMapper[appv1.CommentAi, models.CommentAi]
}

// NewCommentAiCase 创建评价 AI 摘要业务处理对象。
func NewCommentAiCase(
	baseCase *biz.BaseCase,
	commentAiRepo *data.CommentAiRepository,
	commentReactionRepo *data.CommentReactionRepository,
) *CommentAiCase {
	commentAiMapper := mapper.NewCopierMapper[appv1.CommentAi, models.CommentAi]()
	commentAiMapper.AppendConverters(mapper.NewJSONTypeConverter[[]*commonv1.CommentAiContentItem]().NewConverterPair())
	return &CommentAiCase{
		BaseCase:            baseCase,
		CommentAiRepository: commentAiRepo,
		commentReactionRepo: commentReactionRepo,
		commentAiMapper:     commentAiMapper,
	}
}

// GoodsCommentOverview 查询商品详情评价摘要 AI 卡片。
func (c *CommentAiCase) GoodsCommentOverview(ctx context.Context, goodsID, userID int64) (*appv1.CommentAi, error) {
	return c.buildCardByGoodsIDAndScene(ctx, goodsID, _const.COMMENT_AI_SCENE_OVERVIEW, userID)
}

// PageGoodsComment 查询评价列表 AI 卡片。
func (c *CommentAiCase) PageGoodsComment(ctx context.Context, goodsID, userID int64) (*appv1.CommentAi, error) {
	return c.buildCardByGoodsIDAndScene(ctx, goodsID, _const.COMMENT_AI_SCENE_LIST, userID)
}

// FindByID 按编号查询 AI 摘要记录。
func (c *CommentAiCase) FindByID(ctx context.Context, aiID int64) (*models.CommentAi, error) {
	query := c.Query(ctx).CommentAi
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(aiID)))
	record, err := c.Find(ctx, opts...)
	if err != nil {
		// 目标 AI 摘要不存在时，返回资源不存在错误。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("评价摘要不存在")
		}
		return nil, err
	}
	return record, nil
}

// UpsertGoodsCommentAi 保存商品两个场景的评价 AI 摘要。
func (c *CommentAiCase) UpsertGoodsCommentAi(ctx context.Context, goodsID int64, result *llm.CommentAiResult) error {
	// 摘要结果为空时，不覆盖旧摘要，避免异常降级影响前台展示。
	if result == nil {
		return nil
	}
	err := c.upsertSceneContent(ctx, goodsID, _const.COMMENT_AI_SCENE_OVERVIEW, result.Overview.Content)
	if err != nil {
		return err
	}
	return c.upsertSceneContent(ctx, goodsID, _const.COMMENT_AI_SCENE_LIST, result.List.Content)
}

// buildCardByGoodsIDAndScene 按商品和场景查询 AI 摘要卡片。
func (c *CommentAiCase) buildCardByGoodsIDAndScene(ctx context.Context, goodsID int64, scene int32, userID int64) (*appv1.CommentAi, error) {
	query := c.Query(ctx).CommentAi
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Scene.Eq(scene)))
	opts = append(opts, repository.Order(query.ID))
	opts = append(opts, repository.Limit(1))
	recordList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// 当前商品场景未配置 AI 摘要时，返回空卡片结构兜底，并避免把正常空数据打成错误日志。
	if len(recordList) == 0 {
		return &appv1.CommentAi{}, nil
	}
	record := recordList[0]

	card := c.commentAiMapper.ToDTO(record)
	card.LikeCount = record.LikeCount
	card.DislikeCount = record.DislikeCount
	// 当前请求带有登录用户时，再补充点赞 / 点踩状态。
	if userID > 0 {
		reactionQuery := c.commentReactionRepo.Query(ctx).CommentReaction
		reactionOpts := make([]repository.QueryOption, 0, 3)
		reactionOpts = append(reactionOpts, repository.Where(reactionQuery.TargetType.Eq(_const.COMMENT_REACTION_TARGET_TYPE_AI)))
		reactionOpts = append(reactionOpts, repository.Where(reactionQuery.TargetID.Eq(record.ID)))
		reactionOpts = append(reactionOpts, repository.Where(reactionQuery.UserID.Eq(userID)))
		reaction, reactionErr := c.commentReactionRepo.Find(ctx, reactionOpts...)
		// 当前用户已经对该 AI 摘要做过互动时，回填点赞 / 点踩展示状态。
		if reactionErr == nil {
			card.ReactionType = commonv1.CommentReactionType(reaction.ReactionType)
		} else if !errors.Is(reactionErr, gorm.ErrRecordNotFound) {
			return nil, reactionErr
		}
	}
	return card, nil
}

// upsertSceneContent 保存单个场景的 AI 摘要内容。
func (c *CommentAiCase) upsertSceneContent(ctx context.Context, goodsID int64, scene int32, content []llm.CommentAiContentItem) error {
	contentList := make([]*commonv1.CommentAiContentItem, 0, len(content))
	for _, item := range content {
		// 摘要内容为空时不进入最终展示内容。
		if item.Content == "" {
			continue
		}
		contentList = append(contentList, &commonv1.CommentAiContentItem{
			Label:   item.Label,
			Content: item.Content,
		})
	}
	// 当前场景没有有效摘要内容时，不覆盖旧摘要。
	if len(contentList) == 0 {
		return nil
	}

	contentJSON := _string.ConvertAnyToJsonString(contentList)
	query := c.Query(ctx).CommentAi
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

	return c.Create(ctx, &models.CommentAi{
		GoodsID: goodsID,
		Scene:   scene,
		Content: contentJSON,
	})
}
