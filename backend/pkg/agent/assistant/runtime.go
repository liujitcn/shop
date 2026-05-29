package assistant

import (
	"context"
	"fmt"
	"strings"

	"shop/pkg/agent/provider"

	"github.com/go-kratos/blades"
	summaryContext "github.com/go-kratos/blades/context/summary"
	"github.com/go-kratos/blades/tools"
)

const (
	stateTerminal     = "terminal"
	stateUserName     = "user_name"
	stateSessionTitle = "session_title"
	stateSummary      = "session_summary"
)

const aiAssistantInstruction = `你是一个通用 AI 助手，可以自然、友好、准确地回答用户提出的各种问题。
回复要求：
1. 优先直接回答当前问题，不因为问题不属于商城系统而拒绝。
2. 可以处理通用知识、日常问答、写作润色、代码说明、方案整理、思路分析等请求。
3. 如果用户提供了附件、历史上下文或系统上下文，可以按需参考。
4. 涉及商城、订单、用户、字典、配置、报表等系统内数据时，优先调用已提供的内部工具获取真实数据。
5. 内部工具不匹配、工具无结果、或用户问题属于公开实时信息时，可以继续使用联网搜索。
6. 不要编造当前上下文和工具结果没有提供的私有系统数据、精确数值或操作结果。
7. 用中文回复，保持清晰自然，适合直接展示在聊天窗口。`

// Runtime 封装流式 AI 助手运行时。
//
// Runtime 只负责把业务层准备好的输入交给 Blades Agent 执行，不直接处理数据库、
// OSS、鉴权或前端协议。这样 AI 助手链路可以把“业务准备”和“模型运行”分开维护。
type Runtime struct {
	client *provider.ResponsesClient
	tools  []tools.Tool
}

// NewRuntime 创建 AI 助手运行时。
func NewRuntime(client *provider.ResponsesClient) *Runtime {
	return &Runtime{
		client: client,
	}
}

// SetTools 设置 AI 助手可直接调用的生成工具列表。
func (r *Runtime) SetTools(values []tools.Tool) {
	if r == nil {
		return
	}
	r.tools = append([]tools.Tool(nil), values...)
}

// Enabled 判断 AI 助手运行时是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.client != nil && r.client.ModelProvider != nil
}

// Model 返回 AI 助手当前使用的模型名称。
func (r *Runtime) Model() string {
	if !r.Enabled() {
		return ""
	}
	return r.client.Name()
}

// Run 使用生成式模式运行助手。
//
// 该方法用于普通 RPC 或非流式调用：先构建带历史上下文的 Blades Session，
// 再创建当前轮用户消息，最后等待模型完整回复。
func (r *Runtime) Run(ctx context.Context, input RuntimeInput) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("ai assistant client is not configured")
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
	output, err = runner.Run(ctx, r.buildUserMessage(input), blades.WithSession(session))
	if err != nil {
		return nil, err
	}
	return r.buildResponse(output), nil
}

