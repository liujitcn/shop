package app

import (
	"context"
	"fmt"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/app/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CommentInfoService 提供评价服务。
type CommentInfoService struct {
	shopappv1.UnimplementedCommentInfoServiceServer
	commentCase *biz.CommentCase
}

// NewCommentInfoService 创建评价服务。
func NewCommentInfoService(commentCase *biz.CommentCase) *CommentInfoService {
	var ss = CommentInfoService{
		commentCase: commentCase,
	}
	return &ss
}

// PageCommentDiscussion 查询评价讨论分页列表。
func (s *CommentInfoService) PageCommentDiscussion(ctx context.Context, req *shopappv1.PageCommentDiscussionRequest) (*shopappv1.PageCommentDiscussionResponse, error) {
	res, err := s.commentCase.PageCommentDiscussion(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageCommentDiscussion %v", err))
		return nil, errorsx.WrapInternal(err, "查询评价讨论失败")
	}
	return res, nil
}

// PageGoodsComment 查询商品评价分页列表。
func (s *CommentInfoService) PageGoodsComment(ctx context.Context, req *shopappv1.PageGoodsCommentRequest) (*shopappv1.PageGoodsCommentResponse, error) {
	res, err := s.commentCase.PageGoodsComment(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageGoodsComment %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品评价列表失败")
	}
	return res, nil
}

// PageMyComment 查询我的评价分页列表。
func (s *CommentInfoService) PageMyComment(ctx context.Context, req *shopappv1.PageMyCommentRequest) (*shopappv1.PageMyCommentResponse, error) {
	res, err := s.commentCase.PageMyComment(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageMyComment %v", err))
		return nil, errorsx.WrapInternal(err, "查询我的评价失败")
	}
	return res, nil
}

// PagePendingCommentGoods 查询待评价商品分页列表。
func (s *CommentInfoService) PagePendingCommentGoods(ctx context.Context, req *shopappv1.PagePendingCommentGoodsRequest) (*shopappv1.PagePendingCommentGoodsResponse, error) {
	res, err := s.commentCase.PagePendingCommentGoods(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PagePendingCommentGoods %v", err))
		return nil, errorsx.WrapInternal(err, "查询待评价商品失败")
	}
	return res, nil
}

// CreateComment 发布商品评价。
func (s *CommentInfoService) CreateComment(ctx context.Context, req *shopappv1.CreateCommentRequest) (*shopappv1.CreateCommentResponse, error) {
	res, err := s.commentCase.CreateComment(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateComment %v", err))
		return nil, errorsx.WrapInternal(err, "发布商品评价失败")
	}
	return res, nil
}

// CreateCommentDiscussion 发布评价讨论。
func (s *CommentInfoService) CreateCommentDiscussion(ctx context.Context, req *shopappv1.CreateCommentDiscussionRequest) (*shopappv1.CreateCommentDiscussionResponse, error) {
	res, err := s.commentCase.CreateCommentDiscussion(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("CreateCommentDiscussion %v", err))
		return nil, errorsx.WrapInternal(err, "发布评价讨论失败")
	}
	return res, nil
}

// DeleteComment 删除商品评价。
func (s *CommentInfoService) DeleteComment(ctx context.Context, req *shopappv1.DeleteCommentRequest) (*emptypb.Empty, error) {
	err := s.commentCase.DeleteComment(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteComment %v", err))
		return nil, errorsx.WrapInternal(err, "删除商品评价失败")
	}
	return &emptypb.Empty{}, nil
}

// GoodsCommentOverview 查询商品评价摘要。
func (s *CommentInfoService) GoodsCommentOverview(ctx context.Context, req *shopappv1.GoodsCommentOverviewRequest) (*shopappv1.GoodsCommentOverviewResponse, error) {
	res, err := s.commentCase.GoodsCommentOverview(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GoodsCommentOverview %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品评价摘要失败")
	}
	return res, nil
}

// GoodsCommentTag 查询商品评价标签列表。
func (s *CommentInfoService) GoodsCommentTag(ctx context.Context, req *shopappv1.GoodsCommentTagRequest) (*shopappv1.GoodsCommentTagResponse, error) {
	res, err := s.commentCase.GoodsCommentTag(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GoodsCommentTag %v", err))
		return nil, errorsx.WrapInternal(err, "查询商品评价标签失败")
	}
	return res, nil
}

// SaveCommentReaction 保存评价互动状态。
func (s *CommentInfoService) SaveCommentReaction(ctx context.Context, req *shopappv1.SaveCommentReactionRequest) (*shopappv1.SaveCommentReactionResponse, error) {
	res, err := s.commentCase.SaveCommentReaction(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveCommentReaction %v", err))
		return nil, errorsx.WrapInternal(err, "保存评价互动状态失败")
	}
	return res, nil
}
