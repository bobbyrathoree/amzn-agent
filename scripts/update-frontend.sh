#!/bin/bash

# Enable exit on error
set -e

# Set defaults
REGION="${AWS_REGION:-us-west-2}"
ENV="${1:-dev}"
AWS_PROFILE="${AWS_PROFILE:-personaldev}"

echo "🚀 Frontend Update Script 🚀"
echo ""
echo "This script will update only the frontend part of your application."
echo "Region: $REGION"
echo "Environment: $ENV"
echo "AWS Profile: $AWS_PROFILE"
echo ""

# Navigate to project root directory
PROJECT_ROOT="$(dirname "$0")/.."
cd "$PROJECT_ROOT"
echo "Working directory: $(pwd)"

# Check if react-frontend directory exists
if [ ! -d "react-frontend" ]; then
  echo "Error: react-frontend directory not found!"
  exit 1
fi

# Build the frontend
echo "Building frontend application..."
cd react-frontend
npm install
npm run build

# Check if build was successful
if [ $? -ne 0 ]; then
  echo "Error: Frontend build failed!"
  exit 1
fi

echo "Frontend build completed successfully!"

# Get all required CloudFormation stack names
FRONTEND_STACK_NAME="${ENV}-AmazonBuddy-FrontendStack"
AUTH_STACK_NAME="${ENV}-AmazonBuddy-AuthStack"
API_STACK_NAME="${ENV}-AmazonBuddy-ApiStack"

# Get S3 bucket name from CloudFormation outputs
STACK_NAME="$FRONTEND_STACK_NAME"
BUCKET_NAME=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='WebsiteBucketName'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE")

if [ -z "$BUCKET_NAME" ]; then
  echo "Error: Could not retrieve S3 bucket name from CloudFormation stack."
  exit 1
fi

echo "Found S3 bucket: $BUCKET_NAME"

# Get CloudFront distribution ID
DISTRIBUTION_ID=$(aws cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='CloudFrontDistributionId'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE")

if [ -z "$DISTRIBUTION_ID" ]; then
  echo "Error: Could not retrieve CloudFront distribution ID from CloudFormation stack."
  exit 1
fi

echo "Found CloudFront distribution: $DISTRIBUTION_ID"

# Get configuration values from CloudFormation stacks
echo "Fetching configuration from CloudFormation stacks..."

USER_POOL_ID=$(aws cloudformation describe-stacks --stack-name "$AUTH_STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='UserPoolId'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE")

USER_POOL_CLIENT_ID=$(aws cloudformation describe-stacks --stack-name "$AUTH_STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='UserPoolClientId'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE")

API_ENDPOINT=$(aws cloudformation describe-stacks --stack-name "$API_STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='RestApiEndpoint'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE")

WEBSOCKET_ENDPOINT=$(aws cloudformation describe-stacks --stack-name "$API_STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='WebSocketEndpoint'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE")

# Generate config.json
echo "Generating config.json..."
cat > dist/config.json <<EOF
{
  "environment": "$ENV",
  "region": "$REGION",
  "userPoolId": "$USER_POOL_ID",
  "userPoolClientId": "$USER_POOL_CLIENT_ID",
  "apiEndpoint": "$API_ENDPOINT",
  "websocketEndpoint": "$WEBSOCKET_ENDPOINT"
}
EOF

echo "✓ Generated config.json with:"
echo "  - Environment: $ENV"
echo "  - Region: $REGION"
echo "  - User Pool: $USER_POOL_ID"
echo "  - User Pool Client: $USER_POOL_CLIENT_ID"
echo "  - API Endpoint: $API_ENDPOINT"
echo "  - WebSocket Endpoint: $WEBSOCKET_ENDPOINT"

# Sync the build to S3 bucket
echo "Uploading frontend to S3..."
aws s3 sync dist/ s3://$BUCKET_NAME/ --delete --region "$REGION" --profile "$AWS_PROFILE"

# Create CloudFront invalidation
echo "Creating CloudFront invalidation..."
INVALIDATION_ID=$(aws cloudfront create-invalidation \
  --distribution-id "$DISTRIBUTION_ID" \
  --paths "/*" \
  --query "Invalidation.Id" \
  --output text \
  --profile "$AWS_PROFILE")

echo "Created CloudFront invalidation: $INVALIDATION_ID"
echo "Waiting for invalidation to complete (this may take a few minutes)..."

# Wait for invalidation to complete
aws cloudfront wait invalidation-completed \
  --distribution-id "$DISTRIBUTION_ID" \
  --id "$INVALIDATION_ID" \
  --profile "$AWS_PROFILE"

echo "Invalidation completed successfully!"
echo "Frontend update completed! 🎉"
echo ""
echo "You can access your application at:"
aws cloudformation describe-stacks --stack-name "$STACK_NAME" \
  --query "Stacks[0].Outputs[?OutputKey=='WebsiteURL'].OutputValue" \
  --output text --region "$REGION" --profile "$AWS_PROFILE"