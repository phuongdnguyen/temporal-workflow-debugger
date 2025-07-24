package delve_jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"

	"custom-debugger/pkg/utils"
)

type JSONRPCResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

// StateOut represents the State method response structure
type StateOut struct {
	State *api.DebuggerState `json:"State"`
}

// ResponseInterceptingReader wraps a Reader and intercepts JSON-RPC responses from delve -> client
type ResponseInterceptingReader struct {
	Reader           io.Reader
	Name             string
	RequestMethodMap map[string]string
	MapMutex         *sync.Mutex
	ClientAddr       string
	Buffer           []byte
	ModifiedData     []byte // Buffer for modified responses to send to client
	ModifiedOffset   int    // Current position in ModifiedData

	// Enhanced debugging counters
	// Enhanced debugging counters (no locks needed for debug counters)
	StackTraceCount     int
	StackFrameDataCount int
	AllResponseCount    int
	MainThreadMutex     sync.Mutex

	// Frame mapping for JSON-RPC stacktrace filtering
	FrameMapping     map[int]int // Maps filtered frame index -> original frame index
	FrameMappingLock sync.RWMutex

	// Auto-stepping infrastructure
	DelveClient *rpc2.RPCClient // Delve RPC client for auto-stepping

	// Current state tracking for sentinel breakpoint detection
	CurrentFile     string       // Current file location
	CurrentFunction string       // Current function Name
	CurrentLine     int          // Current line number
	StateMutex      sync.RWMutex // Protects current state fields

	// Reference to request Reader for step over tracking
	RequestReader *RequestInterceptingReader
}

func (rir *ResponseInterceptingReader) Read(p []byte) (n int, err error) {
	// First, check if we have modified data to send to client
	if rir.ModifiedOffset < len(rir.ModifiedData) {
		// Send modified data instead of reading from delve
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

	// Normal case: read from delve server
	n, err = rir.Reader.Read(p)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])

		// Append to Buffer for JSON-RPC parsing
		rir.Buffer = append(rir.Buffer, dataCopy...)

		// Try to extract complete JSON-RPC messages and potentially modify them
		modifiedData := rir.parseResponses()

		// Check if response was suppressed (nil means suppress)
		if modifiedData == nil && len(rir.Buffer) == 0 {
			// Response was suppressed - don't send anything to client
			log.Printf("%s: 0 bytes (response suppressed)", rir.Name)
			return 0, nil
		}

		// If we got modified data, we need to replace what we're sending to client
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

