/**
 * TypeScript declarations for breakpoint-manager.js
 */

export class BreakpointManager {
  static instance(): BreakpointManager;
  fetchBreakpointsSync(debuggerAddr: string): number[];
  destroy(): void;
}

export function fetchBreakpointsFromWorkflow(debuggerAddr: string): number[]; 