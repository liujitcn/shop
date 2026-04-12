package rank

import (
	"time"

	_time "github.com/liujitcn/go-utils/time"
)

const (
	defaultFreshnessWindowDays   = 30.0
	defaultHighExposureThreshold = 20
)

// CalculateDayDecay 计算统计指标按天衰减后的权重。
func CalculateDayDecay(statDate time.Time) float64 {
	if statDate.IsZero() {
		// 空统计时间说明当前样本不可用。
		return 0
	}
	daysAgo := time.Since(statDate).Hours() / 24
	if daysAgo <= 0 {
		// 当天数据保持满权重。
		return 1
	}
	return 1 / (1 + daysAgo*0.08)
}

// CalculateExposurePenalty 计算高曝光低点击场景下的惩罚分。
func CalculateExposurePenalty(exposureCount, clickCount int64) float64 {
	if exposureCount < defaultHighExposureThreshold {
		// 低曝光商品不做额外惩罚。
		return 0
	}
	if clickCount <= 0 {
		// 已经被大量曝光但完全没有点击时给最高惩罚。
		return 1.2
	}
	ctr := float64(clickCount) / float64(exposureCount)
	switch {
	case ctr < 0.005:
		// 点击率极低时施加强惩罚。
		return 0.8
	case ctr < 0.01:
		// 点击率偏低时施加中等惩罚。
		return 0.4
	default:
		// 点击率正常时不额外扣分。
		return 0
	}
}

// CalculateFreshnessScore 计算商品的新鲜度分数。
func CalculateFreshnessScore(createdAtStr string) float64 {
	createdAt := _time.StringTimeToTime(createdAtStr)
	if createdAt == nil || createdAt.IsZero() {
		// 缺失创建时间时无法计算新鲜度。
		return 0
	}
	daysAgo := time.Since(*createdAt).Hours() / 24
	if daysAgo <= 0 {
		// 当天创建的商品给予满分。
		return 1
	}
	score := 1 - (daysAgo / defaultFreshnessWindowDays)
	if score < 0 {
		// 超过窗口期后新鲜度归零。
		return 0
	}
	return score
}
