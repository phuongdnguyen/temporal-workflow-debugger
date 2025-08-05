/**
 * Workflow interceptors for replayer adapter debugging.
 * These interceptors notify the runner when workflow operations occur,
 * enabling breakpoint support for workflow replay debugging.
 */

import {
  WorkflowInterceptorsFactory,
  WorkflowInboundCallsInterceptor,
  WorkflowOutboundCallsInterceptor,
  workflowInfo,
} from '@temporalio/workflow';
import { raiseSentinelBreakpoint } from './replayer';

/**
 * Inbound interceptor that catches workflow entry points
 */
class RunnerWorkflowInboundInterceptor implements WorkflowInboundCallsInterceptor {
  async execute(input: any, next: any): Promise<any> {
    raiseSentinelBreakpoint('ExecuteWorkflow', workflowInfo());
    return next(input);
  }

  async handleSignal(input: any, next: any): Promise<any> {
    raiseSentinelBreakpoint('HandleSignal', workflowInfo());
    return next(input);
  }

  async handleQuery(input: any, next: any): Promise<any> {
    raiseSentinelBreakpoint('HandleQuery', workflowInfo());
    return next(input);
  }

  async handleUpdate(input: any, next: any): Promise<any> {
    raiseSentinelBreakpoint('HandleUpdate', workflowInfo());
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
        raiseSentinelBreakpoint('ExecuteActivity', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpoint('ExecuteActivity', null);
      }
    }
  }

  async scheduleLocalActivity(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpoint('ExecuteLocalActivity', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpoint('ExecuteLocalActivity', null);
      }
    }
  }

  async startChildWorkflowExecution(input: any, next: any): Promise<any> {
    try {
      const result = await next(input);
      raiseSentinelBreakpoint('ExecuteChildWorkflow', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('ExecuteChildWorkflow', null);
      throw error;
    }
  }

  async startTimer(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpoint('NewTimer', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpoint('NewTimer', null);
      }
    }
  }

  async signalWorkflow(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpoint('SignalExternalWorkflow', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpoint('SignalExternalWorkflow', null);
      }
    }
  }

  async requestCancelWorkflowExecution(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpoint('RequestCancelExternalWorkflow', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpoint('RequestCancelExternalWorkflow', null);
      }
    }
  }

  async sleep(input: any, next: any): Promise<any> {
    try {
      return await next(input);
    } finally {
      try {
        raiseSentinelBreakpoint('Sleep', workflowInfo());
      } catch (error) {
        raiseSentinelBreakpoint('Sleep', null);
      }
    }
  }

  sideEffect(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpoint('SideEffect', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('SideEffect', null);
      throw error;
    }
  }

  mutableSideEffect(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpoint('MutableSideEffect', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('MutableSideEffect', null);
      throw error;
    }
  }

  now(next: any): any {
    try {
      const result = next();
      raiseSentinelBreakpoint('Now', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('Now', null);
      throw error;
    }
  }

  upsertSearchAttributes(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpoint('UpsertSearchAttributes', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('UpsertSearchAttributes', null);
      throw error;
    }
  }

  upsertMemo(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpoint('UpsertMemo', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('UpsertMemo', null);
      throw error;
    }
  }

  getVersion(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpoint('GetVersion', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('GetVersion', null);
      throw error;
    }
  }

  isReplaying(next: any): any {
    try {
      const result = next();
      raiseSentinelBreakpoint('IsReplaying', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('IsReplaying', null);
      throw error;
    }
  }

  continueAsNew(input: any, next: any): any {
    try {
      const result = next(input);
      raiseSentinelBreakpoint('NewContinueAsNewError', workflowInfo());
      return result;
    } catch (error) {
      raiseSentinelBreakpoint('NewContinueAsNewError', null);
      throw error;
    }
  }
}

/**
 * Factory function that creates interceptors for workflow replay debugging
 */
export const interceptors: WorkflowInterceptorsFactory = () => ({
  inbound: [new RunnerWorkflowInboundInterceptor()],
  outbound: [new RunnerWorkflowOutboundInterceptor()],
}); 