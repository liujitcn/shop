package admin

import (
	"context"
	"fmt"

	shopadminv1 "shop/api/gen/go/shop/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/shop/admin/biz"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CommentInfoService Admin评论管理服务。
type CommentInfoService struct {
	shopadminv1.UnimplementedCommentInfoServiceServer
	commentInfoCase       *biz.CommentInfoCase
	commentDiscussionCase *biz.CommentDiscussionCase
	commentReviewCase     *biz.CommentReviewCase
}

// NewCommentInfoService 创建Admin评论管理服务。
func NewCommentInfoService(
	commentInfoCase *biz.CommentInfoCase,
	commentDiscussionCase *biz.CommentDiscussionCase,
	commentReviewCase *biz.CommentReviewCase,
) *CommentInfoService {
	return &CommentInfoService{
		commentInfoCase:       commentInfoCase,
		commentDiscussionCase: commentDiscussionCase,
		commentReviewCase:     commentReviewCase,
	}
}

// PageCommentInfo 查询评论分页列表。
func (s *CommentInfoService) PageCommentInfo(ctx context.Context, req *shopadminv1.PageCommentInfoRequest) (*shopadminv1.PageCommentInfoResponse, error) {
	page, err := s.commentInfoCase.PageCommentInfo(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageCommentInfo %v", err))
		return nil, errorsx.WrapInternal(err, "查询评论分页列表失败")
	}
	return page, nil
}

// GetGoodsCommentInfo 按商品查询评论聚合信息。
func (s *CommentInfoService) GetGoodsCommentInfo(ctx context.Context, req *shopadminv1.GetGoodsCommentInfoRequest) (*shopadminv1.GoodsCommentInfoResponse, error) {
	res, err := s.commentInfoCase.GetGoodsCommentInfo(ctx, req.GetGoodsId())
	if err != nil {
		log.Error(fmt.Sprintf("GetGoodsCommentInfo %v", err))
		return nil, errorsx.WrapInternal(err, "按商品查询评论聚合信息失败")
	}
	return res, nil
}

// GetCommentInfo 查询评论详情。
func (s *CommentInfoService) GetCommentInfo(ctx context.Context, req *shopadminv1.GetCommentInfoRequest) (*shopadminv1.CommentInfoDetail, error) {
	res, err := s.commentInfoCase.GetCommentInfo(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetCommentInfo %v", err))
		return nil, errorsx.WrapInternal(err, "查询评论详情失败")
	}
	return res, nil
}

// SetCommentInfoStatus 设置评论审核状态。
func (s *CommentInfoService) SetCommentInfoStatus(ctx context.Context, req *shopadminv1.SetCommentInfoStatusRequest) (*emptypb.Empty, error) {
	err := s.commentInfoCase.SetCommentInfoStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetCommentInfoStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置评论审核状态失败")
	}
	return new(emptypb.Empty), nil
}

// PageCommentDiscussion 查询评论讨论分页列表。
func (s *CommentInfoService) PageCommentDiscussion(ctx context.Context, req *shopadminv1.PageCommentDiscussionRequest) (*shopadminv1.PageCommentDiscussionResponse, error) {
	page, err := s.commentDiscussionCase.PageCommentDiscussion(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageCommentDiscussion %v", err))
		return nil, errorsx.WrapInternal(err, "查询评论讨论分页列表失败")
	}
	return page, nil
}

// SetCommentDiscussionStatus 设置评论讨论审核状态。
func (s *CommentInfoService) SetCommentDiscussionStatus(ctx context.Context, req *shopadminv1.SetCommentDiscussionStatusRequest) (*emptypb.Empty, error) {
	err := s.commentDiscussionCase.SetCommentDiscussionStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetCommentDiscussionStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置评论讨论审核状态失败")
	}
	return new(emptypb.Empty), nil
}

// ListCommentReview 查询评论审核记录列表。
func (s *CommentInfoService) ListCommentReview(ctx context.Context, req *shopadminv1.ListCommentReviewRequest) (*shopadminv1.ListCommentReviewResponse, error) {
	list, err := s.commentReviewCase.ListByTarget(ctx, int32(req.GetTargetType()), req.GetTargetId())
	if err != nil {
		log.Error(fmt.Sprintf("ListCommentReview %v", err))
		return nil, errorsx.WrapInternal(err, "查询评论审核记录列表失败")
	}
	return &shopadminv1.ListCommentReviewResponse{CommentReviews: list}, nil
}
