# Background Process Support

The Temporal debugger extension now supports running a background process before starting the debug session. This process will be automatically terminated when the debug session ends.

## Configuration

Configure the background process through VS Code settings:

### Settings

- `temporal.debugger.backgroundProcess.command`: The command to run (e.g., "npm", "go", "python", "docker")
- `temporal.debugger.backgroundProcess.args`: Array of arguments to pass to the command
- `temporal.debugger.backgroundProcess.options`: Additional options like working directory and environment variables

## Examples

### Starting a Node.js Server

```json
{
  "temporal.debugger.backgroundProcess.command": "npm",
  "temporal.debugger.backgroundProcess.args": ["run", "start"],
  "temporal.debugger.backgroundProcess.options": {
    "cwd": "./server",
    "env": {
      "PORT": "3000",
      "NODE_ENV": "development"
    }
  }
}
```

### Starting a Go Server

```json
{
  "temporal.debugger.backgroundProcess.command": "go",
  "temporal.debugger.backgroundProcess.args": ["run", "main.go"],
  "temporal.debugger.backgroundProcess.options": {
    "cwd": "./cmd/server"
  }
}
```

### Starting a Python Server

```json
{
  "temporal.debugger.backgroundProcess.command": "python",
  "temporal.debugger.backgroundProcess.args": ["server.py"],
  "temporal.debugger.backgroundProcess.options": {
    "env": {
      "PYTHONPATH": "./src"
    }
  }
}
```

### Starting a Docker Container

```json
{
  "temporal.debugger.backgroundProcess.command": "docker",
  "temporal.debugger.backgroundProcess.args": ["run", "-d", "--name", "temporal-server", "-p", "7233:7233", "temporalio/temporal-server:latest"]
}
```

## How It Works

1. When you start a debug session, the extension checks if a background process is configured
2. If configured, it starts the background process before launching the debugger
3. The background process runs alongside your debug session
4. When the debug session ends (either by completion or termination), the background process is automatically cleaned up

## Process Management

- The extension uses graceful termination (SIGTERM) first, then forceful termination (SIGKILL) if needed
- Process output is logged to the VS Code console
- If the background process fails to start, debugging continues with a warning message
- Multiple debug sessions will terminate and restart the background process

## Use Cases

This feature is useful for:
- Starting a Temporal server before debugging
- Launching dependent services (databases, message queues, etc.)
- Running setup scripts or initialization processes
- Starting mock servers or test environments 