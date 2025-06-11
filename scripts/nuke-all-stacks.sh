#!/bin/bash

# Nuke All Stacks Script for AmazonBuddy
# This script destroys ALL CloudFormation stacks for the project

set -e

echo "🔥 NUKING ALL STACKS - AmazonBuddy Project 🔥"
echo "=============================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to delete a stack if it exists
delete_stack() {
    local stack_name=$1
    local env=$2
    
    echo -e "${YELLOW}Checking if stack ${stack_name} exists...${NC}"
    
    if aws cloudformation describe-stacks --stack-name "${stack_name}" --region us-east-1 >/dev/null 2>&1; then
        echo -e "${RED}Deleting stack: ${stack_name}${NC}"
        aws cloudformation delete-stack --stack-name "${stack_name}" --region us-east-1
        
        echo -e "${YELLOW}Waiting for ${stack_name} to be deleted...${NC}"
        aws cloudformation wait stack-delete-complete --stack-name "${stack_name}" --region us-east-1
        
        echo -e "${GREEN}✅ ${stack_name} deleted successfully${NC}"
    else
        echo -e "${YELLOW}Stack ${stack_name} does not exist, skipping${NC}"
    fi
}

# List of environments to check
ENVIRONMENTS=("dev" "prod")

# List of stack suffixes (in deletion order - reverse dependency order)
STACK_SUFFIXES=(
    "FrontendStack"
    "ApiStack" 
    "AuthStack"
    "StorageStack"
    "NetworkStack"
    "MonitoringStack"
)

echo -e "${YELLOW}Starting stack deletion process...${NC}"

# Delete stacks for each environment
for env in "${ENVIRONMENTS[@]}"; do
    echo -e "\n${YELLOW}Processing environment: ${env}${NC}"
    
    # Delete stacks in reverse dependency order
    for suffix in "${STACK_SUFFIXES[@]}"; do
        # Try both old and new naming patterns
        delete_stack "${env}-AIChatPlatform-${suffix}" "${env}"
        delete_stack "${env}-AmazonBuddy-${suffix}" "${env}"
    done
done

# Clean up any remaining AIChatPlatform stacks
echo -e "\n${YELLOW}Cleaning up any remaining AIChatPlatform stacks...${NC}"
aws cloudformation list-stacks --region us-east-1 --query 'StackSummaries[?contains(StackName, `AIChatPlatform`) && StackStatus != `DELETE_COMPLETE`].StackName' --output text | while read -r stack; do
    if [ -n "$stack" ]; then
        delete_stack "$stack" "cleanup"
    fi
done

echo -e "\n${GREEN}🎉 ALL STACKS NUKED SUCCESSFULLY! 🎉${NC}"
echo -e "${GREEN}Ready for fresh deployment of AmazonBuddy${NC}"