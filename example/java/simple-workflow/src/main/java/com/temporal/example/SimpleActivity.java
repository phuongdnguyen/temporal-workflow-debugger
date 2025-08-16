package com.temporal.example;

import io.temporal.activity.ActivityInterface;
import io.temporal.activity.ActivityMethod;

/**
 * Activity interface for the simple example workflow.
 */
@ActivityInterface
public interface SimpleActivity {
    
    /**
     * Simple greeting activity that returns a formatted message.
     * 
     * @param name the name to greet
     * @return formatted greeting message
     */
    @ActivityMethod
    String greet(String name);
}
