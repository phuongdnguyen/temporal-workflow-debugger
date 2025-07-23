package main

import (
	"go.temporal.io/sdk/worker"

	"replayer_adapter"
)

func main() {
	replayer_adapter.SetBreakpoints([]int{3, 9, 15})
	err := replayer_adapter.Replay(replayer_adapter.ReplayOptions{
		Mode:                replayer_adapter.Mode_Standalone,
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
		HistoryFilePath:     "/Users/duyphuongnguyen/GolandProjects/temporal-goland-plugin/example/simple-workflow/history.json",
	}, SimpleWorkflow)
	if err != nil {
		panic(err)
	}
}
