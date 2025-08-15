# Temporal Java Replayer Adapter

A Java adapter for debugging Temporal workflows by replaying execution history with breakpoint support.

## Installation

### Maven

```xml
<dependency>
    <groupId>com.temporal</groupId>
    <artifactId>replayer-adapter-java</artifactId>
    <version>0.1.0</version>
</dependency>
```

### Gradle

```gradle
implementation 'com.temporal:replayer-adapter-java:0.1.0'
```

## Overview

This adapter provides workflow replay functionality for Temporal Java SDK applications, enabling debugging through:

- **Standalone Mode**: Replay workflows using local history files
- **IDE Mode**: Replay workflows with debugger UI integration

## Features

- Workflow history replay with breakpoint support
- Interceptor-based debugging hooks
- Support for both standalone and IDE-integrated debugging
- Activity and workflow execution tracking

## Usage

```java
import io.temporal.worker.WorkerOptions;
import com.temporal.replayer.ReplayerAdapter;
import com.temporal.replayer.ReplayMode;
import com.temporal.replayer.ReplayOptions;

// Set replay mode
ReplayerAdapter.setReplayMode(ReplayMode.IDE);

// Configure replay options
ReplayOptions opts = new ReplayOptions.Builder()
    .setHistoryFilePath("/path/to/history.json") // for standalone mode
    .build();

// Replay workflow
ReplayerAdapter.replay(opts, YourWorkflow.class);
```

### Standalone Mode Example

```java
// Set breakpoints at specific event IDs
ReplayerAdapter.setBreakpoints(Arrays.asList(1, 5, 10));

// Set standalone mode
ReplayerAdapter.setReplayMode(ReplayMode.STANDALONE);

// Configure with history file
ReplayOptions opts = new ReplayOptions.Builder()
    .setHistoryFilePath("/path/to/your/workflow-history.json")
    .build();

// Replay workflow
ReplayerAdapter.replay(opts, YourWorkflowInterface.class);
```

### IDE Mode Example

```java
// Set IDE mode
ReplayerAdapter.setReplayMode(ReplayMode.IDE);

// Configure replay options (history will be fetched from IDE)
ReplayOptions opts = new ReplayOptions.Builder()
    .build();

// Replay workflow
ReplayerAdapter.replay(opts, YourWorkflowInterface.class);
```

## Implementation Status

✅ **COMPLETED**: This Java adapter is now fully implemented using the actual Temporal Java SDK!

### Current Status:
- ✅ Complete adapter structure following Go/Python patterns
- ✅ All core classes and interfaces implemented using real Temporal Java SDK
- ✅ HTTP communication logic for IDE integration
- ✅ Breakpoint management system
- ✅ Build configuration (Maven & Gradle)
- ✅ Example usage and documentation
- ✅ Full Temporal Java SDK integration with proper interceptors
- ✅ Working workflow replay functionality
- ✅ Real WorkflowInfo integration for breakpoint detection

### Ready to Use:
The adapter is ready for use with Temporal Java SDK 1.30.1. See `DEVELOPMENT.md` for build instructions and usage examples.

## Dependencies

- `io.temporal:temporal-sdk` 1.30.1
- `io.temporal:temporal-testing` 1.30.1
- `com.fasterxml.jackson.core:jackson-databind` 2.17.0
- `org.apache.httpcomponents.client5:httpclient5` 5.3.1

## Architecture

The adapter uses Temporal SDK interceptors to hook into workflow execution:

- **Workflow Interceptors**: Track workflow execution entry points and operations
- **Activity Interceptors**: Monitor activity execution
- **Breakpoint Management**: Support for setting and checking breakpoints during replay
- **HTTP Client**: Communication with IDE debugger interface

## Environment Variables

- `TEMPORAL_DEBUGGER_PLUGIN_URL`: URL for IDE debugger communication (default: http://localhost:54578)

## Examples

See `example/java/` directory for complete working examples.

## Contributing

Contributions welcome! Please see the main repository for guidelines.

## License

See main repository for license information.
