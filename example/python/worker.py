#!/usr/bin/env python3
"""
Worker script to run the Temporal worker that executes workflows and activities.
"""

import asyncio
import logging
from temporalio import activity, workflow
from temporalio.client import Client
from temporalio.worker import Worker

# Import our workflow and activities
from workflow import (
    UserOnboardingWorkflow,
    fetch_user_data,
    validate_user_data, 
    send_welcome_email,
    create_user_profile
)

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

async def main():
    """Main function to start the Temporal worker"""
    
    # Connect to Temporal server (default localhost:7233)
    client = await Client.connect("localhost:7233")
    
    # Create worker
    worker = Worker(
        client,
        task_queue="user-onboarding-task-queue",
        workflows=[UserOnboardingWorkflow],
        activities=[
            fetch_user_data,
            validate_user_data,
            send_welcome_email,
            create_user_profile
        ]
    )
    
    logger.info("🚀 Starting Temporal worker for user onboarding...")
    logger.info("📋 Task Queue: user-onboarding-task-queue")
    logger.info("🔧 Registered Workflow: UserOnboardingWorkflow")
    logger.info("⚡ Registered Activities: fetch_user_data, validate_user_data, send_welcome_email, create_user_profile")
    logger.info("🔄 Worker is ready to process workflows and activities...")
    
    # Start the worker
    await worker.run()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("🛑 Worker stopped by user")
    except Exception as e:
        logger.error(f"❌ Worker failed: {e}")
        raise 