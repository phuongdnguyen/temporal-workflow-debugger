# Developer Guide

This guide provides detailed technical information for developers who want to understand, modify, or contribute to the Temporal Workflow Debugger.

## 📋 Table of Contents

- [Development Setup](#development-setup)
- [Architecture Deep Dive](#architecture-deep-dive)
- [Protocol Implementation](#protocol-implementation)  
- [Core Algorithms](#core-algorithms)
- [Testing](#testing)
- [Contributing](#contributing)
- [Troubleshooting Development Issues](#troubleshooting-development-issues)

## 🛠️ Development Setup

### Prerequisites

- **Go 1.19+** with modules enabled
- **Java 11+** for JetBrains plugin development
- **Gradle** (included in wrapper)
- **Delve debugger** for testing

### Local Development Environment

1. **Clone and build**:
   ```bash
   git clone https://github.com/temporalio/temporal-goland-plugin.git
   cd temporal-goland-plugin
   
   # Build delve wrapper
   cd delve_wrapper
   go build -o delve-wrapper .
   
   # Build plugin
   cd ../jetbrains-plugin
   ./gradlew buildPlugin
   ```

2. **Run in development mode**:
   ```bash
   # Terminal 1: Start delve wrapper with debug logging
   cd delve_wrapper
   go run main.go -verbose
   
   # Terminal 2: Start test workflow with delve
   cd my-wf
   dlv --accept-multiclient --continue --log --headless --listen=127.0.0.1:60000 debug main.go
   ```

3. **IDE setup for development**:
   - Import the project in GoLand/IntelliJ IDEA
   - Configure remote debugger pointing to `:2345`
   - Set breakpoints in `delve_wrapper/main.go`

## 🏗️ Architecture Deep Dive

### Delve Wrapper Implementation

The core proxy implementation uses a **transparent forwarding** pattern with **side-channel processing**:

```go
// Main proxy loop - transparent forwarding
func handleConnection(clientConn, delveConn net.Conn) {
    var wg sync.WaitGroup
    
    // Client -> Delve with request interception
    wg.Add(1)
    go func() {
        defer wg.Done()
        clientReader := &requestInterceptingReader{
            reader:           clientConn,
            delveConnection:  delveConn,
            requestMethodMap: requestMethodMap,
            mapMutex:         &mapMutex,
        }
        io.Copy(delveConn, clientReader)  // ← Transparent forwarding
    }()
    
    // Delve -> Client with response interception  
    wg.Add(1)
    go func() {
        defer wg.Done()
        responseReader := &responseInterceptingReader{
            reader:           delveConn,
            clientConnection: clientConn,
            requestMethodMap: requestMethodMap,
            mapMutex:         &mapMutex,
        }
        io.Copy(clientConn, responseReader)  // ← Transparent forwarding
    }()
    
    wg.Wait()
}
```

### Side-Channel Processing Pattern

The key insight is that we **never modify the original byte stream**. Instead, we copy data for parsing while allowing original bytes to flow through unchanged:

```go
func (rir *requestInterceptingReader) Read(p []byte) (n int, err error) {
    // Read from original source
    n, err = rir.reader.Read(p)
    
    if n > 0 {
        // Copy bytes for side-channel processing
        rir.buffer = append(rir.buffer, p[:n]...)
        
        // Parse JSON-RPC messages from copy
        rir.parseAndModifyRequests()
    }
    
    // Return original bytes unchanged
    return n, err
}
```

This approach ensures:
- **Protocol integrity**: Original byte streams remain unmodified
- **Full interception**: Complete visibility into protocol traffic
- **Compatibility**: Works with any JSON-RPC/DAP client

## 🔌 Protocol Implementation

### JSON-RPC vs DAP Detection

The proxy auto-detects protocol type by examining message headers:

```go
func detectProtocol(data []byte) ProtocolType {
    // DAP uses Content-Length headers
    if strings.HasPrefix(string(data), "Content-Length:") {
        return ProtocolDAP
    }
    
    // JSON-RPC streams objects directly
    if bytes.Contains(data, []byte(`"method"`)) {
        return ProtocolJSONRPC
    }
    
    return ProtocolUnknown
}
```

### JSON-RPC Object Boundary Detection

**Critical Implementation Detail**: JSON-RPC over TCP streams objects without delimiters. We must parse object boundaries correctly:

```go
func extractJSONObject(data []byte) (jsonObj []byte, remaining []byte, found bool) {
    start := bytes.IndexByte(data, '{')
    if start == -1 {
        return nil, data, false
    }
    
    braceCount := 0
    inString := false
    escaped := false
    
    for i := start; i < len(data); i++ {
        char := data[i]
        
        if escaped {
            escaped = false
            continue
        }
        if char == '\\' {
            escaped = true
            continue
        }
        if char == '"' {
            inString = !inString
            continue
        }
        if inString {
            continue // Ignore braces inside strings
        }
        
        if char == '{' {
            braceCount++
        } else if char == '}' {
            braceCount--
            if braceCount == 0 {
                // Found complete JSON object
                return data[start : i+1], data[i+1:], true
            }
        }
    }
    
    return nil, data, false // Incomplete object, need more data
}
```

**Key Challenges Solved**:
- **No line delimiters**: JSON-RPC streams `{"method":"A"}{"method":"B"}` without `\n`
- **Escaped quotes**: Must handle `{"text": "He said \"Hello\""}` correctly
- **Nested objects**: Support complex parameter structures
- **TCP packet boundaries**: Objects can be split across multiple TCP packets

### DAP Message Parsing

DAP messages use HTTP-like headers with content length:

```go
func parseDAP(data []byte) (message []byte, remaining []byte, found bool) {
    // Look for Content-Length header
    headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
    if headerEnd == -1 {
        return nil, data, false
    }
    
    // Extract content length
    contentLength := extractContentLength(data[:headerEnd])
    if contentLength == -1 {
        return nil, data, false
    }
    
    // Check if we have complete message
    messageStart := headerEnd + 4
    messageEnd := messageStart + contentLength
    if messageEnd > len(data) {
        return nil, data, false // Need more data
    }
    
    return data[:messageEnd], data[messageEnd:], true
}
```

## ⚙️ Core Algorithms

### Stack Trace Filtering Algorithm

The stack filtering algorithm removes adapter frames while preserving frame numbering consistency:

```go
func (rir *responseInterceptingReader) filterStacktraceResponse(jsonObj []byte) []byte {
    var stacktraceOut StacktraceOut
    json.Unmarshal(jsonObj, &stacktraceOut)
    
    // Find first workflow frame (my-wf/main.go)
    mainGoFrameIndex := -1
    for i, frame := range stacktraceOut.Locations {
        if strings.Contains(frame.File, "my-wf/main.go") {
            mainGoFrameIndex = i
            break
        }
    }
    
    if mainGoFrameIndex == -1 {
        // No workflow code found, return empty stack
        stacktraceOut.Locations = []Location{}
        return marshallResponse(stacktraceOut)
    }
    
    // Create frame mapping for variable evaluation
    rir.frameMappingLock.Lock()
    rir.frameMapping = make(map[int]int)
    filteredLocations := stacktraceOut.Locations[mainGoFrameIndex:]
    
    for filteredIndex := 0; filteredIndex < len(filteredLocations); filteredIndex++ {
        originalIndex := mainGoFrameIndex + filteredIndex
        rir.frameMapping[filteredIndex] = originalIndex
        log.Printf("🗺️  Frame mapping: filtered[%d] -> original[%d]", 
                   filteredIndex, originalIndex)
    }
    rir.frameMappingLock.Unlock()
    
    // Return filtered stack
    stacktraceOut.Locations = filteredLocations
    return marshallResponse(stacktraceOut)
}
```

### Frame Translation for Variable Evaluation

When the IDE requests variable evaluation, we translate frame numbers:

```go
func (rir *requestInterceptingReader) translateEvalFrameNumber(jsonObj []byte) []byte {
    var req JSONRPCRequest
    json.Unmarshal(jsonObj, &req)
    
    // JSON-RPC params are arrays: [EvalIn]
    var paramsArray []EvalIn
    paramsBytes, _ := json.Marshal(req.Params)
    json.Unmarshal(paramsBytes, &paramsArray)
    evalParams := paramsArray[0]
    
    // Translate frame number
    clientFrame := evalParams.Scope.Frame
    rir.frameMappingLock.RLock()
    originalFrame, exists := rir.frameMapping[clientFrame]
    rir.frameMappingLock.RUnlock()
    
    if exists {
        log.Printf("🔄 FRAME TRANSLATION: filtered frame %d -> original frame %d", 
                   clientFrame, originalFrame)
        evalParams.Scope.Frame = originalFrame
        
        // Re-encode as array
        modifiedParamsArray := []EvalIn{evalParams}
        req.Params = modifiedParamsArray
    }
    
    modifiedBytes, _ := json.Marshal(req)
    return modifiedBytes
}
```

### Auto-stepping Implementation

Auto-stepping automatically continues through adapter code until reaching workflow code:

```go
func (rir *responseInterceptingReader) performDirectAutoStepping(
    responseID int, 
    state *api.DebuggerState,
) (*api.DebuggerState, error) {
    
    const maxAutoSteps = 100
    currentState := state
    
    for step := 0; step < maxAutoSteps; step++ {
        // Check if we're in workflow code
        if !isInAdapterCodeByPath(currentState.CurrentThread.File) {
            log.Printf("🎯 AUTO-STEP: Reached workflow code at %s:%d", 
                       currentState.CurrentThread.File, 
                       currentState.CurrentThread.Line)
            break
        }
        
        log.Printf("🏃 AUTO-STEP: Stepping through adapter code %s:%d", 
                   currentState.CurrentThread.File, 
                   currentState.CurrentThread.Line)
        
        // Continue stepping
        var err error
        currentState, err = rir.delveClient.Next()
        if err != nil {
            log.Printf("❌ AUTO-STEP: Failed to step: %v", err)
            break
        }
    }
    
    // Determine if we should take an extra UX step
    storedMethod, exists := rir.requestMethodMap[responseID]
    shouldTakeExtraStep := false
    
    if exists && strings.HasPrefix(storedMethod, "RPCServer.Command.") {
        commandParts := strings.Split(storedMethod, ".")
        if len(commandParts) >= 3 {
            originalCommand := commandParts[2]
            shouldTakeExtraStep = originalCommand == "next"
        }
    }
    
    // Take extra step for better visual feedback on step-over
    if shouldTakeExtraStep {
        log.Printf("🎯 AUTO-STEP: Taking additional UX step for step-over")
        finalState, err := rir.delveClient.Next()
        if err != nil {
            log.Printf("⚠️  AUTO-STEP: Extra step failed, using current state")
            return currentState, nil
        }
        return finalState, nil
    }
    
    return currentState, nil
}

func isInAdapterCodeByPath(filePath string) bool {
    return strings.Contains(filePath, "go.temporal.io/sdk/") ||
           strings.Contains(filePath, "go.temporal.io/sdk@") ||
           strings.Contains(filePath, "adapters/go/") ||
           strings.Contains(filePath, "replayer.go") ||
           strings.Contains(filePath, "outbound_interceptor.go") ||
           strings.Contains(filePath, "inbound_interceptor.go")
}
```

## 🧪 Testing

### Unit Testing

Test the core algorithms in isolation:

```bash
cd delve_wrapper
go test ./... -v
```

**Key test areas**:
- JSON object boundary detection
- Stack filtering logic
- Frame mapping creation
- Protocol detection

### Integration Testing

Test with real IDEs and workflows:

```bash
# Terminal 1: Start wrapper with test mode
go run main.go -test-mode

# Terminal 2: Start test workflow
cd my-wf
dlv --accept-multiclient --continue --log --headless --listen=127.0.0.1:60000 debug main.go

# Terminal 3: Connect IDE and test debugging operations
```

### Testing Checklist

- [ ] **Breakpoint Setting**: Breakpoints in workflow code are hit correctly
- [ ] **Stack Filtering**: Adapter frames are hidden from stack traces
- [ ] **Variable Evaluation**: Variables can be inspected in filtered frames
- [ ] **Auto-stepping**: Stepping automatically skips through adapter code
- [ ] **Protocol Support**: Both JSON-RPC (GoLand) and DAP (VS Code) work
- [ ] **Frame Translation**: Frame numbers map correctly for variable requests

### Test Scenarios

1. **Basic Debugging Flow**:
   - Set breakpoint in workflow function
   - Start debugging session
   - Verify breakpoint is hit
   - Inspect variables
   - Step through code

2. **Stack Filtering**:
   - Hit breakpoint that occurs in adapter code
   - Verify stack shows workflow code at top
   - Verify adapter frames are hidden

3. **Variable Evaluation**:
   - In filtered stack, hover over workflow variables
   - Verify values are displayed correctly
   - Test complex expressions

4. **Auto-stepping**:
   - Step over function calls that go through adapter
   - Verify stepping automatically skips adapter code
   - Verify control returns to workflow code

## 🤝 Contributing

### Code Organization

```
delve_wrapper/
├── main.go                 # Main proxy entry point
├── request_interceptor.go  # Request processing
├── response_interceptor.go # Response processing  
├── utils.go               # Utility functions
└── types.go               # Type definitions

jetbrains-plugin/
├── src/main/java/com/temporal/wfdebugger/
│   ├── actions/           # IDE actions
│   ├── config/           # Configuration
│   ├── debug/            # Debug integration
│   ├── model/            # Data models
│   ├── service/          # Services
│   └── ui/               # UI components
└── src/main/resources/
    └── META-INF/
        ├── plugin.xml    # Plugin configuration
        └── go-plugin.xml # Go-specific config
```

### Adding Features

1. **New Protocol Support**:
   - Add protocol detection in `detectProtocol()`
   - Implement message parsing
   - Add response formatting
   - Update tests

2. **Enhanced Filtering**:
   - Modify `isInAdapterCodeByPath()` for new patterns
   - Update stack filtering logic
   - Add configuration options
   - Test with new scenarios

3. **IDE Integration**:
   - Add new actions in JetBrains plugin
   - Implement UI components
   - Add configuration options
   - Update plugin manifest

### Code Style

- **Go**: Follow standard Go conventions, use `gofmt`
- **Java**: Follow standard Java conventions, use IntelliJ formatter
- **Comments**: Document complex algorithms and protocol handling
- **Logging**: Use structured logging with appropriate levels
- **Error Handling**: Provide clear error messages and graceful degradation

### Pull Request Process

1. **Fork and branch**: Create feature branch from `main`
2. **Implement changes**: Follow code style and add tests
3. **Test thoroughly**: Run unit and integration tests
4. **Update documentation**: Update relevant docs
5. **Submit PR**: Provide clear description and test evidence

## 🐛 Troubleshooting Development Issues

### Common Development Problems

#### Protocol Parsing Issues

**Symptom**: "Response for unknown method" errors

**Debug**:
```go
// Add debug logging in parseRequests()
log.Printf("Raw buffer: %q", string(rir.buffer))
log.Printf("Extracted JSON: %q", string(jsonObj))
```

**Common causes**:
- Incomplete JSON object extraction
- Incorrect brace counting in string content
- TCP packet boundary issues

#### Stack Filtering Not Working  

**Symptom**: Adapter frames still visible

**Debug**:
```go
// Add logging in filterStacktraceResponse()
for i, frame := range stacktraceOut.Locations {
    log.Printf("Frame %d: %s", i, frame.File)
}
log.Printf("Main workflow frame at index: %d", mainGoFrameIndex)
```

**Common causes**:
- Incorrect workflow file detection patterns
- Missing frame mapping creation
- Response modification not applied

#### Variable Evaluation Fails

**Symptom**: "Could not find symbol value" errors

**Debug**:
```go
// Add logging in translateEvalFrameNumber()
log.Printf("Original request frame: %d", clientFrame)
log.Printf("Translated to frame: %d", originalFrame)
log.Printf("Frame mapping: %+v", rir.frameMapping)
```

**Common causes**:
- Frame mapping not created during stack filtering
- Incorrect frame number translation
- Request modification not applied

### Development Tools

#### Enable Verbose Logging

```bash
cd delve_wrapper
go run main.go -verbose -log-requests -log-responses
```

#### Network Traffic Analysis

```bash
# Monitor network traffic
sudo tcpdump -i lo0 -A port 2345 or port 60000

# Or use netcat for simple testing
nc -l 2345  # Listen for IDE connections
nc localhost 60000  # Connect to Delve
```

#### IDE Debug Configuration

For debugging the debugger itself:

```json
// GoLand run configuration
{
    "name": "Debug Delve Wrapper",
    "type": "go",
    "request": "launch",
    "mode": "debug",
    "program": "${workspaceFolder}/delve_wrapper",
    "args": ["-verbose"]
}
```

## 📚 References

- **[Delve API Documentation](https://github.com/go-delve/delve/tree/master/service/api)** - Delve's JSON-RPC interface
- **[DAP Specification](https://microsoft.github.io/debug-adapter-protocol/)** - Debug Adapter Protocol
- **[JetBrains Plugin Development](https://plugins.jetbrains.com/docs/intellij/welcome.html)** - IntelliJ platform development
- **[Temporal Go SDK](https://github.com/temporalio/sdk-go)** - Understanding workflow execution

## 🔗 Related Documents

- **[Architecture Guide](./architecture.md)** - High-level system design
- **[User Guide](./user-guide.md)** - End-user documentation  
- **[Troubleshooting Guide](./troubleshooting.md)** - Common issues and solutions 