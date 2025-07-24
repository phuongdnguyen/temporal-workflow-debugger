package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/go-delve/delve/service/rpc2"
)

func handleClientConnection(clientTCP net.Conn) {
	clientAddr := clientTCP.RemoteAddr().String()
	log.Printf("New client connected from %s", clientAddr)

	// Set keep-alive to detect dead connections
	if tcpConn, ok := clientTCP.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Printf("Error enable keep alive on client connection: %v", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			log.Printf("Error setting keep alive period on client connection: %v", err)
		}
	}

	// Set read/write timeouts to prevent hanging
	// No read timeout for client
	if err := clientTCP.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("Error set read deadline on client connection: %v", err)
	}
	// No write timeout for client
	if err := clientTCP.SetWriteDeadline(time.Time{}); err != nil {
		log.Printf("Error set write deadline on client connection: %v", err)
	}

	// Dial real Delve with retry logic
	delveTCP, err := dialDelveWithRetry("localhost:2345", 3, time.Second)
	if err != nil {
		log.Printf("Error connecting to Delve server for %s after retries: %v", clientAddr, err)
		if err := clientTCP.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}
		return
	}
	log.Println("Connected to Delve server")

	// Set keep-alive for delve connection too
	if tcpConn, ok := delveTCP.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Printf("Error enable keep alive on client connection: %v", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			log.Printf("Error setting keep alive period on client connection: %v", err)
		}
	}

	// Set timeouts for delve connection as well
	// No read timeout for delve
	if err := delveTCP.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("Error setting read deadline on client connection: %v", err)
	}
	// No write timeout for delve
	if err := delveTCP.SetWriteDeadline(time.Time{}); err != nil {
		log.Printf("Error setting write deadline on client connection: %v", err)
	}

	// Channel to signal when one side closes
	done := make(chan struct{}, 2)

	// Map to track request IDs to method names for response interception
	requestMethodMap := make(map[string]string)
	var mapMutex sync.Mutex

	// Create delve client for auto-stepping operations
	delveClient := rpc2.NewClient("localhost:2345")

	// Create response interceptor first so we can reference it
	delveReader := &responseInterceptingReader{
		reader:           delveTCP,
		name:             fmt.Sprintf("[%s] Delve->Client", clientAddr),
		requestMethodMap: requestMethodMap,
		mapMutex:         &mapMutex,
		clientAddr:       clientAddr,

		// Enhanced debugging counters
		stackTraceCount:     0,
		stackFrameDataCount: 0,
		allResponseCount:    0,
		mainThreadMutex:     sync.Mutex{},

		// Frame mapping for JSON-RPC stacktrace filtering
		frameMapping:     make(map[int]int),
		frameMappingLock: sync.RWMutex{},

		// Auto-stepping infrastructure
		delveClient: delveClient,

		// Current state tracking for sentinel breakpoint detection
		currentFile:     "",             // Current file location
		currentFunction: "",             // Current function name
		currentLine:     0,              // Current line number
		stateMutex:      sync.RWMutex{}, // Protects current state fields

		// Reference to request reader for step over tracking
		requestReader: nil, // Will be set after clientReader is created
	}

	// Ensure both connections are closed when function exits
	defer func() {
		log.Printf("Closing connections for client %s", clientAddr)
		if err := clientTCP.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}

		if err := delveTCP.Close(); err != nil {
			log.Printf("Error closing delve connection: %v", err)
		}

		// Log debugging summary
		delveReader.logDebuggingSummary()

		log.Printf("Client %s disconnected", clientAddr)
	}()

	// Handle client->delve with request tracking and transparent forwarding
	go func() {
		defer func() {
			log.Printf("Client->Delve goroutine ending for %s", clientAddr)
			done <- struct{}{}
		}()

		log.Println("Client->Delve")

		// Create intercepting reader that also tracks requests
		clientReader := &requestInterceptingReader{
			reader:           clientTCP,
			name:             fmt.Sprintf("[%s] Client->Delve", clientAddr),
			requestMethodMap: requestMethodMap,
			mapMutex:         &mapMutex,
			responseReader:   delveReader,
		}

		// Set the reference from response reader to request reader for step over tracking
		delveReader.requestReader = clientReader

		if _, err := io.Copy(delveTCP, clientReader); err != nil {
			// Check if this is a normal connection close vs an actual error
			if isConnectionClosedError(err) {
				log.Printf("Client->Delve connection closed normally for %s", clientAddr)
			} else {
				log.Printf("Error copying client->delve for %s: %v", clientAddr, err)
			}
		}
	}()

	// Handle delve->client with response interception and transparent forwarding
	go func() {
		defer func() {
			log.Printf("Delve->Client goroutine ending for %s", clientAddr)
			done <- struct{}{}
		}()

		log.Println("Delve->Client")

		if _, err := io.Copy(clientTCP, delveReader); err != nil {
			// Check if this is a normal connection close vs an actual error
			if isConnectionClosedError(err) {
				log.Printf("Delve->Client connection closed normally for %s", clientAddr)
			} else {
				log.Printf("Error copying delve->client for %s: %v", clientAddr, err)
			}
		}
	}()

	// Wait for either direction to close with timeout
	select {
	case <-done:
		log.Printf("Connection closed for client %s", clientAddr)
	case <-time.After(30 * time.Minute): // 30 minute timeout
		log.Printf("Connection timeout for client %s, forcing close", clientAddr)
	}
}
