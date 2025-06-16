package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/bobbyrathore/go-lambda-backend/internal/handlers"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	botHandler           *handlers.BotHandler
	documentHandler      *handlers.DocumentHandler
	knowledgeBaseHandler *handlers.KnowledgeBaseHandler
	chatHandler          *handlers.ChatHandler
	cfg                  *appConfig.Config
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Validate critical environment variables (use what Lambda actually provides)
	requiredEnvVars := []string{"AWS_REGION", "BOTS_TABLE"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			log.Printf("⚠️ WARNING: Required environment variable %s is not set", envVar)
		}
	}

	// Initialize AWS SDK (Lambda provides AWS_REGION by default)
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1" // Fallback (should never happen in Lambda)
		log.Printf("⚠️ AWS_REGION not set, defaulting to: %s", region)
	}
	
	log.Printf("🌍 AWS Region: %s", region)
	log.Printf("🔧 Lambda deployment v2 - VPC removed")
	
	// Log VPC-related environment variables for debugging
	log.Printf("🔍 Lambda Environment Context:")
	log.Printf("   AWS_LAMBDA_FUNCTION_NAME: %s", os.Getenv("AWS_LAMBDA_FUNCTION_NAME"))
	log.Printf("   AWS_LAMBDA_RUNTIME_API: %s", os.Getenv("AWS_LAMBDA_RUNTIME_API"))
	if vpcConfig := os.Getenv("AWS_LAMBDA_INITIALIZATION_TYPE"); vpcConfig != "" {
		log.Printf("   AWS_LAMBDA_INITIALIZATION_TYPE: %s", vpcConfig)
	}
	
	// Check for any VPC-related environment hints
	for _, env := range os.Environ() {
		if strings.Contains(strings.ToLower(env), "vpc") || strings.Contains(strings.ToLower(env), "subnet") {
			log.Printf("   VPC-related env: %s", env)
		}
	}
	
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithClientLogMode(aws.LogRetries | aws.LogRequest | aws.LogResponse),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	
	log.Printf("📊 AWS Config Region: %s", awsConfig.Region)
	log.Printf("🔧 AWS Config EndpointResolver: %T", awsConfig.EndpointResolverWithOptions)
	
	log.Printf("✅ AWS SDK v2 config loaded successfully (only for S3 and other services)")

	// Create AWS service clients (S3 still uses v2, only DynamoDB uses v1)
	s3Client := s3.NewFromConfig(awsConfig)

	// Initialize repository using AWS SDK v1 only
	botRepo, err := repositories.NewBotRepositoryV1(os.Getenv("BOTS_TABLE"))
	if err != nil {
		log.Fatalf("Failed to initialize bot repository: %v", err)
	}

	// No longer need infrastructure path - using CloudFormation SDK directly
	
	// ENV_PREFIX - try ENV_PREFIX first, then STAGE (from SAM template)
	envPrefix := os.Getenv("ENV_PREFIX")
	if envPrefix == "" {
		envPrefix = os.Getenv("STAGE") // SAM template uses STAGE
		if envPrefix == "" {
			envPrefix = "dev" // Default fallback
		}
	}
	
	// Initialize CloudFormation service (replaces CDK service)
	cfnService := services.NewCloudFormationService(awsConfig, region)
	
	// Initialize stack output service
	stackOutputService := services.NewStackOutputService(awsConfig, region)

	// Initialize conversation repository using AWS SDK v1 only
	conversationRepo, err := repositories.NewConversationRepositoryV1(
		os.Getenv("CONVERSATIONS_TABLE"), 
		os.Getenv("MESSAGES_TABLE"), 
		os.Getenv("CONVERSATIONS_S3_BUCKET"),
	)
	if err != nil {
		log.Fatalf("Failed to initialize conversation repository: %v", err)
	}

	// Initialize services
	botService := services.NewBotService(botRepo, cfnService, stackOutputService, envPrefix, region)
	documentService := services.NewDocumentService(s3Client, region)
	knowledgeBaseService := services.NewKnowledgeBaseService(awsConfig, region)
	chatService := services.NewChatService(awsConfig, botService, knowledgeBaseService, conversationRepo, region)

	// Initialize handlers
	botHandler = handlers.NewBotHandler(botService)
	knowledgeBaseHandler = handlers.NewKnowledgeBaseHandler(knowledgeBaseService)
	chatHandler = handlers.NewChatHandler(chatService, conversationRepo, botService)
	
	// DOCUMENTS_BUCKET - try multiple sources
	documentsBucket := os.Getenv("DOCUMENTS_BUCKET")
	if documentsBucket == "" {
		documentsBucket = os.Getenv("S3_BUCKET_NAME") // Alternative from config
		if documentsBucket == "" {
			// Generate a reasonable default bucket name
			documentsBucket = fmt.Sprintf("%s-documents-%s", envPrefix, region)
			log.Printf("⚠️ DOCUMENTS_BUCKET not set, using default: %s", documentsBucket)
		}
	}
	documentHandler = handlers.NewDocumentHandler(documentService, botService, documentsBucket)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// DEBUG: Log everything about the request
	log.Printf("=== LAMBDA REQUEST DEBUG === (VPC Endpoints v2)")
	log.Printf("HTTP Method: %s", request.HTTPMethod)
	log.Printf("Path: %s", request.Path)
	log.Printf("Headers: %+v", request.Headers)
	log.Printf("RequestContext.RequestId: %s", request.RequestContext.RequestID)
	log.Printf("RequestContext.Authorizer: %+v", request.RequestContext.Authorizer)
	
	// Route Knowledge Base endpoints to knowledge base handler
	// Check for knowledge-base-related paths: /knowledge-bases/*
	if strings.HasPrefix(request.Path, "/knowledge-bases") {
		return handleKnowledgeBaseRoutes(ctx, request)
	}
	
	// Route chat endpoints to chat handler
	// Check for chat-related paths: /bots/{id}/chat or /bots/{id}/conversations
	if strings.Contains(request.Path, "/chat") || strings.Contains(request.Path, "/conversations") {
		return chatHandler.HandleChatRequest(ctx, request)
	}
	
	// Route document endpoints to document handler
	// Check for document-related paths: /bots/{id}/documents/*
	if strings.Contains(request.Path, "/documents") {
		return documentHandler.HandleDocumentRequest(ctx, request)
	}
	
	
	// Use the bot handler's routing logic for other requests
	return botHandler.HandleBotRequest(ctx, request)
}

