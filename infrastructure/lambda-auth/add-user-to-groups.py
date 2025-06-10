import json
import boto3
import logging
import os
from typing import Dict, List

logger = logging.getLogger()
logger.setLevel(logging.INFO)

# Initialize Cognito client
cognito_client = boto3.client('cognito-idp')

# Environment variables  
AUTO_JOIN_USER_GROUPS = ["BotCreators"]  # Default group for new users

def add_user_to_groups(user_pool_id: str, username: str, groups: List[str]) -> None:
    """
    Add a user to specified Cognito User Pool groups
    """
    for group_name in groups:
        try:
            cognito_client.admin_add_user_to_group(
                UserPoolId=user_pool_id,
                Username=username,
                GroupName=group_name
            )
            logger.info(f"Added user {username} to group {group_name}")
        except cognito_client.exceptions.ResourceNotFoundException:
            logger.warning(f"Group {group_name} not found in user pool {user_pool_id}")
        except Exception as e:
            logger.error(f"Failed to add user {username} to group {group_name}: {str(e)}")

def handler(event: Dict, context: Dict) -> Dict:
    """
    Cognito Post-confirmation Lambda trigger
    Automatically adds new users to default groups
    """
    logger.info(f"Post-confirmation trigger called with event: {json.dumps(event)}")
    
    try:
        user_name = event["userName"]
        trigger_source = event["triggerSource"]
        user_pool_id = event["userPoolId"]  # Get user pool ID from event
        
        logger.info(f"Processing user: {user_name}, trigger: {trigger_source}, pool: {user_pool_id}")
        
        # Add user to groups after successful signup confirmation
        if trigger_source == "PostConfirmation_ConfirmSignUp":
            logger.info(f"Adding new user {user_name} to default groups")
            add_user_to_groups(user_pool_id, user_name, AUTO_JOIN_USER_GROUPS)
        
        # Also add to groups when admin creates user and they change password
        elif trigger_source == "PostAuthentication_Authentication":
            user_attributes = event["request"]["userAttributes"]
            user_status = user_attributes.get("cognito:user_status", "")
            
            if user_status == "FORCE_CHANGE_PASSWORD":
                logger.info(f"Adding admin-created user {user_name} to default groups")
                add_user_to_groups(user_pool_id, user_name, AUTO_JOIN_USER_GROUPS)
        
        return event
        
    except KeyError as e:
        logger.error(f"Missing required field in event: {e}")
        return event
    except Exception as e:
        logger.error(f"Error in post-confirmation handler: {str(e)}")
        return event