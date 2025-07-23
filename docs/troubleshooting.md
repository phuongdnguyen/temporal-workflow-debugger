# Troubleshooting Guide

This guide provides solutions to common issues when using the Temporal Workflow Debugger.

## 📋 Table of Contents

- [Connection Issues](#connection-issues)
- [Breakpoint Problems](#breakpoint-problems)
- [Variable Evaluation Issues](#variable-evaluation-issues)
- [Stack Trace Problems](#stack-trace-problems)
- [IDE-Specific Issues](#ide-specific-issues)
- [Performance Issues](#performance-issues)
- [Advanced Debugging](#advanced-debugging)

## 🔌 Connection Issues

### IDE Can't Connect to Debugger

**Symptoms**:
- "Could not connect to 127.0.0.1:2345" error
- Connection refused messages
- IDE hangs on "Connecting to debugger"

**Solutions**:

1. **Verify delve wrapper is running**:
   ```bash
   cd custom-debugger
   ./delve-wrapper
   # Should show: "Delve proxy listening on :2345"
   ```

2. **Check port availability**:
   ```bash
   netstat -an | grep 2345
   # Should show: tcp4  0  0  *.2345  *.*  LISTEN
   ```

3. **Verify Delve is listening**:
   ```bash
   dlv --accept-multiclient --continue --log --headless --listen=127.0.0.1:60000 debug main.go
   # Should show: "API server listening at: 127.0.0.1:60000"
   ```

4. **Check firewall settings**:
   ```bash
   # macOS: Allow connections on ports 2345 and 60000
   # Windows: Add firewall exceptions for these ports
   ```

5. **Test with telnet**:
   ```bash
   telnet 127.0.0.1 2345
   # Should connect successfully
   ```

### Proxy Connection Drops

**Symptoms**:
- Debugging session starts but drops after a few operations
- "Connection lost" errors during debugging

**Solutions**:

1. **Check proxy logs**:
   ```bash
   cd custom-debugger
   go run main.go -verbose
   # Look for error messages or connection issues
   ```

2. **Verify network stability**:
   ```bash
   ping 127.0.0.1
   # Should have consistent response times
   ```

3. **Restart in proper order**:
   ```bash
   # 1. Stop all debugging sessions
   # 2. Stop delve wrapper
   # 3. Stop delve server
   # 4. Start delve server
   # 5. Start delve wrapper
   # 6. Connect IDE
   ```

## 🎯 Breakpoint Problems

### Breakpoints Not Being Hit

**Symptoms**:
- Breakpoints set but never triggered
- Code execution doesn't pause at breakpoints
- Breakpoint indicators show as "unverified"

**Diagnosis**:

1. **Verify workflow is executing**:
   ```bash
   # Check Temporal Web UI at http://localhost:8080
   # Ensure workflow is actually running
   ```

2. **Check breakpoint location**:
   ```go
   // ✅ Good: Set in workflow function
   func SimpleWorkflow(ctx workflow.Context, input string) (string, error) {
       // Breakpoint here will work
       logger := workflow.GetLogger(ctx)
       
   // ❌ Bad: Set in SDK internal code
   func (w *workflowEnvironmentImpl) ExecuteActivity(...) {
       // Breakpoints here won't work as expected
   ```

**Solutions**:

1. **Ensure breakpoints are in workflow code**:
   - Set breakpoints in your workflow functions (`my-wf/` directory)
   - Avoid setting breakpoints in SDK code (`go.temporal.io/sdk/`)

2. **Verify code path is reached**:
   ```go
   func SimpleWorkflow(ctx workflow.Context, input string) (string, error) {
       // Add logging to verify execution
       logger := workflow.GetLogger(ctx)
       logger.Info("Workflow started", "input", input)  // ← Set breakpoint here
       
       // Your workflow logic
       return "result", nil
   }
   ```

3. **Check file paths match**:
   - Ensure the file being debugged matches the compiled binary
   - Rebuild your workflow if code has changed

4. **Restart debugging session**:
   ```bash
   # Stop and restart the entire debugging stack
   pkill dlv
   # Restart delve wrapper
   # Restart delve server
   # Reconnect IDE
   ```

### Breakpoints in Wrong Location

**Symptoms**:
- Breakpoints hit in adapter code instead of workflow code
- Execution pauses in `replayer.go` or SDK internal files

**Solutions**:

1. **Let auto-stepping work**:
   - The debugger automatically steps through adapter code
   - Wait for it to return to your workflow code
   - Don't manually step when in adapter code

2. **Check auto-stepping is enabled**:
   ```bash
   # Verify proxy logs show auto-stepping
   cd custom-debugger
   go run main.go -verbose
   # Look for: "🏃 AUTO-STEP: Stepping through adapter code"
   ```

3. **Move breakpoints to workflow functions**:
   ```go
   // ❌ Don't set breakpoints here
   func (r *Replayer) ExecuteWorkflow(...) {
   
   // ✅ Set breakpoints here instead
   func MyWorkflow(ctx workflow.Context, input Input) (Output, error) {
       // Your workflow logic
   }
   ```

## 🔍 Variable Evaluation Issues

### Variables Show "Undefined" or "Not Available"

**Symptoms**:
- Hovering over variables shows "undefined"
- Watch expressions fail with "could not find symbol value"
- Local variables panel is empty

**Diagnosis**:

1. **Check current stack frame**:
   - Ensure you're inspecting variables in the correct frame
   - Click on your workflow function in the call stack

2. **Verify variable scope**:
   ```go
   func MyWorkflow(ctx workflow.Context, input string) (string, error) {
       localVar := "test"  // ← Should be visible
       
       if input != "" {
           scopedVar := "scoped"  // ← Only visible inside if block
           // Try to inspect variables here
       }
       
       return localVar, nil
   }
   ```

**Solutions**:

1. **Ensure you're in workflow code frame**:
   - Click on your workflow function in the call stack
   - Avoid inspecting variables while in adapter frames

2. **Try simpler expressions first**:
   ```go
   // ✅ Start with simple variables
   input   // Should work
   ctx     // Should work
   
   // ✅ Then try more complex expressions
   input.Field     // Should work
   len(input)      // Should work
   ```

3. **Check frame translation is working**:
   ```bash
   # Look for frame translation logs
   cd custom-debugger
   go run main.go -verbose
   # Should see: "🔄 FRAME TRANSLATION: filtered frame 0 -> original frame 3"
   ```

4. **Restart debugging session**:
   - Frame mapping might be corrupted
   - Restart to reinitialize frame translation

### Complex Expression Evaluation Fails

**Symptoms**:
- Simple variables work but complex expressions fail
- Method calls or field access don't work

**Solutions**:

1. **Break down complex expressions**:
   ```go
   // ❌ Instead of this complex expression
   user.Profile.Settings.Theme
   
   // ✅ Try step by step
   user           // Does this work?
   user.Profile   // Does this work?
   user.Profile.Settings  // Does this work?
   ```

2. **Check for nil values**:
   ```go
   // Variables might be nil at breakpoint time
   if user != nil {
       // Inspect user.Profile here
   }
   ```

3. **Use type assertions carefully**:
   ```go
   // ✅ Safe type assertion
   if str, ok := value.(string); ok {
       // Inspect str here
   }
   ```

## 📚 Stack Trace Problems

### Seeing Adapter Frames in Call Stack

**Symptoms**:
- Call stack shows `replayer.go`, `outbound_interceptor.go`, etc.
- Stack frames include `go.temporal.io/sdk/` files

**Solutions**:

1. **Verify stack filtering is enabled**:
   ```bash
   # Check proxy logs for filtering activity
   cd custom-debugger
   go run main.go -verbose
   # Should see: "✂️ Filtering out X frames before my-wf/main.go"
   ```

2. **Check if workflow code is detected**:
   ```bash
   # Look for workflow detection logs
   # Should see: "✅ FILTERING: FOUND my-wf/main.go at frame X"
   ```

3. **Ensure your workflow is in the right directory**:
   - Stack filtering looks for `my-wf/main.go`
   - If your workflow is elsewhere, update filtering rules

4. **Update to latest version**:
   ```bash
   git pull origin main
   cd custom-debugger
   go build -o delve-wrapper .
   ```

### Empty Call Stack

**Symptoms**:
- Call stack panel shows no frames
- "No stack frames available" message

**Solutions**:

1. **Check if breakpoint is hit**:
   - Verify workflow is actually executing
   - Add logging to confirm code paths

2. **Disable stack filtering temporarily**:
   ```bash
   # Modify custom-debugger to skip filtering for debugging
   # Look for filterStacktraceResponse() function
   ```

3. **Verify Delve connection**:
   ```bash
   # Test direct connection to Delve
   nc localhost 60000
   # Should connect successfully
   ```

## 🎮 IDE-Specific Issues

### GoLand/IntelliJ IDEA Issues

**Problem**: Debug configuration not working

**Solutions**:
1. **Verify remote debug configuration**:
   - Go to Run → Edit Configurations
   - Create new "Go Remote" configuration
   - Host: `127.0.0.1`, Port: `2345`

2. **Check plugin installation**:
   ```bash
   cd jetbrains-plugin
   ./gradlew buildPlugin
   # Install from build/distributions/
   ```

3. **Clear IDE caches**:
   - File → Invalidate Caches and Restart

**Problem**: Stepping behaves strangely

**Solutions**:
1. **Use Step Over instead of Step Into**:
   - Step Over (F8) works better with auto-stepping
   - Step Into might interfere with frame filtering

2. **Wait for auto-stepping to complete**:
   - Don't click rapidly during auto-stepping
   - Let the debugger return to workflow code

### VS Code Issues

**Problem**: Debug Adapter Protocol errors

**Solutions**:
1. **Check launch.json configuration**:
   ```json
   {
       "name": "Temporal Workflow Debug",
       "type": "go",
       "request": "attach",
       "mode": "remote",
       "remotePath": "${workspaceFolder}",
       "port": 2345,
       "host": "127.0.0.1"
   }
   ```

2. **Verify Go extension is installed**:
   - Install the official Go extension
   - Reload VS Code window

3. **Check DAP protocol handling**:
   ```bash
   # Look for DAP-specific logs
   cd custom-debugger
   go run main.go -verbose
   # Should see: "📥 DAP STACKTRACE REQUEST"
   ```

## ⚡ Performance Issues

### Slow Debugging Response

**Symptoms**:
- Long delays when setting breakpoints
- Slow variable evaluation
- Sluggish stepping

**Solutions**:

1. **Reduce logging verbosity**:
   ```bash
   # Run without verbose logging
   cd custom-debugger
   ./delve-wrapper
   # Instead of: go run main.go -verbose
   ```

2. **Limit concurrent debug sessions**:
   - Only run one debugging session at a time
   - Close unused debugging connections

3. **Check system resources**:
   ```bash
   # Monitor CPU and memory usage
   top -p $(pgrep dlv)
   top -p $(pgrep delve-wrapper)
   ```

4. **Optimize workflow complexity**:
   - Avoid setting too many breakpoints
   - Reduce complex variable watch expressions

### Memory Usage Growing

**Symptoms**:
- Memory usage increases over time
- System becomes sluggish during long debug sessions

**Solutions**:

1. **Restart debugging session periodically**:
   ```bash
   # Stop and restart every few hours
   pkill dlv
   pkill delve-wrapper
   # Restart components
   ```

2. **Monitor for memory leaks**:
   ```bash
   # Check for growing processes
   ps aux | grep -E "(dlv|delve-wrapper)" | sort -k6 -nr
   ```

3. **Clear frame mapping cache**:
   - Frame mappings accumulate over time
   - Restart proxy to clear cache

## 🔬 Advanced Debugging

### Debug the Debugger

If you're having persistent issues, you can debug the debugger itself:

1. **Enable comprehensive logging**:
   ```bash
   cd custom-debugger
   go run main.go -verbose -log-requests -log-responses > debug.log 2>&1
   ```

2. **Analyze network traffic**:
   ```bash
   # Monitor all traffic between components
   sudo tcpdump -i lo0 -A port 2345 or port 60000
   ```

3. **Run proxy with Go debugger**:
   ```bash
   # Debug the proxy itself
   dlv debug main.go -l 127.0.0.1:9999
   # Connect another IDE to port 9999
   ```

### Environment-Specific Issues

**Docker/Container environments**:
1. **Expose correct ports**:
   ```dockerfile
   EXPOSE 2345 60000
   ```

2. **Use host networking**:
   ```bash
   docker run --network host your-image
   ```

**CI/CD environments**:
1. **Disable GUI features**:
   ```bash
   # Run in headless mode only
   dlv --headless --listen=:60000
   ```

2. **Use non-interactive debugging**:
   ```bash
   # Automated testing without IDE
   ```

### Logging Analysis

**Understanding proxy logs**:

```bash
# Connection established
"Delve proxy listening on :2345"

# Protocol detection
"📥 DAP STACKTRACE REQUEST #8"
"Client->Delve RPC Method: RPCServer.Stacktrace (ID: 5)"

# Stack filtering
"✅ FILTERING: FOUND my-wf/main.go at frame 13!"
"✂️ Filtering out 13 frames before my-wf/main.go"

# Frame translation
"🔄 FRAME TRANSLATION: filtered frame 0 -> original frame 3"

# Auto-stepping
"🏃 AUTO-STEP: Stepping through adapter code"
"🎯 AUTO-STEP: Reached workflow code"
```

**Red flags in logs**:
```bash
# Bad signs
"Response for unknown method"           # Protocol parsing issues
"Failed to parse JSON-RPC response"     # Message corruption
"Frame mapping not found"               # Translation problems
"AUTO-STEP: Failed to step"            # Auto-stepping issues
```

## 🆘 Getting Help

If you've tried these solutions and still have issues:

1. **Gather diagnostic information**:
   ```bash
   # System info
   go version
   dlv version
   uname -a
   
   # Save proxy logs
   cd custom-debugger
   go run main.go -verbose > debug.log 2>&1
   
   # Network status
   netstat -an | grep -E "(2345|60000)"
   ```

2. **Create minimal reproduction**:
   - Use the provided `my-wf` example
   - Document exact steps to reproduce the issue

3. **Report the issue**:
   - [GitHub Issues](https://github.com/temporalio/temporal-goland-plugin/issues)
   - Include diagnostic information and reproduction steps
   - Specify your IDE, OS, and Go version

4. **Community resources**:
   - [Temporal Community](https://community.temporal.io/)
   - [Temporal Slack](https://temporalio.slack.com/)

## 🔗 Related Documents

- **[User Guide](./user-guide.md)** - Complete setup and usage instructions
- **[Architecture Guide](./architecture.md)** - Understanding how it works
- **[Developer Guide](./developer-guide.md)** - Technical implementation details 