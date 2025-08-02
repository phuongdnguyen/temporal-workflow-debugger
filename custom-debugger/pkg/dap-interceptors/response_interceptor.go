package dap_interceptors

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/google/go-dap"

	"custom-debugger/pkg/daptest"
	"custom-debugger/pkg/utils"
)

type ResponseInterceptingReader struct {
	reader           net.Conn
	delve            *rpc2.RPCClient
	debugger         *daptest.Client
	logPrefix        string
	cleanBuffer      []byte
	dirtyBuffer      []byte
	bufferLock       sync.Mutex
	allResponseCount int
}

func NewDAPResponseInterceptingReader(delve *rpc2.RPCClient, debugger *daptest.Client, reader net.Conn, logPrefix string) *ResponseInterceptingReader {
	return &ResponseInterceptingReader{
		delve:      delve,
		debugger:   debugger,
		reader:     reader,
		logPrefix:  logPrefix,
		bufferLock: sync.Mutex{},
	}
}

// Read data in chunks from the server -> debugger stream, process & modify completed chunks
func (rir *ResponseInterceptingReader) Read(p []byte) (n int, err error) {
	n, err = rir.reader.Read(p)
	if err != nil {
		log.Printf("failed to read response from Delve server: %v", err)
	}
	log.Printf("About to process %d bytes from delve", n)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])
		rir.bufferLock.Lock()
		// Append to cleanBuffer for DAP parsing
		log.Printf("Current cleanBuffer: %s\n", string(rir.cleanBuffer))
		if len(rir.dirtyBuffer) > 0 {
			log.Printf("Appending data from previous dirty buffer: %s", string(rir.dirtyBuffer))
			rir.cleanBuffer = append(rir.cleanBuffer, rir.dirtyBuffer...)
			log.Println("Resetting dirty buffer")
			rir.dirtyBuffer = nil
		}
		log.Printf("Appending data from delve: %s", string(dataCopy))
		rir.cleanBuffer = append(rir.cleanBuffer, dataCopy...)
		rir.bufferLock.Unlock()
		// Try to extract complete DAP messages and potentially modify them
		modifiedData := rir.transformResponse()

		// If transformResponse returned nil, it means we don't have a complete message yet
		// Don't send partial data to the client - wait for more data
		if modifiedData == nil {
			log.Printf("%s: 0 bytes (waiting for complete DAP message)", rir.logPrefix)
			return 0, nil
		}

		// If we got modified data, we need to replace what we're sending to client
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
	log.Printf("transform response, buffer: %s", string(rir.cleanBuffer))
	var responseBuffer []byte
	for len(rir.cleanBuffer) > 0 && iterations < maxIterations {
		iterations++
		log.Printf("iterations: %d", iterations)

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
		case *dap.StoppedEvent:
			log.Println("Got stopped event from DAP")
			return rir.handleStoppedEvent(msg, jsonObj, remainingCompletedObjs)
		default:
			// If this is message that we're not interested in, buffer it to return later
			log.Printf("Received response  from DAP, appending to response buffer. Message type: %T", msg)
			responseBuffer = append(responseBuffer, utils.BuildDAPMessage(jsonObj)...)
			// return utils.BuildDAPMessages(jsonObj, remainingCompletedObjs)
		}
	}

	// Check if we hit the iteration limit
	if iterations >= maxIterations {
		log.Printf("Reached maximum iterations (%d) in transformResponse, returning buffered result: %s", maxIterations, string(responseBuffer))
	}

	return responseBuffer // No modifications needed
}

