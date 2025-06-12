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

# Function to empty and delete S3 bucket
empty_s3_bucket() {
    local bucket_name=$1
    
    echo -e "${YELLOW}Checking if bucket ${bucket_name} exists...${NC}"
    
    if aws s3api head-bucket --bucket "${bucket_name}" --region us-east-1 >/dev/null 2>&1; then
        echo -e "${RED}Emptying S3 bucket: ${bucket_name}${NC}"
        aws s3 rm s3://"${bucket_name}" --recursive --region us-east-1
        
        echo -e "${RED}Deleting S3 bucket: ${bucket_name}${NC}"
        aws s3api delete-bucket --bucket "${bucket_name}" --region us-east-1
        
        echo -e "${GREEN}✅ ${bucket_name} deleted successfully${NC}"
    else
        echo -e "${YELLOW}Bucket ${bucket_name} does not exist, skipping${NC}"
    fi
}

# Function to delete DynamoDB table
delete_dynamodb_table() {
    local table_name=$1
    
    echo -e "${YELLOW}Checking if DynamoDB table ${table_name} exists...${NC}"
    
    if aws dynamodb describe-table --table-name "${table_name}" --region us-east-1 >/dev/null 2>&1; then
        echo -e "${RED}Deleting DynamoDB table: ${table_name}${NC}"
        aws dynamodb delete-table --table-name "${table_name}" --region us-east-1
        
        echo -e "${YELLOW}Waiting for ${table_name} to be deleted...${NC}"
        aws dynamodb wait table-not-exists --table-name "${table_name}" --region us-east-1
        
        echo -e "${GREEN}✅ ${table_name} deleted successfully${NC}"
    else
        echo -e "${YELLOW}Table ${table_name} does not exist, skipping${NC}"
    fi
}

# Function to delete a stack if it exists (with S3 cleanup)
delete_stack() {
    local stack_name=$1
    local env=$2
    
    echo -e "${YELLOW}Checking if stack ${stack_name} exists...${NC}"
    
    if aws cloudformation describe-stacks --stack-name "${stack_name}" --region us-east-1 >/dev/null 2>&1; then
        
        # Special handling for FrontendStack - empty S3 buckets first
        if [[ "${stack_name}" == *"FrontendStack"* ]]; then
            echo -e "${YELLOW}FrontendStack detected - cleaning up S3 buckets first...${NC}"
            
            # Get bucket names from stack outputs
            website_bucket=$(aws cloudformation describe-stacks --stack-name "${stack_name}" --region us-east-1 --query 'Stacks[0].Outputs[?OutputKey==`WebsiteBucketName`].OutputValue' --output text 2>/dev/null || echo "")
            
            if [ -n "$website_bucket" ] && [ "$website_bucket" != "None" ]; then
                empty_s3_bucket "$website_bucket"
            fi
            
            # Also try common bucket naming patterns
            empty_s3_bucket "${env}-amazonbuddy-websitebucket"
            empty_s3_bucket "${env}-amazonbuddy-frontend"
        fi
        
        # Special handling for StorageStack - clean up DynamoDB tables and S3 buckets
        if [[ "${stack_name}" == *"StorageStack"* ]]; then
            echo -e "${YELLOW}StorageStack detected - cleaning up DynamoDB tables and S3 buckets...${NC}"
            
            # Delete DynamoDB tables (they have RETAIN policy)
            delete_dynamodb_table "${env}-AmazonBuddy-BotsTable"
            delete_dynamodb_table "${env}-AmazonBuddy-ConversationsTable" 
            delete_dynamodb_table "${env}-AmazonBuddy-MessagesTable"
            delete_dynamodb_table "${env}-AmazonBuddy-Bots"
            delete_dynamodb_table "${env}-AmazonBuddy-Conversations"
            delete_dynamodb_table "${env}-AmazonBuddy-Messages"
            
            # Also try old naming patterns
            delete_dynamodb_table "${env}-AIChatPlatform-BotsTable"
            delete_dynamodb_table "${env}-AIChatPlatform-ConversationsTable"
            delete_dynamodb_table "${env}-AIChatPlatform-MessagesTable"
            
            # Try common storage bucket naming patterns
            empty_s3_bucket "${env}-amazonbuddy-storage"
            empty_s3_bucket "${env}-amazonbuddy-uploads"
        fi
        
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
    "MonitoringStack"  # Delete before ApiStack since it uses ApiStack exports
    "ApiStack" 
    "AuthStack"
    "StorageStack"
    "NetworkStack"
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