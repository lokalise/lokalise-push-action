package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

// Configurations
const (
	binaryName   = "store_translation_paths"
	srcDir       = "src/store_translation_paths"
	upxAvailable = "upx"
)

func main() {
	// Define binDir relative to the project root
	binDir := filepath.Join(getProjectRoot(), "bin")

	// Ensure the bin directory exists
	if err := os.MkdirAll(binDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create bin directory: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(binDir, binaryName)
	if err := buildBinary(binaryPath); err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	// Compress the binary with UPX, if available
	if checkCommand(upxAvailable) {
		if err := compressWithUPX(binaryPath); err != nil {
			log.Printf("Compression failed: %v", err)
		}
	} else {
		fmt.Println("UPX not found; skipping compression.")
	}

	fmt.Printf("Build complete. Binary located at %s\n", binaryPath)
}

// getProjectRoot returns the absolute path of the project root
func getProjectRoot() string {
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get project root: %v", err)
	}
	return root
}

// buildBinary compiles the Go binary
func buildBinary(outputPath string) error {
	fmt.Println("Building binary for Linux (Ubuntu)...")
	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outputPath)
	cmd.Dir = srcDir

	// Set environment variables for cross-compilation to Linux
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// compressWithUPX compresses the binary with UPX, if available
func compressWithUPX(binaryPath string) error {
	fmt.Println("Compressing binary with UPX...")
	cmd := exec.Command("upx", "--best", "--lzma", binaryPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// checkCommand checks if a command is available on the system
func checkCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
