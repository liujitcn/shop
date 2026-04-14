package recommend

import "testing"

// newTestRecommend 创建供测试使用的推荐实例。
func newTestRecommend(t *testing.T, options ...Option) *Recommend {
	t.Helper()

	instance, err := New(options...)
	if err != nil {
		t.Fatalf("创建推荐实例失败: %v", err)
	}
	return instance
}
