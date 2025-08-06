import { ReplayWorkerOptions } from '@temporalio/worker';

/**
 * Replay mode for the workflow replayer
 * - STANDALONE: replay with history file
 * - IDE: replay with debugger UI integration
 */
export enum ReplayMode {
  STANDALONE = 'standalone',
  IDE = 'ide',
}

/**
 * Options for configuring the workflow replayer
 */
export interface ReplayOptions {
  /**
   * Standard Temporal worker replay options
   */
  workerReplayOptions?: ReplayWorkerOptions;

  /**
   * Path to the history JSON file (only used in STANDALONE mode)
   */
  historyFilePath?: string;

  /**
   * Replay mode for the workflow replayer
   */
  mode?: ReplayMode;

  /**
   * Breakpoint event IDs to pause execution at
   */
  breakpoints?: number[];

  /**
   * Debugger address for IDE mode
   */
  debuggerAddr?: string;
}


export var mode: ReplayMode = ReplayMode.STANDALONE;
// Initialize with empty breakpoints - users must call setBreakpoints() to set them
export var breakpoints: Set<number> = new Set<number>();
export var lastNotifiedStartEvent: number = -1;
export var debuggerAddr: string = 'http://127.0.0.1:54578';

// Initialize configuration from generated config file
function initializeFromConfig() {
  try {
    // Try to load the auto-generated config
    const config = require('./replay-config.js');
    if (config && config.replayConfig) {
      mode = config.replayConfig.mode === 'ide' ? ReplayMode.IDE : ReplayMode.STANDALONE;
      breakpoints.clear();
      config.replayConfig.breakpoints.forEach((id: number) => breakpoints.add(id));
      debuggerAddr = config.replayConfig.debuggerAddr || '';
      switch (mode) {
        case ReplayMode.IDE:
          console.log(`Initialized replay config - mode: ${mode}`);
          break;
        case ReplayMode.STANDALONE:
          console.log(`Initialized replay config - mode: ${mode}, breakpoints: [${Array.from(breakpoints).join(', ')}]`);
          break;
      }
    }
  } catch (error) {
    // Config file doesn't exist or couldn't be loaded - use defaults
    console.log('No replay config found, using defaults');
  }
}

// Initialize when module loads
initializeFromConfig();

/**
 * Set breakpoints for standalone mode
 */
export function setBreakpoints(eventIds: number[]): void {
  // Clear existing breakpoints and add new ones
  breakpoints.clear();
  eventIds.forEach(id => breakpoints.add(id));
  console.log(`Breakpoints updated to: [${eventIds.join(', ')}]`);
}

export function getBreakpoints(): Set<number> {
  return breakpoints;
}

/**
 * Set the replay mode (standalone or IDE)
 */
export function setReplayMode(m: ReplayMode): void {
  mode = m;
}

export function getReplayMode(): ReplayMode {
  return mode;
}

/**
 * Set debugger addr
 */

export function setDebuggerAddr(addr: string): void {
  debuggerAddr = addr;
}

export function getDebuggerAddr(): string {
  return debuggerAddr;
}

/**
 * Set lastNotifiedStartEvent
 */
export function setLNSE(eventId: number): void {
  lastNotifiedStartEvent = eventId;
}

export function getLNSE(): number {
  return lastNotifiedStartEvent;
}