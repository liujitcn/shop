package llm

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
	RiskReason string `json:"riskReason" jsonschema:"不通过原因，通过时为空"`
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
