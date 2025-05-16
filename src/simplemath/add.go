package simplemath

// Add 返回两个数值的和，支持int和float64类型
func Add[T int | float64](a, b T) T {
    return a + b
}