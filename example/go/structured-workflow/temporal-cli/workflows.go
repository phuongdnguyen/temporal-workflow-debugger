package main

import (
	"example/pkg/workflows"
)

const (
	// TaskQueue is the task queue name used by the worker and workflow starter
	TaskQueue = "temporal-cli-task-queue"
)

// Use the ExampleWorkflow and ExampleActivity from the workflow-code package
var (
	ExampleWorkflow = workflows.ExampleWorkflow
	ExampleActivity = workflows.ExampleActivity
)
