# Java Examples for Temporal Workflow Debugger

This directory contains Java examples demonstrating how to use the Temporal workflow debugger replayer adapter.

## Examples Overview

### 1. Simple Workflow Example (`simple-workflow/`)
A basic workflow example that demonstrates:
- Simple activity execution
- Side effects and timers
- Basic workflow patterns
- Standalone debugging mode

**Key Features:**
- Single workflow with multiple activities
- Timer-based delays
- Side effect generation
- Rich event history for debugging

### 2. Structured Workflow Example (`structured-workflow/`)
A complex workflow example showcasing advanced patterns:
- Child workflow execution
- Signal and query methods
- Complex activity orchestration
- State management and error handling

**Key Features:**
- Multi-step user onboarding process
- Child workflow for account setup
- Signal-based preference updates
- Query-based status monitoring
- Comprehensive error handling

## Prerequisites

- **Java**: Version 11 or higher
- **Maven**: Version 3.6 or higher
- **Temporal SDK**: Dependencies included in pom.xml files

## Quick Start

### 1. Choose an Example
Navigate to either `simple-workflow/` or `structured-workflow/` directory.

### 2. Build the Project
```bash
mvn clean compile
```

### 3. Run the Example
```bash
mvn exec:java -Dexec.mainClass="com.temporal.example.SimpleWorkflowMain"
# or
mvn exec:java -Dexec.mainClass="com.temporal.example.StructuredWorkflowMain"
```

## Common Operations

### Building
```bash
mvn clean compile
```

### Testing
```bash
mvn test
```

### Packaging
```bash
mvn package
```

### Running with Make (if available)
```bash
make compile
make run
```

## Configuration

### Setting Breakpoints
Set breakpoints at specific event IDs for debugging:
```java
ReplayerAdapter.setBreakpoints(Arrays.asList(3, 9, 15));
```

### Replay Mode
Choose between standalone and IDE modes:
```java
// Standalone mode (local history file)
ReplayerAdapter.setReplayMode(ReplayMode.STANDALONE);

// IDE mode (debugger integration)
ReplayerAdapter.setReplayMode(ReplayMode.IDE);
```

### Replay Options
Configure replay options using the builder pattern:
```java
ReplayOptions options = new ReplayOptions.Builder()
    .setWorkerOptions(WorkerOptions.getDefaultInstance())
    .setHistoryFilePath("/path/to/history.json")
    .build();
```

## Debugging Features

The Java replayer adapter provides:

- **Event-Level Breakpoints**: Pause execution at specific event IDs
- **Workflow Monitoring**: Track workflow execution progress
- **History Replay**: Replay workflows from history files or IDE sources
- **IDE Integration**: Seamless debugging experience in IDEs
- **Comprehensive Logging**: Detailed execution logs for troubleshooting

## Integration with Workflow Debugger

These examples integrate with the Temporal workflow debugger to provide:
- Breakpoint support at specific event IDs
- IDE integration for debugging workflows
- History replay from both local files and IDE sources
- Interceptor-based workflow execution monitoring

## Troubleshooting

### Common Issues

1. **Missing Dependencies**: Ensure all Temporal SDK dependencies are resolved
2. **History File Path**: Verify the history file path is correct and accessible
3. **Debugger URL**: Check the debugger URL when using IDE mode
4. **Java Version**: Ensure Java 11+ is being used

### Getting Help

- Check the individual example README files for specific guidance
- Review the Temporal SDK documentation at https://github.com/temporalio/temporal
- Examine logs for detailed error information during replay

## Contributing

When adding new examples:
- Follow the existing project structure
- Include comprehensive README documentation
- Provide sample history files for testing
- Ensure proper error handling and logging
- Test with both standalone and IDE modes

## Related Documentation

- [Temporal Java SDK](https://github.com/temporalio/temporal)
- [Workflow Debugger Documentation](../README.md)
- [Replayer Adapter Documentation](../../replayer-adapter-java/README.md)
