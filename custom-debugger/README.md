# Custom Debugger

A multi-language debugger for Temporal workflows that supports Go, Python, and JavaScript.

## Overview

The custom debugger provides debugging capabilities for Temporal workflows by:
- Starting appropriate language-specific debuggers (Delve for Go, DebugPy for Python, JS Debug for JavaScript)
- Acting as a proxy between IDEs and language debuggers
- Supporting both DAP (Debug Adapter Protocol) and JSON-RPC protocols
- Enabling workflow debugging from history files

## Usage

```bash
# Start with default settings (Go language, port 60000)
./custom-debugger

# Specify language and port
./custom-debugger -lang python -p 60001

# Auto-install missing debuggers
./custom-debugger -lang js -install

# Suppress dependency check messages
./custom-debugger -quiet
```

## Supported Languages

- **Go**: Uses Delve debugger
- **Python**: Uses DebugPy debugger  
- **JavaScript**: Uses VS Code JS Debug

## Ports

- **60000**: Main debugger proxy port (configurable with `-p`)
- **2345**: Language-specific debugger port

## Dependencies

The debugger automatically checks for and can install required language-specific debuggers:
- Delve for Go workflows
- DebugPy for Python workflows  
- JS Debug for JavaScript workflows 