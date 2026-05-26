package task

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	_const "shop/pkg/const"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/queue"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/liujitcn/gorm-kit/repository"
)

const (
	// commentAuditRetryDefaultBatchSize 表示单次任务默认最多分别补偿投递的评价数和讨论数。
	commentAuditRetryDefaultBatchSize = 50
	// commentAuditRetryDefaultDelayMinutes 表示待审核数据或最近一次异常审核记录至少沉淀多久后才允许补偿，避免和实时审核队列重复抢同一条数据。
	commentAuditRetryDefaultDelayMinutes = 3
	// commentAuditRetryDefaultMaxExceptionCount 表示同一目标最多允许出现的 AI 审核异常次数，达到后停止自动补偿并交给人工处理。
	commentAuditRetryDefaultMaxExceptionCount = 3
	// commentAuditRetryDefaultLockLeaseMinutes 表示进程内执行锁默认租约时间，避免任务卡死后永久跳过后续调度。
	commentAuditRetryDefaultLockLeaseMinutes = 30
	// commentAuditRetryScanFactor 表示扫描候选数据时相对最终补偿批次的放大倍数，用于过滤掉已达到重试上限的数据后仍尽量凑满本批次。
	commentAuditRetryScanFactor = 10
	// commentAuditRetryMinScanLimit 表示单次任务最小候选扫描量，避免小批次配置下因为历史异常记录过多导致补偿吞吐过低。
	commentAuditRetryMinScanLimit = 200
)

// commentAuditReviewState 表示单个目标的 AI 审核记录摘要。
type commentAuditReviewState struct {
	latestStatus   int32
	latestAt       time.Time
	hasLatest      bool
	exceptionCount int
}

// commentAuditRetryTargets 表示单类审核目标的本轮补偿筛选结果。
type commentAuditRetryTargets struct {
	queryCount int
	ids        []int64
}

// commentAuditDispatchStats 表示单类审核目标的队列投递结果。
type commentAuditDispatchStats struct {
	successCount int
	failedCount  int
}

// CommentAuditRetry 评价与讨论自动审核补偿任务。
type CommentAuditRetry struct {
	commentInfoRepo       *data.CommentInfoRepository
	commentDiscussionRepo *data.CommentDiscussionRepository
	commentReviewRepo     *data.CommentReviewRepository
	ctx                   context.Context
	mu                    sync.Mutex
	runningUntil          time.Time
}

// NewCommentAuditRetry 创建评价与讨论自动审核补偿任务实例。
func NewCommentAuditRetry(
	commentInfoRepo *data.CommentInfoRepository,
	commentDiscussionRepo *data.CommentDiscussionRepository,
	commentReviewRepo *data.CommentReviewRepository,
) *CommentAuditRetry {
	return &CommentAuditRetry{
		commentInfoRepo:       commentInfoRepo,
		commentDiscussionRepo: commentDiscussionRepo,
		commentReviewRepo:     commentReviewRepo,
		ctx:                   context.Background(),
	}
}

// Exec 执行评价与讨论自动审核补偿。
func (t *CommentAuditRetry) Exec(args map[string]string) ([]string, error) {
	log.Infof("Job CommentAuditRetry Exec %+v", args)

	lockLeaseMinutes, err := parsePositiveJobInt(args, "lockLeaseMinutes", commentAuditRetryDefaultLockLeaseMinutes)
	if err != nil {
		return []string{err.Error()}, err
	}
	unlock, skippedMessage := t.tryLock(time.Duration(lockLeaseMinutes) * time.Minute)
	if skippedMessage != "" {
		return []string{skippedMessage}, nil
	}
	defer unlock()

	batchSize, err := parsePositiveJobInt(args, "batchSize", commentAuditRetryDefaultBatchSize)
	if err != nil {
		return []string{err.Error()}, err
	}
	var retryDelayMinutes int
	retryDelayMinutes, err = parsePositiveJobInt(args, "retryDelayMinutes", commentAuditRetryDefaultDelayMinutes)
	if err != nil {
		return []string{err.Error()}, err
	}
	var maxRetry int
	maxRetry, err = parsePositiveJobInt(args, "maxRetry", commentAuditRetryDefaultMaxExceptionCount)
	if err != nil {
		return []string{err.Error()}, err
	}

	cutoff := time.Now().Add(-time.Duration(retryDelayMinutes) * time.Minute)
	var commentTargets commentAuditRetryTargets
	commentTargets, err = t.retryCommentTargets(cutoff, batchSize, maxRetry)
	if err != nil {
		return []string{err.Error()}, err
	}
	var discussionTargets commentAuditRetryTargets
	discussionTargets, err = t.retryDiscussionTargets(cutoff, batchSize, maxRetry)
	if err != nil {
		return []string{err.Error()}, err
	}

	commentDispatchStats := dispatchCommentAuditTargets(_const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, commentTargets.ids)
	discussionDispatchStats := dispatchCommentAuditTargets(_const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, discussionTargets.ids)

	message := fmt.Sprintf(
		"评价审核补偿投递完成: 评价本轮查询 %d 条，投递成功 %d 条，投递失败 %d 条，未投递 %d 条；讨论本轮查询 %d 条，投递成功 %d 条，投递失败 %d 条，未投递 %d 条",
		commentTargets.queryCount,
		commentDispatchStats.successCount,
		commentDispatchStats.failedCount,
		commentTargets.queryCount-len(commentTargets.ids),
		discussionTargets.queryCount,
		discussionDispatchStats.successCount,
		discussionDispatchStats.failedCount,
		discussionTargets.queryCount-len(discussionTargets.ids),
	)
	return []string{message}, nil
}

