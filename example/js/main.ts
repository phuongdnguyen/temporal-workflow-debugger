import { Connection, Client } from '@temporalio/client';
import { Worker, NativeConnection } from '@temporalio/worker';
import * as activities from './activities';
import {
  exampleWorkflow,
  WorkflowInput,
  updateNameSignal,
  completeWorkflowSignal,
  getCurrentStatusQuery,
} from './workflow';
import { ReplayMode, setReplayMode, replay, setBreakpoints } from '@temporal/replayer-adapter-nodejs';

// ====================
// WORKER
// ====================

async function createWorker(): Promise<Worker> {
  const connection = await NativeConnection.connect({
    address: 'localhost:7233',
  });

  const worker = await Worker.create({
    connection,
    namespace: 'default',
    taskQueue: 'example-task-queue',
    workflowsPath: require.resolve('./workflow'),
    activities,
  });

  return worker;
}

// ====================
// WORKFLOW STARTER
// ====================

async function startWorkflow(): Promise<string> {
  const connection = await Connection.connect({
    address: 'localhost:7233',
  });

  const client = new Client({ connection, namespace: 'default' });

  const workflowInput: WorkflowInput = {
    initialName: 'Temporal Developer',
    numbers: { a: 10, b: 32 },
    data: 'sample workflow data',
  };

  // Start the workflow
  const handle = await client.workflow.start(exampleWorkflow, {
    args: [workflowInput],
    taskQueue: 'example-task-queue',
    workflowId: `example-workflow-${Date.now()}`,
  });

  console.log(`Started workflow with ID: ${handle.workflowId}`);

  // Demonstrate signals and queries
  setTimeout(async () => {
    console.log('Sending signal to update name...');
    await handle.signal(updateNameSignal, 'Updated Developer');

    const status = await handle.query(getCurrentStatusQuery);
    console.log(`Current status: ${status}`);

    setTimeout(async () => {
      console.log('Sending completion signal...');
      await handle.signal(completeWorkflowSignal);
    }, 3000);
  }, 2000);

  // Wait for result
  const result = await handle.result();
  console.log(`Workflow result: ${result}`);

  await connection.close();
  return result;
}

// ====================
// REPLAYER
// ====================

async function replayFromFile(historyPath: string = './history.json'): Promise<void> {
  try {
    // Configure adapter for standalone replay
    setReplayMode(ReplayMode.STANDALONE);
    setBreakpoints([9, 15])
    const opts = {
      historyFilePath: historyPath,
      workerReplayOptions: {
        workflowsPath: require.resolve('./workflow'),
        bundlerOptions: {
          ignoreModules: [
            'fs/promises',
            '@temporalio/worker',
            'http',
          ]
        }
      },
      name: 'hehe'
    } as any; // adapter types
  
    await replay(opts, exampleWorkflow);

    console.log('Replay completed successfully');
  } catch (error) {
    console.error('Replay failed:', error);
  }
}

// ====================
// MAIN EXECUTION
// ====================

async function main() {
  const command = process.argv[2];

  switch (command) {
    case 'worker':
      console.log('Starting worker...');
      const worker = await createWorker();
      console.log('Worker started. Listening for workflows...');
      await worker.run();
      break;

    case 'start':
      console.log('Starting workflow...');
      await startWorkflow();
      break;

    case 'replay':
      const historyPath = process.argv[3] || '/Users/duyphuongnguyen/GolandProjects/temporal-goland-plugin/example/js/history.json';
      console.log(`Replaying using history file ${historyPath}...`);
      await replayFromFile(historyPath);
      break;

    default:
      console.log('Usage:');
      console.log('  npm run worker                - Start the worker');
      console.log('  npm run start                 - Start a workflow');
      console.log('  npm run replay [historyPath]  - Replay from history.json or given path');
      break;
  }
}

// Run main if this file is executed directly
if (require.main === module) {
  main().catch((error) => {
    console.error('Error:', error);
    process.exit(1);
  });
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}