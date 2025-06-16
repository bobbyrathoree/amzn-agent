package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/bobbyrathore/go-lambda-backend/internal/handlers"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	chatHandler *handlers.ChatHandler
	cfg         *appConfig.Config
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Initialize AWS SDK (Lambda provides AWS_REGION by default)
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Note: S3 client creation moved to repositories that need it

	// Initialize repositories using AWS SDK v1 only
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
	// For chat-only lambda, we'll create a simplified bot service with CloudFormation dependencies
	cfnSvc := services.NewCloudFormationService(awsConfig, os.Getenv("AWS_REGION"))
	stackOutputSvc := services.NewStackOutputService(awsConfig, os.Getenv("AWS_REGION"))
	botSvc := services.NewBotService(botRepo, cfnSvc, stackOutputSvc, os.Getenv("AWS_REGION"), "dev")
	kbSvc := services.NewKnowledgeBaseService(awsConfig, os.Getenv("AWS_REGION"))
	chatSvc := services.NewChatService(awsConfig, botSvc, kbSvc, conversationRepo, os.Getenv("AWS_REGION"))

	// Initialize handler
	chatHandler = handlers.NewChatHandler(chatSvc, conversationRepo, botSvc)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Use the chat handler's routing (user auth context is handled within the handler)
	return chatHandler.HandleChatRequest(ctx, request)
}

func main() {
	lambda.Start(Handler)
}