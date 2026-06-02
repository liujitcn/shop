package assistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode"

	"shop/pkg/agent/provider"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/openai/openai-go/v3/responses"
)

const (
	maxModelToolsPerRequest    = 1
	minToolMatchScore          = 2
	maxToolQueryAttachmentText = 800
)

const aiAssistantInstruction = `你是一个通用 AI 助手，可以自然、友好、准确地回答用户提出的各种问题。
回复要求：
1. 优先直接回答当前问题，不因为问题不属于商城系统而拒绝。
2. 可以处理通用知识、日常问答、写作润色、代码说明、方案整理、思路分析等请求。
3. 如果用户提供了附件、历史上下文或系统上下文，可以按需参考。
4. 涉及商城、订单、用户、字典、配置、报表等系统内私有数据时，优先调用当前终端可用的内部工具获取真实数据。
5. 内部工具不匹配、工具无结果、或用户问题属于公开实时信息时，可以继续使用联网搜索。
6. 不要编造当前上下文和工具结果没有提供的私有系统数据、精确数值或操作结果。
7. 用中文回复，保持清晰自然，适合直接展示在聊天窗口。`

// Runtime 封装流式 AI 助手运行时。
//
// Runtime 只负责把业务层准备好的输入组装为 Eino 消息并交给模型执行，不直接处理数据库、
// OSS、鉴权或前端协议。这样 AI 助手链路可以把“业务准备”和“模型运行”分开维护。
type Runtime struct {
	client     *provider.ResponsesClient
	adminTools []tool.InvokableTool
	appTools   []tool.InvokableTool
}

// NewRuntime 创建 AI 助手运行时。
func NewRuntime(client *provider.ResponsesClient) *Runtime {
	return &Runtime{
		client: client,
	}
}

// SetTools 设置默认 AI 助手可执行的 Eino 工具列表。
func (r *Runtime) SetTools(values []tool.InvokableTool) {
	if r == nil {
		return
	}
	r.adminTools = append([]tool.InvokableTool(nil), values...)
	r.appTools = append([]tool.InvokableTool(nil), values...)
}

// Enabled 判断 AI 助手运行时是否可用。
func (r *Runtime) Enabled() bool {
	return r != nil && r.client != nil && r.client.AgenticModel != nil
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
// 该方法用于普通 RPC 或非流式调用：先构建带历史上下文的 Eino 消息列表，
// 再等待模型完整回复。
func (r *Runtime) Run(ctx context.Context, input RuntimeInput) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("ai assistant client is not configured")
	}
	output, err := r.runGenerate(ctx, input, r.buildMessages(input))
	if err != nil {
		return nil, err
	}
	return r.buildResponse(output), nil
}

// RunStream 使用流式模式运行助手。
//
// 该方法用于管理端 direct SSE：模型返回文本片段时会透传给 onDelta，
// 最终仍返回完整回复供业务层落库。
func (r *Runtime) RunStream(ctx context.Context, input RuntimeInput, onDelta func(string)) (*Response, error) {
	if !r.Enabled() {
		return nil, fmt.Errorf("ai assistant client is not configured")
	}
	var streamOptions []model.Option
	messages := r.buildMessages(input)
	if len(r.toolInfos(ctx, input)) > 0 {
		var usedTools bool
		var err error
		messages, usedTools, err = r.runToolCalls(ctx, input, messages)
		if err != nil {
			return nil, err
		}
		if !usedTools {
			streamOptions = r.modelOptions(ctx, input)
		}
	} else {
		streamOptions = r.modelOptions(ctx, input)
	}
	reader, err := r.client.Stream(ctx, messages, streamOptions...)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var content strings.Builder
	chunks := make([]*schema.AgenticMessage, 0)
	for {
		chunk, recvErr := reader.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			return nil, recvErr
		}
		if chunk == nil {
			continue
		}
		chunks = append(chunks, chunk)
		text := runtimeMessageText(chunk)
		if text == "" {
			continue
		}
		content.WriteString(text)
		if onDelta != nil {
			onDelta(text)
		}
	}
	finalMessage, err := schema.ConcatAgenticMessages(chunks)
	if err != nil {
		return nil, err
	}
	if finalMessage == nil {
		if content.Len() == 0 {
			return nil, fmt.Errorf("ai assistant response is empty")
		}
		finalMessage = assistantAgenticMessage(content.String())
	}
	if runtimeMessageText(finalMessage) == "" && content.Len() > 0 {
		finalMessage = assistantAgenticMessage(content.String())
	}
	return r.buildResponse(finalMessage), nil
}

