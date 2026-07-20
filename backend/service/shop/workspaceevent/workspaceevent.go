package workspaceevent

import (
	"context"
	"fmt"
	"sync"
	"time"

	shopcommonv1 "shop/api/gen/go/shop/common/v1"
	transportSSE "shop/pkg/sse"

	"github.com/go-kratos/kratos/v3/log"
)

const (
	// SSEStreamAdminWorkspace 表示管理后台工作台 SSE 流。
	SSEStreamAdminWorkspace = "shop.admin.workspace"
	// SSEEventWorkspaceRefresh 表示工作台局部刷新事件。
	SSEEventWorkspaceRefresh = "workspace.refresh"
	// AreaMetrics 表示工作台顶部指标区域。
	AreaMetrics = shopcommonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_METRICS
	// AreaTodo 表示工作台待处理事项区域。
	AreaTodo = shopcommonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_TODO
	// AreaRisk 表示工作台风险提醒区域。
	AreaRisk = shopcommonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_RISK
	// AreaReputation 表示工作台口碑洞察区域。
	AreaReputation = shopcommonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_REPUTATION
	// AreaPendingComments 表示工作台待审核评价区域。
	AreaPendingComments = shopcommonv1.SseRefreshTarget_SSE_REFRESH_TARGET_ADMIN_WORKSPACE_PENDING_COMMENTS
	// ReasonOrderChanged 表示订单状态或金额口径发生变化。
	ReasonOrderChanged = shopcommonv1.SseRefreshReason_SSE_REFRESH_REASON_ORDER_CHANGED
	// ReasonGoodsChanged 表示商品、SKU、库存或价格口径发生变化。
	ReasonGoodsChanged = shopcommonv1.SseRefreshReason_SSE_REFRESH_REASON_GOODS_CHANGED
	// ReasonCommentChanged 表示评价或评价讨论口径发生变化。
	ReasonCommentChanged = shopcommonv1.SseRefreshReason_SSE_REFRESH_REASON_COMMENT_CHANGED
	// ReasonPayBillChecked 表示支付账单对账结果发生变化。
	ReasonPayBillChecked = shopcommonv1.SseRefreshReason_SSE_REFRESH_REASON_PAY_BILL_CHECKED
)

// RefreshPayload 表示通过 SSE 推送给管理后台的工作台刷新消息。
type RefreshPayload struct {
	Event      string                          `json:"event"`
	Targets    []shopcommonv1.SseRefreshTarget `json:"targets"`
	Reason     shopcommonv1.SseRefreshReason   `json:"reason,omitempty"`
	OccurredAt string                          `json:"occurred_at"`
}

// Publisher 表示工作台刷新消息发布函数。
type Publisher func(context.Context, RefreshPayload) error

var (
	publisherMu sync.RWMutex
	publisher   Publisher
)

// SSEReady 表示商城工作台 SSE 已注册到基础传输层。
type SSEReady struct{}

// workspaceSSEStream 描述商城工作台 SSE 流。
type workspaceSSEStream struct{}

var _ transportSSE.Stream = workspaceSSEStream{}

// NewSSEReady 注册商城工作台流并装配刷新消息发布器。
func NewSSEReady(registry *transportSSE.Registry, publisher *transportSSE.Publisher) (SSEReady, error) {
	err := registry.Register(workspaceSSEStream{})
	if err != nil {
		return SSEReady{}, err
	}
	SetPublisher(func(ctx context.Context, payload RefreshPayload) error {
		return publisher.PublishJSON(ctx, SSEStreamAdminWorkspace, SSEEventWorkspaceRefresh, payload)
	})
	return SSEReady{}, nil
}

// ID 返回商城工作台 SSE 流标识。
func (workspaceSSEStream) ID() string {
	return SSEStreamAdminWorkspace
}

// Resolve 返回商城工作台固定的传输流标识。
func (workspaceSSEStream) Resolve(_ string, _ int64) (string, error) {
	return SSEStreamAdminWorkspace, nil
}

// SetPublisher 设置工作台刷新消息发布函数。
func SetPublisher(fn Publisher) {
	publisherMu.Lock()
	defer publisherMu.Unlock()
	publisher = fn
}

// Publish 发布工作台局部刷新消息。
func Publish(ctx context.Context, reason shopcommonv1.SseRefreshReason, targets ...shopcommonv1.SseRefreshTarget) {
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
		Event:      SSEEventWorkspaceRefresh,
		Targets:    normalizedTargets,
		Reason:     reason,
		OccurredAt: time.Now().Format(time.RFC3339),
	})
	if err != nil {
		log.Error(fmt.Sprintf("publish workspace refresh event: %v", err.Error()))
	}
}

// normalizeTargets 去重并过滤非法工作台刷新目标。
func normalizeTargets(targets []shopcommonv1.SseRefreshTarget) []shopcommonv1.SseRefreshTarget {
	seen := make(map[shopcommonv1.SseRefreshTarget]struct{}, len(targets))
	normalizedTargets := make([]shopcommonv1.SseRefreshTarget, 0, len(targets))
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
func isValidTarget(target shopcommonv1.SseRefreshTarget) bool {
	switch target {
	case AreaMetrics, AreaTodo, AreaRisk, AreaReputation, AreaPendingComments:
		return true
	default:
		return false
	}
}
