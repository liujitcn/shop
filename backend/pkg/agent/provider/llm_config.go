package provider

import (
	"strings"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/openai/openai-go/v3/shared"
)

// llmExtraFields 返回大模型配置中需要透传的扩展字段。
func llmExtraFields(bootstrapCfg *bootstrapConfigv1.Client_Llm) map[string]any {
	if bootstrapCfg == nil || bootstrapCfg.GetExtraFields() == nil {
		return nil
	}
	extraFields := bootstrapCfg.GetExtraFields().AsMap()
	if len(extraFields) == 0 {
		return nil
	}
	return extraFields
}

// llmReasoningEffort 返回 OpenAI SDK 支持的推理强度。
func llmReasoningEffort(bootstrapCfg *bootstrapConfigv1.Client_Llm) shared.ReasoningEffort {
	if bootstrapCfg == nil {
		return ""
	}
	switch strings.TrimSpace(bootstrapCfg.GetReasoningEffort()) {
	case string(shared.ReasoningEffortMinimal):
		return shared.ReasoningEffortMinimal
	case string(shared.ReasoningEffortLow):
		return shared.ReasoningEffortLow
	case string(shared.ReasoningEffortMedium):
		return shared.ReasoningEffortMedium
	case string(shared.ReasoningEffortHigh):
		return shared.ReasoningEffortHigh
	default:
		return ""
	}
}
