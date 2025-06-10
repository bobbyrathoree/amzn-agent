# Go Lambda Backend Structure

This document outlines the structure and key components of the Go Lambda backend that powers our AI Chat platform.

## Directory Structure

```
go-lambda-backend/
├── cmd/                       # Entry points for applications and lambda functions
│   └── functions/             # Lambda function entrypoints
│       ├── chat/             # Chat API Lambda function
│       │   └── main.go        # Main Lambda handler for chat functionality
│       ├── bots/             # Bot management Lambda function
│       │   └── main.go        # Main Lambda handler for bot CRUD operations
│       ├── knowledge/        # Knowledge base Lambda function
│       │   └── main.go        # Main Lambda handler for knowledge base operations
│       └── websocket/        # WebSocket Lambda function
│           └── main.go        # Main Lambda handler for WebSocket connections
├── deployments/               # Deployment configuration
│   └── template.yaml          # AWS SAM template for CloudFormation deployment
├── internal/                  # Private application code
│   ├── handlers/              # Request handlers
│   │   ├── chat_handler.go    # Chat API handlers
│   │   ├── bot_handler.go     # Bot management handlers
│   │   └── knowledge_handler.go # Knowledge base handlers
│   ├── middleware/            # Middleware components
│   │   ├── auth.go            # Authentication middleware
│   │   ├── logger.go          # Logging middleware
│   │   └── validator.go       # Request validation middleware
│   └── validators/            # Input validation logic
│       └── validator.go       # Request validation functions
├── pkg/                       # Public library code that can be imported by external applications
│   ├── config/                # Application configuration
│   │   └── config.go          # Configuration structure and loading
│   ├── models/                # Data models
│   │   ├── bot.go             # Bot data model and DTOs
│   │   ├── chat.go            # Chat data model and DTOs
│   │   └── knowledge.go       # Knowledge base data model and DTOs
│   ├── repositories/          # Data access layer
│   │   ├── bot_repository.go  # Repository for bot storage operations
│   │   ├── chat_repository.go # Repository for chat storage operations
│   │   └── knowledge_repository.go # Repository for knowledge base operations
│   ├── services/              # Business logic layer
│   │   ├── bot_service.go     # Bot business logic
│   │   ├── chat_service.go    # Chat business logic
│   │   ├── bedrock_service.go # AWS Bedrock integration service
│   │   └── knowledge_service.go # Knowledge base business logic
│   └── utils/                 # Utility functions
│       ├── logger.go          # Logging utilities
│       ├── aws.go             # AWS helper functions
│       └── response.go        # HTTP response utilities
├── tests/                     # Test files
│   └── services/              # Service tests
│       └── chat_service_test.go # Tests for chat service
├── go.mod                     # Go module definition
├── Makefile                   # Build and deployment tasks
└── README.md                  # Project documentation
```

## Key Components and Files

### 1. Lambda Functions (`cmd/functions/`)

#### Chat Lambda (`cmd/functions/chat/main.go`)

```go
package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	"go-lambda-backend/internal/handlers"
	"go-lambda-backend/pkg/config"
	"go-lambda-backend/pkg/repositories"
	"go-lambda-backend/pkg/services"
)

var chatHandler *handlers.ChatHandler

func init() {
	// Load configuration
	cfg, err := config.LoadAppConfig()
	if err != nil {
		panic("failed to load application configuration: " + err.Error())
	}

	// Initialize AWS SDK configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		panic("failed to load AWS configuration: " + err.Error())
	}

	// Initialize repositories
	chatRepo := repositories.NewChatRepository(awsCfg)
	botRepo := repositories.NewBotRepository(awsCfg)

	// Initialize services
	bedrockSvc := services.NewBedrockService(awsCfg)
	chatSvc := services.NewChatService(chatRepo, botRepo, bedrockSvc)

	// Initialize handlers
	chatHandler = handlers.NewChatHandler(chatSvc)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract auth information
	userId := request.RequestContext.Authorizer["userId"].(string)
	
	// Route to appropriate handler based on HTTP method and path
	switch {
	case request.HTTPMethod == "POST" && request.Resource == "/chat":
		return chatHandler.HandleChatMessage(ctx, request, userId)
	case request.HTTPMethod == "GET" && request.Resource == "/conversations":
		return chatHandler.HandleListConversations(ctx, request, userId)
	case request.HTTPMethod == "GET" && request.Resource == "/conversations/{id}":
		return chatHandler.HandleGetConversation(ctx, request, userId)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"message": "Not found"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}
}

func main() {
	lambda.Start(handleRequest)
}
```

