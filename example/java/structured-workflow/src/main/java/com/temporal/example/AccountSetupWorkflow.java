package com.temporal.example;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;

/**
 * Child workflow interface for account setup during user onboarding.
 */
@WorkflowInterface
public interface AccountSetupWorkflow {
    
    /**
     * Sets up an account for a user.
     * 
     * @param userId the user ID
     * @param profileId the profile ID
     * @return the created account ID
     */
    @WorkflowMethod
    String setupAccount(String userId, String profileId);
}
