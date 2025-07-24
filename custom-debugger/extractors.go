package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/go-delve/delve/service/api"
)

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
