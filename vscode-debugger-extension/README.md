<h1 align="center">
  <br>
  <br>
    <img src="https://raw.githubusercontent.com/phuongdnguyen/temporal-workflow-replay-debugger/ad698d27dec8950cf83b629df47763223edbceab/vscode-debugger-extension/banner.png" alt="Temporal Workflow Replay Debugger">
  <br>
</h1>

<h4 align="center">Debug Temporal workflows by their ID or history file.</h4>
<h4 align="center">Set breakpoints in code or on history events.</h4>
<h4 align="center">Support multiple workflow languages.</h4>

## Usage

Follow these instructions:

- Install [the extension](https://marketplace.visualstudio.com/items?itemName=phuongdnguyen.temporal-workflow-replay-debugger)

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

### Examples

#### TypeScript

Create a small `replayer.ts` in your project that runs the Tyepscript replayer adapter in IDE mode and registers your workflow function, for example:

1. Install the replayer first:

```
npm i @phuongdnguyen/replayer-adapter-nodejs --save
```

2. Install the debugger [tdlv](https://github.com/phuongdnguyen/temporal-workflow-replay-debugger/releases/tag/tdlv-v0.0.2) and add it to PATH
3. Verify tldv is installed in PATH

```
tdlv --help
Missing required flags: -lang

Tdlv (Temporal delve) is a Temporal Workflow Replay Debugger

Usage: tdlv [options]

  -help
        Tdlv (Temporal delve) is a Temporal Workflow Replay Debugger, provide ability to focus on user workflow code in debug sessions (alias: -h)
  -install
        auto-install missing language debuggers
  -lang string
        [required] language to use for the workflow, available options: [go, python, js]
  -p int
        port for remote debugging (default 60000)
  -start
        start debugger
```

4. Your entrypoint file should import the replayer adapter and your workflow:

```typescript
import { exampleWorkflow } from "./workflow"
import { ReplayMode, replay } from "@phuongdnguyen/replayer-adapter-nodejs"

async function main() {
  const opts = {
    mode: ReplayMode.IDE,
    workerReplayOptions: {
      workflowsPath: require.resolve("./workflow.ts"),
    },
  }

  await replay(opts, exampleWorkflow)
}

if (require.main === module) {
  main().catch((error) => {
    console.error("Error:", error)
    process.exit(1)
  })
}
```

5. Open or create `.vscode/settings.json` and add the config field:

```json
{
  "temporal.replayerEntryPoint": "replayer.ts"
}
```

_Note that the file must be within your project directory so it can find `node_modules/`._

#### Go

1. Get the replayer code

```
go get -u github.com/phuongdnguyen/temporal-workflow-replay-debugger/replayer-adapter-go@latest
```

2. Create a small `main.go` in your project that runs the Go replayer adapter in IDE mode and registers your workflow function, for example:

```go
package main

import (
    "go.temporal.io/sdk/worker"
    replayer_adapter_go "github.com/phuongdnguyen/temporal-workflow-replay-debugger/replayer-adapter-go"
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

3. Configure the extension:

```json
{
  "temporal.debugLanguage": "go",
  "temporal.replayerEntrypoint": "main.go"
}
```

4. Run "Temporal: Open Panel"
5. Enter a Workflow Id or choose a history JSON file
6. Click `Load History`
7. Select history events that you want the workflow to be stopped on
8. Hit `Start debug session`

#### Python

1. Make sure your Python environment has the required dependencies installed:

```bash
pip install temporalio replayer-adapter-python
```

2. Create a small script (e.g. `replayer.py`) that uses the Python replayer adapter in IDE mode and references your workflow:

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

3. Configure the extension:

```json
{
  "temporal.debugLanguage": "python",
  "temporal.replayerEntryPoint": "replayer.py"
  // If you want use a custom python rather the one in PATH
  // "temporal.python": "/Your/path/to/python"
}
```

4. Run "Temporal: Open Panel"
5. Enter a Workflow Id or choose a history JSON file
6. Click `Load History`
7. Select history events that you want the workflow to be stopped on
8. Hit `Start debug session`
