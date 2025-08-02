package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-delve/delve/pkg/gobuild"
)

// Min returns the smaller of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsConnectionClosedError checks if an error is due to a closed network connection
// This helps distinguish between normal connection closes and actual errors
func IsConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Common patterns for connection closed errors
	closedPatterns := []string{
		"use of closed network connection",
		"connection reset by peer",
		"broken pipe",
		"EOF",
		"io: read/write on closed pipe",
	}

	for _, pattern := range closedPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check for specific error types
	if err == io.EOF {
		return true
	}

	// Check for net.OpError with specific operations
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Op == "read" || opErr.Op == "write" {
			if strings.Contains(opErr.Err.Error(), "closed") {
				return true
			}
		}
	}

	return false
}

// DialWithRetry attempts to connect to Delve server with retry logic
func DialWithRetry(addr string, maxRetries int, delay time.Duration) (net.Conn, error) {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		// Set a connection timeout
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		log.Printf("Failed to connect to Delve (attempt %d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, lastErr)
}

func BuildBinary(args []string, isTest bool) (string, bool) {
	var debugname string
	var err error
	if isTest {
		debugname = gobuild.DefaultDebugBinaryPath("tdlv_debug.test")
	} else {
		debugname = gobuild.DefaultDebugBinaryPath("__tdlv_debug_bin")
	}

	if isTest {
		err = gobuild.GoTestBuild(debugname, args, "")
	} else {
		err = gobuild.GoBuild(debugname, args, "")
	}
	if err != nil {
		gobuild.Remove(debugname)
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return "", false
	}
	return debugname, true
}

// NormalizeID converts various ID types to a consistent string representation
// for reliable map lookups across JSON marshaling/unmarshalling
func NormalizeID(id interface{}) string {
	if id == nil {
		return "null"
	}

	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case json.Number:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
