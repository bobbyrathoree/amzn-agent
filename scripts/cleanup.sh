#!/bin/bash
set -e

# ASCII art banner
echo "
 ██████╗██╗     ███████╗ █████╗ ███╗   ██╗██╗   ██╗██████╗ 
██╔════╝██║     ██╔════╝██╔══██╗████╗  ██║██║   ██║██╔══██╗
██║     ██║     █████╗  ███████║██╔██╗ ██║██║   ██║██████╔╝
██║     ██║     ██╔══╝  ██╔══██║██║╚██╗██║██║   ██║██╔═══╝ 
╚██████╗███████╗███████╗██║  ██║██║ ╚████║╚██████╔╝██║     
 ╚═════╝╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝ ╚═╝     
"

# Function to display usage information
function display_usage {
    echo "Usage: $0 [OPTION]"
    echo "Destroy the AI Chat Platform infrastructure"
    echo ""
    echo "Options:"
    echo "  -e, --environment ENV   Environment to destroy (dev, prod) [default: dev]"
    echo "  -r, --region REGION     AWS region [defaults to AWS CLI configured region]"
    echo "  -f, --force             Skip confirmation prompt"
    echo "  --clean-lambda          Also clean Lambda function builds"
    echo "  --debug                 Print debug information"
    echo "  -h, --help              Display this help message"
    echo ""
}

# Set default values
ENVIRONMENT="dev"
REGION=""
FORCE=false
CLEAN_LAMBDA=false
DEBUG=false

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -r|--region)
            REGION="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        --clean-lambda)
            CLEAN_LAMBDA=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        -h|--help)
            display_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            display_usage
            exit 1
            ;;
    esac
done

# Validate environment
if [[ "$ENVIRONMENT" != "dev" && "$ENVIRONMENT" != "prod" ]]; then
    echo "Error: Environment must be either 'dev' or 'prod'"
    exit 1
fi

# Check for AWS CLI and credentials
if ! command -v aws &> /dev/null; then
    echo "Error: AWS CLI is not installed. Please install the AWS CLI."
    exit 1
fi

# Get account ID from current AWS credentials
ACCOUNT_ID=$(aws sts get-caller-identity --query "Account" --output text 2>/dev/null)
if [ $? -ne 0 ] || [ -z "$ACCOUNT_ID" ]; then
    echo "Error: Unable to determine AWS account ID. Please check your AWS credentials."
    exit 1
fi

# Get default region if not specified
if [ -z "$REGION" ]; then
    REGION=$(aws configure get region 2>/dev/null)
    if [ -z "$REGION" ]; then
        echo "Error: No AWS region specified or found in AWS CLI configuration."
        echo "Please specify a region with -r/--region or configure the AWS CLI."
        exit 1
    fi
fi

# Display information
echo "WARNING: This will DESTROY the AI Chat Platform in:"
echo "  - Environment: $ENVIRONMENT"
echo "  - AWS Region: $REGION"
echo "  - AWS Account: $ACCOUNT_ID"
echo ""
echo "⚠️  This action is IRREVERSIBLE! All data will be lost! ⚠️"

# Ask for confirmation unless force flag is set
if [ "$FORCE" = false ]; then
    echo ""
    read -p "Type 'destroy' to confirm: " -r
    if [[ ! $REPLY == "destroy" ]]; then
        echo "Cleanup canceled."
        exit 0
    fi
fi

# Navigate to project root directory
PROJECT_ROOT="$(dirname "$0")/.."
cd "$PROJECT_ROOT"

# Set up environment variables
export CDK_DEFAULT_ACCOUNT=$ACCOUNT_ID
export CDK_DEFAULT_REGION=$REGION

# Change to infrastructure directory
cd infrastructure

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Destroy the infrastructure
echo "Destroying infrastructure in $ENVIRONMENT environment..."
npm run destroy:$ENVIRONMENT -- --force

# Clean Lambda functions if requested
if [ "$CLEAN_LAMBDA" = true ]; then
    echo "Cleaning Lambda function builds..."
    cd "../go-lambda-backend"
    if [ -f "Makefile" ]; then
        make clean
    else
        echo "Makefile not found, skipping Lambda cleanup."
    fi
fi

echo ""
echo "Cleanup complete! All resources have been destroyed."