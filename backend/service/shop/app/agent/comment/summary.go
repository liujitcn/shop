package comment

import (
	"context"
	"encoding/json"
	"fmt"

	einoStructured "shop/pkg/agent/eino/structured"
)

const (
	// commentSummaryCommentLimit 表示单次摘要生成最多送入模型的评价数量。
	commentSummaryCommentLimit = 50
	// commentSummaryGoodsNameRuneLimit 表示摘要生成时商品名称最多保留的字符数，避免长标题挤占模型上下文。
	commentSummaryGoodsNameRuneLimit = 120
	// commentSummaryContentRuneLimit 表示单条评价正文最多送入模型的字符数。
	commentSummaryContentRuneLimit = 500
	// commentSummaryTagLimit 表示单条评价最多送入模型的标签数量。
	commentSummaryTagLimit = 5
	// commentSummaryTagRuneLimit 表示单个评价标签最多送入模型的字符数。
	commentSummaryTagRuneLimit = 12
	// commentSummaryOverviewContentLimit 表示商品详情页评价摘要最终只保留一条内容。
	commentSummaryOverviewContentLimit = 1
	// commentSummaryListContentLimit 表示评价列表页评价摘要最终最多保留四条内容。
	commentSummaryListContentLimit = 4
	// commentSummaryOverviewDefaultItemName 表示商品详情页摘要缺少标签时的默认标签。
	commentSummaryOverviewDefaultItemName = "AI 总结"
)

// GenerateSummary 生成商品评价摘要数据。
func (r *Runtime) GenerateSummary(ctx context.Context, req SummaryRequest) (*SummaryResult, error) {
	req = cleanSummaryRequest(req)
	// 没有审核通过的评价时，不调用大模型生成空摘要。
	if len(req.Comments) == 0 {
		return &SummaryResult{}, nil
	}

	var outputSchema *einoStructured.Schema
	var err error
	outputSchema, err = cachedSummaryResultSchema()
	if err != nil {
		return nil, fmt.Errorf("build comment summary schema: %w", err)
	}
	result := &SummaryResult{}
	var rawPayload []byte
	rawPayload, err = json.Marshal(req)
	commentSummaryPrompt := "请基于已审核通过的商品评价生成评价摘要。"
	// payload 序列化失败不影响任务继续执行，只是少给模型结构化上下文，后续由空摘要或异常结果兜底。
	if err == nil {
		commentSummaryPrompt = "请基于已审核通过的商品评价生成评价摘要：\n" + string(rawPayload)
	}
	err = r.generateStructured(ctx, commentSummaryInstruction, []*einoStructured.Part{textInputPart(commentSummaryPrompt)}, outputSchema, result)
	if err != nil {
		return nil, err
	}
	// 前端两个展示位容量不同，模型即使返回更多内容也在这里收敛，避免展示层再重复裁剪。
	result.Overview.Content = limitSummaryContentItems(result.Overview.Content, commentSummaryOverviewContentLimit, commentSummaryOverviewDefaultItemName)
	result.List.Content = limitSummaryContentItems(result.List.Content, commentSummaryListContentLimit, "")
	return result, nil
}

// cleanSummaryRequest 清理并限制评价摘要生成请求，避免单次模型输入过大。
func cleanSummaryRequest(req SummaryRequest) SummaryRequest {
	// 摘要只需要商品名辅助限定语境，长标题完整送入模型收益不高，还会挤占评价正文空间。
	req.GoodsName = trimStringByRunes(req.GoodsName, commentSummaryGoodsNameRuneLimit)
	limit := len(req.Comments)
	if limit > commentSummaryCommentLimit {
		limit = commentSummaryCommentLimit
	}
	comments := make([]SummaryComment, 0, limit)
	for _, item := range req.Comments {
		// 单条评价只保留摘要所需的主要语义和标签，评分字段保持原值供模型判断整体倾向。
		item.Content = trimStringByRunes(item.Content, commentSummaryContentRuneLimit)
		item.Tags = limitStringList(cleanStringList(item.Tags), commentSummaryTagLimit, commentSummaryTagRuneLimit)
		// 空正文且无标签的评价对摘要贡献较低，跳过以降低模型输入成本。
		if item.Content == "" && len(item.Tags) == 0 {
			continue
		}
		comments = append(comments, item)
		// 已达到摘要输入上限时，停止继续追加。
		if len(comments) >= commentSummaryCommentLimit {
			break
		}
	}
	req.Comments = comments
	return req
}

// limitSummaryContentItems 清理并限制评价摘要内容项数量。
func limitSummaryContentItems(values []SummaryContentItem, limit int, defaultLabel string) []SummaryContentItem {
	// 限制小于等于 0 时，直接返回空列表。
	if limit <= 0 {
		return []SummaryContentItem{}
	}
	result := make([]SummaryContentItem, 0, len(values))
	for _, value := range values {
		// 摘要内容为空时，不进入最终摘要。
		if value.Content == "" {
			continue
		}
		// 商品详情摘要标签固定兜底，避免模型遗漏标签导致前端展示异常。
		if value.Label == "" {
			value.Label = defaultLabel
		}
		result = append(result, value)
		// 已达到模块上限时，停止继续追加。
		if len(result) >= limit {
			break
		}
	}
	return result
}
