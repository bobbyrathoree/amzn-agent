import json
import logging
from typing import Dict, List

logger = logging.getLogger()
logger.setLevel(logging.INFO)

# Only allow Amazon employees to sign up
ALLOWED_SIGN_UP_EMAIL_DOMAINS = ["amazon.com"]

def check_email_domain(email: str) -> bool:
    """
    Check if the email domain is in the allowed list
    """
    # Always disallow if the number of '@' in the email is not exactly one
    if email.count("@") != 1:
        return False
    
    # Allow if the domain part of the email matches any of the allowed domains
    domain = email.split("@")[1].lower()
    return domain in ALLOWED_SIGN_UP_EMAIL_DOMAINS

def handler(event: Dict, context: Dict) -> Dict:
    """
    Cognito Pre-signup Lambda trigger
    Validates that the user's email domain is allowed
    """
    logger.info(f"Pre-signup trigger called with event: {json.dumps(event)}")
    
    try:
        email = event["request"]["userAttributes"]["email"]
        logger.info(f"Checking email domain for: {email}")
        
        is_allowed = check_email_domain(email)
        
        if is_allowed:
            logger.info(f"Email domain allowed: {email}")
            return event
        else:
            logger.warning(f"Email domain not allowed: {email}")
            raise Exception("Sorry, only Amazon employees can create accounts. Please use your @amazon.com email address.")
            
    except KeyError as e:
        logger.error(f"Missing required field in event: {e}")
        raise Exception("Invalid signup request")
    except Exception as e:
        logger.error(f"Error in pre-signup validation: {str(e)}")
        raise e