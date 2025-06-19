package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/bobbyrathore/go-lambda-backend/internal/handlers"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	toolHandler *handlers.ToolHandler
	cfg         *appConfig.Config
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Initialize AWS SDK
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Initialize DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	// Initialize repositories
	botRepo, err := repositories.NewBotRepositoryV1(os.Getenv("BOTS_TABLE"))
	if err != nil {
		log.Fatalf("Failed to initialize bot repository: %v", err)
	}
	
	conversationRepo, err := repositories.NewConversationRepositoryV1(
		os.Getenv("CONVERSATIONS_TABLE"), 
		os.Getenv("MESSAGES_TABLE"), 
		os.Getenv("CONVERSATIONS_S3_BUCKET"),
	)
	if err != nil {
		log.Fatalf("Failed to initialize conversation repository: %v", err)
	}

	// Initialize services
	cfnSvc := services.NewCloudFormationService(awsConfig, os.Getenv("AWS_REGION"))
	stackOutputSvc := services.NewStackOutputService(awsConfig, os.Getenv("AWS_REGION"))
	botSvc := services.NewBotService(botRepo, cfnSvc, stackOutputSvc, os.Getenv("AWS_REGION"), "dev")
	kbSvc := services.NewKnowledgeBaseService(awsConfig, os.Getenv("AWS_REGION"))
	chatSvc := services.NewChatService(awsConfig, botSvc, kbSvc, conversationRepo, os.Getenv("AWS_REGION"))
	
	// Initialize tool-specific services
	vaultService := services.NewVaultService(dynamoClient, os.Getenv("VAULT_TABLE"))
	toolRegistryService := services.NewToolRegistryService()
	toolExecutionService := services.NewToolExecutionService(
		toolRegistryService,
		vaultService,
		chatSvc,
		conversationRepo,
	)

	// Initialize handler
	toolHandler = handlers.NewToolHandler(toolExecutionService, toolRegistryService)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("🔧 Tools Lambda handling request: %s %s", request.HTTPMethod, request.Path)
	return toolHandler.HandleToolRequest(ctx, request)
}

func main() {
	lambda.Start(Handler)
}