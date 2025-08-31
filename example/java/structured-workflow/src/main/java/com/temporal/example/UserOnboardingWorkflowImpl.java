package com.temporal.example;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;
import io.temporal.workflow.SignalMethod;
import io.temporal.workflow.QueryMethod;
import io.temporal.workflow.Workflow;
import io.temporal.workflow.ChildWorkflowOptions;
import io.temporal.activity.ActivityOptions;
import io.temporal.common.RetryOptions;

import java.time.Duration;

/**
 * Implementation of the UserOnboardingWorkflow interface.
 * This workflow demonstrates complex patterns including child workflows, signals, and queries.
 */
@WorkflowInterface
public class UserOnboardingWorkflowImpl implements UserOnboardingWorkflow {
    
    private final ActivityOptions activityOptions = ActivityOptions.newBuilder()
        .setStartToCloseTimeout(Duration.ofSeconds(30))
        .setRetryOptions(RetryOptions.newBuilder()
            .setInitialInterval(Duration.ofSeconds(1))
            .setMaximumInterval(Duration.ofSeconds(10))
            .setMaximumAttempts(3)
            .build())
        .build();
    
    private final OnboardingActivities activities = Workflow.newActivityStub(OnboardingActivities.class, activityOptions);
    
    // Workflow state
    private UserPreferences preferences;
    private OnboardingStatus status;
    private boolean onboardingCompleted = false;
    
    @Override
    @WorkflowMethod
    public OnboardingResult onboardUser(String userId) {
        // Initialize status
        status = new OnboardingStatus(userId, "Started", 0, false, System.currentTimeMillis());
        
        try {
            // Step 1: Validate user
            status = new OnboardingStatus(userId, "Validating User", 20, false, System.currentTimeMillis());
            boolean userValid = activities.validateUser(userId);
            if (!userValid) {
                throw new RuntimeException("User validation failed for: " + userId);
            }
            
            // Step 2: Create user profile
            status = new OnboardingStatus(userId, "Creating Profile", 40, false, System.currentTimeMillis());
            String profileId = activities.createUserProfile(userId);
            
            // Step 3: Execute child workflow for account setup
            status = new OnboardingStatus(userId, "Setting Up Account", 60, false, System.currentTimeMillis());
            ChildWorkflowOptions childOptions = ChildWorkflowOptions.newBuilder()
                .setWorkflowId("account-setup-" + userId)
                .setTaskQueue("onboarding-task-queue")
                .build();
            
            AccountSetupWorkflow childWorkflow = Workflow.newChildWorkflowStub(AccountSetupWorkflow.class, childOptions);
            String accountId = childWorkflow.setupAccount(userId, profileId);
            
            // Step 4: Configure user preferences
            status = new OnboardingStatus(userId, "Configuring Preferences", 80, false, System.currentTimeMillis());
            if (preferences != null) {
                activities.configurePreferences(userId, preferences);
            }
            
            // Step 5: Finalize onboarding
            status = new OnboardingStatus(userId, "Finalizing", 100, false, System.currentTimeMillis());
            activities.finalizeOnboarding(userId, profileId, accountId);
            
            // Complete onboarding
            onboardingCompleted = true;
            status = new OnboardingStatus(userId, "Completed", 100, true, System.currentTimeMillis());
            
            return new OnboardingResult(userId, true, "Onboarding completed successfully", System.currentTimeMillis());
            
        } catch (Exception e) {
            status = new OnboardingStatus(userId, "Failed", 0, false, System.currentTimeMillis());
            return new OnboardingResult(userId, false, "Onboarding failed: " + e.getMessage(), System.currentTimeMillis());
        }
    }
    
    @Override
    @SignalMethod
    public void updatePreferences(UserPreferences newPreferences) {
        this.preferences = newPreferences;
        if (status != null) {
            status = new OnboardingStatus(
                status.getUserId(),
                status.getCurrentStep(),
                status.getProgress(),
                status.isCompleted(),
                System.currentTimeMillis()
            );
        }
    }
    
    @Override
    @QueryMethod
    public OnboardingStatus getStatus() {
        return status;
    }
}
