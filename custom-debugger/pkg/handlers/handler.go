package handlers

import (
	"bufio"
	"log"
	"net"
)

func Handle(clientTCP net.Conn) {
	clientAddr := clientTCP.RemoteAddr().String()
	log.Printf("New client connected from %s", clientAddr)

	// Wrap the connection in a buffered reader so we can peek without consuming
	br := bufio.NewReader(clientTCP)

	// Peek the first byte to detect protocol (blocking until at least 1 byte)
	firstByte, err := br.Peek(1)
	if err != nil {
		log.Printf("%s: failed to peek first byte: %v", clientAddr, err)
		_ = clientTCP.Close()
		return
	}

	// If the first byte is 'C' (Content-Length header for DAP) treat as DAP, otherwise JSON-RPC
	if len(firstByte) > 0 && firstByte[0] == 'C' {
		log.Printf("%s: Detected DAP protocol", clientAddr)
		dapHandler(clientTCP, br)
	} else {
		log.Printf("%s: Detected JSON-RPC protocol", clientAddr)
		jsonRPCHandler(clientTCP, br)
	}
}
