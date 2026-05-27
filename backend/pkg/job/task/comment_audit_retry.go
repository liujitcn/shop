package task

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"shop/pkg/errorsx"
	appBiz "shop/service/app/biz"

	"github.com/go-kratos/kratos/v2/log"
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
)

// CommentAuditRetry 评价与讨论自动审核补偿任务。
type CommentAuditRetry struct {
	commentCase  *appBiz.CommentCase
	ctx          context.Context
	mu           sync.Mutex
	runningUntil time.Time
}

// NewCommentAuditRetry 创建评价与讨论自动审核补偿任务实例。
func NewCommentAuditRetry(commentCase *appBiz.CommentCase) *CommentAuditRetry {
	return &CommentAuditRetry{
		commentCase: commentCase,
		ctx:         context.Background(),
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

	return t.commentCase.RetryCommentAudit(t.ctx, batchSize, retryDelayMinutes, maxRetry)
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