// runGenerate 执行非流式模型调用，并在需要时继续执行工具回填。
func (r *Runtime) runGenerate(ctx context.Context, input RuntimeInput, messages []*schema.AgenticMessage) (*schema.AgenticMessage, error) {
	currentMessages := append([]*schema.AgenticMessage(nil), messages...)
	output, err := r.client.Generate(ctx, currentMessages, r.modelOptions(ctx, input)...)
	if err != nil {
		return nil, err
	}
	toolCalls := agenticToolCalls(output)
	if len(toolCalls) == 0 {
		return output, nil
	}
	currentMessages = append(currentMessages, output)
	toolOutputs := r.executeToolCalls(ctx, input, toolCalls)
	currentMessages = append(currentMessages, toolOutputs...)
	return r.client.Generate(ctx, currentMessages)
}

// runToolCalls 以 stateless 方式执行模型工具调用。
func (r *Runtime) runToolCalls(ctx context.Context, input RuntimeInput, messages []*schema.AgenticMessage) ([]*schema.AgenticMessage, bool, error) {
	currentMessages := append([]*schema.AgenticMessage(nil), messages...)
	output, err := r.client.Generate(ctx, currentMessages, r.modelOptions(ctx, input)...)
	if err != nil {
		return nil, false, err
	}
	toolCalls := agenticToolCalls(output)
	if len(toolCalls) == 0 {
		return currentMessages, false, nil
	}
	currentMessages = append(currentMessages, output)
	toolOutputs := r.executeToolCalls(ctx, input, toolCalls)
	currentMessages = append(currentMessages, toolOutputs...)
	return currentMessages, true, nil
}

// executeToolCalls 执行一组 Eino 工具调用并构造 tool message。
func (r *Runtime) executeToolCalls(ctx context.Context, input RuntimeInput, calls []schema.ToolCall) []*schema.AgenticMessage {
	toolMap := r.toolMap(ctx, input)
	messages := make([]*schema.AgenticMessage, 0, len(calls))
	for _, call := range calls {
		content := r.executeToolCall(ctx, toolMap, call)
		messages = append(messages, functionToolResultAgenticMessage(call.ID, call.Function.Name, content))
	}
	return messages
}

// executeToolCall 执行单个工具调用并把结果转成模型可消费的字符串。
func (r *Runtime) executeToolCall(ctx context.Context, toolMap map[string]tool.InvokableTool, call schema.ToolCall) string {
	item := toolMap[call.Function.Name]
	if item == nil {
		return marshalToolError(fmt.Sprintf("tool %s is not available", call.Function.Name))
	}
	output, err := item.InvokableRun(ctx, call.Function.Arguments)
	if err != nil {
		return marshalToolError(err.Error())
	}
	if output == "" {
		return "{}"
	}
	return output
}

// modelOptions 构造当前模型调用需要携带的 Eino 工具定义。
func (r *Runtime) modelOptions(ctx context.Context, input RuntimeInput) []model.Option {
	toolInfos := r.toolInfos(ctx, input)
	if len(toolInfos) == 0 {
		return responsesServerToolOptions()
	}
	options := []model.Option{model.WithTools(toolInfos)}
	options = append(options, responsesServerToolOptions()...)
	return options
}

// toolInfos 收集可传给模型的工具定义。
func (r *Runtime) toolInfos(ctx context.Context, input RuntimeInput) []*schema.ToolInfo {
	tools := r.terminalTools(input.Terminal)
	if len(tools) == 0 {
		return nil
	}
	infos := make([]*schema.ToolInfo, 0, len(tools))
	seen := make(map[string]struct{}, len(tools))
	for _, item := range tools {
		if item == nil {
			continue
		}
		info, err := item.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		if _, ok := seen[info.Name]; ok {
			continue
		}
		seen[info.Name] = struct{}{}
		infos = append(infos, info)
	}
	return selectToolInfos(input, infos)
}

