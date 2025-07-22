package main

import (
	"go.temporal.io/sdk/worker"

	"replayer_adapter"
)

func main() {
	err := replayer_adapter.Replay(worker.WorkflowReplayerOptions{
		DisableDeadlockDetection: true,
	}, SimpleWorkflow)
	if err != nil {
		panic(err)
	}
}
