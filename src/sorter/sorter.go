package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/ygdcz/golang-learning/src/sorter/algorithms/bubblesort"
	"github.com/ygdcz/golang-learning/src/sorter/algorithms/qsort"
)

var infile *string = flag.String("i", "infile", "File contains values for sorting")
var outfile *string = flag.String("o", "outfile", "File to receive sorted values")
var algorithm *string = flag.String("a", "qsort", "Sort algorithm")

func readValues(infile string) (values []int, err error) {
	file, openErr := os.Open(infile)
	if openErr != nil {
		fmt.Println("Failed to open the input file ", infile)
		return
	}
	defer file.Close()

	br := bufio.NewReader(file)

	values = make([]int, 0)
	for {
		line, isPrefix, err1 := br.ReadLine()

		if err1 != nil {
			if err1 != io.EOF {
				err = err1
			}
			break
		}

		if isPrefix {
			fmt.Println("A too long line, seems unexpected.")
			return
		}

		str := string(line) // 转换字符数组为字符串

		value, err2 := strconv.Atoi(str)

		if err2 != nil {
			err = err2
			return
		}

		values = append(values, value)
	}
	return
}

func writeValues(values []int, outfile string) error {
	file, createErr := os.Create(outfile)
	if createErr != nil {
		fmt.Println("Failed to open the output file ", outfile)
		return createErr
	}
	defer file.Close()
	for _, value := range values {
		str := strconv.Itoa(value)
		file.WriteString(str + "\n")
	}
	return nil
}

func main() {
	flag.Parse()

	if infile != nil {
		fmt.Println("infile =", *infile, "outfile =", *outfile, "algorithm =", *algorithm)
	}

	values, err := readValues(*infile)
	if err == nil {
		t1 := time.Now()
		switch *algorithm {
		case "qsort":
			qsort.QuickSort(values)
		case "bubblesort":
			bubblesort.BubbleSort(values)
		}
		t2 := time.Now()
		fmt.Println("The sorting process costs ", t2.Sub(t1), " to complete.")
		writeValues(values, *outfile)
	} else {
		fmt.Println(err)
	}
}
