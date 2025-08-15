package com.temporal.replayer.interceptors;

import com.temporal.replayer.ReplayerAdapter;
import io.temporal.common.interceptors.WorkflowOutboundCallsInterceptor;
import io.temporal.common.interceptors.WorkflowOutboundCallsInterceptorBase;
import io.temporal.workflow.Workflow;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Workflow outbound interceptor that tracks workflow operations.
 * 
 * This implementation extends WorkflowOutboundCallsInterceptorBase from the Temporal Java SDK
 * and adds debugging hooks following the same patterns as Go/Python adapters.
 */
public class RunnerWorkflowOutboundInterceptor extends WorkflowOutboundCallsInterceptorBase {
    
    private static final Logger logger = LoggerFactory.getLogger(RunnerWorkflowOutboundInterceptor.class);
    
    public RunnerWorkflowOutboundInterceptor(WorkflowOutboundCallsInterceptor next) {
        super(next);
    }
    
    @Override
    public <R> ActivityOutput<R> executeActivity(ActivityInput<R> input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("ExecuteActivity", Workflow.getInfo());
            
            return super.executeActivity(input);
        } catch (Exception e) {
            logger.warn("Error in executeActivity interceptor: {}", e.getMessage());
            return super.executeActivity(input);
        }
    }
    
    @Override
    public <R> LocalActivityOutput<R> executeLocalActivity(LocalActivityInput<R> input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("ExecuteLocalActivity", Workflow.getInfo());
            
            return super.executeLocalActivity(input);
        } catch (Exception e) {
            logger.warn("Error in executeLocalActivity interceptor: {}", e.getMessage());
            return super.executeLocalActivity(input);
        }
    }
    
    @Override
    public <R> ChildWorkflowOutput<R> executeChildWorkflow(ChildWorkflowInput<R> input) {
        try {
            ReplayerAdapter.raiseSentinelBreakpoint("ExecuteChildWorkflow", Workflow.getInfo());
            
            return super.executeChildWorkflow(input);
        } catch (Exception e) {
            logger.warn("Error in executeChildWorkflow interceptor: {}", e.getMessage());
            return super.executeChildWorkflow(input);
        }
    }
    
    // We'll add the most commonly used interceptor methods from the SDK
    // Note: The WorkflowOutboundCallsInterceptor has many methods, but we'll focus on the key ones
    // that match the patterns from Go/Python adapters
    
    // Additional methods can be added as needed following the same pattern
}
