package comment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

const (
	// commentAICommentLimit 表示单次摘要生成最多送入模型的评价数量。
	commentAICommentLimit = 50
	// commentAIGoodsNameRuneLimit 表示摘要生成时商品名称最多保留的字符数，避免长标题挤占模型上下文。
	commentAIGoodsNameRuneLimit = 120
	// commentAIContentRuneLimit 表示单条评价正文最多送入模型的字符数。
	commentAIContentRuneLimit = 500
	// commentAITagLimit 表示单条评价最多送入模型的标签数量。
	commentAITagLimit = 5
	// commentAITagRuneLimit 表示单个评价标签最多送入模型的字符数。
	commentAITagRuneLimit = 12
	// commentAIOverviewContentLimit 表示商品详情页评价摘要最终只保留一条内容。
	commentAIOverviewContentLimit = 1
	// commentAIListContentLimit 表示评价列表页评价摘要最终最多保留四条内容。
	commentAIListContentLimit = 4
	// commentAIOverviewDefaultItemName 表示商品详情页摘要缺少标签时的默认标签。
	commentAIOverviewDefaultItemName = "AI 总结"
)

// GenerateAI 生成商品评价 AI 摘要数据。
func (r *Runtime) GenerateAI(ctx context.Context, req AIRequest) (*AIResult, error) {
	req = cleanAIRequest(req)
	// 没有审核通过的评价时，不调用大模型生成空摘要。
	if len(req.Comments) == 0 {
		return &AIResult{}, nil
	}

	var schema *jsonschema.Schema
	schema, err := cachedAIResultSchema()
	if err != nil {
		return nil, fmt.Errorf("build comment ai schema: %w", err)
	}
	result := &AIResult{}
	var rawPayload []byte
	rawPayload, err = json.Marshal(req)
	commentAIPrompt := "请基于已审核通过的商品评价生成评价 AI 摘要。"
	// payload 序列化失败不影响任务继续执行，只是少给模型结构化上下文，后续由空摘要或异常结果兜底。
	if err == nil {
		commentAIPrompt = "请基于已审核通过的商品评价生成评价 AI 摘要：\n" + string(rawPayload)
	}
	err = r.generateStructured(ctx, r.commentAIInstruction, []any{commentAIPrompt}, schema, result)
	if err != nil {
		return nil, err
	}
	// 前端两个展示位容量不同，模型即使返回更多内容也在这里收敛，避免展示层再重复裁剪。
	result.Overview.Content = limitAIContentItems(result.Overview.Content, commentAIOverviewContentLimit, commentAIOverviewDefaultItemName)
	result.List.Content = limitAIContentItems(result.List.Content, commentAIListContentLimit, "")
	return result, nil
}

// cleanAIRequest 清理并限制评价摘要生成请求，避免单次模型输入过大。
func cleanAIRequest(req AIRequest) AIRequest {
	// 摘要只需要商品名辅助限定语境，长标题完整送入模型收益不高，还会挤占评价正文空间。
	req.GoodsName = trimStringByRunes(req.GoodsName, commentAIGoodsNameRuneLimit)
	limit := len(req.Comments)
	if limit > commentAICommentLimit {
		limit = commentAICommentLimit
	}
	comments := make([]AIComment, 0, limit)
	for _, item := range req.Comments {
		// 单条评价只保留摘要所需的主要语义和标签，评分字段保持原值供模型判断整体倾向。
		item.Content = trimStringByRunes(item.Content, commentAIContentRuneLimit)
		item.Tags = limitStringList(cleanStringList(item.Tags), commentAITagLimit, commentAITagRuneLimit)
		// 空正文且无标签的评价对摘要贡献较低，跳过以降低模型输入成本。
		if item.Content == "" && len(item.Tags) == 0 {
			continue
		}
		comments = append(comments, item)
		// 已达到摘要输入上限时，停止继续追加。
		if len(comments) >= commentAICommentLimit {
			break
		}
	}
	req.Comments = comments
	return req
}

// limitAIContentItems 清理并限制 AI 摘要内容项数量。
func limitAIContentItems(values []AIContentItem, limit int, defaultLabel string) []AIContentItem {
	// 限制小于等于 0 时，直接返回空列表。
	if limit <= 0 {
		return []AIContentItem{}
	}
	result := make([]AIContentItem, 0, len(values))
	for _, value := range values {
		value.Label = strings.TrimSpace(value.Label)
		value.Content = strings.TrimSpace(value.Content)
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
