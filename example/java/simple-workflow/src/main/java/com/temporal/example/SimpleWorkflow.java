package com.temporal.example;

import io.temporal.workflow.WorkflowInterface;
import io.temporal.workflow.WorkflowMethod;

/**
 * Workflow interface for the simple example workflow.
 */
@WorkflowInterface
public interface SimpleWorkflow {
    
    /**
     * Main workflow method that executes the workflow logic.
     * 
     * @param name input parameter for the workflow
     * @return result string from the workflow execution
     */
    @WorkflowMethod
    String run(String name);
}