func (rir *ResponseInterceptingReader) handleStoppedEvent(event *dap.StoppedEvent,
	jsonObj []byte, remaining []byte) []byte {
	log.Println("handleStoppedEvent start")
	if event.Body.Reason == "exception" || event.Body.Reason == "unknown" || event.Body.Reason == "step" {
		// don't do anything
		log.Printf("Ignoring stopped event with reason %s\n", event.Body.Reason)
		return utils.BuildDAPMessages(jsonObj, remaining)
	}
	lang := os.Getenv("LANGUAGE")
	if len(lang) == 0 {
		lang = "go"
	}
	switch lang {
	case "go":
		threads, err := rir.delve.ListThreads()
		if err != nil {
			log.Printf("Can not list threads: %v", err)
			return utils.BuildDAPMessages(jsonObj, remaining)
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
									return utils.BuildDAPMessages(jsonObj, remaining)
								}
								return utils.BuildDAPMessages(finalResponseBytes, remaining)
							}
						}
					}
				}
			}
		}
		return utils.BuildDAPMessages(jsonObj, remaining)
	case "python":
		println("Pythonnnnn")
		fmt.Printf("Python: stopped event: %+v\n", event)
		log.Println("Python: Create client from existing connection")
		client := daptest.NewClientFromConn(rir.reader)
		log.Printf("Python: Getting stack trace for thread %d", event.Body.ThreadId)
		client.StackTraceRequest(event.Body.ThreadId, 0, 20)
		resp, err := client.GetStacktraceResponse()
		if err != nil {
			log.Printf("Python: Can not get stack response: %v", err)
			return utils.BuildDAPMessages(jsonObj, remaining)
		}
		var totalBuf []byte
		for _, frame := range resp.Body.StackFrames {
			log.Printf("Checking Frame with source: %+v\n", *frame.Source)
			if utils.IsInAdapterCodeByPath(frame.Source.Path) {
				log.Printf("Frame with path %s is in adapter code, stepping out\n", frame.Source.Path)
				// 	Step until user code
				maxSteps := 30
				for step := 0; step < maxSteps; step++ {
					log.Printf("Stepping till workflow code, step %d", step)
					client.NextRequest(event.Body.ThreadId)
					time.Sleep(500 * time.Millisecond)
					nextResponse, buf, err := client.GetNextResponseWithFiltering()
					if err != nil {
						log.Printf("Can not step over, step %d, err: %v\n", step, err)
						continue
					}
					if !nextResponse.Success {
						log.Printf("Can not step over, step %d, success: %v", step, nextResponse.Success)
						break
					}
					log.Println("Appending messages batch to buffer")
					totalBuf = append(totalBuf, buf...)
					log.Printf("Getting stacktrace for thread id: %d", event.Body.ThreadId)
					client.StackTraceRequest(event.Body.ThreadId, 0, 20)
					getStacktraceResp, buf, err := client.GetStacktraceResponseWithFiltering()
					if err != nil {
						log.Printf("Can not get stack response: %v", err)
						break
					}
					if !getStacktraceResp.Success {
						log.Printf("Can not get stack trace, step %d, success: %v", step, nextResponse.Success)
						break
					}
					log.Println("Appending messages batch to buffer")
					totalBuf = append(totalBuf, buf...)
					frame := getStacktraceResp.Body.StackFrames[0]
					// for _, frame := range getStacktraceResp.Body.StackFrames {
					log.Printf("Checking frame file: %s, line: %d", frame.Source.Path, frame.Line)
					if !utils.IsInAdapterCodeByPath(frame.Source.Path) {
						println("WORKFLOW FRAME")
						log.Printf("FOUNDDDD file: %s, line: %d", frame.Source.Path, frame.Line)
						log.Printf("FOUNDDDD Frame.Source: %+v\n", *frame.Source)
						log.Println("==================================")
						log.Printf("total buff: %+v\n", string(totalBuf))
						log.Println("==================================")
						b, err := json.Marshal(event)
						if err != nil {
							log.Printf("Can not marshal event: %v", err)
							return utils.BuildDAPMessages(jsonObj, remaining)
						}
						return utils.BuildDAPMessages(b, remaining)
					}
					// }
				}
			}
		}

		return utils.BuildDAPMessages(jsonObj, remaining)
	}
	return utils.BuildDAPMessages(jsonObj, remaining)
}
