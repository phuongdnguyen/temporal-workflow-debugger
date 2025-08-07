package handlers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/go-delve/delve/service/rpc2"

	delve_jsonrpc "tdlv/pkg/json-rpc-interceptors"
	"tdlv/pkg/utils"
)

// jsonRPCHandler retains all previous logic but uses the buffered reader for the client side.
func jsonRPCHandler(clientTCP net.Conn, br *bufio.Reader) {
	clientAddr := clientTCP.RemoteAddr().String()

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
	delveTCP, err := utils.DialWithRetry("localhost:2345", 3, time.Second)
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

	// Create a response interceptor first so we can reference it
	delveReader := delve_jsonrpc.NewResponseInterceptingReader(delveTCP, fmt.Sprintf("[%s] Delve->Client", clientAddr), requestMethodMap, &mapMutex, clientAddr, delveClient)
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
		delveReader.LogDebuggingSummary()

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
		clientReader := delve_jsonrpc.NewRequestInterceptingReader(br, fmt.Sprintf("[%s] Client->Delve", clientAddr),
			requestMethodMap, &mapMutex, delveReader)

		if _, err := io.Copy(delveTCP, clientReader); err != nil {
			// Check if this is a normal connection close vs an actual error
			if utils.IsConnectionClosedError(err) {
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
			if utils.IsConnectionClosedError(err) {
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
