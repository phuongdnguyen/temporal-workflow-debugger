package main

import (
	"go.temporal.io/sdk/worker"

	"github.com/phuongdnguyen/temporal-workflow-replay-debugger/replayer-adapter-go"

	"example/pkg/workflows"
)

func main() {
	replayer_adapter_go.SetReplayMode(replayer_adapter_go.ReplayModeIde)
	err := replayer_adapter_go.Replay(replayer_adapter_go.ReplayOptions{
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
	}, workflows.ExampleWorkflow)
	if err != nil {
		panic(err)
	}
}
