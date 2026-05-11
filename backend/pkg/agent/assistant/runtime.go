package assistant

import (
	"context"
	"fmt"
	"strings"

	configv1 "shop/api/gen/go/config/v1"
	"shop/pkg/agent/provider"

	"github.com/go-kratos/blades"
	summaryContext "github.com/go-kratos/blades/context/summary"
	bladesMemory "github.com/go-kratos/blades/memory"
	"github.com/go-kratos/blades/tools"
)

const (
	stateTerminal     = "terminal"
	stateScene        = "scene"
	stateUserName     = "user_name"
	stateSessionTitle = "session_title"
	stateSummary      = "session_summary"
)

// Runtime 封装流式 AI 助手智能体运行时。
type Runtime struct {
	client *provider.ChatClient
	prompt *configv1.Prompt
}

// NewRuntime 创建 AI 助手运行时。
func NewRuntime(client *provider.ChatClient, prompt *configv1.Prompt) *Runtime {
	return &Runtime{
		client: client,
		prompt: prompt,
	}
}

// Enabled 判断 AI 助手运行时是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.client != nil && r.client.Enabled()
}

// Model 返回 AI 助手当前使用的模型名称。
func (r *Runtime) Model() string {
	if r == nil || r.client == nil {
		return ""
	}
	return r.client.Model()
}

// Run 使用生成式模式运行助手。
func (r *Runtime) Run(ctx context.Context, input RuntimeInput) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("agent chat client is not configured")
	}
	var err error
	var session blades.Session
	session, err = r.buildSession(ctx, input)
	if err != nil {
		return nil, err
	}
	var runner *blades.Runner
	runner, err = r.buildRunner(input)
	if err != nil {
		return nil, err
	}
	var output *blades.Message
	output, err = runner.Run(ctx, blades.UserMessage(strings.TrimSpace(input.Content)), blades.WithSession(session))
	if err != nil {
		return nil, err
	}
	return r.buildResponse(output), nil
}

// RunStream 使用流式模式运行助手。
func (r *Runtime) RunStream(ctx context.Context, input RuntimeInput, onDelta func(string)) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("agent chat client is not configured")
	}
	var err error
	var session blades.Session
	session, err = r.buildSession(ctx, input)
	if err != nil {
		return nil, err
	}
	var runner *blades.Runner
	runner, err = r.buildRunner(input)
	if err != nil {
		return nil, err
	}
	var finalMessage *blades.Message
	for output, runErr := range runner.RunStream(ctx, blades.UserMessage(strings.TrimSpace(input.Content)), blades.WithSession(session)) {
		if runErr != nil {
			return nil, runErr
		}
		if output == nil || output.Role != blades.RoleAssistant {
			continue
		}
		finalMessage = output
		if output.Status != blades.StatusCompleted && onDelta != nil {
			text := strings.TrimSpace(output.Text())
			if text != "" {
				onDelta(text)
			}
		}
	}
	if finalMessage == nil {
		return nil, fmt.Errorf("ai assistant response is empty")
	}
	return r.buildResponse(finalMessage), nil
}

// buildRunner 构建当前轮次的智能体运行器。
func (r *Runtime) buildRunner(input RuntimeInput) (*blades.Runner, error) {
	var err error
	var memoryTool tools.Tool
	memoryTool, err = r.buildMemoryTool(input)
	if err != nil {
		return nil, err
	}
	var agentInstance blades.Agent
	agentInstance, err = blades.NewAgent(
		"shop_ai_assistant",
		blades.WithModel(r.client.Provider()),
		blades.WithContext(true),
		blades.WithOutputKey("assistant_reply"),
		blades.WithInstruction(r.resolvePromptTemplate(input)),
		blades.WithTools(memoryTool),
	)
	if err != nil {
		return nil, err
	}
	return blades.NewRunner(agentInstance), nil
}

