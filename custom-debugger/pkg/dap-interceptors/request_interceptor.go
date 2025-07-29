package dap_interceptors

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"

	"custom-debugger/pkg/utils"
)

type RequestInterceptingReader struct {
	reader io.Reader
	// For framing mapping when intercepting eval request from client
	responseReader   *ResponseInterceptingReader
	logPrefix        string
	requestMethodMap map[string]string
	mapMutex         *sync.Mutex
	cleanBuffer      []byte
	dirtyBuffer      []byte
	allRequestCount  int
}

func NewRequestInterceptingReader(reader io.Reader, responseReader *ResponseInterceptingReader, mapMutex *sync.Mutex, requestMethodMap map[string]string) *RequestInterceptingReader {
	return &RequestInterceptingReader{
		reader:           reader,
		responseReader:   responseReader,
		mapMutex:         mapMutex,
		requestMethodMap: requestMethodMap,
	}
}

// The same as in jsonrpc, only transformRequest is different
func (rir *RequestInterceptingReader) Read(p []byte) (n int, err error) {
	// First, check if we have modified data to send to delve
	// if rir.modifiedOffset < len(rir.modifiedData) {
	// 	// Send modified data instead of reading from client
	// 	log.Println("RequestInterceptingReader Send modified data instead of reading from client")
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

	// Normal case: read from client

	n, err = rir.reader.Read(p)
	if n > 0 {
		// Create a copy of the data for buffering to avoid modifying the original
		dataCopy := make([]byte, n)
		copy(dataCopy, p[:n])

		// Append to cleanBuffer for JSON-RPC parsing
		log.Printf("RequestInterceptingReader.Read, appending %s to cleanBuffer \n", dataCopy)
		rir.cleanBuffer = append(rir.cleanBuffer, dataCopy...)

		// Try to extract complete JSON-RPC messages and potentially modify them
		modifiedData := rir.transformRequest()

		// If we got modified data, we need to replace what we're sending to delve
		if modifiedData != nil {
			// Clear the cleanBuffer since we're replacing the data
			rir.cleanBuffer = nil

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

func (rir *RequestInterceptingReader) transformRequest() []byte {
	for len(rir.cleanBuffer) > 0 {
		// Try to find a complete JSON object in the cleanBuffer
		log.Println("calling utils.ExtractDAPMessage from request handler")
		jsonObj, remaining, found, _ := utils.ExtractDAPMessage(rir.cleanBuffer)
		if !found {
			log.Printf("RequestInterceptingReader.transformRequest can not extract json object from %s",
				rir.cleanBuffer)
			break
		}

		// Update cleanBuffer to remaining data
		rir.cleanBuffer = remaining

		rir.allRequestCount++
		requestNum := rir.allRequestCount

		jsonStr := string(jsonObj)
		log.Printf("%s 📤 DAP REQUEST #%d (%d bytes): %s", rir.logPrefix, requestNum, len(jsonObj), jsonStr[:min(150, len(jsonStr))])

		// Try to parse as DAP request first
		var dapReq utils.DAPRequest
		if err := json.Unmarshal(jsonObj, &dapReq); err == nil && dapReq.Type == "request" {
			normalizedID := fmt.Sprintf("%d", dapReq.Seq)

			log.Printf("%s 🔍 DAP REQUEST ANALYSIS: Request #%d - seq:%d, command:%s",
				rir.logPrefix, requestNum, dapReq.Seq, dapReq.Command)

			switch dapReq.Command {
			case "evaluate":
				log.Print("Request command is evaluate, doing nothing")
				// CHECK FOR DAP EVALUATE REQUEST AND TRANSLATE FRAME IDs
				// modifiedRequest := rir.translateFrameIDForEvaluateCmd(dapReq, remaining, requestNum)
				// if modifiedRequest != nil {
				// 	log.Printf("%s ✅ *** RETURNING TRANSLATED DAP EVALUATE REQUEST #%d ***", rir.logPrefix, requestNum)
				// 	return modifiedRequest
				// }
			case "scopes":
				log.Print("Request command is scopes, doing nothing")
				// CHECK FOR DAP SCOPES REQUEST AND TRANSLATE FRAME IDs
				// modifiedRequest := rir.translateFrameIDForScopesCmd(dapReq, remaining, requestNum)
				// if modifiedRequest != nil {
				// 	log.Printf("%s ✅ *** RETURNING TRANSLATED DAP SCOPES REQUEST #%d ***", rir.logPrefix, requestNum)
				// 	return modifiedRequest
				// }
			case "stackTrace":
				log.Print("Request command is stackTrace, doing nothing")
				// Check if this is a stackTrace request for thread 1 (main thread) - mark for blocking
				// if dapReq.Arguments != nil {
				// 	if argsMap, ok := dapReq.Arguments.(map[string]interface{}); ok {
				// 		if threadId, ok := argsMap["threadId"]; ok {
				// 			if threadIdFloat, ok := threadId.(float64); ok && int(threadIdFloat) == 1 {
				// 				log.Printf("%s 🚫 DETECTED MAIN THREAD STACKTRACE REQUEST #%d (seq: %d, threadId: 1)", rir.logPrefix, requestNum, dapReq.Seq)
				// 				log.Printf("%s 🚫 This will be blocked at response level to prevent dual highlighting", rir.logPrefix)
				// 			}
				// 		}
				// 	}
				// }
				// log.Printf("%s 📥 DAP STACKTRACE REQUEST #%d (seq: %d) - will be tracked for filtering",
				// 	rir.logPrefix, requestNum, dapReq.Seq)
			default:
				log.Printf("%s DAP Request #%d: %s (seq: %d)", rir.logPrefix, requestNum, dapReq.Command, dapReq.Seq)
			}

			// Track ALL command requests for response correlation
			rir.mapMutex.Lock()
			rir.requestMethodMap[normalizedID] = dapReq.Command
			log.Printf("%s 🗂️  TRACKING: DAP Request #%d (seq:%d) -> command:%s (total tracked: %d)",
				rir.logPrefix, requestNum, dapReq.Seq, dapReq.Command, len(rir.requestMethodMap))
			rir.mapMutex.Unlock()
		}
	}

	return nil // No modifications needed
}
