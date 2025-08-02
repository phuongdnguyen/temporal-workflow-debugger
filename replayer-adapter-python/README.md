# Python Replayer Adapter for Temporal

This package provides a replayer adapter and interceptors for Temporal workflows. It enables debugging and replaying workflows with breakpoint support.

## Features

- **Workflow and Activity Interceptors**: Comprehensive interceptors for debugging workflow execution
- **Breakpoint Support**: Set breakpoints for standalone mode or integrate with IDE debugger
- **History Loading**: Load workflow history from JSON files or HTTP endpoints
- **Replay Modes**: Support for both standalone and IDE-integrated replay modes

## Installation

### Option 1: Install from source (recommended for development)

```bash
# Clone the repository and navigate to the replayer-adapter-python directory
cd replayer-adapter-python

# Install in development mode
pip install -e .

# Or install with development dependencies
pip install -e .[dev]
```

### Option 2: Install from requirements file (basic)

```bash
pip install -r requirements.txt
```

> **Note**: This option requires temporalio>=1.15.0 for optimal compatibility.

### Option 3: Install as a package

```bash
# Install using pip from the directory
pip install .
```

## Usage

### Basic Usage

```python
# Import the module (after installation)
from replayer_adapter_python import (
    ReplayMode, ReplayOptions, set_replay_mode, 
    set_breakpoints, replay
)

# Set replay mode
set_replay_mode(ReplayMode.STANDALONE)

# Set breakpoints (for standalone mode)
set_breakpoints([10, 25, 50])

# Create replay options
opts = ReplayOptions(
    history_file_path="/path/to/history.json"
)

# Replay workflow
replay(opts, YourWorkflowClass)
```

### IDE Integration Mode

```python
from replayer_adapter_python import (
    ReplayMode, ReplayOptions, set_replay_mode, replay
)

# Set IDE mode
set_replay_mode(ReplayMode.IDE)

# The adapter will automatically connect to the IDE debugger
# via the WFDBG_HISTORY_PORT environment variable (default: 54578)
opts = ReplayOptions()
```

### Using Interceptors

```python
from replayer_adapter_python import RunnerWorkerInterceptor
from temporalio.worker import ReplayerConfig

# Add the interceptor to your replay options
opts = ReplayerConfig()
opts['interceptors'] = [RunnerWorkerInterceptor()]
```

## Module Structure

After installation, the module provides the following main components:

- **`ReplayMode`**: Enum for different replay modes (`STANDALONE`, `IDE`)
- **`ReplayOptions`**: Configuration class for replay settings
- **`replay()`**: Main function to replay workflows
- **`set_replay_mode()`**: Function to set the current replay mode
- **`set_breakpoints()`**: Function to set breakpoints for standalone mode
- **`RunnerWorkerInterceptor`**: Main interceptor class for workflow debugging

## Verification

To verify the installation works correctly, run:

```python
import replayer_adapter_python
print(f"Replayer adapter version: {replayer_adapter_python.__version__}")
print(f"Available components: {replayer_adapter_python.__all__}")
```
replay(opts, YourWorkflowClass)
```

### Using with Temporal Worker

```python
from temporalio.worker import Worker
from replayer_adapter_python import RunnerWorkerInterceptor

# Add the interceptor to your worker
worker = Worker(
    client,
    task_queue="your-task-queue",
    workflows=[YourWorkflow],
    activities=[your_activity],
    interceptors=[RunnerWorkerInterceptor()]
)
```

## API Reference

### Classes

#### `ReplayMode`
Enum for replay modes:
- `STANDALONE`: Replay with local history file
- `IDE`: Replay with IDE debugger integration

#### `ReplayOptions`
Configuration for replay:
- `worker_replay_options`: Temporal WorkflowReplayerOptions
- `history_file_path`: Path to history JSON file (standalone mode)

### Functions

#### `set_replay_mode(mode: ReplayMode)`
Set the replay mode for the adapter.

#### `set_breakpoints(event_ids: List[int])`
Set breakpoints for standalone mode.

#### `replay(opts: ReplayOptions, wf: Any)`
Main replay function that handles both modes.

### Interceptors

#### `RunnerWorkerInterceptor`
Main worker interceptor that provides workflow and activity debugging.

#### `RunnerWorkflowInboundInterceptor`
Workflow inbound interceptor for debugging workflow execution.

#### `RunnerWorkflowOutboundInterceptor`
Workflow outbound interceptor for debugging workflow operations.

## Environment Variables

- `WFDBG_HISTORY_PORT`: Port for IDE debugger communication (default: 54578)

## Examples

### Standalone Replay with Breakpoints

```python
import asyncio
from replayer_adapter_python import *

async def main():
    # Configure replay
    set_replay_mode(ReplayMode.STANDALONE)
    set_breakpoints([5, 15, 30])
    
    # Create options
    opts = ReplayOptions(
        history_file_path="workflow_history.json"
    )
    
    # Replay
    await replay(opts, MyWorkflow)

if __name__ == "__main__":
    asyncio.run(main())
```

### IDE Integration

```python
import os
from replayer_adapter_python import *

# Set environment for IDE integration
os.environ["WFDBG_HISTORY_PORT"] = "54578"

# Configure for IDE mode
set_replay_mode(ReplayMode.IDE)

# Replay with IDE debugger
opts = ReplayOptions()
replay(opts, MyWorkflow)
```

## Architecture

The Python replayer adapter follows the same architecture as the Go version:

1. **Replay Modes**: Supports both standalone and IDE-integrated replay
2. **Interceptors**: Comprehensive workflow and activity interceptors
3. **Breakpoint Management**: Dynamic breakpoint checking and IDE communication
4. **History Loading**: Flexible history loading from files or HTTP endpoints

## Contributing

This is a direct port of the Go replayer adapter. When contributing, please ensure compatibility with the original Go implementation. 