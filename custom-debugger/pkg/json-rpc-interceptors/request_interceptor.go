package json_rpc_interceptors

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"custom-debugger/pkg/extractors"
	"custom-debugger/pkg/utils"
)

// RequestInterceptingReader wraps a reader and tracks JSON-RPC requests from client -> delve
type RequestInterceptingReader struct {
	reader           io.Reader
	logPrefix        string
	requestMethodMap map[string]string
	mapMutex         *sync.Mutex
	buffer           []byte
	allRequestCount  int
	responseReader   *ResponseInterceptingReader // Reference to response reader for frame mapping

	// Request modification support
	modifiedData   []byte // buffer for modified requests to send to delve
	modifiedOffset int    // Current position in modifiedData

}

func NewRequestInterceptingReader(reader io.Reader, logPrefix string, requestMethodMap map[string]string, mapMutex *sync.Mutex, responseReader *ResponseInterceptingReader) *RequestInterceptingReader {
	return &RequestInterceptingReader{
		reader:           reader,
		logPrefix:        logPrefix,
		requestMethodMap: requestMethodMap,
		mapMutex:         mapMutex,
		responseReader:   responseReader,
	}
}

func (rir *RequestInterceptingReader) Read(p []byte) (n int, err error) {
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

		log.Printf("%s: %d bytes (modified)", rir.logPrefix, bytesToCopy)
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

			log.Printf("%s: %d bytes (replaced with modified)", rir.logPrefix, bytesToCopy)
			return bytesToCopy, err
		}

		log.Printf("%s: %d bytes", rir.logPrefix, n)
	}
	return n, err
}

