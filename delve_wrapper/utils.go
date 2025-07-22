package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-delve/delve/pkg/gobuild"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isConnectionClosedError checks if an error is due to a closed network connection
// This helps distinguish between normal connection closes and actual errors
func isConnectionClosedError(err error) bool {
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

// dialDelveWithRetry attempts to connect to Delve server with retry logic
func dialDelveWithRetry(addr string, maxRetries int, delay time.Duration) (net.Conn, error) {
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

func buildBinary(args []string, isTest bool) (string, bool) {
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

func handleClientConnection(clientTCP net.Conn) {
	clientAddr := clientTCP.RemoteAddr().String()
	log.Printf("New client connected from %s", clientAddr)

	// Set keep-alive to detect dead connections
	if tcpConn, ok := clientTCP.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Set read/write timeouts to prevent hanging
	clientTCP.SetReadDeadline(time.Time{})  // No read timeout for client
	clientTCP.SetWriteDeadline(time.Time{}) // No write timeout for client

	// Dial real Delve with retry logic
	delveTCP, err := dialDelveWithRetry("localhost:2345", 3, time.Second)
	if err != nil {
		log.Printf("Error connecting to Delve server for %s after retries: %v", clientAddr, err)
		clientTCP.Close()
		return
	}
	log.Println("Connected to Delve server")

	// Set keep-alive for delve connection too
	if tcpConn, ok := delveTCP.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// Set timeouts for delve connection as well
	delveTCP.SetReadDeadline(time.Time{})  // No read timeout for delve
	delveTCP.SetWriteDeadline(time.Time{}) // No write timeout for delve

	// Channel to signal when one side closes
	done := make(chan struct{}, 2)

	// Map to track request IDs to method names for response interception
	requestMethodMap := make(map[string]string)
	var mapMutex sync.Mutex

	// Create delve client for auto-stepping operations
	delveClient := rpc2.NewClient("localhost:2345")

	// Create response interceptor first so we can reference it
	delveReader := &responseInterceptingReader{
		reader:           delveTCP,
		name:             fmt.Sprintf("[%s] Delve->Client", clientAddr),
		requestMethodMap: requestMethodMap,
		mapMutex:         &mapMutex,
		clientAddr:       clientAddr,

		// Enhanced debugging counters
		// Enhanced debugging counters (no locks needed for debug counters)
		stackTraceCount:     0,
		stackFrameDataCount: 0,
		allResponseCount:    0,
		mainThreadMutex:     sync.Mutex{},

		// Frame mapping for JSON-RPC stacktrace filtering
		frameMapping:       make(map[int]int),
		frameMappingLock:   sync.RWMutex{},
		currentGoroutineID: -1,

		// Auto-stepping infrastructure
		delveConnection:    delveTCP,
		delveClient:        delveClient,
		autoSteppingActive: false,
		steppingMutex:      sync.Mutex{},

		// Current state tracking for sentinel breakpoint detection
		currentFile:     "",             // Current file location
		currentFunction: "",             // Current function name
		currentLine:     0,              // Current line number
		stateMutex:      sync.RWMutex{}, // Protects current state fields

		// Reference to request reader for step over tracking
		requestReader: nil, // Will be set after clientReader is created
	}

	// Ensure both connections are closed when function exits
	defer func() {
		log.Printf("Closing connections for client %s", clientAddr)
		clientTCP.Close()
		delveTCP.Close()

		// Log debugging summary
		delveReader.logDebuggingSummary()

		log.Printf("Client %s disconnected", clientAddr)
	}()

	// Handle client->delve with request tracking and transparent forwarding
	go func() {
		defer func() {
			log.Printf("Client->Delve goroutine ending for %s", clientAddr)
			done <- struct{}{}
		}()

		log.Println("Client->Delve")

		// Create intercepting reader that also tracks requests
		clientReader := &requestInterceptingReader{
			reader:           clientTCP,
			name:             fmt.Sprintf("[%s] Client->Delve", clientAddr),
			requestMethodMap: requestMethodMap,
			mapMutex:         &mapMutex,
			responseReader:   delveReader,

			// Initialize step over to continue conversion tracking
			stepOverToContinueMap: make(map[string]string),
			stepOverMapMutex:      sync.Mutex{},
		}

		// Set the reference from response reader to request reader for step over tracking
		delveReader.requestReader = clientReader

		if _, err := io.Copy(delveTCP, clientReader); err != nil {
			// Check if this is a normal connection close vs an actual error
			if isConnectionClosedError(err) {
				log.Printf("Client->Delve connection closed normally for %s", clientAddr)
			} else {
				log.Printf("Error copying client->delve for %s: %v", clientAddr, err)
			}
		}
	}()

	// Handle delve->client with response interception and transparent forwarding
	go func() {
		defer func() {
			log.Printf("Delve->Client goroutine ending for %s", clientAddr)
			done <- struct{}{}
		}()

		log.Println("Delve->Client")

		if _, err := io.Copy(clientTCP, delveReader); err != nil {
			// Check if this is a normal connection close vs an actual error
			if isConnectionClosedError(err) {
				log.Printf("Delve->Client connection closed normally for %s", clientAddr)
			} else {
				log.Printf("Error copying delve->client for %s: %v", clientAddr, err)
			}
		}
	}()

	// Wait for either direction to close with timeout
	select {
	case <-done:
		log.Printf("Connection closed for client %s", clientAddr)
	case <-time.After(30 * time.Minute): // 30 minute timeout
		log.Printf("⚠️  Connection timeout for client %s, forcing close", clientAddr)
	}
}

// extractJSONObject finds and extracts the first complete JSON object from the buffer
// Handles both DAP (with Content-Length headers) and JSON-RPC formats
func extractJSONObject(data []byte) (jsonObj []byte, remaining []byte, found bool) {
	if len(data) == 0 {
		return nil, data, false
	}

	// Check if this is a DAP message (starts with Content-Length header)
	if bytes.HasPrefix(data, []byte("Content-Length:")) {
		return extractDAPMessage(data)
	}

	// Fall back to JSON-RPC parsing
	return extractJSONRPCMessage(data)
}

// extractDAPMessage extracts a DAP message with Content-Length header
func extractDAPMessage(data []byte) (jsonObj []byte, remaining []byte, found bool) {
	// DAP format: Content-Length: XXX\r\n\r\n{JSON}

	// Safety check: don't process unreasonably large data
	const maxMessageSize = 1024 * 1024 // 1MB per message
	if len(data) > maxMessageSize {
		// For DAP, we need the header so we can't just truncate arbitrarily
		// But we can limit our search space
		data = data[:maxMessageSize]
	}

	// Find the end of headers (\r\n\r\n)
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		return nil, data, false // Headers not complete yet
	}

	// Extract the header part
	headerPart := data[:headerEnd]

	// Parse Content-Length with better error handling
	contentLengthLine := string(headerPart)
	var contentLength int
	if n, err := fmt.Sscanf(contentLengthLine, "Content-Length: %d", &contentLength); n != 1 || err != nil {
		// Invalid Content-Length header, fall back to JSON-RPC
		return extractJSONRPCMessage(data)
	}

	// Sanity check on content length
	if contentLength < 0 || contentLength > 10*1024*1024 { // 10MB limit
		// Invalid content length, fall back to JSON-RPC
		return extractJSONRPCMessage(data)
	}

	// Calculate where the JSON payload starts and ends
	jsonStart := headerEnd + 4 // Skip \r\n\r\n
	jsonEnd := jsonStart + contentLength

	// Check if we have the complete message
	if len(data) < jsonEnd {
		return nil, data, false // Message not complete yet
	}

	// Extract JSON payload and remaining data
	jsonObj = data[jsonStart:jsonEnd]
	remaining = data[jsonEnd:]

	return jsonObj, remaining, true
}

// extractJSONRPCMessage extracts a JSON-RPC message (plain JSON object)
func extractJSONRPCMessage(data []byte) (jsonObj []byte, remaining []byte, found bool) {
	// Safety check: don't process unreasonably large data
	const maxMessageSize = 1024 * 1024 // 1MB per message
	if len(data) > maxMessageSize {
		// Try to find a JSON object in the first part of the data
		data = data[:maxMessageSize]
	}

	// Find the start of a JSON object
	start := bytes.IndexByte(data, '{')
	if start == -1 {
		return nil, data, false
	}

	// Find the matching closing brace
	braceCount := 0
	inString := false
	escaped := false

	// Safety limit to prevent infinite loops
	maxIterations := len(data)
	iterations := 0

	for i := start; i < len(data) && iterations < maxIterations; i++ {
		iterations++
		char := data[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
			if braceCount == 0 {
				// Found complete JSON object
				jsonObj := data[start : i+1]
				remaining := data[i+1:]
				return jsonObj, remaining, true
			}
		}
	}

	// No complete JSON object found or hit iteration limit
	return nil, data, false
}

// normalizeID converts various ID types to a consistent string representation
// for reliable map lookups across JSON marshaling/unmarshaling
func normalizeID(id interface{}) string {
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

// extractLocationFromCommandResponse helper function to extract location info from command response
func extractLocationFromCommandResponse(jsonObj []byte) *struct {
	File     string
	Line     int
	Function string
} {
	var response JSONRPCResponse
	if err := json.Unmarshal(jsonObj, &response); err != nil {
		return nil
	}

	if response.Result == nil {
		return nil
	}

	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil
	}

	var commandOut struct {
		State *api.DebuggerState `json:"State"`
	}
	if err := json.Unmarshal(resultBytes, &commandOut); err != nil {
		return nil
	}

	if commandOut.State == nil || commandOut.State.CurrentThread == nil {
		return nil
	}

	result := &struct {
		File     string
		Line     int
		Function string
	}{
		File: commandOut.State.CurrentThread.File,
		Line: commandOut.State.CurrentThread.Line,
	}

	// Try to get function name
	if commandOut.State.CurrentThread.BreakpointInfo != nil &&
		len(commandOut.State.CurrentThread.BreakpointInfo.Stacktrace) > 0 {
		topFrame := commandOut.State.CurrentThread.BreakpointInfo.Stacktrace[0]
		if topFrame.Function != nil {
			result.Function = topFrame.Function.Name()
		}
	}

	return result
}
