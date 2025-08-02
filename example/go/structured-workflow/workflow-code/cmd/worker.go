package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"example/pkg/workflows"
)

const (
	// TaskQueue is the task queue name used by the worker and workflow starter
	TaskQueue = "example-task-queue"
)

func runWorker() error {
	// Create the client object just once per process
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	// Create the worker object that can be used to run multiple instances
	w := worker.New(c, TaskQueue, worker.Options{})

	// This worker hosts both Workflow and Activity functions
	w.RegisterWorkflow(workflows.ExampleWorkflow)
	w.RegisterActivity(workflows.ExampleActivity)

	log.Printf("Starting worker on task queue: %s", TaskQueue)
	log.Println("Worker is ready to process workflows and activities...")

	// Start listening to the Task Queue
	return w.Run(worker.InterruptCh())
}
