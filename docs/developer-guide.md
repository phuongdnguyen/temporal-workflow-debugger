# Developer Guide

## Setup

- Clone repository.
- Build custom-debugger: `go build` in custom-debugger/.
- Build plugin: `./gradlew buildPlugin` in jetbrains-plugin/.

## Structure

- custom-debugger/: Intercept message from language debugger.
- jetbrains-plugin/: GoLand Plugin.
- replayer-adapter-go/: Inject sentinel breakpoint for Temporal Go SDK.
- replayer-adapter-python/: Inject sentinel breakpoint for Temporal Python SDK.
- replayer-adapter-nodejs/: Inject sentinel breakpoint for Temporal Typescript SDK.
- example/: Test workflows.

## Testing

- Run plugin in sandbox GoLand.
- Test with example workflows.

## Contributing

- Fork and branch from main.
- Add tests.
- Update docs.
- Submit PR. 