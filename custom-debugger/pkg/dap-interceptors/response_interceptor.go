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

	"custom-debugger/pkg/dap-client"
	"custom-debugger/pkg/extractors"
	"custom-debugger/pkg/locators"
	"custom-debugger/pkg/utils"
)

type ResponseInterceptingReader struct {
	reader      net.Conn
	delve       *rpc2.RPCClient
	debugger    *dap_client.Client
	logPrefix   string
	cleanBuffer []byte
	// incomplete messages from the previous reads
	inCompletedMessages []byte
	bufferLock          sync.Mutex
	allResponseCount    int
	log                 *log.Logger
}

func NewDAPResponseInterceptingReader(delve *rpc2.RPCClient, debugger *dap_client.Client, reader net.Conn, logPrefix string) *ResponseInterceptingReader {
	return &ResponseInterceptingReader{
		delve:      delve,
		debugger:   debugger,
		reader:     reader,
		logPrefix:  logPrefix,
		bufferLock: sync.Mutex{},
		log:        log.New(os.Stdout, logPrefix, log.LstdFlags),
	}
}

// Read data in chunks from the server -> debugger stream, process & modify completed chunks
func (rir *ResponseInterceptingReader) Read(p []byte) (n int, err error) {
	n, err = rir.reader.Read(p)

	if err != nil {
		return n, fmt.Errorf("failed to read response from Debugger: %w", err)
	}
	rir.log.Printf("About to process %d bytes from debugger", n)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])
		rir.bufferLock.Lock()
		// Append to cleanBuffer for DAP parsing
		rir.log.Printf("Current cleanBuffer: %s\n", string(rir.cleanBuffer))
		if len(rir.inCompletedMessages) > 0 {
			rir.log.Printf("Appending data from previous dirty buffer: %s", string(rir.inCompletedMessages))
			rir.cleanBuffer = append(rir.cleanBuffer, rir.inCompletedMessages...)
			rir.log.Println("Resetting dirty buffer")
			rir.inCompletedMessages = nil
		}
		rir.log.Printf("Appending data from debugger: %s", string(dataCopy))
		rir.cleanBuffer = append(rir.cleanBuffer, dataCopy...)
		rir.bufferLock.Unlock()
		// Try to extract complete DAP messages and potentially modify them
		modifiedData := rir.transformResponse()

		// If transformResponse returned nil, it means we don't have a complete message yet
		// Don't send partial data to the client - wait for more data
		if modifiedData == nil {
			rir.log.Printf("%s: 0 bytes (waiting for complete DAP message)", rir.logPrefix)
			return 0, nil
		}

		// If we got modified data, we need to replace what we're sending to client
		// Clear the cleanBuffer since we're replacing the data
		rir.bufferLock.Lock()
		if len(rir.cleanBuffer) != 0 {
			rir.log.Printf("Will clear cleanBuffer that has %s in it", string(rir.cleanBuffer))
			rir.cleanBuffer = nil
		}
		if len(rir.inCompletedMessages) != 0 {
			rir.log.Printf("Keeping dirty buffer that have in-completed data in it")
		}
		rir.bufferLock.Unlock()

		// Send the first part of modified data
		bytesToCopy := len(p)
		if len(modifiedData) < bytesToCopy {
			bytesToCopy = len(modifiedData)
		}

		copy(p, modifiedData[:bytesToCopy])
		// rir.modifiedOffset = bytesToCopy

		rir.log.Printf("%s: %d bytes (replaced with modified)", rir.logPrefix, bytesToCopy)
		return bytesToCopy, err
	}
	return n, err
}