// handleKnowledgeBaseRoutes routes Knowledge Base related requests
func handleKnowledgeBaseRoutes(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("📚 Routing Knowledge Base request: %s %s", request.HTTPMethod, request.Path)
	
	switch {
	case request.HTTPMethod == "GET" && request.Path == "/knowledge-bases":
		// GET /knowledge-bases - List all Knowledge Bases
		return knowledgeBaseHandler.HandleListKnowledgeBases(ctx, request)
		
	case request.HTTPMethod == "GET" && strings.HasPrefix(request.Path, "/knowledge-bases/") && !strings.Contains(request.Path, "/validate"):
		// GET /knowledge-bases/{id} - Get specific Knowledge Base
		return knowledgeBaseHandler.HandleGetKnowledgeBase(ctx, request)
		
	case request.HTTPMethod == "POST" && request.Path == "/knowledge-bases/validate":
		// POST /knowledge-bases/validate - Validate Knowledge Base access
		return knowledgeBaseHandler.HandleKnowledgeBaseValidation(ctx, request)
		
	case request.HTTPMethod == "OPTIONS":
		// Handle CORS preflight
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			},
		}, nil
		
	default:
		// Unknown Knowledge Base endpoint
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Headers: map[string]string{
				"Content-Type":                "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: fmt.Sprintf(`{"error": "Knowledge Base endpoint not found: %s %s"}`, request.HTTPMethod, request.Path),
		}, nil
	}
}

func main() {
	lambda.Start(Handler)
}