package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpccommon"

	"custom-debugger/pkg/handlers"
	"custom-debugger/pkg/utils"
)

func main() {
	// -----------------------------------------------
	// Command-line flags
	// -----------------------------------------------
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "tdlv is a temporal workflow debugger, provide ability to debug temporal workflow from history file from workflows in multiple programming languages (alias: -h)")
	var proxyPort int
	flag.IntVar(&proxyPort, "p", 60000, "port for tdlv")
	var lang string
	flag.StringVar(&lang, "lang", "go", "language to use for the workflow, available options: [go, python, js]")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Tdlv is a temporal workflow debugger, (ports 2345 / 60000)\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	var install bool
	flag.BoolVar(&install, "install", false, "install required language debugger if missing")

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	debuggerStopCh := make(chan struct{}, 1)
	switch lang {
	case "go":
		startDelve(debuggerStopCh, install)
	case "python":
		startDebugPy(debuggerStopCh, install)
	case "js":
		startJsDebug(debuggerStopCh, install)
	default:
		log.Printf("Running with lang %s, expect a language debugger to be started on port 2345", lang)
	}

	addr := fmt.Sprintf(":%d", proxyPort)
	log.Printf("Starting tdlv on %s", addr)
	proxyListener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(fmt.Errorf("could not start tdlv: %w", err))
	}
	defer proxyListener.Close()

	// Handle shutdown signals only
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-ch
		log.Println("Received shutdown signal...")
		log.Println("Shutting down...")
		_ = proxyListener.Close()
		debuggerStopCh <- struct{}{}
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

func startDelve(stopCh <-chan struct{}, install bool) {
	if install {
		// Install delve
	}
	// Listen on TCP port for Delve server
	l, err := net.Listen("tcp", ":2345")
	if err != nil {
		log.Fatal(err)
	}
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("error getting working directory: %w", err))
	}
	// Setup debugger config for headless mode
	// Foreground: true enables headless mode with automatic protocol detection
	// The server will automatically detect DAP (Content-Length header) vs JSON-RPC
	debuggerConfig := debugger.Config{
		WorkingDir:           workingDir,
		Backend:              "default",
		Foreground:           false,
		CheckGoVersion:       true,
		DebugInfoDirectories: []string{},
		DisableASLR:          false,
	}
	bin, ok := utils.BuildBinary([]string{}, false)
	if !ok {
		log.Fatal("could not build binary")
	}
	// pwd
	log.Println(fmt.Printf("built binary at %s", bin))

	// Create RPC2 server
	server := rpccommon.NewServer(&service.Config{
		Listener: l,
		Debugger: debuggerConfig,
		// TODO: figure out why IDE need this set to true
		AcceptMulti: true, // Allow multiple connections and reconnections
		APIVersion:  2,
		ProcessArgs: []string{bin},
	})

	// Enable additional logging to help debug RPC timeouts
	log.Printf("Delve server configuration: WorkingDir=%s, Backend=%s, Binary=%s",
		debuggerConfig.WorkingDir, debuggerConfig.Backend, bin)

	// Start Delve server in background
	go func() {
		if err := server.Run(); err != nil {
			log.Fatalf("run delve server failed: %v", err)
		}
	}()
	log.Println("Delve headless server started on :2345 (supports both JSON-RPC and DAP)")
	go func() {
		select {
		case <-stopCh:
			if err := server.Stop(); err != nil {
				log.Printf("Error stopping Delve server: %v", err)
			}
			log.Println("Delve headless server stopped")
			if err := l.Close(); err != nil {
				log.Fatal(fmt.Errorf("error closing listener: %w", err))
			}
		}
	}()
}

func startDebugPy(stopCh <-chan struct{}, install bool) {
	if install {
		// Install debugpy
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "python", "-m", "debugpy", "--listen", "2345", "--wait-for-client", "standalone_replay.py")
	cmd.Dir = "example/python" // Set working directory to the Python example
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go func() {
		log.Println("Starting Python debugpy server on :2345")
		if err := cmd.Run(); err != nil {
			log.Printf("Error running Python debugpy: %v", err)
		}
	}()
	go func() {
		<-stopCh
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Error killing Python debugpy: %v", err)
		}
		log.Println("Python debugger stopped")
	}()
}

func startJsDebug(stopCh <-chan struct{}, install bool) {
	if install {
		// Install js-debug
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "node", "dapDebugServer.js", "2345", "127.0.0.1")
	go func() {
		log.Println("Starting JS debug on :2345")
		if err := cmd.Run(); err != nil {
			log.Printf("Error running JS Debug: %v", err)
		}
	}()
	go func() {
		<-stopCh
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Error killing JS debug: %v", err)
		}
		log.Println("JS debug stopped")
	}()

}

func init() {
	log.SetOutput(os.Stdout)
}
