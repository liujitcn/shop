package workflow

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
)

const unsupportedActionNode = "unsupported_action"

// FlowName 表示移动端固定流程名称。
type FlowName string

const (
	// FlowShopping 表示推荐、选规格、确认订单、支付流程。
	FlowShopping FlowName = "shopping"
	// FlowPendingPayment 表示待支付订单流程。
	FlowPendingPayment FlowName = "pending_payment"
	// FlowPendingReview 表示待评价流程。
	FlowPendingReview FlowName = "pending_review"
	// FlowOrderLogistics 表示订单物流流程。
	FlowOrderLogistics FlowName = "order_logistics"
)

// Action 表示流程动作定义。
type Action struct {
	// Flow 流程名称。
	Flow FlowName
	// Step 流程步骤。
	Step string
	// Type 动作类型。
	Type string
}

// ActionRequest 表示固定流程动作执行输入。
type ActionRequest struct {
	// Flow 前端当前所在流程。列表项选择后会随按钮动作一并回传，用于区分相同动作类型在不同流程下的语义。
	Flow FlowName
	// ActionType 前端按钮动作类型。
	ActionType string
	// Payload 前端按钮携带的上下文，例如商品 ID、SKU、订单 ID、评价商品信息。
	Payload map[string]any
}

// LookupRequest 表示固定流程动作查询输入。
type LookupRequest struct {
	// Flow 流程名称。
	Flow FlowName
	// ActionType 动作类型。
	ActionType string
}

// LookupResult 表示固定流程动作查询结果。
type LookupResult struct {
	// Definition 命中的流程定义。
	Definition Definition
	// Action 命中的动作定义。
	Action Action
	// Found 表示是否命中动作。
	Found bool
}

// ActionResult 表示固定流程动作执行结果。
type ActionResult[T any] struct {
	// Definition 命中的流程定义。
	Definition Definition
	// Action 命中的动作定义。
	Action Action
	// Output 业务动作输出。
	Output T
	// Found 表示是否命中动作。
	Found bool
}

// ActionHandler 表示固定流程动作处理函数。
type ActionHandler[T any] func(ctx context.Context, action Action, payload map[string]any) (T, error)

// Definition 表示一个固定流程的动作图定义。
type Definition struct {
	// Name 流程名称。
	Name FlowName
	// EntryAction 入口动作类型。
	EntryAction string
	// Actions 流程可处理动作。
	Actions []Action
}

// Registry 保存移动端固定流程定义。
type Registry[T any] struct {
	definitions          map[FlowName]Definition
	actionsByType        map[string][]Action
	actionsByFlowAndType map[string]Action
	lookupRunnable       compose.Runnable[LookupRequest, LookupResult]
	actionRunnable       compose.Runnable[ActionRequest, LookupResult]
}

