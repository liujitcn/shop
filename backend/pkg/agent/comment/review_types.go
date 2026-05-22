package comment

// ReviewRequest 表示评论图文审核请求。
type ReviewRequest struct {
	// GoodsName 商品名称快照，用于判断评价内容是否与商品相关。
	GoodsName string `json:"goodsName"`
	// SKUDesc 商品规格描述快照，用于辅助判断尺码、颜色、规格等评价内容。
	SKUDesc string `json:"skuDesc"`
	// Content 评价或讨论文本内容。
	Content string `json:"content"`
	// ImageURLs 评价图片地址列表，用于多模态审核。
	ImageURLs []string `json:"imageUrls"`
	// ImageData 评价图片字节列表，用于审核本地或非公网图片。
	ImageData []ReviewImageData `json:"-"`
}

// ReviewImageData 表示评论审核使用的图片字节数据。
type ReviewImageData struct {
	// Name 图片名称，用于标识多模态输入。
	Name string
	// Bytes 图片字节内容。
	Bytes []byte
	// MIMEType 图片 MIME 类型。
	MIMEType string
}

// ReviewResult 表示评论图文审核结果。
type ReviewResult struct {
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
