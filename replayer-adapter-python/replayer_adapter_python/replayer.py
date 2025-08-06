# Main replay logic for Temporal Python replayer adapter
import enum
import json
import os
import requests
from typing import Any, Dict, List, Optional, Type
from temporalio.client import WorkflowHistory
from temporalio.worker import Replayer, ReplayerConfig, UnsandboxedWorkflowRunner
from temporalio.worker import (
    WorkflowInboundInterceptor, WorkflowOutboundInterceptor,
    ActivityInboundInterceptor, ActivityOutboundInterceptor, Interceptor,
    WorkflowInterceptorClassInput, ExecuteWorkflowInput, HandleSignalInput,
    HandleQueryInput, HandleUpdateInput
)
from temporalio.workflow import info as workflow_info
import logging

logger = logging.getLogger("replayer_adapter")

class ReplayMode(enum.Enum):
    STANDALONE = "standalone"
    IDE = "ide"

mode: ReplayMode = ReplayMode.STANDALONE
breakpoints: set = set()
debugger_addr: str = ""
last_notified_start_event: int = -1

class ReplayOptions:
    def __init__(self, worker_replay_options: Optional[ReplayerConfig] = None, history_file_path: Optional[str] = None):
        self.worker_replay_options = worker_replay_options or {}
        self.history_file_path = history_file_path

def set_breakpoints(event_ids: List[int]):
    """Set breakpoints for standalone mode"""
    global breakpoints
    breakpoints = set(event_ids)

def set_replay_mode(m: ReplayMode):
    """Set the replay mode (standalone or IDE)"""
    global mode
    mode = m

def is_breakpoint(event_id: int) -> bool:
    """Check if the given event ID is a breakpoint"""
    global breakpoints, mode, debugger_addr
    if mode == ReplayMode.STANDALONE:
        print(f"Standalone checking breakpoints: {breakpoints}, even_id: {event_id}")
        if event_id in breakpoints:
            print(f"Hit breakpoint at event_id: {event_id}")
        return event_id in breakpoints
    elif mode == ReplayMode.IDE:
        if not debugger_addr:
            return False
        try:
            resp = requests.get(f"{debugger_addr}/breakpoints", timeout=2)
            resp.raise_for_status()
            payload = resp.json()
            return event_id in payload.get("breakpoints", [])
        except Exception as e:
            logger.warning(f"Could not get breakpoints from IDE: {e}")
            return False

def highlight_current_event_in_ide(event_id: int):
    """Send highlight request to IDE for current event"""
    global debugger_addr
    if not debugger_addr:
        logger.warning("debugger_addr is empty, cannot send highlight request")
        return
    
    logger.info(f"Sending highlight request for event {event_id} to {debugger_addr}/current-event")
    payload = {"eventId": event_id}
    try:
        resp = requests.post(f"{debugger_addr}/current-event", json=payload, timeout=5)
        logger.info(f"Highlight response status: {resp.status_code}, body: {resp.text}")
        if resp.status_code != 200:
            logger.warning(f"Highlight request failed: {resp.status_code} {resp.text}")
        else:
            logger.info(f"✓ Successfully highlighted event {event_id} in debugger UI")
    except Exception as e:
        logger.warning(f"Failed to send highlight request: {e}")

def raise_sentinel_breakpoint(caller: str, info: Optional[Any]):
    """Raise a breakpoint for debugging - called from interceptors"""
    global last_notified_start_event
    event_id = None
    
    if info is not None:
        # Try to get event ID from workflow info
        try:
            event_id = info.get_current_history_length()
        except (AttributeError, Exception):
            event_id = None
    
    if event_id is not None:
        if event_id <= last_notified_start_event:
            return
        last_notified_start_event = event_id
        logger.info(f"runner notified at {caller}, eventId: {event_id}")
        
        if is_breakpoint(event_id):
            logger.info(f"Pause at event {event_id}")
            if mode == ReplayMode.IDE:
                highlight_current_event_in_ide(event_id)
            # Python equivalent of runtime.Breakpoint()
            # Since we run outside the sandbox, breakpoint() works normally
            breakpoint()

