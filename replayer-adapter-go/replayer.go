// Package replayer_adapter_go provides workflow replay functionality for Temporal Go SDK applications.
// It enables debugging through standalone mode (using local history files) and IDE mode (with debugger UI integration).
package replayer_adapter_go

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

	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// ReplayMode is the mode of the replay, either ReplayModeStandalone or ReplayModeIde
// ReplayModeStandalone: replay with history file
// ReplayModeIde: replay with debugger UI
type ReplayMode int

const (
	ReplayModeStandalone ReplayMode = iota
	ReplayModeIde
)

func (m ReplayMode) String() string {
	switch m {
	case ReplayModeStandalone:
		return "standalone"
	case ReplayModeIde:
		return "ide"
	default:
		return "unknown"
	}
}

var (
	mode ReplayMode
	// breakpoints only used in standalone mode
	breakpoints = make(map[int]struct{})
)

// ReplayOptions configures the workflow replay behavior.
type ReplayOptions struct {
	WorkerReplayOptions worker.WorkflowReplayerOptions
	// HistoryFilePath only used in Standalone mode, absolute path to the history file
	HistoryFilePath string
}

// SetBreakpoints sets the event IDs where execution should pause during replay.
// This function is only used in Standalone mode.
func SetBreakpoints(eventIds []int) {
	for _, eventId := range eventIds {
		breakpoints[eventId] = struct{}{}
	}
}

// SetReplayMode configures the replay mode (Standalone or IDE).
func SetReplayMode(m ReplayMode) {
	mode = m
}

// Replay executes workflow replay with the specified options and workflow function.
// The behavior depends on the configured ReplayMode:
// - ReplayModeStandalone: replays using the history file specified in opts.HistoryFilePath
// - ReplayModeIde: replays by fetching history from the IDE debugger interface
func Replay(opts ReplayOptions, wf any) error {
	fmt.Printf("Replaying in mode %s", mode)
	if mode == ReplayModeStandalone {
		return replayWithJSONFile(opts.WorkerReplayOptions, wf, opts.HistoryFilePath)
	}
	hist, err := getHistoryFromIDE()
	if err != nil {
		return fmt.Errorf("could not get history: %v", err)
	}
	// replay with history
	return replayWithHistory(opts.WorkerReplayOptions, hist, wf)
}

var (
	lastNotifiedStartEvent = -1
	debuggerAddr           = "" // Store the debugger address for breakpoint checks
)

func isBreakpoint(id int) bool {
	switch mode {
	case ReplayModeStandalone:
		fmt.Printf("Standalone checking breakpoints: %v\n", breakpoints)
		for breakpointID := range breakpoints {
			if breakpointID == id {
				return true
			}
		}
	case ReplayModeIde:
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
	default:
		return false
	}
	return false
}

func replayWithHistory(opts worker.WorkflowReplayerOptions, hist *historypb.History, wf any) error {
	opts.Interceptors = append(opts.Interceptors, &runnerWorkerInterceptor{})
	replayer, err := worker.NewWorkflowReplayerWithOptions(opts)
	if err != nil {
		return fmt.Errorf("create workflow replayer failed: %w", err)
	}
	logger := slog.Default()
	replayer.RegisterWorkflow(wf)
	return replayer.ReplayWorkflowHistory(logger, hist)
}

func replayWithJSONFile(opts worker.WorkflowReplayerOptions, wf any, jsonFileName string) error {
	opts.Interceptors = append(opts.Interceptors, &runnerWorkerInterceptor{})
	replayer, err := worker.NewWorkflowReplayerWithOptions(opts)
	if err != nil {
		return fmt.Errorf("create workflow replayer failed: %w", err)
	}
	logger := slog.Default()
	replayer.RegisterWorkflow(wf)
	return replayer.ReplayWorkflowHistoryFromJSONFile(logger, jsonFileName)
}

func raiseSentinelBreakpoint(caller string, info *workflow.Info) {
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
		fmt.Printf("runner notified at %+v by %s\n eventId: %d \n", time.Now(), caller, eventId)
		if isBreakpoint(eventId) {
			fmt.Printf("Pause at event %d \n", eventId)
			if mode == ReplayModeIde {
				highlightCurrentEventInIDE(eventId)
			}
			runtime.Breakpoint() // Sentinel breakpoint for auto-stepping detection
		}
	}
}

// highlightCurrentEventInIDE sends a POST request to highlight the current event being debugged
func highlightCurrentEventInIDE(eventId int) {
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

func getHistoryFromIDE() (*historypb.History, error) {

	addr := os.Getenv("TEMPORAL_DEBUGGER_PLUGIN_URL")
	if addr == "" {
		addr = "http://localhost:54578"
	}
	// Store runner address for breakpoint checks
	debuggerAddr = addr
	// Create client with timeout to match other implementations
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(debuggerAddr + "/history")
	if err != nil {
		return nil, fmt.Errorf("could not get history from IDE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get history from IDE: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read history response: %v", err)
	}

	// Decode binary protobuf directly
	hist := &historypb.History{}
	if err := hist.Unmarshal(body); err != nil {
		return nil, fmt.Errorf("could not unmarshal protobuf history: %v", err)
	}

	return hist, nil
}
