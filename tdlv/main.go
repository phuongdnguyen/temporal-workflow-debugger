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

// Java Debug configuration
const (
	jdtlsVersion           = "1.40.0"
	jdtlsURL               = "https://download.eclipse.org/jdtls/snapshots/jdt-language-server-latest.tar.gz"
	jdtlsDir               = ".jdt-language-server"
	vscodeJavaDebugVersion = "0.53.1"
	vscodeJavaDebugURL     = "https://github.com/microsoft/java-debug/archive/refs/tags/" + vscodeJavaDebugVersion + ".tar.gz"
	vscodeJavaDebugDir     = ".vscode-java-debug"
)

func main() {
	// -----------------------------------------------
	// Command-line flags
	// -----------------------------------------------
	var showHelp bool
	flag.BoolVar(&showHelp, "help", false, "Tdlv (Temporal delve) is a temporal workflow debugger, provide ability to focus on user workflow code in debug sessions (alias: -h)")

	var proxyPort int
	flag.IntVar(&proxyPort, "p", 60000, "port for remote debugging")

	var lang string
	flag.StringVar(&lang, "lang", "", "[required] language to use for the workflow, available options: [go, python, js, java]")

	var install bool
	flag.BoolVar(&install, "install", false, "auto-install missing language debuggers")

	var start bool
	flag.BoolVar(&start, "start", false, "start debugger")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Tdlv (Temporal delve) is a temporal workflow debugger\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	requireFlags("lang")

	if showHelp {
		flag.Usage()
		return
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	debuggerStopCh := make(chan struct{}, 1)

	// Check and handle dependencies based on language
	switch lang {
	case "go":
		if err := ensureDelveAvailable(install); err != nil {
			log.Printf("Delve dependency issue: %v", err)
			printDelveInstallationGuidance()
			os.Exit(1)
		}
		if start {
			startDelve(debuggerStopCh)
		}
	case "python":
		if err := ensureDebugPyAvailable(install); err != nil {
			log.Printf("DebugPy dependency issue: %v", err)
			printDebugPyInstallationGuidance()
			os.Exit(1)
		}
		if start {
			startDebugPy(debuggerStopCh)
		}
	case "js":
		if err := ensureJsDebugAvailable(install); err != nil {
			log.Printf("JS Debug dependency issue: %v", err)
			printJsDebugInstallationGuidance()
			os.Exit(1)
		}
		if start {
			startJsDebug(debuggerStopCh)
		}
	case "java":
		if err := ensureJavaDebugAvailable(install); err != nil {
			log.Printf("Java Debug dependency issue: %v", err)
			printJavaDebugInstallationGuidance()
			os.Exit(1)
		}
		if start {
			startJavaDebug(debuggerStopCh)
		}
	default:
		log.Fatalf("Unknown lang: %s", lang)
	}

	if !start {
		return
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
		<-debuggerStopCh
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

// ensureDelveAvailable checks if delve is available and optionally installs it
func ensureDelveAvailable(autoInstall bool) error {
	// First check if dlv is available in PATH
	if _, err := exec.LookPath("dlv"); err == nil {
		log.Println("Found delve (dlv) in PATH")
		return nil
	}

	// Check if delve is available as go-delve/delve
	if _, err := exec.LookPath("go-delve"); err == nil {
		log.Println("Found delve (go-delve) in PATH")
		return nil
	}

	// Delve not found
	if !autoInstall {
		return fmt.Errorf("delve debugger not found in PATH")
	}

	fmt.Print("Delve debugger not found, will install it. ")
	return installDelve()
}

// installDelve installs delve using go install
func installDelve() error {
	log.Println("Installing delve...")

	// Check if Go is available
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go is required to install delve, but 'go' command not found in PATH")
	}

	// Install delve
	cmd := exec.Command(goPath, "install", "github.com/go-delve/delve/cmd/dlv@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install delve: %w", err)
	}

	log.Println("Delve installed successfully")
	return nil
}

// ensureDebugPyAvailable checks if debugpy is available
func ensureDebugPyAvailable(autoInstall bool) error {
	// Check if python and debugpy are available
	cmd := exec.Command("python", "-c", "import debugpy")
	if err := cmd.Run(); err == nil {
		log.Println("Found debugpy for Python")
		return nil
	}

	if !autoInstall {
		return fmt.Errorf("debugpy not found for Python")
	}

	// Auto-install debugpy
	fmt.Print("DebugPy not found, will install it")
	log.Println("Installing debugpy...")

	cmd = exec.Command("python", "-m", "pip", "install", "debugpy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install debugpy: %w", err)
	}

	log.Println("DebugPy installed successfully")
	return nil
}

