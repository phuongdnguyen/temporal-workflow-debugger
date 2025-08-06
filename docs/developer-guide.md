# Developer Guide

## Setup

- Clone repository.
- Build custom-debugger: `go build` in custom-debugger/.
- Build Jetbrainsplugin: see [jetbrains plugin readme](../jetbrains-plugin/README.md)
- Build vscode extension: see [vscode extension readme](../vscode-debugger-extension/README.md)
## Structure

- custom-debugger/: Intercept message from language debugger.
- jetbrains-plugin/: Jetbrains Plugin (support Go).
- vscode-debugger-extension: Vscode Extension (support Go, Python, Js/TS).
- replayer-adapter-go/: Inject sentinel breakpoints for Temporal Go SDK.
- replayer-adapter-python/: Inject sentinel breakpoints for Temporal Python SDK.
- replayer-adapter-nodejs/: Inject sentinel breakpoints for Temporal Typescript SDK.
- example/: Test workflows.

## Contributing

- Fork and branch from main.
- Add tests.
- Update docs.
- Submit PR. 