# Interceptor Classes
class RunnerWorkflowInboundInterceptor(WorkflowInboundInterceptor):
    def __init__(self, next: WorkflowInboundInterceptor):
        super().__init__(next)

    def init(self, outbound: WorkflowOutboundInterceptor) -> None:
        """Initialize with the outbound interceptor"""
        super().init(RunnerWorkflowOutboundInterceptor(outbound))

    async def execute_workflow(self, input: ExecuteWorkflowInput):
        raise_sentinel_breakpoint("ExecuteWorkflow", workflow_info())
        return await super().execute_workflow(input)

    async def handle_signal(self, input: HandleSignalInput):
        raise_sentinel_breakpoint("HandleSignal", workflow_info())
        return await super().handle_signal(input)

    async def handle_query(self, input: HandleQueryInput):
        raise_sentinel_breakpoint("HandleQuery", workflow_info())
        return await super().handle_query(input)

    def handle_update_validator(self, input: HandleUpdateInput):
        raise_sentinel_breakpoint("ValidateUpdate", workflow_info())
        return super().handle_update_validator(input)

    async def handle_update_handler(self, input: HandleUpdateInput):
        raise_sentinel_breakpoint("ExecuteUpdate", workflow_info())
        return await super().handle_update_handler(input)

class RunnerWorkflowOutboundInterceptor(WorkflowOutboundInterceptor):
    def __init__(self, next: WorkflowOutboundInterceptor):
        super().__init__(next)

    def start_activity(self, input):
        # Get info safely without calling workflow_info() to avoid recursion
        try:
            info = super().info()
            raise_sentinel_breakpoint("ExecuteActivity", info)
        except:
            raise_sentinel_breakpoint("ExecuteActivity", None)
        return super().start_activity(input)

    def start_local_activity(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("ExecuteLocalActivity", info)
        except:
            raise_sentinel_breakpoint("ExecuteLocalActivity", None)
        return super().start_local_activity(input)

    async def start_child_workflow(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("ExecuteChildWorkflow", info)
        except:
            raise_sentinel_breakpoint("ExecuteChildWorkflow", None)
        return await super().start_child_workflow(input)

    def start_timer(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("NewTimer", info)
        except:
            raise_sentinel_breakpoint("NewTimer", None)
        return super().start_timer(input)

    async def signal_external_workflow(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("SignalExternalWorkflow", info)
        except:
            raise_sentinel_breakpoint("SignalExternalWorkflow", None)
        return await super().signal_external_workflow(input)

    async def signal_child_workflow(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("SignalChildWorkflow", info)
        except:
            raise_sentinel_breakpoint("SignalChildWorkflow", None)
        return await super().signal_child_workflow(input)

    async def request_cancel_external_workflow(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("RequestCancelExternalWorkflow", info)
        except:
            raise_sentinel_breakpoint("RequestCancelExternalWorkflow", None)
        return await super().request_cancel_external_workflow(input)

    def side_effect(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("SideEffect", info)
        except:
            raise_sentinel_breakpoint("SideEffect", None)
        return super().side_effect(input)

    def mutable_side_effect(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("MutableSideEffect", info)
        except:
            raise_sentinel_breakpoint("MutableSideEffect", None)
        return super().mutable_side_effect(input)

    def sleep(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("Sleep", info)
        except:
            raise_sentinel_breakpoint("Sleep", None)
        return super().sleep(input)

    def info(self):
        # Don't call raise_sentinel_breakpoint here to avoid recursion
        return super().info()

    def now(self):
        try:
            info = super().info()
            raise_sentinel_breakpoint("Now", info)
        except:
            raise_sentinel_breakpoint("Now", None)
        return super().now()

    def upsert_search_attributes(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("UpsertSearchAttributes", info)
        except:
            raise_sentinel_breakpoint("UpsertSearchAttributes", None)
        return super().upsert_search_attributes(input)

    def upsert_memo(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("UpsertMemo", info)
        except:
            raise_sentinel_breakpoint("UpsertMemo", None)
        return super().upsert_memo(input)

    def get_version(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("GetVersion", info)
        except:
            raise_sentinel_breakpoint("GetVersion", None)
        return super().get_version(input)

    def is_replaying(self):
        try:
            info = super().info()
            raise_sentinel_breakpoint("IsReplaying", info)
        except:
            raise_sentinel_breakpoint("IsReplaying", None)
        return super().is_replaying()

    def continue_as_new(self, input):
        try:
            info = super().info()
            raise_sentinel_breakpoint("NewContinueAsNewError", info)
        except:
            raise_sentinel_breakpoint("NewContinueAsNewError", None)
        return super().continue_as_new(input)

class RunnerActivityInboundInterceptor(ActivityInboundInterceptor):
    def __init__(self, next: ActivityInboundInterceptor):
        super().__init__(next)

    def init(self, outbound: ActivityOutboundInterceptor) -> None:
        super().init(RunnerActivityOutboundInterceptor(outbound))

    async def execute_activity(self, input):
        raise_sentinel_breakpoint("ExecuteActivity", None)
        return await super().execute_activity(input)

class RunnerActivityOutboundInterceptor(ActivityOutboundInterceptor):
    def __init__(self, next: ActivityOutboundInterceptor):
        super().__init__(next)

    def heartbeat(self, *details):
        raise_sentinel_breakpoint("RecordHeartbeat", None)
        return super().heartbeat(*details)

class RunnerWorkerInterceptor(Interceptor):
    def workflow_interceptor_class(
        self, input: WorkflowInterceptorClassInput
    ) -> Optional[Type[WorkflowInboundInterceptor]]:
        """Return the workflow interceptor class to use."""
        return RunnerWorkflowInboundInterceptor
    
    def intercept_activity(self, next: ActivityInboundInterceptor) -> ActivityInboundInterceptor:
        """Intercept activity calls."""
        return RunnerActivityInboundInterceptor(next)

def get_history_from_ide() -> WorkflowHistory:
    """Get workflow history from IDE via HTTP"""
    global debugger_addr
    port = os.environ.get("WFDBG_HISTORY_PORT", "54578")
    runner_addr = f"http://127.0.0.1:{port}"
    try:
        resp = requests.get(f"{runner_addr}/history", timeout=5)
        resp.raise_for_status()
        debugger_addr = runner_addr
        hist_json = resp.json()
        return WorkflowHistory.from_json("replayed-worker", hist_json)
    except Exception as e:
        logger.error(f"Could not get history from IDE: {e}")
        raise

def replay(opts: ReplayOptions, wf: Any):
    """Main replay function that handles both standalone and IDE modes"""
    logger.info(f"Replaying in mode {mode}")
    if mode == ReplayMode.STANDALONE:
        print("Replaying in standalone mode")
        return replay_with_json_file(opts.worker_replay_options, wf, opts.history_file_path)
    else:
        print("Replaying in IDE Integrated mode")
        hist = get_history_from_ide()
        return replay_with_history(opts.worker_replay_options, hist, wf)

def replay_with_history(opts: ReplayerConfig, hist: WorkflowHistory, wf: Any):
    """Replay workflow with history data"""
    # Add our custom interceptor to the config
    interceptors = list(opts.get('interceptors', []))
    interceptors.append(RunnerWorkerInterceptor())
    replayer = Replayer(
        workflows=[wf],
        interceptors=interceptors,
        workflow_runner=UnsandboxedWorkflowRunner(),  # Run outside sandbox for debugging
        debug_mode=True,
    )
    return replayer.replay_workflow(hist)

def replay_with_json_file(opts: ReplayerConfig, wf: Any, json_file_name: str):
    """Replay workflow with history from JSON file"""
    with open(json_file_name, "r") as f:
        hist = json.load(f)
    interceptors = list(opts.get('interceptors', []))
    interceptors.append(RunnerWorkerInterceptor())
    replayer = Replayer(
        workflows=[wf],
        interceptors=interceptors,
        workflow_runner=UnsandboxedWorkflowRunner(),  # Run outside sandbox for debugging,
        debug_mode=True,
    )
    return replayer.replay_workflow(WorkflowHistory.from_json("replayed-worker", hist))
