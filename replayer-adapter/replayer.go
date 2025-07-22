package replayer_adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Replay(opts worker.WorkflowReplayerOptions, wf any) error {
	port := os.Getenv("WFDBG_HISTORY_PORT")
	if port == "" {
		port = "54578"
	}
	runnerAddr := "http://127.0.0.1:" + port
	client := http.DefaultClient
	resp, err := client.Get(runnerAddr + "/history")
	if err != nil {
		return fmt.Errorf("could not get history: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not get history: %v", resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read history: %v", err)
	}
	var hist history.History
	if err := proto.Unmarshal(body, &hist); err != nil {
		// Try JSON
		if jsonErr := protojson.Unmarshal(body, &hist); jsonErr != nil {
			return fmt.Errorf("cannot parse history: binaryErr=%v jsonErr=%v", err, jsonErr)
		}
	}

	// Store runner address for breakpoint checks
	debuggerAddr = runnerAddr

	// replay with history
	return replayWithHistory(opts, &hist, wf)
}

var (
	lastNotifiedStartEvent = -1
	debuggerAddr           = "" // Store the debugger address for breakpoint checks
)

func isBreakpoint(id int) bool {
	if debuggerAddr == "" {
		return false
	}

	// Fetch current breakpoints from debugger
	client := http.DefaultClient
	resp, err := client.Get(debuggerAddr + "/breakpoints")
	if err != nil {
		fmt.Printf("could not get breakpoints: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	var payload struct {
		Breakpoints []int `json:"breakpoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		fmt.Printf("could not decode breakpoints: %v\n", err)
		return false
	}

	// Check if current event ID is in breakpoints
	for _, breakpointID := range payload.Breakpoints {
		if breakpointID == id {
			return true
		}
	}
	return false
}

func replayWithHistory(opts worker.WorkflowReplayerOptions, hist *history.History, wf any) error {
	opts.Interceptors = append(opts.Interceptors, &runnerWorkerInterceptor{})
	replayer, err := worker.NewWorkflowReplayerWithOptions(opts)
	if err != nil {
		return fmt.Errorf("create workflow replayer failed: %w", err)
	}
	logger := slog.Default()
	replayer.RegisterWorkflow(wf)
	return replayer.ReplayWorkflowHistory(logger, hist)
}

func shouldStop(eventID int) bool {
	return isBreakpoint(eventID)
}

func notifyRunner(caller string, info *workflow.Info) {
	// activity interceptors
	if info == nil {
		// should let user decide to stop on activity or not
		// if shouldStop(lastNotifiedStartEvent) {
		// 	runtime.Breakpoint()
		// }
	} else {
		eventId := info.GetCurrentHistoryLength()
		if eventId <= lastNotifiedStartEvent {
			return
		}
		lastNotifiedStartEvent = eventId
		// body := map[string]int{"eventId": eventId}
		fmt.Printf("runner notified at %+v by %s\n eventId: %d \n", time.Now(), caller, eventId)
		if shouldStop(eventId) {
			fmt.Printf("Pause at event %d \n", eventId)
			highlightCurrentEvent(eventId)
			runtime.Breakpoint() // Sentinel breakpoint for auto-stepping detection
		}
	}
}

// highlightCurrentEvent sends a POST request to highlight the current event being debugged
func highlightCurrentEvent(eventId int) {
	if debuggerAddr == "" {
		fmt.Printf("WARNING: debuggerAddr is empty, cannot send highlight request\n")
		return
	}

	fmt.Printf("Sending highlight request for event %d to %s\n", eventId, debuggerAddr+"/current-event")

	payload := map[string]int{"eventId": eventId}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("failed to marshal highlight payload: %v\n", err)
		return
	}

	fmt.Printf("Highlight payload: %s\n", string(jsonData))

	// Create client with timeout
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(debuggerAddr+"/current-event", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("failed to send highlight request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read response body for debugging
	responseBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Highlight response status: %d, body: %s\n", resp.StatusCode, string(responseBody))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("highlight request failed with status: %d, response: %s\n", resp.StatusCode, string(responseBody))
		return
	}

	fmt.Printf("✓ Successfully highlighted event %d in debugger UI\n", eventId)
}
