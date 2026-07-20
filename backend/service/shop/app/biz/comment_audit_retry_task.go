package biz

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	_const "shop/service/shop/consts"
	"shop/pkg/errorsx"
	"shop/pkg/gen/models"
	"shop/pkg/job"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
)

const (
	// commentAuditRetryDefaultBatchSize 表示单次任务默认最多分别补偿审核的评价数和讨论数。
	commentAuditRetryDefaultBatchSize = 50
	// commentAuditRetryDefaultDelayMinutes 表示待审核数据或最近一次异常审核记录至少沉淀多久后才允许补偿，避免和实时审核队列并发处理同一条数据。
	commentAuditRetryDefaultDelayMinutes = 3
	// commentAuditRetryDefaultMaxExceptionCount 表示同一目标最多允许出现的 AI 审核异常次数，达到后停止自动补偿并交给人工处理。
	commentAuditRetryDefaultMaxExceptionCount = 3
	// commentAuditRetryDefaultLockLeaseMinutes 表示进程内执行锁默认租约时间，避免任务卡死后永久跳过后续调度。
	commentAuditRetryDefaultLockLeaseMinutes = 30
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

// CommentAuditRetry 评价与讨论自动审核补偿任务。
type CommentAuditRetry struct {
	commentCase           *CommentCase
	commentInfoCase       *CommentInfoCase
	commentDiscussionCase *CommentDiscussionCase
	commentReviewCase     *CommentReviewCase
	ctx                   context.Context
	mu                    sync.Mutex
	runningUntil          time.Time
}

var _ job.TaskExec = (*CommentAuditRetry)(nil)

// NewCommentAuditRetry 创建评价与讨论自动审核补偿任务实例。
func NewCommentAuditRetry(
	commentCase *CommentCase,
	commentInfoCase *CommentInfoCase,
	commentDiscussionCase *CommentDiscussionCase,
	commentReviewCase *CommentReviewCase,
) *CommentAuditRetry {
	return &CommentAuditRetry{
		commentCase:           commentCase,
		commentInfoCase:       commentInfoCase,
		commentDiscussionCase: commentDiscussionCase,
		commentReviewCase:     commentReviewCase,
		ctx:                   context.Background(),
	}
}

// Exec 执行评价与讨论自动审核补偿。
func (t *CommentAuditRetry) Exec(args map[string]string) ([]string, error) {
	log.Info(fmt.Sprintf("Job CommentAuditRetry Exec %+v", args))

	lockLeaseMinutes, err := parsePositiveJobInt(args, "lockLeaseMinutes", commentAuditRetryDefaultLockLeaseMinutes)
	if err != nil {
		return []string{err.Error()}, err
	}
	unlock, skippedMessage := t.tryLock(time.Duration(lockLeaseMinutes) * time.Minute)
	if skippedMessage != "" {
		return []string{skippedMessage}, nil
	}
	defer unlock()

	var batchSize int
	batchSize, err = parsePositiveJobInt(args, "batchSize", commentAuditRetryDefaultBatchSize)
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

	return t.retryCommentAudit(batchSize, retryDelayMinutes, maxRetry)
}

// retryCommentAudit 同步补偿执行评价与讨论 AI 审核。
func (t *CommentAuditRetry) retryCommentAudit(batchSize int, retryDelayMinutes int, maxRetry int) ([]string, error) {
	cutoff := time.Now().Add(-time.Duration(retryDelayMinutes) * time.Minute)
	commentTargets, err := t.retryCommentAuditTargets(cutoff, batchSize, maxRetry)
	if err != nil {
		return []string{err.Error()}, err
	}
	var discussionTargets commentAuditRetryDiscussionTargets
	discussionTargets, err = t.retryDiscussionAuditTargets(cutoff, batchSize, maxRetry)
	if err != nil {
		return []string{err.Error()}, err
	}

	commentRunStats := t.auditCommentTargets(commentTargets.records)
	discussionRunStats := t.auditDiscussionTargets(discussionTargets.records)

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
func (t *CommentAuditRetry) retryCommentAuditTargets(cutoff time.Time, batchSize int, maxRetry int) (commentAuditRetryCommentTargets, error) {
	query := t.commentInfoCase.Query(t.ctx).CommentInfo
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	opts = append(opts, repository.Where(query.CreatedAt.Lte(cutoff)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc()))
	opts = append(opts, repository.Limit(commentAuditRetryScanLimit(batchSize)))
	list, err := t.commentInfoCase.List(t.ctx, opts...)
	if err != nil {
		return commentAuditRetryCommentTargets{}, err
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	var reviewMap map[int64]commentAuditReviewState
	reviewMap, err = t.loadAuditReviewState(_const.COMMENT_REVIEW_TARGET_TYPE_COMMENT, ids)
	if err != nil {
		return commentAuditRetryCommentTargets{}, err
	}
	return commentAuditRetryCommentTargets{
		queryCount: len(list),
		records:    filterRetryCommentRecords(list, reviewMap, cutoff, batchSize, maxRetry),
	}, nil
}

// retryDiscussionAuditTargets 查询需要同步补偿自动审核的讨论目标。
func (t *CommentAuditRetry) retryDiscussionAuditTargets(cutoff time.Time, batchSize int, maxRetry int) (commentAuditRetryDiscussionTargets, error) {
	query := t.commentDiscussionCase.Query(t.ctx).CommentDiscussion
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.Status.Eq(_const.COMMENT_STATUS_PENDING_REVIEW)))
	opts = append(opts, repository.Where(query.CreatedAt.Lte(cutoff)))
	opts = append(opts, repository.Order(query.CreatedAt.Asc()))
	opts = append(opts, repository.Limit(commentAuditRetryScanLimit(batchSize)))
	list, err := t.commentDiscussionCase.List(t.ctx, opts...)
	if err != nil {
		return commentAuditRetryDiscussionTargets{}, err
	}

	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	var reviewMap map[int64]commentAuditReviewState
	reviewMap, err = t.loadAuditReviewState(_const.COMMENT_REVIEW_TARGET_TYPE_DISCUSSION, ids)
	if err != nil {
		return commentAuditRetryDiscussionTargets{}, err
	}
	return commentAuditRetryDiscussionTargets{
		queryCount: len(list),
		records:    filterRetryDiscussionRecords(list, reviewMap, cutoff, batchSize, maxRetry),
	}, nil
}

// loadAuditReviewState 查询目标最近一次 AI 审核结果与异常次数。
func (t *CommentAuditRetry) loadAuditReviewState(targetType int32, targetIDs []int64) (map[int64]commentAuditReviewState, error) {
	result := make(map[int64]commentAuditReviewState, len(targetIDs))
	if len(targetIDs) == 0 {
		return result, nil
	}

	query := t.commentReviewCase.Query(t.ctx).CommentReview
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Where(query.TargetType.Eq(targetType)))
	opts = append(opts, repository.Where(query.TargetID.In(targetIDs...)))
	opts = append(opts, repository.Where(query.Type.Eq(_const.COMMENT_REVIEW_TYPE_AI)))
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	list, err := t.commentReviewCase.List(t.ctx, opts...)
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

// auditCommentTargets 同步审核本轮筛选出的评价目标。
func (t *CommentAuditRetry) auditCommentTargets(records []*models.CommentInfo) commentAuditRunStats {
	stats := commentAuditRunStats{}
	for _, record := range records {
		err := t.commentCase.AuditComment(t.ctx, record)
		if err == nil {
			stats.successCount++
			continue
		}
		stats.failedCount++
		stats.addFailureDetail("评价", record.ID, err)
		log.Error(fmt.Sprintf("comment audit retry comment failed, commentID=%d err=%v", record.ID, err))
	}
	return stats
}

// auditDiscussionTargets 同步审核本轮筛选出的讨论目标。
func (t *CommentAuditRetry) auditDiscussionTargets(records []*models.CommentDiscussion) commentAuditRunStats {
	stats := commentAuditRunStats{}
	for _, record := range records {
		err := t.commentCase.AuditDiscussion(t.ctx, record)
		if err == nil {
			stats.successCount++
			continue
		}
		stats.failedCount++
		stats.addFailureDetail("讨论", record.ID, err)
		log.Error(fmt.Sprintf("comment audit retry discussion failed, discussionID=%d err=%v", record.ID, err))
	}
	return stats
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

// addFailureDetail 追加单目标失败摘要，避免任务日志输出过长。
func (s *commentAuditRunStats) addFailureDetail(targetName string, targetID int64, err error) {
	if len(s.failureDetails) >= commentAuditRetryFailureDetailLimit {
		return
	}
	s.failureDetails = append(s.failureDetails, fmt.Sprintf("%s %d: %v", targetName, targetID, err))
}

// commentAuditRetryScanLimit 返回单次补偿扫描上限。
func commentAuditRetryScanLimit(batchSize int) int {
	scanLimit := batchSize * commentAuditRetryScanFactor
	if scanLimit < commentAuditRetryMinScanLimit {
		return commentAuditRetryMinScanLimit
	}
	return scanLimit
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