// ensureJsDebugAvailable checks if Node.js and js-debug-dap are available
func ensureJsDebugAvailable(autoInstall bool) error {
	// Check if node is available
	if _, err := exec.LookPath("node"); err != nil {
		return fmt.Errorf("node not found in PATH (required for JavaScript/Typescript debugging)")
	}

	// Check if js-debug-dap is already set up
	jsDebugPath := getJsDebugPath()
	dapServerPath := filepath.Join(jsDebugPath, "js-debug", "src", "dapDebugServer.js")

	if _, err := os.Stat(dapServerPath); err == nil {
		log.Printf("Found js-debug-dap at %s", jsDebugPath)
		return nil
	}

	// js-debug-dap not found
	if !autoInstall {
		return fmt.Errorf("js-debug-dap not found (required for JavaScript debugging)")
	}

	fmt.Print("JS Debug adapter not found. Will download and install it. ")
	return setupJsDebug()
}

// ensureJavaDebugAvailable checks if Java, Eclipse JDT LS, and vscode-java-debug are available
func ensureJavaDebugAvailable(autoInstall bool) error {
	// Check if java is available
	if _, err := exec.LookPath("java"); err != nil {
		return fmt.Errorf("java not found in PATH (required for Java debugging)")
	}

	// Check Java version (needs Java 21+)
	cmd := exec.Command("java", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check Java version: %w", err)
	}

	outputStr := string(output)
	log.Printf("Java version check: %s", strings.TrimSpace(outputStr))

	// Check if Eclipse JDT LS is already set up
	jdtlsPath := getJdtlsPath()
	launcherPath := filepath.Join(jdtlsPath, "plugins", "org.eclipse.equinox.launcher_*.jar")

	matches, err := filepath.Glob(launcherPath)
	if err == nil && len(matches) > 0 {
		log.Printf("Found Eclipse JDT LS at %s", jdtlsPath)
	} else {
		if !autoInstall {
			return fmt.Errorf("eclipse JDT Language Server not found (required for Java debugging)")
		}
		log.Println("Eclipse JDT LS not found, will install it")
		if err := setupJdtls(); err != nil {
			return fmt.Errorf("failed to setup Eclipse JDT LS: %w", err)
		}
	}

	// Check if vscode-java-debug is already built
	debugPluginPath := getVscodeJavaDebugPluginPath()
	if _, err := os.Stat(debugPluginPath); err == nil {
		log.Printf("Found vscode-java-debug plugin at %s", debugPluginPath)
		return nil
	}

	// vscode-java-debug not found
	if !autoInstall {
		return fmt.Errorf("vscode-java-debug not found (required for Java debugging)")
	}

	log.Println("vscode-java-debug not found, will build it")
	return setupVscodeJavaDebug()
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

// getJdtlsPath returns the path where Eclipse JDT LS should be installed
func getJdtlsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "tdlv-cache", jdtlsDir)
	}
	return filepath.Join(homeDir, ".cache", "tdlv", jdtlsDir)
}

// getVscodeJavaDebugPath returns the path where vscode-java-debug should be installed
func getVscodeJavaDebugPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "tdlv-cache", vscodeJavaDebugDir)
	}
	return filepath.Join(homeDir, ".cache", "tdlv", vscodeJavaDebugDir)
}

