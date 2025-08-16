package com.temporal.replayer.interceptors;

import com.temporal.replayer.ReplayerAdapter;
import io.temporal.common.interceptors.WorkflowInboundCallsInterceptor;
import io.temporal.common.interceptors.WorkflowInboundCallsInterceptorBase;
import io.temporal.common.interceptors.WorkflowOutboundCallsInterceptor;
import io.temporal.workflow.Workflow;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import javax.annotation.Nonnull;
import javax.annotation.Nullable;
import io.temporal.workflow.WorkflowInfo;

/**
 * Workflow inbound interceptor that tracks workflow execution entry points.
 * 
 * This implementation extends WorkflowInboundCallsInterceptorBase from the Temporal Java SDK
 * and adds debugging hooks following the same patterns as Go/Python adapters.
 */
public class RunnerWorkflowInboundInterceptor extends WorkflowInboundCallsInterceptorBase {
    
    private static final Logger logger = LoggerFactory.getLogger(RunnerWorkflowInboundInterceptor.class);
    
    public RunnerWorkflowInboundInterceptor(WorkflowInboundCallsInterceptor next) {
        super(next);
    }
    
    @Override
    public void init(WorkflowOutboundCallsInterceptor outboundCalls) {
        // Initialize with our custom outbound interceptor
        super.init(new RunnerWorkflowOutboundInterceptor(outboundCalls));
    }
    
    @Override
    public WorkflowOutput execute(WorkflowInput input) {
        try {
            // Get workflow info and notify debugger
            ReplayerAdapter.raiseSentinelBreakpoint("ExecuteWorkflow", Workflow.getInfo());
            
            return super.execute(input);
        } catch (Exception e) {
            logger.warn("Error in executeWorkflow interceptor: {}", e.getMessage());
            return super.execute(input);
        }
    }
    
    @Override
    public void handleSignal(SignalInput input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("HandleSignal", Workflow.getInfo());
            
            super.handleSignal(input);
        } catch (Exception e) {
            logger.warn("Error in handleSignal interceptor: {}", e.getMessage());
            super.handleSignal(input);
        }
    }
    
    @Override
    public QueryOutput handleQuery(QueryInput input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("HandleQuery", Workflow.getInfo());
            
            return super.handleQuery(input);
        } catch (Exception e) {
            logger.warn("Error in handleQuery interceptor: {}", e.getMessage());
            return super.handleQuery(input);
        }
    }
    
    @Override
    public void validateUpdate(UpdateInput input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("ValidateUpdate", Workflow.getInfo());
            
            super.validateUpdate(input);
        } catch (Exception e) {
            logger.warn("Error in validateUpdate interceptor: {}", e.getMessage());
            super.validateUpdate(input);
        }
    }
    
    @Override
    public UpdateOutput executeUpdate(UpdateInput input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("ExecuteUpdate", Workflow.getInfo());
            
            return super.executeUpdate(input);
        } catch (Exception e) {
            logger.warn("Error in executeUpdate interceptor: {}", e.getMessage());
            return super.executeUpdate(input);
        }
    }
    
    @Nonnull
    @Override
    public Object newWorkflowMethodThread(Runnable runnable, @Nullable String name) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("NewWorkflowMethodThread", Workflow.getInfo());
            
            return super.newWorkflowMethodThread(runnable, name);
        } catch (Exception e) {
            logger.warn("Error in newWorkflowMethodThread interceptor: {}", e.getMessage());
            return super.newWorkflowMethodThread(runnable, name);
        }
    }
    
    @Nonnull
    @Override
    public Object newCallbackThread(Runnable runnable, @Nullable String name) {
        try {
            // Only try to get workflow info if we're in a workflow context
            WorkflowInfo workflowInfo = null;
            try {
                // Check if we're in a workflow context before calling getInfo
                if (Workflow.isReplaying()) {
                    workflowInfo = Workflow.getInfo();
                }
            } catch (Exception e) {
                logger.debug("Could not get workflow info in newCallbackThread: {}", e.getMessage());
                // Continue without workflow info
            }
            
            ReplayerAdapter.raiseSentinelBreakpoint("NewCallbackThread", workflowInfo);
            
            return super.newCallbackThread(runnable, name);
        } catch (Exception e) {
            logger.warn("Error in newCallbackThread interceptor: {}", e.getMessage());
            return super.newCallbackThread(runnable, name);
        }
    }
}
