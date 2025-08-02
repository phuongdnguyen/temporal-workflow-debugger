package handlers

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"custom-debugger/pkg/dap-client"
	dapinterceptors "custom-debugger/pkg/dap-interceptors"
	"custom-debugger/pkg/utils"
)

// dapHandler proxies DAP traffic transparently between the client and Debugger.
func dapHandler(clientTCP net.Conn, br *bufio.Reader) {
	clientAddr := clientTCP.RemoteAddr().String()

	// Dial Debugger
	debuggerTCP, err := utils.DialWithRetry("localhost:2345", 3, time.Second)
	if err != nil {
		log.Printf("Error connecting to Debugger for %s: %v", clientAddr, err)
		_ = clientTCP.Close()
		return
	}
	log.Printf("%s: Connected to Debugger for DAP forwarding", clientAddr)

	// Set keep-alive for Debugger connection too
	if tcpConn, ok := debuggerTCP.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Printf("Error enable keep alive on client connection: %v", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
			log.Printf("Error setting keep alive period on client connection: %v", err)
		}
	}

	// Set timeouts for delve connection as well
	// No read timeout for delve
	if err := debuggerTCP.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("Error setting read deadline on client connection: %v", err)
	}
	// No write timeout for delve
	if err := debuggerTCP.SetWriteDeadline(time.Time{}); err != nil {
		log.Printf("Error setting write deadline on client connection: %v", err)
	}

	// Ensure both connections are closed when function exits
	defer func() {
		log.Printf("Closing connections for client %s", clientAddr)
		if err := clientTCP.Close(); err != nil {
			log.Printf("Error closing client connection: %v", err)
		}

		if err := debuggerTCP.Close(); err != nil {
			log.Printf("Error closing delve connection: %v", err)
		}

		log.Printf("Client %s disconnected", clientAddr)
	}()

	// Channel to signal when one side closes
	done := make(chan struct{}, 2)

	// Create dap client for auto-stepping operations
	debugger := dap_client.NewClient("localhost:2345")
	dapReader := dapinterceptors.NewDAPResponseInterceptingReader(nil, debugger, debuggerTCP,
		fmt.Sprintf("Debugger -> Client %s", clientAddr))

	// goroutine: client -> debugger (use buffered reader to include the peeked byte)
	go func() {
		defer func() {
			log.Printf("Client->Debugger goroutine ending for %s", clientAddr)
			done <- struct{}{}
		}()
		clientReader := dapinterceptors.NewRequestInterceptingReader(br, "Client -> Debugger")
		written, err := io.Copy(debuggerTCP, clientReader)
		if err != nil {
			log.Printf("Error copying request from client to debugger: %v", err)
		}
		log.Printf("%d bytes copied from client to debugger", written)
	}()

	// goroutine: debugger -> client (direct)
	go func() {
		defer func() {
			log.Printf("Delve->Client goroutine ending for %s", clientAddr)
			done <- struct{}{}
		}()
		written, err := io.Copy(clientTCP, dapReader)
		if err != nil {
			log.Printf("Error copying response from debugger to client: %v", err)
		}
		log.Printf("%d bytes copied from debugger to client", written)
	}()

	// Wait for either direction to close with timeout
	select {
	case <-done:
		log.Printf("Connection closed for client %s", clientAddr)
	case <-time.After(30 * time.Minute): // 30 minute timeout
		log.Printf("Connection timeout for client %s, forcing close", clientAddr)
	}
}