// getVscodeJavaDebugPluginPath returns the path to the built debug plugin JAR
func getVscodeJavaDebugPluginPath() string {
	basePath := getVscodeJavaDebugPath()
	return filepath.Join(basePath, "java-debug-"+vscodeJavaDebugVersion,
		"com.microsoft.java.debug.plugin", "target", "com.microsoft.java.debug.plugin-"+vscodeJavaDebugVersion+".jar")
}

// setupJsDebug downloads and sets up the js-debug-dap package
func setupJsDebug() error {
	jsDebugPath := getJsDebugPath()

	log.Printf("Setting up js-debug-dap in %s", jsDebugPath)

	// Create cache directory
	if err := os.MkdirAll(jsDebugPath, 0755); err != nil {
		return fmt.Errorf("failed to create js-debug directory: %w", err)
	}

	// Download the package
	tarGzPath := filepath.Join(jsDebugPath, "js-debug-dap.tar.gz")
	if err := downloadJsDebug(jsDebugURL, tarGzPath); err != nil {
		return fmt.Errorf("failed to download js-debug-dap: %w", err)
	}

	// Extract the package
	if err := extractTarGz(tarGzPath, jsDebugPath); err != nil {
		return fmt.Errorf("failed to extract js-debug-dap: %w", err)
	}

	// Clean up the tar.gz file
	os.Remove(tarGzPath)

	// Verify the installation
	dapServerPath := filepath.Join(jsDebugPath, "js-debug", "src", "dapDebugServer.js")
	if _, err := os.Stat(dapServerPath); err != nil {
		return fmt.Errorf("js-debug-dap installation verification failed: dapDebugServer.js not found in path %s", dapServerPath)
	}

	log.Println("js-debug-dap installed successfully")
	return nil
}

// downloadJsDebug downloads the js-debug-dap package with progress indication
func downloadJsDebug(url, filepath string) error {
	log.Printf("Downloading js-debug-dap from %s", url)

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

	// Copy with progress indication

	// Simple progress indication
	contentLength := resp.ContentLength
	if contentLength > 0 {
		log.Printf("Downloading %d bytes...", contentLength)
	}

	// Use a simple counter for progress
	counter := &progressCounter{total: contentLength}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))

	fmt.Println() // New line after progress

	return err
}

// progressCounter tracks download progress
type progressCounter struct {
	downloaded int64
	total      int64
	lastPrint  time.Time
}

