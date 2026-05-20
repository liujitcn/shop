package assistant

import (
	"context"
	"fmt"
	"strings"

	configv1 "shop/api/gen/go/config/v1"
	"shop/pkg/agent/provider"

	"github.com/go-kratos/blades"
	summaryContext "github.com/go-kratos/blades/context/summary"
)

const (
	stateTerminal     = "terminal"
	stateUserName     = "user_name"
	stateSessionTitle = "session_title"
	stateSummary      = "session_summary"
)

// Runtime 封装流式 AI 助手运行时。
type Runtime struct {
	client *provider.ResponsesClient
	prompt *configv1.Prompt
}

// NewRuntime 创建 AI 助手运行时。
func NewRuntime(client *provider.ResponsesClient, prompt *configv1.Prompt) *Runtime {
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
		if output == nil || output.Role != blades.RoleAssistant {
			continue
		}
		finalMessage = output
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

// buildRunner 构建当前轮次的 AI 助手运行器。
func (r *Runtime) buildRunner(input RuntimeInput) (*blades.Runner, error) {
	var err error
	var agentInstance blades.Agent
	agentInstance, err = blades.NewAgent(
		"shop_ai_assistant",
		blades.WithModel(r.client.Provider()),
		blades.WithContext(true),
		blades.WithOutputKey("assistant_reply"),
		blades.WithInstruction(r.resolvePromptTemplate(input)),
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
	session.SetState(stateUserName, strings.TrimSpace(input.UserName))
	session.SetState(stateSessionTitle, strings.TrimSpace(input.SessionTitle))
	session.SetState(stateSummary, strings.TrimSpace(input.Summary))
	var err error
	for _, item := range input.History {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(item.Role)) {
		case RoleAssistant:
			err = session.Append(ctx, blades.AssistantMessage(content))
			if err != nil {
				return nil, err
			}
		case "system":
			err = session.Append(ctx, blades.SystemMessage(content))
			if err != nil {
				return nil, err
			}
		default:
			err = session.Append(ctx, blades.UserMessage(content))
			if err != nil {
				return nil, err
			}
		}
	}
	return session, nil
}

// buildUserMessage 构建当前轮次发送给模型的用户消息。
func (r *Runtime) buildUserMessage(input RuntimeInput) *blades.Message {
	content := strings.TrimSpace(input.Content)
	attachmentLines := make([]string, 0, len(input.Attachments)*2)
	imageParts := make([]blades.DataPart, 0, len(input.Attachments))
	for _, item := range input.Attachments {
		attachmentContent := strings.TrimSpace(item.Content)
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = "未命名附件"
		}
		cleanMIMEType := strings.ToLower(strings.TrimSpace(strings.SplitN(item.MIMEType, ";", 2)[0]))
		if cleanMIMEType == "image/jpg" {
			cleanMIMEType = "image/jpeg"
		}
		if cleanMIMEType == "" {
			switch strings.ToLower(pathExt(item.Name)) {
			case ".png":
				cleanMIMEType = "image/png"
			case ".jpg", ".jpeg":
				cleanMIMEType = "image/jpeg"
			case ".webp":
				cleanMIMEType = "image/webp"
			case ".gif":
				cleanMIMEType = "image/gif"
			}
		}
		switch cleanMIMEType {
		case "image/png", "image/jpeg", "image/webp", "image/gif":
			if len(item.Bytes) == 0 {
				break
			}
			attachmentLines = append(attachmentLines, fmt.Sprintf("图片附件《%s》已作为视觉输入提供给模型。", name))
			imageParts = append(imageParts, blades.DataPart{
				Name:     name,
				Bytes:    item.Bytes,
				MIMEType: blades.MIMEType(cleanMIMEType),
			})
			continue
		}
		if attachmentContent != "" {
			attachmentLines = append(attachmentLines, fmt.Sprintf("附件《%s》内容：\n%s", name, attachmentContent))
			continue
		}
		details := []string{fmt.Sprintf("附件《%s》", name)}
		if item.MIMEType != "" {
			details = append(details, fmt.Sprintf("类型：%s", item.MIMEType))
		}
		if item.Size > 0 {
			details = append(details, fmt.Sprintf("大小：%d 字节", item.Size))
		}
		if strings.TrimSpace(item.URL) != "" {
			details = append(details, fmt.Sprintf("地址：%s", strings.TrimSpace(item.URL)))
		}
		attachmentLines = append(attachmentLines, strings.Join(details, "，"))
	}
	if len(attachmentLines) == 0 && len(imageParts) == 0 {
		return blades.UserMessage(content)
	}
	textParts := []string{content}
	if len(attachmentLines) > 0 {
		textParts = append(textParts, "本轮消息附带以下附件内容，请在回答时按需参考：", strings.Join(attachmentLines, "\n\n"))
	}
	messageParts := make([]any, 0, 1+len(imageParts))
	if text := strings.TrimSpace(strings.Join(textParts, "\n\n")); text != "" {
		messageParts = append(messageParts, text)
	}
	for _, item := range imageParts {
		messageParts = append(messageParts, item)
	}
	return blades.UserMessage(messageParts...)
}

// buildResponse 收敛当前轮次的助手回复。
func (r *Runtime) buildResponse(message *blades.Message) *Response {
	content := ""
	if message != nil {
		content = strings.TrimSpace(message.Text())
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

// resolvePromptTemplate 渲染 AI 助手提示词模板。
func (r *Runtime) resolvePromptTemplate(input RuntimeInput) string {
	promptText := ""
	if r.prompt != nil {
		promptText = strings.TrimSpace(r.prompt.GetAiAssistant())
	}
	if promptText == "" {
		promptText = "你是一个通用 AI 助手，直接、自然、准确地回答用户的问题。"
	}
	lines := []string{
		promptText,
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
