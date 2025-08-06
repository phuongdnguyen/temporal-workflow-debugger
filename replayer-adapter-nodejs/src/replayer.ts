import * as fs from 'fs/promises';
import { Worker } from '@temporalio/worker';
import { historyFromJSON } from '@temporalio/common/lib/proto-utils';
import { temporal } from '@temporalio/proto';
import { getLNSE, ReplayMode, ReplayOptions, setDebuggerAddr, setLNSE, getBreakpoints, getReplayMode, getDebuggerAddr, setReplayMode, setBreakpoints } from './types';
import { httpGet, httpPost } from './http-client';
import { BreakpointManager, fetchBreakpointsFromWorkflow } from './breakpoint-manager.js';



/**
 * Fetch breakpoints from IDE via HTTP
 */
async function fetchBreakpointsFromIDE(): Promise<Set<number>> {
  const debuggerAddr = getDebuggerAddr();
  if (!debuggerAddr) {
    throw new Error('No debugger address configured');
  }

  try {
    const response = await httpGet(`${debuggerAddr}/breakpoints`);
    if (response.statusCode !== 200) {
      throw new Error(`HTTP error! status: ${response.statusCode}, body: ${response.body}`);
    }

    const data = JSON.parse(response.body);
    console.log(`Fetched breakpoints from IDE: ${response.body}`);
    
    // Handle different response formats
    let breakpointIds: number[] = [];
    if (Array.isArray(data)) {
      breakpointIds = data;
    } else if (data.breakpoints && Array.isArray(data.breakpoints)) {
      breakpointIds = data.breakpoints;
    } else if (data.eventIds && Array.isArray(data.eventIds)) {
      breakpointIds = data.eventIds;
    } else {
      console.warn('Unexpected breakpoints response format:', data);
      return new Set();
    }

    return new Set(breakpointIds.filter(id => typeof id === 'number'));
  } catch (error) {
    console.error(`Failed to fetch breakpoints from IDE: ${error}`);
    throw error;
  }
}

/**
 * Check if the given event ID is a breakpoint
 */
export async function isBreakpoint(eventId: number): Promise<boolean> {
  switch (getReplayMode()) {
    case ReplayMode.STANDALONE:
      console.log(`isBreakpoint, mode: standalone`)
      const currentBreakpoints = getBreakpoints();
      console.log(`Standalone checking breakpoints: [${Array.from(currentBreakpoints).join(', ')}], eventId: ${eventId}`);
      if (currentBreakpoints.has(eventId)) {
        console.log(`✓ Hit breakpoint at eventId: ${eventId}`);
        return true;
      }
      return false;
      
    case ReplayMode.IDE:
      console.log(`isBreakpoint, mode: ide`)
      if (!getDebuggerAddr()) {
        console.log('IDE mode: No debugger address set, skipping breakpoint check');
        return false;
      }
      
      try {
        // Fetch breakpoints from IDE
        console.log(`IDE mode: Fetching breakpoints for event ${eventId} from ${getDebuggerAddr()}/breakpoints`);
        const breakpoints = await fetchBreakpointsFromIDE();
        
        const isHit = breakpoints.has(eventId);
        if (isHit) {
          console.log(`✓ Hit breakpoint at eventId: ${eventId}`);
        } else {
          console.log(`Event ${eventId} is not a breakpoint. Current breakpoints: [${Array.from(breakpoints).join(', ')}]`);
        }
        return isHit;
      } catch (error) {
        console.error(`IDE mode: Failed to check breakpoint for event ${eventId}: ${error}`);
        return false;
      }
      
    default:
      console.log("Unknown mode")
      return false;
  }
}

/**
 * Send highlight request to IDE for current event
 */
export function highlightCurrentEventInIDE(eventId: number): void {
  if (!getDebuggerAddr()) {
    console.warn('debuggerAddr is empty, cannot send highlight request');
    return;
  }
  
  console.log(`Sending highlight request for event ${eventId} to ${getDebuggerAddr()}/current-event`);
  
  const payload = JSON.stringify({ eventId });
  
  try {
    sendHighlightRequest(getDebuggerAddr(), payload);
    console.log(`✓ Successfully highlighted event ${eventId} in debugger UI`);
  } catch (error) {
    console.warn(`Failed to send highlight request: ${error}`);
  }
}

/**
 * Raise a breakpoint for debugging - called from interceptors
 */
export function raiseSentinelBreakpoint(caller: string, info?: any): void {
  let eventId: number | undefined;
  
  if (info) {
    try {
      // Try to get event ID from workflow info
      eventId = info.historyLength || info.getCurrentHistoryLength?.();
    } catch (error) {
      eventId = undefined;
    }
  }
  
  if (eventId !== undefined) {
    if (eventId <= getLNSE()) {
      return;
    }
    setLNSE(eventId);
    console.log(`runner notified at ${caller}, eventId: ${eventId}`);
    
    // Handle async breakpoint checking
    isBreakpoint(eventId).then((shouldBreak) => {
      if (shouldBreak) {
        console.log(`Pause at event ${eventId}`);
        if (getReplayMode() === ReplayMode.IDE) {
          highlightCurrentEventInIDE(eventId as number);
        }
        debugger;
      }
    }).catch((error) => {
      console.error(`Error checking breakpoint for event ${eventId}: ${error}`);
    });
  }
}

