package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ExampleWorkflow is a sample workflow that demonstrates various Temporal features
func ExampleWorkflow(ctx workflow.Context, name string) (string, error) {
	// Set activity options
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	logger := workflow.GetLogger(ctx)
	logger.Info("ExampleWorkflow started", "name", name)

	// Execute an activity
	var greeting string
	err := workflow.ExecuteActivity(ctx, ExampleActivity, name).Get(ctx, &greeting)
	if err != nil {
		logger.Error("Activity failed", "error", err)
		return "", err
	}

	logger.Info("Activity completed", "result", greeting)

	// Add a side effect to demonstrate deterministic execution
	var sideEffectResult int
	err = workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		return 42 // Some deterministic value
	}).Get(&sideEffectResult)
	if err != nil {
		return "", err
	}

	// Sleep for a short duration to demonstrate timers
	err = workflow.Sleep(ctx, 3*time.Second)
	if err != nil {
		return "", err
	}

	// Create final result
	result := fmt.Sprintf("%s (side effect: %d)", greeting, sideEffectResult)
	logger.Info("ExampleWorkflow completed", "result", result)

	return result, nil
}
