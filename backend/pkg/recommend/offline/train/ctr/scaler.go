package ctr

import (
	"math"
	"sort"
)

// MinMaxScaler 将数值压缩到 [0,1] 范围。
type MinMaxScaler struct {
	Min float32
	Max float32
}

// NewMinMaxScaler 创建最小最大缩放器。
func NewMinMaxScaler() *MinMaxScaler {
	return &MinMaxScaler{
		Min: float32(math.Inf(1)),
		Max: float32(math.Inf(-1)),
	}
}

// Fit 计算当前样本的最小值和最大值。
func (s *MinMaxScaler) Fit(values []float32) {
	for _, value := range values {
		if value < s.Min {
			s.Min = value
		}
		if value > s.Max {
			s.Max = value
		}
	}
}

// Transform 返回缩放后的数值。
func (s *MinMaxScaler) Transform(value float32) float32 {
	if s.Min > s.Max {
		return value
	}
	if s.Max == s.Min {
		return 1
	}
	return (value - s.Min) / (s.Max - s.Min)
}

// RobustScaler 通过中位数与四分位距抑制离群点影响。
type RobustScaler struct {
	Median float32
	Q1     float32
	Q3     float32
	IQR    float32
}

// NewRobustScaler 创建鲁棒缩放器。
func NewRobustScaler() *RobustScaler {
	return &RobustScaler{}
}

// Fit 根据样本估计中位数与四分位距。
func (s *RobustScaler) Fit(values []float32) {
	if len(values) == 0 {
		return
	}
	cloned := make([]float32, len(values))
	copy(cloned, values)
	sort.Slice(cloned, func(i int, j int) bool {
		return cloned[i] < cloned[j]
	})
	length := len(cloned)
	if length%2 == 0 {
		s.Median = (cloned[length/2-1] + cloned[length/2]) / 2
	} else {
		s.Median = cloned[length/2]
	}
	s.Q1 = cloned[length/4]
	s.Q3 = cloned[(length*3)/4]
	s.IQR = s.Q3 - s.Q1
	if s.IQR == 0 {
		s.IQR = 1
	}
}

// Transform 返回鲁棒缩放后的值。
func (s *RobustScaler) Transform(value float32) float32 {
	if s.IQR == 0 {
		return value
	}
	return (value - s.Median) / s.IQR
}

// AutoScaler 根据数据分布自动选择缩放方式。
type AutoScaler struct {
	UseLog bool
	MinMax MinMaxScaler
	Robust RobustScaler
}

// NewAutoScaler 创建自动缩放器。
func NewAutoScaler() *AutoScaler {
	return &AutoScaler{
		MinMax: *NewMinMaxScaler(),
	}
}

// Fit 根据样本选择对数缩放或鲁棒缩放。
func (s *AutoScaler) Fit(values []float32) {
	if len(values) == 0 {
		return
	}
	hasNegative := false
	for _, value := range values {
		if value < 0 {
			hasNegative = true
			break
		}
	}
	if hasNegative {
		s.UseLog = false
		s.Robust.Fit(values)
		normalized := make([]float32, len(values))
		for i, value := range values {
			normalized[i] = s.Robust.Transform(value)
		}
		s.MinMax.Fit(normalized)
		return
	}
	s.UseLog = true
	normalized := make([]float32, len(values))
	for i, value := range values {
		normalized[i] = float32(math.Log1p(float64(value)))
	}
	s.MinMax.Fit(normalized)
}

// Transform 返回自动缩放后的值。
func (s *AutoScaler) Transform(value float32) float32 {
	if !s.UseLog {
		return s.MinMax.Transform(s.Robust.Transform(value))
	}
	if value < 0 {
		value = 0
	}
	return s.MinMax.Transform(float32(math.Log1p(float64(value))))
}
