# Debugger for Temporal Workflows

<img src="docs/images/logo.png" alt="Temporal Debugger Logo" width="200">

# Introduction

A comprehensive debugging solution that enables step-through debugging of [Temporal](https://github.com/temporalio/temporal) workflows. Unlike traditional debuggers that struggle with Temporal's distributed execution model, this debugger provides a seamless development experience by allowing you to set breakpoints, inspect variables, and trace execution flow within your workflow code.

## 🚀 Why This Debugger?

Debugging Temporal workflows has traditionally been challenging because:

- **Distributed Execution**: Workflows can pause, resume, and retry across multiple processes and machines
- **Complex State Management**: Execution is driven by history events rather than direct code execution. Workflow state is managed externally by the Temporal service, the progress of a workflow depends on interaction between Temporal server and a thick SDK that know how to use history event to trigger the actual workflow code execution. 

This debugger solves these challenges by implementing **deterministic replay debugging** - it reconstructs workflow execution from Temporal's event history, allowing you to debug exactly what happened during the original execution.

## ✨ Key Features

### 🎯 **Seamless IDE Integration**
- **JetBrains Plugin**: Native integration with GoLand, IntelliJ IDEA, and other JetBrains IDEs
- **Familiar Debugging Experience**: Use standard IDE debugging controls (breakpoints, step-over, step-into, variable inspection)
- **Tool Window**: Dedicated panel for workflow history visualization and breakpoint management

### 🔍 **Workflow History Debugging**
- **History Upload**: Load Temporal workflow execution history (JSON format)
- **Event Visualization**: Browse through workflow events with timestamps and details
- **Breakpoint Management**: Set breakpoints on specific workflow events or code locations



## 🏛 Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   JetBrains     │    │   Serving Layer  │    │ Debugger Server │
│   IDE Plugin    │◄──►│   (tdlv)         │◄──►│   + Workflow    │
│                 │    │                  │    │   Replayer      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
        │                       │                       │
        │              ┌────────▼────────┐              │
        │              │ History Server  │              │
        │              │ (Breakpoints +  │              │
        └─────────────►│  Events)        │◄─────────────┘
                       └─────────────────┘
```

The debugger consists of several integrated components:

1. **JetBrains Plugin**: Provides the user interface, manages workflow history, and integrates with IDE debugging
2. **Servubg layer (`tdlv`)**: Intercepts debugging commands and enhances them with workflow-specific logic
3. **Workflow Replayer**: Executes workflow code deterministically using Temporal's replayer
4. **History Server**: Manages workflow event history and breakpoint state
5. **Adapter Layer**: Connects the replay execution with the debugging infrastructure

## 👥 Who Is This For?

- **Temporal Workflow Developers**: Anyone building workflows with Temporal's SDK

Whether you're debugging a complex workflow that's failing in production or just want a better development experience while building new workflows, this debugger provides the tools you need to understand and fix your Temporal workflow code efficiently.


# Usage
You can run the debugger in:
- Standalone mode: run the debugger with your workflow code and connect your IDE to it. This approach is lower-level and not recommended for end user. Install the debugger

```bash
brew install tdlv
```

- IDE Integrated: install the plugin and debug your workflow via a debugging UI. This approach provides a more complete debugging experience and is the recommended approach.

Install the plugin from:
- [Jetbrains marketplace](https://plugins.jetbrains.com/search?excludeTags=internal&products=androidstudio&products=aqua&products=clion&products=dataspell&products=dbe&products=fleet&products=go&products=idea&products=idea_ce&products=mps&products=phpstorm&products=pycharm&products=rider&products=ruby&products=rust&products=webstorm&products=writerside&search=Temporal%20workflow%20debugger)
- [Vscode marketplace](https://marketplace.visualstudio.com/search?term=Temporal%20workflow%20debugger&target=VSCode&category=All%20categories&sortBy=Relevance)
