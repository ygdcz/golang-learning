package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ygdcz/golang-learning/src/simplemath"
)

var Usage = func() {
	fmt.Println("USAGE: calc command [arguments] ...")
	fmt.Println("\nThe commands are:\n\tadd\tAddition of two values.\n\tsqrt\tSquare root of a non-negative value.")
}

func main() {
	args := os.Args

	if args == nil || len(args) < 2 {
		Usage()
		return
	}

	switch args[1] {
	case "add":
		if len(args) != 4 {
			fmt.Println("USAGE: calc add <number1> <number2>") // 修改提示为number
			return
		}
		v1, err1 := strconv.ParseFloat(args[2], 64) // 替换Atoi为ParseFloat
		v2, err2 := strconv.ParseFloat(args[3], 64)
		if err1 != nil || err2 != nil {
			fmt.Println("USAGE: calc add 参数需要是数字（支持整数和浮点数）")
			return
		}
		ret := simplemath.Add(v1, v2)     // 使用float64类型调用Add
		fmt.Printf("Result: %.2f\n", ret) // 格式化输出小数位
	case "sqrt":
		if len(args) != 3 {
			fmt.Println("USAGE: calc sqrt <number1>")
			return
		}
		v, err := strconv.ParseFloat(args[2], 64) // 替换Atoi为ParseFloat
		if err != nil || v < 0 {
			fmt.Println("USAGE: calc sqrt <integer>")
			return
		}
		ret := simplemath.Sqrt(v)
		fmt.Println("Result: ", ret)
	default:
		Usage()
	}
}
