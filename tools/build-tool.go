package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	upxAvailable = "upx"
	rootSrcDir   = "src"
)

var binaries = []string{
	"store_translation_paths",
	"find_all_files",
	"lokalise_upload",
}

func main() {
	projectRoot := getProjectRoot()
	binDir := filepath.Join(projectRoot, "bin")

	// Create bin directory
	if err := os.MkdirAll(binDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create bin directory: %v", err)
	}

	// Ensure gofumpt is installed
	if !checkCommand("gofumpt") {
		if err := runCommand("go", []string{"install", "mvdan.cc/gofumpt@latest"}); err != nil {
			log.Printf("Warning: failed to install gofumpt: %v", err)
		}
	}

	for _, binaryName := range binaries {
		fullPkgPath := filepath.Join(projectRoot, rootSrcDir, binaryName)

		// Lint once per directory
		lint(fullPkgPath)

		// Build binaries
		builtBinaries, err := buildBinary(fullPkgPath, binDir, binaryName)
		if err != nil {
			log.Printf("Build failed for %s: %v", binaryName, err)
			continue
		}

		// Optional UPX compression for Linux targets
		if checkCommand(upxAvailable) {
			for _, binPath := range builtBinaries {
				if isLinuxBinary(binPath) {
					if err := compressWithUPX(binPath); err != nil {
						log.Printf("Compression failed for %s: %v", binPath, err)
					}
				}
			}
		} else {
			fmt.Println("UPX not found; skipping compression.")
		}

		fmt.Printf("Build complete for %s. Binaries at: %s\n", binaryName, binDir)
	}
}

func runCommand(cmd string, args []string) error {
	command := exec.Command(cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func lint(dir string) {
	fmt.Printf("Running lint on %s...\n", dir)

	// Run go fmt from inside the module
	cmd := exec.Command("go", "fmt", "./...")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("go fmt error:", err)
	}

	// Run gofumpt with dir path
	if err := runCommand("gofumpt", []string{"-l", "-w", dir}); err != nil {
		fmt.Println("gofumpt error:", err)
	}

	fmt.Println("Lint done.")
}

func isLinuxBinary(path string) bool {
	return filepath.Ext(path) == "" &&
		(strings.HasSuffix(path, "_linux_amd64") || strings.HasSuffix(path, "_linux_arm64"))
}

func getProjectRoot() string {
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}
	return root
}

func buildBinary(srcDir, outputDir, binaryName string) ([]string, error) {
	targets := []struct {
		goos   string
		goarch string
		suffix string
	}{
		{"linux", "amd64", "_linux_amd64"},
		{"linux", "arm64", "_linux_arm64"},
		{"darwin", "amd64", "_mac_amd64"},
		{"darwin", "arm64", "_mac_arm64"},
	}

	var builtBinaries []string

	for _, target := range targets {
		outputPath := filepath.Join(outputDir, binaryName+target.suffix)
		fmt.Printf("Building for %s/%s -> %s\n", target.goos, target.goarch, outputPath)

		ldflags := "-s -w -extldflags=-static"
		cmd := exec.Command("go", "build", "-tags=netgo,osusergo", "-trimpath", "-ldflags", ldflags, "-o", outputPath)
		cmd.Dir = srcDir
		cmd.Env = append(os.Environ(),
			"GOOS="+target.goos,
			"GOARCH="+target.goarch,
			"CGO_ENABLED=0",
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to build for %s/%s: %w", target.goos, target.goarch, err)
		}
		builtBinaries = append(builtBinaries, outputPath)
	}

	return builtBinaries, nil
}

func compressWithUPX(binaryPath string) error {
	fmt.Println("Compressing with UPX:", binaryPath)
	return runCommand("upx", []string{"--best", "--lzma", binaryPath})
}

func checkCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