func (rir *ResponseInterceptingReader) parseResponses() []byte {
	// Safety check: prevent Buffer from growing too large
	const maxBufferSize = 10 * 1024 * 1024 // 10MB limit
	if len(rir.Buffer) > maxBufferSize {
		log.Printf("%s Buffer too large (%d bytes), resetting to prevent memory issues", rir.Name, len(rir.Buffer))
		rir.Buffer = nil
		return nil
	}

	// Prevent infinite loops by limiting iterations
	maxIterations := 100
	iterations := 0

	for len(rir.Buffer) > 0 && iterations < maxIterations {
		iterations++

		// Try to find a complete JSON object in the Buffer
		jsonObj, remaining, found := ExtractJSONObject(rir.Buffer)
		if !found {
			// No complete JSON object found, wait for more data
			break
		}

		// Safety check: ensure we're making progress
		if len(remaining) >= len(rir.Buffer) {
			log.Printf("%s No progress in Buffer parsing, breaking to prevent infinite loop", rir.Name)
			break
		}

		// Update Buffer to remaining data
		rir.Buffer = remaining

		// ENHANCED DEBUG: Track ALL responses with unique IDs
		rir.AllResponseCount++
		responseNum := rir.AllResponseCount

		jsonStr := string(jsonObj)
		log.Printf("%s RESPONSE #%d (%d bytes): %s", rir.Name, responseNum, len(jsonObj), jsonStr[:utils.Min(150, len(jsonStr))])

		// DEBUG: Log ALL complete JSON objects to catch missed stackTrace responses
		if strings.Contains(strings.ToLower(jsonStr), "stackframe") {
			rir.StackFrameDataCount++
			globalCount := rir.StackFrameDataCount

			log.Printf("%s DETECTED POTENTIAL STACKTRACE DATA #%d (Response #%d): %s", rir.Name, globalCount, responseNum, jsonStr[:utils.Min(400, len(jsonStr))])
		}

		// DEBUG: Log ANY response that might contain location information
		if strings.Contains(strings.ToLower(jsonStr), "line") ||
			strings.Contains(strings.ToLower(jsonStr), "file") ||
			strings.Contains(strings.ToLower(jsonStr), "location") ||
			strings.Contains(strings.ToLower(jsonStr), "source") {
			log.Printf("%s LOCATION INFO DETECTED in Response #%d: %s", rir.Name, responseNum, jsonStr[:utils.Min(300, len(jsonStr))])
		}

		// Parse JSON-RPC response only (DAP support removed)
		var resp JSONRPCResponse
		if err := json.Unmarshal(jsonObj, &resp); err == nil {
			normalizedID := utils.NormalizeID(resp.ID)

			// Process internal auto-stepping responses before filtering them out
			log.Printf("%s RESPONSE ID CHECK: %s (isAutoStepInternal: %v)", rir.Name, normalizedID, rir.isAutoStepInternalResponse(normalizedID))
			if rir.isAutoStepInternalResponse(normalizedID) {
				log.Printf("%s PROCESSING AUTO-STEP INTERNAL RESPONSE: ID %s before filtering", rir.Name, normalizedID)
				log.Printf("%s RESPONSE TIMING: Response %s received at %v", rir.Name, normalizedID, time.Now())

				// Update our stored location from the step response before filtering
				rir.storeCurrentLocationFromCommandResponse(jsonObj)

				log.Printf("%s FILTERING AUTO-STEP INTERNAL RESPONSE: ID %s (not forwarding to GoLand)", rir.Name, normalizedID)
				return nil // Don't forward to client
			}

			// Check which method this response corresponds to
			rir.MapMutex.Lock()
			method, hasMethod := rir.RequestMethodMap[normalizedID]
			if hasMethod {
				delete(rir.RequestMethodMap, normalizedID) // Clean up
			}
			rir.MapMutex.Unlock()

			log.Printf("%s JSON-RPC ANALYSIS: Response #%d - ID:%v, method:%s, hasMethod:%v",
				rir.Name, responseNum, resp.ID, method, hasMethod)

			// COMPREHENSIVE STACKTRACE DETECTION - check both tracked method and actual content
			isStackTraceByMethod := hasMethod && method == "RPCServer.Stacktrace"
			isStackTraceByContent := strings.Contains(strings.ToLower(jsonStr), "locations") &&
				strings.Contains(strings.ToLower(jsonStr), "file") &&
				strings.Contains(strings.ToLower(jsonStr), "line")

			if isStackTraceByMethod || isStackTraceByContent {
				log.Printf("%s *** JSON-RPC STACKTRACE DETECTED! Response #%d ***", rir.Name, responseNum)
				log.Printf("%s Detection method: byMethod=%v, byContent=%v", rir.Name, isStackTraceByMethod, isStackTraceByContent)

				if isStackTraceByMethod {
					log.Printf("%s *** INTERCEPTING STACKTRACE RESPONSE (tracked) ***", rir.Name)
				} else {
					log.Printf("%s *** INTERCEPTING STACKTRACE RESPONSE (content-detected) ***", rir.Name)
				}

				rir.logStacktraceResponse(string(jsonObj))

				// Filter the stacktrace and return modified response
				filteredResponse := rir.filterStacktraceResponse(jsonObj, remaining)
				if filteredResponse != nil {
					log.Printf("%s *** RETURNING FILTERED JSON-RPC STACKTRACE Response #%d TO CLIENT ***", rir.Name, responseNum, len(filteredResponse))
					return filteredResponse
				}
				log.Printf("%s *** END JSON-RPC STACKTRACE Response #%d ***", rir.Name, responseNum)
			} else {
				// Non-stacktrace JSON-RPC response
				if hasMethod {
					log.Printf("%s RPC Response #%d for %s (ID: %v)", rir.Name, responseNum, method, resp.ID)
				} else {
					log.Printf("%s RPC Response #%d for unknown method (ID: %v, type: %T)", rir.Name, responseNum, resp.ID, resp.ID)
				}
			}

			// Special handling for Command responses - implement auto-stepping when user steps into adapter code
			if hasMethod && strings.HasPrefix(method, "RPCServer.Command.") {
				log.Printf("%s *** COMMAND RESPONSE DETECTED! Response #%d (method: %s) ***", rir.Name, responseNum, method)

				// Store current location from Command response for state tracking
				rir.storeCurrentLocationFromCommandResponse(jsonObj)

				// Check if user stepped into adapter code (sentinel breakpoint detection)
				if rir.isCommandResponseAtSentinelBreakpoint(jsonObj) {
					log.Printf("%s USER STEPPED INTO ADAPTER CODE!", rir.Name)
					log.Printf("%s AUTO-STEPPING: Automatically stepping through adapter code back to workflow", rir.Name)

					// Perform auto-stepping through adapter code and return the final workflow location
					finalResponse := rir.performDirectAutoStepping(jsonObj, remaining, responseNum, normalizedID)
					if finalResponse != nil {
						log.Printf("%s *** AUTO-STEP COMPLETE: Returning user to workflow code ***", rir.Name)
						return finalResponse
					} else {
						log.Printf("%s AUTO-STEP: Suppressed adapter code response, stepping to workflow", rir.Name)
						return nil // CRITICAL: Return nil to suppress the adapter code response
					}
				}

				log.Printf("%s Step-over completed in user code, forwarding normal response to GoLand", rir.Name)
			}

			// Special handling for State responses
			if hasMethod && method == "RPCServer.State" {
				log.Printf("%s *** INTERCEPTING STATE RESPONSE #%d ***", rir.Name, responseNum)
				rir.logStateResponse(string(jsonObj))
			}

			if resp.Error != nil {
				if hasMethod {
					log.Printf("%s RPC Error Response #%d for %s (ID: %v): %v", rir.Name, responseNum, method, resp.ID, resp.Error)
				} else {
					log.Printf("%s RPC Error Response #%d for unknown method (ID: %v): %v", rir.Name, responseNum, resp.ID, resp.Error)
				}
			}
		} else {
			// Fall back to JSON-RPC response
			var resp JSONRPCResponse
			if err := json.Unmarshal(jsonObj, &resp); err == nil {
				normalizedID := utils.NormalizeID(resp.ID)

				// Process internal auto-stepping responses before filtering them out
				log.Printf("%s FALLBACK RESPONSE ID CHECK: %s (isAutoStepInternal: %v)", rir.Name, normalizedID, rir.isAutoStepInternalResponse(normalizedID))
				if rir.isAutoStepInternalResponse(normalizedID) {
					log.Printf("%s PROCESSING AUTO-STEP INTERNAL RESPONSE: ID %s before filtering", rir.Name, normalizedID)
					log.Printf("%s RESPONSE TIMING: Response %s received at %v", rir.Name, normalizedID, time.Now())

					// Update our stored location from the step response before filtering
					rir.storeCurrentLocationFromCommandResponse(jsonObj)

					log.Printf("%s FILTERING AUTO-STEP INTERNAL RESPONSE: ID %s (not forwarding to GoLand)", rir.Name, normalizedID)
					return nil // Don't forward to client
				}

				// Check which method this response corresponds to
				rir.MapMutex.Lock()
				method, hasMethod := rir.RequestMethodMap[normalizedID]
				if hasMethod {
					delete(rir.RequestMethodMap, normalizedID) // Clean up
				}
				rir.MapMutex.Unlock()

				log.Printf("%s JSON-RPC ANALYSIS: Response #%d - ID:%v, method:%s, hasMethod:%v",
					rir.Name, responseNum, resp.ID, method, hasMethod)

				// COMPREHENSIVE STACKTRACE DETECTION - check both tracked method and actual content
				isStackTraceByMethod := hasMethod && method == "RPCServer.Stacktrace"
				isStackTraceByContent := strings.Contains(strings.ToLower(jsonStr), "locations") &&
					strings.Contains(strings.ToLower(jsonStr), "file") &&
					strings.Contains(strings.ToLower(jsonStr), "line")

				if isStackTraceByMethod || isStackTraceByContent {
					log.Printf("%s *** JSON-RPC STACKTRACE DETECTED! Response #%d ***", rir.Name, responseNum)
					log.Printf("%s Detection method: byMethod=%v, byContent=%v", rir.Name, isStackTraceByMethod, isStackTraceByContent)

					if isStackTraceByMethod {
						log.Printf("%s *** INTERCEPTING STACKTRACE RESPONSE (tracked) ***", rir.Name)
					} else {
						log.Printf("%s *** INTERCEPTING STACKTRACE RESPONSE (content-detected) ***", rir.Name)
					}

					rir.logStacktraceResponse(string(jsonObj))

					// Filter the stacktrace and return modified response
					filteredResponse := rir.filterStacktraceResponse(jsonObj, remaining)
					if filteredResponse != nil {
						log.Printf("%s *** RETURNING FILTERED JSON-RPC STACKTRACE Response #%d TO CLIENT ***", rir.Name, responseNum, len(filteredResponse))
						return filteredResponse
					}
					log.Printf("%s *** END JSON-RPC STACKTRACE Response #%d ***", rir.Name, responseNum)
				} else {
					// Non-stacktrace JSON-RPC response
					if hasMethod {
						log.Printf("%s RPC Response #%d for %s (ID: %v)", rir.Name, responseNum, method, resp.ID)
					} else {
						log.Printf("%s RPC Response #%d for unknown method (ID: %v, type: %T)", rir.Name, responseNum, resp.ID, resp.ID)
					}
				}

				// Special handling for Command responses - implement auto-stepping when user steps into adapter code
				if hasMethod && strings.HasPrefix(method, "RPCServer.Command.") {
					log.Printf("%s *** COMMAND RESPONSE DETECTED! Response #%d (method: %s) ***", rir.Name, responseNum, method)

					// Store current location from Command response for state tracking
					rir.storeCurrentLocationFromCommandResponse(jsonObj)

					// Check if user stepped into adapter code (sentinel breakpoint detection)
					if rir.isCommandResponseAtSentinelBreakpoint(jsonObj) {
						log.Printf("%s USER STEPPED INTO ADAPTER CODE!", rir.Name)
						log.Printf("%s AUTO-STEPPING: Automatically stepping through adapter code back to workflow", rir.Name)

						// Perform auto-stepping through adapter code and return the final workflow location
						finalResponse := rir.performDirectAutoStepping(jsonObj, remaining, responseNum, normalizedID)
						if finalResponse != nil {
							log.Printf("%s *** AUTO-STEP COMPLETE: Returning user to workflow code ***", rir.Name)
							return finalResponse
						} else {
							log.Printf("%s AUTO-STEP: Suppressed adapter code response, stepping to workflow", rir.Name)
							return nil // CRITICAL: Return nil to suppress the adapter code response
						}
					}

					log.Printf("%s Step-over completed in user code, forwarding normal response to GoLand", rir.Name)
				}

				// Special handling for State responses
				if hasMethod && method == "RPCServer.State" {
					log.Printf("%s *** INTERCEPTING STATE RESPONSE #%d ***", rir.Name, responseNum)
					rir.logStateResponse(string(jsonObj))
				}

				if resp.Error != nil {
					if hasMethod {
						log.Printf("%s RPC Error Response #%d for %s (ID: %v): %v", rir.Name, responseNum, method, resp.ID, resp.Error)
					} else {
						log.Printf("%s RPC Error Response #%d for unknown method (ID: %v): %v", rir.Name, responseNum, resp.ID, resp.Error)
					}
				}
			} else {
				log.Printf("%s UNPARSEABLE Response #%d (tried both DAP and JSON-RPC): %v", rir.Name, responseNum, err)
				log.Printf("%s Raw data: %s", rir.Name, jsonStr[:utils.Min(200, len(jsonStr))])
			}
		}
	}

	// Check if we hit the iteration limit
	if iterations >= maxIterations {
		log.Printf("%s Reached maximum iterations (%d) in parseResponses, Buffer length: %d", rir.Name, maxIterations, len(rir.Buffer))
	}

	return nil // No modifications needed
}

