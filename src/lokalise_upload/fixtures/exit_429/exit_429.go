package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "API request error 429: Rate limit exceeded")
	os.Exit(1)
}
