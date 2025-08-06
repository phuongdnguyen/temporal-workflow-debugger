# Temporal Go Replayer Adapter

A Go adapter for debugging Temporal workflows by replaying execution history with breakpoint support.

## Installation

```bash
go get github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go
```

## Overview

This adapter provides workflow replay functionality for Temporal Go SDK applications, enabling debugging through:

- **Standalone Mode**: Replay workflows using local history files
- **IDE Mode**: Replay workflows with debugger UI integration

## Features

- Workflow history replay with breakpoint support
- Interceptor-based debugging hooks
- Support for both standalone and IDE-integrated debugging
- Activity and workflow execution tracking

## Usage

```go
import (
    "go.temporal.io/sdk/worker"
    replayeradapter "github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go"
)

// Set replay mode
replayeradapter.SetReplayMode(replayeradapter.ReplayModeIde)

// Configure replay options
opts := replayeradapter.ReplayOptions{
    WorkerReplayOptions: worker.WorkflowReplayerOptions{},
    HistoryFilePath:     "/path/to/history.json", // for standalone mode
}

// Replay workflow
err := replayeradapter.Replay(opts, yourWorkflow)
if err != nil {
    log.Fatal(err)
}
```

### Standalone Mode Example

```go
// Set breakpoints at specific event IDs
replayeradapter.SetBreakpoints([]int{1, 5, 10})

// Set standalone mode
replayeradapter.SetReplayMode(replayeradapter.ReplayModeStandalone)

// Configure with history file
opts := replayeradapter.ReplayOptions{
    WorkerReplayOptions: worker.WorkflowReplayerOptions{},
    HistoryFilePath:     "/path/to/your/workflow-history.json",
}

// Replay workflow
err := replayeradapter.Replay(opts, yourWorkflowFunction)
```

## Dependencies

- `go.temporal.io/sdk` v1.35.0
- `go.temporal.io/api` v1.49.1
- `github.com/bufbuild/connect-go` v1.10.0

## Architecture

The adapter uses Temporal SDK interceptors to hook into workflow execution:

- **Inbound Interceptors**: Track workflow and activity execution entry points
- **Outbound Interceptors**: Monitor workflow operations (activities, timers, signals, etc.)
- **Breakpoint Management**: Support for setting and checking breakpoints during replay 