func (rir *ResponseInterceptingReader) filterStacktraceResponse(jsonObj []byte, remaining []byte) []byte {
	var response JSONRPCResponse
	if err := json.Unmarshal(jsonObj, &response); err != nil {
		log.Printf("[%s] Failed to parse JSON-RPC response for filtering: %v", rir.ClientAddr, err)
		return nil
	}

	// Extract the stacktrace from the response
	if response.Result == nil {
		log.Printf("[%s] Stacktrace response has no result", rir.ClientAddr)
		return nil
	}

	// Convert result to StacktraceOut
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		log.Printf("[%s] Failed to marshal result for filtering: %v", rir.ClientAddr, err)
		return nil
	}

	var stacktraceOut rpc2.StacktraceOut
	if err := json.Unmarshal(resultBytes, &stacktraceOut); err != nil {
		log.Printf("[%s] Failed to parse StacktraceOut for filtering: %v", rir.ClientAddr, err)
		return nil
	}

	// Filter stack frames: find the deepest frame containing user code (working directory)
	// and keep all frames from 0 up to and including that frame (this includes user code in other files)
	filteredLocations := stacktraceOut.Locations
	userCodeFrameIndex := -1
	workingDir := getCurrentWorkingDir()

	// Find the LAST/DEEPEST occurrence of user code (highest index) - this is the actual user entry point
	for i := len(stacktraceOut.Locations) - 1; i >= 0; i-- {
		frame := stacktraceOut.Locations[i]
		if rir.isUserCodeFile(frame.File, workingDir) {
			userCodeFrameIndex = i
			break // Found the deepest user code frame
		}
	}

	if userCodeFrameIndex == -1 {
		log.Printf("[%s] No user code frame found in working directory, not filtering", rir.ClientAddr)
		return nil // Don't filter if we can't find the target frame
	}

	// Keep frames from 0 up to and including the user code frame (filters out adapter frames above it)
	filteredLocations = stacktraceOut.Locations[0 : userCodeFrameIndex+1]
	framesRemoved := len(stacktraceOut.Locations) - len(filteredLocations)

	log.Printf("[%s] Found user code entry point at frame %d", rir.ClientAddr, userCodeFrameIndex)
	log.Printf("[%s] Keeping frames 0-%d (includes user code in working directory), filtering out %d adapter frames (original: %d, filtered: %d)",
		rir.ClientAddr, userCodeFrameIndex, framesRemoved, len(stacktraceOut.Locations), len(filteredLocations))

	// UPDATE FRAME MAPPING: Create mapping from filtered frame index to original frame index
	rir.FrameMappingLock.Lock()
	rir.FrameMapping = make(map[int]int)

	// Create 1:1 mapping for frames 0 to userCodeFrameIndex (no offset needed since we keep frames from the beginning)
	for filteredIndex := 0; filteredIndex < len(filteredLocations); filteredIndex++ {
		originalIndex := filteredIndex // Direct mapping since we keep frames 0 to userCodeFrameIndex
		rir.FrameMapping[filteredIndex] = originalIndex
		log.Printf("[%s] Frame mapping: filtered[%d] -> original[%d]", rir.ClientAddr, filteredIndex, originalIndex)
	}
	rir.FrameMappingLock.Unlock()

	log.Printf("[%s] Created frame mapping with %d entries for frame translation", rir.ClientAddr, len(rir.FrameMapping))

	log.Printf("[%s] FRAME MAPPING SOLUTION: When stacktrace filtering removes adapter frames,", rir.ClientAddr)
	log.Printf("[%s] the proxy now translates frame numbers in Eval/ListLocalVars/ListFunctionArgs requests", rir.ClientAddr)
	log.Printf("[%s] This ensures variables are evaluated in the correct original frame context", rir.ClientAddr)

	// Update the stacktrace with filtered locations
	stacktraceOut.Locations = filteredLocations

	// Re-encode the filtered stacktrace
	filteredResultBytes, err := json.Marshal(stacktraceOut)
	if err != nil {
		log.Printf("[%s] Failed to marshal filtered stacktrace: %v", rir.ClientAddr, err)
		return nil
	}

	// Update the response with filtered result
	var filteredResult interface{}
	if err := json.Unmarshal(filteredResultBytes, &filteredResult); err != nil {
		log.Printf("[%s] Failed to unmarshal filtered result: %v", rir.ClientAddr, err)
		return nil
	}
	response.Result = filteredResult

	// Re-encode the complete response
	filteredResponseBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("[%s] Failed to marshal filtered response: %v", rir.ClientAddr, err)
		return nil
	}

	// Combine filtered response with remaining Buffer data
	modifiedBuffer := make([]byte, len(filteredResponseBytes)+len(remaining))
	copy(modifiedBuffer, filteredResponseBytes)
	copy(modifiedBuffer[len(filteredResponseBytes):], remaining)

	return modifiedBuffer
}

