package com.temporal.example;

import io.temporal.client.WorkflowClient;
import io.temporal.serviceclient.WorkflowServiceStubs;
import io.temporal.worker.Worker;
import io.temporal.worker.WorkerFactory;
import io.temporal.worker.WorkerOptions;

/**
 * Worker class that registers and runs the SimpleWorkflow and SimpleActivity.
 * This worker will process workflow and activity tasks from the Temporal server.
 */
public class SimpleWorkflowWorker {
    
    private static final String TASK_QUEUE = "SIMPLE_WORKFLOW_TASK_QUEUE";
    private static final String NAMESPACE = "default";
    
    public static void main(String[] args) {
        try {
            // Create a gRPC stub for the Temporal service
            WorkflowServiceStubs service = WorkflowServiceStubs.newLocalServiceStubs();
            
            // Create a workflow client
            WorkflowClient client = WorkflowClient.newInstance(service);
            
            // Create a worker factory
            WorkerFactory factory = WorkerFactory.newInstance(client);
            
            // Create a worker with custom options
            WorkerOptions workerOptions = WorkerOptions.newBuilder()
                .setMaxConcurrentActivityExecutionSize(10)
                .setMaxConcurrentWorkflowTaskExecutionSize(10)
                .build();
            
            Worker worker = factory.newWorker(TASK_QUEUE, workerOptions);
            
            // Register workflow implementation
            worker.registerWorkflowImplementationTypes(SimpleWorkflowImpl.class);
            
            // Register activity implementation
            worker.registerActivitiesImplementations(new SimpleActivityImpl());
            
            // Start the worker
            System.out.println("Starting SimpleWorkflow worker...");
            System.out.println("Task Queue: " + TASK_QUEUE);
            System.out.println("Namespace: " + NAMESPACE);
            System.out.println("Press Ctrl+C to stop the worker");
            
            factory.start();
            
            // Keep the worker running
            Thread.currentThread().join();
            
        } catch (Exception e) {
            System.err.println("Error starting worker: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}
