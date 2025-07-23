package workflows

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
)

// ExampleActivity is a sample activity that creates a greeting message
func ExampleActivity(ctx context.Context, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ExampleActivity started", "name", name)

	// Simulate some work
	greeting := fmt.Sprintf("Hello, %s! Welcome to Temporal workflows.", name)

	logger.Info("ExampleActivity completed", "greeting", greeting)
	return greeting, nil
}