#### Bot Management Lambda (`cmd/functions/bots/main.go`)

```go
package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	"go-lambda-backend/internal/handlers"
	"go-lambda-backend/pkg/config"
	"go-lambda-backend/pkg/repositories"
	"go-lambda-backend/pkg/services"
)

var botHandler *handlers.BotHandler

func init() {
	// Load configuration
	cfg, err := config.LoadAppConfig()
	if err != nil {
		panic("failed to load application configuration: " + err.Error())
	}

	// Initialize AWS SDK configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		panic("failed to load AWS configuration: " + err.Error())
	}

	// Initialize repositories
	botRepo := repositories.NewBotRepository(awsCfg)
	knowledgeRepo := repositories.NewKnowledgeRepository(awsCfg)

	// Initialize services
	botSvc := services.NewBotService(botRepo, knowledgeRepo)

	// Initialize handlers
	botHandler = handlers.NewBotHandler(botSvc)
}

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract auth information
	userId := request.RequestContext.Authorizer["userId"].(string)
	
	// Route to appropriate handler based on HTTP method and path
	switch {
	case request.HTTPMethod == "GET" && request.Resource == "/bots":
		return botHandler.HandleListBots(ctx, request, userId)
	case request.HTTPMethod == "GET" && request.Resource == "/bots/{id}":
		return botHandler.HandleGetBot(ctx, request, userId)
	case request.HTTPMethod == "POST" && request.Resource == "/bots":
		return botHandler.HandleCreateBot(ctx, request, userId)
	case request.HTTPMethod == "PUT" && request.Resource == "/bots/{id}":
		return botHandler.HandleUpdateBot(ctx, request, userId)
	case request.HTTPMethod == "DELETE" && request.Resource == "/bots/{id}":
		return botHandler.HandleDeleteBot(ctx, request, userId)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"message": "Not found"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}
}

func main() {
	lambda.Start(handleRequest)
}
```

#### WebSocket Lambda (`cmd/functions/websocket/main.go`)

```go
package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	"go-lambda-backend/internal/handlers"
	"go-lambda-backend/pkg/config"
	"go-lambda-backend/pkg/repositories"
	"go-lambda-backend/pkg/services"
)

var websocketHandler *handlers.WebSocketHandler

func init() {
	// Load configuration
	cfg, err := config.LoadAppConfig()
	if err != nil {
		panic("failed to load application configuration: " + err.Error())
	}

	// Initialize AWS SDK configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		panic("failed to load AWS configuration: " + err.Error())
	}

	// Initialize repositories
	connectionRepo := repositories.NewConnectionRepository(awsCfg)
	chatRepo := repositories.NewChatRepository(awsCfg)
	botRepo := repositories.NewBotRepository(awsCfg)

	// Initialize services
	bedrockSvc := services.NewBedrockService(awsCfg)
	chatSvc := services.NewChatService(chatRepo, botRepo, bedrockSvc)
	websocketSvc := services.NewWebSocketService(connectionRepo)

	// Initialize handlers
	websocketHandler = handlers.NewWebSocketHandler(websocketSvc, chatSvc)
}

func handleRequest(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Handle WebSocket events
	switch request.RequestContext.EventType {
	case "CONNECT":
		return websocketHandler.HandleConnect(ctx, request)
	case "DISCONNECT":
		return websocketHandler.HandleDisconnect(ctx, request)
	case "MESSAGE":
		return websocketHandler.HandleMessage(ctx, request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"message": "Unsupported event type"}`,
		}, nil
	}
}

