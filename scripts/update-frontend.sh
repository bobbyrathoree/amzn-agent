#!/bin/bash

# Enable exit on error
set -e

# Check if a region was provided
if [ -z "$1" ]; then
  echo "Error: AWS region must be specified."
  echo "Usage: $0 <aws-region> [environment]"
  echo "Example: $0 us-east-1 prod"
  exit 1
fi

# Set the region
REGION="$1"

# Set the environment (default to prod if not specified)
ENV="${2:-prod}"
echo "Using environment: $ENV"

# ASCII art banner
echo "
🚀 Frontend Update Script 🚀
"

echo "This script will update only the frontend part of your application without redeploying the backend."
echo "Region: $REGION"
echo "Environment: $ENV"
echo ""

# Navigate to project root directory
PROJECT_ROOT="$(dirname "$0")/.."
cd "$PROJECT_ROOT"
echo "Working directory: $(pwd)"

# Check if frontend directory exists
if [ ! -d "frontend" ]; then
  echo "Error: frontend directory not found!"
  exit 1
fi

# Build the frontend
echo "Building frontend application..."
cd frontend
npm install
npm run build

# Check if build was successful
if [ $? -ne 0 ]; then
  echo "Error: Frontend build failed!"
  exit 1
fi

echo "Frontend build completed successfully!"

# Get S3 bucket name from CloudFormation outputs
STACK_NAME="${ENV}-AIChatPlatform-FrontendStack"
BUCKET_NAME=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='BucketName'].OutputValue" \
  --output text --region "$REGION")

if [ -z "$BUCKET_NAME" ]; then
  echo "Error: Could not retrieve S3 bucket name from CloudFormation stack."
  exit 1
fi

echo "Found S3 bucket: $BUCKET_NAME"

# Get CloudFront distribution ID
DISTRIBUTION_ID=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='DistributionId'].OutputValue" \
  --output text --region "$REGION")

if [ -z "$DISTRIBUTION_ID" ]; then
  echo "Error: Could not retrieve CloudFront distribution ID from CloudFormation stack."
  exit 1
fi

echo "Found CloudFront distribution: $DISTRIBUTION_ID"

# Sync the build to S3 bucket
echo "Uploading frontend to S3..."
aws s3 sync out/ s3://$BUCKET_NAME/ --delete --region "$REGION"

# Create CloudFront invalidation
echo "Creating CloudFront invalidation..."
INVALIDATION_ID=$(aws cloudfront create-invalidation \
  --distribution-id "$DISTRIBUTION_ID" \
  --paths "/*" \
  --query "Invalidation.Id" \
  --output text \
  --region "$REGION")

echo "Created CloudFront invalidation: $INVALIDATION_ID"
echo "Wait for invalidation to complete (this may take a few minutes)..."

# Wait for invalidation to complete
aws cloudfront wait invalidation-completed \
  --distribution-id "$DISTRIBUTION_ID" \
  --id "$INVALIDATION_ID" \
  --region "$REGION"

echo "Invalidation completed successfully!"
echo "Frontend update completed! 🎉"
echo ""
echo "You can access your application at:"
aws cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='WebsiteUrl'].OutputValue" \
  --output text --region "$REGION"