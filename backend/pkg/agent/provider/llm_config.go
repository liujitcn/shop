package provider

import bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

// aiModelConfigured 判断大模型启动配置是否完整。
func aiModelConfigured(modelCfg *bootstrapConfigv1.AI_Model) bool {
	if modelCfg == nil || modelCfg.GetModelName() == "" {
		return false
	}
	switch modelCfg.GetType() {
	case bootstrapConfigv1.AI_Model_CLOUD_MODEL:
		cloud := modelCfg.GetCloud()
		return cloud != nil && cloud.GetApiKey() != ""
	case bootstrapConfigv1.AI_Model_LOCAL_MODEL:
		return modelCfg.GetLocal() != nil
	default:
		return false
	}
}