func (rir *ResponseInterceptingReader) logStacktraceResponse(jsonLine string) {
	var response JSONRPCResponse
	if err := json.Unmarshal([]byte(jsonLine), &response); err != nil {
		log.Printf("[%s] Failed to parse JSON-RPC response: %v", rir.ClientAddr, err)
		return
	}

	var stacktraceOut rpc2.StacktraceOut
	if response.Result != nil {
		resultBytes, err := json.Marshal(response.Result)
		if err != nil {
			log.Printf("[%s] Failed to marshal result: %v", rir.ClientAddr, err)
			return
		}
		if err := json.Unmarshal(resultBytes, &stacktraceOut); err != nil {
			log.Printf("[%s] Failed to parse StacktraceOut: %v", rir.ClientAddr, err)
			return
		}
	}

	// Get goroutine ID from the original request if available
	var goroutineID interface{} = "unknown"
	rir.MapMutex.Lock()
	if reqMethod, ok := rir.RequestMethodMap[utils.NormalizeID(response.ID)]; ok {
		if reqMethod == "RPCServer.Stacktrace" {
			// For now, we'll just show "unknown" since we don't parse the request params
			// Could be enhanced to parse the request parameters to get the actual goroutine ID
		}
	}
	rir.MapMutex.Unlock()

	rir.logStacktraceResponseDetails(stacktraceOut, goroutineID)
}

func (rir *ResponseInterceptingReader) LogDebuggingSummary() {
	totalResponses := rir.AllResponseCount
	totalStackFrames := rir.StackFrameDataCount
	totalStackTraces := rir.StackTraceCount

	log.Printf("[%s] DEBUGGING SUMMARY (Client Disconnected):", rir.ClientAddr)
	log.Printf("[%s]   Total Responses: %d", rir.ClientAddr, totalResponses)
	log.Printf("[%s]   Total Stack Frames in Responses: %d", rir.ClientAddr, totalStackFrames)
	log.Printf("[%s]   Total Stack Traces Detected: %d", rir.ClientAddr, totalStackTraces)

	// Check for potential missed stackTrace responses
	if totalStackFrames > totalStackTraces {
		log.Printf("[%s] Potential missed stackTrace responses detected! (Total Frames: %d, Detected Traces: %d)", rir.ClientAddr, totalStackFrames, totalStackTraces)
		log.Printf("[%s]    This might indicate that stackTrace requests were not intercepted or filtered correctly.")
	}
}

