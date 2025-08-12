package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpccommon"

	"tdlv/pkg/handlers"
	"tdlv/pkg/utils"
)

// JS Debug configuration
const (
	jsDebugVersion = "v1.102.0"
	jsDebugURL     = "https://github.com/microsoft/vscode-js-debug/releases/download/" + jsDebugVersion + "/js-debug-dap-" + jsDebugVersion + ".tar.gz"
	jsDebugDir     = ".js-debug-dap"
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
	var pythonPath string
	flag.StringVar(&pythonPath, "python", "python", "path to the python executable")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Tdlv is a temporal workflow debugger\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	var install bool
	flag.BoolVar(&install, "install", false, "auto-install missing language debuggers (requires user confirmation)")
	var quiet bool
	flag.BoolVar(&quiet, "quiet", false, "suppress dependency check messages (for IDE integration)")

	var entryPoint string
	flag.StringVar(&entryPoint, "entrypoint", "", "launch entry point for the debugger")

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	debuggerStopCh := make(chan struct{}, 1)

	// Check and handle dependencies based on language
	switch lang {
	case "go":
		if err := ensureDelveAvailable(install, quiet); err != nil {
			if !quiet {
				log.Printf("Delve dependency issue: %v", err)
				printDelveInstallationGuidance()
			}
			os.Exit(1)
		}
		startDelve(debuggerStopCh)
	case "python":
		if err := ensureDebugPyAvailable(install, quiet); err != nil {
			if !quiet {
				log.Printf("DebugPy dependency issue: %v", err)
				printDebugPyInstallationGuidance()
			}
			os.Exit(1)
		}
		startDebugPy(debuggerStopCh)
	case "js":
		if err := ensureJsDebugAvailable(install, quiet); err != nil {
			if !quiet {
				log.Printf("JS Debug dependency issue: %v", err)
				printJsDebugInstallationGuidance()
			}
			os.Exit(1)
		}
		startJsDebug(debuggerStopCh)
	default:
		if !quiet {
			log.Printf("Running with lang %s, expect a language debugger to be started on port 2345", lang)
		}
	}

	addr := fmt.Sprintf(":%d", proxyPort)
	if !quiet {
		log.Printf("Starting tdlv on %s", addr)
	}
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
		if !quiet {
			log.Println("Received shutdown signal...")
			log.Println("Shutting down...")
		}
		_ = proxyListener.Close()
		debuggerStopCh <- struct{}{}
		os.Exit(0)
	}()

	// Accept connections and handle them. Allow multiple clients and reconnections.
	for {
		clientTCP, err := proxyListener.Accept()
		if err != nil {
			if !quiet {
				log.Printf("Error accepting connection: %v", err)
			}
			continue
		}

		// Handle client connection in a goroutine
		// Don't signal shutdown on disconnect - allow reconnections
		go func() {
			handlers.Handle(clientTCP)
			if !quiet {
				log.Printf("Client connection ended, but server continues running for reconnections")
			}
		}()
	}
}

// ensureDelveAvailable checks if delve is available and optionally installs it
func ensureDelveAvailable(autoInstall, quiet bool) error {
	// First check if dlv is available in PATH
	if _, err := exec.LookPath("dlv"); err == nil {
		if !quiet {
			log.Println("Found delve (dlv) in PATH")
		}
		return nil
	}

	// Check if delve is available as go-delve/delve
	if _, err := exec.LookPath("go-delve"); err == nil {
		if !quiet {
			log.Println("Found delve (go-delve) in PATH")
		}
		return nil
	}

	// Delve not found
	if !autoInstall {
		return fmt.Errorf("delve debugger not found in PATH")
	}

	// Auto-install with user confirmation (unless in quiet mode)
	if !quiet {
		fmt.Print("Delve debugger not found. Install it automatically? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("delve installation declined by user")
		}
	}

	return installDelve(quiet)
}

// installDelve installs delve using go install
func installDelve(quiet bool) error {
	if !quiet {
		log.Println("Installing delve...")
	}

	// Check if Go is available
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("Go is required to install delve, but 'go' command not found in PATH")
	}

	// Install delve
	cmd := exec.Command(goPath, "install", "github.com/go-delve/delve/cmd/dlv@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install delve: %w", err)
	}

	if !quiet {
		log.Println("Delve installed successfully")
	}
	return nil
}

