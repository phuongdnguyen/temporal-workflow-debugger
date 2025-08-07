<img src="docs/images/logo.svg" alt="Temporal Debugger Logo" width="700">

## Introduction

A comprehensive debugging solution that enables step-through debugging of [Temporal](https://github.com/temporalio/temporal) workflows. Unlike traditional debuggers that aren't aware of Temporal's execution model, this debugger provides a seamless development experience by allowing you to set breakpoints, inspect variables, and trace execution flow within your workflow code.

## Why This Debugger?

Debugging Temporal workflows has traditionally been challenging. Execution of a workflow is driven by history events rather than direct code execution. Workflow state is managed externally by the Temporal service, and the progress of a workflow depends on interaction between the Temporal server and a thick SDK that knows how to use history events to trigger the actual workflow code execution.

This debugger solves these challenges by leveraging the **workflow replayer** - it reconstructs workflow execution from Temporal's event history, allowing you to debug exactly what happened during the original execution.

## Key Features

### **Seamless IDE Integration**
- **Multi-language Support**: Debug workflows written in multiple SDK languages (Go, TypeScript, and Python examples provided, with support for other languages coming soon)
- **JetBrains Plugin**: Native integration with GoLand via a debugging plugin, using standard IDE debugging controls (breakpoints, step-over, step-into, variable inspection) or set breakpoints in workflow history

## Who Is This For?

- **Temporal Workflow Developers**: Anyone building workflows with Temporal's SDK

Whether you're debugging a complex workflow that's failing in production or just want a better development experience while building new workflows, this debugger provides the tools you need to understand and fix your Temporal workflow code efficiently.

## Usage

### **Installation Options**

Pre-requisite: install `tdlv` debugger from [Github Release](https://github.com/phuongdnguyen/temporal-workflow-debugger/releases/tag/tdlv-v0.0.1)

**IDE Plugins:**
Jetbrains (preview, Go support only): <a href="https://plugins.jetbrains.com/plugin/28127-temporal-workflow-debugger"><img src="https://img.shields.io/badge/Install%20from%20JetBrains%20Marketplace-000000?logo=jetbrains&logoColor=white" alt="Install from JetBrains Marketplace"></a>

Vscode (Go, Python and JS): <a href="https://marketplace.visualstudio.com/items?itemName=phuongdnguyen.temporal-workflow-debugger"><img src="https://img.shields.io/badge/Install%20from%20VS%20Code%20Marketplace-007ACC?logo=visual-studio-code&logoColor=white" alt="Install from VS Code Marketplace"></a>

**Replayer Adapters for Temporal SDK Languages:**
- [Go](https://pkg.go.dev/github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go)
- [Python](https://pypi.org/project/temporal-replayer-adapter-python/)
- [TypeScript](https://www.npmjs.com/package/@phuongdnguyen/replayer-adapter-nodejs)



### Replay a workflow with the plugin/extension

1) Load workflow history in the IDE
- Open the Temporal Workflow Debugger view in your IDE (JetBrains or VS Code)
- Load a Temporal workflow history file (`.json`)
- The IDE starts a local server on `http://127.0.0.1:54578` that serves:
  - `GET /history` (workflow history)
  - `GET /breakpoints` (enabled breakpoints)
  - `POST /current-event` (highlight current event)

2) Run your adapter in IDE mode to replay
- Go
  ```go
  import (
      "go.temporal.io/sdk/worker"
      replayeradapter "github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go"
  )

  func main() {
      replayeradapter.SetReplayMode(replayeradapter.ReplayModeIde)
      opts := replayeradapter.ReplayOptions{
          WorkerReplayOptions: worker.WorkflowReplayerOptions{},
      }
      if err := replayeradapter.Replay(opts, YourWorkflow); err != nil { panic(err) }
  }
  ```

- TypeScript/Node.js
  ```ts
  import { ReplayMode, replay } from '@phuongdnguyen/replayer-adapter-nodejs'

  await replay({
    mode: ReplayMode.IDE,
    workerReplayOptions: { workflowsPath: require.resolve('./workflows') }
  }, yourWorkflow)
  ```

- Python
  ```python
  from replayer_adapter_python import *

  set_replay_mode(ReplayMode.IDE)
  opts = ReplayOptions()
  replay(opts, YourWorkflowClass)
  ```

3) Debug
- Set breakpoints in the IDE history view
- Start your adapter code. It will fetch history/breakpoints from the IDE and pause on hits

