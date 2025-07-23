# Debugger for Temporal go

<img src="docs/images/logo.png" alt="Temporal Go Debugger Logo" width="200">

# Introduction
Œ
The **Temporal Go Workflow Debugger** is a comprehensive debugging solution that enables step-through debugging of [Temporal](https://github.com/temporalio/temporal) workflows in Go. Unlike traditional debuggers that struggle with Temporal's distributed execution model, this debugger provides a seamless development experience by allowing you to set breakpoints, inspect variables, and trace execution flow within your workflow code.

## 🚀 Why This Debugger?

Debugging Temporal workflows has traditionally been challenging because:

- **Distributed Execution**: Workflows can pause, resume, and retry across multiple processes and machines
- **Event-Driven Model**: Execution is driven by history events rather than direct code execution  
- **Non-Deterministic Replay**: Standard debuggers break Temporal's deterministic replay requirements
- **Complex State Management**: Workflow state is managed externally by the Temporal service

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
- **Call Stack Filtering**: Clean call stacks that hide internal adapter code, showing only your workflow logic

### 🛠 **Advanced Debugging Capabilities**
- **Variable Inspection**: Hover over variables to see their values at any point in execution
- **Local Variables Panel**: View all local variables and function arguments in the current frame
- **Expression Evaluation**: Evaluate expressions in the context of the workflow execution
- **Multi-Protocol Support**: Works with both GoLand (JSON-RPC) and VS Code (DAP) debugging protocols

### 🏗 **Robust Architecture**
- **Delve Proxy**: Transparent proxy between IDE and Delve debugger that intercepts and enhances debugging commands
- **Frame Translation**: Automatically maps between filtered stack frames and original Delve frames for accurate variable inspection
- **Protocol Compatibility**: Maintains full compatibility with standard Go debugging tools while adding Temporal-specific enhancements

## 🏛 Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   JetBrains     │    │   Delve Proxy    │    │   Delve Server  │
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
2. **Delve Proxy (`tdlv`)**: Intercepts debugging commands and enhances them with workflow-specific logic
3. **Workflow Replayer**: Executes workflow code deterministically using Temporal's replay mechanism
4. **History Server**: Manages workflow event history and breakpoint state
5. **Adapter Layer**: Connects the replay execution with the debugging infrastructure

## 👥 Who Is This For?

- **Temporal Go Developers**: Anyone building workflows with Temporal's Go SDK
- **DevOps Engineers**: Teams debugging production workflow issues using execution history
- **Development Teams**: Organizations wanting to improve their Temporal workflow development experience
- **Go Developers**: Developers familiar with standard Go debugging who want to extend those skills to Temporal workflows

Whether you're debugging a complex workflow that's failing in production or just want a better development experience while building new workflows, this debugger provides the tools you need to understand and fix your Temporal Go code efficiently.


# Usage


