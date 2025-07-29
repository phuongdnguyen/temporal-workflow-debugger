package dap_interceptors

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/google/go-dap"

	"custom-debugger/pkg/utils"
)

type ResponseInterceptingReader struct {
	reader      io.Reader
	delve       *rpc2.RPCClient
	logPrefix   string
	cleanBuffer []byte
	dirtyBuffer []byte
	bufferLock  sync.Mutex
	// modifiedData        []byte
	// modifiedOffset      int
	allResponseCount    int
	stackFrameDataCount int
	mapMutex            *sync.Mutex
	requestMethodMap    map[string]string
	clientAddr          string

	frameMapping     map[int]int // Maps filtered frame index -> original frame index
	frameMappingLock sync.RWMutex
}

func NewDAPResponseInterceptingReader(delve *rpc2.RPCClient, reader io.Reader,
	logPrefix string, mapMutex *sync.Mutex, requestMethodMap map[string]string,
	clientAddr string) *ResponseInterceptingReader {
	return &ResponseInterceptingReader{
		delve:            delve,
		reader:           reader,
		logPrefix:        logPrefix,
		mapMutex:         mapMutex,
		requestMethodMap: requestMethodMap,
		clientAddr:       clientAddr,
		// TODO: might not need to init frameMapping here
		frameMapping:     make(map[int]int),
		frameMappingLock: sync.RWMutex{},
		bufferLock:       sync.Mutex{},
	}
}

// Same logic as jsonrpc, consider refactoring it later
func (rir *ResponseInterceptingReader) Read(p []byte) (n int, err error) {
	// First, check if we have modified data to send to client
	// if rir.modifiedOffset < len(rir.modifiedData) {
	// 	log.Println("we have modified data to send to client")
	// 	// Send modified data instead of reading from delve
	// 	remaining := len(rir.modifiedData) - rir.modifiedOffset
	// 	bytesToCopy := len(p)
	// 	if remaining < bytesToCopy {
	// 		bytesToCopy = remaining
	// 	}
	//
	// 	copy(p, rir.modifiedData[rir.modifiedOffset:rir.modifiedOffset+bytesToCopy])
	// 	rir.modifiedOffset += bytesToCopy
	//
	// 	// If we've sent all modified data, reset
	// 	if rir.modifiedOffset >= len(rir.modifiedData) {
	// 		rir.modifiedData = nil
	// 		rir.modifiedOffset = 0
	// 	}
	//
	// 	log.Printf("%s: %d bytes (modified)", rir.logPrefix, bytesToCopy)
	// 	return bytesToCopy, nil
	// }
	// Normal case: read from delve server
	n, err = rir.reader.Read(p)
	if err != nil {
		log.Printf("failed to read response from Delve server: %v", err)
	}
	log.Printf("Read %d bytes from delve", n)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])
		rir.bufferLock.Lock()
		// Append to cleanBuffer for JSON-RPC parsing
		log.Printf("Current cleanBuffer: %s", string(rir.cleanBuffer))
		if len(rir.dirtyBuffer) > 0 {
			log.Printf("Appending data from previous dirty buffer: %s", string(rir.dirtyBuffer))
			rir.cleanBuffer = append(rir.cleanBuffer, rir.dirtyBuffer...)
			log.Println("Reseting dirty buffer")
			rir.dirtyBuffer = nil
		}
		log.Printf("Appending data from delve: %s", string(dataCopy))
		rir.cleanBuffer = append(rir.cleanBuffer, dataCopy...)
		rir.bufferLock.Unlock()
		// Try to extract complete JSON-RPC messages and potentially modify them
		modifiedData := rir.transformResponse()

		// Check if response was suppressed (nil means suppress)
		if modifiedData == nil && len(rir.cleanBuffer) == 0 {
			// Response was suppressed - don't send anything to client
			log.Printf("%s: 0 bytes (response suppressed)", rir.logPrefix)
			return 0, nil
		}

		// If we got modified data, we need to replace what we're sending to client
		if modifiedData != nil {
			// Clear the cleanBuffer since we're replacing the data
			rir.bufferLock.Lock()
			if len(rir.cleanBuffer) != 0 {
				log.Printf("Will clear cleanBuffer that has %s in it", string(rir.cleanBuffer))
				rir.cleanBuffer = nil
			}
			if len(rir.dirtyBuffer) != 0 {
				log.Printf("Keeping dirty buffer that have in-completed data in it")
			}
			rir.bufferLock.Unlock()

			// rir.modifiedData = modifiedData
			// rir.modifiedOffset = 0

			// Send the first part of modified data
			bytesToCopy := len(p)
			if len(modifiedData) < bytesToCopy {
				bytesToCopy = len(modifiedData)
			}

			copy(p, modifiedData[:bytesToCopy])
			// rir.modifiedOffset = bytesToCopy

			log.Printf("%s: %d bytes (replaced with modified)", rir.logPrefix, bytesToCopy)
			return bytesToCopy, err
		}
		log.Printf("%s: %d bytes", rir.logPrefix, n)
	}
	return n, err
}

