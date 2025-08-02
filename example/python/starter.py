#!/usr/bin/env python3
"""
Starter script to trigger workflow executions.
"""

import asyncio
import logging
import sys
import uuid
from datetime import timedelta
from temporalio.client import Client

# Import our workflow
from workflow import UserOnboardingWorkflow

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def start_workflow(client: Client, user_id: str, wait_for_result: bool = True):
    """
    Start a UserOnboardingWorkflow execution
    
    Args:
        client: Temporal client
        user_id: User ID to process
        wait_for_result: Whether to wait for workflow completion
    """
    
    # Generate a unique workflow ID
    workflow_id = f"user-onboarding-{user_id}-{uuid.uuid4().hex[:8]}"
    
    logger.info(f"🚀 Starting workflow for user_id: {user_id}")
    logger.info(f"📋 Workflow ID: {workflow_id}")
    logger.info(f"📋 Task Queue: user-onboarding-task-queue")
    
    # Start the workflow
    handle = await client.start_workflow(
        UserOnboardingWorkflow.run,
        user_id,
        id=workflow_id,
        task_queue="user-onboarding-task-queue",
        execution_timeout=timedelta(minutes=5),
        retry_policy=None  # Use default retry policy
    )
    
    logger.info(f"✅ Workflow started successfully!")
    logger.info(f"🔗 Workflow Handle: {handle.id}")
    
    if wait_for_result:
        logger.info("⏳ Waiting for workflow completion...")
        try:
            result = await handle.result()
            logger.info(f"🎉 Workflow completed successfully!")
            logger.info(f"📊 Result: {result}")
            return result
        except Exception as e:
            logger.error(f"❌ Workflow failed: {e}")
            raise
    else:
        logger.info("🔄 Workflow started asynchronously (not waiting for result)")
        return handle

async def start_multiple_workflows(client: Client, user_ids: list, wait_for_all: bool = True):
    """
    Start multiple workflow executions
    
    Args:
        client: Temporal client
        user_ids: List of user IDs to process
        wait_for_all: Whether to wait for all workflows to complete
    """
    
    logger.info(f"🚀 Starting {len(user_ids)} workflows...")
    
    if wait_for_all:
        # Start all workflows and wait for results
        tasks = []
        for user_id in user_ids:
            task = start_workflow(client, user_id, wait_for_result=True)
            tasks.append(task)
        
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        successful = [r for r in results if not isinstance(r, Exception)]
        failed = [r for r in results if isinstance(r, Exception)]
        
        logger.info(f"✅ {len(successful)} workflows completed successfully")
        if failed:
            logger.error(f"❌ {len(failed)} workflows failed")
        
        return results
    else:
        # Start all workflows without waiting
        handles = []
        for user_id in user_ids:
            handle = await start_workflow(client, user_id, wait_for_result=False)
            handles.append(handle)
        
        logger.info(f"🔄 All {len(user_ids)} workflows started asynchronously")
        return handles

async def main():
    """Main function to start workflow executions"""
    
    # Connect to Temporal server (default localhost:7233)
    try:
        client = await Client.connect("localhost:7233")
        logger.info("🔗 Connected to Temporal server at localhost:7233")
    except Exception as e:
        logger.error(f"❌ Failed to connect to Temporal server: {e}")
        logger.error("💡 Make sure Temporal server is running on localhost:7233")
        logger.error("💡 Make sure the worker is running (python worker.py)")
        return
    
    # Check command line arguments for user ID
    if len(sys.argv) > 1:
        user_id = sys.argv[1]
        logger.info(f"📝 Using user_id from command line: {user_id}")
    else:
        user_id = "user123"
        logger.info(f"📝 Using default user_id: {user_id}")
    
    # Check for batch mode
    if len(sys.argv) > 2 and sys.argv[2] == "--batch":
        # Start multiple workflows for demo
        user_ids = [f"user{i}" for i in range(1, 4)]
        logger.info("🔄 Running in batch mode...")
        await start_multiple_workflows(client, user_ids, wait_for_all=True)
    else:
        # Start single workflow
        await start_workflow(client, user_id, wait_for_result=True)

def print_usage():
    """Print usage instructions"""
    print("\n" + "="*60)
    print("📖 USAGE INSTRUCTIONS")
    print("="*60)
    print("python starter.py [user_id] [--batch]")
    print("")
    print("Examples:")
    print("  python starter.py                    # Start workflow for 'user123'")
    print("  python starter.py user456            # Start workflow for 'user456'")
    print("  python starter.py user789 --batch   # Start multiple workflows")
    print("")
    print("Prerequisites:")
    print("  1. Temporal server running on localhost:7233")
    print("  2. Worker running: python worker.py")
    print("="*60)

if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] in ["-h", "--help"]:
        print_usage()
        sys.exit(0)
    
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("🛑 Starter stopped by user")
    except Exception as e:
        logger.error(f"❌ Starter failed: {e}")
        print_usage()
        raise 