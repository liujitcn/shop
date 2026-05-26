package assistant

import (
	"encoding/json"
	"fmt"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
)

// replyPayload 表示助手回复落库 JSON 结构。
type replyPayload struct {
	Content        string `json:"content"`
	ReplySource    string `json:"reply_source"`
	Model          string `json:"model"`
	Fallback       bool   `json:"fallback"`
	FallbackReason string `json:"fallback_reason"`
}

// BuildUserContent 生成用户消息落库正文。
func BuildUserContent(content string, attachments []*basev1.AiAssistantAttachment) string {
	// 有用户文本时保留文本作为主问题，附件内容通过附件字段独立保存。
	if content != "" {
		return content
	}
	// 文本和附件都为空时交给上层参数校验处理。
	if len(attachments) == 0 {
		return ""
	}
	return "请结合附件内容继续分析"
}

// BuildFallbackReply 生成模型不可用时的本地降级回复。
func BuildFallbackReply(content string, attachments []*basev1.AiAssistantAttachment) string {
	// 附件场景不回显文件名，避免降级文案过长或暴露不必要的文件路径。
	if len(attachments) > 0 {
		return fmt.Sprintf("已收到你的问题和 %d 个附件，但当前大模型暂时不可用，无法生成完整回复，请稍后再试。", len(attachments))
	}
	return fmt.Sprintf("已收到你的问题：%s。但当前大模型暂时不可用，无法生成完整回复，请稍后再试。", NormalizePreview(content))
}

// BuildDefaultSummary 生成新会话默认摘要。
func BuildDefaultSummary() string {
	return "新对话"
}

// BuildDynamicSummary 根据本轮用户文本或附件数量生成会话摘要。
func BuildDynamicSummary(content string, attachments []*basev1.AiAssistantAttachment) string {
	preview := NormalizePreview(content)
	// 只有附件没有文本时，用附件数量表达本轮会话主题。
	if preview == "" && len(attachments) > 0 {
		preview = fmt.Sprintf("%d 个附件", len(attachments))
	}
	// 兜底默认摘要，避免会话列表出现空白标题区域。
	if preview == "" {
		return BuildDefaultSummary()
	}
	return preview
}

// NormalizePreview 将用户输入整理为适合会话列表展示的短文本。
func NormalizePreview(content string) string {
	trimmed := strings.ReplaceAll(content, "\n", " ")
	// 空输入没有可展示摘要，交给调用方决定默认值。
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	// 按 rune 截断，避免中文内容被字节截断成乱码。
	if len(runes) <= previewSize {
		return trimmed
	}
	return string(runes[:previewSize]) + "..."
}

// MarshalReplyContent 序列化助手回复正文和元信息。
func MarshalReplyContent(response *Response) string {
	// nil 回复通常表示上游未生成内容，直接落空字符串避免 panic。
	if response == nil {
		return ""
	}
	payload := replyPayload{
		Content:        response.Content,
		ReplySource:    response.Source,
		Model:          response.Model,
		Fallback:       response.Fallback,
		FallbackReason: response.FallbackReason,
	}
	raw, err := json.Marshal(payload)
	// 极端情况下 JSON 序列化失败时保留正文，避免用户完全看不到回复。
	if err != nil {
		return response.Content
	}
	return string(raw)
}

// ParseReplyContent 从落库内容中解析助手回复正文。
func ParseReplyContent(raw string) string {
	// 空内容直接返回，兼容历史空消息。
	if raw == "" {
		return ""
	}
	payload := replyPayload{}
	err := json.Unmarshal([]byte(raw), &payload)
	// 兼容旧版本直接存纯文本的助手消息。
	if err != nil {
		return raw
	}
	return payload.Content
}

// ParseReplyMeta 从落库内容中解析助手回复元信息。
func ParseReplyMeta(raw string) ReplyMeta {
	meta := ReplyMeta{}
	// 空内容没有元信息，前端按普通助手消息展示。
	if raw == "" {
		return meta
	}
	payload := replyPayload{}
	err := json.Unmarshal([]byte(raw), &payload)
	// 纯文本历史消息没有元信息，返回零值即可。
	if err != nil {
		return meta
	}
	meta.ReplySource = payload.ReplySource
	meta.Model = payload.Model
	meta.Fallback = payload.Fallback
	meta.FallbackReason = payload.FallbackReason
	return meta
}