func (rir *ResponseInterceptingReader) logStateResponse(jsonLine string) {
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(jsonLine), &resp); err != nil {
		log.Printf("[%s] Failed to parse JSON-RPC response: %v", rir.ClientAddr, err)
		return
	}

	// Extract the State from the response
	if resp.Result == nil {
		log.Printf("[%s] State response has no result", rir.ClientAddr)
		return
	}

	// Convert result to DebuggerState
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		log.Printf("[%s] Failed to marshal result: %v", rir.ClientAddr, err)
		return
	}

	var stateOut StateOut
	if err := json.Unmarshal(resultBytes, &stateOut); err != nil {
		log.Printf("[%s] Failed to parse StateOut: %v", rir.ClientAddr, err)
		return
	}

	if stateOut.State == nil {
		log.Printf("[%s] StateOut has no State", rir.ClientAddr)
		return
	}

	// Store current location for sentinel breakpoint detection
	if stateOut.State.CurrentThread != nil {
		rir.StateMutex.Lock()
		rir.CurrentFile = stateOut.State.CurrentThread.File
		rir.CurrentLine = stateOut.State.CurrentThread.Line

		// Try multiple approaches to get function Name
		rir.CurrentFunction = ""

		// Method 1: Try from breakpoint info stacktrace
		if stateOut.State.CurrentThread.BreakpointInfo != nil &&
			len(stateOut.State.CurrentThread.BreakpointInfo.Stacktrace) > 0 {
			topFrame := stateOut.State.CurrentThread.BreakpointInfo.Stacktrace[0]
			if topFrame.Function != nil {
				rir.CurrentFunction = topFrame.Function.Name()
				log.Printf("%s Function from BreakpointInfo.Stacktrace: %s", rir.Name, rir.CurrentFunction)
			}
		}

		// Method 2: Try from selected goroutine
		if rir.CurrentFunction == "" && stateOut.State.SelectedGoroutine != nil {
			if stateOut.State.SelectedGoroutine.CurrentLoc.Function != nil {
				rir.CurrentFunction = stateOut.State.SelectedGoroutine.CurrentLoc.Function.Name()
				log.Printf("%s Function from SelectedGoroutine.CurrentLoc: %s", rir.Name, rir.CurrentFunction)
			}
		}

		// Method 3: Fallback - derive function from file location
		if rir.CurrentFunction == "" {
			if strings.Contains(rir.CurrentFile, "replayer-adapter/replayer.go") && rir.CurrentLine >= 100 && rir.CurrentLine <= 110 {
				rir.CurrentFunction = "adapter_go.notifyRunner"
				log.Printf("%s 🔍 Function inferred from location: %s", rir.Name, rir.CurrentFunction)
			}
		}

		rir.StateMutex.Unlock()

		log.Printf("%s CURRENT LOCATION STORED: %s:%d (%s)", rir.Name, rir.CurrentFile, rir.CurrentLine, rir.CurrentFunction)
	}

	// Log detailed state information
	log.Printf("[%s] === STATE INTERCEPTION ===", rir.ClientAddr)
	log.Printf("[%s] Running: %v", rir.ClientAddr, stateOut.State.Running)
	log.Printf("[%s] Exited: %v", rir.ClientAddr, stateOut.State.Exited)

	if stateOut.State.CurrentThread != nil {
		thread := stateOut.State.CurrentThread
		log.Printf("[%s] Current Thread ID: %d", rir.ClientAddr, thread.ID)
		log.Printf("[%s] Current Location: %s:%d", rir.ClientAddr, thread.File, thread.Line)

		if thread.BreakpointInfo != nil {
			// Check if this goroutine has adapter_go.notifyRunner as top frame
			if len(thread.BreakpointInfo.Stacktrace) > 0 {
				topFrame := thread.BreakpointInfo.Stacktrace[0]
				fmt.Printf("topFrame: %+v\n", topFrame)
			}

			// Log goroutine information
			if thread.BreakpointInfo.Goroutine != nil {
				goroutine := thread.BreakpointInfo.Goroutine
				log.Printf("[%s] === GOROUTINE INFO ===", rir.ClientAddr)
				log.Printf("[%s] Goroutine ID: %d", rir.ClientAddr, goroutine.ID)
				log.Printf("[%s] Thread ID: %d", rir.ClientAddr, goroutine.ThreadID)
				log.Printf("[%s] Status: %d", rir.ClientAddr, goroutine.Status)
				log.Printf("[%s] Current Location: %s:%d", rir.ClientAddr, goroutine.CurrentLoc.File, goroutine.CurrentLoc.Line)
				if goroutine.CurrentLoc.Function != nil {
					log.Printf("[%s] Current Function: %s", rir.ClientAddr, goroutine.CurrentLoc.Function.Name)
				}
				log.Printf("[%s] Start Location: %s:%d", rir.ClientAddr, goroutine.StartLoc.File, goroutine.StartLoc.Line)
				if goroutine.StartLoc.Function != nil {
					log.Printf("[%s] Start Function: %s", rir.ClientAddr, goroutine.StartLoc.Function.Name)
				}
				if len(goroutine.Labels) > 0 {
					log.Printf("[%s] Labels:", rir.ClientAddr)
					for key, value := range goroutine.Labels {
						log.Printf("[%s]   %s: %s", rir.ClientAddr, key, value)
					}
				}
				log.Printf("[%s] === END GOROUTINE INFO ===", rir.ClientAddr)
			}
		}
	}
	if stateOut.State.SelectedGoroutine != nil {
		fmt.Printf("Selected Goroutine: %+v\n", *stateOut.State.SelectedGoroutine)
	}
	for _, thread := range stateOut.State.Threads {
		fmt.Printf("Thread: %+v\n", *thread)
	}
	// Search all threads for adapter_go.notifyRunner

	log.Printf("[%s] === END STATE INTERCEPTION ===", rir.ClientAddr)
}

// logStacktraceResponseDetails logs detailed information about a Stacktrace RPC response
func (rir *ResponseInterceptingReader) logStacktraceResponseDetails(stacktraceOut rpc2.StacktraceOut, goroutineID interface{}) {
	log.Printf("[%s] === STACKTRACE INTERCEPTION ===", rir.ClientAddr)

	if len(stacktraceOut.Locations) == 0 {
		log.Printf("[%s] No stack frames found", rir.ClientAddr)
		return
	}

	// Filter stack frames: remove frames from the top until we find user code (working directory)
	filteredLocations := stacktraceOut.Locations
	userCodeFrameIndex := -1

	// Get working directory (you may need to pass this in or get it from context)
	workingDir := getCurrentWorkingDir() // This function needs to be implemented

	for i, frame := range stacktraceOut.Locations {
		if rir.isUserCodeFile(frame.File, workingDir) {
			userCodeFrameIndex = i
			filteredLocations = stacktraceOut.Locations[i:]
			break
		}
	}

	if userCodeFrameIndex == -1 {
		log.Printf("[%s] No user code frame found in working directory, showing all %d frames", rir.ClientAddr, len(stacktraceOut.Locations))
	} else {
		log.Printf("[%s] Filtered out %d frames before user code (original: %d, filtered: %d)", rir.ClientAddr, userCodeFrameIndex, len(stacktraceOut.Locations), len(filteredLocations))
	}

	log.Printf("[%s] Goroutine %v Stack Trace (%d frames):", rir.ClientAddr, goroutineID, len(filteredLocations))

	// Check if any frame contains adapter_go.notifyRunner (using filtered frames)
	hasNotifyRunner := false
	notifyRunnerFrameIndex := -1
	for i, frame := range filteredLocations {
		if frame.Function != nil && rir.isNotifyRunnerFunction(frame.Function.Name()) {
			hasNotifyRunner = true
			notifyRunnerFrameIndex = i
			break
		}
	}

	if hasNotifyRunner {
		log.Printf("[%s] FOUND ADAPTER_GO.NOTIFYRUNNER IN STACK TRACE! (Frame %d)", rir.ClientAddr, notifyRunnerFrameIndex)
	}

	for i, frame := range filteredLocations {
		log.Printf("[%s] Frame %d:", rir.ClientAddr, i)
		log.Printf("[%s]   %s:%d", rir.ClientAddr, frame.File, frame.Line)
		log.Printf("[%s]   PC: 0x%x", rir.ClientAddr, frame.PC)

		if frame.Function != nil {
			if i == notifyRunnerFrameIndex {
				log.Printf("[%s]   Function: %s (NOTIFY_RUNNER FRAME)", rir.ClientAddr, frame.Function.Name())
			} else {
				log.Printf("[%s]   Function: %s", rir.ClientAddr, frame.Function.Name())
			}
		}

		// Frame offsets
		log.Printf("[%s]   Frame Offset: 0x%x, Frame Pointer Offset: 0x%x", rir.ClientAddr, frame.FrameOffset, frame.FramePointerOffset)

		// Arguments
		if len(frame.Arguments) > 0 {
			log.Printf("[%s]   Arguments (%d):", rir.ClientAddr, len(frame.Arguments))
			for j, arg := range frame.Arguments {
				log.Printf("[%s]     [%d] %s = %s (%s)", rir.ClientAddr, j, arg.Name, arg.Value, arg.Type)
			}
		}

		// Local variables
		if len(frame.Locals) > 0 {
			log.Printf("[%s]   Locals (%d):", rir.ClientAddr, len(frame.Locals))
			for j, local := range frame.Locals {
				log.Printf("[%s]     [%d] %s = %s (%s)", rir.ClientAddr, j, local.Name, local.Value, local.Type)
			}
		}

		// Deferred functions
		if len(frame.Defers) > 0 {
			log.Printf("[%s]   Deferred Functions (%d):", rir.ClientAddr, len(frame.Defers))
			for j, defer_ := range frame.Defers {
				log.Printf("[%s]     [%d] %s at %s:%d", rir.ClientAddr, j,
					defer_.DeferredLoc.Function.Name(), defer_.DeferredLoc.File, defer_.DeferredLoc.Line)
			}
		}

		// Bottom frame indicator
		if frame.Bottom {
			log.Printf("[%s]   (Bottom frame)", rir.ClientAddr)
		}

		// Frame errors
		if frame.Err != "" {
			log.Printf("[%s]   Error: %s", rir.ClientAddr, frame.Err)
		}

		log.Printf("[%s]", rir.ClientAddr) // Empty line for readability
	}

	if hasNotifyRunner {
		log.Printf("[%s] === NOTIFY_RUNNER STACK TRACE COMPLETE ===", rir.ClientAddr)
	}

	log.Printf("[%s] === END STACKTRACE INTERCEPTION ===", rir.ClientAddr)
}