func (pc *progressCounter) Write(p []byte) (int, error) {
	n := len(p)
	pc.downloaded += int64(n)

	// Print progress every second
	if time.Since(pc.lastPrint) > time.Second {
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
func extractTarGz(src, dest string) error {
	log.Printf("Extracting %s to %s", src, dest)

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
			log.Printf("Skipping file type %c: %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

// setupJdtls downloads and sets up Eclipse JDT Language Server
func setupJdtls() error {
	jdtlsPath := getJdtlsPath()

	log.Printf("Setting up Eclipse JDT LS in %s", jdtlsPath)

	// Create cache directory
	if err := os.MkdirAll(jdtlsPath, 0755); err != nil {
		return fmt.Errorf("failed to create JDT LS directory: %w", err)
	}

	// Download JDT LS
	tarGzPath := filepath.Join(jdtlsPath, "jdt-language-server.tar.gz")
	if err := downloadFile(jdtlsURL, tarGzPath); err != nil {
		return fmt.Errorf("failed to download JDT LS: %w", err)
	}

	// Extract JDT LS
	if err := extractTarGz(tarGzPath, jdtlsPath); err != nil {
		return fmt.Errorf("failed to extract JDT LS: %w", err)
	}

	// Clean up tar.gz
	os.Remove(tarGzPath)

	log.Println("Eclipse JDT LS installed successfully")
	return nil
}

// setupVscodeJavaDebug downloads and builds vscode-java-debug
func setupVscodeJavaDebug() error {
	vscodeJavaDebugPath := getVscodeJavaDebugPath()

	log.Printf("Setting up vscode-java-debug in %s", vscodeJavaDebugPath)

	// Create directory
	if err := os.MkdirAll(vscodeJavaDebugPath, 0755); err != nil {
		return fmt.Errorf("failed to create vscode-java-debug directory: %w", err)
	}

	// Download vscode-java-debug
	tarGzPath := filepath.Join(vscodeJavaDebugPath, "java-debug.tar.gz")
	if err := downloadFile(vscodeJavaDebugURL, tarGzPath); err != nil {
		return fmt.Errorf("failed to download vscode-java-debug: %w", err)
	}

	// Extract
	if err := extractTarGz(tarGzPath, vscodeJavaDebugPath); err != nil {
		return fmt.Errorf("failed to extract vscode-java-debug: %w", err)
	}

	// Clean up tar.gz
	os.Remove(tarGzPath)

	// Build the plugin
	buildDir := filepath.Join(vscodeJavaDebugPath, "java-debug-"+vscodeJavaDebugVersion)
	if err := buildJavaDebugPlugin(buildDir); err != nil {
		return fmt.Errorf("failed to build vscode-java-debug: %w", err)
	}

	log.Println("vscode-java-debug built successfully")
	return nil
}

// buildJavaDebugPlugin builds the vscode-java-debug plugin using Maven
func buildJavaDebugPlugin(buildDir string) error {
	log.Println("Building vscode-java-debug plugin...")

	// Check if mvn or mvnw is available
	var cmd *exec.Cmd
	mvnwPath := filepath.Join(buildDir, "mvnw")
	if _, err := os.Stat(mvnwPath); err == nil {
		cmd = exec.Command(mvnwPath, "clean", "install", "-DskipTests")
	} else if _, err := exec.LookPath("mvn"); err == nil {
		cmd = exec.Command("mvn", "clean", "install", "-DskipTests")
	} else {
		return fmt.Errorf("neither mvnw nor mvn found - Maven is required to build vscode-java-debug")
	}

	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("maven build failed: %w", err)
	}

	return nil
}

// downloadFile downloads a file from URL to filepath with progress indication
func downloadFile(url, filepath string) error {
	log.Printf("Downloading from %s", url)

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

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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

// printJavaDebugInstallationGuidance provides helpful installation instructions
func printJavaDebugInstallationGuidance() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Java Debug Setup Required")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\nTo debug Java applications, you need:")
	fmt.Println("1. Java 21 or later")
	fmt.Println("2. Eclipse JDT Language Server")
	fmt.Println("3. vscode-java-debug plugin")
	fmt.Println("4. Maven (for building the debug plugin)")

	fmt.Println("\nPrerequisites:")
	fmt.Println("1. Install Java 21+:")
	fmt.Println("   Visit: https://adoptium.net/")

	switch runtime.GOOS {
	case "darwin":
		fmt.Println("   Or: brew install openjdk@21")
	case "linux":
		fmt.Println("   Or: apt-get install openjdk-21-jdk  # Ubuntu/Debian")
		fmt.Println("       dnf install java-21-openjdk     # Fedora")
	case "windows":
		fmt.Println("   Or: choco install openjdk21         # Chocolatey")
		fmt.Println("       scoop install openjdk21         # Scoop")
	}

	fmt.Println("\n2. Install Maven:")
	fmt.Println("   Visit: https://maven.apache.org/install.html")

	switch runtime.GOOS {
	case "darwin":
		fmt.Println("   Or: brew install maven")
	case "linux":
		fmt.Println("   Or: apt-get install maven           # Ubuntu/Debian")
		fmt.Println("       dnf install maven               # Fedora")
	case "windows":
		fmt.Println("   Or: choco install maven             # Chocolatey")
		fmt.Println("       scoop install maven             # Scoop")
	}

	fmt.Println("\nAutomatic setup:")
	fmt.Printf("   %s --lang=java --install\n", os.Args[0])
	fmt.Println("\nManual setup:")
	fmt.Printf("   1. Download Eclipse JDT LS: %s\n", jdtlsURL)
	fmt.Printf("   2. Extract to: %s\n", getJdtlsPath())
	fmt.Printf("   3. Download vscode-java-debug: %s\n", vscodeJavaDebugURL)
	fmt.Printf("   4. Extract and build with Maven\n")
	fmt.Println("\nFor more details: https://github.com/microsoft/java-debug")
}

func startDelve(stopCh chan struct{}) {
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
			log.Fatal(fmt.Errorf("error closing delve net listener: %w", err))
		}
		stopCh <- struct{}{}
	}()
}