func main() {
	lambda.Start(handleRequest)
}
```

### 2. Service Layer (`pkg/services/`)

#### Bedrock Service (`pkg/services/bedrock_service.go`)

```go
package services

import (
	"context"
	"encoding/json"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"go-lambda-backend/pkg/models"
)

// BedrockService provides methods to interact with AWS Bedrock
type BedrockService struct {
	client *bedrockruntime.Client
}

// NewBedrockService creates a new BedrockService instance
func NewBedrockService(cfg aws.Config) *BedrockService {
	return &BedrockService{
		client: bedrockruntime.NewFromConfig(cfg),
	}
}

// GenerateResponse generates a response from Bedrock model
func (s *BedrockService) GenerateResponse(ctx context.Context, input *models.ChatInput) (*models.ChatResponse, error) {
	// Prepare request payload based on the model
	payload, err := s.prepareRequestPayload(input)
	if err != nil {
		return nil, err
	}

	// Call Bedrock model
	resp, err := s.client.InvokeModelWithResponseStream(ctx, &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(input.ModelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return nil, err
	}

	return s.processStreamResponse(resp.GetStream())
}

// prepareRequestPayload prepares the request payload for the model
func (s *BedrockService) prepareRequestPayload(input *models.ChatInput) ([]byte, error) {
	// Determine payload format based on model ID
	if isAnthropicModel(input.ModelID) {
		return s.prepareAnthropicPayload(input)
	} else if isLlamaModel(input.ModelID) {
		return s.prepareLlamaPayload(input)
	} else {
		return s.prepareDefaultPayload(input)
	}
}

// processStreamResponse processes the streaming response
func (s *BedrockService) processStreamResponse(stream io.ReadCloser) (*models.ChatResponse, error) {
	defer stream.Close()

	// Process the streaming response
	// Implementation depends on the response format of the chosen model
	// Here we'll use a simplified implementation
	
	response := &models.ChatResponse{
		Content: "",
	}
	
	// Read and process chunks from stream
	// This is a simplified example
	buffer := make([]byte, 4096)
	for {
		n, err := stream.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		// Process chunk (this would need to handle different model formats)
		chunk := buffer[:n]
		var resp map[string]interface{}
		if err := json.Unmarshal(chunk, &resp); err != nil {
			return nil, err
		}
		
		// Extract content based on model format
		// This is simplified and would need to be adapted for each model
		if content, ok := resp["completion"].(string); ok {
			response.Content += content
		}
	}
	
	return response, nil
}

// Helper functions to determine model type
func isAnthropicModel(modelID string) bool {
	// Check if model ID is from Anthropic (Claude)
	return len(modelID) >= 9 && modelID[:9] == "anthropic."
}

func isLlamaModel(modelID string) bool {
	// Check if model ID is from Meta (Llama)
	return len(modelID) >= 5 && modelID[:5] == "meta."
}

// Model-specific payload preparation functions
func (s *BedrockService) prepareAnthropicPayload(input *models.ChatInput) ([]byte, error) {
	// Structure for Anthropic models (Claude)
	type AnthropicMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	
	anthropicPayload := struct {
		System    string            `json:"system,omitempty"`
		Messages  []AnthropicMessage `json:"messages"`
		MaxTokens int               `json:"max_tokens"`
	}{
		System:    input.SystemPrompt,
		MaxTokens: input.MaxTokens,
		Messages:  []AnthropicMessage{},
	}
	
	// Convert messages to Anthropic format
	for _, msg := range input.Messages {
		anthropicPayload.Messages = append(anthropicPayload.Messages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	return json.Marshal(anthropicPayload)
}

func (s *BedrockService) prepareLlamaPayload(input *models.ChatInput) ([]byte, error) {
	// Implementation for Llama models
	// ...
	return nil, nil
}

func (s *BedrockService) prepareDefaultPayload(input *models.ChatInput) ([]byte, error) {
	// Generic implementation for other models
	// ...
	return nil, nil
}
```

#### Chat Service (`pkg/services/chat_service.go`)

```go
package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"go-lambda-backend/pkg/models"
	"go-lambda-backend/pkg/repositories"
)

// ChatService provides methods for chat functionality
type ChatService struct {
	chatRepo    repositories.ChatRepository
	botRepo     repositories.BotRepository
	bedrockSvc  *BedrockService
}

// NewChatService creates a new ChatService
func NewChatService(
	chatRepo repositories.ChatRepository,
	botRepo repositories.BotRepository,
	bedrockSvc *BedrockService,
) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		botRepo:     botRepo,
		bedrockSvc:  bedrockSvc,
	}
}