// isNotifyRunnerFunction checks if a function Name matches the adapter_go.notifyRunner pattern
func (rir *ResponseInterceptingReader) isNotifyRunnerFunction(functionName string) bool {
	// Check for various patterns of adapter_go.notifyRunner
	return functionName == "adapter_go.notifyRunner" ||
		functionName == "(*adapter_go).notifyRunner" ||
		functionName == "github.com/phuongdnguyen/temporal-goland-plugin/replayer-adapter/adapter_go.notifyRunner" ||
		(len(functionName) >= len("notifyRunner") &&
			functionName[len(functionName)-len("notifyRunner"):] == "notifyRunner") ||
		(len(functionName) >= len("adapter_go.notifyRunner") &&
			functionName[len(functionName)-len("adapter_go.notifyRunner"):] == "adapter_go.notifyRunner")
}

// isInAdapterCodeByPath checks if a file path is in adapter code
func (rir *ResponseInterceptingReader) isInAdapterCodeByPath(filePath string) bool {
	if filePath == "" {
		return false
	}

	// Check if this is user code (should NOT be considered adapter code)
	workingDir := getCurrentWorkingDir()
	if rir.isUserCodeFile(filePath, workingDir) {
		return false
	}

	// Check for adapter code patterns in the file path
	return strings.Contains(filePath, "replayer-adapter/") ||
		strings.Contains(filePath, "replayer.go") ||
		strings.Contains(filePath, "outbound_interceptor.go") ||
		strings.Contains(filePath, "inbound_interceptor.go") ||
		// ALL Temporal SDK code (both versioned and non-versioned paths)
		strings.Contains(filePath, "go.temporal.io/sdk/") ||
		strings.Contains(filePath, "go.temporal.io/sdk@") ||
		// Other Go runtime/reflection code that might be encountered
		strings.Contains(filePath, "/runtime/") ||
		strings.Contains(filePath, "/reflect/")
}

// isAutoStepInternalResponse checks if this is an internal auto-stepping response (90xxx or 99xxx range)
func (rir *ResponseInterceptingReader) isAutoStepInternalResponse(responseID string) bool {
	// Auto-stepping uses IDs in the range 90000-90999 (direct) and 99000-99999 (legacy)
	if len(responseID) >= 5 && (responseID[:2] == "90" || responseID[:2] == "99") {
		// Convert to int to validate it's in the expected range
		if id, err := json.Number(responseID).Int64(); err == nil {
			return (id >= 90000 && id <= 90999) || (id >= 99000 && id <= 99999)
		}
	}

	// Also filter autostep_* responses from legacy auto-stepping
	if strings.HasPrefix(responseID, "autostep_") {
		log.Printf("%s FILTERING LEGACY AUTO-STEP RESPONSE: ID %s", rir.Name, responseID)
		return true
	}

	return false
}

// storeCurrentLocationFromCommandResponse extracts and stores current location from Command response
func (rir *ResponseInterceptingReader) storeCurrentLocationFromCommandResponse(jsonObj []byte) {
	var response JSONRPCResponse
	if err := json.Unmarshal(jsonObj, &response); err != nil {
		log.Printf("%s ❌ Failed to parse Command response for location storage: %v", rir.Name, err)
		return
	}

	// Extract the Command response state
	if response.Result == nil {
		return
	}

	// Convert result to check current location
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return
	}

	var commandOut struct {
		State *api.DebuggerState `json:"State"`
	}
	if err := json.Unmarshal(resultBytes, &commandOut); err != nil {
		return
	}

	if commandOut.State == nil || commandOut.State.Running {
		return
	}

	// Store current location from command response
	if commandOut.State.CurrentThread != nil {
		rir.StateMutex.Lock()
		rir.CurrentFile = commandOut.State.CurrentThread.File
		rir.CurrentLine = commandOut.State.CurrentThread.Line

		// Try multiple approaches to get function Name
		rir.CurrentFunction = ""

		// Method 1: Try from breakpoint info stacktrace
		if commandOut.State.CurrentThread.BreakpointInfo != nil &&
			len(commandOut.State.CurrentThread.BreakpointInfo.Stacktrace) > 0 {
			topFrame := commandOut.State.CurrentThread.BreakpointInfo.Stacktrace[0]
			if topFrame.Function != nil {
				rir.CurrentFunction = topFrame.Function.Name()
				log.Printf("%s 🔍 Function from BreakpointInfo.Stacktrace: %s", rir.Name, rir.CurrentFunction)
			}
		}

		// Method 2: Try from selected goroutine
		if rir.CurrentFunction == "" && commandOut.State.SelectedGoroutine != nil {
			if commandOut.State.SelectedGoroutine.CurrentLoc.Function != nil {
				rir.CurrentFunction = commandOut.State.SelectedGoroutine.CurrentLoc.Function.Name()
				log.Printf("%s 🔍 Function from SelectedGoroutine.CurrentLoc: %s", rir.Name, rir.CurrentFunction)
			}
		}

		// Method 3: Fallback - derive function from file location
		if rir.CurrentFunction == "" {
			if strings.Contains(rir.CurrentFile, "replayer-adapter/replayer.go") && rir.CurrentLine >= 100 && rir.CurrentLine <= 110 {
				rir.CurrentFunction = "adapter_go.notifyRunner"
				log.Printf("%s Function inferred from location: %s", rir.Name, rir.CurrentFunction)
			}
		}

		rir.StateMutex.Unlock()

		log.Printf("%s COMMAND LOCATION STORED: %s:%d (%s)", rir.Name, rir.CurrentFile, rir.CurrentLine, rir.CurrentFunction)
	}
}

