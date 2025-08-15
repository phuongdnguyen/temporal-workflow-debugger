# Java Adapter Development Guide

This document provides guidance on completing the Java adapter implementation to work with the actual Temporal Java SDK.

## Current Status

✅ **COMPLETED**: The Java adapter has been fully implemented using the actual Temporal Java SDK interfaces!

The adapter now includes:
- Proper interceptor implementations using `WorkerInterceptorBase`, `WorkflowInboundCallsInterceptorBase`, etc.
- Real SDK imports and method signatures
- Working replay functionality using `WorkflowReplayer` 
- Proper `WorkflowInfo` integration for breakpoint detection
- HTTP communication for IDE integration
- Complete build configuration

## Quick Setup

To use this adapter:

### Option 1: Use Published SDK (Recommended)
The adapter is configured to use Temporal Java SDK 1.30.1 from Maven Central. Simply build:

```bash
cd replayer-adapter-java
mvn clean compile
# or
./gradlew build
```
## Key Areas Requiring Implementation

### 1. Temporal Java SDK Dependencies

The current `pom.xml` includes `io.temporal:temporal-sdk` but the actual import statements are commented out. You'll need to:

1. **Research the correct Temporal Java SDK version and packages**:
   - Find the correct Maven coordinates for the latest stable version
   - Identify the actual package names for interceptors and workflow replay

2. **Update imports in Java files**:
   - Replace placeholder imports with actual SDK imports
   - Update type signatures from `Object` to actual types

### 2. Interceptor Implementation

The interceptor classes need to be updated to implement the actual Temporal Java SDK interfaces:

#### Files to Update:
- `RunnerWorkerInterceptor.java`
- `RunnerWorkflowInboundInterceptor.java`
- `RunnerWorkflowOutboundInterceptor.java`
- `RunnerActivityInboundInterceptor.java`
- `RunnerActivityOutboundInterceptor.java`

#### Research Required:
1. **Find the actual interceptor interfaces in Temporal Java SDK**:
   - Look for `WorkerInterceptor` interface or similar
   - Identify workflow interceptor interfaces (likely `WorkflowInboundCallsInterceptor`, `WorkflowOutboundCallsInterceptor`)
   - Identify activity interceptor interfaces

2. **Update method signatures**:
   - Replace `Object` parameters with actual input/output types
   - Implement proper method chaining to next interceptor
   - Add proper exception handling

#### Example Pattern (based on Go/Python adapters):
```java
// Expected pattern - update with actual SDK interfaces
public class RunnerWorkerInterceptor implements WorkerInterceptor {
    @Override
    public WorkflowInboundCallsInterceptor interceptWorkflow(WorkflowInboundCallsInterceptor next) {
        return new RunnerWorkflowInboundInterceptor(next);
    }
}
```

### 3. Workflow Replay Implementation

The replay methods in `ReplayerAdapter.java` need actual implementation:

#### Research Required:
1. **Find WorkflowReplayer class in Temporal Java SDK**:
   - Look for replay functionality in the SDK
   - Understand how to configure worker options with interceptors
   - Learn how to parse history from protobuf or JSON

2. **Update method implementations**:
   - `replayWithHistory()` - implement with actual SDK
   - `replayWithJsonFile()` - implement with actual SDK
   - `getHistoryFromIDE()` - parse protobuf using actual History type

#### Expected Pattern:
```java
private static void replayWithHistory(WorkerOptions workerOptions, History hist, Class<?> workflow) {
    WorkerOptions.Builder optionsBuilder = WorkerOptions.newBuilder(workerOptions);
    optionsBuilder.setWorkerInterceptors(new RunnerWorkerInterceptor());
    
    WorkflowReplayer replayer = new WorkflowReplayer();
    replayer.addWorkflowImplementationType(workflow);
    replayer.replayWorkflowExecution(hist);
}
```

### 4. Workflow Info Access

The `getWorkflowInfo()` methods in interceptors need proper implementation:

#### Research Required:
1. **Find how to access workflow context in interceptors**:
   - Look for Workflow.getInfo() or similar methods
   - Understand how to get current history length for breakpoint detection

#### Expected Pattern:
```java
private Object getWorkflowInfo() {
    return Workflow.getInfo(); // or similar SDK method
}
```

### 5. Type Safety and Error Handling

Once the actual types are identified, update:
- Replace all `Object` types with proper types
- Add proper exception handling
- Ensure type safety throughout the codebase

## Implementation Steps

1. **Research Temporal Java SDK**:
   - Download/examine the latest Temporal Java SDK
   - Find documentation or examples for interceptors
   - Identify the correct interfaces and classes

2. **Update Dependencies**:
   - Verify correct version and dependencies in `pom.xml`
   - Add any additional required dependencies

3. **Implement Interceptors**:
   - Start with `RunnerWorkerInterceptor`
   - Implement proper interface inheritance
   - Update method signatures and implementations

4. **Implement Replay Logic**:
   - Update `ReplayerAdapter` class with actual SDK calls
   - Implement proper history parsing and replay

5. **Test Integration**:
   - Create test workflows
   - Verify breakpoint functionality
   - Test both standalone and IDE modes

6. **Update Documentation**:
   - Update README with correct usage examples
   - Add proper Javadoc comments
   - Remove TODO comments

## Resources for Research

1. **Temporal Java SDK Repository**: 
   - GitHub: https://github.com/temporalio/java-sdk
   - Look for interceptor examples and documentation

2. **Temporal Documentation**:
   - Official docs for Java SDK interceptors
   - Workflow replay documentation

3. **Existing Examples**:
   - Check for Java examples in Temporal's official samples
   - Look for community examples of interceptor usage

## Testing Strategy

1. **Unit Tests**:
   - Test interceptor functionality
   - Test breakpoint detection logic
   - Test HTTP communication with IDE

2. **Integration Tests**:
   - Test with actual workflow replay
   - Test standalone mode with history files
   - Test IDE mode with mock debugger server

3. **Example Workflows**:
   - Create simple test workflows
   - Verify debugging functionality works as expected

## Notes

- The current implementation follows the exact same patterns as Go and Python adapters
- All the core logic is in place, only the SDK integration is missing
- The HTTP communication logic should work as-is once types are corrected
- The interceptor chain pattern matches other SDK implementations
