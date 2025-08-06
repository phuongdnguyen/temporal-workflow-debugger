"""
Python Replayer Adapter for Temporal

This package provides a replayer adapter and interceptors for Temporal workflows.
It enables debugging and replaying workflows with breakpoint support.
"""

__version__ = "0.1.0"
__author__ = "Temporal Technologies"

# Import main classes and functions for public API
from .replayer import (
    ReplayMode,
    ReplayOptions,
    replay,
    replay_with_history,
    replay_with_json_file,
    set_breakpoints,
    set_replay_mode,
    is_breakpoint,
    get_history_from_ide,
    highlight_current_event_in_ide,
    raise_sentinel_breakpoint,
    RunnerWorkerInterceptor,
    RunnerWorkflowInboundInterceptor,
    RunnerWorkflowOutboundInterceptor,
    RunnerActivityInboundInterceptor,
    RunnerActivityOutboundInterceptor,
)

# Define what gets exported when using "from replayer_adapter_python import *"
__all__ = [
    # Core classes
    "ReplayMode",
    "ReplayOptions",
    # Main functions
    "replay",
    "replay_with_history", 
    "replay_with_json_file",
    "set_breakpoints",
    "set_replay_mode",
    "is_breakpoint",
    "get_history_from_ide",
    "highlight_current_event_in_ide",
    "raise_sentinel_breakpoint",
    # Interceptors
    "RunnerWorkerInterceptor",
    "RunnerWorkflowInboundInterceptor",
    "RunnerWorkflowOutboundInterceptor",
    "RunnerActivityInboundInterceptor",
    "RunnerActivityOutboundInterceptor",
] 