// TODO: change this function's name to parseAndModifyResponses
func (rir *ResponseInterceptingReader) transformResponse() []byte {
	// Safety check: prevent cleanBuffer from growing too large
	// const maxBufferSize = 10 * 1024 * 1024 // 10MB limit
	// if len(rir.cleanBuffer) > maxBufferSize {
	// 	log.Printf("%s cleanBuffer too large (%d bytes), resetting to prevent memory issues", rir.logPrefix, len(rir.cleanBuffer))
	// 	rir.bufferLock.Lock()
	// 	if len(rir.cleanBuffer) != 0 {
	// 		log.Printf("Buffer is too large, will clear cleanBuffer that has %s in it", string(rir.cleanBuffer))
	// 	}
	// 	rir.cleanBuffer = nil
	// 	rir.bufferLock.Unlock()
	// 	return nil
	// }

	// Prevent infinite loops by limiting iterations
	maxIterations := 100
	iterations := 0
	if len(rir.cleanBuffer) == 0 {
		log.Printf("parseReponses, no data in cleanBuffer")
	}
	for len(rir.cleanBuffer) > 0 && iterations < maxIterations {
		iterations++

		// Try to find a complete JSON object in the cleanBuffer
		log.Println("calling utils.ExtractDAPMessage from response handler")
		jsonObj, remainingCompletedObjs, found, remainingIncompleted := utils.ExtractDAPMessage(rir.cleanBuffer)
		if !found {
			log.Printf("transformResponse: failed to extract json object from cleanBuffer")
			// No complete JSON object found, wait for more data
			break
		}

		// Safety check: ensure we're making progress
		if len(remainingCompletedObjs) >= len(rir.cleanBuffer) {
			log.Printf("%s No progress in cleanBuffer parsing, breaking to prevent infinite loop", rir.logPrefix)
			break
		}

		// Update cleanBuffer to remainingCompletedObjs data
		rir.bufferLock.Lock()
		rir.cleanBuffer = remainingCompletedObjs
		if len(remainingIncompleted) > 0 {

			rir.dirtyBuffer = append(rir.dirtyBuffer, remainingIncompleted...)
		}
		rir.bufferLock.Unlock()

		rir.allResponseCount++
		responseNum := rir.allResponseCount

		jsonStr := string(jsonObj)
		log.Printf("📤 DAP RESPONSE #%d (%d bytes): %s", responseNum, len(jsonObj),
			jsonStr[:utils.Min(150, len(jsonStr))])

		// Parse as DAP message
		msg, err := dap.DecodeProtocolMessage(jsonObj)
		if err != nil {
			log.Printf("Can not unmarmal to dap.Message: %v", err)
		}

		switch msg := msg.(type) {
		case *dap.ThreadsResponse:
			if len(msg.Body.Threads) == 1 && msg.Body.Threads[0].Id == -1 && msg.Body.Threads[0].Name == "Current" {
				log.Printf("Doing nothing with baseline DAP.ThreadsResponse, msg: %+v, seq: %d\n", msg, msg.RequestSeq)
			}
			// Filter the threads response and return modified response
			// filteredResponse := rir.filterDAPThreadsResponse(jsonObj, msg, remainingCompletedObjs)
			// if filteredResponse != nil {
			// 	log.Printf("RETURNING FILTERED DAP THREADS Response #%d TO CLIENT \n", responseNum)
			// 	return filteredResponse
			// }
			log.Printf("Doing nothing with DAP.ThreadsResponse")
			return rir.buildDAPMessage(jsonObj, remainingCompletedObjs)
			// filteredResponse := rir.filterDAPThreadsResponse(jsonObj, msg,  remainingCompletedObjs)
			// if filteredResponse != nil {
			// 	log.Printf("%s ✅ *** RETURNING FILTERED DAP THREADS Response #%d TO CLIENT ***", rir.logPrefix,
			// 		responseNum)
			// 	return filteredResponse
			// }
		case *dap.StackTraceResponse:
			log.Printf("Doing nothing with DAP.StackTraceResponse")
			return rir.buildDAPMessage(jsonObj, remainingCompletedObjs)
			// // // Filter the stacktrace and return modified response
			// filteredResponse := rir.filterDAPStacktraceResponse(jsonObj, remainingCompletedObjs)
			// if filteredResponse != nil {
			// 	// 	// EXPERIMENTAL: Try to force VS Code to use the correct current location
			// 	// 	// by ensuring the first frame in our filtered response is clearly marked as current
			// 	// 	rir.debugCurrentLocationOverride(filteredResponse)
			// 	return filteredResponse
			// }
		case *dap.ContinueResponse, *dap.NextResponse:
			log.Println("Got continue/next response from DAP, doing nothing")
			return rir.buildDAPMessage(jsonObj, remainingCompletedObjs)
		case *dap.StoppedEvent:
			log.Println("Got stopped event from DAP")
			return rir.handleStoppedEvent(msg, jsonObj, remainingCompletedObjs)
			// return rir.buildDAPMessage(jsonObj, remainingCompletedObjs)
		default:
			log.Printf("Received response  from DAP, doing nothing. Message type: %T", msg)
			return rir.buildDAPMessage(jsonObj, remainingCompletedObjs)
		}
	}

	// Check if we hit the iteration limit
	if iterations >= maxIterations {
		log.Printf("%s Reached maximum iterations (%d) in transformResponse, cleanBuffer length: %d", rir.logPrefix, maxIterations, len(rir.cleanBuffer))
	}

	return nil // No modifications needed
}

