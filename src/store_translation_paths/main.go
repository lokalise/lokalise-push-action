package main

import (
	"githuboutput"
	"os"
)

func main() {
	name := "output_variable_name"
	value := "output_value"

	if !githuboutput.WriteToGitHubOutput(name, value) {
		os.Exit(1)
	}

	os.Exit(0)
}