// isCommandResponseAtSentinelBreakpoint checks if a Command response shows we've stopped at a sentinel breakpoint
// This now detects ANY step-over that lands in adapter code, not just specific notifyRunner function
func (rir *ResponseInterceptingReader) isCommandResponseAtSentinelBreakpoint(jsonObj []byte) bool {
	var response JSONRPCResponse
	if err := json.Unmarshal(jsonObj, &response); err != nil {
		log.Printf("%s ❌ Failed to parse Command response for sentinel check: %v", rir.Name, err)
		return false
	}

	// Extract the Command response state
	if response.Result == nil {
		return false
	}

	// Convert result to check current location
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return false
	}

	var commandOut struct {
		State *api.DebuggerState `json:"State"`
	}
	if err := json.Unmarshal(resultBytes, &commandOut); err != nil {
		return false
	}

	if commandOut.State == nil || commandOut.State.Running {
		return false
	}

	// Check current thread location
	if commandOut.State.CurrentThread == nil {
		return false
	}

	currentFile := commandOut.State.CurrentThread.File
	currentLine := commandOut.State.CurrentThread.Line
	currentFunction := ""

	// Get function Name if available from breakpoint info
	if commandOut.State.CurrentThread.BreakpointInfo != nil &&
		len(commandOut.State.CurrentThread.BreakpointInfo.Stacktrace) > 0 {
		topFrame := commandOut.State.CurrentThread.BreakpointInfo.Stacktrace[0]
		if topFrame.Function != nil {
			currentFunction = topFrame.Function.Name()
		}
	}

	// Fallback: try to get function from SelectedGoroutine
	if currentFunction == "" && commandOut.State.SelectedGoroutine != nil &&
		commandOut.State.SelectedGoroutine.CurrentLoc.Function != nil {
		currentFunction = commandOut.State.SelectedGoroutine.CurrentLoc.Function.Name()
	}

	// ENHANCED SENTINEL DETECTION: Check if we've stepped into ANY adapter code
	// This includes replayer-adapter/, Temporal SDK code, or any non-workflow code
	isInAdapter := rir.isInAdapterCodeByPath(currentFile)

	if isInAdapter {
		log.Printf("%s SENTINEL DETECTED (ADAPTER CODE): %s:%d in %s", rir.Name, currentFile, currentLine, currentFunction)
		log.Printf("%s User stepped into adapter code - will auto-step back to workflow", rir.Name)
	} else {
		log.Printf("%s Command response in workflow code: %s:%d in %s", rir.Name, currentFile, currentLine, currentFunction)
	}

	return isInAdapter
}

