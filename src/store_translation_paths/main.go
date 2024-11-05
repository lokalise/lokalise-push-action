package main

import (
	"githuboutput"
	"log"
)

func main() {
	name := "output_variable_name"
	value := "output_value"

	if err := githuboutput.WriteToGitHubOutput(name, value); err != nil {
		log.Fatalf("Failed to write to GitHub output: %v", err)
	}
}
