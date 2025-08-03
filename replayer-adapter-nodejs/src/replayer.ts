import * as fs from 'fs/promises';
import { Worker } from '@temporalio/worker';
import { historyFromJSON } from '@temporalio/common/lib/proto-utils';
import { temporal } from '@temporalio/proto';
import { workflowInfo } from '@temporalio/workflow';
import { ReplayMode, ReplayOptions } from './types';
import { httpGet, httpPost } from './http-client';
import { activityInterceptors } from './activity-interceptors';

// Global state
let mode: ReplayMode = ReplayMode.STANDALONE;
// TODO: make this updatable, currently hard-coded to test other things
let breakpoints: Set<number> = new Set([9,15]);
let lastNotifiedStartEvent: number = -1;
let debuggerAddr: string = '';

/**
 * Set the replay mode (standalone or IDE)
 */
export function setReplayMode(m: ReplayMode): void {
  mode = m;
}

/**
 * Set breakpoints for standalone mode
 */
export function setBreakpoints(eventIds: number[]): void {
  breakpoints = new Set(eventIds);
}

/**
 * Check if the given event ID is a breakpoint
 */
export function isBreakpoint(eventId: number): boolean {
  switch (mode) {
    case ReplayMode.STANDALONE:
      console.log(`Standalone checking breakpoints: ${Array.from(breakpoints)}, eventId: ${eventId}`);
      if (breakpoints.has(eventId)) {
        console.log(`Hit breakpoint at eventId: ${eventId}`);
        return true;
      }
      return false;
      
    case ReplayMode.IDE:
      if (!debuggerAddr) {
        return false;
      }
      
      try {
        // Make async HTTP request to check breakpoints
        return checkBreakpointWithIDE(eventId);
      } catch (error) {
        console.warn(`Could not get breakpoints from IDE: ${error}`);
        return false;
      }
      
    default:
      return false;
  }
}

/**
 * Send highlight request to IDE for current event
 */
export function highlightCurrentEventInIDE(eventId: number): void {
  if (!debuggerAddr) {
    console.warn('debuggerAddr is empty, cannot send highlight request');
    return;
  }
  
  console.log(`Sending highlight request for event ${eventId} to ${debuggerAddr}/current-event`);
  
  const payload = JSON.stringify({ eventId });
  
  try {
    sendHighlightRequest(debuggerAddr, payload);
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
    if (eventId <= lastNotifiedStartEvent) {
      return;
    }
    lastNotifiedStartEvent = eventId;
    console.log(`runner notified at ${caller}, eventId: ${eventId}`);
    
    if (isBreakpoint(eventId)) {
      console.log(`Pause at event ${eventId}`);
      if (mode === ReplayMode.IDE) {
        highlightCurrentEventInIDE(eventId);
      }
        debugger;
    }
  }
}

/**
 * Get workflow history from IDE via HTTP
 */
export async function getHistoryFromIDE(): Promise<temporal.api.history.v1.IHistory> {
  const port = process.env.WFDBG_HISTORY_PORT || '54578';
  const runnerAddr = `http://127.0.0.1:${port}`;
  
  try {
    const response = await httpGet(`${runnerAddr}/history`);
    if (response.statusCode !== 200) {
      throw new Error(`HTTP error! status: ${response.statusCode}`);
    }
    
    const historyData = JSON.parse(response.body);
    debuggerAddr = runnerAddr;
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
  console.log(`Replaying in mode ${mode}`);
  
  if (mode === ReplayMode.STANDALONE) {
    console.log('Replaying in standalone mode');
    return replayWithJsonFile(opts.workerReplayOptions || {}, workflow, opts.historyFilePath!);
  } else {
    console.log('Replaying in IDE integrated mode');
    const hist = await getHistoryFromIDE();
    return replayWithHistory(opts.workerReplayOptions || {}, hist, workflow);
  }
}

/**
 * Replay workflow with history data
 */
export async function replayWithHistory(
  opts: any,
  hist: temporal.api.history.v1.IHistory,
  workflow: any
): Promise<void> {
  // Add our custom interceptors to the options
  const interceptors = opts.interceptors || {};
  const workflowModules = interceptors.workflowModules || [];
  const activity = interceptors.activity || [];
  
  // Add our interceptor modules
  workflowModules.push(require.resolve('./workflow-interceptors'));
  activity.push(activityInterceptors);
  
  const replayOptions = {
    ...opts,
    interceptors: {
      ...interceptors,
      workflowModules,
      activity,
    },
  };
  
  return Worker.runReplayHistory(replayOptions, hist);
}

/**
 * Replay workflow with history from JSON file
 */
export async function replayWithJsonFile(
  opts: any,
  workflow: any,
  jsonFileName: string
): Promise<void> {
  const historyData = await fs.readFile(jsonFileName, 'utf-8');
  const historyJson = JSON.parse(historyData);
  const history = historyFromJSON(historyJson);
  
  return replayWithHistory(opts, history, workflow);
}

// Helper functions for IDE communication
function checkBreakpointWithIDE(eventId: number): boolean {
  try {
    // This should be an async call in practice, but for simplicity keeping it sync
    // In a production implementation, you'd want to cache breakpoints or make this async
    const response = httpGet(`${debuggerAddr}/breakpoints`, 2000);
    response.then((res) => {
      if (res.statusCode === 200) {
        const payload = JSON.parse(res.body);
        return payload.breakpoints?.includes(eventId) || false;
      }
      return false;
    }).catch(() => false);
    return false; // Default to false for sync implementation
  } catch (error) {
    return false;
  }
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