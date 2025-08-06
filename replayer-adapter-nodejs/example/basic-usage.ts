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
    
    // IMPORTANT: Set replay mode and breakpoints BEFORE calling replay()
    setReplayMode(ReplayMode.STANDALONE);
    
    // Set breakpoints at specific event IDs
    // These should correspond to actual event IDs in your workflow history
    setBreakpoints([1, 5, 10]);
    console.log('✓ Breakpoints configured for events: [1, 5, 10]');
    
    // Configure replay options
    const standaloneOpts: ReplayOptions = {
      historyFilePath: './example-history.json',
      workerReplayOptions: {
        // Configure your worker options here
        workflowsPath: require.resolve('./workflows'),
      }
    };
    
    console.log('Starting replay with breakpoints...');
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
    
    // In IDE mode, breakpoints are managed by the IDE, not set manually
    console.log('✓ IDE mode configured - breakpoints will be managed by the IDE');
    
    // Configure replay options (no history file needed for IDE mode)
    const ideOpts: ReplayOptions = {
      workerReplayOptions: {
        workflowsPath: require.resolve('./workflows'),
      }
    };
    
    // Make sure WFDBG_HISTORY_PORT environment variable is set
    process.env.WFDBG_HISTORY_PORT = process.env.WFDBG_HISTORY_PORT || '54578';
    console.log(`IDE will connect on port: ${process.env.WFDBG_HISTORY_PORT}`);
    
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