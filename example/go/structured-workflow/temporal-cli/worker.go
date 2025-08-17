package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func startWorker() error {
	// Create the Temporal client
	c, err := client.Dial(client.Options{
		HostPort: client.DefaultHostPort,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	// Create a worker that listens on the specified task queue
	w := worker.New(c, TaskQueue, worker.Options{})

	// Register the workflow and activity functions
	w.RegisterWorkflow(ExampleWorkflow)
	w.RegisterActivity(ExampleActivity)

	log.Printf("Worker starting on task queue: %s", TaskQueue)
	log.Println("Available workflows:")
	log.Println("  - ExampleWorkflow")
	log.Println("Available activities:")
	log.Println("  - ExampleActivity")
	log.Println()
	log.Println("Worker is ready to process workflows and activities...")
	log.Println("Press Ctrl+C to stop the worker")

	// Start the worker and block until interrupted
	return w.Run(worker.InterruptCh())
}
