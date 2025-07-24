package delve_jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"custom-debugger/pkg/utils"
)

// JSON-RPC message structures
type JSONRPCRequest struct {
	ID     interface{} `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// RequestInterceptingReader wraps a Reader and tracks JSON-RPC requests from client -> delve
type RequestInterceptingReader struct {
	Reader           io.Reader
	Name             string
	RequestMethodMap map[string]string
	MapMutex         *sync.Mutex
	Buffer           []byte
	AllRequestCount  int
	ResponseReader   *ResponseInterceptingReader // Reference to response Reader for frame mapping

	// Request modification support
	ModifiedData   []byte // Buffer for modified requests to send to delve
	ModifiedOffset int    // Current position in ModifiedData

}

func (rir *RequestInterceptingReader) Read(p []byte) (n int, err error) {
	// First, check if we have modified data to send to delve
	if rir.ModifiedOffset < len(rir.ModifiedData) {
		// Send modified data instead of reading from client
		remaining := len(rir.ModifiedData) - rir.ModifiedOffset
		bytesToCopy := len(p)
		if remaining < bytesToCopy {
			bytesToCopy = remaining
		}

		copy(p, rir.ModifiedData[rir.ModifiedOffset:rir.ModifiedOffset+bytesToCopy])
		rir.ModifiedOffset += bytesToCopy

		// If we've sent all modified data, reset
		if rir.ModifiedOffset >= len(rir.ModifiedData) {
			rir.ModifiedData = nil
			rir.ModifiedOffset = 0
		}

		log.Printf("%s: %d bytes (modified)", rir.Name, bytesToCopy)
		return bytesToCopy, nil
	}

	// Normal case: read from client
	n, err = rir.Reader.Read(p)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])

		// Append to Buffer for JSON-RPC parsing
		rir.Buffer = append(rir.Buffer, dataCopy...)

		// Try to extract complete JSON-RPC messages and potentially modify them
		modifiedData := rir.parseAndModifyRequests()

		// If we got modified data, we need to replace what we're sending to delve
		if modifiedData != nil {
			// Clear the Buffer since we're replacing the data
			rir.Buffer = nil

			rir.ModifiedData = modifiedData
			rir.ModifiedOffset = 0

			// Send the first part of modified data
			bytesToCopy := len(p)
			if len(modifiedData) < bytesToCopy {
				bytesToCopy = len(modifiedData)
			}

			copy(p, modifiedData[:bytesToCopy])
			rir.ModifiedOffset = bytesToCopy

			log.Printf("%s: %d bytes (replaced with modified)", rir.Name, bytesToCopy)
			return bytesToCopy, err
		}

		log.Printf("%s: %d bytes", rir.Name, n)
	}
	return n, err
}

func (rir *RequestInterceptingReader) parseAndModifyRequests() []byte {
	for len(rir.Buffer) > 0 {
		// Try to find a complete JSON object in the Buffer
		jsonObj, remaining, found := ExtractJSONObject(rir.Buffer)
		if !found {
			break
		}

		// Update Buffer to remaining data
		rir.Buffer = remaining

		// ENHANCED DEBUG: Track ALL requests with unique IDs
		rir.AllRequestCount++
		requestNum := rir.AllRequestCount

		jsonStr := string(jsonObj)
		log.Printf("%s REQUEST #%d (%d bytes): %s", rir.Name, requestNum, len(jsonObj), jsonStr[:utils.Min(150, len(jsonStr))])

		// Parse JSON-RPC request
		var req JSONRPCRequest
		if err := json.Unmarshal(jsonObj, &req); err == nil {
			normalizedID := utils.NormalizeID(req.ID)

			log.Printf("%s JSON-RPC REQUEST ANALYSIS: Request #%d - ID:%v, method:%s",
				rir.Name, requestNum, req.ID, req.Method)
			log.Printf("%s RPC Request #%d: %s (ID: %v)", rir.Name, requestNum, req.Method, req.ID)

			// CHECK FOR STEP OVER COMMAND - Remove auto-stepping complexity
			if req.Method == "RPCServer.Command" {
				if rir.isStepOverCommand(req) {
					log.Printf("%s STEP OVER DETECTED: Request #%d (ID:%v) - will handle normally", rir.Name, requestNum, req.ID)
				}
			}

			// CHECK FOR EVAL REQUEST AND TRANSLATE FRAME NUMBERS
			if req.Method == "RPCServer.Eval" {
				modifiedRequest := rir.translateEvalFrameNumber(jsonObj, remaining, requestNum)
				if modifiedRequest != nil {
					log.Printf("%s *** RETURNING TRANSLATED EVAL REQUEST #%d ***", rir.Name, requestNum)
					return modifiedRequest
				}
			}

			// CHECK FOR OTHER FRAME-BASED REQUESTS AND TRANSLATE FRAME NUMBERS
			if req.Method == "RPCServer.ListLocalVars" || req.Method == "RPCServer.ListFunctionArgs" {
				modifiedRequest := rir.translateFrameBasedRequest(jsonObj, remaining, requestNum, req.Method)
				if modifiedRequest != nil {
					log.Printf("%s *** RETURNING TRANSLATED %s REQUEST #%d ***", rir.Name, req.Method, requestNum)
					return modifiedRequest
				}
			}

			// Track ALL method requests for response correlation
			rir.MapMutex.Lock()

			// For Command requests, store the specific command type for better tracking
			var methodToStore string
			if req.Method == "RPCServer.Command" {
				commandName := rir.extractCommandNameFromRequest(req)
				methodToStore = fmt.Sprintf("RPCServer.Command.%s", commandName)
				log.Printf("%s COMMAND TRACKING: Request #%d (ID:%v) -> %s command", rir.Name, requestNum, req.ID, commandName)
			} else {
				methodToStore = req.Method
			}

			rir.RequestMethodMap[normalizedID] = methodToStore
			log.Printf("%s TRACKING: Request #%d (ID:%v) -> method:%s (total tracked: %d)",
				rir.Name, requestNum, req.ID, methodToStore, len(rir.RequestMethodMap))
			rir.MapMutex.Unlock()

			// Special handling for State method requests
			if req.Method == "RPCServer.State" {
				log.Printf("%s Tracking State request #%d with ID: %v", rir.Name, requestNum, req.ID)
			}

			if req.Method == "RPCServer.Stacktrace" {
				log.Printf("%s Tracking JSON-RPC Stacktrace request #%d with ID: %v", rir.Name, requestNum, req.ID)
			}
		} else {
			log.Printf("%s UNPARSEABLE REQUEST #%d (JSON-RPC): %v", rir.Name, requestNum, err)
			log.Printf("%s Raw data: %s", rir.Name, jsonStr[:utils.Min(200, len(jsonStr))])
		}
	}

	return nil // No modifications needed
}

// translateEvalFrameNumber translates frame numbers in RPCServer.Eval requests from filtered to original
func (rir *RequestInterceptingReader) translateEvalFrameNumber(jsonObj []byte, remaining []byte, requestNum int) []byte {
	log.Printf("%s ENTERING translateEvalFrameNumber for request #%d", rir.Name, requestNum)

	// Parse the JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(jsonObj, &req); err != nil {
		log.Printf("%s Failed to parse Eval request for frame translation: %v", rir.Name, err)
		return nil
	}

	// Extract the EvalIn parameters
	if req.Params == nil {
		log.Printf("%s Eval request has no params", rir.Name)
		return nil
	}

	// Convert params to EvalIn struct
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		log.Printf("%s Failed to marshal Eval params: %v", rir.Name, err)
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
		log.Printf("%s Failed to parse EvalIn params array: %v", rir.Name, err)
		return nil
	}

	if len(paramsArray) == 0 {
		log.Printf("%s EvalIn params array is empty", rir.Name)
		return nil
	}

	evalParams := paramsArray[0]

	originalFrame := evalParams.Scope.Frame
	log.Printf("%s EVAL REQUEST ANALYSIS: expr='%s', goroutine=%d, frame=%d",
		rir.Name, evalParams.Expr, evalParams.Scope.GoroutineID, originalFrame)

	// Get frame mapping from response Reader
	if rir.ResponseReader == nil {
		log.Printf("%s No response Reader available for frame mapping", rir.Name)
		return nil
	}

	rir.ResponseReader.FrameMappingLock.RLock()
	frameMapping := rir.ResponseReader.FrameMapping
	if len(frameMapping) == 0 {
		rir.ResponseReader.FrameMappingLock.RUnlock()
		log.Printf("%s No frame mapping available, request may fail", rir.Name)
		return nil
	}

	// Translate frame number: client's filtered frame -> original delve frame
	translatedFrame, exists := frameMapping[originalFrame]
	rir.ResponseReader.FrameMappingLock.RUnlock()

	if !exists {
		log.Printf("%s Frame %d not found in mapping (available: %v)", rir.Name, originalFrame, frameMapping)
		return nil
	}

	log.Printf("%s FRAME TRANSLATION: filtered frame %d -> original frame %d", rir.Name, originalFrame, translatedFrame)

	// Modify the request with the translated frame number
	evalParams.Scope.Frame = translatedFrame

	// Re-encode the modified parameters as array (JSON-RPC format)
	modifiedParamsArray := []EvalIn{evalParams}
	modifiedParamsBytes, err := json.Marshal(modifiedParamsArray)
	if err != nil {
		log.Printf("%s Failed to marshal modified EvalIn params array: %v", rir.Name, err)
		return nil
	}

	// Convert back to interface{} for the request
	var modifiedParams interface{}
	if err := json.Unmarshal(modifiedParamsBytes, &modifiedParams); err != nil {
		log.Printf("%s Failed to unmarshal modified params array: %v", rir.Name, err)
		return nil
	}

	// Update the request with modified params
	req.Params = modifiedParams

	// Re-encode the complete request
	modifiedRequestBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("%s Failed to marshal modified Eval request: %v", rir.Name, err)
		return nil
	}

	log.Printf("%s Successfully translated Eval request: frame %d -> %d", rir.Name, originalFrame, translatedFrame)

	// Combine modified request with remaining Buffer data
	modifiedBuffer := make([]byte, len(modifiedRequestBytes)+len(remaining))
	copy(modifiedBuffer, modifiedRequestBytes)
	copy(modifiedBuffer[len(modifiedRequestBytes):], remaining)

	return modifiedBuffer
}

// translateFrameBasedRequest translates frame numbers in frame-based requests like ListLocalVars and ListFunctionArgs
func (rir *RequestInterceptingReader) translateFrameBasedRequest(jsonObj []byte, remaining []byte, requestNum int, method string) []byte {
	log.Printf("%s ENTERING translateFrameBasedRequest for %s request #%d", rir.Name, method, requestNum)

	// Parse the JSON-RPC request
	var req JSONRPCRequest
	if err := json.Unmarshal(jsonObj, &req); err != nil {
		log.Printf("%s Failed to parse %s request for frame translation: %v", rir.Name, method, err)
		return nil
	}

	// Extract the parameters
	if req.Params == nil {
		log.Printf("%s %s request has no params", rir.Name, method)
		return nil
	}

	// Convert params to check for frame-based parameters
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		log.Printf("%s Failed to marshal %s params: %v", rir.Name, method, err)
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
		log.Printf("%s Failed to parse %s params array: %v", rir.Name, method, err)
		return nil
	}

	if len(paramsArray) == 0 {
		log.Printf("%s %s params array is empty", rir.Name, method)
		return nil
	}

	params := paramsArray[0]

	originalFrame := params.Scope.Frame
	log.Printf("%s %s REQUEST ANALYSIS: goroutine=%d, frame=%d",
		rir.Name, method, params.Scope.GoroutineID, originalFrame)

	// Get frame mapping from response Reader
	if rir.ResponseReader == nil {
		log.Printf("%s No response Reader available for frame mapping", rir.Name)
		return nil
	}

	rir.ResponseReader.FrameMappingLock.RLock()
	frameMapping := rir.ResponseReader.FrameMapping
	if len(frameMapping) == 0 {
		rir.ResponseReader.FrameMappingLock.RUnlock()
		log.Printf("%s No frame mapping available, %s request may fail", rir.Name, method)
		return nil
	}

	// Translate frame number: client's filtered frame -> original delve frame
	translatedFrame, exists := frameMapping[originalFrame]
	rir.ResponseReader.FrameMappingLock.RUnlock()

	if !exists {
		log.Printf("%s Frame %d not found in mapping for %s (available: %v)", rir.Name, originalFrame, method, frameMapping)
		return nil
	}

	log.Printf("%s %s FRAME TRANSLATION: filtered frame %d -> original frame %d", rir.Name, method, originalFrame, translatedFrame)

	// Modify the request with the translated frame number
	params.Scope.Frame = translatedFrame

	// Re-encode the modified parameters as array (JSON-RPC format)
	modifiedParamsArray := []FrameBasedParams{params}
	modifiedParamsBytes, err := json.Marshal(modifiedParamsArray)
	if err != nil {
		log.Printf("%s Failed to marshal modified %s params array: %v", rir.Name, method, err)
		return nil
	}

	// Convert back to interface{} for the request
	var modifiedParams interface{}
	if err := json.Unmarshal(modifiedParamsBytes, &modifiedParams); err != nil {
		log.Printf("%s Failed to unmarshal modified %s params array: %v", rir.Name, method, err)
		return nil
	}

	// Update the request with modified params
	req.Params = modifiedParams

	// Re-encode the complete request
	modifiedRequestBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("%s Failed to marshal modified %s request: %v", rir.Name, method, err)
		return nil
	}

	log.Printf("%s Successfully translated %s request: frame %d -> %d", rir.Name, method, originalFrame, translatedFrame)

	// Combine modified request with remaining Buffer data
	modifiedBuffer := make([]byte, len(modifiedRequestBytes)+len(remaining))
	copy(modifiedBuffer, modifiedRequestBytes)
	copy(modifiedBuffer[len(modifiedRequestBytes):], remaining)

	return modifiedBuffer
}

// isStepOverCommand checks if a JSON-RPC request is a step over command
func (rir *RequestInterceptingReader) isStepOverCommand(req JSONRPCRequest) bool {
	if req.Params == nil {
		return false
	}

	// Convert params to check the command Name
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

// extractCommandNameFromRequest extracts the actual command Name (next, continue, step, etc.) from a Command request
func (rir *RequestInterceptingReader) extractCommandNameFromRequest(req JSONRPCRequest) string {
	if req.Params == nil {
		return "unknown"
	}

	// Convert params to check the command Name
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

	// Return the actual command Name (next, continue, step, stepout, etc.)
	return paramsArray[0].Name
}
