package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/llm"
	adminService "shop/service/admin"
	baseBiz "shop/service/base/biz"
	baseDTO "shop/service/base/dto"

	"github.com/go-kratos/blades/tools"
)

// aiAssistantToolRuntime 负责在 AI 助手中执行本地 Agent 工具。
type aiAssistantToolRuntime struct {
	workspaceTools []tools.Tool
	orderInfoTools []tools.Tool
}

// NewAiAssistantToolRuntime 创建 AI 助手 Agent 工具运行时。
func NewAiAssistantToolRuntime(workspaceService *adminService.WorkspaceService, orderInfoService *adminService.OrderInfoService) (baseBiz.AiAssistantToolRuntime, error) {
	workspaceTools, err := adminv1.NewWorkspaceServiceAgentTools(workspaceService)
	if err != nil {
		return nil, err
	}
	orderInfoTools, err := adminv1.NewOrderInfoServiceAgentTools(orderInfoService)
	if err != nil {
		return nil, err
	}
	return &aiAssistantToolRuntime{
		workspaceTools: workspaceTools,
		orderInfoTools: orderInfoTools,
	}, nil
}

// RunToolCalls 执行当前输入命中的工具集合。
func (r *aiAssistantToolRuntime) RunToolCalls(ctx context.Context, input baseBiz.AiAssistantToolRuntimeInput) (*baseBiz.AiAssistantToolRuntimeResult, error) {
	if normalizeToolScene(input.Scene) != "workspace" {
		return &baseBiz.AiAssistantToolRuntimeResult{}, nil
	}

	result := &baseBiz.AiAssistantToolRuntimeResult{
		Tools: make([]llm.AiAssistantToolCall, 0, len(r.workspaceTools)+2),
	}
	summaryList := make([]string, 0, len(r.workspaceTools)+2)

	selectedWorkspaceTools := selectWorkspaceTools(input.Content)
	for _, toolName := range selectedWorkspaceTools {
		tool, ok := findToolByName(r.workspaceTools, toolName)
		if !ok {
			continue
		}
		toolInput := resolveWorkspaceToolInput(tool.Name())
		toolCall := runTool(ctx, tool, toolInput)
		if toolCall.Status == baseDTO.AiAssistantToolStatusSuccess {
			summaryList = append(summaryList, fmt.Sprintf("%s => %s", tool.Name(), toolCall.Output))
		}
		result.Tools = append(result.Tools, toolCall)
	}

	if shipmentOrderID := extractShipmentOrderID(input.Content); shipmentOrderID > 0 {
		shipmentTool, ok := findToolByName(r.orderInfoTools, "admin_v1_order_info_service_get_order_info_shipment")
		if ok {
			toolInput := fmt.Sprintf(`{"id":%d}`, shipmentOrderID)
			toolCall := runTool(ctx, shipmentTool, toolInput)
			if toolCall.Status == baseDTO.AiAssistantToolStatusSuccess {
				summaryList = append(summaryList, fmt.Sprintf("%s => %s", shipmentTool.Name(), toolCall.Output))
			}
			result.Tools = append(result.Tools, toolCall)
		}
		result.Confirm = buildShipmentConfirmRequest(shipmentOrderID)
	}
	result.PromptAugment = strings.Join(summaryList, "\n")
	return result, nil
}

// ExecuteConfirm 执行确认卡动作。
func (r *aiAssistantToolRuntime) ExecuteConfirm(ctx context.Context, input baseBiz.AiAssistantConfirmRuntimeInput) (*baseBiz.AiAssistantConfirmRuntimeResult, error) {
	if input.Confirm == nil {
		return &baseBiz.AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusFailed,
			Summary: "确认信息不存在",
			Reply:   "当前确认请求缺少执行上下文，请重新发起。",
		}, nil
	}

	if strings.TrimSpace(strings.ToLower(input.Action)) == "reject" {
		return &baseBiz.AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusRejected,
			Summary: "已拒绝执行",
			Reply:   "好的，我先不执行这个操作。",
		}, nil
	}

	switch strings.TrimSpace(input.Confirm.Action) {
	case "workspace.shipment.confirm":
		return r.executeShipmentConfirm(ctx, input)
	default:
		return &baseBiz.AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusFailed,
			Summary: "暂不支持的确认动作",
			Reply:   "当前版本还不支持执行这个确认动作，我已经保留上下文供后续处理。",
		}, nil
	}
}

