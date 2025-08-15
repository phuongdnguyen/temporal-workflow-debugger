package com.temporal.replayer.interceptors;

import io.nexusrpc.handler.OperationContext;
import io.temporal.common.interceptors.ActivityInboundCallsInterceptor;
import io.temporal.common.interceptors.NexusOperationInboundCallsInterceptor;
import io.temporal.common.interceptors.WorkerInterceptorBase;
import io.temporal.common.interceptors.WorkflowInboundCallsInterceptor;

/**
 * Worker interceptor that adds workflow and activity interceptors for debugging.
 * 
 * This implementation extends WorkerInterceptorBase from the Temporal Java SDK
 * and adds debugging hooks following the same patterns as Go/Python adapters.
 */
public class RunnerWorkerInterceptor extends WorkerInterceptorBase {
    
    @Override
    public WorkflowInboundCallsInterceptor interceptWorkflow(WorkflowInboundCallsInterceptor next) {
        return new RunnerWorkflowInboundInterceptor(next);
    }
    
    @Override
    public ActivityInboundCallsInterceptor interceptActivity(ActivityInboundCallsInterceptor next) {
        return new RunnerActivityInboundInterceptor(next);
    }
    
    @Override
    public NexusOperationInboundCallsInterceptor interceptNexusOperation(
            OperationContext context, NexusOperationInboundCallsInterceptor next) {
        // For debugging purposes, we don't need to intercept Nexus operations for now
        return super.interceptNexusOperation(context, next);
    }
}
