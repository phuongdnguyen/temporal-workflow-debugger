/**
 * TypeScript declarations for breakpoint-manager.js
 */

export class BreakpointManager {
  static instance(): BreakpointManager;
  fetchBreakpointsSync(debuggerAddr: string): number[];
  highlightEvent(debuggerAddr: string, eventId: number): void;
  destroy(): void;
}

export function fetchBreakpointsFromWorkflow(debuggerAddr: string): number[]; 
export function sendHighlightFromWorkflow(debuggerAddr: string, eventId: number): void;