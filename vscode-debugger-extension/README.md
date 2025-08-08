<h1 align="center">
  Temporal for VS Code
  <br>
  <br>
    <img src="https://assets.temporal.io/w/vscode.png" alt="Temporal for VS Code">
  <br>
</h1>

<h4 align="center">Debug Temporal workflows by their ID or history file.</h4>
<h4 align="center">Set breakpoints in code or on history events.</h4>

## Usage

Watch [the demo](https://www.youtube.com/watch?v=3IjQde9HMNY) or follow these instructions:

- Install [the extension](https://marketplace.visualstudio.com/items?itemName=phuongdnguyen.temporal-workflow-debugger)

- Follow the examples for:
- [Typescript](../example/js/vscode-replayer.ts)
- [Go](../example/go/structured-workflow/replay-debug-ide-integrated/)
- [Python](../example/python/vscode-replay.py)

- Edit the `'./workflows'` path to match the location of your workflows file
- Run `Temporal: Open Panel` (use `Cmd/Ctrl-Shift-P` to open Command Palette)
- Enter a Workflow Id or choose a history JSON file
- Click `Start`
- The Workflow Execution will start replaying and hit a breakpoint set on the first event
- Set breakpoints in Workflow code (the extension uses a [replay Worker](https://typescript.temporal.io/api/classes/worker.Worker#runreplayhistory), so Activity code is not run) or on history events
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

#### Typescript

By default, the extension will look for the file that calls [`startDebugReplayer`](https://typescript.temporal.io/api/namespaces/worker#startdebugreplayer) at `src/debug-replayer.ts`. To use a different TypeScript or JavaScript file, set the `temporal.typeScriptReplayerEntrypoint` config:

- Open or create `.vscode/settings.json`
- Add the config field:

  ```json
  {
    "temporal.typeScriptReplayerEntrypoint": "test/different-file.ts"
  }
  ```

_Note that the file must be within your project directory so it can find `node_modules/`._

#### Go

Go entrypoints are started via the background process. Create a small `main.go` in your project that runs the Go replayer adapter in IDE mode and registers your workflow function, for example:

```go
package main

import (
    "go.temporal.io/sdk/worker"
    replayer "github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go"
    "your/module/workflows"
)

func main() {
    replayer.SetReplayMode(replayer.ReplayModeIde)
    _ = replayer.Replay(replayer.ReplayOptions{
        WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
    }, workflows.YourWorkflow)
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
from replayer_adapter_python.replayer import ReplayMode, ReplayOptions, set_replay_mode, replay
from workflow import YourWorkflow

async def main():
    set_replay_mode(ReplayMode.IDE)
    await replay(ReplayOptions(worker_replay_options={}), YourWorkflow)

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
