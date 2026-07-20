package biz

import (
	"context"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// CommentSummaryCase 评论摘要业务实例。
type CommentSummaryCase struct {
	*biz.BaseCase
	*data.CommentSummaryRepository
	commentSummaryMapper *mapper.CopierMapper[shopadminv1.CommentSummary, models.CommentSummary]
}

// NewCommentSummaryCase 创建评论摘要业务实例。
func NewCommentSummaryCase(baseCase *biz.BaseCase, commentSummaryRepo *data.CommentSummaryRepository) *CommentSummaryCase {
	commentSummaryMapper := mapper.NewCopierMapper[shopadminv1.CommentSummary, models.CommentSummary]()
	commentSummaryMapper.AppendConverters(mapper.NewJSONTypeConverter[[]*shopcommonv1.CommentSummaryContentItem]().NewConverterPair())
	return &CommentSummaryCase{
		BaseCase:                 baseCase,
		CommentSummaryRepository: commentSummaryRepo,
		commentSummaryMapper:     commentSummaryMapper,
	}
}

// ListByGoodsID 查询商品评论摘要。
func (c *CommentSummaryCase) ListByGoodsID(ctx context.Context, goodsID int64) ([]*shopadminv1.CommentSummary, error) {
	query := c.Query(ctx).CommentSummary
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Order(query.Scene.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*shopadminv1.CommentSummary, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.commentSummaryMapper.ToDTO(item))
	}
	return resList, nil
}