// CreateChatMessage creates a new chat message and gets a response
func (s *ChatService) CreateChatMessage(ctx context.Context, input models.ChatInput, userID string) (*models.ChatResponse, error) {
	// Get the bot information
	bot, err := s.botRepo.GetBot(ctx, input.BotID, userID)
	if err != nil {
		return nil, err
	}
	
	// Add system prompt from bot if not provided
	if input.SystemPrompt == "" && bot.SystemPrompt != "" {
		input.SystemPrompt = bot.SystemPrompt
	}
	
	// Set default model from bot if not provided
	if input.ModelID == "" && bot.DefaultModel != "" {
		input.ModelID = bot.DefaultModel
	}
	
	// Generate response from Bedrock
	response, err := s.bedrockSvc.GenerateResponse(ctx, &input)
	if err != nil {
		return nil, err
	}
	
	// Create conversation if it doesn't exist
	var conversationID string
	if input.ConversationID == "" {
		// Create new conversation
		conversation := models.Conversation{
			ID:        uuid.New().String(),
			UserID:    userID,
			BotID:     input.BotID,
			Title:     "New Conversation", // Can be updated later
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		
		if err := s.chatRepo.CreateConversation(ctx, conversation); err != nil {
			return nil, err
		}
		
		conversationID = conversation.ID
	} else {
		conversationID = input.ConversationID
	}
	
	// Save user message
	userMessage := models.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "user",
		Content:        input.Messages[len(input.Messages)-1].Content,
		CreatedAt:      time.Now().UTC(),
	}
	
	if err := s.chatRepo.CreateMessage(ctx, userMessage); err != nil {
		return nil, err
	}
	
	// Save assistant response
	assistantMessage := models.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response.Content,
		CreatedAt:      time.Now().UTC(),
	}
	
	if err := s.chatRepo.CreateMessage(ctx, assistantMessage); err != nil {
		return nil, err
	}
	
	// Update conversation timestamp
	if err := s.chatRepo.UpdateConversationTimestamp(ctx, conversationID, time.Now().UTC()); err != nil {
		return nil, err
	}
	
	// Set conversation ID in response
	response.ConversationID = conversationID
	
	return response, nil
}

// GetConversation gets a conversation by ID
func (s *ChatService) GetConversation(ctx context.Context, conversationID string, userID string) (*models.ConversationWithMessages, error) {
	// Get conversation
	conversation, err := s.chatRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	
	// Get messages
	messages, err := s.chatRepo.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	
	return &models.ConversationWithMessages{
		Conversation: *conversation,
		Messages:     messages,
	}, nil
}

// ListConversations lists all conversations for a user
func (s *ChatService) ListConversations(ctx context.Context, userID string, limit int, offset int) ([]models.Conversation, error) {
	return s.chatRepo.ListConversations(ctx, userID, limit, offset)
}

// DeleteConversation deletes a conversation
func (s *ChatService) DeleteConversation(ctx context.Context, conversationID string, userID string) error {
	// Check if conversation exists and belongs to user
	_, err := s.chatRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	
	// Delete all messages first
	if err := s.chatRepo.DeleteMessages(ctx, conversationID); err != nil {
		return err
	}
	
	// Delete conversation
	return s.chatRepo.DeleteConversation(ctx, conversationID, userID)
}
```

#### Bot Service (`pkg/services/bot_service.go`)

```go
package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"go-lambda-backend/pkg/models"
	"go-lambda-backend/pkg/repositories"
)

// BotService provides methods for bot management
type BotService struct {
	botRepo       repositories.BotRepository
	knowledgeRepo repositories.KnowledgeRepository
}

