package main

import (
	"context"
	"fmt"
)

// SimpleActivity is a minimal activity definition.
func SimpleActivity(ctx context.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %s!", name), nil
}
