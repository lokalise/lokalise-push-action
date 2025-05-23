package main

import (
	"github.com/bodrovis/lokalise-actions-common/v2/builder"
	"log"
)

const (
	outputDir  = "bin"
	rootSrcDir = "src"
)

var binaries = []string{
	"store_translation_paths",
	"find_all_files",
	"lokalise_upload",
}

func main() {
	err := builder.Run(builder.Options{
		SourceRoot: rootSrcDir,
		OutputDir:  outputDir,
		Binaries:   binaries,
		Compress:   true,
		Build:      true,
		Lint:       true,
	})
	if err != nil {
		log.Fatal(err)
	}
}