func (rir *ResponseInterceptingReader) transformResponse() []byte {
	// Prevent infinite loops by limiting iterations
	maxIterations := 100
	iterations := 0
	if len(rir.cleanBuffer) == 0 {
		rir.log.Printf("No data in cleanBuffer")
	}
	rir.log.Printf("Buffer: %s", string(rir.cleanBuffer))
	var responseBuffer []byte
	for len(rir.cleanBuffer) > 0 && iterations < maxIterations {
		iterations++
		rir.log.Printf("iterations: %d", iterations)

		// Try to find a complete JSON object in the cleanBuffer
		jsonObj, remainingCompletedObjs, found, remainingIncompleted := extractors.ExtractDAPMessage(rir.cleanBuffer)
		if !found {
			rir.log.Printf("Failed to extract json object from cleanBuffer")
			// No complete JSON object found, wait for more data
			break
		}

		// Safety check: ensure we're making progress
		if len(remainingCompletedObjs) >= len(rir.cleanBuffer) {
			rir.log.Printf("%s No progress in cleanBuffer parsing, breaking to prevent infinite loop", rir.logPrefix)
			break
		}

		// Update cleanBuffer to remainingCompletedObjs data
		rir.bufferLock.Lock()
		rir.cleanBuffer = remainingCompletedObjs
		if len(remainingIncompleted) > 0 {
			rir.inCompletedMessages = append(rir.inCompletedMessages, remainingIncompleted...)
		}
		rir.bufferLock.Unlock()

		rir.allResponseCount++
		responseNum := rir.allResponseCount

		jsonStr := string(jsonObj)
		rir.log.Printf("📤 DAP RESPONSE #%d (%d bytes): %s", responseNum, len(jsonObj),
			jsonStr[:utils.Min(150, len(jsonStr))])

		// Parse as DAP message
		msg, err := dap.DecodeProtocolMessage(jsonObj)
		if err != nil {
			rir.log.Printf("Can not unmarmal to dap.Message: %v", err)
		}

		switch msg := msg.(type) {
		case *dap.StoppedEvent:
			rir.log.Println("Got stopped event from Debugger")
			return rir.handleStoppedEvent(msg, jsonObj, remainingCompletedObjs)
		default:
			// If this is the message that we're not interested in, buffer it to return later
			rir.log.Printf("Received response  from DAP, appending to response buffer. Message type: %T", msg)
			responseBuffer = append(responseBuffer, extractors.BuildDAPMessage(jsonObj)...)
		}
	}

	// Check if we hit the iteration limit
	if iterations >= maxIterations {
		rir.log.Printf("Reached maximum iterations (%d) in transformResponse, returning buffered result: %s", maxIterations, string(responseBuffer))
	}

	return responseBuffer
}

