import * as fs from 'fs/promises';
import { Worker } from '@temporalio/worker';
import { historyFromJSON } from '@temporalio/common/lib/proto-utils';
import { temporal } from '@temporalio/proto';
import { getLNSE, ReplayMode, ReplayOptions, setDebuggerAddr, setLNSE, getBreakpoints, getReplayMode, getDebuggerAddr, setReplayMode, setBreakpoints } from './types';
import { httpGet } from './http-client';
import { BreakpointManager, fetchBreakpointsFromWorkflow, sendHighlightFromWorkflow, destroyWorkerThread } from './breakpoint-manager.js';


/**
 * Get workflow history from IDE via HTTP
 */
export async function getHistoryFromIDE(): Promise<temporal.api.history.v1.IHistory> {
  const addr = process.env.TEMPORAL_DEBUGGER_PLUGIN_URL || `http://127.0.0.1:54578`;

  try {
    setDebuggerAddr(addr);
    const response = await httpGet(`${getDebuggerAddr()}/history`);
    if (response.statusCode !== 200) {
      throw new Error(`HTTP error! status: ${response.statusCode}`);
    }

    return JSON.parse(response.body);
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
    (globalThis as any).sendHighlightFromWorkflow = sendHighlightFromWorkflow;
    (globalThis as any).destroyWorkerThread = () => destroyWorkerThread;

    const hist = await getHistoryFromIDE();
    await replayWithHistory(opts.workerReplayOptions || {}, hist, workflow, opts);
    destroyWorkerThread()
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
