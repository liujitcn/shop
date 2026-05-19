package provider

import bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

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

// cloneExtraFields 复制扩展字段，避免后续参数补充污染原始配置。
func cloneExtraFields(extraFields map[string]any) map[string]any {
	if len(extraFields) == 0 {
		return nil
	}
	result := make(map[string]any, len(extraFields))
	for key, value := range extraFields {
		result[key] = value
	}
	return result
}
