#!/usr/bin/env python3
"""
Workflow and activity definitions for the Python replayer adapter example.
"""

import asyncio
from datetime import timedelta
from temporalio import workflow, activity


# ============================================================================
# Activities
# ============================================================================

@activity.defn
async def fetch_user_data(user_id: str) -> dict:
    """Simulate fetching user data from a database"""
    print(f"🔍 Fetching user data for user_id: {user_id}")
    # Simulate some processing time
    await asyncio.sleep(0.1)
    return {
        "user_id": user_id,
        "name": f"User {user_id}",
        "email": f"user{user_id}@example.com",
        "status": "active"
    }

@activity.defn
async def validate_user_data(user_data: dict) -> bool:
    """Validate user data"""
    print(f"✅ Validating user data: {user_data}")
    required_fields = ["user_id", "name", "email"]
    return all(field in user_data for field in required_fields)

@activity.defn
async def send_welcome_email(user_data: dict) -> str:
    """Send welcome email to user"""
    print(f"📧 Sending welcome email to: {user_data['email']}")
    await asyncio.sleep(0.1)
    return f"Welcome email sent to {user_data['email']}"

@activity.defn
async def create_user_profile(user_data: dict) -> dict:
    """Create user profile in system"""
    print(f"👤 Creating user profile for: {user_data['name']}")
    await asyncio.sleep(0.1)
    return {
        "profile_id": f"profile_{user_data['user_id']}",
        "user_data": user_data,
        "created_at": "2024-01-01T00:00:00Z"
    }


# ============================================================================
# Workflow
# ============================================================================

@workflow.defn
class UserOnboardingWorkflow:
    """Example workflow for user onboarding process"""
    
    @workflow.run
    async def run(self, user_id: str) -> dict:
        """
        Main workflow execution with multiple steps that will trigger breakpoints
        """
        print(f"🚀 Starting user onboarding workflow for user_id: {user_id}")
        
        # Step 1: Fetch user data (Event ID ~2)
        print("Step 1: Fetching user data...")
        user_data = await workflow.execute_activity(
            fetch_user_data,
            user_id,
            start_to_close_timeout=timedelta(seconds=30)
        )
        
        # Step 2: Validate user data (Event ID ~4)
        print("Step 2: Validating user data...")
        is_valid = await workflow.execute_activity(
            validate_user_data,
            user_data,
            start_to_close_timeout=timedelta(seconds=30)
        )
        
        if not is_valid:
            raise ValueError("User data validation failed")
        
        # Step 3: Send welcome email (Event ID ~6)
        print("Step 3: Sending welcome email...")
        email_result = await workflow.execute_activity(
            send_welcome_email,
            user_data,
            start_to_close_timeout=timedelta(seconds=30)
        )
        
        # Step 4: Create user profile (Event ID ~8)
        print("Step 4: Creating user profile...")
        profile = await workflow.execute_activity(
            create_user_profile,
            user_data,
            start_to_close_timeout=timedelta(seconds=30)
        )
        
        # Step 5: Return completion result (Event ID ~10)
        result = {
            "user_id": user_id,
            "status": "completed",
            "profile": profile,
            "email_sent": email_result
        }
        
        print(f"✅ User onboarding completed successfully: {result}")
        return result
