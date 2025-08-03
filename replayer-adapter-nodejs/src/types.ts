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