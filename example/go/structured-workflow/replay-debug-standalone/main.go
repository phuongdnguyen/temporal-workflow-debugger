package main

import (
	"go.temporal.io/sdk/worker"

	"github.com/phuongdnguyen/temporal-workflow-replay-debugger/replayer-adapter-go"

	"example/pkg/workflows"
)

func main() {
	replayer_adapter_go.SetBreakpoints([]int{3, 9})
	replayer_adapter_go.SetReplayMode(replayer_adapter_go.ReplayModeStandalone)
	err := replayer_adapter_go.Replay(replayer_adapter_go.ReplayOptions{
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
		HistoryFilePath:     "/Users/duyphuongnguyen/GolandProjects/temporal-goland-plugin/example/go/structured-workflow/replay-debug-standalone/history.json",
	}, workflows.ExampleWorkflow)
	if err != nil {
		panic(err)
	}
}
