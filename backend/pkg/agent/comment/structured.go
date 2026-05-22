package comment

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
)

var (
	// reviewResultSchemaOnce 确保审核结果 Schema 只生成一次，避免高频审核时重复反射类型。
	reviewResultSchemaOnce sync.Once
	reviewResultSchema     *jsonschema.Schema
	reviewResultSchemaErr  error

	// aiResultSchemaOnce 确保摘要结果 Schema 只生成一次，减少定时摘要刷新时的固定开销。
	aiResultSchemaOnce sync.Once
	aiResultSchema     *jsonschema.Schema
	aiResultSchemaErr  error
)

// cachedReviewResultSchema 返回缓存后的评论审核结构化输出 Schema。
func cachedReviewResultSchema() (*jsonschema.Schema, error) {
	reviewResultSchemaOnce.Do(func() {
		reviewResultSchema, reviewResultSchemaErr = jsonschema.For[ReviewResult](nil)
	})
	return reviewResultSchema, reviewResultSchemaErr
}

// cachedAIResultSchema 返回缓存后的评价摘要结构化输出 Schema。
func cachedAIResultSchema() (*jsonschema.Schema, error) {
	aiResultSchemaOnce.Do(func() {
		aiResultSchema, aiResultSchemaErr = jsonschema.For[AIResult](nil)
	})
	return aiResultSchema, aiResultSchemaErr
}

// decodeStructuredContent 解码模型返回的结构化 JSON 文本。
func decodeStructuredContent(content string, out any) error {
	cleanContent := strings.TrimSpace(content)
	// 大部分模型在配置 JSON Schema 后会直接返回纯 JSON，先走最快路径。
	err := json.Unmarshal([]byte(cleanContent), out)
	if err == nil {
		return nil
	}
	// 少数模型仍可能包一层说明文字或 Markdown 围栏，这里只提取可被 JSON decoder 接受的片段。
	for _, jsonCandidate := range findJSONCandidates(cleanContent) {
		if json.Unmarshal([]byte(jsonCandidate), out) == nil {
			return nil
		}
	}
	return err
}

// findJSONCandidates 从模型额外说明文本中提取合法 JSON 片段。
func findJSONCandidates(content string) []string {
	result := make([]string, 0, 1)
	for index, value := range content {
		// JSON 结构只可能从对象或数组起始符开始，跳过其他字符可以减少无意义 decoder 尝试。
		if value != '{' && value != '[' {
			continue
		}
		decoder := json.NewDecoder(strings.NewReader(content[index:]))
		var rawMessage json.RawMessage
		// RawMessage 会在第一个完整 JSON 值结束处停止，适合从“说明文字 + JSON + 说明文字”中截出候选值。
		if decoder.Decode(&rawMessage) == nil {
			result = append(result, string(rawMessage))
		}
	}
	return result
}