// ensureDebugPyAvailable checks if debugpy is available
func ensureDebugPyAvailable(autoInstall, quiet bool) error {
	// Check if python and debugpy are available
	cmd := exec.Command("python", "-c", "import debugpy")
	if err := cmd.Run(); err == nil {
		if !quiet {
			log.Println("Found debugpy for Python")
		}
		return nil
	}

	if !autoInstall {
		return fmt.Errorf("debugpy not found for Python")
	}

	// Auto-install debugpy
	if !quiet {
		fmt.Print("DebugPy not found. Install it automatically? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("debugpy installation declined by user")
		}
		log.Println("Installing debugpy...")
	}

	cmd = exec.Command("python", "-m", "pip", "install", "debugpy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install debugpy: %w", err)
	}

	if !quiet {
		log.Println("DebugPy installed successfully")
	}
	return nil
}

// ensureJsDebugAvailable checks if Node.js and js-debug-dap are available
func ensureJsDebugAvailable(autoInstall, quiet bool) error {
	// Check if node is available
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("Node.js not found in PATH (required for JavaScript debugging)")
	}

	// Check if js-debug-dap is already set up
	jsDebugPath := getJsDebugPath()
	dapServerPath := filepath.Join(jsDebugPath, "js-debug", "src", "dapDebugServer.js")

	if _, err := os.Stat(dapServerPath); err == nil {
		if !quiet {
			log.Printf("Found js-debug-dap at %s", jsDebugPath)
		}
		return nil
	}

	// js-debug-dap not found
	if !autoInstall {
		return fmt.Errorf("js-debug-dap not found (required for JavaScript debugging)")
	}

	// Auto-install with user confirmation (unless in quiet mode)
	if !quiet {
		fmt.Print("JS Debug adapter not found. Download and install it automatically? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("js-debug-dap installation declined by user")
		}
	}

	return setupJsDebug(quiet)
}

// getJsDebugPath returns the path where js-debug-dap should be installed
func getJsDebugPath() string {
	// Use a cache directory in the user's home directory or system temp
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "tdlv-cache", jsDebugDir)
	}
	return filepath.Join(homeDir, ".cache", "tdlv", jsDebugDir)
}

// setupJsDebug downloads and sets up the js-debug-dap package
func setupJsDebug(quiet bool) error {
	jsDebugPath := getJsDebugPath()

	if !quiet {
		log.Printf("Setting up js-debug-dap in %s", jsDebugPath)
	}

	// Create cache directory
	if err := os.MkdirAll(jsDebugPath, 0755); err != nil {
		return fmt.Errorf("failed to create js-debug directory: %w", err)
	}

	// Download the package
	tarGzPath := filepath.Join(jsDebugPath, "js-debug-dap.tar.gz")
	if err := downloadJsDebug(jsDebugURL, tarGzPath, quiet); err != nil {
		return fmt.Errorf("failed to download js-debug-dap: %w", err)
	}

	// Extract the package
	if err := extractTarGz(tarGzPath, jsDebugPath, quiet); err != nil {
		return fmt.Errorf("failed to extract js-debug-dap: %w", err)
	}

	// Clean up the tar.gz file
	os.Remove(tarGzPath)

	// Verify the installation
	dapServerPath := filepath.Join(jsDebugPath, "out", "src", "dapDebugServer.js")
	if _, err := os.Stat(dapServerPath); err != nil {
		return fmt.Errorf("js-debug-dap installation verification failed: dapDebugServer.js not found")
	}

	if !quiet {
		log.Println("js-debug-dap installed successfully")
	}
	return nil
}

// downloadJsDebug downloads the js-debug-dap package with progress indication
func downloadJsDebug(url, filepath string, quiet bool) error {
	if !quiet {
		log.Printf("Downloading js-debug-dap from %s", url)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "tdlv-debugger/1.0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy with progress indication (if not quiet)
	if quiet {
		_, err = io.Copy(out, resp.Body)
	} else {
		// Simple progress indication
		contentLength := resp.ContentLength
		if contentLength > 0 {
			log.Printf("Downloading %d bytes...", contentLength)
		}

		// Use a simple counter for progress
		counter := &progressCounter{total: contentLength, quiet: quiet}
		_, err = io.Copy(out, io.TeeReader(resp.Body, counter))

		if !quiet {
			fmt.Println() // New line after progress
		}
	}

	return err
}

// progressCounter tracks download progress
type progressCounter struct {
	downloaded int64
	total      int64
	quiet      bool
	lastPrint  time.Time
}

func (pc *progressCounter) Write(p []byte) (int, error) {
	n := len(p)
	pc.downloaded += int64(n)

	// Print progress every second (if not quiet)
	if !pc.quiet && time.Since(pc.lastPrint) > time.Second {
		if pc.total > 0 {
			progress := float64(pc.downloaded) / float64(pc.total) * 100
			fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", progress, pc.downloaded, pc.total)
		} else {
			fmt.Printf("\rDownloaded: %d bytes", pc.downloaded)
		}
		pc.lastPrint = time.Now()
	}

	return n, nil
}

// extractTarGz extracts a tar.gz file to the specified directory
func extractTarGz(src, dest string, quiet bool) error {
	if !quiet {
		log.Printf("Extracting %s to %s", src, dest)
	}

	// Open the tar.gz file
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Sanitize the path to prevent directory traversal
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Create file
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			outFile.Close()

		default:
			// Skip other file types (symlinks, etc.)
			if !quiet {
				log.Printf("Skipping file type %c: %s", header.Typeflag, header.Name)
			}
		}
	}

	return nil
}

