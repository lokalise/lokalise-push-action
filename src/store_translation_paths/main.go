package main

import (
	"os"
	"strings"
)

func parsePaths(envVar string) []string {
	paths := strings.FieldsFunc(envVar, func(r rune) bool {
		return r == '\n' || r == '\r'
	})

	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
	}

	return paths
}

// storeTranslationPaths generates paths and writes them to paths.txt based on environment variables
func storeTranslationPaths(paths []string, flatNaming, baseLang, fileFormat string) error {
	file, err := os.Create("paths.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	for _, path := range paths {
		if path == "" {
			continue
		}

		var formattedPath string
		if flatNaming == "true" {
			formattedPath = "./" + path + "/" + baseLang + "." + fileFormat
		} else {
			formattedPath = "./" + path + "/" + baseLang + "/**/*." + fileFormat
		}

		if _, err := file.WriteString(formattedPath + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	translationsPaths := parsePaths(os.Getenv("TRANSLATIONS_PATH"))
	flatNaming := os.Getenv("FLAT_NAMING")
	baseLang := os.Getenv("BASE_LANG")
	fileFormat := os.Getenv("FILE_FORMAT")

	if err := storeTranslationPaths(translationsPaths, flatNaming, baseLang, fileFormat); err != nil {
		os.Exit(1)
	}
}
