package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// CommentAiCase 评论 AI 摘要业务实例。
type CommentAiCase struct {
	*biz.BaseCase
	*data.CommentAiRepository
	commentAiMapper *mapper.CopierMapper[adminv1.CommentAi, models.CommentAi]
}

// NewCommentAiCase 创建评论 AI 摘要业务实例。
func NewCommentAiCase(baseCase *biz.BaseCase, commentAiRepo *data.CommentAiRepository) *CommentAiCase {
	commentAiMapper := mapper.NewCopierMapper[adminv1.CommentAi, models.CommentAi]()
	commentAiMapper.AppendConverters(mapper.NewJSONTypeConverter[[]*commonv1.CommentAiContentItem]().NewConverterPair())
	return &CommentAiCase{
		BaseCase:            baseCase,
		CommentAiRepository: commentAiRepo,
		commentAiMapper:     commentAiMapper,
	}
}

// ListByGoodsID 查询商品评论 AI 摘要。
func (c *CommentAiCase) ListByGoodsID(ctx context.Context, goodsID int64) ([]*adminv1.CommentAi, error) {
	query := c.Query(ctx).CommentAi
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Order(query.Scene.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.CommentAi, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.commentAiMapper.ToDTO(item))
	}
	return resList, nil
}
