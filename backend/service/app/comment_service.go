package app

import (
	"context"

	appv1 "shop/api/gen/go/app/v1"
	"shop/pkg/errorsx"
	"shop/service/app/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CommentService 评价服务。
type CommentService struct {
	appv1.UnimplementedCommentServiceServer
	commentCase *biz.CommentCase
}

// NewCommentService 创建评价服务。
func NewCommentService(commentCase *biz.CommentCase) *CommentService {
	var ss = CommentService{
		commentCase: commentCase,
	}
	return &ss
}

// GoodsCommentOverview 查询商品评价摘要。
func (s *CommentService) GoodsCommentOverview(ctx context.Context, req *appv1.GoodsCommentOverviewRequest) (*appv1.GoodsCommentOverviewResponse, error) {
	res, err := s.commentCase.GoodsCommentOverview(ctx, req)
	if err != nil {
		log.Errorf("GoodsCommentOverview %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品评价摘要失败")
	}
	return res, nil
}

// PageGoodsComment 查询商品评价分页列表。
func (s *CommentService) PageGoodsComment(ctx context.Context, req *appv1.PageGoodsCommentRequest) (*appv1.PageGoodsCommentResponse, error) {
	res, err := s.commentCase.PageGoodsComment(ctx, req)
	if err != nil {
		log.Errorf("PageGoodsComment %v", err)
		return nil, errorsx.WrapInternal(err, "查询商品评价列表失败")
	}
	return res, nil
}

// PageCommentDiscussion 查询评价讨论分页列表。
func (s *CommentService) PageCommentDiscussion(ctx context.Context, req *appv1.PageCommentDiscussionRequest) (*appv1.PageCommentDiscussionResponse, error) {
	res, err := s.commentCase.PageCommentDiscussion(ctx, req)
	if err != nil {
		log.Errorf("PageCommentDiscussion %v", err)
		return nil, errorsx.WrapInternal(err, "查询评价讨论失败")
	}
	return res, nil
}

// CreateCommentDiscussion 发布评价讨论。
func (s *CommentService) CreateCommentDiscussion(ctx context.Context, req *appv1.CreateCommentDiscussionRequest) (*appv1.CreateCommentDiscussionResponse, error) {
	res, err := s.commentCase.CreateCommentDiscussion(ctx, req)
	if err != nil {
		log.Errorf("CreateCommentDiscussion %v", err)
		return nil, errorsx.WrapInternal(err, "发布评价讨论失败")
	}
	return res, nil
}

// SaveCommentReaction 保存评价互动状态。
func (s *CommentService) SaveCommentReaction(ctx context.Context, req *appv1.SaveCommentReactionRequest) (*appv1.SaveCommentReactionResponse, error) {
	res, err := s.commentCase.SaveCommentReaction(ctx, req)
	if err != nil {
		log.Errorf("SaveCommentReaction %v", err)
		return nil, errorsx.WrapInternal(err, "保存评价互动状态失败")
	}
	return res, nil
}

// PagePendingCommentGoods 查询待评价商品分页列表。
func (s *CommentService) PagePendingCommentGoods(ctx context.Context, req *appv1.PagePendingCommentGoodsRequest) (*appv1.PagePendingCommentGoodsResponse, error) {
	res, err := s.commentCase.PagePendingCommentGoods(ctx, req)
	if err != nil {
		log.Errorf("PagePendingCommentGoods %v", err)
		return nil, errorsx.WrapInternal(err, "查询待评价商品失败")
	}
	return res, nil
}

// CreateComment 发布商品评价。
func (s *CommentService) CreateComment(ctx context.Context, req *appv1.CreateCommentRequest) (*appv1.CreateCommentResponse, error) {
	res, err := s.commentCase.CreateComment(ctx, req)
	if err != nil {
		log.Errorf("CreateComment %v", err)
		return nil, errorsx.WrapInternal(err, "发布商品评价失败")
	}
	return res, nil
}

// DeleteComment 删除商品评价。
func (s *CommentService) DeleteComment(ctx context.Context, req *appv1.DeleteCommentRequest) (*emptypb.Empty, error) {
	err := s.commentCase.DeleteComment(ctx, req.GetId())
	if err != nil {
		log.Errorf("DeleteComment %v", err)
		return nil, errorsx.WrapInternal(err, "删除商品评价失败")
	}
	return &emptypb.Empty{}, nil
}

// PageMyComment 查询我的评价分页列表。
func (s *CommentService) PageMyComment(ctx context.Context, req *appv1.PageMyCommentRequest) (*appv1.PageMyCommentResponse, error) {
	res, err := s.commentCase.PageMyComment(ctx, req)
	if err != nil {
		log.Errorf("PageMyComment %v", err)
		return nil, errorsx.WrapInternal(err, "查询我的评价失败")
	}
	return res, nil
}
