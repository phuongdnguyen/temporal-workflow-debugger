/**
 * Basic usage example for the Node.js Replayer Adapter
 */

import {
  ReplayMode,
  ReplayOptions,
  setReplayMode,
  setBreakpoints,
  replay
} from '@temporal/replayer-adapter-nodejs';

// Example workflow (this would typically be imported from your workflow code)
async function greetingWorkflow(name: string): Promise<string> {
  return `Hello, ${name}!`;
}

async function main() {
  try {
    console.log('=== Standalone Mode Example ===');
    
    // Set replay mode to standalone
    setReplayMode(ReplayMode.STANDALONE);
    
    // Set breakpoints at specific event IDs
    setBreakpoints([1, 5, 10]);
    
    // Configure replay options
    const standaloneOpts: ReplayOptions = {
      historyFilePath: './example-history.json',
      workerReplayOptions: {
        // Configure your worker options here
        workflowsPath: require.resolve('./workflows'),
      }
    };
    
    // Replay the workflow
    await replay(standaloneOpts, greetingWorkflow);
    console.log('Standalone replay completed successfully');
    
  } catch (error) {
    console.error('Standalone replay failed:', error);
  }

  try {
    console.log('\n=== IDE Mode Example ===');
    
    // Set replay mode to IDE integration
    setReplayMode(ReplayMode.IDE);
    
    // Configure replay options (no history file needed for IDE mode)
    const ideOpts: ReplayOptions = {
      workerReplayOptions: {
        workflowsPath: require.resolve('./workflows'),
      }
    };
    
    // Make sure WFDBG_HISTORY_PORT environment variable is set
    process.env.WFDBG_HISTORY_PORT = process.env.WFDBG_HISTORY_PORT || '54578';
    
    // Replay the workflow (will connect to IDE on the specified port)
    await replay(ideOpts, greetingWorkflow);
    console.log('IDE replay completed successfully');
    
  } catch (error) {
    console.error('IDE replay failed:', error);
  }
}

// Run the example
if (require.main === module) {
  main().catch(console.error);
} 