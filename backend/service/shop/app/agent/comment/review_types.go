package comment

import "encoding/json"

// ReviewRequest 表示评论图文审核请求。
type ReviewRequest struct {
	// GoodsName 商品名称快照，用于判断评价内容是否与商品相关。
	GoodsName string `json:"goodsName"`
	// SKUDesc 商品规格描述快照，用于辅助判断尺码、颜色、规格等评价内容。
	SKUDesc string `json:"skuDesc"`
	// Content 评价或讨论文本内容。
	Content string `json:"content"`
	// ExistingTags 当前商品已有评价标签，用于引导模型优先复用。
	ExistingTags []string `json:"existingTags"`
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
	Tags []string `json:"tags" jsonschema:"商品体验标签，最多 5 个，每个不超过 8 个中文字符；有 existingTags 时优先原样复用已有标签，确无合适项才生成新标签"`

	approvedSet  bool
	textRiskSet  bool
	imageRiskSet bool
}

// UnmarshalJSON 解析审核结果并记录关键判定字段是否由模型显式返回。
func (r *ReviewResult) UnmarshalJSON(data []byte) error {
	type reviewResultJSON struct {
		Approved   *bool    `json:"approved"`
		TextRisk   *bool    `json:"textRisk"`
		ImageRisk  *bool    `json:"imageRisk"`
		RiskReason string   `json:"riskReason"`
		Tags       []string `json:"tags"`
	}

	var value reviewResultJSON
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	r.Approved = false
	r.approvedSet = false
	if value.Approved != nil {
		r.Approved = *value.Approved
		r.approvedSet = true
	}

	r.TextRisk = false
	r.textRiskSet = false
	if value.TextRisk != nil {
		r.TextRisk = *value.TextRisk
		r.textRiskSet = true
	}

	r.ImageRisk = false
	r.imageRiskSet = false
	if value.ImageRisk != nil {
		r.ImageRisk = *value.ImageRisk
		r.imageRiskSet = true
	}

	r.RiskReason = value.RiskReason
	r.Tags = value.Tags
	return nil
}
