# Go Lambda Backend

A well-organized Go application for serverless backend services using AWS Lambda, DynamoDB, and API Gateway.

## Directory Structure

```
go-lambda-backend/
├── cmd/                       # Entry points for applications and lambda functions
│   └── functions/             # Lambda function entrypoints
│       ├── items/             # Items API Lambda function
│       │   └── main.go        # Main Lambda handler for items functionality
│       └── message-processor/ # SQS message processing Lambda
│           └── main.go        # Main Lambda handler for asynchronous processing
├── deployments/               # Deployment configuration
│   └── template.yaml          # AWS SAM template for CloudFormation deployment
├── internal/                  # Private application code
│   ├── handlers/              # Request handlers
│   │   └── item_handler.go    # Item API handlers
│   ├── middleware/            # Middleware components
│   │   └── validator.go       # Request validation middleware
│   └── validators/            # Input validation logic
├── pkg/                       # Public library code that can be imported by external applications
│   ├── config/                # Application configuration
│   │   └── config.go          # Configuration structure and loading
│   ├── models/                # Data models
│   │   └── item.go            # Item data model and DTOs
│   ├── repositories/          # Data access layer
│   │   └── item_repository.go # Repository for item storage operations
│   ├── services/              # Business logic layer
│   │   └── item_service.go    # Item business logic
│   └── utils/                 # Utility functions
│       └── response.go        # HTTP response utilities
├── tests/                     # Test files
│   └── services/              # Service tests
│       └── item_service_test.go # Tests for item service
├── go.mod                     # Go module definition
├── go.sum                     # Go dependencies checksums
├── Makefile                   # Build and deployment tasks
└── README.md                  # Project documentation
```

## Key Components

### Lambda Functions

Located in `cmd/functions/`, each subfolder corresponds to a separate Lambda function:

- `items/main.go`: Handles CRUD operations for items through AWS API Gateway
- `message-processor/main.go`: Processes messages from SQS queue for background tasks

### Handlers

Located in `internal/handlers/`, these process API Gateway events:

- `item_handler.go`: Routes API Gateway requests to appropriate service methods

### Services

Located in `pkg/services/`, they contain business logic:

- `item_service.go`: Business logic for item operations

### Repositories

Located in `pkg/repositories/`, they handle data persistence:

- `item_repository.go`: DynamoDB operations for item data

### Models

Located in `pkg/models/`, they define data structures:

- `item.go`: Data models for items and API requests/responses

### Configuration

Located in `pkg/config/`, handles app settings:

- `config.go`: Loads configuration from environment variables and Secrets Manager

## Development

### Prerequisites

- Go 1.22 or later
- AWS CLI
- AWS SAM CLI
- Docker (for local testing)

### Building

```bash
# Build all Lambda functions
make build

# Run tests
make test
```

### Local Testing

Start a local DynamoDB:

```bash
make run-local-dynamodb
make create-local-dynamodb
```

### Deployment

```bash
# Deploy all functions
make deploy S3_BUCKET=your-deployment-bucket

# Deploy a single function
make deploy-function FUNCTION=items
```

## AWS Resources

This project uses the following AWS services:

- **API Gateway**: REST API endpoints
- **Lambda**: Serverless function execution
- **DynamoDB**: NoSQL database for item storage
- **SQS**: Message queuing for asynchronous processing
- **Cognito**: User authentication and authorization
- **Secrets Manager**: Storage for sensitive configuration

## API Structure

### Items API

- `GET /items`: List all items (paginated)
- `GET /items/{id}`: Get a specific item
- `POST /items`: Create a new item
- `PUT /items/{id}`: Update an existing item
- `DELETE /items/{id}`: Delete an item