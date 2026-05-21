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
