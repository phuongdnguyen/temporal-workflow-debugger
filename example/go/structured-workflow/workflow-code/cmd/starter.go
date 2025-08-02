package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"example/pkg/workflows"
)

func startWorkflow() error {
	// Create the client object just once per process
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	// Set workflow options
	workflowOptions := client.StartWorkflowOptions{
		ID:        "example-workflow-" + generateWorkflowID(),
		TaskQueue: TaskQueue,
	}

	// Start the workflow
	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, workflows.ExampleWorkflow, "Temporal")
	if err != nil {
		return err
	}

	log.Printf("Started workflow with ID: %s and RunID: %s", we.GetID(), we.GetRunID())
	log.Printf("WorkflowExecution: %+v", we)

	// Optionally wait for the workflow result
	var result string
	err = we.Get(context.Background(), &result)
	if err != nil {
		return err
	}

	fmt.Printf("Workflow completed with result: %s\n", result)
	return nil
}

func generateWorkflowID() string {
	// Simple timestamp-based ID generation
	// In production, you might want to use UUIDs or other unique identifiers
	return fmt.Sprintf("%d", getCurrentTimestamp())
}

func getCurrentTimestamp() int64 {
	// Return current Unix timestamp in nanoseconds
	return getCurrentTimeNano()
}

func getCurrentTimeNano() int64 {
	// Use the time package to get current timestamp
	return time.Now().UnixNano()
}
