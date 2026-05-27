package biz

import (
	"context"
	"fmt"
	"strings"
	"time"

	_const "shop/pkg/const"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repository"
)

const (
	// commentAuditRetryScanFactor 表示扫描候选数据时相对最终补偿批次的放大倍数，用于过滤掉已达到重试上限的数据后仍尽量凑满本批次。
	commentAuditRetryScanFactor = 10
	// commentAuditRetryMinScanLimit 表示单次任务最小候选扫描量，避免小批次配置下因为历史异常记录过多导致补偿吞吐过低。
	commentAuditRetryMinScanLimit = 200
	// commentAuditRetryFailureDetailLimit 表示任务日志中最多保留的单目标失败明细数量。
	commentAuditRetryFailureDetailLimit = 10
)

// commentAuditReviewState 表示单个目标的 AI 审核记录摘要。
type commentAuditReviewState struct {
	latestStatus   int32
	latestAt       time.Time
	hasLatest      bool
	exceptionCount int
}

// commentAuditRetryCommentTargets 表示评价审核目标的本轮补偿筛选结果。
type commentAuditRetryCommentTargets struct {
	queryCount int
	records    []*models.CommentInfo
}

// commentAuditRetryDiscussionTargets 表示讨论审核目标的本轮补偿筛选结果。
type commentAuditRetryDiscussionTargets struct {
	queryCount int
	records    []*models.CommentDiscussion
}

// commentAuditRunStats 表示单类审核目标的同步执行结果。
type commentAuditRunStats struct {
	successCount   int
	failedCount    int
	failureDetails []string
}

// RetryCommentAudit 同步补偿执行评价与讨论 AI 审核。
func (c *CommentCase) RetryCommentAudit(ctx context.Context, batchSize int, retryDelayMinutes int, maxRetry int) ([]string, error) {
	cutoff := time.Now().Add(-time.Duration(retryDelayMinutes) * time.Minute)
	commentTargets, err := c.retryCommentAuditTargets(ctx, cutoff, batchSize, maxRetry)
	if err != nil {
		return []string{err.Error()}, err
	}
	var discussionTargets commentAuditRetryDiscussionTargets
	discussionTargets, err = c.retryDiscussionAuditTargets(ctx, cutoff, batchSize, maxRetry)
	if err != nil {
		return []string{err.Error()}, err
	}

	commentRunStats := c.auditCommentTargets(ctx, commentTargets.records)
	discussionRunStats := c.auditDiscussionTargets(ctx, discussionTargets.records)

	message := fmt.Sprintf(
		"评价审核补偿执行完成: 评价本轮查询 %d 条，执行 %d 条，成功 %d 条，失败 %d 条，跳过 %d 条；讨论本轮查询 %d 条，执行 %d 条，成功 %d 条，失败 %d 条，跳过 %d 条",
		commentTargets.queryCount,
		len(commentTargets.records),
		commentRunStats.successCount,
		commentRunStats.failedCount,
		commentTargets.queryCount-len(commentTargets.records),
		discussionTargets.queryCount,
		len(discussionTargets.records),
		discussionRunStats.successCount,
		discussionRunStats.failedCount,
		discussionTargets.queryCount-len(discussionTargets.records),
	)
	output := []string{message}
	failureDetails := append(commentRunStats.failureDetails, discussionRunStats.failureDetails...)
	if len(failureDetails) > 0 {
		output = append(output, "失败明细: "+strings.Join(failureDetails, "；"))
	}
	return output, nil
}

