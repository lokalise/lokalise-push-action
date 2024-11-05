package main

import (
	"fmt"
	"os"
)

func main() {
	k1 := os.Getenv("K1")
	k2 := os.Getenv("K2")
	k3 := os.Getenv("K3")

	// Log other information to stderr
	if k1 == "" || k2 == "" {
		fmt.Fprintln(os.Stderr, "Missing required environment variables.")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "API Token: %s\n", k1)
	fmt.Fprintf(os.Stderr, "Project ID: %s\n", k2)
	fmt.Fprintf(os.Stderr, "Base Language: %s\n", k3)

	// Only output the result to stdout
	output := "some output value"
	fmt.Println(output) // Expected to be captured as $output in GitHub Action

	// Additional confirmation output
	fmt.Fprintln(os.Stderr, "Output sent to stdout:", output)
}