// selectToolInfos 从当前终端完整工具池中挑选本轮请求相关工具。
func selectToolInfos(input RuntimeInput, infos []*schema.ToolInfo) []*schema.ToolInfo {
	if len(infos) <= maxModelToolsPerRequest {
		return infos
	}
	terms := toolQueryTerms(input)
	if len(terms) == 0 {
		return nil
	}
	scoredTools := make([]scoredToolInfo, 0, len(infos))
	for index, info := range infos {
		score := scoreToolInfo(info, terms)
		if score < minToolMatchScore {
			continue
		}
		scoredTools = append(scoredTools, scoredToolInfo{
			info:  info,
			score: score,
			index: index,
		})
	}
	if len(scoredTools) == 0 {
		return nil
	}
	sort.SliceStable(scoredTools, func(i, j int) bool {
		if scoredTools[i].score == scoredTools[j].score {
			return scoredTools[i].index < scoredTools[j].index
		}
		return scoredTools[i].score > scoredTools[j].score
	})
	limit := maxModelToolsPerRequest
	if len(scoredTools) < limit {
		limit = len(scoredTools)
	}
	result := make([]*schema.ToolInfo, 0, limit)
	for _, item := range scoredTools[:limit] {
		result = append(result, item.info)
	}
	return result
}

type scoredToolInfo struct {
	info  *schema.ToolInfo
	score int
	index int
}

// toolQueryTerms 提取本轮问题用于匹配工具名称和描述的关键词。
func toolQueryTerms(input RuntimeInput) []string {
	textParts := []string{input.Content, input.SessionTitle, input.Summary}
	for _, item := range input.Attachments {
		textParts = append(textParts, item.Name)
		if len(item.Content) > maxToolQueryAttachmentText {
			textParts = append(textParts, item.Content[:maxToolQueryAttachmentText])
			continue
		}
		textParts = append(textParts, item.Content)
	}
	raw := strings.ToLower(strings.Join(textParts, " "))
	seen := map[string]struct{}{}
	terms := make([]string, 0, 16)
	for _, term := range splitToolQueryTerms(raw) {
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	return terms
}

// splitToolQueryTerms 同时兼容中文短语和英文/数字工具名。
func splitToolQueryTerms(raw string) []string {
	values := make([]string, 0, 16)
	var word strings.Builder
	var hans strings.Builder
	flushWord := func() {
		if word.Len() == 0 {
			return
		}
		if value := word.String(); len([]rune(value)) > 1 {
			values = append(values, value)
		}
		word.Reset()
	}
	flushHans := func() {
		if hans.Len() == 0 {
			return
		}
		values = append(values, chineseNgrams(hans.String())...)
		hans.Reset()
	}
	for _, r := range raw {
		if unicode.Is(unicode.Han, r) {
			flushWord()
			hans.WriteRune(r)
			continue
		}
		flushHans()
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
			continue
		}
		flushWord()
	}
	flushWord()
	flushHans()
	return values
}

// chineseNgrams 把连续中文切成短语，提升“商品列表”对“查询商品信息列表”的命中率。
func chineseNgrams(value string) []string {
	runes := []rune(value)
	if len(runes) == 0 {
		return nil
	}
	if len(runes) <= 2 {
		return []string{value}
	}
	values := make([]string, 0, len(runes)*2)
	if len(runes) <= 6 {
		values = append(values, value)
	}
	for size := 2; size <= 4; size++ {
		if len(runes) < size {
			break
		}
		for i := 0; i+size <= len(runes); i++ {
			values = append(values, string(runes[i:i+size]))
		}
	}
	return values
}

// scoreToolInfo 计算工具与用户问题的相关性。
func scoreToolInfo(info *schema.ToolInfo, terms []string) int {
	if info == nil {
		return 0
	}
	text := strings.ToLower(info.Name + " " + info.Desc)
	score := 0
	for _, term := range terms {
		if !strings.Contains(text, term) {
			continue
		}
		score += len([]rune(term))
		if strings.Contains(info.Name, term) {
			score += 3
		}
	}
	return score
}