func (rir *RequestInterceptingReader) parseAndModifyRequests() []byte {
	for len(rir.buffer) > 0 {
		// Try to find a complete JSON object in the buffer
		jsonObj, remaining, found := extractors.ExtractJSONRPCMessage(rir.buffer)
		if !found {
			break
		}

		// Update buffer to remaining data
		rir.buffer = remaining

		// Track all requests with unique IDs
		rir.allRequestCount++
		requestNum := rir.allRequestCount

		jsonStr := string(jsonObj)
		log.Printf("%s REQUEST #%d (%d bytes): %s", rir.logPrefix, requestNum, len(jsonObj), jsonStr[:utils.Min(150, len(jsonStr))])

		// Parse JSON-RPC request
		var req extractors.JSONRPCRequest
		if err := json.Unmarshal(jsonObj, &req); err == nil {
			normalizedID := utils.NormalizeID(req.ID)

			log.Printf("%s JSON-RPC REQUEST ANALYSIS: Request #%d - ID:%v, method:%s",
				rir.logPrefix, requestNum, req.ID, req.Method)
			log.Printf("%s RPC Request #%d: %s (ID: %v)", rir.logPrefix, requestNum, req.Method, req.ID)

			// CHECK FOR EVAL REQUEST AND TRANSLATE FRAME NUMBERS
			if req.Method == "RPCServer.Eval" {
				modifiedRequest := rir.translateEvalFrameNumber(jsonObj, remaining, requestNum)
				if modifiedRequest != nil {
					log.Printf("%s *** RETURNING TRANSLATED EVAL REQUEST #%d ***", rir.logPrefix, requestNum)
					return modifiedRequest
				}
			}

			// CHECK FOR OTHER FRAME-BASED REQUESTS AND TRANSLATE FRAME NUMBERS
			if req.Method == "RPCServer.ListLocalVars" || req.Method == "RPCServer.ListFunctionArgs" {
				modifiedRequest := rir.translateFrameBasedRequest(jsonObj, remaining, requestNum, req.Method)
				if modifiedRequest != nil {
					log.Printf("%s *** RETURNING TRANSLATED %s REQUEST #%d ***", rir.logPrefix, req.Method, requestNum)
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
				log.Printf("%s COMMAND TRACKING: Request #%d (ID:%v) -> %s command", rir.logPrefix, requestNum, req.ID, commandName)
			} else {
				methodToStore = req.Method
			}

			rir.requestMethodMap[normalizedID] = methodToStore
			log.Printf("%s TRACKING: Request #%d (ID:%v) -> method:%s (total tracked: %d)",
				rir.logPrefix, requestNum, req.ID, methodToStore, len(rir.requestMethodMap))
			rir.mapMutex.Unlock()

			// Special handling for State method requests
			if req.Method == "RPCServer.State" {
				log.Printf("%s Tracking State request #%d with ID: %v", rir.logPrefix, requestNum, req.ID)
			}

			if req.Method == "RPCServer.Stacktrace" {
				log.Printf("%s Tracking JSON-RPC Stacktrace request #%d with ID: %v", rir.logPrefix, requestNum, req.ID)
			}
		} else {
			log.Printf("%s UNPARSEABLE REQUEST #%d (JSON-RPC): %v", rir.logPrefix, requestNum, err)
			log.Printf("%s Raw data: %s", rir.logPrefix, jsonStr[:utils.Min(200, len(jsonStr))])
		}
	}

	return nil // No modifications needed
}

// translateEvalFrameNumber translates frame numbers in RPCServer.Eval requests from filtered to original
func (rir *RequestInterceptingReader) translateEvalFrameNumber(jsonObj []byte, remaining []byte, requestNum int) []byte {
	log.Printf("%s ENTERING translateEvalFrameNumber for request #%d", rir.logPrefix, requestNum)

	// Parse the JSON-RPC request
	var req extractors.JSONRPCRequest
	if err := json.Unmarshal(jsonObj, &req); err != nil {
		log.Printf("%s Failed to parse Eval request for frame translation: %v", rir.logPrefix, err)
		return nil
	}

	// Extract the EvalIn parameters
	if req.Params == nil {
		log.Printf("%s Eval request has no params", rir.logPrefix)
		return nil
	}

	// Convert params to EvalIn struct
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		log.Printf("%s Failed to marshal Eval params: %v", rir.logPrefix, err)
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
		log.Printf("%s Failed to parse EvalIn params array: %v", rir.logPrefix, err)
		return nil
	}

	if len(paramsArray) == 0 {
		log.Printf("%s EvalIn params array is empty", rir.logPrefix)
		return nil
	}

	evalParams := paramsArray[0]

	originalFrame := evalParams.Scope.Frame
	log.Printf("%s EVAL REQUEST ANALYSIS: expr='%s', goroutine=%d, frame=%d",
		rir.logPrefix, evalParams.Expr, evalParams.Scope.GoroutineID, originalFrame)

	// Get frame mapping from response reader
	if rir.responseReader == nil {
		log.Printf("%s No response reader available for frame mapping", rir.logPrefix)
		return nil
	}

	rir.responseReader.frameMappingLock.RLock()
	frameMapping := rir.responseReader.frameMapping
	if len(frameMapping) == 0 {
		rir.responseReader.frameMappingLock.RUnlock()
		log.Printf("%s No frame mapping available, request may fail", rir.logPrefix)
		return nil
	}

	// Translate frame number: client's filtered frame -> original delve frame
	translatedFrame, exists := frameMapping[originalFrame]
	rir.responseReader.frameMappingLock.RUnlock()

	if !exists {
		log.Printf("%s Frame %d not found in mapping (available: %v)", rir.logPrefix, originalFrame, frameMapping)
		return nil
	}

	log.Printf("%s FRAME TRANSLATION: filtered frame %d -> original frame %d", rir.logPrefix, originalFrame, translatedFrame)

	// Modify the request with the translated frame number
	evalParams.Scope.Frame = translatedFrame

	// Re-encode the modified parameters as array (JSON-RPC format)
	modifiedParamsArray := []EvalIn{evalParams}
	modifiedParamsBytes, err := json.Marshal(modifiedParamsArray)
	if err != nil {
		log.Printf("%s Failed to marshal modified EvalIn params array: %v", rir.logPrefix, err)
		return nil
	}

	// Convert back to interface{} for the request
	var modifiedParams interface{}
	if err := json.Unmarshal(modifiedParamsBytes, &modifiedParams); err != nil {
		log.Printf("%s Failed to unmarshal modified params array: %v", rir.logPrefix, err)
		return nil
	}

	// Update the request with modified params
	req.Params = modifiedParams

	// Re-encode the complete request
	modifiedRequestBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("%s Failed to marshal modified Eval request: %v", rir.logPrefix, err)
		return nil
	}

	log.Printf("%s Successfully translated Eval request: frame %d -> %d", rir.logPrefix, originalFrame, translatedFrame)

	// Combine modified request with remaining buffer data
	modifiedBuffer := make([]byte, len(modifiedRequestBytes)+len(remaining))
	copy(modifiedBuffer, modifiedRequestBytes)
	copy(modifiedBuffer[len(modifiedRequestBytes):], remaining)

	return modifiedBuffer
}

