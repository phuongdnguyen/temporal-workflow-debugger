package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"
)

func runWorkflow() error {
	// Create the Temporal client
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	// Generate a unique workflow ID
	workflowID := fmt.Sprintf("sample-workflow-%d", time.Now().UnixNano())

	// Set workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: TaskQueue,
	}

	log.Printf("Starting workflow with ID: %s", workflowID)
	log.Printf("Task Queue: %s", TaskQueue)

	// Start the workflow execution
	workflowExecution, err := c.ExecuteWorkflow(
		context.Background(),
		workflowOptions,
		ExampleWorkflow,
		"Temporal User", // workflow parameter
	)
	if err != nil {
		return err
	}

	log.Printf("Workflow started successfully!")
	log.Printf("Workflow ID: %s", workflowExecution.GetID())
	log.Printf("Run ID: %s", workflowExecution.GetRunID())
	log.Println()
	log.Println("Waiting for workflow to complete...")

	// Wait for the workflow result
	var result string
	err = workflowExecution.Get(context.Background(), &result)
	if err != nil {
		return err
	}

	log.Println()
	log.Println("✅ Workflow completed successfully!")
	log.Printf("Final result: %s", result)

	return nil
}