// toolMap 按工具名构造本地执行索引。
func (r *Runtime) toolMap(ctx context.Context, input RuntimeInput) map[string]tool.InvokableTool {
	tools := r.terminalTools(input.Terminal)
	result := make(map[string]tool.InvokableTool, len(tools))
	if r == nil {
		return result
	}
	for _, item := range tools {
		if item == nil {
			continue
		}
		info, err := item.Info(ctx)
		if err != nil || info == nil || info.Name == "" {
			continue
		}
		if _, ok := result[info.Name]; ok {
			continue
		}
		result[info.Name] = item
	}
	return result
}

// terminalTools 按终端选择当前智能体可用工具。
func (r *Runtime) terminalTools(terminal string) []tool.InvokableTool {
	if r == nil {
		return nil
	}
	if terminal == "app" {
		return r.appTools
	}
	return r.adminTools
}

// buildMessages 构建当前轮次发送给 Eino 模型的消息列表。
func (r *Runtime) buildMessages(input RuntimeInput) []*schema.AgenticMessage {
	messages := []*schema.AgenticMessage{schema.SystemAgenticMessage(r.resolvePrompt(input))}
	for _, item := range input.History {
		if item.Content == "" {
			continue
		}
		messages = append(messages, buildHistoryMessage(item.Role, item.Content))
	}
	messages = append(messages, r.buildUserMessage(input))
	return messages
}

// buildHistoryMessage 按消息角色追加历史上下文。
func buildHistoryMessage(role string, content string) *schema.AgenticMessage {
	// 历史角色需要还原到 Eino 原生消息类型，未知角色按用户消息处理以兼容旧数据。
	switch strings.ToLower(role) {
	case RoleAssistant:
		return assistantAgenticMessage(content)
	case RoleSystem:
		return schema.SystemAgenticMessage(content)
	default:
		return schema.UserAgenticMessage(content)
	}
}

// resolvePrompt 渲染 AI 助手提示词。
func (r *Runtime) resolvePrompt(input RuntimeInput) string {
	lines := []string{
		aiAssistantInstruction,
		"",
		"当前会话：",
		fmt.Sprintf("- 终端：%s", input.Terminal),
		fmt.Sprintf("- 用户：%s", input.UserName),
		fmt.Sprintf("- 标题：%s", input.SessionTitle),
		fmt.Sprintf("- 摘要：%s", input.Summary),
	}
	if len(input.Attachments) > 0 {
		lines = append(lines, "", "用户本轮提供了附件，附件内容会出现在消息中，回答时按需参考。")
	}
	return strings.Join(lines, "\n")
}

// buildUserMessage 构建当前轮次发送给模型的用户消息。
func (r *Runtime) buildUserMessage(input RuntimeInput) *schema.AgenticMessage {
	content := input.Content
	attachmentLines := make([]string, 0, len(input.Attachments)*2)
	imageBlocks := make([]*schema.ContentBlock, 0, len(input.Attachments))

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
			imageBlocks = append(imageBlocks, buildImageInputBlock(item.Bytes, cleanMIMEType))
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

	if len(attachmentLines) == 0 && len(imageBlocks) == 0 {
		return schema.UserAgenticMessage(content)
	}
	return buildUserMessageParts(content, attachmentLines, imageBlocks)
}