// NewAppRegistry 创建商城端固定流程注册表。
func NewAppRegistry[T any]() (*Registry[T], error) {
	definitions := []Definition{
		{
			Name:        FlowShopping,
			EntryAction: "open_shopping",
			Actions: []Action{
				{Flow: FlowShopping, Step: "goods", Type: "open_shopping"},
				{Flow: FlowShopping, Step: "sku", Type: "select_goods"},
				{Flow: FlowShopping, Step: "checkout", Type: "select_sku"},
				{Flow: FlowShopping, Step: "address", Type: "create_address"},
				{Flow: FlowShopping, Step: "confirm", Type: "select_address"},
				{Flow: FlowShopping, Step: "confirm", Type: "confirm_order"},
				{Flow: FlowShopping, Step: "payment", Type: "start_payment"},
			},
		},
		{
			Name:        FlowPendingPayment,
			EntryAction: "open_pending_payment",
			Actions: []Action{
				{Flow: FlowPendingPayment, Step: "list", Type: "open_pending_payment"},
				{Flow: FlowPendingPayment, Step: "payment", Type: "start_payment"},
			},
		},
		{
			Name:        FlowPendingReview,
			EntryAction: "open_pending_review",
			Actions: []Action{
				{Flow: FlowPendingReview, Step: "list", Type: "open_pending_review"},
				{Flow: FlowPendingReview, Step: "form", Type: "open_review_form"},
				{Flow: FlowPendingReview, Step: "submit", Type: "submit_review"},
			},
		},
		{
			Name:        FlowOrderLogistics,
			EntryAction: "open_order_logistics",
			Actions: []Action{
				{Flow: FlowOrderLogistics, Step: "list", Type: "open_order_logistics"},
				{Flow: FlowOrderLogistics, Step: "detail", Type: "view_order"},
				{Flow: FlowOrderLogistics, Step: "receipt", Type: "receive_order"},
			},
		},
	}
	var err error
	err = validateDefinitions(definitions)
	if err != nil {
		return nil, err
	}
	registry := &Registry[T]{
		definitions:          make(map[FlowName]Definition, len(definitions)),
		actionsByType:        make(map[string][]Action, 16),
		actionsByFlowAndType: make(map[string]Action, 16),
	}
	for _, definition := range definitions {
		registry.definitions[definition.Name] = definition
		for _, action := range definition.Actions {
			// 同一个动作类型可以出现在不同流程中，例如 start_payment 既可以来自下单流程，也可以来自待支付列表。
			registry.actionsByType[action.Type] = append(registry.actionsByType[action.Type], action)
			registry.actionsByFlowAndType[actionKey(action.Flow, action.Type)] = action
		}
	}
	registry.lookupRunnable, err = compileLookupWorkflow(registry)
	if err != nil {
		return nil, err
	}
	registry.actionRunnable, err = compileActionGraph(registry)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// MustNewAppRegistry 创建商城端固定流程注册表，编排定义错误时直接失败。
func MustNewAppRegistry[T any]() *Registry[T] {
	registry, err := NewAppRegistry[T]()
	if err != nil {
		panic(err)
	}
	return registry
}

// EntryAction 返回流程入口动作。
func (r *Registry[T]) EntryAction(flow FlowName) string {
	// 注册表为空时返回空入口，调用方可继续走默认购物流程。
	if r == nil {
		return ""
	}
	return r.definitions[flow].EntryAction
}

// Action 返回指定流程内的动作定义。
func (r *Registry[T]) Action(flow FlowName, actionType string) (Action, bool) {
	// 流程或动作为空都无法形成稳定路由，直接返回未命中。
	if r == nil || flow == "" || actionType == "" {
		return Action{}, false
	}
	result := r.lookup(LookupRequest{Flow: flow, ActionType: actionType})
	if !result.Found {
		return Action{}, false
	}
	return result.Action, true
}

// UniqueAction 返回全局唯一动作定义。
//
// 列表型流程中会复用动作类型，例如 start_payment 既可能来自下单后的支付面板，
// 也可能来自待支付订单列表。调用方没有流程上下文时只能查询全局唯一动作，
// 命中多个流程时必须改用 Action(flow, actionType)，避免把后续步骤推进到错误流程。
func (r *Registry[T]) UniqueAction(actionType string) (Action, bool) {
	// 空动作没有业务语义，直接返回未命中。
	if r == nil || actionType == "" {
		return Action{}, false
	}
	result := r.lookup(LookupRequest{ActionType: actionType})
	if !result.Found {
		return Action{}, false
	}
	return result.Action, true
}

// Run 通过 Eino Graph 执行固定流程动作。
//
// Quick Start 的 Graph Tool 思路是把“先判断去哪个节点，再执行节点能力”交给 Graph。
// 这里 Graph 只负责路由出动作元信息，业务处理函数在 Graph 外部执行，
// 避免把不可序列化的运行时对象放进 Graph 输入，便于后续接 Interrupt/Resume。
func (r *Registry[T]) Run(ctx context.Context, request ActionRequest, handler ActionHandler[T]) (ActionResult[T], error) {
	if r == nil {
		return ActionResult[T]{}, nil
	}
	var lookup LookupResult
	if r.actionRunnable == nil {
		lookup = r.lookup(LookupRequest{Flow: request.Flow, ActionType: request.ActionType})
	} else {
		var err error
		lookup, err = r.actionRunnable.Invoke(ctx, request)
		if err != nil {
			return ActionResult[T]{}, err
		}
	}
	return r.executeAction(ctx, request, lookup, handler)
}

// HasFlow 判断流程是否存在。
func (r *Registry[T]) HasFlow(flow FlowName) bool {
	// nil 注册表表示固定流程能力未初始化。
	if r == nil {
		return false
	}
	_, ok := r.definitions[flow]
	return ok
}

// Lookup 通过 Eino Workflow 查询固定流程动作。
func (r *Registry[T]) Lookup(ctx context.Context, request LookupRequest) (LookupResult, error) {
	if r == nil {
		return LookupResult{}, nil
	}
	if r.lookupRunnable == nil {
		return r.lookup(request), nil
	}
	return r.lookupRunnable.Invoke(ctx, request)
}

// lookup 执行固定流程动作查询的纯业务逻辑。
func (r *Registry[T]) lookup(request LookupRequest) LookupResult {
	if request.ActionType == "" {
		return LookupResult{}
	}
	if request.Flow != "" {
		action, ok := r.actionsByFlowAndType[actionKey(request.Flow, request.ActionType)]
		// 前端已声明流程时，动作必须属于该流程，避免跨流程误推进。
		if !ok {
			return LookupResult{Definition: r.definitions[request.Flow]}
		}
		return LookupResult{Definition: r.definitions[action.Flow], Action: action, Found: true}
	}
	actions := r.actionsByType[request.ActionType]
	if len(actions) == 0 {
		return LookupResult{}
	}
	// 同一个 actionType 分属多个流程时必须带 flow，避免漏传 flow 后误推进到第一个注册流程。
	if len(actions) > 1 {
		return LookupResult{}
	}
	action := actions[0]
	definition := r.definitions[action.Flow]
	return LookupResult{Definition: definition, Action: action, Found: true}
}

// validateDefinitions 校验固定流程定义，避免运行期才发现路由配置错误。
func validateDefinitions(definitions []Definition) error {
	flows := make(map[FlowName]bool, len(definitions))
	actions := make(map[string]bool, len(definitions)*4)
	for _, definition := range definitions {
		// 流程名是前端 action 协议的一部分，不能为空也不能重复。
		if definition.Name == "" {
			return fmt.Errorf("商城固定流程名称不能为空")
		}
		if flows[definition.Name] {
			return fmt.Errorf("商城固定流程重复定义: %s", definition.Name)
		}
		flows[definition.Name] = true
		if definition.EntryAction == "" {
			return fmt.Errorf("商城固定流程 %s 缺少入口动作", definition.Name)
		}
		for _, action := range definition.Actions {
			// 动作必须归属当前流程，否则 flow + action_type 路由会指向错误流程。
			if action.Flow != definition.Name {
				return fmt.Errorf("商城固定流程 %s 包含跨流程动作 %s", definition.Name, action.Type)
			}
			if action.Step == "" || action.Type == "" {
				return fmt.Errorf("商城固定流程 %s 包含无效动作", definition.Name)
			}
			key := actionKey(action.Flow, action.Type)
			if actions[key] {
				return fmt.Errorf("商城固定流程动作重复定义: %s/%s", action.Flow, action.Type)
			}
			actions[key] = true
		}
		if !actions[actionKey(definition.Name, definition.EntryAction)] {
			return fmt.Errorf("商城固定流程 %s 入口动作未注册: %s", definition.Name, definition.EntryAction)
		}
	}
	return nil
}

// compileLookupWorkflow 编译固定流程查询 Workflow。
func compileLookupWorkflow[T any](registry *Registry[T]) (compose.Runnable[LookupRequest, LookupResult], error) {
	workflow := compose.NewWorkflow[LookupRequest, LookupResult]()
	workflow.AddLambdaNode("lookup_action", compose.InvokableLambda(func(_ context.Context, input LookupRequest) (LookupResult, error) {
		if registry == nil {
			return LookupResult{}, nil
		}
		return registry.lookup(input), nil
	}, compose.WithLambdaType("shop.workflow.lookup_action"))).AddDependency(compose.START)
	workflow.End().AddDependency("lookup_action")
	runnable, err := workflow.Compile(context.Background(), compose.WithGraphName("shop_app_fixed_flow"))
	if err != nil {
		return nil, fmt.Errorf("编译商城固定流程 Workflow 失败: %w", err)
	}
	return runnable, nil
}

// compileActionGraph 编译固定流程动作 Graph。
func compileActionGraph[T any](registry *Registry[T]) (compose.Runnable[ActionRequest, LookupResult], error) {
	graph := compose.NewGraph[ActionRequest, LookupResult]()
	endNodes := make(map[string]bool, len(registry.actionsByFlowAndType)+1)
	for _, actions := range registry.actionsByType {
		for _, action := range actions {
			nodeKey := actionNodeKey(action)
			endNodes[nodeKey] = true
			currentAction := action
			err := graph.AddLambdaNode(nodeKey, compose.InvokableLambda(func(_ context.Context, input ActionRequest) (LookupResult, error) {
				lookup := registry.lookup(LookupRequest{Flow: input.Flow, ActionType: input.ActionType})
				if !lookup.Found {
					return LookupResult{}, nil
				}
				return LookupResult{Definition: lookup.Definition, Action: currentAction, Found: true}, nil
			}, compose.WithLambdaType("shop.workflow."+nodeKey)))
			if err != nil {
				return nil, fmt.Errorf("注册商城固定流程动作节点失败: %w", err)
			}
			err = graph.AddEdge(nodeKey, compose.END)
			if err != nil {
				return nil, fmt.Errorf("连接商城固定流程动作节点失败: %w", err)
			}
		}
	}

	endNodes[unsupportedActionNode] = true
	err := graph.AddLambdaNode(unsupportedActionNode, compose.InvokableLambda(func(context.Context, ActionRequest) (LookupResult, error) {
		return LookupResult{}, nil
	}, compose.WithLambdaType("shop.workflow.unsupported_action")))
	if err != nil {
		return nil, fmt.Errorf("注册商城固定流程兜底节点失败: %w", err)
	}
	err = graph.AddEdge(unsupportedActionNode, compose.END)
	if err != nil {
		return nil, fmt.Errorf("连接商城固定流程兜底节点失败: %w", err)
	}

	err = graph.AddBranch(compose.START, compose.NewGraphBranch(func(_ context.Context, input ActionRequest) (string, error) {
		lookup := registry.lookup(LookupRequest{Flow: input.Flow, ActionType: input.ActionType})
		// 未注册动作进入兜底节点，由调用方按 Found=false 转成业务错误。
		if !lookup.Found {
			return unsupportedActionNode, nil
		}
		return actionNodeKey(lookup.Action), nil
	}, endNodes))
	if err != nil {
		return nil, fmt.Errorf("注册商城固定流程路由分支失败: %w", err)
	}

	runnable, err := graph.Compile(context.Background(), compose.WithGraphName("shop_app_fixed_flow_action_graph"))
	if err != nil {
		return nil, fmt.Errorf("编译商城固定流程 Action Graph 失败: %w", err)
	}
	return runnable, nil
}

// executeAction 执行已路由到具体节点的固定流程动作。
func (r *Registry[T]) executeAction(ctx context.Context, request ActionRequest, lookup LookupResult, handler ActionHandler[T]) (ActionResult[T], error) {
	if !lookup.Found {
		return ActionResult[T]{Definition: lookup.Definition, Action: lookup.Action}, nil
	}
	if handler == nil {
		return ActionResult[T]{Definition: lookup.Definition, Action: lookup.Action, Found: true}, fmt.Errorf("商城固定流程动作处理器未配置")
	}
	output, err := handler(ctx, lookup.Action, request.Payload)
	return ActionResult[T]{Definition: lookup.Definition, Action: lookup.Action, Output: output, Found: true}, err
}

// actionKey 返回流程和动作类型的复合索引键。
func actionKey(flow FlowName, actionType string) string {
	return string(flow) + "\x00" + actionType
}

// actionNodeKey 返回 Eino Graph 中对应动作的节点名称。
func actionNodeKey(action Action) string {
	return fmt.Sprintf("%s_%s", action.Flow, action.Type)
}
