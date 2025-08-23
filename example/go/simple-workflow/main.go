package main

import (
	"go.temporal.io/sdk/worker"

	"github.com/phuongdnguyen/temporal-workflow-replay-debugger/replayer-adapter-go"
)

func main() {
	//
	replayer_adapter_go.SetBreakpoints([]int{3, 9, 15})
	replayer_adapter_go.SetReplayMode(replayer_adapter_go.ReplayModeStandalone)
	err := replayer_adapter_go.Replay(replayer_adapter_go.ReplayOptions{
		WorkerReplayOptions: worker.WorkflowReplayerOptions{DisableDeadlockDetection: true},
		HistoryFilePath: "/Users/duyphuongnguyen/GolandProjects/temporal-goland-plugin/example/go/simple-workflow" +
			"/history.json",
	}, SimpleWorkflow)
	if err != nil {
		panic(err)
	}
}
