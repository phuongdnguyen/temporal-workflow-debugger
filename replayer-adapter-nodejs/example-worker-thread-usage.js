/**
 * Example usage of the worker thread-based breakpoint fetching approach
 */

const { replay, ReplayMode } = require('./src/replayer');

// Example workflow (this would normally be imported from your workflow file)
async function exampleWorkflow() {
  console.log('Workflow started');
  // Your workflow logic here
  return 'Workflow completed';
}

async function runReplay() {
  try {
    // Configure for IDE mode
    const replayOptions = {
      mode: ReplayMode.IDE,
      debuggerAddr: 'http://localhost:54578', // Your IDE debugger address
      workerReplayOptions: {
        workflowsPath: require.resolve('./path/to/your/workflows'), // Update this path
        // Other worker options...
      },
      // For standalone mode, you would also set:
      // historyFilePath: './path/to/history.json'
    };

    console.log('Starting replay with worker thread breakpoint fetching...');
    await replay(replayOptions, exampleWorkflow);
    console.log('Replay completed successfully!');
    
  } catch (error) {
    console.error('Replay failed:', error);
  }
}

// Run the example
if (require.main === module) {
  runReplay();
}

module.exports = { runReplay }; 