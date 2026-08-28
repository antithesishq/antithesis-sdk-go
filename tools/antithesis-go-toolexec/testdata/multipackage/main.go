package main

import (
	"fmt"
	"os"

	"multipkg/alpha"
	"multipkg/beta"
	"multipkg/shared"
)

var sink string

func run(command, arg string) string {
	switch command {
	case "describe":
		n := 0
		fmt.Sscanf(arg, "%d", &n)
		return alpha.Describe(n)
	case "total":
		n := 0
		fmt.Sscanf(arg, "%d", &n)
		return fmt.Sprint(beta.Total(n))
	case "eager":
		return shared.Eager
	}
	return "unknown"
}

func main() {
	arg := ""
	if len(os.Args) > 2 {
		arg = os.Args[2]
	}
	sink = run(os.Args[1], arg)
	_ = sink
}