// NewBotService creates a new BotService
func NewBotService(
	botRepo repositories.BotRepository,
	knowledgeRepo repositories.KnowledgeRepository,
) *BotService {
	return &BotService{
		botRepo:       botRepo,
		knowledgeRepo: knowledgeRepo,
	}
}

// CreateBot creates a new bot
func (s *BotService) CreateBot(ctx context.Context, input models.CreateBotInput, userID string) (*models.Bot, error) {
	// Generate ID for the new bot
	botID := uuid.New().String()
	
	// Create bot
	bot := models.Bot{
		ID:             botID,
		Name:           input.Name,
		Description:    input.Description,
		OwnerID:        userID,
		SystemPrompt:   input.SystemPrompt,
		IsPublic:       input.IsPublic,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		KnowledgeBaseID: input.KnowledgeBaseID,
		DefaultModel:   input.DefaultModel,
	}
	
	// Validate knowledge base ID if provided
	if input.KnowledgeBaseID != "" {
		// Check if knowledge base exists
		exists, err := s.knowledgeRepo.KnowledgeBaseExists(ctx, input.KnowledgeBaseID)
		if err != nil {
			return nil, err
		}
		
		if !exists {
			return nil, models.ErrKnowledgeBaseNotFound
		}
	}
	
	// Save bot to repository
	if err := s.botRepo.CreateBot(ctx, bot); err != nil {
		return nil, err
	}
	
	return &bot, nil
}

// GetBot gets a bot by ID
func (s *BotService) GetBot(ctx context.Context, botID string, userID string) (*models.Bot, error) {
	return s.botRepo.GetBot(ctx, botID, userID)
}

// ListBots lists all bots accessible to a user
func (s *BotService) ListBots(ctx context.Context, userID string, onlyOwned bool, limit int, offset int) ([]models.Bot, error) {
	return s.botRepo.ListBots(ctx, userID, onlyOwned, limit, offset)
}

// UpdateBot updates a bot
func (s *BotService) UpdateBot(ctx context.Context, botID string, input models.UpdateBotInput, userID string) (*models.Bot, error) {
	// Get existing bot
	bot, err := s.botRepo.GetBot(ctx, botID, userID)
	if err != nil {
		return nil, err
	}
	
	// Check ownership
	if bot.OwnerID != userID {
		return nil, models.ErrNotAuthorized
	}
	
	// Update fields if provided
	if input.Name != nil {
		bot.Name = *input.Name
	}
	
	if input.Description != nil {
		bot.Description = *input.Description
	}
	
	if input.SystemPrompt != nil {
		bot.SystemPrompt = *input.SystemPrompt
	}
	
	if input.IsPublic != nil {
		bot.IsPublic = *input.IsPublic
	}
	
	if input.DefaultModel != nil {
		bot.DefaultModel = *input.DefaultModel
	}
	
	if input.KnowledgeBaseID != nil {
		// Validate knowledge base ID
		if *input.KnowledgeBaseID != "" {
			exists, err := s.knowledgeRepo.KnowledgeBaseExists(ctx, *input.KnowledgeBaseID)
			if err != nil {
				return nil, err
			}
			
			if !exists {
				return nil, models.ErrKnowledgeBaseNotFound
			}
		}
		
		bot.KnowledgeBaseID = *input.KnowledgeBaseID
	}
	
	// Update timestamp
	bot.UpdatedAt = time.Now().UTC()
	
	// Save updated bot
	if err := s.botRepo.UpdateBot(ctx, *bot); err != nil {
		return nil, err
	}
	
	return bot, nil
}

// DeleteBot deletes a bot
func (s *BotService) DeleteBot(ctx context.Context, botID string, userID string) error {
	// Get existing bot
	bot, err := s.botRepo.GetBot(ctx, botID, userID)
	if err != nil {
		return err
	}
	
	// Check ownership
	if bot.OwnerID != userID {
		return models.ErrNotAuthorized
	}
	
	// Delete bot
	return s.botRepo.DeleteBot(ctx, botID, userID)
}
```

### 3. Repository Layer (`pkg/repositories/`)

#### Bot Repository (`pkg/repositories/bot_repository.go`)

```go
package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"go-lambda-backend/pkg/models"
)

