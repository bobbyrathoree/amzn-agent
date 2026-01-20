# Foundry - Intelligent Bot Platform

Build, share, and chat with intelligent bots powered by AWS Bedrock and RAG.

## Features

- **Multi-Model Support** - Claude 4, Claude 3.7 Sonnet, Haiku, Llama, Mistral, Nova, and more
- **Knowledge Base Integration** - RAG with smart intent classification to skip unnecessary searches
- **Streaming Chat** - Real-time responses with token-by-token streaming
- **Bot Tools** - Web search, research assistant, calculator integrations
- **Guardrails** - Content filtering and safety controls via AWS Bedrock Guardrails
- **API Key Vault** - Secure storage for bot-specific API credentials
- **Bot Marketplace** - Share bots publicly or keep them private
- **Extended Thinking** - Support for Claude's extended thinking mode

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  React + Vite   │────▶│   API Gateway   │────▶│  Go Lambdas     │
│    Frontend     │     │   + WebSocket   │     │  (VPC)          │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                        │
                        ┌───────────────────────────────┼───────────────────────────────┐
                        │                               │                               │
                        ▼                               ▼                               ▼
                ┌───────────────┐             ┌─────────────────┐             ┌─────────────────┐
                │   DynamoDB    │             │  AWS Bedrock    │             │   S3 Storage    │
                │  Bots/Convos  │             │  LLMs + KBs     │             │  Conversations  │
                └───────────────┘             └─────────────────┘             └─────────────────┘
```

**Stack:**
- **Frontend**: React 18, Vite, TailwindCSS, Framer Motion
- **Backend**: Go 1.21+, AWS Lambda (ARM64), API Gateway
- **Storage**: DynamoDB, S3
- **AI**: AWS Bedrock (Claude, Llama, Mistral, Nova), Bedrock Knowledge Bases
- **Auth**: Amazon Cognito
- **IaC**: AWS CDK (TypeScript)

## Quick Start

### Prerequisites
- AWS CLI configured with credentials
- Node.js 18+
- Go 1.21+

### Deploy

```bash
# Clone and deploy to dev environment
git clone <repo-url>
cd foundry

# Deploy everything (builds Go, deploys CDK stacks)
./scripts/deploy.sh

# Or deploy to production
./scripts/deploy.sh -e prod
```

### Local Development

```bash
# Frontend (runs on localhost:5173)
cd react-frontend
npm install
npm run dev

# Backend (build Lambda functions)
cd go-lambda-backend
make build
```

## Directory Structure

```
├── react-frontend/       # React + Vite frontend
├── go-lambda-backend/    # Go Lambda functions
│   ├── cmd/functions/    # Lambda entry points
│   ├── pkg/services/     # Business logic
│   └── internal/         # Handlers
├── infrastructure/       # AWS CDK stacks
├── lambda/               # Built Lambda binaries
└── scripts/              # Deploy/cleanup scripts
```

## Configuration

### Environment Variables (Frontend)

Create `react-frontend/.env.local`:
```env
VITE_USER_POOL_ID=us-west-2_xxxxx
VITE_USER_POOL_CLIENT_ID=xxxxx
VITE_API_ENDPOINT=https://xxx.execute-api.us-west-2.amazonaws.com/dev
VITE_WEBSOCKET_ENDPOINT=wss://xxx.execute-api.us-west-2.amazonaws.com/dev
VITE_AWS_REGION=us-west-2
```

### CDK Context

Deploy with custom domain:
```bash
./scripts/deploy.sh -d chat.example.com -c arn:aws:acm:us-east-1:xxx:certificate/xxx
```

## Scripts

```bash
# Deploy
./scripts/deploy.sh -e dev|prod [-d domain] [-c cert-arn] [-s skip-build]

# Cleanup
./scripts/cleanup.sh -e dev|prod [-f force] [--clean-lambda]
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/bots` | List user's bots |
| POST | `/bots` | Create a bot |
| GET | `/bots/{id}` | Get bot details |
| PUT | `/bots/{id}` | Update bot |
| DELETE | `/bots/{id}` | Delete bot |
| POST | `/bots/{id}/chat` | Chat with bot |
| GET | `/bots/{id}/conversations` | List conversations |
| GET | `/knowledge-bases` | List available KBs |
| GET/POST | `/vault/{botId}/keys` | Manage API keys |

## License

MIT
