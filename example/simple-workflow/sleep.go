package main

import (
	"go.temporal.io/sdk/workflow"
	"time"
)

func zzz(ctx workflow.Context) error {
	// Sleep a bit between iterations to create timer events
	if err := workflow.Sleep(ctx, 2*time.Second); err != nil {
		return err
	}
	return nil
}
