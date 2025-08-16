package com.temporal.example;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;
import io.temporal.workflow.SignalMethod;
import io.temporal.workflow.QueryMethod;

/**
 * Workflow interface for user onboarding process.
 * This workflow demonstrates complex patterns including child workflows, signals, and queries.
 */
@WorkflowInterface
public interface UserOnboardingWorkflow {
    
    /**
     * Main workflow method that orchestrates the user onboarding process.
     * 
     * @param userId the user ID to onboard
     * @return onboarding result
     */
    @WorkflowMethod
    OnboardingResult onboardUser(String userId);
    
    /**
     * Signal method to update user preferences during onboarding.
     * 
     * @param preferences user preferences to update
     */
    @SignalMethod
    void updatePreferences(UserPreferences preferences);
    
    /**
     * Query method to get current onboarding status.
     * 
     * @return current onboarding status
     */
    @QueryMethod
    OnboardingStatus getStatus();
}
