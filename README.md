# Debugger for Temporal Workflows 🐛

<img src="docs/images/logo.svg" alt="Temporal Debugger Logo" width="500">

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

You can run the debugger in two modes:

### **Standalone Mode** 
Run the debugger with your workflow code and connect your IDE to it. This approach is lower-level and not recommended for end users. To install the debugger, download it from the [GitHub Release Page](https://github.com/phuongdnguyen/temporal-workflow-debugger/releases).

### **IDE Integrated Mode** (Recommended)
Currently available for Go users on GoLand: install the plugin and debug your workflow via a debugging UI. This approach provides a more complete debugging experience and is the recommended approach.

### **Installation Options**

**IDE Plugins:**
- [JetBrains Marketplace](https://plugins.jetbrains.com/plugin/28127-temporal-workflow-debugger)
- [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=phuongdnguyen.temporal-workflow-debugger&ssr=false#overview)

**Replayer Adapters for Temporal SDK Languages:**
- [Go](https://pkg.go.dev/github.com/phuongdnguyen/temporal-workflow-debugger/replayer-adapter-go)
- [Python](https://pypi.org/project/temporal-replayer-adapter-python/)
- [TypeScript](https://www.npmjs.com/package/@phuongdnguyen/replayer-adapter-nodejs)


