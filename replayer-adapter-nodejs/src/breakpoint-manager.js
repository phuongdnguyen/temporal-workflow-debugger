/**
 * Breakpoint manager that uses worker threads to fetch breakpoints synchronously
 * from within workflow interceptors.
 */
const { Worker } = require('node:worker_threads');
const path = require('node:path');
// Note: HTTP calls are delegated to the worker thread

let breakpointThread = undefined;

class BreakpointManager {
  static _instance = undefined;
  
  /**
   * Get singleton instance
   */
  static instance() {
    if (!this._instance) {
      this._instance = new this();
    }
    return this._instance;
  }
  
  constructor() {
    // Create worker thread once per process
    if (!breakpointThread) {
      const workerPath = path.resolve(__dirname, 'breakpoint-fetcher-thread.js');
      breakpointThread = new Worker(workerPath);
      // Handle worker errors
      breakpointThread.on('error', (error) => {
        console.error('Breakpoint worker thread error:', error);
      });
      
      breakpointThread.on('exit', (code) => {
        if (code !== 0) {
          // Need a cleaner way to shut down
          console.info(`Breakpoint worker thread stopped with exit code ${code}`);
        }
      });
    }
  }
  
  /**
   * Synchronously fetch breakpoints from IDE using worker thread.
   * This can be called from workflow interceptors!
   * 
   * @param {string} debuggerAddr - The debugger address to fetch breakpoints from
   * @returns {number[]} Array of breakpoint event IDs
   */
  fetchBreakpointsSync(debuggerAddr) {
    if (!debuggerAddr) {
      console.warn('No debugger address provided to fetchBreakpointsSync');
      return [];
    }
    
    try {
      // Create shared buffers for communication
      const responseSab = new SharedArrayBuffer(4);
      const responseBuffer = new Int32Array(responseSab);
      
      // 2KB buffer for breakpoint data should be plenty
      const dataSab = new SharedArrayBuffer(2048);
      
      // Send request to worker thread
      breakpointThread.postMessage({
        type: 'fetch-breakpoints',
        debuggerAddr,
        responseBuffer,
        dataBuffer: dataSab
      });
      
      // Wait synchronously for response (this blocks the current thread)
      Atomics.wait(responseBuffer, 0, 0);
      
      if (responseBuffer[0] === 2) {
        console.error('Worker thread failed to fetch breakpoints');
        return [];
      }
      
      // Read breakpoint data from shared buffer
      const lengthView = new Uint32Array(dataSab, 0, 1);
      const length = lengthView[0];
      
      if (length === 0) {
        return [];
      }
      
      const dataView = new Uint8Array(dataSab, 4, length);
      const breakpointData = Buffer.from(dataView).toString('utf8');
      
      const breakpoints = JSON.parse(breakpointData);
      console.log(`Fetched ${breakpoints.length} breakpoints synchronously: [${breakpoints.join(', ')}]`);
      return breakpoints;
      
    } catch (error) {
      console.error('Error in fetchBreakpointsSync:', error);
      return [];
    }
  }
  
  // /**
  //  * Clean up worker thread
  //  */
  // destroy() {
  //   if (breakpointThread) {
  //     breakpointThread.terminate();
  //     breakpointThread = undefined;
  //   }
  // }

  /**
   * Send a highlight request to IDE to focus current event in UI.
   * This runs outside of the workflow sandbox.
   *
   * @param {string} debuggerAddr - The debugger address
   * @param {number} eventId - The current event ID to highlight
   */
  highlightEvent(debuggerAddr, eventId) {
    console.log(
      `breakpoint-manager.highlightEvent, debuggerAddr: ${debuggerAddr} eventId: ${eventId}`
    );
    if (!debuggerAddr) {
      console.warn('No debugger address provided to highlightEvent');
      return;
    }

    try {
      // Create a small shared buffer to receive completion status
      const responseSab = new SharedArrayBuffer(4);
      const responseBuffer = new Int32Array(responseSab);

      // Delegate to worker thread to perform the HTTP POST
      breakpointThread.postMessage({
        type: 'highlight-event',
        debuggerAddr,
        eventId,
        responseBuffer,
      });

      // Wait synchronously for worker to complete
      Atomics.wait(responseBuffer, 0, 0);

      if (responseBuffer[0] === 2) {
        console.warn('Worker thread failed to send highlight request');
      }
    } catch (error) {
      console.warn(`Failed to send highlight request via worker: ${error}`);
    }
  }
}

/**
 * Global function that can be called from workflow context.
 * This breaks out of the workflow sandbox to access the breakpoint manager.
 * 
 * @param {string} debuggerAddr - The debugger address
 * @returns {number[]} Array of breakpoint event IDs
 */
function fetchBreakpointsFromWorkflow(debuggerAddr) {
  try {
    // Break out of workflow context to access the breakpoint manager
    // This uses the same technique as the TypeScript SDK debug replayer
    // Modified to work with ESM by capturing the BreakpointManager reference
    const manager = BreakpointManager.instance();
    const fetchFn = globalThis.constructor.constructor(`
      return (debuggerAddr, manager) => manager.fetchBreakpointsSync(debuggerAddr);
    `)();
    return fetchFn(debuggerAddr, manager);
  } catch (error) {
    console.error('Failed to fetch breakpoints from workflow context:', error);
    return [];
  }
}

/**
 * Global function to send highlight request from workflow context.
 * Breaks out of the workflow sandbox to the manager running in Node.
 *
 * @param {string} debuggerAddr
 * @param {number} eventId
 */
function sendHighlightFromWorkflow(debuggerAddr, eventId) {
  console.log("breakpoint-manager.sendHighlightFromWorkflow")
  try {
    const manager = BreakpointManager.instance();
    const sendFn = globalThis.constructor.constructor(`
      return (debuggerAddr, eventId, manager) => manager.highlightEvent(debuggerAddr, eventId);
    `)();
    return sendFn(debuggerAddr, eventId, manager);
  } catch (error) {
    console.error('Failed to send highlight from workflow context:', error);
  }
}


/**
 * Clean up worker thread
 */
function destroyWorkerThread() {
  console.log("Destroying worker thread..")
    if (breakpointThread) {
      breakpointThread.terminate();
      breakpointThread = undefined;
    }
}

module.exports = {
  BreakpointManager,
  fetchBreakpointsFromWorkflow,
  sendHighlightFromWorkflow,
  destroyWorkerThread
}; 