// translateFrameBasedRequest translates frame numbers in frame-based requests like ListLocalVars and ListFunctionArgs
func (rir *RequestInterceptingReader) translateFrameBasedRequest(jsonObj []byte, remaining []byte, requestNum int, method string) []byte {
	log.Printf("%s ENTERING translateFrameBasedRequest for %s request #%d", rir.logPrefix, method, requestNum)

	// Parse the JSON-RPC request
	var req extractors.JSONRPCRequest
	if err := json.Unmarshal(jsonObj, &req); err != nil {
		log.Printf("%s Failed to parse %s request for frame translation: %v", rir.logPrefix, method, err)
		return nil
	}

	// Extract the parameters
	if req.Params == nil {
		log.Printf("%s %s request has no params", rir.logPrefix, method)
		return nil
	}

	// Convert params to check for frame-based parameters
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		log.Printf("%s Failed to marshal %s params: %v", rir.logPrefix, method, err)
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
		log.Printf("%s Failed to parse %s params array: %v", rir.logPrefix, method, err)
		return nil
	}

	if len(paramsArray) == 0 {
		log.Printf("%s %s params array is empty", rir.logPrefix, method)
		return nil
	}

	params := paramsArray[0]

	originalFrame := params.Scope.Frame
	log.Printf("%s %s REQUEST ANALYSIS: goroutine=%d, frame=%d",
		rir.logPrefix, method, params.Scope.GoroutineID, originalFrame)

	// Get frame mapping from response reader
	if rir.responseReader == nil {
		log.Printf("%s No response reader available for frame mapping", rir.logPrefix)
		return nil
	}

	rir.responseReader.frameMappingLock.RLock()
	frameMapping := rir.responseReader.frameMapping
	if len(frameMapping) == 0 {
		rir.responseReader.frameMappingLock.RUnlock()
		log.Printf("%s No frame mapping available, %s request may fail", rir.logPrefix, method)
		return nil
	}

	// Translate frame number: client's filtered frame -> original delve frame
	translatedFrame, exists := frameMapping[originalFrame]
	rir.responseReader.frameMappingLock.RUnlock()

	if !exists {
		log.Printf("%s Frame %d not found in mapping for %s (available: %v)", rir.logPrefix, originalFrame, method, frameMapping)
		return nil
	}

	log.Printf("%s %s FRAME TRANSLATION: filtered frame %d -> original frame %d", rir.logPrefix, method, originalFrame, translatedFrame)

	// Modify the request with the translated frame number
	params.Scope.Frame = translatedFrame

	// Re-encode the modified parameters as array (JSON-RPC format)
	modifiedParamsArray := []FrameBasedParams{params}
	modifiedParamsBytes, err := json.Marshal(modifiedParamsArray)
	if err != nil {
		log.Printf("%s Failed to marshal modified %s params array: %v", rir.logPrefix, method, err)
		return nil
	}

	// Convert back to interface{} for the request
	var modifiedParams interface{}
	if err := json.Unmarshal(modifiedParamsBytes, &modifiedParams); err != nil {
		log.Printf("%s Failed to unmarshal modified %s params array: %v", rir.logPrefix, method, err)
		return nil
	}

	// Update the request with modified params
	req.Params = modifiedParams

	// Re-encode the complete request
	modifiedRequestBytes, err := json.Marshal(req)
	if err != nil {
		log.Printf("%s Failed to marshal modified %s request: %v", rir.logPrefix, method, err)
		return nil
	}

	log.Printf("%s Successfully translated %s request: frame %d -> %d", rir.logPrefix, method, originalFrame, translatedFrame)

	// Combine modified request with remaining buffer data
	modifiedBuffer := make([]byte, len(modifiedRequestBytes)+len(remaining))
	copy(modifiedBuffer, modifiedRequestBytes)
	copy(modifiedBuffer[len(modifiedRequestBytes):], remaining)

	return modifiedBuffer
}

// isStepOverCommand checks if a JSON-RPC request is a step over command
func (rir *RequestInterceptingReader) isStepOverCommand(req extractors.JSONRPCRequest) bool {
	if req.Params == nil {
		return false
	}

	// Convert params to check the command logPrefix
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return false
	}

	// JSON-RPC Command params come as an array: [CommandIn]
	type CommandIn struct {
		Name string `json:"logPrefix"`
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

// extractCommandNameFromRequest extracts the actual command logPrefix (next, continue, step, etc.) from a Command request
func (rir *RequestInterceptingReader) extractCommandNameFromRequest(req extractors.JSONRPCRequest) string {
	if req.Params == nil {
		return "unknown"
	}

	// Convert params to check the command logPrefix
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return "unknown"
	}

	// JSON-RPC Command params come as an array: [CommandIn]
	type CommandIn struct {
		Name string `json:"logPrefix"`
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