// performDirectAutoStepping performs direct step-over commands to Delve until reaching workflow code
func (rir *ResponseInterceptingReader) performDirectAutoStepping(jsonObj []byte, remaining []byte, responseNum int, responseID string) []byte {
	log.Printf("%s AUTO-STEP: User stepped into adapter code - automatically stepping back to workflow", rir.Name)
	log.Printf("%s AUTO-STEP: Starting direct communication with Delve to step through adapter code", rir.Name)

	if rir.DelveClient == nil {
		log.Printf("%s AUTO-STEP: No Delve client available", rir.Name)
		return nil
	}

	maxSteps := 30                   // Increased limit to handle complex adapter call chains
	var lastState *api.DebuggerState // Track the last state for final response
	startTime := time.Now()

	// Extract starting location from current response for logging
	var startFile, startFunction string
	var startLine int
	if response := ExtractLocationFromCommandResponse(jsonObj); response != nil {
		startFile = response.File
		startLine = response.Line
		startFunction = response.Function
	}

	// Determine the original command type to decide if we should take an extra UX step
	// Instead of using flawed heuristics, get the actual command type from request tracking
	rir.MapMutex.Lock()
	storedMethod, exists := rir.RequestMethodMap[responseID]
	rir.MapMutex.Unlock()

	var originalCommand string
	var shouldTakeExtraStep bool

	if exists && strings.HasPrefix(storedMethod, "RPCServer.Command.") {
		// Extract the actual command Name from the stored method
		commandParts := strings.Split(storedMethod, ".")
		if len(commandParts) >= 3 {
			originalCommand = commandParts[2] // e.g., "next" or "continue"
		} else {
			originalCommand = "unknown"
		}

		// Only take extra step for step-over commands (next), not continue commands
		shouldTakeExtraStep = originalCommand == "next"
	} else {
		log.Printf("%s AUTO-STEP: Could not determine original command type for ID %s (stored: %s)", rir.Name, responseID, storedMethod)
		originalCommand = "unknown"
		shouldTakeExtraStep = false // Default to safe behavior - no extra step
	}

	log.Printf("%s AUTO-STEP: Starting from adapter code: %s:%d (%s)", rir.Name, startFile, startLine, startFunction)
	log.Printf("%s AUTO-STEP: Original command: %s, will take extra UX step: %v", rir.Name, originalCommand, shouldTakeExtraStep)
	log.Printf("%s AUTO-STEP: Will step until reaching user code (working directory)", rir.Name)

	for stepCount := 1; stepCount <= maxSteps; stepCount++ {
		log.Printf("%s AUTO-STEP: Step %d/%d - stepping through adapter code", rir.Name, stepCount, maxSteps)

		// Use delve client to send step-over command
		state, err := rir.DelveClient.Next()
		if err != nil {
			log.Printf("%s AUTO-STEP: Failed to send step command: %v", rir.Name, err)
			break
		}

		// Track the last successful state
		lastState = state

		// Check the returned state directly
		if state == nil || state.Running {
			log.Printf("%s AUTO-STEP: Received nil or running state, continuing", rir.Name)
			time.Sleep(200 * time.Millisecond) // Reduced wait time
			continue
		}

		// Update our stored state from the delve client response
		rir.StateMutex.Lock()
		var currentFile, currentFunction string
		var currentLine int
		if state.CurrentThread != nil {
			rir.CurrentFile = state.CurrentThread.File
			rir.CurrentLine = state.CurrentThread.Line
			currentFile = rir.CurrentFile
			currentLine = rir.CurrentLine

			// Try to get function Name from breakpoint info
			rir.CurrentFunction = ""
			if state.CurrentThread.BreakpointInfo != nil &&
				len(state.CurrentThread.BreakpointInfo.Stacktrace) > 0 {
				topFrame := state.CurrentThread.BreakpointInfo.Stacktrace[0]
				if topFrame.Function != nil {
					rir.CurrentFunction = topFrame.Function.Name()
				}
			}

			// Fallback: try to get function from SelectedGoroutine
			if rir.CurrentFunction == "" && state.SelectedGoroutine != nil &&
				state.SelectedGoroutine.CurrentLoc.Function != nil {
				rir.CurrentFunction = state.SelectedGoroutine.CurrentLoc.Function.Name()
			}
			currentFunction = rir.CurrentFunction
		}
		rir.StateMutex.Unlock()

		// Check if we're still in adapter code
		isInAdapter := rir.isInAdapterCodeByPath(currentFile)

		if isInAdapter {
			log.Printf("%s AUTO-STEP: Step %d - still in adapter: %s:%d (%s)",
				rir.Name, stepCount, currentFile, currentLine, currentFunction)
		} else {
			// We've reached user code!
			duration := time.Since(startTime)
			log.Printf("%s AUTO-STEP: SUCCESS! Reached user code after %d steps (%.2fs)",
				rir.Name, stepCount, duration.Seconds())
			log.Printf("%s AUTO-STEP: At user code: %s:%d (%s)",
				rir.Name, currentFile, currentLine, currentFunction)

			// Take one additional step for better UX only if this was a step-over command
			// Don't take extra step for continue commands that hit breakpoints
			var finalState *api.DebuggerState
			if shouldTakeExtraStep {
				log.Printf("%s AUTO-STEP: Taking one additional step for better user experience (step-over)", rir.Name)
				var err error
				finalState, err = rir.DelveClient.Next()
				if err != nil {
					log.Printf("%s AUTO-STEP: Failed to take final UX step: %v", rir.Name, err)
					// Use the current state as fallback
					finalState = state
				} else {
					// Update our stored state with the final step
					rir.StateMutex.Lock()
					if finalState != nil && finalState.CurrentThread != nil {
						rir.CurrentFile = finalState.CurrentThread.File
						rir.CurrentLine = finalState.CurrentThread.Line
						// Try to get function Name
						rir.CurrentFunction = ""
						if finalState.CurrentThread.BreakpointInfo != nil &&
							len(finalState.CurrentThread.BreakpointInfo.Stacktrace) > 0 {
							topFrame := finalState.CurrentThread.BreakpointInfo.Stacktrace[0]
							if topFrame.Function != nil {
								rir.CurrentFunction = topFrame.Function.Name()
							}
						}
						if rir.CurrentFunction == "" && finalState.SelectedGoroutine != nil &&
							finalState.SelectedGoroutine.CurrentLoc.Function != nil {
							rir.CurrentFunction = finalState.SelectedGoroutine.CurrentLoc.Function.Name()
						}
					}
					currentFile = rir.CurrentFile
					currentLine = rir.CurrentLine
					currentFunction = rir.CurrentFunction
					rir.StateMutex.Unlock()

					log.Printf("%s AUTO-STEP: Final location after UX step: %s:%d (%s)",
						rir.Name, currentFile, currentLine, currentFunction)
				}
			} else {
				log.Printf("%s AUTO-STEP: Skipping extra step (continue command hit breakpoint)", rir.Name)
				finalState = state
			}

			// Create a Command response with the final state for GoLand
			finalCommandResponse := map[string]interface{}{
				"id": responseID,
				"result": map[string]interface{}{
					"State": finalState, // Use the final state after the additional step
				},
			}

			finalResponseBytes, err := json.Marshal(finalCommandResponse)
			if err != nil {
				log.Printf("%s AUTO-STEP: Failed to marshal final response: %v", rir.Name, err)
				return nil
			}

			// Combine with remaining data and return to client
			modifiedBuffer := make([]byte, len(finalResponseBytes)+len(remaining))
			copy(modifiedBuffer, finalResponseBytes)
			copy(modifiedBuffer[len(finalResponseBytes):], remaining)

			if shouldTakeExtraStep {
				log.Printf("%s AUTO-STEP: Sending final Command response to GoLand - cursor moved to show progress", rir.Name)
			} else {
				log.Printf("%s AUTO-STEP: Sending final Command response to GoLand - stopped at breakpoint location", rir.Name)
			}
			return modifiedBuffer
		}
	}

	// Reached max steps - still return what we have
	duration := time.Since(startTime)
	log.Printf("%s AUTO-STEP: Reached max steps (%d) after %.2fs - may still be in adapter code",
		rir.Name, maxSteps, duration.Seconds())

	// Create a Command response with the final state for GoLand
	finalCommandResponse := map[string]interface{}{
		"id": responseID,
		"result": map[string]interface{}{
			"State": lastState, // Use the last state from the stepping loop
		},
	}

	finalResponseBytes, err := json.Marshal(finalCommandResponse)
	if err != nil {
		log.Printf("%s AUTO-STEP: Failed to marshal final response: %v", rir.Name, err)
		return nil
	}

	// Combine with remaining data and return to client
	modifiedBuffer := make([]byte, len(finalResponseBytes)+len(remaining))
	copy(modifiedBuffer, finalResponseBytes)
	copy(modifiedBuffer[len(finalResponseBytes):], remaining)

	log.Printf("%s AUTO-STEP: Sending final Command response to GoLand (max steps reached)", rir.Name)
	return modifiedBuffer
}

// getCurrentWorkingDir returns the current working directory
func getCurrentWorkingDir() string {
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

// isUserCodeFile checks if a file path is within the user's working directory
// and not part of framework/adapter code
func (rir *ResponseInterceptingReader) isUserCodeFile(filePath, workingDir string) bool {
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
	if strings.Contains(filePath, "replayer-adapter/") ||
		strings.Contains(filePath, "custom-debugger/") ||
		strings.Contains(filePath, "vendor/") ||
		strings.Contains(filePath, ".git/") {
		return false
	}

	// Also exclude Temporal SDK and Go runtime code (should be outside working dir anyway)
	if strings.Contains(filePath, "go.temporal.io/sdk/") ||
		strings.Contains(filePath, "go.temporal.io/sdk@") ||
		strings.Contains(filePath, "/runtime/") ||
		strings.Contains(filePath, "/reflect/") {
		return false
	}

	log.Printf("User code detected: %s (in working dir: %s)", filePath, workingDir)
	return true
}
