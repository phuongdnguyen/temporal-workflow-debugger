package com.temporal.example;

import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.client.WorkflowStub;
import io.temporal.serviceclient.WorkflowServiceStubs;

import java.time.Duration;

/**
 * Starter class that initiates a SimpleWorkflow execution.
 * This class demonstrates how to start a workflow and wait for its completion.
 */
public class SimpleWorkflowStarter {
    
    private static final String TASK_QUEUE = "SIMPLE_WORKFLOW_TASK_QUEUE";
    private static final String NAMESPACE = "default";
    private static final String WORKFLOW_ID = "simple-workflow-" + System.currentTimeMillis();
    
    public static void main(String[] args) {
        try {
            // Create a gRPC stub for the Temporal service
            WorkflowServiceStubs service = WorkflowServiceStubs.newLocalServiceStubs();
            
            // Create a workflow client
            WorkflowClient client = WorkflowClient.newInstance(service);
            
            // Create workflow options
            WorkflowOptions options = WorkflowOptions.newBuilder()
                .setTaskQueue(TASK_QUEUE)
                .setWorkflowId(WORKFLOW_ID)
                .setWorkflowExecutionTimeout(Duration.ofMinutes(10))
                .setWorkflowRunTimeout(Duration.ofMinutes(5))
                .build();
            
            // Create a workflow stub
            SimpleWorkflow workflow = client.newWorkflowStub(SimpleWorkflow.class, options);
            
            // Start the workflow asynchronously
            System.out.println("Starting SimpleWorkflow...");
            System.out.println("Workflow ID: " + WORKFLOW_ID);
            System.out.println("Task Queue: " + TASK_QUEUE);
            System.out.println("Input: World");
            
            WorkflowStub workflowStub = WorkflowStub.fromTyped(workflow);
            workflowStub.start("World");
            
            // Wait for the workflow to complete
            System.out.println("Waiting for workflow completion...");
            String result = workflowStub.getResult(String.class);
            
            System.out.println("Workflow completed successfully!");
            System.out.println("Result:");
            System.out.println(result);
            
        } catch (Exception e) {
            System.err.println("Error starting workflow: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}
