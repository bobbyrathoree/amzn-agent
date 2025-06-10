#!/bin/bash

# Setup frontend environment variables from CDK outputs
# Usage: ./scripts/setup-frontend-env.sh -r us-east-1 -e prod

REGION=""
ENV=""

while getopts "r:e:" opt; do
  case $opt in
    r) REGION="$OPTARG";;
    e) ENV="$OPTARG";;
    \?) echo "Invalid option -$OPTARG" >&2; exit 1;;
  esac
done

if [ -z "$REGION" ] || [ -z "$ENV" ]; then
  echo "Usage: $0 -r <region> -e <environment>"
  echo "Example: $0 -r us-east-1 -e prod"
  exit 1
fi

echo "Setting up frontend environment for $ENV in $REGION..."

# Get CDK outputs
USER_POOL_ID=$(aws cloudformation describe-stacks \
  --stack-name "${ENV}-AIChatPlatform-AuthStack" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`UserPoolId`].OutputValue' \
  --output text 2>/dev/null)

USER_POOL_CLIENT_ID=$(aws cloudformation describe-stacks \
  --stack-name "${ENV}-AIChatPlatform-AuthStack" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`UserPoolClientId`].OutputValue' \
  --output text 2>/dev/null)

IDENTITY_POOL_ID=$(aws cloudformation describe-stacks \
  --stack-name "${ENV}-AIChatPlatform-AuthStack" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`IdentityPoolId`].OutputValue' \
  --output text 2>/dev/null)

API_ENDPOINT=$(aws cloudformation describe-stacks \
  --stack-name "${ENV}-AIChatPlatform-ApiStack" \
  --region "$REGION" \
  --query 'Stacks[0].Outputs[?OutputKey==`RestApiEndpoint`].OutputValue' \
  --output text 2>/dev/null)

# Create .env.local file
ENV_FILE="frontend/.env.local"

echo "# AWS Amplify Configuration - Auto-generated from CDK outputs" > "$ENV_FILE"
echo "NEXT_PUBLIC_USER_POOL_ID=$USER_POOL_ID" >> "$ENV_FILE"
echo "NEXT_PUBLIC_USER_POOL_CLIENT_ID=$USER_POOL_CLIENT_ID" >> "$ENV_FILE"
echo "NEXT_PUBLIC_IDENTITY_POOL_ID=$IDENTITY_POOL_ID" >> "$ENV_FILE"
echo "NEXT_PUBLIC_AWS_REGION=$REGION" >> "$ENV_FILE"
echo "NEXT_PUBLIC_API_ENDPOINT=$API_ENDPOINT" >> "$ENV_FILE"
echo "" >> "$ENV_FILE"
echo "# Build Configuration" >> "$ENV_FILE"
echo "NEXT_PUBLIC_BUILD_MODE=development" >> "$ENV_FILE"

echo "Frontend environment configured:"
echo "- User Pool ID: $USER_POOL_ID"
echo "- User Pool Client ID: $USER_POOL_CLIENT_ID"
echo "- Identity Pool ID: $IDENTITY_POOL_ID"
echo "- API Endpoint: $API_ENDPOINT"
echo "- Environment file: $ENV_FILE"