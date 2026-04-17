package floats

import "math"

// Add 将源向量逐项累加到目标向量。
func Add(dst []float32, src []float32) {
	if len(dst) != len(src) {
		panic("floats: slice lengths do not match")
	}
	for i := range dst {
		dst[i] += src[i]
	}
}

// AddConst 将常量逐项累加到向量。
func AddConst(dst []float32, value float32) {
	for i := range dst {
		dst[i] += value
	}
}

// Sub 将源向量逐项从目标向量中扣减。
func Sub(dst []float32, src []float32) {
	if len(dst) != len(src) {
		panic("floats: slice lengths do not match")
	}
	for i := range dst {
		dst[i] -= src[i]
	}
}

// SubTo 计算两个向量的差并写入目标向量。
func SubTo(left []float32, right []float32, dst []float32) {
	if len(left) != len(right) || len(left) != len(dst) {
		panic("floats: slice lengths do not match")
	}
	for i := range left {
		dst[i] = left[i] - right[i]
	}
}

// MulTo 计算两个向量的逐项乘积并写入目标向量。
func MulTo(left []float32, right []float32, dst []float32) {
	if len(left) != len(right) || len(left) != len(dst) {
		panic("floats: slice lengths do not match")
	}
	for i := range left {
		dst[i] = left[i] * right[i]
	}
}

// MulConst 将向量按常量原地缩放。
func MulConst(dst []float32, value float32) {
	for i := range dst {
		dst[i] *= value
	}
}

// MulConstAdd 将源向量按常量缩放后累加到目标向量。
func MulConstAdd(src []float32, value float32, dst []float32) {
	if len(src) != len(dst) {
		panic("floats: slice lengths do not match")
	}
	for i := range src {
		dst[i] += src[i] * value
	}
}

// MulConstAddTo 计算 a*c+b 并写入目标向量。
func MulConstAddTo(a []float32, value float32, b []float32, dst []float32) {
	if len(a) != len(b) || len(a) != len(dst) {
		panic("floats: slice lengths do not match")
	}
	for i := range a {
		dst[i] = a[i]*value + b[i]
	}
}

// DivTo 计算两个向量的逐项除法并写入目标向量。
func DivTo(left []float32, right []float32, dst []float32) {
	if len(left) != len(right) || len(left) != len(dst) {
		panic("floats: slice lengths do not match")
	}
	for i := range left {
		dst[i] = left[i] / right[i]
	}
}

// SqrtTo 计算向量逐项平方根并写入目标向量。
func SqrtTo(src []float32, dst []float32) {
	if len(src) != len(dst) {
		panic("floats: slice lengths do not match")
	}
	for i := range src {
		dst[i] = float32(math.Sqrt(float64(src[i])))
	}
}

// Dot 计算两个向量的点积。
func Dot(left []float32, right []float32) float32 {
	if len(left) != len(right) {
		panic("floats: slice lengths do not match")
	}
	result := float32(0)
	for i := range left {
		result += left[i] * right[i]
	}
	return result
}

// MM 计算矩阵乘法并写入目标矩阵切片。
func MM(transposeLeft bool, transposeRight bool, m int, n int, k int, left []float32, lda int, right []float32, ldb int, dst []float32, ldc int) {
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			sum := float32(0)
			for l := 0; l < k; l++ {
				leftIndex := i*lda + l
				if transposeLeft {
					leftIndex = l*lda + i
				}
				rightIndex := l*ldb + j
				if transposeRight {
					rightIndex = j*ldb + l
				}
				sum += left[leftIndex] * right[rightIndex]
			}
			dst[i*ldc+j] = sum
		}
	}
}