// BotRepository provides methods to interact with bot storage
type BotRepository interface {
	CreateBot(ctx context.Context, bot models.Bot) error
	GetBot(ctx context.Context, botID string, userID string) (*models.Bot, error)
	ListBots(ctx context.Context, userID string, onlyOwned bool, limit int, offset int) ([]models.Bot, error)
	UpdateBot(ctx context.Context, bot models.Bot) error
	DeleteBot(ctx context.Context, botID string, userID string) error
}

// DynamoBotRepository implements BotRepository using DynamoDB
type DynamoBotRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewBotRepository creates a new DynamoBotRepository
func NewBotRepository(cfg aws.Config) BotRepository {
	return &DynamoBotRepository{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: "Bots", // This could be loaded from configuration
	}
}

// CreateBot creates a new bot in DynamoDB
func (r *DynamoBotRepository) CreateBot(ctx context.Context, bot models.Bot) error {
	item, err := attributevalue.MarshalMap(bot)
	if err != nil {
		return fmt.Errorf("failed to marshal bot: %w", err)
	}
	
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	
	return err
}

// GetBot gets a bot by ID
func (r *DynamoBotRepository) GetBot(ctx context.Context, botID string, userID string) (*models.Bot, error) {
	// Query the bot
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: botID},
		},
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}
	
	if result.Item == nil {
		return nil, models.ErrBotNotFound
	}
	
	// Unmarshal the bot
	var bot models.Bot
	if err := attributevalue.UnmarshalMap(result.Item, &bot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bot: %w", err)
	}
	
	// Check if user can access this bot
	if bot.OwnerID != userID && !bot.IsPublic {
		return nil, models.ErrNotAuthorized
	}
	
	return &bot, nil
}

// ListBots lists all bots accessible to a user
func (r *DynamoBotRepository) ListBots(ctx context.Context, userID string, onlyOwned bool, limit int, offset int) ([]models.Bot, error) {
	var queryInput *dynamodb.QueryInput
	
	if onlyOwned {
		// Query only bots owned by the user
		queryInput = &dynamodb.QueryInput{
			TableName:              aws.String(r.tableName),
			IndexName:              aws.String("OwnerIDIndex"),
			KeyConditionExpression: aws.String("OwnerID = :ownerID"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":ownerID": &types.AttributeValueMemberS{Value: userID},
			},
			Limit: aws.Int32(int32(limit)),
		}
	} else {
		// Query bots owned by the user or public bots
		queryInput = &dynamodb.QueryInput{
			TableName:              aws.String(r.tableName),
			FilterExpression:       aws.String("OwnerID = :ownerID OR IsPublic = :isPublic"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":ownerID":  &types.AttributeValueMemberS{Value: userID},
				":isPublic": &types.AttributeValueMemberBOOL{Value: true},
			},
			Limit: aws.Int32(int32(limit)),
		}
	}
	
	// Handle pagination with ExclusiveStartKey if offset is provided
	if offset > 0 {
		// This is a simplified approach to pagination
		// In a real implementation, you'd use tokens for pagination
		// For demonstration, we're just skipping records
		result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(r.tableName),
			Limit:     aws.Int32(int32(offset)),
		})
		
		if err != nil {
			return nil, fmt.Errorf("failed to scan for pagination: %w", err)
		}
		
		if result.LastEvaluatedKey != nil {
			queryInput.ExclusiveStartKey = result.LastEvaluatedKey
		}
	}
	
	// Query the bots
	result, err := r.client.Query(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to list bots: %w", err)
	}
	
	// Unmarshal the bots
	var bots []models.Bot
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &bots); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bots: %w", err)
	}
	
	return bots, nil
}

// UpdateBot updates a bot in DynamoDB
func (r *DynamoBotRepository) UpdateBot(ctx context.Context, bot models.Bot) error {
	// Update the bot's timestamp
	bot.UpdatedAt = time.Now().UTC()
	
	// Marshal the bot
	item, err := attributevalue.MarshalMap(bot)
	if err != nil {
		return fmt.Errorf("failed to marshal bot: %w", err)
	}
	
	// Update the bot
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	
	return err
}

