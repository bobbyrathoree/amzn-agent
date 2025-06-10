# AI Chat Platform with AWS Bedrock

A scalable AI chat platform built on AWS services that leverages AWS Bedrock for generative AI capabilities. This platform enables users to create, manage, and chat with AI bots powered by large language models.

## Architecture

The system consists of several key components:

- **Frontend**: Next.js application with Vercel AI SDK integration
- **Backend**: Go Lambda functions exposed via API Gateway
- **Storage**: DynamoDB for structured data, S3 for file storage
- **AI**: AWS Bedrock for large language models and knowledge bases
- **Authentication**: Amazon Cognito for user management
- **Networking**: CloudFront for content delivery

## Features

- Create and manage AI bots
- Chat with AI bots using various Bedrock models
- Integration with Bedrock knowledge bases for RAG (Retrieval Augmented Generation)
- Real-time chat with streaming responses
- User authentication and authorization
- Monitoring and alerts

## Directory Structure

```
/
├── docs/                 # Documentation
├── frontend/             # Next.js frontend application
├── go-lambda-backend/    # Go Lambda functions
├── infrastructure/       # AWS CDK infrastructure code
├── lambda/               # Built Lambda functions for deployment
└── scripts/              # Utility scripts for deployment and management
```

## Prerequisites

- AWS Account with configured AWS CLI credentials
- Node.js 18 or later
- Go 1.20 or later

## Getting Started

### Initial Setup

1. Clone this repository:

```bash
git clone <repository-url>
cd ai-chat-platform
```

2. Set up environment variables (optional):

```bash
cd infrastructure
cp .env.example .env
# Edit .env with your values
```

### Deployment

The deployment script will automatically:
- Build the Go Lambda functions 
- Install Node.js dependencies
- Build the TypeScript CDK code
- Deploy all stacks to AWS

```bash
# Deploy to development environment
./scripts/deploy.sh

# Deploy to production environment
./scripts/deploy.sh -e prod
```

### Clean Up Resources

To remove all deployed resources:

```bash
./scripts/cleanup.sh
```

## Using the Scripts

### Deployment Script

```bash
./scripts/deploy.sh --help

Options:
  -e, --environment ENV   Deploy to environment (dev, prod) [default: dev]
  -r, --region REGION     AWS region to deploy to [defaults to AWS CLI configured region]
  -d, --domain DOMAIN     Custom domain name (optional)
  -c, --cert-arn ARN      ACM certificate ARN for custom domain (optional)
  -s, --skip-build        Skip building the Go Lambda functions
  -h, --help              Display this help message
```

### Cleanup Script

```bash
./scripts/cleanup.sh --help

Options:
  -e, --environment ENV   Environment to destroy (dev, prod) [default: dev]
  -r, --region REGION     AWS region [defaults to AWS CLI configured region]
  -f, --force             Skip confirmation prompt
  --clean-lambda          Also clean Lambda function builds
  -h, --help              Display this help message
```

## Manual Building

### Go Lambda Functions

You can manually build the Lambda functions:

```bash
cd go-lambda-backend
make build  # Builds all functions
make chat   # Builds only the chat function
```

### Infrastructure

You can manually build and deploy the infrastructure:

```bash
cd infrastructure
npm install
npm run build
npm run deploy:dev
```

## Development

### Frontend Development

```bash
cd frontend
npm install
npm run dev
```

### Backend Development

```bash
cd go-lambda-backend
go mod tidy
go test ./...
```

### Infrastructure Development

```bash
cd infrastructure
npm install
npm run diff
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.