# Temporal workflow debugger

<h4 align="center">Debug Temporal workflows by their ID or history file.</h4>
<h4 align="center">Set breakpoints in code or on history events.</h4>

## Usage

Follow these instructions:

- Install [the extension](https://marketplace.visualstudio.com/items?itemName=phuongdnguyen.temporal-workflow-debugger)

- Follow the examples for:
- [TypeScript](../example/js/vscode-replayer.ts)
- [Go](../example/go/structured-workflow/replay-debug-ide-integrated/)
- [Python](../example/python/vscode-replay.py)

- Run `Temporal: Open Panel` (use `Cmd/Ctrl-Shift-P` to open Command Palette)
- Enter a Workflow Id or choose a history JSON file
- Click `Load History`
- Select history events that you want the workflow to be stopped on
- The Workflow Execution will start replaying until it hit a breakpoint
- Set breakpoints in Workflow code (the extension uses a Workflow Replayer, so Activity code is not run) or on history events
- Hit play or step forward
- To restart from the beginning, click the green restart icon at the top of the screen, or if the debug session has ended, go back to the `MAIN` tab and `Start` again

## Configuration

### Server

When starting a replay by Workflow Id, the extension downloads the history from the Temporal Server. By default, it connects to a Server running on the default `localhost:7233`.

To connect to a different Server:

- Open the `SETTINGS` tab
- Edit the `Address` field
- If you're using TLS (e.g. to connect to Temporal Cloud), check the box and select your client cert and key

### Entrypoint

#### TypeScript

By default, the extension will look for the file that calls the TypeScript replayer adapter at `src/debug-replayer.ts`. To use a different TypeScript or JavaScript file, set the `temporal.replayerEntryPoint` config:

- Open or create `.vscode/settings.json`
- Add the config field:

  ```json
  {
    "temporal.replayerEntryPoint": "test/different-file.ts"
  }
  ```

Your entrypoint file should import the replayer adapter and your workflow:

```typescript
import { exampleWorkflow } from './workflow';
import { ReplayMode, replay } from '@phuongdnguyen/replayer-adapter-nodejs';

async function main() {
    const opts = {
        mode: ReplayMode.IDE,
        workerReplayOptions: {
            workflowsPath: require.resolve('./workflow.ts'),
            bundlerOptions: {
                ignoreModules: [
                    'fs/promises',
                    '@temporalio/worker',
                    'path',
                    'child_process'
                ]
            },
            debugMode: true,
        },
        debuggerAddr: 'http://127.0.0.1:54578'
    };

    await replay(opts, exampleWorkflow);
}

if (require.main === module) {
    main().catch((error) => {
        console.error('Error:', error);
        process.exit(1);
    });
}
```

_Note that the file must be within your project directory so it can find `node_modules/`._

#### Go

Go entrypoints are started via the background process. Create a small `main.go` in your project that runs the Go replayer adapter in IDE mode and registers your workflow function, for example:

```go
package main

import (
    "go.temporal.io/sdk/worker"
    replayer_adapter_go "github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go"
    "example/pkg/workflows"
)

func main() {
    replayer_adapter_go.SetReplayMode(replayer_adapter_go.ReplayModeIde)
    err := replayer_adapter_go.Replay(replayer_adapter_go.ReplayOptions{
        WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
    }, workflows.ExampleWorkflow)
    if err != nil {
        panic(err)
    }
}
```

Configure the background process to run `tdlv` which builds and runs your entrypoint under Delve and exposes a DAP proxy on port 60000. Set `cwd` to the folder that contains your `package main` (the entrypoint):

```json
{
  "temporal.debugLanguage": "go",
  "temporal.debugger.backgroundProcess.command": "tdlv",
  "temporal.debugger.backgroundProcess.args": ["--lang=go", "--install", "--quiet"],
  "temporal.debugger.backgroundProcess.options": { "cwd": "./path-to-your-entrypoint-folder" }
}
```

The extension automatically attaches to the proxy. If your workspace root is the Go module, you may omit `options.cwd`.

#### Python

Python entrypoints are also started via the background process. Create a small script (e.g. `vscode-replay.py`) that uses the Python replayer adapter in IDE mode and references your workflow:

```python
import asyncio
from replayer_adapter_python.replayer import (
    ReplayMode, ReplayOptions, set_replay_mode, replay
)
from workflow import UserOnboardingWorkflow

async def main():
    """Run ide examples"""
    try:
        # Set up ide mode
        set_replay_mode(ReplayMode.IDE)
        
        # Create replay options
        opts = ReplayOptions(
            worker_replay_options={},
        )
        result = await replay(opts, UserOnboardingWorkflow)
        print(f"Result: {result}")
    except Exception as e:
        print(f"Replay failed: {e}")

if __name__ == "__main__":
    asyncio.run(main())
```

Configure the background process to start `tdlv` for Python and point it at your entrypoint script (DebugPy will be launched by `tdlv` and proxied via port 60000):

```json
{
  "temporal.debugLanguage": "python",
  "temporal.debugger.backgroundProcess.command": "tdlv",
  "temporal.debugger.backgroundProcess.args": [
    "--lang=python",
    "--install",
    "--quiet",
    "--entrypoint",
    "${workspaceFolder}/vscode-replay.py"
  ]
}
```

Make sure your Python environment has the required dependencies installed:

```bash
pip install temporalio replayer-adapter-python
```

### Languages

You can choose which language to debug via the `temporal.debugLanguage` setting. Supported values:

- `typescript` (default)
- `go`
- `java`
- `python`

Set it in your workspace settings:

```json
{
  "temporal.debugLanguage": "go"
}
```

### Background Process

The extension supports running a background process before starting the debug session. This process will be automatically terminated when the debug session ends.

Configure through VS Code settings:

```json
{
  "temporal.debugger.backgroundProcess.command": "npm",
  "temporal.debugger.backgroundProcess.args": ["run", "start"],
  "temporal.debugger.backgroundProcess.options": {
    "cwd": "./server",
    "env": {
      "PORT": "3000"
    }
  }
}
```

Common use cases:

- Starting a Temporal server before debugging
- Running setup scripts or initialization processes

The extension uses graceful termination (SIGTERM) first, then forceful termination (SIGKILL) if needed. Process output is logged to the VS Code console.

### Adapter integration (IDE server)

When a history is loaded in the panel, the extension starts a local server used by language adapters:

- Address: `http://127.0.0.1:54578`
- Endpoints:
  - `GET /history` – returns the workflow history (JSON bytes)
  - `GET /breakpoints` – returns the enabled breakpoint event IDs
  - `POST /current-event` – highlight the current event in the UI

Adapters may honor the `WFDBG_HISTORY_PORT` environment variable to override the default port.

Notes:

- Only Workflow code executes during replay; Activity code isn’t run (effects are driven by history).
