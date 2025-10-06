#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
ENVIRONMENT="dev"
REGION=""
DEBUG=false
FORCE_YES=false

# Function to display usage
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Deploy Foundry infrastructure to AWS"
    echo ""
    echo "Options:"
    echo "  -e, --environment ENV    Environment to deploy (dev|prod) [default: dev]"
    echo "  -r, --region REGION      AWS region [default: from AWS config]"
    echo "  -y, --yes                Skip confirmation prompts"
    echo "  -d, --debug              Enable debug mode"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                       # Deploy to dev environment"
    echo "  $0 -e prod -r us-east-1  # Deploy to prod in us-east-1"
    echo "  $0 -e prod -y            # Deploy to prod with auto-confirmation"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -r|--region)
            REGION="$2"
            shift 2
            ;;
        -y|--yes)
            FORCE_YES=true
            shift
            ;;
        -d|--debug)
            DEBUG=true
            set -x
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(dev|prod)$ ]]; then
    echo -e "${RED}Error: Environment must be 'dev' or 'prod'${NC}"
    exit 1
fi

# Set region if not provided
if [[ -z "$REGION" ]]; then
    REGION=$(aws configure get region 2>/dev/null || echo "us-east-1")
fi

# ASCII Art Banner
echo -e "${BLUE}"
echo "    ___                                   ____            __    __     "
echo "   /   |  ____ ___  ____ _____  ____    / __ )__  ______/ /___/ /_  __"
echo "  / /| | / __ \`__ \/ __ \`/_  / / __ \  / __  / / / / __  / __  / / / /"
echo " / ___ |/ / / / / / /_/ / / /_ / /_/ / / /_/ / /_/ / /_/ / /_/ / /_/ / "
echo "/_/  |_/_/ /_/ /_/\__,_/ /___/\____/ /_____/\__,_/\__,_/\__,_/\__, /  "
echo "                                                             /____/   "
echo -e "${NC}"

echo -e "${GREEN}🚀 Foundry Deployment Script${NC}"
echo "================================="
echo -e "Environment: ${YELLOW}$ENVIRONMENT${NC}"
echo -e "Region:      ${YELLOW}$REGION${NC}"
echo -e "AWS Account: ${YELLOW}$(aws sts get-caller-identity --query Account --output text 2>/dev/null || echo 'Unknown')${NC}"
echo ""

# Confirmation prompt
if [[ "$FORCE_YES" != true ]]; then
    echo -e "${YELLOW}This will deploy Foundry infrastructure to the above environment.${NC}"
    read -p "Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Deployment cancelled."
        exit 0
    fi
fi

echo -e "${BLUE}📁 Working directory: $(pwd)${NC}"

# Step 1: Build Go Lambda functions
echo -e "\n${YELLOW}🏗️  Building Go Lambda functions...${NC}"
cd go-lambda-backend
if ! make build; then
    echo -e "${RED}❌ Failed to build Go Lambda functions${NC}"
    exit 1
fi
cd ..
echo -e "${GREEN}✅ Go Lambda functions built successfully${NC}"

# Step 2: Build React frontend
echo -e "\n${YELLOW}🎨 Building React frontend...${NC}"
cd react-frontend

# Install dependencies if needed
if [[ ! -d "node_modules" ]]; then
    echo "Installing React frontend dependencies..."
    npm install
fi

# Build the React app
if ! npm run build; then
    echo -e "${RED}❌ Failed to build React frontend${NC}"
    exit 1
fi
cd ..
echo -e "${GREEN}✅ React frontend built successfully${NC}"

# Step 3: Deploy infrastructure
echo -e "\n${YELLOW}☁️  Deploying infrastructure...${NC}"
cd infrastructure

# Install CDK dependencies if needed
if [[ ! -d "node_modules" ]]; then
    echo "Installing infrastructure dependencies..."
    npm install
fi

# Build CDK project
echo "Building CDK project..."
if ! npm run build; then
    echo -e "${RED}❌ Failed to build CDK project${NC}"
    exit 1
fi

# Deploy all stacks
echo -e "\n${YELLOW}🚀 Deploying stacks to $ENVIRONMENT environment...${NC}"
export CDK_DEFAULT_REGION="$REGION"

if ! npx cdk deploy --all --require-approval never --context env="$ENVIRONMENT"; then
    echo -e "${RED}❌ Failed to deploy infrastructure${NC}"
    exit 1
fi

cd ..

# Step 4: Get deployment outputs
echo -e "\n${YELLOW}📊 Getting deployment outputs...${NC}"
STACK_PREFIX="${ENVIRONMENT}-Foundry-"

# Get stack outputs
WEBSITE_URL=$(aws cloudformation describe-stacks \
    --stack-name "${STACK_PREFIX}FrontendStack" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`WebsiteURL`].OutputValue' \
    --output text 2>/dev/null || echo "Not found")

API_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "${STACK_PREFIX}ApiStack" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`RestApiEndpoint`].OutputValue' \
    --output text 2>/dev/null || echo "Not found")

WEBSOCKET_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "${STACK_PREFIX}ApiStack" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`WebSocketEndpoint`].OutputValue' \
    --output text 2>/dev/null || echo "Not found")

USER_POOL_ID=$(aws cloudformation describe-stacks \
    --stack-name "${STACK_PREFIX}AuthStack" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`UserPoolId`].OutputValue' \
    --output text 2>/dev/null || echo "Not found")

USER_POOL_CLIENT_ID=$(aws cloudformation describe-stacks \
    --stack-name "${STACK_PREFIX}AuthStack" \
    --region "$REGION" \
    --query 'Stacks[0].Outputs[?OutputKey==`UserPoolClientId`].OutputValue' \
    --output text 2>/dev/null || echo "Not found")

# Success message
echo -e "\n${GREEN}🎉 DEPLOYMENT SUCCESSFUL! 🎉${NC}"
echo "========================================="
echo ""
echo -e "${BLUE}📱 Application URLs:${NC}"
echo -e "Website:    ${GREEN}$WEBSITE_URL${NC}"
echo -e "API:        ${GREEN}$API_ENDPOINT${NC}"
echo -e "WebSocket:  ${GREEN}$WEBSOCKET_ENDPOINT${NC}"
echo ""
echo -e "${BLUE}🔐 Authentication:${NC}"
echo -e "User Pool ID:        ${GREEN}$USER_POOL_ID${NC}"
echo -e "User Pool Client ID: ${GREEN}$USER_POOL_CLIENT_ID${NC}"
echo ""
echo -e "${YELLOW}💡 Next Steps:${NC}"
echo "1. Visit the website URL to access Foundry"
echo "2. Create an account or sign in"
echo "3. Start creating and chatting with bots!"
echo ""
echo -e "${GREEN}Deployment completed successfully! 🚀${NC}"