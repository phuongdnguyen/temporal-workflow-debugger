package locators

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// IsInAdapterCodeByPath checks if a file path is in adapter code
func IsInAdapterCodeByPath(filePath string) bool {
	if filePath == "" {
		return true
	}
	log.Printf("IsInAdapterCodeByPath file: %s \n", filePath)

	// Check if this is user code (should NOT be considered adapter code)
	workingDir := Pwd()
	fmt.Printf("workingDir: %s\n", workingDir)
	if IsUserCodeFile(filePath, workingDir) {
		return false
	}

	// Check for adapter code patterns in the file path
	return strings.Contains(filePath, "replayer-adapter-go/") ||
		strings.Contains(filePath, "replayer.go") ||
		strings.Contains(filePath, "outbound_interceptor.go") ||
		strings.Contains(filePath, "inbound_interceptor.go") ||
		// ALL Temporal SDK code (both versioned and non-versioned paths)
		strings.Contains(filePath, "go.temporal.io/sdk/") ||
		strings.Contains(filePath, "go.temporal.io/sdk@") ||
		// Other GoDelve runtime/reflection code that might be encountered
		strings.Contains(filePath, "/runtime/") ||
		strings.Contains(filePath, "/reflect/") ||
		strings.Contains(filePath, "replayer-adapter-python/") ||
		strings.Contains(filePath, "replayer.py") ||
		strings.Contains(filePath, "replayer-adapter-nodejs/") ||
		strings.Contains(filePath, "replayer.ts")
}

// Pwd returns the current working directory
func Pwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v", err)
		return ""
	}
	// Convert to absolute path for consistent comparison
	absWd, err := filepath.Abs(wd)
	if err != nil {
		log.Printf("Failed to get absolute path for working directory: %v", err)
		return wd
	}
	return absWd
}

// IsUserCodeFile checks if a file path is within the user's working directory
// and not part of framework/adapter code
func IsUserCodeFile(filePath, workingDir string) bool {
	if filePath == "" || workingDir == "" {
		return false
	}

	// Convert file path to absolute path for consistent comparison
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Failed to get absolute path for %s: %v", filePath, err)
		// Fallback to string comparison
		absFilePath = filePath
	}

	// Check if file is within working directory
	isInWorkingDir := strings.HasPrefix(absFilePath, workingDir)

	if !isInWorkingDir {
		// File is outside working directory - definitely not user code
		return false
	}

	// File is in working directory, but check if it's adapter/framework code
	// Exclude known adapter/framework paths even if they're in working directory
	if strings.Contains(filePath, "replayer-adapter-go") ||
		strings.Contains(filePath, "tdlv/") ||
		strings.Contains(filePath, "replayer-adapter-python/") ||
		strings.Contains(filePath, "replayer-adapter-nodejs/") ||
		strings.Contains(filePath, "vendor/") ||
		strings.Contains(filePath, ".git/") {
		return false
	}

	// Also exclude Temporal SDK and GoDelve runtime code (should be outside working dir anyway)
	if strings.Contains(filePath, "go.temporal.io/sdk/") ||
		strings.Contains(filePath, "go.temporal.io/sdk@") ||
		strings.Contains(filePath, "/runtime/") ||
		strings.Contains(filePath, "/reflect/") {
		return false
	}

	log.Printf("User code detected: %s (in working dir: %s)", filePath, workingDir)
	return true
}
