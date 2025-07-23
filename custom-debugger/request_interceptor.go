package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
)

// JSON-RPC message structures
type JSONRPCRequest struct {
	ID     interface{} `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// requestInterceptingReader wraps a reader and tracks JSON-RPC requests
type requestInterceptingReader struct {
	reader           io.Reader
	name             string
	requestMethodMap map[string]string
	mapMutex         *sync.Mutex
	buffer           []byte
	allRequestCount  int
	responseReader   *responseInterceptingReader // Reference to response reader for frame mapping

	// Request modification support
	modifiedData   []byte // Buffer for modified requests to send to delve
	modifiedOffset int    // Current position in modifiedData

	// Step over to continue conversion tracking
	stepOverToContinueMap map[string]string // Maps continue ID -> original step over ID
	stepOverMapMutex      sync.Mutex        // Protects the step over mapping
}

func (rir *requestInterceptingReader) Read(p []byte) (n int, err error) {
	// First, check if we have modified data to send to delve
	if rir.modifiedOffset < len(rir.modifiedData) {
		// Send modified data instead of reading from client
		remaining := len(rir.modifiedData) - rir.modifiedOffset
		bytesToCopy := len(p)
		if remaining < bytesToCopy {
			bytesToCopy = remaining
		}

		copy(p, rir.modifiedData[rir.modifiedOffset:rir.modifiedOffset+bytesToCopy])
		rir.modifiedOffset += bytesToCopy

		// If we've sent all modified data, reset
		if rir.modifiedOffset >= len(rir.modifiedData) {
			rir.modifiedData = nil
			rir.modifiedOffset = 0
		}

		log.Printf("%s: %d bytes (modified)", rir.name, bytesToCopy)
		return bytesToCopy, nil
	}

	// Normal case: read from client
	n, err = rir.reader.Read(p)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])

		// Append to buffer for JSON-RPC parsing
		rir.buffer = append(rir.buffer, dataCopy...)

		// Try to extract complete JSON-RPC messages and potentially modify them
		modifiedData := rir.parseAndModifyRequests()

		// If we got modified data, we need to replace what we're sending to delve
		if modifiedData != nil {
			// Clear the buffer since we're replacing the data
			rir.buffer = nil

			rir.modifiedData = modifiedData
			rir.modifiedOffset = 0

			// Send the first part of modified data
			bytesToCopy := len(p)
			if len(modifiedData) < bytesToCopy {
				bytesToCopy = len(modifiedData)
			}

			copy(p, modifiedData[:bytesToCopy])
			rir.modifiedOffset = bytesToCopy

			log.Printf("%s: %d bytes (replaced with modified)", rir.name, bytesToCopy)
			return bytesToCopy, err
		}

		log.Printf("%s: %d bytes", rir.name, n)
	}
	return n, err
}

func (rir *requestInterceptingReader) parseAndModifyRequests() []byte {
	for len(rir.buffer) > 0 {
		// Try to find a complete JSON object in the buffer
		jsonObj, remaining, found := extractJSONObject(rir.buffer)
		if !found {
			break
		}

		// Update buffer to remaining data
		rir.buffer = remaining

		// ENHANCED DEBUG: Track ALL requests with unique IDs
		rir.allRequestCount++
		requestNum := rir.allRequestCount

		jsonStr := string(jsonObj)
		log.Printf("%s REQUEST #%d (%d bytes): %s", rir.name, requestNum, len(jsonObj), jsonStr[:min(150, len(jsonStr))])

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(jsonObj, &req); err == nil {
			normalizedID := normalizeID(req.ID)

			log.Printf("%s JSON-RPC REQUEST ANALYSIS: Request #%d - ID:%v, method:%s",
				rir.name, requestNum, req.ID, req.Method)
			log.Printf("%s RPC Request #%d: %s (ID: %v)", rir.name, requestNum, req.Method, req.ID)

			// CHECK FOR STEP OVER COMMAND - Remove auto-stepping complexity
			if req.Method == "RPCServer.Command" {
				if rir.isStepOverCommand(req) {
					log.Printf("%s STEP OVER DETECTED: Request #%d (ID:%v) - will handle normally", rir.name, requestNum, req.ID)
				}
			}

			// CHECK FOR EVAL REQUEST AND TRANSLATE FRAME NUMBERS
			if req.Method == "RPCServer.Eval" {
				modifiedRequest := rir.translateEvalFrameNumber(jsonObj, remaining, requestNum)
				if modifiedRequest != nil {
					log.Printf("%s *** RETURNING TRANSLATED EVAL REQUEST #%d ***", rir.name, requestNum)
					return modifiedRequest
				}
			}

			// CHECK FOR OTHER FRAME-BASED REQUESTS AND TRANSLATE FRAME NUMBERS
			if req.Method == "RPCServer.ListLocalVars" || req.Method == "RPCServer.ListFunctionArgs" {
				modifiedRequest := rir.translateFrameBasedRequest(jsonObj, remaining, requestNum, req.Method)
				if modifiedRequest != nil {
					log.Printf("%s *** RETURNING TRANSLATED %s REQUEST #%d ***", rir.name, req.Method, requestNum)
					return modifiedRequest
				}
			}

			// Track ALL method requests for response correlation
			rir.mapMutex.Lock()

			// For Command requests, store the specific command type for better tracking
			var methodToStore string
			if req.Method == "RPCServer.Command" {
				commandName := rir.extractCommandNameFromRequest(req)
				methodToStore = fmt.Sprintf("RPCServer.Command.%s", commandName)
				log.Printf("%s COMMAND TRACKING: Request #%d (ID:%v) -> %s command", rir.name, requestNum, req.ID, commandName)
			} else {
				methodToStore = req.Method
			}

			rir.requestMethodMap[normalizedID] = methodToStore
			log.Printf("%s TRACKING: Request #%d (ID:%v) -> method:%s (total tracked: %d)",
				rir.name, requestNum, req.ID, methodToStore, len(rir.requestMethodMap))
			rir.mapMutex.Unlock()

			// Special handling for State method requests
			if req.Method == "RPCServer.State" {
				log.Printf("%s Tracking State request #%d with ID: %v", rir.name, requestNum, req.ID)
			}

			if req.Method == "RPCServer.Stacktrace" {
				log.Printf("%s Tracking JSON-RPC Stacktrace request #%d with ID: %v", rir.name, requestNum, req.ID)
			}
		} else {
			log.Printf("%s UNPARSEABLE REQUEST #%d (JSON-RPC): %v", rir.name, requestNum, err)
			log.Printf("%s Raw data: %s", rir.name, jsonStr[:min(200, len(jsonStr))])
		}
	}

	return nil // No modifications needed
}

// translateEvalFrameNumber translates frame numbers in RPCServer.Eval requests from filtered to original
func (rir *requestInterceptingReader) translateEvalFrameNumber(jsonObj []byte, remaining []byte, requestNum int) []byte {
	log.Printf("%s ENTERING translateEvalFrameNumber for request #%d", rir.name, requestNum)

	// Parse the JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(jsonObj, &req); err != nil {
		log.Printf("%s Failed to parse Eval request for frame translation: %v", rir.name, err)
		return nil
	}

	// Extract the EvalIn parameters
	if req.Params == nil {
		log.Printf("%s Eval request has no params", rir.name)
		return nil
	}

	// Convert params to EvalIn struct
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		log.Printf("%s Failed to marshal Eval params: %v", rir.name, err)
		return nil
	}

	// JSON-RPC params come as an array: [EvalIn], not EvalIn directly
	// We need to define the EvalIn struct to parse the parameters
	type EvalScope struct {
		GoroutineID  int64 `json:"GoroutineID"`
		Frame        int   `json:"Frame"`
		DeferredCall int   `json:"DeferredCall"`
	}

	type EvalIn struct {
		Scope EvalScope `json:"Scope"`
		Expr  string    `json:"Expr"`
	}

	// First parse as array to get the first element
	var paramsArray []EvalIn
	if err := json.Unmarshal(paramsBytes, &paramsArray); err != nil {
		log.Printf("%s Failed to parse EvalIn params array: %v", rir.name, err)
		return nil
	}

	if len(paramsArray) == 0 {
		log.Printf("%s EvalIn params array is empty", rir.name)
		return nil
	}

	evalParams := paramsArray[0]

	originalFrame := evalParams.Scope.Frame
	log.Printf("%s EVAL REQUEST ANALYSIS: expr='%s', goroutine=%d, frame=%d",
		rir.name, evalParams.Expr, evalParams.Scope.GoroutineID, originalFrame)

	// Get frame mapping from response reader
	if rir.responseReader == nil {
		log.Printf("%s No response reader available for frame mapping", rir.name)
		return nil
	}

	rir.responseReader.frameMappingLock.RLock()
	frameMapping := rir.responseReader.frameMapping
	if len(frameMapping) == 0 {
		rir.responseReader.frameMappingLock.RUnlock()
		log.Printf("%s No frame mapping available, request may fail", rir.name)
		return nil
	}

	// Translate frame number: client's filtered frame -> original delve frame
	translatedFrame, exists := frameMapping[originalFrame]
	rir.responseReader.frameMappingLock.RUnlock()

	if !exists {
		log.Printf("%s Frame %d not found in mapping (available: %v)", rir.name, originalFrame, frameMapping)
		return nil
	}

	log.Printf("%s FRAME TRANSLATION: filtered frame %d -> original frame %d", rir.name, originalFrame, translatedFrame)

	// Modify the request with the translated frame number
	evalParams.Scope.Frame = translatedFrame

	// Re-encode the modified parameters as array (JSON-RPC format)
	modifiedParamsArray := []EvalIn{evalParams}
	modifiedParamsBytes, err := json.Marshal(modifiedParamsArray)
	if err != nil {
		log.Printf("%s Failed to marshal modified EvalIn params array: %v", rir.name, err)
		return nil
	}

	// Convert back to interface{} for the request
	var modifiedParams interface{}
	if err := json.Unmarshal(modifiedParamsBytes, &modifiedParams); err != nil {
		log.Printf("%s Failed to unmarshal modified params array: %v", rir.name, err)
		return nil
	}

	// Update the request with modified params
	req.Params = modifiedParams

	// Re-encode the complete request
	modifiedRequestBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("%s Failed to marshal modified Eval request: %v", rir.name, err)
		return nil
	}

	log.Printf("%s Successfully translated Eval request: frame %d -> %d", rir.name, originalFrame, translatedFrame)

	// Combine modified request with remaining buffer data
	modifiedBuffer := make([]byte, len(modifiedRequestBytes)+len(remaining))
	copy(modifiedBuffer, modifiedRequestBytes)
	copy(modifiedBuffer[len(modifiedRequestBytes):], remaining)

	return modifiedBuffer
}

// translateFrameBasedRequest translates frame numbers in frame-based requests like ListLocalVars and ListFunctionArgs
func (rir *requestInterceptingReader) translateFrameBasedRequest(jsonObj []byte, remaining []byte, requestNum int, method string) []byte {
	log.Printf("%s ENTERING translateFrameBasedRequest for %s request #%d", rir.name, method, requestNum)

	// Parse the JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(jsonObj, &req); err != nil {
		log.Printf("%s Failed to parse %s request for frame translation: %v", rir.name, method, err)
		return nil
	}

	// Extract the parameters
	if req.Params == nil {
		log.Printf("%s %s request has no params", rir.name, method)
		return nil
	}

	// Convert params to check for frame-based parameters
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		log.Printf("%s Failed to marshal %s params: %v", rir.name, method, err)
		return nil
	}

	// JSON-RPC params come as an array: [FrameBasedParams], not FrameBasedParams directly
	// Define parameter structs for frame-based requests
	type EvalScope struct {
		GoroutineID  int64 `json:"GoroutineID"`
		Frame        int   `json:"Frame"`
		DeferredCall int   `json:"DeferredCall"`
	}

	type FrameBasedParams struct {
		Scope EvalScope `json:"Scope"`
	}

	// First parse as array to get the first element
	var paramsArray []FrameBasedParams
	if err := json.Unmarshal(paramsBytes, &paramsArray); err != nil {
		log.Printf("%s Failed to parse %s params array: %v", rir.name, method, err)
		return nil
	}

	if len(paramsArray) == 0 {
		log.Printf("%s %s params array is empty", rir.name, method)
		return nil
	}

	params := paramsArray[0]

	originalFrame := params.Scope.Frame
	log.Printf("%s %s REQUEST ANALYSIS: goroutine=%d, frame=%d",
		rir.name, method, params.Scope.GoroutineID, originalFrame)

	// Get frame mapping from response reader
	if rir.responseReader == nil {
		log.Printf("%s No response reader available for frame mapping", rir.name)
		return nil
	}

	rir.responseReader.frameMappingLock.RLock()
	frameMapping := rir.responseReader.frameMapping
	if len(frameMapping) == 0 {
		rir.responseReader.frameMappingLock.RUnlock()
		log.Printf("%s No frame mapping available, %s request may fail", rir.name, method)
		return nil
	}

	// Translate frame number: client's filtered frame -> original delve frame
	translatedFrame, exists := frameMapping[originalFrame]
	rir.responseReader.frameMappingLock.RUnlock()

	if !exists {
		log.Printf("%s Frame %d not found in mapping for %s (available: %v)", rir.name, originalFrame, method, frameMapping)
		return nil
	}

	log.Printf("%s %s FRAME TRANSLATION: filtered frame %d -> original frame %d", rir.name, method, originalFrame, translatedFrame)

	// Modify the request with the translated frame number
	params.Scope.Frame = translatedFrame

	// Re-encode the modified parameters as array (JSON-RPC format)
	modifiedParamsArray := []FrameBasedParams{params}
	modifiedParamsBytes, err := json.Marshal(modifiedParamsArray)
	if err != nil {
		log.Printf("%s Failed to marshal modified %s params array: %v", rir.name, method, err)
		return nil
	}

	// Convert back to interface{} for the request
	var modifiedParams interface{}
	if err := json.Unmarshal(modifiedParamsBytes, &modifiedParams); err != nil {
		log.Printf("%s Failed to unmarshal modified %s params array: %v", rir.name, method, err)
		return nil
	}

	// Update the request with modified params
	req.Params = modifiedParams

	// Re-encode the complete request
	modifiedRequestBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("%s Failed to marshal modified %s request: %v", rir.name, method, err)
		return nil
	}

	log.Printf("%s Successfully translated %s request: frame %d -> %d", rir.name, method, originalFrame, translatedFrame)

	// Combine modified request with remaining buffer data
	modifiedBuffer := make([]byte, len(modifiedRequestBytes)+len(remaining))
	copy(modifiedBuffer, modifiedRequestBytes)
	copy(modifiedBuffer[len(modifiedRequestBytes):], remaining)

	return modifiedBuffer
}

// isStepOverCommand checks if a JSON-RPC request is a step over command
func (rir *requestInterceptingReader) isStepOverCommand(req JSONRPCRequest) bool {
	if req.Params == nil {
		return false
	}

	// Convert params to check the command name
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return false
	}

	// JSON-RPC Command params come as an array: [CommandIn]
	type CommandIn struct {
		Name string `json:"Name"`
	}

	var paramsArray []CommandIn
	if err := json.Unmarshal(paramsBytes, &paramsArray); err != nil {
		return false
	}

	if len(paramsArray) == 0 {
		return false
	}

	// Check if this is a "next" command (step over)
	return paramsArray[0].Name == "next"
}

// extractCommandNameFromRequest extracts the actual command name (next, continue, step, etc.) from a Command request
func (rir *requestInterceptingReader) extractCommandNameFromRequest(req JSONRPCRequest) string {
	if req.Params == nil {
		return "unknown"
	}

	// Convert params to check the command name
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return "unknown"
	}

	// JSON-RPC Command params come as an array: [CommandIn]
	type CommandIn struct {
		Name string `json:"Name"`
	}

	var paramsArray []CommandIn
	if err := json.Unmarshal(paramsBytes, &paramsArray); err != nil {
		return "unknown"
	}

	if len(paramsArray) == 0 {
		return "unknown"
	}

	// Return the actual command name (next, continue, step, stepout, etc.)
	return paramsArray[0].Name
}
