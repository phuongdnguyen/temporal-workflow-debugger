package com.temporal.replayer.interceptors;

import com.temporal.replayer.ReplayerAdapter;
import io.temporal.activity.ActivityExecutionContext;
import io.temporal.common.interceptors.ActivityInboundCallsInterceptor;
import io.temporal.common.interceptors.ActivityInboundCallsInterceptorBase;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Activity inbound interceptor that tracks activity execution.
 * 
 * This implementation extends ActivityInboundCallsInterceptorBase from the Temporal Java SDK
 * and adds debugging hooks following the same patterns as Go/Python adapters.
 */
public class RunnerActivityInboundInterceptor extends ActivityInboundCallsInterceptorBase {
    
    private static final Logger logger = LoggerFactory.getLogger(RunnerActivityInboundInterceptor.class);
    
    public RunnerActivityInboundInterceptor(ActivityInboundCallsInterceptor next) {
        super(next);
    }
    
    @Override
    public void init(ActivityExecutionContext context) {
        super.init(context);
    }
    
    @Override
    public ActivityOutput execute(ActivityInput input) {
        try {
            // For activities, we don't have workflow info, so pass null
            ReplayerAdapter.raiseSentinelBreakpoint("ExecuteActivity", null);
            
            return super.execute(input);
        } catch (Exception e) {
            logger.warn("Error in executeActivity interceptor: {}", e.getMessage());
            return super.execute(input);
        }
    }
}