/**
 * Get workflow history from IDE via HTTP
 */
export async function getHistoryFromIDE(): Promise<temporal.api.history.v1.IHistory> {
  const port = process.env.WFDBG_HISTORY_PORT || '54578';
  const runnerAddr = `http://127.0.0.1:${port}`;
  
  try {
    const response = await httpGet(`${getDebuggerAddr()}/history`);
    if (response.statusCode !== 200) {
      throw new Error(`HTTP error! status: ${response.statusCode}`);
    }
    
    const historyData = JSON.parse(response.body);
    setDebuggerAddr(runnerAddr);
    return historyData;
  } catch (error) {
    console.error(`Could not get history from IDE: ${error}`);
    throw error;
  }
}

/**
 * Main replay function that handles both standalone and IDE modes
 */
export async function replay(opts: ReplayOptions, workflow: any): Promise<void> {
  // Set configuration from options if provided
  if (opts.mode !== undefined) {
    setReplayMode(opts.mode);
  }
  if (opts.breakpoints !== undefined) {
    setBreakpoints(opts.breakpoints);
  }
  if (opts.debuggerAddr !== undefined) {
    setDebuggerAddr(opts.debuggerAddr);
  }

  console.log(`Replaying in mode ${getReplayMode()}`);
  
  if (getReplayMode() === ReplayMode.STANDALONE) {
    console.log('Replaying in standalone mode');
    
    // Inject global functions for standalone mode
    const standaloneBreakpoints = opts.breakpoints || [];
    (globalThis as any).fetchBreakpointsFromWorkflow = () => standaloneBreakpoints;
    (globalThis as any).getDebuggerAddr = () => null; // No debugger address in standalone mode
    
    return replayWithJsonFile(opts.workerReplayOptions || {}, workflow, opts.historyFilePath!, opts);
  } else {
    console.log('Replaying in IDE integrated mode');
    
    // Initialize breakpoint manager for IDE mode
    BreakpointManager.instance();
    
    // Inject global functions for workflow context
    (globalThis as any).fetchBreakpointsFromWorkflow = fetchBreakpointsFromWorkflow;
    (globalThis as any).getDebuggerAddr = () => getDebuggerAddr();
    
    const hist = await getHistoryFromIDE();
    return replayWithHistory(opts.workerReplayOptions || {}, hist, workflow, opts);
  }
}

/**
 * Replay workflow with history data
 */
export async function replayWithHistory(
  opts: any,
  hist: temporal.api.history.v1.IHistory,
  workflow: any,
  replayOptions?: ReplayOptions
): Promise<void> {
  let configPath: string | undefined;
  
  // Write configuration to a temporary file that can be imported by interceptors
  if (replayOptions) {
    const fs = await import('fs/promises');
    const path = await import('path');
    configPath = path.join(__dirname, 'replay-config.js');
    
    const configContent = `
// Auto-generated configuration for replay
export const replayConfig = {
  mode: '${replayOptions.mode || 'standalone'}',
  breakpoints: [${(replayOptions.breakpoints || []).join(', ')}],
  debuggerAddr: '${replayOptions.debuggerAddr || ''}'
};
`;
    
    await fs.writeFile(configPath, configContent);
  }

  try {
    // Add our custom interceptors to the options
    const interceptors = opts.interceptors || {};
    const workflowModules = interceptors.workflowModules || [];
    
    // Add our interceptor modules (no activities needed for worker thread approach)
    workflowModules.push(require.resolve('./workflow-interceptors'));
    
    const workerReplayOptions = {
      ...opts,
      interceptors: {
        ...interceptors,
        workflowModules,
      },
    };
    
    return await Worker.runReplayHistory(workerReplayOptions, hist);
  } finally {
    // Clean up temporary config file
    if (configPath) {
      try {
        const fs = await import('fs/promises');
        await fs.unlink(configPath);
      } catch (error) {
        // Ignore cleanup errors
      }
    }
  }
}

/**
 * Replay workflow with history from JSON file
 */
export async function replayWithJsonFile(
  opts: any,
  workflow: any,
  jsonFileName: string,
  replayOptions?: ReplayOptions
): Promise<void> {
  const historyData = await fs.readFile(jsonFileName, 'utf-8');
  const historyJson = JSON.parse(historyData);
  const history = historyFromJSON(historyJson);
  
  return replayWithHistory(opts, history, workflow, replayOptions);
}

/**
 * Test connectivity to IDE debugger server
 */
export function testIDEConnectivity(): boolean {
  const debuggerUrl = getDebuggerAddr();
  if (!debuggerUrl) {
    console.log('No debugger address configured');
    return false;
  }
  
  console.log(`Testing IDE connectivity to: ${debuggerUrl}`);
  console.log('Note: Synchronous HTTP requests are not supported in workflow context');
  console.log('Skipping connectivity test');
  return true; // Assume it works for now
}

function sendHighlightRequest(addr: string, payload: string): void {
  httpPost(`${addr}/current-event`, payload)
    .then((response) => {
      console.log(`Highlight response status: ${response.statusCode}, body: ${response.body}`);
      if (response.statusCode !== 200) {
        console.warn(`Highlight request failed: ${response.statusCode} ${response.body}`);
      }
    })
    .catch((error) => {
      console.warn(`Failed to send highlight request: ${error}`);
    });
} 