// DeleteBot deletes a bot from DynamoDB
func (r *DynamoBotRepository) DeleteBot(ctx context.Context, botID string, userID string) error {
	// Check if the bot exists and is owned by the user
	bot, err := r.GetBot(ctx, botID, userID)
	if err != nil {
		return err
	}
	
	if bot.OwnerID != userID {
		return models.ErrNotAuthorized
	}
	
	// Delete the bot
	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"ID": &types.AttributeValueMemberS{Value: botID},
		},
	})
	
	return err
}
```

### 4. Data Models (`pkg/models/`)

#### Bot Model (`pkg/models/bot.go`)

```go
package models

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrBotNotFound          = errors.New("bot not found")
	ErrNotAuthorized        = errors.New("not authorized")
	ErrKnowledgeBaseNotFound = errors.New("knowledge base not found")
)

// Bot represents a bot entity
type Bot struct {
	ID             string    `json:"id" dynamodbav:"ID"`
	Name           string    `json:"name" dynamodbav:"Name"`
	Description    string    `json:"description" dynamodbav:"Description"`
	OwnerID        string    `json:"ownerId" dynamodbav:"OwnerID"`
	SystemPrompt   string    `json:"systemPrompt" dynamodbav:"SystemPrompt"`
	IsPublic       bool      `json:"isPublic" dynamodbav:"IsPublic"`
	CreatedAt      time.Time `json:"createdAt" dynamodbav:"CreatedAt"`
	UpdatedAt      time.Time `json:"updatedAt" dynamodbav:"UpdatedAt"`
	KnowledgeBaseID string    `json:"knowledgeBaseId,omitempty" dynamodbav:"KnowledgeBaseID,omitempty"`
	DefaultModel   string    `json:"defaultModel,omitempty" dynamodbav:"DefaultModel,omitempty"`
}

// CreateBotInput represents the input for creating a bot
type CreateBotInput struct {
	Name           string `json:"name" validate:"required,min=1,max=100"`
	Description    string `json:"description" validate:"required,min=1,max=500"`
	SystemPrompt   string `json:"systemPrompt" validate:"required,min=1,max=4000"`
	IsPublic       bool   `json:"isPublic"`
	KnowledgeBaseID string `json:"knowledgeBaseId,omitempty"`
	DefaultModel   string `json:"defaultModel,omitempty" validate:"required"`
}

// UpdateBotInput represents the input for updating a bot
type UpdateBotInput struct {
	Name           *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description    *string `json:"description,omitempty" validate:"omitempty,min=1,max=500"`
	SystemPrompt   *string `json:"systemPrompt,omitempty" validate:"omitempty,min=1,max=4000"`
	IsPublic       *bool   `json:"isPublic,omitempty"`
	KnowledgeBaseID *string `json:"knowledgeBaseId,omitempty"`
	DefaultModel   *string `json:"defaultModel,omitempty"`
}

// BotSummary represents a summary of a bot
type BotSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	OwnerID      string    `json:"ownerId"`
	IsPublic     bool      `json:"isPublic"`
	CreatedAt    time.Time `json:"createdAt"`
	DefaultModel string    `json:"defaultModel,omitempty"`
}
```

#### Chat Models (`pkg/models/chat.go`)

```go
package models

import "time"

