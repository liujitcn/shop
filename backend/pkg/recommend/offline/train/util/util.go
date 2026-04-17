package util

import (
	"math/rand"
	"sync"
)

// RandomGenerator 封装训练阶段使用的随机数生成器。
type RandomGenerator struct {
	*rand.Rand
}

// NewRandomGenerator 创建带固定种子的随机数生成器。
func NewRandomGenerator(seed int64) RandomGenerator {
	return RandomGenerator{Rand: rand.New(rand.NewSource(seed))}
}

// NewMatrix32 创建指定形状的二维浮点矩阵。
func NewMatrix32(row int, col int) [][]float32 {
	result := make([][]float32, row)
	for i := range result {
		result[i] = make([]float32, col)
	}
	return result
}

// NormalMatrix 创建正态分布初始化矩阵。
func (r RandomGenerator) NormalMatrix(row int, col int, mean float32, stdDev float32) [][]float32 {
	result := make([][]float32, row)
	for i := range result {
		result[i] = make([]float32, col)
		for j := range result[i] {
			result[i][j] = float32(r.NormFloat64())*stdDev + mean
		}
	}
	return result
}

// CheckPanic 兜底恢复并行训练中的 panic。
func CheckPanic() {
	_ = recover()
}

// lockedSource 让随机源可以安全地被多个协程复用。
type lockedSource struct {
	mutex sync.Mutex
	src   rand.Source
}

// NewRand 创建线程安全的随机数实例。
func NewRand(seed int64) *rand.Rand {
	return rand.New(&lockedSource{
		src: rand.NewSource(seed),
	})
}

// Int63 返回下一个 63 位随机整数。
func (r *lockedSource) Int63() int64 {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.src.Int63()
}

// Seed 重置随机源种子。
func (r *lockedSource) Seed(seed int64) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.src.Seed(seed)
}