// printDelveInstallationGuidance provides helpful installation instructions
func printDelveInstallationGuidance() {
	fmt.Println("\n=== Delve Installation Guide ===")
	fmt.Println("Delve is required for Go workflow debugging.")
	fmt.Println("\nInstallation options:")
	fmt.Println("1. Using Go (recommended):")
	fmt.Println("   go install github.com/go-delve/delve/cmd/dlv@latest")
	fmt.Println("\n2. Using package managers:")

	switch runtime.GOOS {
	case "darwin":
		fmt.Println("   brew install delve")
	case "linux":
		fmt.Println("   # Ubuntu/Debian: apt-get install delve")
		fmt.Println("   # Fedora: dnf install delve")
		fmt.Println("   # Arch: pacman -S delve")
	case "windows":
		fmt.Println("   # Using Chocolatey: choco install delve")
		fmt.Println("   # Using Scoop: scoop install delve")
	}

	fmt.Println("\n3. Auto-install next time:")
	fmt.Printf("   %s --lang=go --install\n", os.Args[0])
	fmt.Println("\nFor more details: https://github.com/go-delve/delve/tree/master/Documentation/installation")
}

// printDebugPyInstallationGuidance provides helpful installation instructions
func printDebugPyInstallationGuidance() {
	fmt.Println("\n=== DebugPy Installation Guide ===")
	fmt.Println("DebugPy is required for Python workflow debugging.")
	fmt.Println("\nInstallation:")
	fmt.Println("   python -m pip install debugpy")
	fmt.Println("\nOr auto-install next time:")
	fmt.Printf("   %s --lang=python --install\n", os.Args[0])
}

// printJsDebugInstallationGuidance provides helpful installation instructions
func printJsDebugInstallationGuidance() {
	fmt.Println("\n=== JavaScript Debug Setup Guide ===")
	fmt.Println("JavaScript debugging requires Node.js and the VS Code JS Debug adapter.")
	fmt.Println("\nPrerequisites:")
	fmt.Println("1. Install Node.js:")
	fmt.Println("   Visit: https://nodejs.org/")

	switch runtime.GOOS {
	case "darwin":
		fmt.Println("   Or: brew install node")
	case "linux":
		fmt.Println("   Or: apt-get install nodejs npm  # Ubuntu/Debian")
		fmt.Println("       dnf install nodejs npm      # Fedora")
	case "windows":
		fmt.Println("   Or: choco install nodejs        # Chocolatey")
		fmt.Println("       scoop install nodejs        # Scoop")
	}

	fmt.Println("\n2. Auto-install JS Debug adapter:")
	fmt.Printf("   %s --lang=js --install\n", os.Args[0])
	fmt.Println("\n3. Manual setup (alternative):")
	fmt.Printf("   Download: %s\n", jsDebugURL)
	fmt.Printf("   Extract to: %s\n", getJsDebugPath())
	fmt.Println("\nFor more details: https://github.com/microsoft/vscode-js-debug")
}

func startDelve(stopCh <-chan struct{}) {
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
		<-stopCh
		if err := server.Stop(); err != nil {
			log.Printf("Error stopping Delve server: %v", err)
		}
		log.Println("Delve headless server stopped")
		if err := l.Close(); err != nil {
			log.Fatal(fmt.Errorf("error closing listener: %w", err))
		}
	}()
}

func startDebugPy(stopCh <-chan struct{}) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("error getting working directory: %w", err))
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "python", "-m", "debugpy.adapter", "--port", "2345")
	cmd.Dir = workingDir // Set working directory to the Python example
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

func startJsDebug(stopCh <-chan struct{}) {
	jsDebugPath := getJsDebugPath()
	dapServerPath := filepath.Join(jsDebugPath, "js-debug", "src", "dapDebugServer.js")

	// Verify the dapDebugServer.js exists
	if _, err := os.Stat(dapServerPath); err != nil {
		log.Fatalf("dapDebugServer.js not found at %s. Run with --install flag to set up js-debug-dap", dapServerPath)
	}

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "node", dapServerPath, "2345", "127.0.0.1")
	cmd.Dir = jsDebugPath // Set working directory to js-debug path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		log.Printf("Starting JS debug server on :2345 using %s", dapServerPath)
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
