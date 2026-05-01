package workspaceevent

import (
	"context"
	"strconv"
	"sync"
	"time"

	commonv1 "shop/api/gen/go/common/v1"

	"github.com/go-kratos/kratos/v2/log"
)

const (
	// StreamAdmin 表示管理后台通用 SSE 流。
	StreamAdmin = commonv1.SseStream_SSE_STREAM_ADMIN
	// EventWorkspaceRefresh 表示工作台局部刷新事件。
	EventWorkspaceRefresh = commonv1.SseEvent_SSE_EVENT_PAGE_REFRESH
	// AreaMetrics 表示工作台顶部指标区域。
	AreaMetrics = commonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_METRICS
	// AreaTodo 表示工作台待处理事项区域。
	AreaTodo = commonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_TODO
	// AreaRisk 表示工作台风险提醒区域。
	AreaRisk = commonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_RISK
	// AreaReputation 表示工作台口碑洞察区域。
	AreaReputation = commonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_REPUTATION
	// AreaPendingComments 表示工作台待审核评价区域。
	AreaPendingComments = commonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_PENDING_COMMENTS
	// ReasonOrderChanged 表示订单状态或金额口径发生变化。
	ReasonOrderChanged = commonv1.SseRefreshReason_SSE_REFRESH_REASON_ORDER_CHANGED
	// ReasonGoodsChanged 表示商品、SKU、库存或价格口径发生变化。
	ReasonGoodsChanged = commonv1.SseRefreshReason_SSE_REFRESH_REASON_GOODS_CHANGED
	// ReasonCommentChanged 表示评价或评价讨论口径发生变化。
	ReasonCommentChanged = commonv1.SseRefreshReason_SSE_REFRESH_REASON_COMMENT_CHANGED
	// ReasonPayBillChecked 表示支付账单对账结果发生变化。
	ReasonPayBillChecked = commonv1.SseRefreshReason_SSE_REFRESH_REASON_PAY_BILL_CHECKED
)

// RefreshPayload 表示通过 SSE 推送给管理后台的工作台刷新消息。
type RefreshPayload struct {
	Event      commonv1.SseEvent           `json:"event"`
	Targets    []commonv1.SseRefreshTarget `json:"targets"`
	Reason     commonv1.SseRefreshReason   `json:"reason,omitempty"`
	OccurredAt string                      `json:"occurred_at"`
}

// Publisher 表示工作台刷新消息发布函数。
type Publisher func(context.Context, RefreshPayload) error

var (
	publisherMu sync.RWMutex
	publisher   Publisher
)

// SetPublisher 设置工作台刷新消息发布函数。
func SetPublisher(fn Publisher) {
	publisherMu.Lock()
	defer publisherMu.Unlock()
	publisher = fn
}

// StreamID 返回工作台 SSE 流的传输层标识。
func StreamID(stream commonv1.SseStream) string {
	return strconv.FormatInt(int64(stream), 10)
}

// EventID 返回工作台 SSE 事件的传输层标识。
func EventID(event commonv1.SseEvent) string {
	return strconv.FormatInt(int64(event), 10)
}

// Publish 发布工作台局部刷新消息。
func Publish(ctx context.Context, reason commonv1.SseRefreshReason, targets ...commonv1.SseRefreshTarget) {
	normalizedTargets := normalizeTargets(targets)
	if len(normalizedTargets) == 0 {
		return
	}

	publisherMu.RLock()
	fn := publisher
	publisherMu.RUnlock()
	if fn == nil {
		return
	}

	err := fn(ctx, RefreshPayload{
		Event:      EventWorkspaceRefresh,
		Targets:    normalizedTargets,
		Reason:     reason,
		OccurredAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		log.Errorf("publish workspace refresh event: %v", err.Error())
	}
}

// normalizeTargets 去重并过滤非法工作台刷新目标。
func normalizeTargets(targets []commonv1.SseRefreshTarget) []commonv1.SseRefreshTarget {
	seen := make(map[commonv1.SseRefreshTarget]struct{}, len(targets))
	normalizedTargets := make([]commonv1.SseRefreshTarget, 0, len(targets))
	for _, target := range targets {
		if !isValidTarget(target) {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		normalizedTargets = append(normalizedTargets, target)
	}
	return normalizedTargets
}

// isValidTarget 判断是否为受支持的工作台刷新目标。
func isValidTarget(target commonv1.SseRefreshTarget) bool {
	switch target {
	case AreaMetrics, AreaTodo, AreaRisk, AreaReputation, AreaPendingComments:
		return true
	default:
		return false
	}
}
