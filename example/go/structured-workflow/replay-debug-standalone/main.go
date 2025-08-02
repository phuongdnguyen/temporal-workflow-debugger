package main

import (
	"go.temporal.io/sdk/worker"

	"example/pkg/workflows"
	"replayer_adapter"
)

func main() {
	replayer_adapter.SetBreakpoints([]int{3, 9})
	replayer_adapter.SetReplayMode(replayer_adapter.ReplayModeStandalone)
	err := replayer_adapter.Replay(replayer_adapter.ReplayOptions{
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
		HistoryFilePath:     "/Users/duyphuongnguyen/GolandProjects/temporal-goland-plugin/example/go/structured-workflow/replay-debug-standalone/history.json",
	}, workflows.ExampleWorkflow)
	if err != nil {
		panic(err)
	}
}
