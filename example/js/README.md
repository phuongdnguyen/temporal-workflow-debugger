# Temporal TypeScript Example

This example demonstrates a complete Temporal workflow setup including:
- **Workflow Definition**: A multi-step workflow with activities, signals, and queries
- **Worker**: A worker that executes workflows and activities  
- **Workflow Starter**: A client that starts and interacts with workflows
- **Replayer**: Functionality to replay workflow history for testing and debugging

## Prerequisites

1. **Temporal Server**: Make sure you have Temporal server running locally
   ```bash
   # Using Temporal CLI (recommended)
   temporal server start-dev
   
   # Or using Docker
   git clone https://github.com/temporalio/docker-compose.git
   cd docker-compose
   docker-compose up
   ```

2. **Node.js**: Version 16 or higher

## Setup

1. Install dependencies:
   ```bash
   npm install
   ```

2. Build the TypeScript code:
   ```bash
   npm run build
   ```

## Usage

### 1. Start the Worker

In one terminal, start the worker to listen for workflow tasks:

```bash
npm run worker
```

You should see:
```
Starting worker...
Worker started. Listening for workflows...
```

### 2. Start a Workflow

In another terminal, start a workflow:

```bash
npm run start
```

This will:
- Start a new workflow with a unique ID
- Send signals to update the workflow state
- Query the workflow status
- Wait for the workflow to complete
- Display the final result

Expected output:
```
Starting workflow...
Started workflow with ID: example-workflow-1234567890
Sending signal to update name...
Current status: name updated to Updated Developer
Sending completion signal...
Workflow result: Workflow completed successfully! Greeting: Hello, Updated Developer!, Result: 42, Data: Processed: SAMPLE WORKFLOW DATA
```

### 3. Replay a Workflow

To replay a completed workflow for testing/debugging:

```bash
npm run replay <workflow-id>
```

For example:
```bash
npm run replay example-workflow-1234567890
```

## Workflow Features

### Activities
- **greetActivity**: Simple greeting with a name
- **calculateActivity**: Performs a calculation with retry logic
- **processDataActivity**: Processes and transforms data

### Signals
- **updateNameSignal**: Updates the name used in the workflow
- **completeWorkflowSignal**: Triggers workflow completion

### Queries
- **getCurrentStatusQuery**: Returns the current workflow status

### Workflow Flow
1. **Greeting**: Calls the greet activity with the initial name
2. **Calculation**: Performs a mathematical operation
3. **Data Processing**: Transforms input data
4. **Wait for Completion**: Waits for a signal or timeout (30 seconds)
5. **Return Result**: Combines all results into a final message

## Advanced Usage

### Direct TypeScript Execution

You can also run the commands directly with ts-node:

```bash
# Start worker
npx ts-node main.ts worker

# Start workflow  
npx ts-node main.ts start

# Replay workflow
npx ts-node main.ts replay <workflow-id>
```

### Customizing the Example

The workflow accepts input parameters that you can modify in the `startWorkflow()` function:

```typescript
const workflowInput: WorkflowInput = {
  initialName: 'Your Name Here',
  numbers: { a: 5, b: 10 },
  data: 'your custom data',
};
```

### Integration with Replayer Adapter

This example is designed to work with the `replayer-adapter-nodejs` package for enhanced replay capabilities. You can integrate it by:

1. Installing the replayer adapter:
   ```bash
   npm install ../../../replayer-adapter-nodejs
   ```

2. Using the replayer adapter in your workflow code for advanced debugging features.

## Project Structure

```
example/js/
├── main.ts           # Complete workflow example
├── package.json      # Dependencies and scripts
├── tsconfig.json     # TypeScript configuration
├── README.md         # This file
└── dist/            # Compiled JavaScript (after build)
```

## Troubleshooting

1. **Connection Errors**: Make sure Temporal server is running on `localhost:7233`
2. **Build Errors**: Ensure you have TypeScript installed and run `npm run build`
3. **Worker Not Receiving Tasks**: Check that both worker and client use the same task queue name
4. **Signal/Query Errors**: Ensure the workflow is running before sending signals or queries

## Next Steps

- Explore the [Temporal TypeScript SDK documentation](https://docs.temporal.io/typescript)
- Check out more [Temporal samples](https://github.com/temporalio/samples-typescript)
- Learn about [Temporal best practices](https://docs.temporal.io/typescript/best-practices) 