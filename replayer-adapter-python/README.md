# Python Replayer Adapter for Temporal

A Python adapter for debugging and replaying Temporal workflows with breakpoint support.

## Features

- **Workflow Debugging**: Comprehensive interceptors for debugging workflow execution
- **Breakpoint Support**: Set breakpoints for standalone mode or integrate with IDE debugger
- **Dual Modes**: Support for both standalone (JSON file) and IDE-integrated replay modes
- **Activity Debugging**: Interceptors for activity execution debugging

## Installation

```bash
# Install in development mode
pip install -e .

# Or install with development dependencies
pip install -e .[dev]
```

## Quick Start

### Standalone Mode

```python
from replayer_adapter_python import *

# Configure replay
set_replay_mode(ReplayMode.STANDALONE)
set_breakpoints([5, 15, 30])

# Replay workflow
opts = ReplayOptions(history_file_path="workflow_history.json")
replay(opts, YourWorkflowClass)
```

### IDE Integration Mode

```python
from replayer_adapter_python import *

# Set IDE mode (connects via WFDBG_HISTORY_PORT env var)
set_replay_mode(ReplayMode.IDE)

# Replay with IDE debugger
opts = ReplayOptions()
replay(opts, YourWorkflowClass)
```

## API Reference

### Core Functions

- `set_replay_mode(mode)`: Set replay mode (`ReplayMode.STANDALONE` or `ReplayMode.IDE`)
- `set_breakpoints(event_ids)`: Set breakpoints for standalone mode
- `replay(opts, workflow_class)`: Main replay function

### Classes

- `ReplayOptions`: Configuration for replay settings
- `RunnerWorkerInterceptor`: Main interceptor for workflow debugging

## Environment Variables

- `WFDBG_HISTORY_PORT`: Port for IDE debugger communication (default: 54578)

## Dependencies

- `temporalio>=1.15.0`
- `requests>=2.32.0` 