package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "some permanent error happened")
	os.Exit(1)
}