// buildImageInputBlock 构造 Eino 图片输入片段。
func buildImageInputBlock(data []byte, mimeType string) *schema.ContentBlock {
	base64Data := base64.StdEncoding.EncodeToString(data)
	return schema.NewContentBlock(&schema.UserInputImage{
		Base64Data: base64Data,
		MIMEType:   mimeType,
		Detail:     schema.ImageURLDetailAuto,
	})
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

// buildUserMessageParts 合并文本提示和图片输入，形成 Eino 用户消息。
func buildUserMessageParts(content string, attachmentLines []string, imageBlocks []*schema.ContentBlock) *schema.AgenticMessage {
	textParts := []string{content}
	// 附件说明统一追加到用户正文后，避免模型误以为附件内容来自系统指令。
	if len(attachmentLines) > 0 {
		textParts = append(textParts, "本轮消息附带以下附件内容，请在回答时按需参考：", strings.Join(attachmentLines, "\n\n"))
	}

	contentBlocks := make([]*schema.ContentBlock, 0, 1+len(imageBlocks))
	// 用户可能只上传图片不输入文本，空文本不传入 message parts。
	if text := strings.Join(textParts, "\n\n"); text != "" {
		contentBlocks = append(contentBlocks, schema.NewContentBlock(&schema.UserInputText{Text: text}))
	}
	contentBlocks = append(contentBlocks, imageBlocks...)
	return &schema.AgenticMessage{
		Role:          schema.AgenticRoleTypeUser,
		ContentBlocks: contentBlocks,
	}
}

// buildResponse 将 Eino 消息收敛为业务层统一回复结构。
func (r *Runtime) buildResponse(message *schema.AgenticMessage) *Response {
	tokenUsage := int64(0)
	if message != nil && message.ResponseMeta != nil && message.ResponseMeta.TokenUsage != nil {
		tokenUsage = int64(message.ResponseMeta.TokenUsage.TotalTokens)
	}
	return &Response{
		Content:        runtimeMessageText(message),
		TokenUsage:     tokenUsage,
		Source:         "llm",
		Model:          r.Model(),
		Fallback:       false,
		FallbackReason: "",
	}
}

// runtimeMessageText 提取 Eino 消息中的文本内容。
func runtimeMessageText(message *schema.AgenticMessage) string {
	if message == nil {
		return ""
	}
	parts := make([]string, 0, len(message.ContentBlocks))
	for _, item := range message.ContentBlocks {
		if item == nil {
			continue
		}
		switch {
		case item.AssistantGenText != nil && item.AssistantGenText.Text != "":
			parts = append(parts, item.AssistantGenText.Text)
		case item.UserInputText != nil && item.UserInputText.Text != "":
			parts = append(parts, item.UserInputText.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// marshalToolError 将工具执行错误转换为稳定 JSON 文本。
func marshalToolError(message string) string {
	raw, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		return `{"error":"tool execution failed"}`
	}
	return string(raw)
}

// agenticToolCalls 从 Agentic 消息中提取函数工具调用。
func agenticToolCalls(message *schema.AgenticMessage) []schema.ToolCall {
	if message == nil {
		return nil
	}
	calls := make([]schema.ToolCall, 0, len(message.ContentBlocks))
	for _, item := range message.ContentBlocks {
		if item == nil || item.FunctionToolCall == nil {
			continue
		}
		call := item.FunctionToolCall
		calls = append(calls, schema.ToolCall{
			ID: call.CallID,
			Function: schema.FunctionCall{
				Name:      call.Name,
				Arguments: call.Arguments,
			},
		})
	}
	return calls
}

// functionToolResultAgenticMessage 构造函数工具执行结果消息。
func functionToolResultAgenticMessage(callID string, name string, content string) *schema.AgenticMessage {
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeUser,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.FunctionToolResult{
				CallID: callID,
				Name:   name,
				Content: []*schema.FunctionToolResultContentBlock{
					{
						Type: schema.FunctionToolResultContentBlockTypeText,
						Text: &schema.UserInputText{Text: content},
					},
				},
			}),
		},
	}
}

// assistantAgenticMessage 构造助手文本消息。
func assistantAgenticMessage(content string) *schema.AgenticMessage {
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			schema.NewContentBlock(&schema.AssistantGenText{Text: content}),
		},
	}
}

// responsesServerToolOptions 构造 Responses 内置服务端工具选项。
func responsesServerToolOptions() []model.Option {
	return []model.Option{
		agenticopenai.WithResponsesServerTools([]*agenticopenai.ResponsesServerToolConfig{
			{
				WebSearch: &responses.WebSearchToolParam{
					Type: responses.WebSearchToolTypeWebSearch,
				},
			},
		}),
	}
}
