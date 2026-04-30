package admin

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/errorsx"
	"shop/service/admin/biz"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// CommentInfoService Admin评论管理服务。
type CommentInfoService struct {
	adminv1.UnimplementedCommentInfoServiceServer
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

// PageCommentInfos 查询评论分页列表。
func (s *CommentInfoService) PageCommentInfos(ctx context.Context, req *adminv1.PageCommentInfosRequest) (*adminv1.PageCommentInfosResponse, error) {
	page, err := s.commentInfoCase.PageCommentInfos(ctx, req)
	if err != nil {
		log.Errorf("PageCommentInfos %v", err)
		return nil, errorsx.WrapInternal(err, "查询评论分页列表失败")
	}
	return page, nil
}

// GetGoodsCommentInfo 按商品查询评论聚合信息。
func (s *CommentInfoService) GetGoodsCommentInfo(ctx context.Context, req *adminv1.GetGoodsCommentInfoRequest) (*adminv1.GoodsCommentInfoResponse, error) {
	res, err := s.commentInfoCase.GetGoodsCommentInfo(ctx, req.GetGoodsId())
	if err != nil {
		log.Errorf("GetGoodsCommentInfo %v", err)
		return nil, errorsx.WrapInternal(err, "按商品查询评论聚合信息失败")
	}
	return res, nil
}

// GetCommentInfo 查询评论详情。
func (s *CommentInfoService) GetCommentInfo(ctx context.Context, req *adminv1.GetCommentInfoRequest) (*adminv1.CommentInfoDetail, error) {
	res, err := s.commentInfoCase.GetCommentInfo(ctx, req.GetId())
	if err != nil {
		log.Errorf("GetCommentInfo %v", err)
		return nil, errorsx.WrapInternal(err, "查询评论详情失败")
	}
	return res, nil
}

// SetCommentInfoStatus 设置评论审核状态。
func (s *CommentInfoService) SetCommentInfoStatus(ctx context.Context, req *adminv1.SetCommentInfoStatusRequest) (*emptypb.Empty, error) {
	err := s.commentInfoCase.SetCommentInfoStatus(ctx, req)
	if err != nil {
		log.Errorf("SetCommentInfoStatus %v", err)
		return nil, errorsx.WrapInternal(err, "设置评论审核状态失败")
	}
	return new(emptypb.Empty), nil
}

// PageCommentDiscussions 查询评论讨论分页列表。
func (s *CommentInfoService) PageCommentDiscussions(ctx context.Context, req *adminv1.PageCommentDiscussionsRequest) (*adminv1.PageCommentDiscussionsResponse, error) {
	page, err := s.commentDiscussionCase.PageCommentDiscussions(ctx, req)
	if err != nil {
		log.Errorf("PageCommentDiscussions %v", err)
		return nil, errorsx.WrapInternal(err, "查询评论讨论分页列表失败")
	}
	return page, nil
}

// SetCommentDiscussionStatus 设置评论讨论审核状态。
func (s *CommentInfoService) SetCommentDiscussionStatus(ctx context.Context, req *adminv1.SetCommentDiscussionStatusRequest) (*emptypb.Empty, error) {
	err := s.commentDiscussionCase.SetCommentDiscussionStatus(ctx, req)
	if err != nil {
		log.Errorf("SetCommentDiscussionStatus %v", err)
		return nil, errorsx.WrapInternal(err, "设置评论讨论审核状态失败")
	}
	return new(emptypb.Empty), nil
}

// ListCommentReviews 查询评论审核记录列表。
func (s *CommentInfoService) ListCommentReviews(ctx context.Context, req *adminv1.ListCommentReviewsRequest) (*adminv1.ListCommentReviewsResponse, error) {
	list, err := s.commentReviewCase.ListByTarget(ctx, int32(req.GetTargetType()), req.GetTargetId())
	if err != nil {
		log.Errorf("ListCommentReviews %v", err)
		return nil, errorsx.WrapInternal(err, "查询评论审核记录列表失败")
	}
	return &adminv1.ListCommentReviewsResponse{CommentReviews: list}, nil
}
