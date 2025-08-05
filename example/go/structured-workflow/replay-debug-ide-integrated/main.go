package main

import (
	"go.temporal.io/sdk/worker"

	"example/pkg/workflows"
	"replayer_adapter_go"
)

func main() {
	replayer_adapter_go.SetReplayMode(replayer_adapter_go.ReplayModeStandalone)
	err := replayer_adapter_go.Replay(replayer_adapter_go.ReplayOptions{
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
	}, workflows.ExampleWorkflow)
	if err != nil {
		panic(err)
	}
}
