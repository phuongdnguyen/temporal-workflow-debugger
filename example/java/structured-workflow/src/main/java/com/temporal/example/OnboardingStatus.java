package com.temporal.example;

/**
 * Current status of the onboarding workflow.
 */
public class OnboardingStatus {
    private final String userId;
    private final String currentStep;
    private final int progress;
    private final boolean completed;
    private final long lastUpdated;
    
    public OnboardingStatus(String userId, String currentStep, int progress, boolean completed, long lastUpdated) {
        this.userId = userId;
        this.currentStep = currentStep;
        this.progress = progress;
        this.completed = completed;
        this.lastUpdated = lastUpdated;
    }
    
    public String getUserId() {
        return userId;
    }
    
    public String getCurrentStep() {
        return currentStep;
    }
    
    public int getProgress() {
        return progress;
    }
    
    public boolean isCompleted() {
        return completed;
    }
    
    public long getLastUpdated() {
        return lastUpdated;
    }
    
    @Override
    public String toString() {
        return "OnboardingStatus{" +
                "userId='" + userId + '\'' +
                ", currentStep='" + currentStep + '\'' +
                ", progress=" + progress +
                ", completed=" + completed +
                ", lastUpdated=" + lastUpdated +
                '}';
    }
}