func normalizeToolScene(scene string) string {
	switch strings.TrimSpace(scene) {
	case "recommend":
		return "recommend"
	case "comment":
		return "comment"
	default:
		return "workspace"
	}
}

func shouldRunWorkspaceTools(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	keywords := []string{"工作台", "订单", "成交", "发货", "库存", "风险", "评价", "待处理", "分析", "概览"}
	for _, keyword := range keywords {
		if strings.Contains(trimmed, keyword) {
			return true
		}
	}
	return false
}

func selectWorkspaceTools(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	selected := make([]string, 0, 4)
	if strings.Contains(trimmed, "风险") {
		selected = append(selected, "admin_v1_workspace_service_summary_workspace_risk")
	}
	if strings.Contains(trimmed, "评价") || strings.Contains(trimmed, "口碑") {
		selected = append(selected, "admin_v1_workspace_service_summary_workspace_reputation", "admin_v1_workspace_service_list_workspace_pending_comments")
	}
	if strings.Contains(trimmed, "待处理") || strings.Contains(trimmed, "发货") || strings.Contains(trimmed, "订单") {
		selected = append(selected, "admin_v1_workspace_service_summary_workspace_todo")
	}
	if strings.Contains(trimmed, "工作台") || strings.Contains(trimmed, "概览") || strings.Contains(trimmed, "成交") || strings.Contains(trimmed, "指标") {
		selected = append(selected, "admin_v1_workspace_service_summary_workspace_metrics")
	}
	if len(selected) == 0 && shouldRunWorkspaceTools(content) {
		selected = append(selected, "admin_v1_workspace_service_summary_workspace_metrics")
	}
	return uniqueToolNames(selected)
}

func resolveWorkspaceToolInput(toolName string) string {
	switch toolName {
	case "admin_v1_workspace_service_list_workspace_pending_comments":
		return `{"limit":3}`
	default:
		return `{}`
	}
}

func summarizeToolOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "工具已执行，未返回可展示内容"
	}
	runes := []rune(trimmed)
	if len(runes) <= 80 {
		return trimmed
	}
	return string(runes[:80]) + "..."
}

func buildShipmentConfirmRequest(orderID int64) *baseDTO.AiAssistantConfirmRequest {
	payload, _ := json.Marshal(map[string]any{
		"operation": "shipment_confirm",
		"scope":     "order_info",
		"order_id":  orderID,
	})
	return &baseDTO.AiAssistantConfirmRequest{
		Title:   "确认执行订单发货",
		Lines:   []string{fmt.Sprintf("检测到你想处理订单 %d 的发货。", orderID), "确认后会按订单发货接口继续执行该动作。"},
		Action:  "workspace.shipment.confirm",
		Summary: fmt.Sprintf("待确认执行订单 %d 发货", orderID),
		Payload: payload,
		FormSchema: []baseDTO.AiAssistantConfirmFormField{
			{
				Prop:        "name",
				Label:       "物流公司名称",
				Placeholder: "请输入物流公司名称",
				Required:    true,
			},
			{
				Prop:        "no",
				Label:       "物流单号",
				Placeholder: "请输入物流单号",
				Required:    true,
			},
			{
				Prop:        "contact",
				Label:       "联系方式",
				Placeholder: "请输入联系方式",
				Required:    true,
			},
		},
	}
}