// buildSession 构建当前轮次的 Blades Session。
func (r *Runtime) buildSession(ctx context.Context, input RuntimeInput) (blades.Session, error) {
	session := blades.NewSession(
		blades.WithContextCompressor(summaryContext.NewContextCompressor(
			r.client.Provider(),
			summaryContext.WithKeepRecent(8),
			summaryContext.WithBatchSize(12),
			summaryContext.WithMaxTokens(1800),
		)),
	)
	session.SetState(stateTerminal, strings.TrimSpace(input.Terminal))
	session.SetState(stateScene, strings.TrimSpace(input.Scene))
	session.SetState(stateUserName, strings.TrimSpace(input.UserName))
	session.SetState(stateSessionTitle, strings.TrimSpace(input.SessionTitle))
	session.SetState(stateSummary, strings.TrimSpace(input.Summary))
	for _, item := range input.History {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(item.Role)) {
		case RoleAssistant:
			if err := session.Append(ctx, blades.AssistantMessage(content)); err != nil {
				return nil, err
			}
		case "system":
			if err := session.Append(ctx, blades.SystemMessage(content)); err != nil {
				return nil, err
			}
		default:
			if err := session.Append(ctx, blades.UserMessage(content)); err != nil {
				return nil, err
			}
		}
	}
	return session, nil
}

// buildMemoryTool 构建当前轮次的附件记忆工具。
func (r *Runtime) buildMemoryTool(input RuntimeInput) (tools.Tool, error) {
	store := bladesMemory.NewInMemoryStore()
	for _, item := range input.Attachments {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		if err := store.AddMemory(context.Background(), &bladesMemory.Memory{
			Content: blades.UserMessage(fmt.Sprintf("附件《%s》内容：\n%s", item.Name, content)),
			Metadata: map[string]any{
				"name": item.Name,
				"url":  item.URL,
			},
		}); err != nil {
			return nil, err
		}
	}
	return bladesMemory.NewMemoryTool(store)
}

// buildResponse 收敛当前轮次的助手回复。
func (r *Runtime) buildResponse(message *blades.Message) *Response {
	content := ""
	if message != nil {
		content = strings.TrimSpace(message.Text())
	}
	return &Response{
		Content:    content,
		TokenUsage: 0,
		Source:     "llm",
		Model:      r.Model(),
		Fallback:   false,
	}
}

// resolvePromptText 读取 AI 助手提示词文本。
func (r *Runtime) resolvePromptText() string {
	if r == nil || r.prompt == nil {
		return ""
	}
	return strings.TrimSpace(r.prompt.GetAiAssistant())
}

// resolvePromptTemplate 渲染 AI 助手提示词模板。
func (r *Runtime) resolvePromptTemplate(input RuntimeInput) string {
	promptText := r.resolvePromptText()
	if promptText == "" {
		promptText = "你是商城管理后台 AI 助手。请结合会话状态、记忆和当前提问，给出简洁、准确、可执行的回答。"
	}
	lines := []string{
		promptText,
		"",
		"补充约束（优先级高于上方可能存在的泛化限制表述）：",
		"1. 没有命中系统工具时，仍然默认使用模型能力直接回答，不要机械拒绝。",
		"2. 如果问题属于通用知识、公开常识、日常问答、文案润色、方案整理、思路分析等非系统专属内容，可以直接正常回答。",
		"3. 不要仅因为问题不属于商城系统内部模块，就直接回复“我只负责系统内问题”或类似拒绝文案。",
		"4. 如果问题属于天气、新闻、股价、实时交通、实时汇率、实时比赛结果等强实时信息，而当前上下文没有提供实时工具结果，你可以继续回答，但必须明确说明这不是实时查询结果，只能基于常识、经验或历史规律给出参考。",
		"5. 只有在明显缺少关键事实、容易误导、涉及高风险建议或需要实时结果却无法确认时，才说明边界与不确定性。",
		"",
		"当前会话状态：",
		"- 终端：{{.terminal}}",
		"- 场景：{{.scene}}",
		"- 用户：{{.user_name}}",
		"- 会话标题：{{.session_title}}",
		"- 会话摘要：{{.session_summary}}",
		"",
		"工作要求：",
		"1. 优先直接回答当前问题。",
		"2. 当附件记忆能补充事实时，自主使用 Memory 工具检索。",
		"3. 回答以纯文本为主，不输出工具调用过程说明。",
		"4. 若信息不足，明确指出缺少什么，不编造业务事实。",
		"5. 如果当前问题同时包含系统上下文和通用问题，先回答用户最关心的问题，再补充系统侧可落地建议。",
	}
	if len(input.Attachments) > 0 {
		lines = append(lines, "", "本轮输入还附带了可检索附件记忆。")
	}
	return strings.Join(lines, "\n")
}
