package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/go-delve/delve/service/api"
)

// ExtractJSONObject finds and extracts the first complete JSON object from the buffer
// Handles both DAP (with Content-Length headers) and JSON-RPC formats
// func ExtractJSONObject(data []byte) (jsonObj []byte, remaining []byte, found bool) {
// 	if len(data) == 0 {
// 		return nil, data, false
// 	}
//
// 	// Check if this is a DAP message (starts with Content-Length header)
// 	if bytes.HasPrefix(data, []byte("Content-Length:")) {
// 		return ExtractDAPMessage(data)
// 	}
//
// 	// Fall back to JSON-RPC parsing
// 	return ExtractJSONRPCMessage(data)
// }

// ExtractDAPMessage extracts a DAP message with Content-Length header
func ExtractDAPMessage(data []byte) (jsonObj []byte, remainingCompletedJsonObjs []byte, found bool,
	remainingIncompleted []byte) {
	if len(data) == 0 {
		log.Printf("ExtractDAPMessage, input data is empty")
		return nil, data, false, nil
	}
	fmt.Printf("#########\n ExtractDAPMessage: input data:\n %s \n########\n", string(data))
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
		log.Printf("ExtractDAPMessage, header is not complete")
		return nil, data, false, nil // Headers not complete yet
	}

	// Extract the header part
	headerPart := data[:headerEnd]

	// Parse Content-Length with better error handling
	contentLengthLine := string(headerPart)
	var contentLength int
	if n, err := fmt.Sscanf(contentLengthLine, "Content-Length: %d", &contentLength); n != 1 || err != nil {
		// Invalid Content-Length header, fall back to JSON-RPC
		// return ExtractJSONRPCMessage(data)
		log.Printf("ExtractDAPMessage, invalid Content-Length header")
		return nil, data, false, nil
	}

	// Sanity check on content length
	if contentLength < 0 || contentLength > 10*1024*1024 { // 10MB limit
		// Invalid content length, fall back to JSON-RPC
		// return ExtractJSONRPCMessage(data)
		log.Printf("ExtractDAPMessage, Content-Length header value out of range")
		return nil, data, false, nil
	}

	// Calculate where the JSON payload starts and ends
	jsonStart := headerEnd + 4 // Skip \r\n\r\n
	jsonEnd := jsonStart + contentLength

	// Check if we have the complete message
	if len(data) < jsonEnd {
		log.Printf("ExtractDAPMessage, message is not complete yet")
		return nil, data, false, nil // Message not complete yet
	}

	// Extract JSON payload and remainingCompletedJsonObjs data
	jsonObj = data[jsonStart:jsonEnd]
	log.Printf("ExtractDAPMessage, jsonObj (skip content-length) is %s\n", string(jsonObj))
	// Remaining in buffer: complete jsonObjects along with possible an in-complete jsonObject
	// We should only forward completed ones
	remaining := data[jsonEnd:]
	offset := FirstInvalidDAP(remaining)
	if offset == -1 {
		log.Printf("ExtractDAPMessage, the remainings are valid json objects %s\n", string(remaining))
		return jsonObj, remaining, true, nil
	}
	log.Printf("ExtractDAPMessage, remaining incompleted jsonObjs is %s\n", string(remaining[offset:]))
	return jsonObj, remaining[:offset], true, remaining[offset:]
}

// FirstInvalidDAP scans consecutive DAP frames in buf and
// returns the byte offset where the stream stops being well-formed.
// It returns -1 when everything up to len(buf) is valid.
func FirstInvalidDAP(buf []byte) int {
	if len(buf) == 0 {
		return -1
	}

	const maxBody = 10 * 1024 * 1024 // 10 MiB hard limit per frame

	off := 0
	for off < len(buf) {
		startOffset := off // Remember where this frame started
		contentLength := -1
		headerComplete := false

		// -------- Parse header line-by-line --------
		for off < len(buf) {
			// Find end of current line
			nl := bytes.IndexByte(buf[off:], '\n')
			if nl == -1 {
				// No more newlines - header is incomplete
				return startOffset
			}

			line := bytes.TrimSpace(buf[off : off+nl])
			off += nl + 1 // skip '\n'

			// blank line marks end-of-headers
			if len(line) == 0 {
				headerComplete = true
				break
			}

			// Parse Content-Length header (case-insensitive)
			const pfx = "content-length:"
			if bytes.HasPrefix(bytes.ToLower(line), []byte(pfx)) {
				val := bytes.TrimSpace(line[len(pfx):])
				cl, err := strconv.Atoi(string(val))
				if err != nil || cl < 0 {
					return startOffset // malformed number
				}
				contentLength = cl
			}
		}

		// Check if header was completed properly
		if !headerComplete {
			return startOffset
		}

		// missing or unreasonable Content-Length
		if contentLength < 0 || contentLength > maxBody {
			return startOffset
		}

		// is the body complete?
		if off+contentLength > len(buf) {
			return startOffset // truncated body - frame starts at startOffset
		}

		body := buf[off : off+contentLength]
		if !json.Valid(body) { // invalid JSON payload
			return startOffset // invalid JSON - frame starts at startOffset
		}

		// move to next frame
		off += contentLength
	}

	return -1 // everything valid
}

// ExtractJSONRPCMessage extracts a JSON-RPC message (plain JSON object)
func ExtractJSONRPCMessage(data []byte) (jsonObj []byte, remaining []byte, found bool) {
	if len(data) == 0 {
		return nil, data, false
	}
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

// ExtractLocationFromCommandResponse helper function to extract location info from command response
func ExtractLocationFromCommandResponse(jsonObj []byte) *struct {
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