func (rir *ResponseInterceptingReader) handleStoppedEvent(event *dap.StoppedEvent,
	jsonObj []byte, remaining []byte) []byte {
	rir.log.Println("handleStoppedEvent start")
	if event.Body.Reason == "exception" || event.Body.Reason == "unknown" || event.Body.Reason == "step" {
		// don't do anything
		rir.log.Printf("Ignoring stopped event with reason %s", event.Body.Reason)
		return extractors.BuildDAPMessages(jsonObj, remaining)
	}
	lang := utils.GetLang()
	switch lang {
	case utils.GoDelve:
		threads, err := rir.delve.ListThreads()
		if err != nil {
			rir.log.Printf("Can not list threads: %v", err)
			return extractors.BuildDAPMessages(jsonObj, remaining)
		}

		for _, thread := range threads {
			if thread.GoroutineID == int64(event.Body.ThreadId) {
				if thread.Function.Name() == "replayer_adapter.raiseSentinelBreakpoint" {
					rir.log.Printf("Found thread %d with goroutine id %d is adapter code", thread.ID, thread.GoroutineID)
					// 	Step until user code
					maxSteps := 50
					for step := 0; step < maxSteps; step++ {
						rir.log.Printf("Stepping till workflow code, step %d", step)
						state, err := rir.delve.Next()
						if err != nil {
							rir.log.Printf("Can not step over, step %d, err: %v\n", step, err)
							continue
						}
						if state == nil || state.Running {
							rir.log.Printf("%s AUTO-STEP: Received nil or running state, continuing", rir.logPrefix)
							time.Sleep(200 * time.Millisecond) // Reduced wait time
							continue
						}
						for _, sThread := range state.Threads {
							// Try to get function name from breakpoint info
							currentFile := sThread.File
							// Check if we're still in adapter code
							isInAdapter := locators.IsInAdapterCodeByPath(currentFile)
							if !isInAdapter {
								rir.log.Printf("Reached workflow code, file %s, line %d, function %s", sThread.File, sThread.Line, sThread.Function.Name())
								// 	TODO: UX Step
								rir.log.Printf("Changing thread id from %d to %d", event.Body.ThreadId, sThread.GoroutineID)
								event.Body.ThreadId = int(sThread.GoroutineID)
								finalResponseBytes, err := json.Marshal(event)
								if err != nil {
									rir.log.Printf("Can not marshal event: %v", err)
									return extractors.BuildDAPMessages(jsonObj, remaining)
								}
								return extractors.BuildDAPMessages(finalResponseBytes, remaining)
							}
						}
					}
				}
			}
		}
		return extractors.BuildDAPMessages(jsonObj, remaining)
	case utils.Python, utils.GoDAP:
		fmt.Printf("Stopped event: %+v", event)
		rir.log.Println("Create client from existing connection")
		client := dap_client.NewClientFromConn(rir.reader)
		rir.log.Printf("Getting stack trace for thread %d", event.Body.ThreadId)
		client.StackTraceRequest(event.Body.ThreadId, 0, 20)
		resp, err := client.GetStacktraceResponse()
		if err != nil {
			rir.log.Printf("Can not get stack response: %v", err)
			return extractors.BuildDAPMessages(jsonObj, remaining)
		}
		// var totalBuf []byte
		for _, frame := range resp.Body.StackFrames {
			rir.log.Printf("Checking Frame with source: %+v", *frame.Source)
			if locators.IsInAdapterCodeByPath(frame.Source.Path) {
				rir.log.Printf("Frame with path %s is in adapter code, stepping out", frame.Source.Path)
				// 	Step until user code
				maxSteps := 30
				for step := 0; step < maxSteps; step++ {
					rir.log.Printf("Stepping until workflow code, step %d", step)
					client.NextRequest(event.Body.ThreadId)
					// Wait for debugger state to settle down
					time.Sleep(500 * time.Millisecond)
					nextResponse, _, err := client.GetNextResponseWithFiltering()
					if err != nil {
						rir.log.Printf("Can not step over, step %d, err: %v", step, err)
						continue
					}
					if !nextResponse.Success {
						rir.log.Printf("Can not step over, step %d, success: %v", step, nextResponse.Success)
						break
					}
					rir.log.Println("Appending messages batch to buffer")
					// totalBuf = append(totalBuf, buf...)
					rir.log.Printf("Getting stacktrace for thread id: %d", event.Body.ThreadId)
					client.StackTraceRequest(event.Body.ThreadId, 0, 20)
					getStacktraceResp, _, err := client.GetStacktraceResponseWithFiltering()
					if err != nil {
						rir.log.Printf("Can not get stack response: %v", err)
						break
					}
					if !getStacktraceResp.Success {
						rir.log.Printf("Can not get stack trace, step %d, success: %v", step, nextResponse.Success)
						break
					}
					rir.log.Println("Appending messages batch to buffer")
					// totalBuf = append(totalBuf, buf...)
					frame := getStacktraceResp.Body.StackFrames[0]
					rir.log.Printf("Checking frame file: %s, line: %d", frame.Source.Path, frame.Line)
					if !locators.IsInAdapterCodeByPath(frame.Source.Path) {
						println("WORKFLOW FRAME")
						rir.log.Printf("Found user code frame file: %s, line: %d", frame.Source.Path, frame.Line)
						b, err := json.Marshal(event)
						if err != nil {
							rir.log.Printf("Can not marshal event: %v", err)
							return extractors.BuildDAPMessages(jsonObj, remaining)
						}
						return extractors.BuildDAPMessages(b, remaining)
					}
				}
			}
		}

		return extractors.BuildDAPMessages(jsonObj, remaining)
	}
	return extractors.BuildDAPMessages(jsonObj, remaining)
}