// dispatchCommentAuditTargets 投递指定类型的审核目标并统计投递结果。
func dispatchCommentAuditTargets(targetType int32, targetIDs []int64) commentAuditDispatchStats {
	stats := commentAuditDispatchStats{}
	for _, targetID := range targetIDs {
		if queue.DispatchCommentAudit(targetType, targetID) {
			stats.successCount++
			continue
		}
		stats.failedCount++
	}
	return stats
}

// parsePositiveJobInt 解析正整数任务参数。
func parsePositiveJobInt(args map[string]string, key string, defaultValue int) (int, error) {
	value := args[key]
	if value == "" {
		return defaultValue, nil
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil || parsedValue <= 0 {
		return 0, errorsx.InvalidArgument(fmt.Sprintf("%s 必须是正整数", key))
	}
	return parsedValue, nil
}

// tryLock 尝试获取当前进程内的任务执行租约。
func (t *CommentAuditRetry) tryLock(lease time.Duration) (func(), string) {
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	if now.Before(t.runningUntil) {
		return func() {}, fmt.Sprintf("评价审核补偿仍在执行中，跳过本轮调度，锁预计释放时间 %s", t.runningUntil.Format(time.DateTime))
	}
	t.runningUntil = now.Add(lease)
	return func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.runningUntil = time.Time{}
	}, ""
}

// retryCommentTargets 查询需要重新投递自动审核的评价目标。
func (t *CommentAuditRetry) retryCommentTargets(cutoff time.Time, batchSize int, maxRetry int) (commentAuditRetryTargets, error) {
	query := t.commentInfoRepo.Query(t.ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	opts = append(opts, repository.Where(query.CreatedAt.Lte(cutoff)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc()))
	opts = append(opts, repository.Limit(commentAuditRetryScanLimit(batchSize)))
	list, err := t.commentInfoRepo.List(t.ctx, opts...)
	if err != nil {
		return commentAuditRetryTargets{}, err
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	var reviewMap map[int64]commentAuditReviewState
	reviewMap, err = t.loadReviewState(_const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, ids)
	if err != nil {
		return commentAuditRetryTargets{}, err
	}
	return commentAuditRetryTargets{
		queryCount: len(list),
		ids:        filterRetryTargetIDs(ids, reviewMap, cutoff, batchSize, maxRetry),
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

// loadReviewState 查询目标最近一次 AI 审核结果与异常次数。
func (t *CommentAuditRetry) loadReviewState(targetType int32, targetIDs []int64) (map[int64]commentAuditReviewState, error) {
	result := make(map[int64]commentAuditReviewState, len(targetIDs))
	if len(targetIDs) == 0 {
		return result, nil
	}

	query := t.commentReviewRepo.Query(t.ctx).CommentReview
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.TargetType.Eq(targetType)))
	opts = append(opts, repository.Where(query.TargetID.In(targetIDs...)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.COMMENT_REVIEW_TYPE_AI)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := t.commentReviewRepo.List(t.ctx, opts...)
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

// filterRetryTargetIDs 按审核记录状态筛选可重新投递的目标。
func filterRetryTargetIDs(targetIDs []int64, reviewMap map[int64]commentAuditReviewState, cutoff time.Time, batchSize int, maxRetry int) []int64 {
	result := make([]int64, 0, batchSize)
	for _, targetID := range targetIDs {
		state := reviewMap[targetID]
		// 异常次数达到上限后停止自动补偿，避免配置错误或外部资源永久不可用时无限重试。
		if state.exceptionCount >= maxRetry {
			continue
		}
		// 从未写入 AI 审核记录，或最近一次 AI 审核异常且已过冷却时间，才重新投递队列。
		if !state.hasLatest || (state.latestStatus == _const.COMMENT_REVIEW_STATUS_EXCEPTION && !state.latestAt.After(cutoff)) {
			result = append(result, targetID)
		}
		if len(result) >= batchSize {
			break
		}
	}
	return result
}

// retryDiscussionTargets 查询需要重新投递自动审核的讨论目标。
func (t *CommentAuditRetry) retryDiscussionTargets(cutoff time.Time, batchSize int, maxRetry int) (commentAuditRetryTargets, error) {
	query := t.commentDiscussionRepo.Query(t.ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	opts = append(opts, repository.Where(query.CreatedAt.Lte(cutoff)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc()))
	opts = append(opts, repository.Limit(commentAuditRetryScanLimit(batchSize)))
	list, err := t.commentDiscussionRepo.List(t.ctx, opts...)
	if err != nil {
		return commentAuditRetryTargets{}, err
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	var reviewMap map[int64]commentAuditReviewState
	reviewMap, err = t.loadReviewState(_const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, ids)
	if err != nil {
		return commentAuditRetryTargets{}, err
	}
	return commentAuditRetryTargets{
		queryCount: len(list),
		ids:        filterRetryTargetIDs(ids, reviewMap, cutoff, batchSize, maxRetry),
	}, nil
}
