# Temporal Node.js Replayer Adapter

Replayer adapter and interceptors for Temporal workflows in Node.js with breakpoint support for debugging.

## Features

- **Standalone Mode**: Replay workflows from JSON history files with breakpoints
- **IDE Integration**: Connect to IDEs for interactive debugging
- **Workflow Interceptors**: Automatically intercept workflow operations
- **Breakpoint Support**: Set breakpoints at specific workflow event IDs

## Installation

```bash
npm install @temporal/replayer-adapter-nodejs
```

## Quick Start

### Standalone Mode

```typescript
import { ReplayMode, replay } from '@temporal/replayer-adapter-nodejs';

const options = {
  mode: ReplayMode.STANDALONE,
  breakpoints: [1, 5, 10, 15],
  historyFilePath: './workflow-history.json',
  workerReplayOptions: {
    workflowsPath: require.resolve('./workflows'),
  }
};

await replay(options, myWorkflow);
```

### IDE Integration Mode

```typescript
import { ReplayMode, replay } from '@temporal/replayer-adapter-nodejs';

const options = {
  mode: ReplayMode.IDE,
  debuggerAddr: 'http://127.0.0.1:54578',
  workerReplayOptions: {
    workflowsPath: require.resolve('./workflows'),
  }
};

await replay(options, myWorkflow);
```

## Configuration Options

```typescript
interface ReplayOptions {
  mode?: ReplayMode;                    // STANDALONE or IDE
  breakpoints?: number[];               // Event IDs to pause at
  historyFilePath?: string;             // Required for STANDALONE mode
  debuggerAddr?: string;                // Required for IDE mode
  workerReplayOptions?: ReplayWorkerOptions;
}
```

## Alternative Configuration

You can also configure using separate functions:

```typescript
import { setReplayMode, setBreakpoints, setDebuggerAddr } from '@temporal/replayer-adapter-nodejs';

setReplayMode(ReplayMode.STANDALONE);
setBreakpoints([1, 5, 10, 15]);
setDebuggerAddr('http://127.0.0.1:54578');

await replay(options, myWorkflow);
```

## API Reference

### Functions

- `replay(options: ReplayOptions, workflow: any): Promise<void>`
- `setReplayMode(mode: ReplayMode)`
- `setBreakpoints(eventIds: number[])`
- `setDebuggerAddr(addr: string)`

### Types

- `ReplayMode.STANDALONE`: Replay using local history file
- `ReplayMode.IDE`: Replay with IDE integration

## Troubleshooting

### Breakpoints Not Working

1. **Configuration**: Ensure breakpoints are set via options or functions
2. **Event IDs**: Verify event IDs exist in workflow history
3. **Mode**: Confirm correct replay mode is set
4. **Console**: Check for breakpoint messages in console

### IDE Connectivity

For IDE mode, ensure:
- Correct `debuggerAddr` is set
- IDE debugger server is running
- `WFDBG_HISTORY_PORT` environment variable is set if needed

## Examples

See `example/js/` directory for complete working examples.

## Contributing

Contributions welcome! See main repository for guidelines.

## License

See main repository for license information. 