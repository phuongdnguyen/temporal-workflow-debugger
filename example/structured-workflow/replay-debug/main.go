package main

import (
	"go.temporal.io/sdk/worker"

	"example/pkg/workflows"
	"replayer_adapter"
)

func main() {
	replayer_adapter.SetBreakpoints([]int{3, 9})
	err := replayer_adapter.Replay(replayer_adapter.ReplayOptions{
		Mode:                replayer_adapter.Mode_Standalone,
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
		HistoryFilePath:     "/Users/duyphuongnguyen/GolandProjects/temporal-goland-plugin/example/structured-workflow/replay-debug/history.json",
	}, workflows.ExampleWorkflow)
	if err != nil {
		panic(err)
	}
}
