package simplemath

import "math"

// Sqrt 返回一个数的平方根，支持int和float64类型
func Sqrt[T int | float64](a T) T {
	if a < 0 {
		return 0
	}
	v := math.Sqrt(float64(a))
	return T(v)
}
