# Developer Guide

## Setup

- Clone repository.
- Build custom-debugger: `go build` in custom-debugger/.
- Build plugin: `./gradlew buildPlugin` in jetbrains-plugin/.

## Structure

- custom-debugger/: Proxy for Delve.
- jetbrains-plugin/: GoLand integration.
- replayer-adapter/: Workflow replayer.
- example/: Test workflows.

## Key Components

- WfDebugRunState.java: Manages tdlv process.
- WorkflowDebuggerPanel.java: UI for history and controls.
- WfDebuggerService.java: State management.

## Testing

- Run plugin in sandbox GoLand.
- Test with example workflows.

## Contributing

- Fork and branch from main.
- Add tests.
- Update docs.
- Submit PR. 