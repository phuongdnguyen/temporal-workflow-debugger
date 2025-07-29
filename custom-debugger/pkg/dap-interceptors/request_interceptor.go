package dap_interceptors

import (
	"encoding/json"
	"io"
	"log"

	"custom-debugger/pkg/utils"
)

type RequestInterceptingReader struct {
	reader io.Reader
	// For framing mapping when intercepting eval request from client
	responseReader  *ResponseInterceptingReader
	logPrefix       string
	cleanBuffer     []byte
	dirtyBuffer     []byte
	allRequestCount int
}

func NewRequestInterceptingReader(reader io.Reader, responseReader *ResponseInterceptingReader) *RequestInterceptingReader {
	return &RequestInterceptingReader{
		reader:         reader,
		responseReader: responseReader,
	}
}

// The same as in json-rpc, only transformRequest is different
func (rir *RequestInterceptingReader) Read(p []byte) (n int, err error) {
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

			// Send the first part of modified data
			bytesToCopy := len(p)
			if len(modifiedData) < bytesToCopy {
				bytesToCopy = len(modifiedData)
			}

			copy(p, modifiedData[:bytesToCopy])

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

		var dapReq utils.DAPRequest
		if err := json.Unmarshal(jsonObj, &dapReq); err == nil && dapReq.Type == "request" {
			switch dapReq.Command {
			default:
				log.Printf("%s DAP Request #%d: %s (seq: %d)", rir.logPrefix, requestNum, dapReq.Command, dapReq.Seq)
			}
		}
	}

	return nil // No modifications needed
}
