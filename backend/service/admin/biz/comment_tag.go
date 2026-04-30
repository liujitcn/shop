package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// CommentTagCase 评论标签业务实例。
type CommentTagCase struct {
	*biz.BaseCase
	*data.CommentTagRepository
	commentTagMapper *mapper.CopierMapper[adminv1.CommentTag, models.CommentTag]
}

// NewCommentTagCase 创建评论标签业务实例。
func NewCommentTagCase(baseCase *biz.BaseCase, commentTagRepo *data.CommentTagRepository) *CommentTagCase {
	return &CommentTagCase{
		BaseCase:             baseCase,
		CommentTagRepository: commentTagRepo,
		commentTagMapper:     mapper.NewCopierMapper[adminv1.CommentTag, models.CommentTag](),
	}
}

// ListByGoodsID 查询商品评论标签。
func (c *CommentTagCase) ListByGoodsID(ctx context.Context, goodsID int64) ([]*adminv1.CommentTag, error) {
	query := c.Query(ctx).CommentTag
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.MentionCount.Desc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.CommentTag, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.commentTagMapper.ToDTO(item))
	}
	return resList, nil
}

// DecreaseMentionCount 批量减少标签提及次数。
func (c *CommentTagCase) DecreaseMentionCount(ctx context.Context, tagIDs []int64) error {
	// 没有命中的标签编号时，无需执行计数更新。
	if len(tagIDs) == 0 {
		return nil
	}

	query := c.Query(ctx).CommentTag
	// 仅对大于 0 的记录执行原子递减，避免提及次数出现负数。
	_, err := query.WithContext(ctx).
		Where(
			query.ID.In(tagIDs...),
			query.MentionCount.Gt(0),
		).
		UpdateSimple(query.MentionCount.Sub(1))
	return err
}
