#\!/bin/bash
set -e

# Function to display usage information
function display_usage {
    echo "Usage: $0 [OPTION]"
    echo "Clean up just the CloudFormation stacks for AI Chat Platform"
    echo ""
    echo "Options:"
    echo "  -e, --environment ENV   Environment to clean (dev, prod) [default: dev]"
    echo "  -r, --region REGION     AWS region [defaults to us-west-2]"
    echo "  -f, --force             Skip confirmation prompt"
    echo "  -h, --help              Display this help message"
    echo ""
}

# Set default values
ENVIRONMENT="dev"
REGION="us-west-2"
FORCE=false

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

# Display cleanup message
echo "WARNING: This will remove the CloudFormation stacks for AI Chat Platform in:"
echo "  - Environment: $ENVIRONMENT"
echo "  - AWS Region: $REGION"

# Ask for confirmation unless force flag is set
if [ "$FORCE" = false ]; then
    echo ""
    read -p "Type 'clean' to confirm: " -r
    if [[ \! $REPLY == "clean" ]]; then
        echo "Cleanup canceled."
        exit 0
    fi
fi

echo "Removing stacks in reverse order..."
STACK_PREFIX="${ENVIRONMENT}-AIChatPlatform"

# Delete in reverse order of deployment
echo "Removing monitoring stack..."
aws cloudformation delete-stack --stack-name ${STACK_PREFIX}-MonitoringStack --region ${REGION}
aws cloudformation wait stack-delete-complete --stack-name ${STACK_PREFIX}-MonitoringStack --region ${REGION}

echo "Removing frontend stack..."
aws cloudformation delete-stack --stack-name ${STACK_PREFIX}-FrontendStack --region ${REGION}
aws cloudformation wait stack-delete-complete --stack-name ${STACK_PREFIX}-FrontendStack --region ${REGION}

echo "Removing API stack..."
aws cloudformation delete-stack --stack-name ${STACK_PREFIX}-ApiStack --region ${REGION}
aws cloudformation wait stack-delete-complete --stack-name ${STACK_PREFIX}-ApiStack --region ${REGION}

echo "Removing auth stack..."
aws cloudformation delete-stack --stack-name ${STACK_PREFIX}-AuthStack --region ${REGION}
aws cloudformation wait stack-delete-complete --stack-name ${STACK_PREFIX}-AuthStack --region ${REGION}

echo "Removing storage stack..."
aws cloudformation delete-stack --stack-name ${STACK_PREFIX}-StorageStack --region ${REGION}
aws cloudformation wait stack-delete-complete --stack-name ${STACK_PREFIX}-StorageStack --region ${REGION}

echo "Removing network stack..."
aws cloudformation delete-stack --stack-name ${STACK_PREFIX}-NetworkStack --region ${REGION}
aws cloudformation wait stack-delete-complete --stack-name ${STACK_PREFIX}-NetworkStack --region ${REGION}

echo "CloudFormation stacks removed successfully\!"
