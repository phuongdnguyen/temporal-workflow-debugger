# Developer Changelog

## [Unreleased] - 2025-07-17

### Enhanced

#### Auto-Stepping User Experience Improvement

**Feature**: Enhanced auto-stepping to take one additional step forward after returning to workflow code for better visual feedback in GoLand, but only for step-over commands.

**Issue**: When auto-stepping activated from step-over commands, users couldn't visually see that their step-over action had any effect, making the debugging experience confusing.

**Solution**: After auto-stepping through adapter/SDK code and returning to workflow code, the system now takes one additional step forward in the user's workflow code, but only when the original trigger was a step-over command (not continue commands that hit breakpoints).

**Technical Implementation**:
```go
// Determine if this was triggered by step-over vs continue command
originalCommand := extractCommandFromResponse(jsonObj)
shouldTakeExtraStep := originalCommand == "next" // Only for step-over, not continue

// After reaching workflow code, conditionally take additional UX step
if shouldTakeExtraStep {
    log.Printf("%s 🎯 AUTO-STEP: Taking one additional step for better user experience (step-over)", rir.name)
    finalState, err := rir.delveClient.Next()
} else {
    log.Printf("%s 🛑 AUTO-STEP: Skipping extra step (continue command hit breakpoint)", rir.name)
    finalState = state
}
```

**User Experience - Step Over Scenario**:
1. User clicks "Step Over" in GoLand
2. Auto-stepping occurs through adapter code (invisible to user)
3. Returns to workflow code and takes one additional step forward
4. **Cursor visibly moves to next line** - clear visual feedback ✨

**User Experience - Continue Scenario**:
1. User clicks "Continue" in GoLand
2. Hits breakpoint in adapter code, auto-stepping activates
3. Returns to workflow code and **stops at intended location**
4. **Cursor shows breakpoint location** - respects user's continue intent

**Impact**:
- ✅ **Better Visual Feedback**: Users can see their step-over action had an effect
- ✅ **Intuitive Debugging**: Debugger cursor moves forward for step-over, stops appropriately for continue
- ✅ **Seamless Experience**: Auto-stepping + forward progress feels like normal stepping
- ✅ **Respects User Intent**: Continue commands stop at breakpoint location without extra stepping
- ✅ **Error Handling**: Graceful fallback if additional step fails

**Files Changed**:
- `custom-debugger/main.go`: Enhanced `performDirectAutoStepping()` to take additional UX step after reaching workflow code

### Fixed

#### Command Type Detection for Auto-Stepping UX Enhancement

**Issue**: The auto-stepping system was using flawed heuristics to determine whether the original user command was "step over" or "continue", which affected whether to take an extra UX step after returning to workflow code.

**Root Cause**: The `extractCommandFromResponse()` function attempted to guess the original command type by examining the response state:
```go
// Flawed heuristic - unreliable detection
if commandOut.State.CurrentThread.Breakpoint != nil {
    return "continue"  // Assumed continue if breakpoint present
}
return "next"  // Assumed step-over otherwise
```

This heuristic was unreliable because:
- Breakpoint presence in responses doesn't correlate with the original command type
- Continue commands that hit sentinel breakpoints in adapter code would be misidentified as step-over commands
- Led to incorrect UX behavior where continue commands would take unwanted extra steps

**Solution**: Implemented proper command tracking from the request side:

1. **Enhanced Request Tracking**: Modified `requestInterceptingReader` to extract and store the actual command type:
   ```go
   // Extract actual command name from request parameters
   func (rir *requestInterceptingReader) extractCommandNameFromRequest(req JSONRPCRequest) string {
       // Parse JSON-RPC Command params: [{"Name": "next"}] or [{"Name": "continue"}]
       // Return actual command: "next", "continue", "step", "stepout", etc.
   }
   
   // Store specific command type instead of generic "RPCServer.Command"
   if req.Method == "RPCServer.Command" {
       commandName := rir.extractCommandNameFromRequest(req)
       methodToStore = fmt.Sprintf("RPCServer.Command.%s", commandName)
       // Results in: "RPCServer.Command.next" or "RPCServer.Command.continue"
   }
   ```

2. **Reliable Command Type Retrieval**: Updated auto-stepping logic to use stored command type:
   ```go
   // Before (flawed)
   originalCommand := extractCommandFromResponse(jsonObj)
   shouldTakeExtraStep := originalCommand == "next"
   
   // After (reliable)
   storedMethod, exists := rir.requestMethodMap[responseID]
   if exists && strings.HasPrefix(storedMethod, "RPCServer.Command.") {
       commandParts := strings.Split(storedMethod, ".")
       originalCommand = commandParts[2]  // "next" or "continue"
       shouldTakeExtraStep = originalCommand == "next"
   }
   ```

