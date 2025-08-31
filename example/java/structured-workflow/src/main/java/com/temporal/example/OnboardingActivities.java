package com.temporal.example;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;

/**
 * Activity interface for user onboarding operations.
 */
@ActivityInterface
public interface OnboardingActivities {
    
    /**
     * Validates a user for onboarding.
     * 
     * @param userId the user ID to validate
     * @return true if user is valid, false otherwise
     */
    @ActivityMethod
    boolean validateUser(String userId);
    
    /**
     * Creates a user profile.
     * 
     * @param userId the user ID
     * @return the created profile ID
     */
    @ActivityMethod
    String createUserProfile(String userId);
    
    /**
     * Configures user preferences.
     * 
     * @param userId the user ID
     * @param preferences the user preferences to configure
     */
    @ActivityMethod
    void configurePreferences(String userId, UserPreferences preferences);
    
    /**
     * Finalizes the onboarding process.
     * 
     * @param userId the user ID
     * @param profileId the profile ID
     * @param accountId the account ID
     */
    @ActivityMethod
    void finalizeOnboarding(String userId, String profileId, String accountId);
}
