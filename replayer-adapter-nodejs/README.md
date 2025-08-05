# Node.js Replayer Adapter for Temporal

This package provides a replayer adapter and interceptors for Temporal Node.js workflows. It enables debugging and replaying workflows with breakpoint support, following the same pattern as the Go and Python implementations.

## Features

- **Workflow and Activity Interceptors**: Comprehensive interceptors for debugging workflow execution
- **Breakpoint Support**: Set breakpoints for standalone mode or integrate with IDE debugger
- **History Loading**: Load workflow history from JSON files or HTTP endpoints  
- **Replay Modes**: Support for both standalone and IDE-integrated replay modes
- **TypeScript Support**: Full TypeScript definitions included

## Installation

### Option 1: Install from source (recommended for development)

```bash
# Clone the repository and navigate to the replayer-adapter-nodejs directory
cd replayer-adapter-nodejs

# Install dependencies
npm install

# Build the package
npm run build

# Install locally
npm link
```

### Option 2: Install as a package

```bash
# Install using npm from the directory
npm install ./replayer-adapter-nodejs
```

## Usage

### Basic Usage (Standalone Mode)

```typescript
import {
  ReplayMode,
  ReplayOptions,
  setReplayMode,
  setBreakpoints,
  replay
} from '@temporal/replayer-adapter-nodejs';

// Set replay mode
setReplayMode(ReplayMode.STANDALONE);

// Set breakpoints (for standalone mode)
setBreakpoints([10, 25, 50]);

// Create replay options
const opts: ReplayOptions = {
  historyFilePath: "/path/to/history.json",
  workerReplayOptions: {
    workflowsPath: require.resolve('./workflows'),
  }
};

// Replay workflow
await replay(opts, YourWorkflowClass);
```

### IDE Integration Mode

```typescript
import {
  ReplayMode,
  ReplayOptions,
  setReplayMode,
  replay
} from '@temporal/replayer-adapter-nodejs';

// Set IDE mode
setReplayMode(ReplayMode.IDE);

// The adapter will automatically connect to the IDE debugger
// via the WFDBG_HISTORY_PORT environment variable (default: 54578)
const opts: ReplayOptions = {
  workerReplayOptions: {
    workflowsPath: require.resolve('./workflows'),
  }
};

// Replay workflow
await replay(opts, YourWorkflowClass);
```

### Using Interceptors Directly

The interceptors are automatically included when using the `replay` function, but you can also use them directly with a Temporal Worker:

```typescript
import { Worker } from '@temporalio/worker';
import { interceptors as workflowInterceptors } from '@temporal/replayer-adapter-nodejs/dist/workflow-interceptors';
import { activityInterceptors } from '@temporal/replayer-adapter-nodejs/dist/activity-interceptors';

const worker = Worker.create({
  taskQueue: 'your-task-queue',
  workflowsPath: require.resolve('./workflows'),
  interceptors: {
    workflowModules: [workflowInterceptors],
    activity: [activityInterceptors],
  },
});
```

## API Reference

### Enums

#### `ReplayMode`
Enum for replay modes:
- `STANDALONE`: Replay with local history file
- `IDE`: Replay with IDE debugger integration

### Interfaces

#### `ReplayOptions`
Configuration for replay:
- `workerReplayOptions`: Temporal Worker replay options
- `historyFilePath`: Path to history JSON file (standalone mode)

### Functions

#### `setReplayMode(mode: ReplayMode)`
Set the replay mode for the adapter.

#### `setBreakpoints(eventIds: number[])`
Set breakpoints for standalone mode.

#### `replay(opts: ReplayOptions, workflow: any): Promise<void>`
Main replay function that handles both modes.

#### `replayWithHistory(opts: any, hist: any, workflow: any): Promise<void>`
Replay workflow with history data.

#### `replayWithJsonFile(opts: any, workflow: any, jsonFileName: string): Promise<void>`
Replay workflow with history from JSON file.

### Interceptors

#### `workflowInterceptors`
Workflow interceptor factory for debugging support. Automatically injected when using the `replay` function.

#### `activityInterceptors`
Activity interceptor factory for debugging support. Automatically injected when using the `replay` function.

## Environment Variables

- `WFDBG_HISTORY_PORT`: Port for IDE debugger communication (default: 54578)

## Examples

See the `example/` directory for complete examples of:
- Standalone workflow replay with breakpoints
- IDE-integrated workflow debugging

### Basic Usage Example

```typescript
import {
  ReplayMode,
  ReplayOptions,
  setReplayMode,
  setBreakpoints,
  replay
} from '@temporal/replayer-adapter-nodejs';

// Example workflow
async function greetingWorkflow(name: string): Promise<string> {
  return `Hello, ${name}!`;
}

async function main() {
  // Standalone mode
  setReplayMode(ReplayMode.STANDALONE);
  setBreakpoints([1, 5, 10]);
  
  const standaloneOpts: ReplayOptions = {
    historyFilePath: './example-history.json',
    workerReplayOptions: {
      workflowsPath: require.resolve('./workflows'),
    }
  };
  
  await replay(standaloneOpts, greetingWorkflow);
  
  // IDE mode
  setReplayMode(ReplayMode.IDE);
  const ideOpts: ReplayOptions = {
    workerReplayOptions: {
      workflowsPath: require.resolve('./workflows'),
    }
  };
  
  await replay(ideOpts, greetingWorkflow);
}
```

## Architecture

This replayer adapter follows the same architecture as the Go and Python implementations:

1. **Core Functions**: Handle mode management, breakpoints, and replay orchestration
2. **Interceptors**: Inject breakpoint detection into workflow and activity execution
3. **HTTP Client**: Communicate with IDE debugger for breakpoint status and highlighting
4. **History Loading**: Support loading from both JSON files and HTTP endpoints

### Key Components

- **`replayer.ts`**: Main replay logic and breakpoint handling
- **`workflow-interceptors.ts`**: Workflow interceptors for debugging support
- **`activity-interceptors.ts`**: Activity interceptors for debugging support
- **`types.ts`**: Type definitions and state management
- **`http-client.ts`**: HTTP communication with IDE debugger

## Compatibility

- Node.js 16.x or higher
- Temporal TypeScript SDK 1.12.0 or higher
- TypeScript 4.9.0 or higher

## Development

### Building

```bash
npm run build
```

### Development Mode

```bash
npm run dev
```

### Clean Build

```bash
npm run clean
npm run build
```

## Contributing

This package follows the same patterns as the existing Go and Python replayer adapters. When making changes, ensure compatibility across all three implementations.

## License

MIT - See LICENSE file for details. 