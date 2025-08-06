#!/usr/bin/env python3
"""
IDE example demonstrating the Python replayer adapter for Temporal workflows.
This example shows how to use the adapter in ide mode with breakpoints.

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
# IDE Replay Examples
# ============================================================================

async def example_replay_with_breakpoints():
    """Replay with breakpoints at events"""
    print("\n" + "="*60)
    
    try:
        # Set up ide mode
        set_replay_mode(ReplayMode.IDE)
        
        # Create replay options
        opts = ReplayOptions(
            worker_replay_options={},
        )
        result = await replay(opts, UserOnboardingWorkflow)
        print(f"Result: {result}")
    except FileNotFoundError as e:
        print(f"History file not found: {e}")
        print("💡 Make sure user_onboarding_history.json exists in the current directory")
    except Exception as e:
        print(f"Replay failed: {e}")

async def main():
    """Run all ide examples"""
    await example_replay_with_breakpoints()

if __name__ == "__main__":
    asyncio.run(main()) 