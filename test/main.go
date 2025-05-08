package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var reader *bufio.Reader

func solve() {
	nStr, _ := reader.ReadString('\n')
	n, _ := strconv.Atoi(strings.TrimSpace(nStr))
	sStr, _ := reader.ReadString('\n')
	s := strings.TrimSpace(sStr)

	zerosInS := 0
	for _, char := range s {
		if char == '0' {
			zerosInS++
		}
	}
	onesInS := n - zerosInS

	var result int64

	if onesInS == 0 { // String s is all '0's
		if n == 1 {
			result = 0
		} else if n == 2 {
			result = 1
		} else { // n >= 3
			result = int64(n) * int64(n-1)
		}
	} else if zerosInS == 0 { // String s is all '1's
		// This implies n >= 1 based on problem constraints
		result = 1
	} else { // String s has a mix of '0's and '1's
		// N0 * (n-1) + N1
		// N0 is zerosInS, N1 is onesInS
		result = int64(zerosInS)*int64(n-1) + int64(onesInS)
	}
	fmt.Println(result)
}

func main() {
	reader = bufio.NewReader(os.Stdin)
	tStr, _ := reader.ReadString('\n')
	t, _ := strconv.Atoi(strings.TrimSpace(tStr))
	for i := 0; i < t; i++ {
		solve()
	}
}