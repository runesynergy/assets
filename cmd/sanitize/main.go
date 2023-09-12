package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	for i := 1; i < len(os.Args); i++ {
		arg := strings.ToLower(os.Args[i])
		var result []rune
		for _, r := range arg {
			if r == ' ' {
				result = append(result, '_')
			} else if r == '/' || r == '.' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				result = append(result, r)
			}
		}
		fmt.Println(string(result))
	}
}
