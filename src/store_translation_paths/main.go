package main

import (
	"fmt"
	"os"
)

func main() {
	name := "output_variable_name"
	value := "output_value"

	githubOutput := os.Getenv("GITHUB_OUTPUT")

	file, err := os.OpenFile(githubOutput, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening GITHUB_OUTPUT file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%s=%s\n", name, value))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to GITHUB_OUTPUT file: %v\n", err)
		os.Exit(1)
	}
}
