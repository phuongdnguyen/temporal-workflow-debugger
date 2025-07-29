package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpccommon"

	"custom-debugger/pkg/handlers"
	"custom-debugger/pkg/utils"
)

var (
	workingDir string
)

func main() {
	// -----------------------------------------------
	// Command-line flags
	// -----------------------------------------------
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "tdlv is a temporal workflow debugger. This is a wrapper for delve to provide a seamless experience for debugging temporal workflows. (alias: -h)")
	var proxyPort int
	flag.IntVar(&proxyPort, "p", 60000, "port for the tdlv proxy (default 60000)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Tdlv is a temporal workflow debugger. This is a wrapper for delve to provide a seamless experience for debugging temporal workflows. (ports 2345 / 60000)\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	// Enable verbose logging for debugging RPC issues
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Enable Go RPC debug logging via environment variables
	os.Setenv("GODEBUG", "rpclog=1") // Enable RPC debug logging if supported

	// Listen on TCP port for Delve server
	l, err := net.Listen("tcp", ":2345")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			log.Fatal(fmt.Errorf("error closing listener: %w", err))
		}
	}()
	workingDir, err = os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("error getting working directory: %w", err))
	}
	// Setup debugger config for headless mode
	// Foreground: true enables headless mode with automatic protocol detection
	// The server will automatically detect DAP (Content-Length header) vs JSON-RPC
	debuggerConfig := debugger.Config{
		WorkingDir: workingDir,
		Backend:    "default",
		// Set to false to be able to cancel the debugger process when testing
		// TODO: might need to enable it when debugging in Jetbrains IDE
		Foreground:     false, // Enable headless mode
		CheckGoVersion: true,
		// Enable debug logging to see RPC communication issues
		DebugInfoDirectories: []string{},
		DisableASLR:          false,
	}
	debugname, ok := utils.BuildBinary([]string{}, false)
	if !ok {
		log.Fatal("could not build binary")
	}
	// pwd
	log.Println(fmt.Printf("built binary at %s", debugname))

	// Create RPC2 server with headless mode
	// TODO: figure out if we should use headless mode or not in Goland IDE integration
	server := rpccommon.NewServer(&service.Config{
		Listener: l,
		Debugger: debuggerConfig,
		// TODO: figure out why IDE need this set to true
		AcceptMulti: true, // Allow multiple connections and reconnections
		APIVersion:  2,
		ProcessArgs: []string{debugname},
	})

	// Enable additional logging to help debug RPC timeouts
	log.Printf("Delve server configuration: WorkingDir=%s, Backend=%s, Binary=%s",
		debuggerConfig.WorkingDir, debuggerConfig.Backend, debugname)

	// Start Delve server in background
	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("server.Run failed: %v", err)
		}
	}()
	log.Println("Delve headless server started on :2345 (supports both JSON-RPC and DAP, single-client mode)")

	addr := fmt.Sprintf(":%d", proxyPort)
	log.Printf("Starting delve proxy on %s (supports both DAP and JSON-RPC)", addr)
	proxyListener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(fmt.Errorf("could not start proxy listener: %w", err))
	}
	defer proxyListener.Close()

	// Handle shutdown signals only (allow client reconnections)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-ch
		log.Println("Received shutdown signal...")
		log.Println("Shutting down...")
		_ = proxyListener.Close()
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping Delve server: %v", err)
		}
		log.Println("Delve headless server stopped")
		os.Exit(0)
	}()

	// Accept connections and handle them. Allow multiple clients and reconnections.
	for {
		clientTCP, err := proxyListener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle client connection in a goroutine
		// Don't signal shutdown on disconnect - allow reconnections
		go func() {
			handlers.Handle(clientTCP)
			log.Printf("Client connection ended, but server continues running for reconnections")
		}()
	}
}

func init() {
	log.SetOutput(os.Stdout)
}