3. **Updated Response Handling**: Modified response processing to handle the new command tracking format.

**Technical Implementation**:
- **Added**: `extractCommandNameFromRequest()` to parse actual command from request parameters
- **Enhanced**: Request method mapping to store `"RPCServer.Command.{commandType}"` instead of generic `"RPCServer.Command"`
- **Removed**: Flawed `extractCommandFromResponse()` function and its unreliable heuristics
- **Updated**: Response handlers to check for `strings.HasPrefix(method, "RPCServer.Command.")` pattern

**Impact**:
- ✅ **Accurate Command Detection**: System now knows the exact original command type from request parameters
- ✅ **Correct UX Behavior**: Step-over commands take extra UX step, continue commands stop at breakpoint location
- ✅ **Eliminated False Positives**: Continue commands hitting sentinel breakpoints no longer misidentified as step-over
- ✅ **Improved Reliability**: No more guessing based on response state - uses authoritative request data

**Files Changed**:
- `custom-debugger/main.go`: Added `extractCommandNameFromRequest()`, enhanced request tracking, removed flawed response heuristics

#### Auto-Stepping Logic & GoLand Crash Prevention

**Issue**: Auto-stepping was incorrectly stopping at Temporal SDK public API code instead of continuing to actual workflow code, and GoLand was crashing when receiving the final auto-stepping response.

**Root Causes**:
1. **Incomplete Adapter Detection**: The `isInAdapterCodeByPath()` function was only treating Temporal SDK `/internal/` code as adapter code, but not other parts of the SDK like `workflow/workflow.go`
2. **Wrong Response Format**: After auto-stepping, the system was sending a new `State` request to Delve instead of returning a proper `Command` response to GoLand
3. **Connection Mismatch**: Writing to `delveConnection` instead of returning response through normal proxy flow
4. **ID Correlation Issues**: GoLand expected a `Command` response with the original ID but was getting async `State` requests

**Technical Fixes**:

1. **Enhanced Adapter Detection Logic**:
   ```go
   // Before (incomplete)
   strings.Contains(filePath, "go.temporal.io/sdk/internal/") ||
   (strings.Contains(filePath, "go.temporal.io/sdk@") && strings.Contains(filePath, "/internal/"))
   
   // After (complete)
   strings.Contains(filePath, "go.temporal.io/sdk/") ||
   strings.Contains(filePath, "go.temporal.io/sdk@")
   ```
   - Now treats **ALL** Temporal SDK code as adapter code, not just internal parts
   - Handles both versioned (`@v1.35.0`) and non-versioned Go module paths
   - Only stops auto-stepping when reaching actual user workflow code (`my-wf/`)

2. **Fixed Response Flow**:
   ```go
   // Before (causing crash)
   finalStateCommand := map[string]interface{}{
       "method": "RPCServer.State",
       "params": []interface{}{},
       "id": responseID,
   }
   rir.delveConnection.Write(finalStateBytes) // Wrong connection
   return nil
   
   // After (working)
   finalCommandResponse := map[string]interface{}{
       "id": responseID,
       "result": map[string]interface{}{
           "State": state, // Embed state in Command response
       },
   }
   return modifiedBuffer // Return directly to GoLand
   ```

3. **State Tracking**:
   - Added `lastState` tracking throughout auto-stepping loop
   - Ensures final response always has valid debugger state
   - Handles both successful completion and max-steps timeout scenarios

**Impact**:
- ✅ Auto-stepping now correctly continues through **all** Temporal SDK code until reaching user workflow code
- ✅ GoLand no longer crashes when receiving auto-stepping responses  
- ✅ Debugger properly shows final location in `my-wf/main.go` after transparent auto-stepping
- ✅ Maintains proper JSON-RPC request/response correlation for IDE compatibility

**Testing**:
- Verified with Temporal SDK v1.35.0 with versioned Go module paths
- Confirmed auto-stepping traverses: `adapters/go/replayer.go` → `outbound_interceptor.go` → `go.temporal.io/sdk@v1.35.0/internal/workflow.go` → `go.temporal.io/sdk@v1.35.0/workflow/workflow.go` → `my-wf/main.go`
- GoLand debugging session remains stable throughout auto-stepping process

**Files Changed**:
- `custom-debugger/main.go`: Enhanced `isInAdapterCodeByPath()` logic and fixed `performDirectAutoStepping()` response handling 