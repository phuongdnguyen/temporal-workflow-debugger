# Temporal Node.js Replayer Adapter

This package provides a replayer adapter and interceptors for Temporal workflows in Node.js. It enables debugging and replaying workflows with breakpoint support in both standalone and IDE-integrated modes.

## Features

- **Standalone Mode**: Replay workflows from JSON history files with custom breakpoints
- **IDE Integration Mode**: Connect to IDEs for interactive debugging
- **Workflow Interceptors**: Automatically intercept workflow operations for debugging
- **Activity Interceptors**: Intercept activity executions for comprehensive debugging
- **Breakpoint Support**: Set breakpoints at specific workflow event IDs

## Installation

```bash
npm install @temporal/replayer-adapter-nodejs
```

## Quick Start

### Standalone Mode

```typescript
import {
  ReplayMode,
  setReplayMode,
  setBreakpoints,
  replay
} from '@temporal/replayer-adapter-nodejs';

// IMPORTANT: Configure mode and breakpoints BEFORE calling replay()
setReplayMode(ReplayMode.STANDALONE);
setBreakpoints([1, 5, 10, 15]); // Set breakpoints at specific event IDs

const options = {
  historyFilePath: './workflow-history.json',
  workerReplayOptions: {
    workflowsPath: require.resolve('./workflows'),
  }
};

await replay(options, myWorkflow);
```

### IDE Integration Mode

```typescript
import {
  ReplayMode,
  setReplayMode,
  replay
} from '@temporal/replayer-adapter-nodejs';

setReplayMode(ReplayMode.IDE);

const options = {
  workerReplayOptions: {
    workflowsPath: require.resolve('./workflows'),
  }
};

// Set environment variable for IDE connection
process.env.WFDBG_HISTORY_PORT = '54578';

await replay(options, myWorkflow);
```

## Breakpoint Management

### Setting Breakpoints

Breakpoints are set by event ID numbers that correspond to events in your workflow history:

```typescript
import { setBreakpoints } from '@temporal/replayer-adapter-nodejs';

// Set breakpoints at events 1, 5, 10, and 15
setBreakpoints([1, 5, 10, 15]);

// Update breakpoints (replaces previous ones)
setBreakpoints([2, 4, 6, 8]);

// Clear all breakpoints
setBreakpoints([]);
```

### Important Notes

1. **Order Matters**: Always call `setReplayMode()` and `setBreakpoints()` BEFORE calling `replay()`
2. **Event IDs**: Breakpoint numbers should correspond to actual event IDs in your workflow history
3. **Standalone vs IDE**: In standalone mode, you manage breakpoints manually. In IDE mode, the IDE manages them
4. **Empty by Default**: Breakpoints start empty - you must explicitly set them

## API Reference

### Functions

#### `setReplayMode(mode: ReplayMode)`
Set the replay mode (STANDALONE or IDE).

#### `setBreakpoints(eventIds: number[])`
Set breakpoints at specific workflow event IDs. Replaces any existing breakpoints.

#### `replay(options: ReplayOptions, workflow: any): Promise<void>`
Replay a workflow with the configured options and breakpoints.

### Types

#### `ReplayMode`
- `STANDALONE`: Replay using local history file
- `IDE`: Replay with IDE integration

#### `ReplayOptions`
```typescript
interface ReplayOptions {
  workerReplayOptions?: ReplayWorkerOptions;
  historyFilePath?: string; // Required for STANDALONE mode
}
```

## Troubleshooting

### Breakpoints Not Working

If breakpoints aren't triggering, check:

1. **Configuration Order**: Ensure you call `setBreakpoints()` before `replay()`
2. **Event IDs**: Verify the event IDs exist in your workflow history
3. **Mode Setting**: Confirm you've set the correct replay mode
4. **Console Output**: Look for breakpoint checking messages in the console

### Example Console Output

When working correctly, you should see output like:
```
Breakpoints updated to: [1, 5, 10, 15]
Standalone checking breakpoints: [1, 5, 10, 15], eventId: 1
✓ Hit breakpoint at eventId: 1
```

## Testing the Fix

A test script is included to verify breakpoints work correctly:

```bash
npm run build
node test-breakpoints.js
```

All tests should show "✓ PASS" if the breakpoint system is working correctly.

## Examples

See the `example/` directory for complete working examples of both standalone and IDE integration modes.

## Contributing

Contributions are welcome! Please see the main repository for contribution guidelines.

## License

See the main repository for license information. 