func (r *aiAssistantToolRuntime) executeShipmentConfirm(ctx context.Context, input baseBiz.AiAssistantConfirmRuntimeInput) (*baseBiz.AiAssistantConfirmRuntimeResult, error) {
	orderID := extractConfirmOrderID(input.Confirm)
	if orderID <= 0 {
		return &baseBiz.AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusFailed,
			Summary: "缺少订单编号",
			Reply:   "当前确认动作缺少订单编号，暂时无法继续发货。",
		}, nil
	}
	formPayload, err := parseShipmentConfirmForm(input.FormJSON)
	if err != nil {
		return &baseBiz.AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusFailed,
			Summary: "发货表单不完整",
			Reply:   err.Error(),
		}, nil
	}

	toolCalls := make([]llm.AiAssistantToolCall, 0, 1)
	targetTool := "admin_v1_order_info_service_ship_order_info"
	for _, tool := range r.orderInfoTools {
		if tool.Name() != targetTool {
			continue
		}
		toolInput := fmt.Sprintf(`{"order_id":%d,"name":%q,"no":%q,"contact":%q}`, orderID, formPayload.Name, formPayload.No, formPayload.Contact)
		toolCall := runTool(ctx, tool, toolInput)
		if toolCall.Status == baseDTO.AiAssistantToolStatusFailed {
			toolCalls = append(toolCalls, toolCall)
			return &baseBiz.AiAssistantConfirmRuntimeResult{
				Status:  baseDTO.AiAssistantConfirmStatusFailed,
				Summary: "确认后执行失败",
				Reply:   fmt.Sprintf("已收到确认，但订单 %d 发货失败，请稍后重试。", orderID),
				Tools:   toolCalls,
			}, nil
		}
		toolCalls = append(toolCalls, toolCall)
		return &baseBiz.AiAssistantConfirmRuntimeResult{
			Status:  baseDTO.AiAssistantConfirmStatusApproved,
			Summary: fmt.Sprintf("已确认执行订单 %d 发货", orderID),
			Reply:   fmt.Sprintf("已收到确认。订单 %d 的发货动作已经提交。", orderID),
			Tools:   toolCalls,
		}, nil
	}

	return &baseBiz.AiAssistantConfirmRuntimeResult{
		Status:  baseDTO.AiAssistantConfirmStatusFailed,
		Summary: "缺少待执行工具",
		Reply:   "当前环境没有可执行的订单发货工具，请稍后重试。",
	}, nil
}

func findToolByName(toolList []tools.Tool, target string) (tools.Tool, bool) {
	for _, tool := range toolList {
		if tool.Name() == target {
			return tool, true
		}
	}
	return nil, false
}

func runTool(ctx context.Context, tool tools.Tool, toolInput string) llm.AiAssistantToolCall {
	startedAt := time.Now()
	output, err := tool.Handle(ctx, toolInput)
	elapsed := time.Since(startedAt).Round(time.Millisecond).String()
	toolCall := llm.AiAssistantToolCall{
		Name:    tool.Name(),
		Status:  baseDTO.AiAssistantToolStatusSuccess,
		Elapsed: elapsed,
		Input:   toolInput,
		Summary: summarizeToolOutput(output),
		Output:  strings.TrimSpace(output),
	}
	if err != nil {
		toolCall.Status = baseDTO.AiAssistantToolStatusFailed
		toolCall.Summary = "工具调用失败"
		toolCall.ErrorMessage = err.Error()
		toolCall.Output = ""
	}
	return toolCall
}

func uniqueToolNames(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func extractShipmentOrderID(content string) int64 {
	fields := strings.FieldsFunc(content, func(r rune) bool {
		return r < '0' || r > '9'
	})
	for _, field := range fields {
		if len(field) < 3 {
			continue
		}
		orderID, err := strconv.ParseInt(field, 10, 64)
		if err == nil && orderID > 0 {
			return orderID
		}
	}
	return 0
}

func extractConfirmOrderID(confirm *baseDTO.AiAssistantConfirmState) int64 {
	if confirm == nil || len(confirm.Payload) == 0 {
		return 0
	}
	payload := struct {
		OrderID int64 `json:"order_id"`
	}{}
	if err := json.Unmarshal(confirm.Payload, &payload); err != nil {
		return 0
	}
	return payload.OrderID
}

type shipmentConfirmForm struct {
	Name    string `json:"name"`
	No      string `json:"no"`
	Contact string `json:"contact"`
}

func parseShipmentConfirmForm(raw string) (*shipmentConfirmForm, error) {
	form := &shipmentConfirmForm{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), form); err != nil {
		return nil, fmt.Errorf("请完整填写发货信息后再确认")
	}
	form.Name = strings.TrimSpace(form.Name)
	form.No = strings.TrimSpace(form.No)
	form.Contact = strings.TrimSpace(form.Contact)
	if form.Name == "" || form.No == "" || form.Contact == "" {
		return nil, fmt.Errorf("请完整填写物流公司、物流单号和联系方式")
	}
	return form, nil
}
