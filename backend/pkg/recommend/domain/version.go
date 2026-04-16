package domain

import "fmt"

// StrategyVersion 表示可驱动在线推荐的策略版本。
type StrategyVersion struct {
	Scene      int32  // 推荐场景
	ModelName  string // 模型或策略名称
	ModelType  string // 模型或策略类型
	Version    string // 版本号
	ConfigJSON string // 当前版本配置 JSON
	Status     int32  // 版本状态
}

// FullName 返回模型名与版本号拼接后的完整名称。
func (v *StrategyVersion) FullName() string {
	// 版本对象为空时，无法继续拼接完整名称。
	if v == nil {
		return ""
	}
	// 版本号为空时，仅返回模型名称，兼容历史记录。
	if v.Version == "" {
		return v.ModelName
	}
	return fmt.Sprintf("%s:%s", v.ModelName, v.Version)
}

// CacheNamespace 返回当前版本对应的缓存命名空间。
func (v *StrategyVersion) CacheNamespace() string {
	// 版本对象为空时，无法生成缓存命名空间。
	if v == nil {
		return ""
	}
	return fmt.Sprintf("scene/%d/%s", v.Scene, v.FullName())
}
