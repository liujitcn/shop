package comment

// SummaryRequest 表示商品评价摘要生成请求。
type SummaryRequest struct {
	// GoodsName 商品名称快照，用于限定摘要所属商品。
	GoodsName string `json:"goodsName"`
	// Comments 已审核通过的评价列表。
	Comments []SummaryComment `json:"comments"`
}

// SummaryComment 表示用于生成评价摘要的单条评价。
type SummaryComment struct {
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

// SummaryResult 表示商品评价摘要生成结果。
type SummaryResult struct {
	// Overview 商品详情页评价摘要，最终只保留一条内容。
	Overview SummarySceneResult `json:"overview" jsonschema:"商品详情页评价摘要"`
	// List 评价列表页评价摘要，最终最多保留四条内容。
	List SummarySceneResult `json:"list" jsonschema:"评价列表页评价摘要"`
}

// SummarySceneResult 表示单个评价摘要展示场景。
type SummarySceneResult struct {
	// Content 当前展示场景下的摘要内容列表。
	Content []SummaryContentItem `json:"content" jsonschema:"评价摘要内容列表"`
}

// SummaryContentItem 表示评价摘要内容项。
type SummaryContentItem struct {
	// Label 摘要标签。
	Label string `json:"label" jsonschema:"摘要标签"`
	// Content 摘要内容。
	Content string `json:"content" jsonschema:"摘要内容"`
}