func (rir *ResponseInterceptingReader) handleStoppedEvent(event *dap.StoppedEvent,
	jsonObj []byte, remaining []byte) []byte {
	log.Println("handleStoppedEvent start")
	if event.Body.Reason == "exception" || event.Body.Reason == "unknown" {
		// don't do anything if there is an exception
		log.Printf("Ignoring stopped event with reason %s\n", event.Body.Reason)
		return rir.buildDAPMessage(jsonObj, remaining)
	}

	threads, err := rir.delve.ListThreads()
	if err != nil {
		log.Printf("Can not list threads: %v", err)
		return rir.buildDAPMessage(jsonObj, remaining)
	}

	for _, thread := range threads {
		if thread.GoroutineID == int64(event.Body.ThreadId) {
			if thread.Function.Name() == "replayer_adapter.raiseSentinelBreakpoint" {
				log.Printf("Found thread %d with goroutine id %d is adapter code", thread.ID, thread.GoroutineID)
				// 	Step until user code
				maxSteps := 30
				for step := 0; step < maxSteps; step++ {
					log.Printf("Stepping till workflow code, step %d", step)
					state, err := rir.delve.Next()
					if err != nil {
						log.Printf("Can not step over, step %d, err: %v\n", step, err)
						continue
					}
					if state == nil || state.Running {
						log.Printf("%s AUTO-STEP: Received nil or running state, continuing", rir.logPrefix)
						time.Sleep(200 * time.Millisecond) // Reduced wait time
						continue
					}
					for _, sThread := range state.Threads {
						// Try to get function name from breakpoint info
						currentFile := sThread.File
						// Check if we're still in adapter code
						isInAdapter := utils.IsInAdapterCodeByPath(currentFile)
						if !isInAdapter {
							log.Printf("Reached workflow code, file %s, line %d, function %s", sThread.File, sThread.Line, sThread.Function.Name())
							// 	TODO: UX Step
							log.Printf("Changing thread id from %d to %d", event.Body.ThreadId, sThread.GoroutineID)
							event.Body.ThreadId = int(sThread.GoroutineID)
							finalResponseBytes, err := json.Marshal(event)
							if err != nil {
								log.Printf("Can not marshal event: %v", err)
								return rir.buildDAPMessage(jsonObj, remaining)
							}
							return rir.buildDAPMessage(finalResponseBytes, remaining)
						}
					}
				}
			}
		}
	}
	return rir.buildDAPMessage(jsonObj, remaining)
}

func (rir *ResponseInterceptingReader) logDAPStacktraceResponse(jsonLine string) {
	log.Printf("DAP Stack trace response (%s)", jsonLine)
}

func (rir *ResponseInterceptingReader) filterDAPStacktraceResponse(jsonObj []byte, remaining []byte) []byte {
	return jsonObj
}

// Consider using a cleaner approach dap.WriteBaseMessage function to build message
// buildDAPMessage constructs a properly formatted DAP message with correct Content-Length header
func (rir *ResponseInterceptingReader) buildDAPMessage(jsonPayload []byte, remaining []byte) []byte {
	log.Printf("building DAP Message, jsonPayload: %s", string(jsonPayload))
	log.Printf("building DAP Message, remaining: %s", string(remaining))
	// DAP messages format: Content-Length: XXX\r\n\r\n{JSON}
	contentLength := len(jsonPayload)
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", contentLength)
	headerBytes := []byte(header)

	// Build complete DAP message
	dapMessage := make([]byte, len(headerBytes)+len(jsonPayload))
	copy(dapMessage, headerBytes)
	copy(dapMessage[len(headerBytes):], jsonPayload)

	// Combine DAP message with remaining cleanBuffer data
	modifiedBuffer := make([]byte, len(dapMessage)+len(remaining))
	copy(modifiedBuffer, dapMessage)
	copy(modifiedBuffer[len(dapMessage):], remaining)

	log.Printf("[%s] buildDAPMessage Complete message: %s", rir.clientAddr, string(dapMessage))
	log.Printf("[%s] buildDAPMessage modifiedBuffer: %s", rir.clientAddr, string(modifiedBuffer))

	return modifiedBuffer
}
