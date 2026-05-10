package llm

import "encoding/json"

// CommentReviewRequest 表示评论图文审核请求。
type CommentReviewRequest struct {
	// GoodsName 商品名称快照，用于判断评价内容是否与商品相关。
	GoodsName string `json:"goodsName"`
	// SKUDesc 商品规格描述快照，用于辅助判断尺码、颜色、规格等评价内容。
	SKUDesc string `json:"skuDesc"`
	// Content 评价或讨论文本内容。
	Content string `json:"content"`
	// ImageURLs 评价图片地址列表，用于多模态审核。
	ImageURLs []string `json:"imageUrls"`
	// ImageData 评价图片字节列表，用于审核本地或非公网图片。
	ImageData []CommentReviewImageData `json:"-"`
}

// CommentReviewImageData 表示评论审核使用的图片字节数据。
type CommentReviewImageData struct {
	// Name 图片名称，用于标识多模态输入。
	Name string
	// Bytes 图片字节内容。
	Bytes []byte
	// MIMEType 图片 MIME 类型。
	MIMEType string
}

// CommentReviewResult 表示评论图文审核结果。
type CommentReviewResult struct {
	// Approved 是否允许公开展示。
	Approved bool `json:"approved" jsonschema:"是否允许公开展示"`
	// TextRisk 文本是否命中审核风险。
	TextRisk bool `json:"textRisk" jsonschema:"文本是否存在风险"`
	// ImageRisk 图片是否命中审核风险。
	ImageRisk bool `json:"imageRisk" jsonschema:"图片是否存在风险"`
	// RiskReason 不通过原因，通过时为空。
	RiskReason string `json:"riskReason" jsonschema:"不通过原因，通过时为空；不通过时必须包含违规类别、命中文本片段或图片序号，以及判定依据"`
	// Tags 商品体验标签，最多保留 5 个。
	Tags []string `json:"tags" jsonschema:"商品体验标签，最多 5 个，每个不超过 8 个中文字符"`
}

// CommentAiRequest 表示商品评价 AI 摘要生成请求。
type CommentAiRequest struct {
	// GoodsName 商品名称快照，用于限定摘要所属商品。
	GoodsName string `json:"goodsName"`
	// Comments 已审核通过的评价列表。
	Comments []CommentAiComment `json:"comments"`
}

// CommentAiComment 表示用于生成 AI 摘要的单条评价。
type CommentAiComment struct {
	// Content 评价文本内容。
	Content string `json:"content"`
	// GoodsScore 商品评分。
	GoodsScore int32 `json:"goodsScore"`
	// PackageScore 包装评分。
	PackageScore int32 `json:"packageScore"`
	// DeliveryScore 配送评分。
	DeliveryScore int32 `json:"deliveryScore"`
	// Tags 评价审核阶段提取出的商品体验标签。
	Tags []string `json:"tags"`
}

// CommentAiResult 表示商品评价 AI 摘要生成结果。
type CommentAiResult struct {
	// Overview 商品详情页评价摘要，最终只保留一条内容。
	Overview CommentAiSceneResult `json:"overview" jsonschema:"商品详情页评价摘要"`
	// List 评价列表页评价摘要，最终最多保留四条内容。
	List CommentAiSceneResult `json:"list" jsonschema:"评价列表页评价摘要"`
}

// CommentAiSceneResult 表示单个 AI 摘要展示场景。
type CommentAiSceneResult struct {
	// Content 当前展示场景下的摘要内容列表。
	Content []CommentAiContentItem `json:"content" jsonschema:"AI摘要内容列表"`
}

// CommentAiContentItem 表示评价 AI 摘要内容项。
type CommentAiContentItem struct {
	// Label 摘要标签。
	Label string `json:"label" jsonschema:"摘要标签"`
	// Content 摘要内容。
	Content string `json:"content" jsonschema:"摘要内容"`
}

// AiAssistantMessage 表示 AI 助手上下文消息。
type AiAssistantMessage struct {
	// Role 消息角色。
	Role string `json:"role"`
	// Content 消息文本内容。
	Content string `json:"content"`
}

// AiAssistantAttachment 表示 AI 助手附件元数据。
type AiAssistantAttachment struct {
	// Name 附件名称。
	Name string `json:"name"`
	// Size 附件大小。
	Size int64 `json:"size"`
	// URL 附件地址。
	URL string `json:"url"`
	// MIMEType 附件 MIME 类型。
	MIMEType string `json:"mimeType"`
	// Content 附件文本内容，用于注入模型上下文。
	Content string `json:"content"`
	// Bytes 附件原始字节，用于图片等多模态输入。
	Bytes []byte `json:"-"`
}

// AiAssistantToolCall 表示 AI 助手工具调用记录。
type AiAssistantToolCall struct {
	// Name 工具名称。
	Name string `json:"name"`
	// Status 工具执行状态。
	Status string `json:"status"`
	// Elapsed 工具耗时。
	Elapsed string `json:"elapsed"`
	// Input 工具入参摘要。
	Input string `json:"input"`
	// Summary 工具结果摘要。
	Summary string `json:"summary"`
	// ErrorMessage 工具失败原因。
	ErrorMessage string `json:"errorMessage"`
	// Output 工具原始输出摘要。
	Output string `json:"output"`
}

// AiAssistantConfirmRequest 表示模型建议的确认动作。
type AiAssistantConfirmRequest struct {
	// Title 确认卡标题。
	Title string `json:"title"`
	// Lines 确认卡内容。
	Lines []string `json:"lines"`
	// Status 确认卡当前状态。
	Status string `json:"status"`
	// Action 确认动作编码。
	Action string `json:"action"`
	// Summary 确认动作摘要。
	Summary string `json:"summary"`
	// Payload 确认动作载荷。
	Payload json.RawMessage `json:"payload"`
	// FormSchema 确认动作需要填写的表单结构。
	FormSchema []map[string]any `json:"formSchema"`
}

// AiAssistantResponse 表示 AI 助手回复结果。
type AiAssistantResponse struct {
	// Content 回复文本内容。
	Content string `json:"content"`
	// TokenUsage 本次调用 token 消耗。
	TokenUsage int64 `json:"tokenUsage"`
	// Source 回复来源：llm/fallback/tool。
	Source string `json:"source"`
	// Model 使用的模型名称。
	Model string `json:"model"`
	// Fallback 是否为降级回复。
	Fallback bool `json:"fallback"`
	// FallbackReason 降级原因。
	FallbackReason string `json:"fallbackReason"`
	// Tools 本次参与的工具调用记录。
	Tools []AiAssistantToolCall `json:"tools"`
	// Confirm 表示本次回复是否需要用户确认。
	Confirm *AiAssistantConfirmRequest `json:"confirm"`
}

// AiAssistantRequest 表示 AI 助手问答请求。
type AiAssistantRequest struct {
	// Terminal 当前终端：admin/app。
	Terminal string `json:"terminal"`
	// Scene 当前会话场景。
	Scene string `json:"scene"`
	// UserName 当前登录用户名称。
	UserName string `json:"userName"`
	// SessionTitle 当前会话标题。
	SessionTitle string `json:"sessionTitle"`
	// Content 当前用户输入。
	Content string `json:"content"`
	// History 历史消息。
	History []AiAssistantMessage `json:"history"`
	// Attachments 当前输入附件。
	Attachments []AiAssistantAttachment `json:"attachments"`
}
