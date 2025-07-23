# Temporal Go Example

This is an example Go module that demonstrates basic Temporal workflow and activity patterns with two main commands.

## Prerequisites

- Go 1.23.0 or later
- Temporal server running locally (see [Temporal documentation](https://docs.temporal.io/))

## Usage

This module supports two main commands:

### 1. Run Worker

Start a Temporal worker that will listen for workflow and activity tasks:

```bash
go run . run
# or
go run . worker
```

The worker will:
- Connect to the local Temporal server
- Register the `ExampleWorkflow` and `ExampleActivity`
- Listen on the `example-task-queue` for tasks
- Continue running until interrupted (Ctrl+C)

### 2. Start Workflow

Execute a workflow instance:

```bash
go run . start
# or 
go run . workflow
```

This will:
- Connect to the Temporal server
- Start an instance of `ExampleWorkflow` 
- Generate a unique workflow ID
- Wait for the workflow to complete and show the result

## Example Flow

1. Start the worker in one terminal:
   ```bash
   go run . run
   ```

2. In another terminal, start a workflow:
   ```bash
   go run . start
   ```

## What's Included

- **ExampleWorkflow**: Demonstrates activity execution, side effects, and timers
- **ExampleActivity**: Simple activity that creates a greeting message
- **Worker setup**: Shows how to register and run workflows/activities
- **Workflow starter**: Shows how to execute workflows programmatically

## Files

- `main.go` - Entry point handling command line arguments
- `worker.go` - Worker setup and configuration
- `starter.go` - Workflow execution logic
- `pkg/workflow.go` - Example workflow definition
- `pkg/activity.go` - Example activity definition

## Project Structure

```
example/
├── main.go              # CLI entry point
├── worker.go            # Worker setup
├── starter.go           # Workflow starter
├── pkg/                 # Business logic package
│   ├── workflow.go      # Workflow definitions
│   └── activity.go      # Activity definitions
├── go.mod               # Module definition
└── README.md            # Documentation
``` 