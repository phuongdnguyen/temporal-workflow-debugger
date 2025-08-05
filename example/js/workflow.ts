import { proxyActivities, defineSignal, defineQuery, condition, setHandler } from '@temporalio/workflow';

// Define signals and queries
export const updateNameSignal = defineSignal<[string]>('updateName');
export const completeWorkflowSignal = defineSignal('completeWorkflow');
export const getCurrentStatusQuery = defineQuery<string>('getCurrentStatus');

// Define activity types for type safety
export interface Activities {
  greetActivity(name: string): Promise<string>;
  calculateActivity(a: number, b: number): Promise<number>;
  processDataActivity(data: string): Promise<string>;
}

// Proxy activities for use in workflow
const activities = proxyActivities<Activities>({
  startToCloseTimeout: '1 minute',
  retry: {
    initialInterval: '1s',
    maximumInterval: '10s',
    maximumAttempts: 3,
  },
});

export interface WorkflowInput {
  initialName: string;
  numbers: { a: number; b: number };
  data: string;
}

export async function exampleWorkflow(input: WorkflowInput): Promise<string> {
  let currentName = input.initialName;
  let status = 'started';
  let shouldComplete = false;

  // Set up signal and query handlers
  setHandler(updateNameSignal, (newName: string) => {
    currentName = newName;
    status = `name updated to ${newName}`;
  });

  setHandler(completeWorkflowSignal, () => {
    shouldComplete = true;
    status = 'completion requested';
  });

  setHandler(getCurrentStatusQuery, () => status);

  try {
    // Step 1: Greet
    status = 'greeting';
    // Event 9
    const greeting = await activities.greetActivity(currentName);
    console.log(`Workflow: ${greeting}`);

    // Step 2: Calculate
    // Event 15
    status = 'calculating';
    const result = await activities.calculateActivity(input.numbers.a, input.numbers.b);
    console.log(`Workflow: Calculation result is ${result}`);

    // Step 3: Process data
    status = 'processing data';
    const processedData = await activities.processDataActivity(input.data);
    console.log(`Workflow: ${processedData}`);

    // Step 4: Wait for completion signal or timeout
    status = 'waiting for completion';
    await condition(() => shouldComplete, '30s');

    status = 'completed';
    return `Workflow completed successfully! Greeting: ${greeting}, Result: ${result}, Data: ${processedData}`;
  } catch (error) {
    status = `failed: ${error}`;
    throw error;
  }
} 