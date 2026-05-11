package assistant

// Message 表示 AI 助手上下文消息。
type Message struct {
	// Role 消息角色。
	Role string `json:"role"`
	// Content 消息文本内容。
	Content string `json:"content"`
}

// Attachment 表示 AI 助手附件。
type Attachment struct {
	// Name 附件名称。
	Name string `json:"name"`
	// Size 附件大小。
	Size int64 `json:"size"`
	// URL 附件地址。
	URL string `json:"url"`
	// MIMEType 附件 MIME 类型。
	MIMEType string `json:"mimeType"`
	// Content 附件文本内容。
	Content string `json:"content"`
	// Bytes 附件原始字节。
	Bytes []byte `json:"-"`
}

// Response 表示 AI 助手回复结果。
type Response struct {
	// Content 回复文本内容。
	Content string `json:"content"`
	// TokenUsage 本次调用 token 消耗。
	TokenUsage int64 `json:"tokenUsage"`
	// Source 回复来源。
	Source string `json:"source"`
	// Model 使用的模型名称。
	Model string `json:"model"`
	// Fallback 是否为降级回复。
	Fallback bool `json:"fallback"`
	// FallbackReason 降级原因。
	FallbackReason string `json:"fallbackReason"`
}

// RuntimeInput 表示 AI 助手运行时输入。
type RuntimeInput struct {
	Terminal     string
	Scene        string
	UserName     string
	SessionTitle string
	SessionID    string
	Summary      string
	Content      string
	Attachments  []Attachment
	History      []Message
}
