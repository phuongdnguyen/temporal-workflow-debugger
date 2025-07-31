#!/usr/bin/env python3
"""
Example usage of the Python replayer adapter for Temporal workflows.
This demonstrates both standalone and IDE integration modes.
"""

import asyncio
import json
import logging
import os
from temporalio import workflow, activity
from temporalio.worker import Worker, ReplayerConfig
# Import from the local module (when running from the same directory)
try:
    from replayer_adapter_python import (
        ReplayMode, ReplayOptions, set_replay_mode, 
        set_breakpoints, replay, RunnerWorkerInterceptor
    )
except ImportError:
    # Fallback for local development
    from replayer import (
        ReplayMode, ReplayOptions, set_replay_mode, 
        set_breakpoints, replay
    )
    from replayer import RunnerWorkerInterceptor
from datetime import timedelta

# Configure logging - you can change this level
def setup_logging():
    """Configure logging for the replayer adapter"""
    # Get log level from environment or default to INFO
    log_level = os.environ.get('LOGLEVEL', 'INFO').upper()
    
    logging.basicConfig(
        level=getattr(logging, log_level),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    
    # Configure specific loggers
    logging.getLogger("replayer_adapter").setLevel(getattr(logging, log_level))
    print(f"Logging configured at {log_level} level")

# Example workflow and activity
@activity.defn
async def greet_activity(name: str) -> str:
    return f"Hello, {name}!"

@workflow.defn
class GreetingWorkflow:
    @workflow.run
    async def run(self, name: str) -> str:
        # This will trigger breakpoints during replay
        result = await workflow.execute_activity(
            greet_activity, name, start_to_close_timeout=timedelta(seconds=10)
        )
        return result

# Example history data (simplified)
EXAMPLE_HISTORY = {
    "events": [
        {"eventId": 1, "eventType": "WorkflowExecutionStarted"},
        {"eventId": 2, "eventType": "ActivityTaskScheduled"},
        {"eventId": 3, "eventType": "ActivityTaskStarted"},
        {"eventId": 4, "eventType": "ActivityTaskCompleted"},
        {"eventId": 5, "eventType": "WorkflowExecutionCompleted"}
    ]
}

async def example_standalone_replay():
    """Example of standalone replay with breakpoints"""
    print("=== Standalone Replay Example ===")
    
    # Set up standalone mode
    set_replay_mode(ReplayMode.STANDALONE)
    set_breakpoints([2, 4])  # Break at events 2 and 4
    
    # Create history file
    with open("example_history.json", "w") as f:
        json.dump(EXAMPLE_HISTORY, f)
    
    # Create replay options
    opts = ReplayOptions(
        worker_replay_options={},  # ReplayerConfig is a dict
        history_file_path="example_history.json"
    )
    
    try:
        # Replay the workflow
        await replay(opts, GreetingWorkflow)
        print("✓ Standalone replay completed successfully")
    except Exception as e:
        print(f"✗ Standalone replay failed: {e}")
    finally:
        # Clean up
        if os.path.exists("example_history.json"):
            os.remove("example_history.json")

async def example_ide_integration():
    """Example of IDE integration mode"""
    print("\n=== IDE Integration Example ===")
    
    # Set up IDE mode
    set_replay_mode(ReplayMode.IDE)
    
    # Set environment variable for IDE communication
    os.environ["WFDBG_HISTORY_PORT"] = "54578"
    
    # Create replay options (no history file needed in IDE mode)
    opts = ReplayOptions(
        worker_replay_options={}  # ReplayerConfig is a dict
    )
    
    try:
        # Replay the workflow (will connect to IDE debugger)
        await replay(opts, GreetingWorkflow)
        print("✓ IDE integration replay completed successfully")
    except Exception as e:
        print(f"✗ IDE integration replay failed: {e}")

async def example_worker_integration():
    """Example of using interceptors with a Temporal worker"""
    print("\n=== Worker Integration Example ===")
    
    # This would typically be used with a real Temporal client
    # For demonstration, we show the worker setup
    try:
        # Example worker setup with interceptor
        worker = Worker(
            client=None,  # Would be a real Temporal client
            task_queue="example-task-queue",
            workflows=[GreetingWorkflow],
            activities=[greet_activity],
            interceptors=[RunnerWorkerInterceptor()]
        )
        print("✓ Worker created with replayer interceptor")
    except Exception as e:
        print(f"✗ Worker creation failed: {e}")

def main():
    """Run all examples"""
    print("Python Replayer Adapter Examples")
    print("=" * 40)
    
    # Run examples
    asyncio.run(example_standalone_replay())
    asyncio.run(example_ide_integration())
    asyncio.run(example_worker_integration())
    
    print("\n" + "=" * 40)
    print("Examples completed!")

if __name__ == "__main__":
    main() 