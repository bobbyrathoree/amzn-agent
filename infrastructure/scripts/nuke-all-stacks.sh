#!/bin/bash

# Script to safely delete all AmazonBuddy CloudFormation stacks
# Usage: ./nuke-all-stacks.sh <environment>

set -e

ENVIRONMENT=$1

if [ -z "$ENVIRONMENT" ]; then
    echo "Usage: $0 <environment>"
    echo "Example: $0 dev"
    exit 1
fi

# Check required environment variables
if [ -z "$AWS_PROFILE" ]; then
    echo "Error: AWS_PROFILE environment variable is not set"
    exit 1
fi

if [ -z "$AWS_REGION" ]; then
    echo "Error: AWS_REGION environment variable is not set"
    exit 1
fi

echo "=========================================="
echo "AmazonBuddy Stack Deletion Script"
echo "=========================================="
echo "Environment: $ENVIRONMENT"
echo "AWS Profile: $AWS_PROFILE"
echo "AWS Region: $AWS_REGION"
echo ""

# List all stacks
echo "Listing existing AmazonBuddy stacks..."
STACKS=$(aws cloudformation list-stacks \
    --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE UPDATE_ROLLBACK_COMPLETE ROLLBACK_COMPLETE \
    --query "StackSummaries[?contains(StackName, '${ENVIRONMENT}-AmazonBuddy')].StackName" \
    --output text)

if [ -z "$STACKS" ]; then
    echo "No stacks found for environment: $ENVIRONMENT"
    exit 0
fi

echo "Found the following stacks:"
for stack in $STACKS; do
    echo "  - $stack"
done
echo ""

# Confirmation prompt
read -p "Are you sure you want to delete ALL these stacks? This cannot be undone. Type 'yes' to proceed: " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Deletion cancelled."
    exit 0
fi

echo ""
echo "Starting deletion process..."
echo ""

# Function to wait for stack deletion
wait_for_deletion() {
    local stack_name=$1
    echo "Waiting for $stack_name to be deleted..."

    while true; do
        STATUS=$(aws cloudformation describe-stacks --stack-name "$stack_name" --query 'Stacks[0].StackStatus' --output text 2>/dev/null || echo "DELETE_COMPLETE")

        if [ "$STATUS" == "DELETE_COMPLETE" ] || [ "$STATUS" == "DELETE_FAILED" ]; then
            break
        fi

        echo "  Status: $STATUS"
        sleep 10
    done

    if [ "$STATUS" == "DELETE_FAILED" ]; then
        echo "  WARNING: Stack deletion failed for $stack_name"
        return 1
    else
        echo "  Successfully deleted $stack_name"
        return 0
    fi
}

# Function to empty S3 bucket
empty_s3_bucket() {
    local bucket_name=$1
    echo "Emptying S3 bucket: $bucket_name"

    # Check if bucket exists
    if aws s3 ls "s3://$bucket_name" 2>/dev/null; then
        echo "  Deleting all objects in $bucket_name..."
        aws s3 rm "s3://$bucket_name" --recursive

        echo "  Deleting all versions in $bucket_name..."
        aws s3api delete-objects --bucket "$bucket_name" \
            --delete "$(aws s3api list-object-versions --bucket "$bucket_name" --query '{Objects: Versions[].{Key:Key,VersionId:VersionId}}' --output json)" 2>/dev/null || true

        echo "  Bucket $bucket_name emptied"
    else
        echo "  Bucket $bucket_name does not exist or already deleted"
    fi
}

# Delete stacks in reverse dependency order
# Order: Frontend -> Monitoring -> Api -> Auth -> Storage -> Network

STACK_ORDER=(
    "${ENVIRONMENT}-AmazonBuddy-FrontendStack"
    "${ENVIRONMENT}-AmazonBuddy-MonitoringStack"
    "${ENVIRONMENT}-AmazonBuddy-ApiStack"
    "${ENVIRONMENT}-AmazonBuddy-AuthStack"
    "${ENVIRONMENT}-AmazonBuddy-StorageStack"
    "${ENVIRONMENT}-AmazonBuddy-NetworkStack"
)

for stack in "${STACK_ORDER[@]}"; do
    # Check if stack exists
    if aws cloudformation describe-stacks --stack-name "$stack" &>/dev/null; then
        echo "=========================================="
        echo "Deleting stack: $stack"
        echo "=========================================="

        # Special handling for Frontend stack (has S3 bucket)
        if [[ $stack == *"FrontendStack"* ]]; then
            # Get bucket name from stack outputs
            BUCKET_NAME=$(aws cloudformation describe-stacks --stack-name "$stack" \
                --query 'Stacks[0].Outputs[?OutputKey==`WebsiteBucketName`].OutputValue' \
                --output text 2>/dev/null || echo "")

            if [ -n "$BUCKET_NAME" ]; then
                empty_s3_bucket "$BUCKET_NAME"
            fi
        fi

        # Special handling for Storage stack (has S3 buckets)
        if [[ $stack == *"StorageStack"* ]]; then
            # Try to get bucket names from stack
            STORAGE_BUCKET=$(aws cloudformation describe-stacks --stack-name "$stack" \
                --query 'Stacks[0].Outputs[?OutputKey==`StorageBucketName`].OutputValue' \
                --output text 2>/dev/null || echo "")

            if [ -n "$STORAGE_BUCKET" ]; then
                empty_s3_bucket "$STORAGE_BUCKET"
            fi
        fi

        # Delete the stack
        aws cloudformation delete-stack --stack-name "$stack"

        # Wait for deletion to complete
        if wait_for_deletion "$stack"; then
            echo "Successfully deleted $stack"
        else
            echo "Failed to delete $stack - continuing with remaining stacks"
        fi

        echo ""
    else
        echo "Stack $stack does not exist, skipping..."
        echo ""
    fi
done

echo "=========================================="
echo "Deletion process complete!"
echo "=========================================="
echo ""

# Final verification
echo "Checking for remaining stacks..."
REMAINING=$(aws cloudformation list-stacks \
    --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE UPDATE_ROLLBACK_COMPLETE ROLLBACK_COMPLETE \
    --query "StackSummaries[?contains(StackName, '${ENVIRONMENT}-AmazonBuddy')].StackName" \
    --output text)

if [ -z "$REMAINING" ]; then
    echo "Success! All AmazonBuddy stacks have been deleted."
else
    echo "Warning: The following stacks still exist:"
    for stack in $REMAINING; do
        echo "  - $stack"
    done
fi
