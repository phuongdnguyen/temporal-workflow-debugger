/**
 * Workflow interceptors for replayer adapter debugging.
 * These interceptors notify the runner when workflow operations occur,
 * enabling breakpoint support for workflow replay debugging.
 * 
 * This version uses worker threads for synchronous breakpoint fetching.
 */

import {
  WorkflowInterceptorsFactory,
  WorkflowInboundCallsInterceptor,
  WorkflowOutboundCallsInterceptor,
  workflowInfo,
} from '@temporalio/workflow';

// import { httpPost } from './http-client';

/**
 * Workflow-safe breakpoint checker using worker threads (synchronous)
 */
function isBreakpointSync(eventId: number): boolean {
  try {
    // Get the global function that was injected by the replayer
    const fetchBreakpoints = globalThis.constructor.constructor('return globalThis.fetchBreakpointsFromWorkflow')();
    if (!fetchBreakpoints) {
      console.warn('fetchBreakpointsFromWorkflow not available, breakpoint checking disabled');
      return false;
    }

    // Get debugger address from global context
    const getDebuggerAddr = globalThis.constructor.constructor('return globalThis.getDebuggerAddr')();
    if (!getDebuggerAddr) {
      console.warn('getDebuggerAddr not available, breakpoint checking disabled');
      return false;
    }

    const debuggerAddr = getDebuggerAddr();

    // For standalone mode, debuggerAddr will be null, and fetchBreakpoints returns the static list
    let breakpointIds: number[];
    if (!debuggerAddr) {
      // Standalone mode: use breakpoints from options
      breakpointIds = fetchBreakpoints();
    } else {
      // IDE mode: fetch breakpoints from debugger
      breakpointIds = fetchBreakpoints(debuggerAddr);
    }

    const isHit = breakpointIds.includes(eventId);

    if (isHit) {
      console.log(`✓ Hit breakpoint at eventId: ${eventId}`);
      // Send highlight request using the same global escape hatch technique
      if (debuggerAddr) {
        try {
          const sendHighlight = globalThis.constructor.constructor('return globalThis.sendHighlightFromWorkflow')();
          if (typeof sendHighlight === 'function') {
            console.log("sending highlight request")
            sendHighlight(debuggerAddr, eventId)
          }
        } catch (err) {
          console.warn('Failed to send highlight request from workflow context:', err);
        }
      }

    } else {
      console.log(`Event ${eventId} is not a breakpoint. Current breakpoints: [${breakpointIds.join(', ')}]`);
    }
    return isHit;
  } catch (error) {
    console.error(`Failed to check breakpoint for event ${eventId}: ${error}`);
    return false;
  }
}


// function sendHighlightRequest(addr: string, payload: string): void {
//   console.log("Sending higlight request")
//   httpPost(`${addr}/current-event`, payload)
//     .then((response) => {
//       console.log(`Highlight response status: ${response.statusCode}, body: ${response.body}`);
//       if (response.statusCode !== 200) {
//         console.warn(`Highlight request failed: ${response.statusCode} ${response.body}`);
//       }
//     })
//     .catch((error) => {
//       console.warn(`Failed to send highlight request: ${error}`);
//     });
// }

/**
 * Raise a breakpoint for debugging - called from interceptors
 * This version uses worker threads for synchronous breakpoint fetching
 */
function raiseSentinelBreakpointSync(caller: string, info?: any): void {
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
    console.log(`runner notified at ${caller}, eventId: ${eventId}`);

    // Synchronous breakpoint checking using worker threads
    const shouldBreak = isBreakpointSync(eventId);
    if (shouldBreak) {
      console.log(`Pause at event ${eventId}`);
      debugger;
    }
  }
}

/**
 * Inbound interceptor that catches workflow entry points
 */
class RunnerWorkflowInboundInterceptor implements WorkflowInboundCallsInterceptor {
  async execute(input: any, next: any): Promise<any> {
    raiseSentinelBreakpointSync('ExecuteWorkflow', workflowInfo());
    return next(input);
  }

  async handleSignal(input: any, next: any): Promise<any> {
    raiseSentinelBreakpointSync('HandleSignal', workflowInfo());
    return next(input);
  }

  async handleQuery(input: any, next: any): Promise<any> {
    raiseSentinelBreakpointSync('HandleQuery', workflowInfo());
    return next(input);
  }

  async handleUpdate(input: any, next: any): Promise<any> {
    raiseSentinelBreakpointSync('HandleUpdate', workflowInfo());
    return next(input);
  }
}

/**
 * Outbound interceptor that catches workflow operations
 */
class RunnerWorkflowOutboundInterceptor implements WorkflowOutboundCallsInterceptor {
  async scheduleActivity(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpointSync('ExecuteActivity', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpointSync('ExecuteActivity', null);
      }
    }
  }

  async scheduleLocalActivity(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpointSync('ExecuteLocalActivity', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpointSync('ExecuteLocalActivity', null);
      }
    }
  }

  async startChildWorkflowExecution(input: any, next: any): Promise<any> {
    try {
      const result = await next(input);
      raiseSentinelBreakpointSync('ExecuteChildWorkflow', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('ExecuteChildWorkflow', null);
      throw error;
    }
  }

  async startTimer(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpointSync('NewTimer', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpointSync('NewTimer', null);
      }
    }
  }

  async signalWorkflow(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpointSync('SignalExternalWorkflow', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpointSync('SignalExternalWorkflow', null);
      }
    }
  }

  async requestCancelWorkflowExecution(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpointSync('RequestCancelExternalWorkflow', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpointSync('RequestCancelExternalWorkflow', null);
      }
    }
  }

  continueAsNew(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('NewContinueAsNewError', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('NewContinueAsNewError', null);
      throw error;
    }
  }

  getLogAttributes(input: any, next: any): any {
    return next(input);
  }

  now(next: any): any {
    try {
      const result = next();
      raiseSentinelBreakpointSync('Now', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('Now', null);
      throw error;
    }
  }

  upsertSearchAttributes(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('UpsertSearchAttributes', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('UpsertSearchAttributes', null);
      throw error;
    }
  }

  upsertMemo(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('UpsertMemo', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('UpsertMemo', null);
      throw error;
    }
  }

  getVersion(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('GetVersion', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('GetVersion', null);
      throw error;
    }
  }

  random(next: any): any {
    try {
      const result = next();
      raiseSentinelBreakpointSync('Random', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('Random', null);
      throw error;
    }
  }

  uuid4(next: any): any {
    try {
      const result = next();
      raiseSentinelBreakpointSync('UUID4', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('UUID4', null);
      throw error;
    }
  }

  sideEffect(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('SideEffect', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('SideEffect', null);
      throw error;
    }
  }

  mutableSideEffect(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('MutableSideEffect', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('MutableSideEffect', null);
      throw error;
    }
  }

  sleep(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpointSync('Sleep', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpointSync('Sleep', null);
      throw error;
    }
  }

  deprecatePatch(input: any, next: any): any {
    return next(input);
  }

  patched(input: any, next: any): any {
    return next(input);
  }
}

/**
 * Factory function to create the interceptors
 */
export const interceptors: WorkflowInterceptorsFactory = () => ({
  inbound: [new RunnerWorkflowInboundInterceptor()],
  outbound: [new RunnerWorkflowOutboundInterceptor()],
}); 