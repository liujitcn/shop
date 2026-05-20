package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"strings"

	basev1 "shop/api/gen/go/base/v1"
	commonv1 "shop/api/gen/go/common/v1"
)

const (
	// TerminalAdmin 表示管理端。
	TerminalAdmin int32 = 2
	// TerminalApp 表示商城端。
	TerminalApp int32 = 1

	// RoleUser 表示用户消息。
	RoleUser = "user"
	// RoleAssistant 表示助手消息。
	RoleAssistant = "assistant"

	// KindText 表示普通文本消息。
	KindText = "text"

	previewSize = 18
)

// ReplyMeta 表示助手消息中的回复元信息。
type ReplyMeta struct {
	ReplySource    string `json:"reply_source"`
	Model          string `json:"model"`
	Fallback       bool   `json:"fallback"`
	FallbackReason string `json:"fallback_reason"`
}

type replyPayload struct {
	Content        string `json:"content"`
	ReplySource    string `json:"reply_source"`
	Model          string `json:"model"`
	Fallback       bool   `json:"fallback"`
	FallbackReason string `json:"fallback_reason"`
}

// NormalizeTerminal 规范化终端类型。
func NormalizeTerminal(terminal commonv1.Terminal) int32 {
	switch terminal {
	case commonv1.Terminal_TERMINAL_APP:
		return TerminalApp
	default:
		return TerminalAdmin
	}
}

// NormalizeTerminalString 规范化终端文本。
func NormalizeTerminalString(terminal int32) string {
	switch terminal {
	case TerminalApp:
		return "app"
	default:
		return "admin"
	}
}

// NormalizeTerminalEnum 将终端整型值转换为 proto 枚举。
func NormalizeTerminalEnum(terminal int32) commonv1.Terminal {
	switch terminal {
	case TerminalApp:
		return commonv1.Terminal_TERMINAL_APP
	default:
		return commonv1.Terminal_TERMINAL_ADMIN
	}
}

// NormalizeAttachments 清理附件列表。
func NormalizeAttachments(values []*basev1.AiAssistantAttachment) []*basev1.AiAssistantAttachment {
	result := make([]*basev1.AiAssistantAttachment, 0, len(values))
	for _, item := range values {
		if item == nil {
			continue
		}
		name := strings.TrimSpace(item.GetName())
		if name == "" {
			name = "未命名附件"
		}
		result = append(result, &basev1.AiAssistantAttachment{
			Id:       strings.TrimSpace(item.GetId()),
			Name:     name,
			Size:     item.GetSize(),
			Url:      strings.TrimSpace(item.GetUrl()),
			MimeType: strings.TrimSpace(item.GetMimeType()),
		})
	}
	return result
}

// BuildDefaultSummary 生成默认会话摘要。
func BuildDefaultSummary() string {
	return "新对话"
}

// BuildDynamicSummary 根据当前问题更新会话摘要。
func BuildDynamicSummary(content string, attachments []*basev1.AiAssistantAttachment) string {
	preview := NormalizePreview(content)
	if preview == "" && len(attachments) > 0 {
		preview = fmt.Sprintf("%d 个附件", len(attachments))
	}
	if preview == "" {
		return BuildDefaultSummary()
	}
	return preview
}

// BuildUserContent 在只有附件时补一条可读提示。
func BuildUserContent(content string, attachments []*basev1.AiAssistantAttachment) string {
	if strings.TrimSpace(content) != "" {
		return strings.TrimSpace(content)
	}
	if len(attachments) == 0 {
		return ""
	}
	return "请结合附件内容继续分析"
}

// BuildFallbackReply 在未启用大模型时返回本地兜底文本。
func BuildFallbackReply(content string, attachments []*basev1.AiAssistantAttachment) string {
	if len(attachments) > 0 {
		return fmt.Sprintf("已收到你的问题和 %d 个附件，但当前大模型暂时不可用，无法生成完整回复，请稍后再试。", len(attachments))
	}
	return fmt.Sprintf("已收到你的问题：%s。但当前大模型暂时不可用，无法生成完整回复，请稍后再试。", NormalizePreview(content))
}

