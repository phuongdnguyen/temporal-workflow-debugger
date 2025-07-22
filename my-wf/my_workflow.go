package main

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/workflow"
)

// SimpleWorkflow is a more complex workflow that executes multiple activities,
// side-effects, sleeps and timers to create a lengthy event history.
func SimpleWorkflow(ctx workflow.Context, name string) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	var builder strings.Builder

	for i := 0; i < 3; i++ {
		var result string
		// Execute the activity multiple times to create more events
		err := workflow.ExecuteActivity(ctx, SimpleActivity, fmt.Sprintf("%s-%d", name, i)).Get(ctx, &result)
		if err != nil {
			return "", err
		}
		builder.WriteString(result)
		builder.WriteString("\n")

		// Add a side effect to generate a marker in history
		var side int
		if err := workflow.SideEffect(ctx, func(workflow.Context) any { return i }).Get(&side); err != nil {
			return "", err
		}

		// Sleep a bit between iterations to create timer events
		if err := workflow.Sleep(ctx, 2*time.Second); err != nil {
			return "", err
		}
	}

	// Final timer to add more events
	if err := workflow.NewTimer(ctx, 5*time.Second).Get(ctx, nil); err != nil {
		return "", err
	}
	return builder.String(), nil
}
