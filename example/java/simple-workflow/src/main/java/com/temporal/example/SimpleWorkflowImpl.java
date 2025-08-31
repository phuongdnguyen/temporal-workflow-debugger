package com.temporal.example;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;
import io.temporal.workflow.Workflow;
import io.temporal.activity.ActivityOptions;
import io.temporal.common.RetryOptions;

import java.time.Duration;

/**
 * Implementation of the SimpleWorkflow interface.
 * This workflow demonstrates basic Temporal workflow patterns including activities, timers, and side effects.
 */
@WorkflowInterface
public class SimpleWorkflowImpl implements SimpleWorkflow {
    
    private final ActivityOptions activityOptions = ActivityOptions.newBuilder()
        .setStartToCloseTimeout(Duration.ofSeconds(10))
        .setRetryOptions(RetryOptions.newBuilder()
            .setInitialInterval(Duration.ofSeconds(1))
            .setMaximumInterval(Duration.ofSeconds(10))
            .setMaximumAttempts(3)
            .build())
        .build();
    
    private final SimpleActivity activities = Workflow.newActivityStub(SimpleActivity.class, activityOptions);
    
    @Override
    @WorkflowMethod
    public String run(String name) {
        StringBuilder result = new StringBuilder();
        
        // Execute activities multiple times to create more events
        for (int i = 0; i < 3; i++) {
            String activityResult = activities.greet(name + "-" + i);
            result.append(activityResult).append("\n");
            
            // Add a side effect to generate a marker in history
            final int currentI = i; // Create a final copy for the lambda
            int sideEffect = Workflow.sideEffect(Integer.class, () -> currentI);
            result.append("Side effect value: ").append(sideEffect).append("\n");
            
            // Sleep between iterations to create timer events
            Workflow.sleep(Duration.ofSeconds(1));
        }
        
        // Final timer to add more events
        Workflow.sleep(Duration.ofSeconds(5));
        
        return result.toString();
    }
}