// NormalizePreview 截断摘要预览文本。
func NormalizePreview(content string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(content, "\n", " "))
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= previewSize {
		return trimmed
	}
	return string(runes[:previewSize]) + "..."
}

// DetectAttachmentMIME 规范化附件 MIME 类型。
func DetectAttachmentMIME(fileName string, rawMIMEType string) string {
	if strings.TrimSpace(rawMIMEType) != "" {
		return strings.TrimSpace(rawMIMEType)
	}
	extension := strings.ToLower(pathExt(fileName))
	switch extension {
	case ".md", ".markdown", ".log":
		return "text/plain; charset=utf-8"
	default:
		return mime.TypeByExtension(extension)
	}
}

// ExtractAttachmentText 提取文本类附件内容。
func ExtractAttachmentText(fileBytes []byte, mimeType string) string {
	if len(fileBytes) == 0 {
		return ""
	}
	cleanMIMEType := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(cleanMIMEType, "text/") || strings.Contains(cleanMIMEType, "json") || strings.Contains(cleanMIMEType, "xml") || strings.Contains(cleanMIMEType, "csv") {
		text := strings.TrimSpace(string(bytes.TrimSpace(fileBytes)))
		if len([]rune(text)) > 4000 {
			return string([]rune(text)[:4000])
		}
		return text
	}
	return ""
}

// MarshalReplyContent 序列化助手回复内容与元信息。
func MarshalReplyContent(response *Response) string {
	if response == nil {
		return ""
	}
	payload := replyPayload{
		Content:        strings.TrimSpace(response.Content),
		ReplySource:    strings.TrimSpace(response.Source),
		Model:          strings.TrimSpace(response.Model),
		Fallback:       response.Fallback,
		FallbackReason: strings.TrimSpace(response.FallbackReason),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return strings.TrimSpace(response.Content)
	}
	return string(raw)
}

// ParseReplyContent 解析助手回复正文。
func ParseReplyContent(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	payload := replyPayload{}
	err := json.Unmarshal([]byte(raw), &payload)
	if err != nil {
		return raw
	}
	return strings.TrimSpace(payload.Content)
}

// ParseReplyMeta 解析助手回复元信息。
func ParseReplyMeta(raw string) ReplyMeta {
	meta := ReplyMeta{}
	if strings.TrimSpace(raw) == "" {
		return meta
	}
	payload := replyPayload{}
	err := json.Unmarshal([]byte(raw), &payload)
	if err != nil {
		return meta
	}
	meta.ReplySource = strings.TrimSpace(payload.ReplySource)
	meta.Model = strings.TrimSpace(payload.Model)
	meta.Fallback = payload.Fallback
	meta.FallbackReason = strings.TrimSpace(payload.FallbackReason)
	return meta
}

// MarshalAttachments 序列化附件 JSON。
func MarshalAttachments(attachments []*basev1.AiAssistantAttachment) string {
	payload := make([]map[string]any, 0, len(attachments))
	for _, item := range attachments {
		payload = append(payload, map[string]any{
			"id":        item.GetId(),
			"name":      item.GetName(),
			"size":      item.GetSize(),
			"url":       item.GetUrl(),
			"mime_type": item.GetMimeType(),
		})
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// ParseAttachments 反序列化附件列表。
func ParseAttachments(raw string) []*basev1.AiAssistantAttachment {
	if strings.TrimSpace(raw) == "" {
		return []*basev1.AiAssistantAttachment{}
	}
	type attachmentPayload struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Size     int64  `json:"size"`
		URL      string `json:"url"`
		MIMEType string `json:"mime_type"`
	}
	values := make([]attachmentPayload, 0)
	err := json.Unmarshal([]byte(raw), &values)
	if err != nil {
		return []*basev1.AiAssistantAttachment{}
	}
	result := make([]*basev1.AiAssistantAttachment, 0, len(values))
	for _, item := range values {
		result = append(result, &basev1.AiAssistantAttachment{
			Id:       item.ID,
			Name:     item.Name,
			Size:     item.Size,
			Url:      item.URL,
			MimeType: item.MIMEType,
		})
	}
	return result
}

func pathExt(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 || index == len(name)-1 {
		return ""
	}
	return name[index:]
}
