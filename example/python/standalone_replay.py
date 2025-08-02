#!/usr/bin/env python3
"""
Standalone example demonstrating the Python replayer adapter for Temporal workflows.
This example shows how to use the adapter in standalone mode with breakpoints.

The example reads workflow history from 'user_onboarding_history.json' file.
Make sure this file exists in the current directory before running the examples.
"""

import asyncio
import json
import os
import sys

# Add paths to Python path for imports
project_root = os.path.join(os.path.dirname(__file__), '..', '..')
replayer_adapter_path = os.path.join(project_root, 'replayer-adapter-python')
print("replayer_adapter_path", replayer_adapter_path)
sys.path.insert(0, project_root)
sys.path.insert(0, replayer_adapter_path)

from replayer import (
            ReplayMode, ReplayOptions, set_replay_mode, 
            set_breakpoints, replay
        )

# Import workflow and activities from workflow.py
from workflow import (
    UserOnboardingWorkflow,
)

# ============================================================================
# History Loading Functions
# ============================================================================

def load_workflow_history(history_file_path: str = "user_onboarding_history.json") -> dict:
    """
    Load workflow history from a JSON file
    
    Args:
        history_file_path: Path to the history JSON file
        
    Returns:
        dict: The workflow history data
        
    Raises:
        FileNotFoundError: If the history file doesn't exist
        json.JSONDecodeError: If the history file is not valid JSON
    """
    if not os.path.exists(history_file_path):
        raise FileNotFoundError(
            f"History file not found: {history_file_path}\n"
            f"Please ensure the history file exists in the current directory."
        )
    
    try:
        with open(history_file_path, "r") as f:
            history_data = json.load(f)
        
        print(f"Loaded workflow history from: {history_file_path}")
        print(f"Found {len(history_data.get('events', []))} events in history")
        return history_data
        
    except json.JSONDecodeError as e:
        raise json.JSONDecodeError(
            f"Invalid JSON in history file {history_file_path}: {e}",
            e.doc, e.pos
        )

# ============================================================================
# Standalone Replay Examples
# ============================================================================

async def example_replay_with_breakpoints():
    """Replay with breakpoints at events"""
    print("\n" + "="*60)
    print("REPLAY WITH BREAKPOINTS EXAMPLE")
    
    # Set up standalone mode
    set_replay_mode(ReplayMode.STANDALONE)
    
    # Set breakpoints at workflow task started events:
    breakpoint_events =  [3, 9, 15, 21]
    set_breakpoints(breakpoint_events)
    
    print(f"Set breakpoints at events: {breakpoint_events}")
    
    try:
        # Load workflow history from external file
        history_file = "user_onboarding_history.json"
        
        # Create replay options
        opts = ReplayOptions(
            worker_replay_options={},
            history_file_path=history_file
        )
        
        print("\n📋 Replaying UserOnboardingWorkflow with breakpoints...")
        result = await replay(opts, UserOnboardingWorkflow)
        print("✅ Replay with breakpoints completed successfully!")
        # print(f"📊 Result: {result}")
    except FileNotFoundError as e:
        print(f"❌ History file not found: {e}")
        print("💡 Make sure user_onboarding_history.json exists in the current directory")
    except Exception as e:
        print(f"❌ Replay with breakpoints failed: {e}")

async def main():
    """Run all standalone examples"""
    print("🚀 PYTHON REPLAYER ADAPTER - STANDALONE EXAMPLES")
    print("="*60)
    print("This example demonstrates the replayer adapter in standalone mode")
    print("with realistic workflow history and breakpoint debugging.")
    
    
    # Run examples
    await example_replay_with_breakpoints()
    
    print("\n" + "="*60)
    print("🎉 All standalone examples completed!")
    print("="*60)

if __name__ == "__main__":
    asyncio.run(main()) 