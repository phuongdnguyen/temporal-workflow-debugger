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
} 


export var mode: ReplayMode = ReplayMode.STANDALONE;
// TODO: make this updatable, currently hard-coded to test other things
export var breakpoints: Set<number> = new Set([9,15]);
export var lastNotifiedStartEvent: number = -1;
export var debuggerAddr: string = '';


/**
 * Set breakpoints for standalone mode
 */
export function setBreakpoints(eventIds: number[]): void {
  breakpoints = new Set(eventIds);
}

export function getBreakpoints() {
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