func startDebugPy(stopCh chan struct{}) {
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
		stopCh <- struct{}{}
	}()
}

func startJsDebug(stopCh chan struct{}) {
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
		stopCh <- struct{}{}
	}()
}

func startJavaDebug(stopCh chan struct{}) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(fmt.Errorf("error getting working directory: %w", err))
	}

	jdtlsPath := getJdtlsPath()
	debugPluginPath := getVscodeJavaDebugPluginPath()

	// Find the launcher JAR
	launcherPattern := filepath.Join(jdtlsPath, "plugins", "org.eclipse.equinox.launcher_*.jar")
	matches, err := filepath.Glob(launcherPattern)
	if err != nil || len(matches) == 0 {
		log.Fatal("Eclipse JDT LS launcher not found")
	}
	launcherJar := matches[0]

	// Determine config directory based on OS
	var configDir string
	switch runtime.GOOS {
	case "linux":
		configDir = filepath.Join(jdtlsPath, "config_linux")
	case "darwin":
		configDir = filepath.Join(jdtlsPath, "config_mac")
	case "windows":
		configDir = filepath.Join(jdtlsPath, "config_win")
	default:
		configDir = filepath.Join(jdtlsPath, "config_linux")
	}

	// Create workspace directory
	workspaceDir := filepath.Join(os.TempDir(), "tdlv-java-workspace")
	os.MkdirAll(workspaceDir, 0755)

	ctx := context.Background()

	// Start Eclipse JDT LS
	cmd := exec.CommandContext(ctx, "java",
		"-Declipse.application=org.eclipse.jdt.ls.core.id1",
		"-Dosgi.bundles.defaultStartLevel=4",
		"-Declipse.product=org.eclipse.jdt.ls.core.product",
		"-jar", launcherJar,
		"-configuration", configDir,
		"-data", workspaceDir,
		"--add-modules=ALL-SYSTEM",
		"--add-opens", "java.base/java.util=ALL-UNNAMED",
		"--add-opens", "java.base/java.lang=ALL-UNNAMED")

	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment for debug plugin
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("JAVA_DEBUG_PLUGIN=%s", debugPluginPath))

	go func() {
		log.Println("Starting Eclipse JDT LS with vscode-java-debug plugin on :2345")
		if err := cmd.Run(); err != nil {
			log.Printf("Error running Eclipse JDT LS: %v", err)
		}
	}()

	go func() {
		<-stopCh
		if err := cmd.Process.Kill(); err != nil {
			log.Printf("Error killing Eclipse JDT LS: %v", err)
		}
		log.Println("Java debugger stopped")
		stopCh <- struct{}{}
	}()
}

func requireFlags(names ...string) {
	provided := map[string]bool{}
	flag.CommandLine.Visit(func(f *flag.Flag) { provided[f.Name] = true })

	var missing []string
	for _, n := range names {
		if !provided[n] {
			missing = append(missing, "-"+n)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing required flags: %s\n\n", strings.Join(missing, ", "))
		flag.Usage()
		os.Exit(2)
	}
}

func init() {
	log.SetOutput(os.Stdout)
}