// retryCommentAuditTargets 查询需要同步补偿自动审核的评价目标。
func (c *CommentCase) retryCommentAuditTargets(ctx context.Context, cutoff time.Time, batchSize int, maxRetry int) (commentAuditRetryCommentTargets, error) {
	query := c.commentInfoCase.Query(ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	opts = append(opts, repository.Where(query.CreatedAt.Lte(cutoff)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc()))
	opts = append(opts, repository.Limit(commentAuditRetryScanLimit(batchSize)))
	list, err := c.commentInfoCase.List(ctx, opts...)
	if err != nil {
		return commentAuditRetryCommentTargets{}, err
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	var reviewMap map[int64]commentAuditReviewState
	reviewMap, err = c.loadAuditReviewState(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, ids)
	if err != nil {
		return commentAuditRetryCommentTargets{}, err
	}
	return commentAuditRetryCommentTargets{
		queryCount: len(list),
		records:    filterRetryCommentRecords(list, reviewMap, cutoff, batchSize, maxRetry),
	}, nil
}

// retryDiscussionAuditTargets 查询需要同步补偿自动审核的讨论目标。
func (c *CommentCase) retryDiscussionAuditTargets(ctx context.Context, cutoff time.Time, batchSize int, maxRetry int) (commentAuditRetryDiscussionTargets, error) {
	query := c.commentDiscussionCase.Query(ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	opts = append(opts, repository.Where(query.CreatedAt.Lte(cutoff)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc()))
	opts = append(opts, repository.Limit(commentAuditRetryScanLimit(batchSize)))
	list, err := c.commentDiscussionCase.List(ctx, opts...)
	if err != nil {
		return commentAuditRetryDiscussionTargets{}, err
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	var reviewMap map[int64]commentAuditReviewState
	reviewMap, err = c.loadAuditReviewState(ctx, _const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, ids)
	if err != nil {
		return commentAuditRetryDiscussionTargets{}, err
	}
	return commentAuditRetryDiscussionTargets{
		queryCount: len(list),
		records:    filterRetryDiscussionRecords(list, reviewMap, cutoff, batchSize, maxRetry),
	}, nil
}

// commentAuditRetryScanLimit 返回单次补偿扫描上限。
func commentAuditRetryScanLimit(batchSize int) int {
	scanLimit := batchSize * commentAuditRetryScanFactor
	if scanLimit < commentAuditRetryMinScanLimit {
		return commentAuditRetryMinScanLimit
	}
	return scanLimit
}

// loadAuditReviewState 查询目标最近一次 AI 审核结果与异常次数。
func (c *CommentCase) loadAuditReviewState(ctx context.Context, targetType int32, targetIDs []int64) (map[int64]commentAuditReviewState, error) {
	result := make(map[int64]commentAuditReviewState, len(targetIDs))
	if len(targetIDs) == 0 {
		return result, nil
	}

	query := c.commentReviewCase.Query(ctx).CommentReview
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.TargetType.Eq(targetType)))
	opts = append(opts, repository.Where(query.TargetID.In(targetIDs...)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.COMMENT_REVIEW_TYPE_AI)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := c.commentReviewCase.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	for _, item := range list {
		state := result[item.TargetID]
		// 审核记录按创建时间倒序返回，首次命中的就是该目标最近一次 AI 审核结果。
		if !state.hasLatest {
			state.latestStatus = item.Status
			state.latestAt = item.CreatedAt
			state.hasLatest = true
		}
		if item.Status == _const.COMMENT_REVIEW_STATUS_EXCEPTION {
			state.exceptionCount++
		}
		result[item.TargetID] = state
	}
	return result, nil
}

// filterRetryCommentRecords 按审核记录状态筛选可重新审核的评价目标。
func filterRetryCommentRecords(records []*models.CommentInfo, reviewMap map[int64]commentAuditReviewState, cutoff time.Time, batchSize int, maxRetry int) []*models.CommentInfo {
	result := make([]*models.CommentInfo, 0, batchSize)
	for _, record := range records {
		if !shouldRetryAuditTarget(record.ID, reviewMap, cutoff, maxRetry) {
			continue
		}
		result = append(result, record)
		if len(result) >= batchSize {
			break
		}
	}
	return result
}

// filterRetryDiscussionRecords 按审核记录状态筛选可重新审核的讨论目标。
func filterRetryDiscussionRecords(records []*models.CommentDiscussion, reviewMap map[int64]commentAuditReviewState, cutoff time.Time, batchSize int, maxRetry int) []*models.CommentDiscussion {
	result := make([]*models.CommentDiscussion, 0, batchSize)
	for _, record := range records {
		if !shouldRetryAuditTarget(record.ID, reviewMap, cutoff, maxRetry) {
			continue
		}
		result = append(result, record)
		if len(result) >= batchSize {
			break
		}
	}
	return result
}

// shouldRetryAuditTarget 判断单个目标是否仍允许自动补偿审核。
func shouldRetryAuditTarget(targetID int64, reviewMap map[int64]commentAuditReviewState, cutoff time.Time, maxRetry int) bool {
	state := reviewMap[targetID]
	// 异常次数达到上限后停止自动补偿，避免配置错误或外部资源永久不可用时无限重试。
	if state.exceptionCount >= maxRetry {
		return false
	}
	// 从未写入 AI 审核记录，或最近一次 AI 审核异常且已过冷却时间，才同步执行补偿审核。
	return !state.hasLatest || (state.latestStatus == _const.COMMENT_REVIEW_STATUS_EXCEPTION && !state.latestAt.After(cutoff))
}

// auditCommentTargets 同步审核本轮筛选出的评价目标。
func (c *CommentCase) auditCommentTargets(ctx context.Context, records []*models.CommentInfo) commentAuditRunStats {
	stats := commentAuditRunStats{}
	for _, record := range records {
		err := c.AuditComment(ctx, record)
		if err == nil {
			stats.successCount++
			continue
		}
		stats.failedCount++
		stats.addFailureDetail("评价", record.ID, err)
		log.Errorf("comment audit retry comment failed, commentID=%d err=%v", record.ID, err)
	}
	return stats
}

// auditDiscussionTargets 同步审核本轮筛选出的讨论目标。
func (c *CommentCase) auditDiscussionTargets(ctx context.Context, records []*models.CommentDiscussion) commentAuditRunStats {
	stats := commentAuditRunStats{}
	for _, record := range records {
		err := c.AuditDiscussion(ctx, record)
		if err == nil {
			stats.successCount++
			continue
		}
		stats.failedCount++
		stats.addFailureDetail("讨论", record.ID, err)
		log.Errorf("comment audit retry discussion failed, discussionID=%d err=%v", record.ID, err)
	}
	return stats
}

// addFailureDetail 追加单目标失败摘要，避免任务日志输出过长。
func (s *commentAuditRunStats) addFailureDetail(targetName string, targetID int64, err error) {
	if len(s.failureDetails) >= commentAuditRetryFailureDetailLimit {
		return
	}
	s.failureDetails = append(s.failureDetails, fmt.Sprintf("%s %d: %v", targetName, targetID, err))
}
