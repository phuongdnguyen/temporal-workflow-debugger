# Temporal Workflow Debugger Plugin for JetBrains IDEs

This plugin provides debugging support for Temporal workflows using the custom `tdlv` debugger wrapper.

## Features

- **Automatic Build**: The plugin automatically builds the `tdlv` binary when you start debugging
- **Smart Project Detection**: Automatically detects wf-debugger projects and creates appropriate debug configurations
- **Integrated Debugging**: Seamlessly integrates with GoLand/IntelliJ's debugging interface
- **History Panel**: View workflow execution history and events

## Quick Start

1. **Open Your Project**: Open a project containing a wf-debugger setup (with `Makefile`, `tdlv/`, and `my-wf/` directories)

2. **Create Debug Configuration**: 
   - Right-click on any file in your wf-debugger project
   - Select "Create 'Debug Temporal Workflow'..." from the context menu
   - Or manually create a new "Temporal Workflow Debug" run configuration

3. **Configure Working Directory**: 
   - Set the working directory to your wf-debugger project root
   - The plugin will automatically detect and validate the project structure

4. **Start Debugging**: 
   - Click the debug button or press Ctrl+D (Cmd+D on Mac)
   - The plugin will automatically:
     - Build the `tdlv` binary using `make tdlv`
     - Start the delve wrapper process
     - Connect GoLand's debugger to the proxy

## Configuration Options

### Working Directory
- **Required**: Path to your wf-debugger project root
- **Auto-detection**: The plugin automatically finds the project root containing `Makefile`, `tdlv/`, and `my-wf/`

### Additional Arguments
- **Default**: `-p 60000` (sets proxy port)
- **Optional**: Additional command-line arguments for the `tdlv` wrapper
- **Common options**:
  - `-p PORT`: Set custom proxy port (default: 60000)
  - `-h`: Show help

## Project Structure Requirements

Your wf-debugger project should contain:
```
wf-debugger/
├── Makefile                 # Build configuration
├── tdlv/          # Custom delve wrapper source
├── my-wf/                  # Your workflow code
└── build                   # Generated binary (created automatically)
```

## Automatic Build Process

When you start debugging, the plugin:

1. **Validates Project**: Checks for required files and directories
2. **Builds Binary**: Runs `make tdlv` to build the delve wrapper
3. **Starts Process**: Executes the `tdlv` binary with appropriate arguments
4. **Connects Debugger**: Establishes connection between GoLand and the proxy

## Troubleshooting

### "Could not find wf-debugger project root"
- Ensure your working directory is within a valid wf-debugger project
- Check that `Makefile`, `tdlv/`, and `my-wf/` directories exist

### "Failed to build tdlv binary"
- Ensure you have Go installed and configured
- Check that the `tdlv/` directory contains valid Go source code
- Verify that `make` is available in your PATH

### "Go remote debug configuration not available"
- This error should no longer occur with the auto-build feature
- If it persists, check that the `tdlv` process is running on the expected port

## Manual Usage (Advanced)

If you prefer to build and run `tdlv` manually:

```bash
# Build the binary
cd /path/to/wf-debugger
make tdlv

# Run the debugger
./build
```

Then configure GoLand to connect to `localhost:60000` (or your custom port) using the "Go Remote" debug configuration. 