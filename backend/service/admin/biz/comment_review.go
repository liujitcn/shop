package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// CommentReviewCase 评论统一审核记录业务实例。
type CommentReviewCase struct {
	*biz.BaseCase
	*data.CommentReviewRepository
	reviewMapper *mapper.CopierMapper[adminv1.CommentReview, models.CommentReview]
}

// NewCommentReviewCase 创建评论统一审核记录业务实例。
func NewCommentReviewCase(baseCase *biz.BaseCase, commentReviewRepo *data.CommentReviewRepository) *CommentReviewCase {
	reviewMapper := mapper.NewCopierMapper[adminv1.CommentReview, models.CommentReview]()
	reviewMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &CommentReviewCase{
		BaseCase:                baseCase,
		CommentReviewRepository: commentReviewRepo,
		reviewMapper:            reviewMapper,
	}
}

// CreateReview 创建评论或讨论审核记录。
func (c *CommentReviewCase) CreateReview(ctx context.Context, review *models.CommentReview) error {
	// 审核记录为空时，无需写入。
	if review == nil {
		return nil
	}
	if review.Tags == "" {
		review.Tags = _string.ConvertAnyToJsonString([]string{})
	}
	return c.Create(ctx, review)
}

// ListByTarget 查询指定评论或讨论的审核记录。
func (c *CommentReviewCase) ListByTarget(ctx context.Context, targetType int32, targetID int64) ([]*adminv1.CommentReview, error) {
	query := c.Query(ctx).CommentReview
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TargetType.Eq(targetType)))
	opts = append(opts, repository.Where(query.TargetID.Eq(targetID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	recordList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*adminv1.CommentReview, 0, len(recordList))
	for _, record := range recordList {
		list = append(list, c.reviewMapper.ToDTO(record))
	}
	return list, nil
}
