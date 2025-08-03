# Debugging Setup Guide

## Correct Process Order

Follow this exact sequence to avoid the "Cannot find target" error:

### Step 1: Start the DAP Server
```bash
node js-debug/js-debug/src/dapDebugServer.js 8123 127.0.0.1
```
Wait for the server to start (you should see it listening on port 8123).

### Step 2: Start the Node.js Process with Debugging
```bash
npm run debug:replay
```
This runs: `node --inspect-brk=127.0.0.1:9229 -r ts-node/register ./main.ts replay`

The process will pause at startup waiting for debugger attachment.

### Step 3: Attach the Debugger
In VSCode/Cursor, use the debug configuration:
- First try: "Fixed DAP Server Configuration"
- If that fails, try: "Attach via DAP Proxy"

## Troubleshooting

### Check if processes are running:
```bash
# Check DAP server
lsof -i :8123

# Check Node.js debug port
lsof -i :9229
```

### Verify DAP server logs:
The DAP server should show connection attempts and target registrations.

### Alternative: Use the working configuration
If DAP server continues to fail, use the "Working" configuration which directly attaches to port 9229.

## Common Issues

1. **Target ID not found**: Usually means timing issue or DAP server not properly forwarding
2. **Connection refused**: DAP server not running or wrong port
3. **Process exits immediately**: Node.js process not waiting for debugger (check --inspect-brk flag) 