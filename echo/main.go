// echos command line arguments
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func withLoop() {
	var s, sep string
	for i := 1; i < len(os.Args); i++ {
		s += sep + os.Args[i]
		sep = " "
	}
	fmt.Println(s)
}

func withRange() {
	s, sep := "", " "
	for index, arg := range os.Args[1:] {
		s += strconv.Itoa(index) + sep + arg + sep
	}
	fmt.Println(s)
}

func withJoin() {
	fmt.Println(strings.Join(os.Args[1:], " "))
}

func main() {
	withLoop()
	withRange()
	withJoin()
}
