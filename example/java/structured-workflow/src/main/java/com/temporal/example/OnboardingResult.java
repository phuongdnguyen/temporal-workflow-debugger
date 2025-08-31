package com.temporal.example;

/**
 * Result of the user onboarding workflow.
 */
public class OnboardingResult {
    private final String userId;
    private final boolean success;
    private final String message;
    private final long completionTime;
    
    public OnboardingResult(String userId, boolean success, String message, long completionTime) {
        this.userId = userId;
        this.success = success;
        this.message = message;
        this.completionTime = completionTime;
    }
    
    public String getUserId() {
        return userId;
    }
    
    public boolean isSuccess() {
        return success;
    }
    
    public String getMessage() {
        return message;
    }
    
    public long getCompletionTime() {
        return completionTime;
    }
    
    @Override
    public String toString() {
        return "OnboardingResult{" +
                "userId='" + userId + '\'' +
                ", success=" + success +
                ", message='" + message + '\'' +
                ", completionTime=" + completionTime +
                '}';
    }
}
