# Java Simple Workflow Example

This example demonstrates how to use the Temporal workflow debugger replayer adapter with Java workflows.

## Overview

The example includes:
- A simple workflow that executes activities, creates side effects, and uses timers
- Activity implementations for workflow tasks
- Main class showing how to use the replayer adapter for debugging
- Support for both standalone mode (local history files) and IDE mode

## Project Structure

```
src/main/java/com/temporal/example/
├── SimpleWorkflowMain.java      # Main class with replayer setup
├── SimpleWorkflow.java          # Workflow interface
├── SimpleWorkflowImpl.java      # Workflow implementation
├── SimpleActivity.java          # Activity interface
└── SimpleActivityImpl.java      # Activity implementation
```

## Prerequisites

- Java 11 or higher
- Maven 3.6 or higher
- Temporal SDK dependencies (included in pom.xml)

## Building the Project

```bash
mvn clean compile
```

## Running the Example

### Standalone Mode (Local History File)

1. Place a `history.json` file in the project root directory
2. Run the main class:

```bash
mvn exec:java -Dexec.mainClass="com.temporal.example.SimpleWorkflowMain"
```

### IDE Mode

1. Set the environment variable for the debugger URL:
   ```bash
   export TEMPORAL_DEBUGGER_PLUGIN_URL=http://localhost:54578
   ```

2. Run the main class with IDE mode:
   ```java
   ReplayerAdapter.setReplayMode(ReplayMode.IDE);
   ```

## Configuration

### Setting Breakpoints

Set breakpoints at specific event IDs:

```java
ReplayerAdapter.setBreakpoints(Arrays.asList(3, 9, 15));
```

### Replay Options

Configure replay options using the builder pattern:

```java
ReplayOptions options = new ReplayOptions.Builder()
    .setWorkerOptions(WorkerOptions.getDefaultInstance())
    .setHistoryFilePath("/path/to/history.json")
    .build();
```

## Workflow Behavior

The example workflow:
1. Executes a greeting activity 3 times
2. Creates side effects after each activity
3. Sleeps between iterations to create timer events
4. Adds a final 5-second timer

This creates a rich event history suitable for debugging and replay testing.

## Integration with Workflow Debugger

The replayer adapter integrates with the Temporal workflow debugger to provide:
- Breakpoint support at specific event IDs
- IDE integration for debugging workflows
- History replay from both local files and IDE sources
- Interceptor-based workflow execution monitoring

## Troubleshooting

- Ensure all Temporal SDK dependencies are properly resolved
- Check that the history file path is correct and accessible
- Verify the debugger URL is correct when using IDE mode
- Check logs for detailed error information during replay