// Message represents a chat message
type Message struct {
	ID             string    `json:"id" dynamodbav:"ID"`
	ConversationID string    `json:"conversationId" dynamodbav:"ConversationID"`
	Role           string    `json:"role" dynamodbav:"Role"` // user, assistant, system
	Content        string    `json:"content" dynamodbav:"Content"`
	CreatedAt      time.Time `json:"createdAt" dynamodbav:"CreatedAt"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID        string    `json:"id" dynamodbav:"ID"`
	UserID    string    `json:"userId" dynamodbav:"UserID"`
	BotID     string    `json:"botId" dynamodbav:"BotID"`
	Title     string    `json:"title" dynamodbav:"Title"`
	CreatedAt time.Time `json:"createdAt" dynamodbav:"CreatedAt"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"UpdatedAt"`
}

// ConversationWithMessages represents a conversation with its messages
type ConversationWithMessages struct {
	Conversation Conversation `json:"conversation"`
	Messages     []Message    `json:"messages"`
}

// ChatMessageInput represents a single message in the chat input
type ChatMessageInput struct {
	Role    string `json:"role" validate:"required,oneof=user assistant system"`
	Content string `json:"content" validate:"required"`
}

// ChatInput represents the input for generating a chat response
type ChatInput struct {
	ConversationID string             `json:"conversationId,omitempty"`
	BotID          string             `json:"botId" validate:"required"`
	Messages       []ChatMessageInput `json:"messages" validate:"required,min=1"`
	SystemPrompt   string             `json:"systemPrompt,omitempty"`
	ModelID        string             `json:"modelId,omitempty"`
	MaxTokens      int                `json:"maxTokens,omitempty"`
	Temperature    float64            `json:"temperature,omitempty"`
}

// ChatResponse represents the response from a chat request
type ChatResponse struct {
	Content        string `json:"content"`
	ConversationID string `json:"conversationId"`
	FinishReason   string `json:"finishReason,omitempty"`
}
```

## API Endpoints

### Chat API

- **POST /chat**
  - Start or continue a conversation
  - Request body: `ChatInput`
  - Response: `ChatResponse`

- **GET /conversations**
  - Get user's conversation history
  - Query parameters: `limit`, `offset`
  - Response: Array of `Conversation`

- **GET /conversations/{id}**
  - Get a specific conversation with messages
  - Path parameter: `id`
  - Response: `ConversationWithMessages`

- **DELETE /conversations/{id}**
  - Delete a conversation
  - Path parameter: `id`
  - Response: Success status

### Bot API

- **GET /bots**
  - List available bots
  - Query parameters: `limit`, `offset`, `onlyOwned`
  - Response: Array of `BotSummary`

- **GET /bots/{id}**
  - Get a specific bot
  - Path parameter: `id`
  - Response: `Bot`

- **POST /bots**
  - Create a new bot
  - Request body: `CreateBotInput`
  - Response: `Bot`

- **PUT /bots/{id}**
  - Update a bot
  - Path parameter: `id`
  - Request body: `UpdateBotInput`
  - Response: `Bot`

- **DELETE /bots/{id}**
  - Delete a bot
  - Path parameter: `id`
  - Response: Success status

### Knowledge Base API

- **GET /knowledge-bases**
  - List available knowledge bases from Bedrock
  - Query parameters: `limit`, `offset`
  - Response: Array of knowledge base summaries

## WebSocket Implementation

The WebSocket implementation enables real-time streaming of chat responses:

1. **Connection Lifecycle**:
   - `CONNECT`: Establishes a new WebSocket connection
   - `DISCONNECT`: Cleans up when a client disconnects
   - `MESSAGE`: Processes incoming messages

2. **Message Types**:
   - `START`: Starts a new chat session
   - `USER_MESSAGE`: Sends a user message
   - `STOP`: Stops generation of a response

3. **Response Streaming**:
   - Responses from Bedrock are streamed in real-time
   - Each token is sent as a separate WebSocket message
   - Special messages indicate start/end of responses

## Security Considerations

1. **Authentication**:
   - All API endpoints require authentication
   - API Gateway integrates with Cognito for JWT validation
   - User identity is extracted from JWT claims

2. **Authorization**:
   - Bot access is restricted to owners and public bots
   - Knowledge base operations require appropriate permissions
   - Conversation access is restricted to owners

3. **Input Validation**:
   - All input is validated using struct tags
   - Request validation middleware ensures data integrity
   - Error messages are sanitized to prevent information disclosure

## Error Handling

The system implements a comprehensive error handling strategy:

1. **Domain Errors**: Defined in model packages for specific business rules
2. **HTTP Status Codes**: Mapped appropriately to domain errors
3. **Logging**: Structured logging for debugging and monitoring
4. **Client-Friendly Messages**: Sanitized error responses for clients