// RunStream 使用流式模式运行助手。
//
// 该方法用于管理端 direct SSE：模型返回未完成的 assistant 消息时会把文本片段
// 透传给 onDelta，最终仍返回完整回复供业务层落库。
func (r *Runtime) RunStream(ctx context.Context, input RuntimeInput, onDelta func(string)) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("ai assistant client is not configured")
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
	for output, runErr := range runner.RunStream(ctx, r.buildUserMessage(input), blades.WithSession(session)) {
		if runErr != nil {
			return nil, runErr
		}
		// 工具调用结果由 Runner 内部继续驱动，这里只把助手文本增量透传给前端。
		if output == nil || output.Role != blades.RoleAssistant {
			continue
		}
		finalMessage = output
		// 未完成消息表示流式增量；完成消息用于最终落库，不再重复推送给前端。
		if output.Status != blades.StatusCompleted && onDelta != nil {
			text := output.Text()
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

// buildSession 构建当前轮次的 Blades Session。
func (r *Runtime) buildSession(ctx context.Context, input RuntimeInput) (blades.Session, error) {
	session := blades.NewSession(
		blades.WithContextCompressor(summaryContext.NewContextCompressor(
			r.client,
			summaryContext.WithKeepRecent(8),
			summaryContext.WithBatchSize(12),
			summaryContext.WithMaxTokens(1800),
		)),
	)
	session.SetState(stateTerminal, input.Terminal)
	session.SetState(stateUserName, input.UserName)
	session.SetState(stateSessionTitle, input.SessionTitle)
	session.SetState(stateSummary, input.Summary)

	var err error
	for _, item := range input.History {
		content := item.Content
		// 空历史对模型没有帮助，跳过可减少上下文噪音。
		if content == "" {
			continue
		}
		err = appendHistoryMessage(ctx, session, item.Role, content)
		if err != nil {
			return nil, err
		}
	}
	return session, nil
}

// appendHistoryMessage 按消息角色追加历史上下文。
func appendHistoryMessage(ctx context.Context, session blades.Session, role string, content string) error {
	// 历史角色需要还原到 Blades 原生消息类型，未知角色按用户消息处理以兼容旧数据。
	switch strings.ToLower(role) {
	case RoleAssistant:
		return session.Append(ctx, blades.AssistantMessage(content))
	case RoleSystem:
		return session.Append(ctx, blades.SystemMessage(content))
	default:
		return session.Append(ctx, blades.UserMessage(content))
	}
}

// buildRunner 构建当前轮次的 AI 助手运行器。
func (r *Runtime) buildRunner(input RuntimeInput) (*blades.Runner, error) {
	var err error
	var agentInstance blades.Agent
	options := []blades.AgentOption{
		blades.WithModel(r.client),
		blades.WithContext(true),
		blades.WithOutputKey("assistant_reply"),
		blades.WithInstruction(r.resolvePromptTemplate(input)),
	}
	if len(r.tools) > 0 {
		options = append(options, blades.WithTools(r.tools...))
	}
	agentInstance, err = blades.NewAgent(
		"shop_ai_assistant",
		options...,
	)
	if err != nil {
		return nil, err
	}
	return blades.NewRunner(agentInstance), nil
}

// resolvePromptTemplate 渲染 AI 助手提示词模板。
func (r *Runtime) resolvePromptTemplate(input RuntimeInput) string {
	lines := []string{
		aiAssistantInstruction,
		"",
		"当前会话：",
		"- 终端：{{.terminal}}",
		"- 用户：{{.user_name}}",
		"- 标题：{{.session_title}}",
		"- 摘要：{{.session_summary}}",
	}
	if len(input.Attachments) > 0 {
		lines = append(lines, "", "用户本轮提供了附件，附件内容会出现在消息中，回答时按需参考。")
	}
	return strings.Join(lines, "\n")
}

// buildUserMessage 构建当前轮次发送给模型的用户消息。
func (r *Runtime) buildUserMessage(input RuntimeInput) *blades.Message {
	content := input.Content
	attachmentLines := make([]string, 0, len(input.Attachments)*2)
	imageParts := make([]blades.DataPart, 0, len(input.Attachments))

	for _, item := range input.Attachments {
		attachmentContent := item.Content
		name := normalizeAttachmentName(item.Name)
		cleanMIMEType := normalizeRuntimeMIMEType(item)

		// 图片附件走多模态输入，避免把本地 /shop 地址误当成公网图片 URL。
		if isRuntimeImageMIME(cleanMIMEType) {
			// 图片必须具备原始字节，才能作为多模态视觉输入传给模型。
			if len(item.Bytes) == 0 {
				continue
			}
			attachmentLines = append(attachmentLines, fmt.Sprintf("图片附件《%s》已作为视觉输入提供给模型。", name))
			imageParts = append(imageParts, blades.DataPart{
				Name:     name,
				Bytes:    item.Bytes,
				MIMEType: blades.MIMEType(cleanMIMEType),
			})
			continue
		}

		// 文本类附件优先拼入正文，保证模型能直接读取附件内容。
		if attachmentContent != "" {
			attachmentLines = append(attachmentLines, fmt.Sprintf("附件《%s》内容：\n%s", name, attachmentContent))
			continue
		}

		// 没有可读内容的附件仍保留文件元信息，模型至少能知道用户提供了什么文件。
		attachmentLines = append(attachmentLines, buildAttachmentDetailLine(name, item))
	}

	// 没有附件内容时保持普通文本消息，减少对 Blades 消息结构的额外包装。
	if len(attachmentLines) == 0 && len(imageParts) == 0 {
		return blades.UserMessage(content)
	}
	return blades.UserMessage(buildUserMessageParts(content, attachmentLines, imageParts)...)
}

// normalizeAttachmentName 规范化模型提示词中展示的附件名称。
func normalizeAttachmentName(name string) string {
	trimmed := name
	// 模型提示词里避免出现空附件名，便于用户和模型对齐附件引用。
	if trimmed == "" {
		return "未命名附件"
	}
	return trimmed
}

// normalizeRuntimeMIMEType 规范化运行时 MIME 类型。
func normalizeRuntimeMIMEType(item Attachment) string {
	cleanMIMEType := strings.ToLower(strings.SplitN(item.MIMEType, ";", 2)[0])
	// image/jpg 不是标准 MIME，统一转成模型更常见支持的 image/jpeg。
	if cleanMIMEType == "image/jpg" {
		return "image/jpeg"
	}
	// 已有 MIME 时不再从文件名推断，避免后缀与真实内容冲突。
	if cleanMIMEType != "" {
		return cleanMIMEType
	}

	// MIME 缺失时仅对图片后缀兜底推断，其他文件保持普通附件元信息。
	switch strings.ToLower(pathExt(item.Name)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

// isRuntimeImageMIME 判断 MIME 类型是否可作为图片输入。
func isRuntimeImageMIME(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

// buildAttachmentDetailLine 构造模型无法直接读取附件内容时的元信息说明。
func buildAttachmentDetailLine(name string, item Attachment) string {
	details := []string{fmt.Sprintf("附件《%s》", name)}
	// 类型、大小、地址都是给模型的弱提示，缺失时不强行补默认值。
	if item.MIMEType != "" {
		details = append(details, fmt.Sprintf("类型：%s", item.MIMEType))
	}
	if item.Size > 0 {
		details = append(details, fmt.Sprintf("大小：%d 字节", item.Size))
	}
	if item.URL != "" {
		details = append(details, fmt.Sprintf("地址：%s", item.URL))
	}
	return strings.Join(details, "，")
}

// buildUserMessageParts 合并文本提示和图片输入，形成 Blades 用户消息参数。
func buildUserMessageParts(content string, attachmentLines []string, imageParts []blades.DataPart) []any {
	textParts := []string{content}
	// 附件说明统一追加到用户正文后，避免模型误以为附件内容来自系统指令。
	if len(attachmentLines) > 0 {
		textParts = append(textParts, "本轮消息附带以下附件内容，请在回答时按需参考：", strings.Join(attachmentLines, "\n\n"))
	}

	messageParts := make([]any, 0, 1+len(imageParts))
	// 用户可能只上传图片不输入文本，空文本不传入 message parts。
	if text := strings.Join(textParts, "\n\n"); text != "" {
		messageParts = append(messageParts, text)
	}
	for _, item := range imageParts {
		messageParts = append(messageParts, item)
	}
	return messageParts
}

// buildResponse 将 Blades 消息收敛为业务层统一回复结构。
func (r *Runtime) buildResponse(message *blades.Message) *Response {
	content := ""
	if message != nil {
		content = message.Text()
	}
	return &Response{
		Content:        content,
		TokenUsage:     0,
		Source:         "llm",
		Model:          r.Model(),
		Fallback:       false,
		FallbackReason: "",
	}
}
