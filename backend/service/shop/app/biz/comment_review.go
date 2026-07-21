package biz

import (
	"context"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	_const "shop/service/shop/consts"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// CommentReviewCase 评价统一审核记录业务处理对象。
type CommentReviewCase struct {
	*biz.BaseCase
	*data.CommentReviewRepository
	reviewMapper *mapper.CopierMapper[shopadminv1.CommentReview, models.CommentReview]
}

// NewCommentReviewCase 创建评价统一审核记录业务处理对象。
func NewCommentReviewCase(baseCase *biz.BaseCase, commentReviewRepo *data.CommentReviewRepository) *CommentReviewCase {
	reviewMapper := mapper.NewCopierMapper[shopadminv1.CommentReview, models.CommentReview]()
	reviewMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &CommentReviewCase{
		BaseCase:                baseCase,
		CommentReviewRepository: commentReviewRepo,
		reviewMapper:            reviewMapper,
	}
}

// ListByTarget 查询指定评价或讨论的审核记录。
func (c *CommentReviewCase) ListByTarget(ctx context.Context, targetType int32, targetID int64) ([]*shopadminv1.CommentReview, error) {
	query := c.Query(ctx).CommentReview
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TargetType.Eq(targetType)))
	opts = append(opts, repository.Where(query.TargetID.Eq(targetID)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	recordList, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	list := make([]*shopadminv1.CommentReview, 0, len(recordList))
	for _, record := range recordList {
		list = append(list, c.reviewMapper.ToDTO(record))
	}
	return list, nil
}

// CreateReview 创建一条评价或讨论审核记录。
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

// createAIReview 创建 AI 审核记录。
func (c *CommentReviewCase) createAIReview(ctx context.Context, tenantID, tenantStoreID int64, targetType int32, targetID int64, status int32, tags []string, reason string, model string) error {
	operatorName := model
	// 模型名称为空时，使用统一名称区分 AI 审核来源。
	if operatorName == "" {
		operatorName = "LLM"
	}
	return c.CreateReview(ctx, &models.CommentReview{
		TenantID:      tenantID,
		TenantStoreID: tenantStoreID,
		TargetType:    targetType,
		TargetID:      targetID,
		Type:          _const.COMMENT_REVIEW_TYPE_AI,
		Status:        status,
		Tags:          _string.ConvertAnyToJsonString(cleanCommentTagNames(tags)),
		OperatorID:    0,
		OperatorName:  operatorName,
		Reason:        reason,
	})
}
