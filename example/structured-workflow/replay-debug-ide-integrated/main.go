package main

import (
	"go.temporal.io/sdk/worker"

	"example/pkg/workflows"
	"replayer_adapter"
)

func main() {
	replayer_adapter.SetReplayMode(replayer_adapter.ReplayModeStandalone)
	err := replayer_adapter.Replay(replayer_adapter.ReplayOptions{
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
	}, workflows.ExampleWorkflow)
	if err != nil {
		panic(err)
	}
}
