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
	chatHandler *handlers.ChatHandler
	cfg         *appConfig.Config
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Initialize AWS SDK
	awsConfig, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(os.Getenv("REGION")),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	// Initialize repositories
	chatRepo := repositories.NewChatRepository(
		dynamoClient,
		os.Getenv("CONVERSATIONS_TABLE"),
		os.Getenv("MESSAGES_TABLE"),
	)
	botRepo := repositories.NewBotRepository(dynamoClient, os.Getenv("BOTS_TABLE"))

	// Initialize services
	bedrockSvc := services.NewBedrockService(awsConfig)
	chatSvc := services.NewChatService(chatRepo, botRepo, bedrockSvc)

	// Initialize handler
	chatHandler = handlers.NewChatHandler(chatSvc)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract user ID from the authorizer context
	userID, ok := request.RequestContext.Authorizer["sub"].(string)
	if !ok {
		userID, ok = request.RequestContext.Authorizer["userId"].(string)
		if !ok {
			return events.APIGatewayProxyResponse{
				StatusCode: 401,
				Body:       `{"error":"Unauthorized - no user ID found"}`,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			}, nil
		}
	}

	// Route based on the HTTP method and path
	switch {
	case request.HTTPMethod == "POST" && request.Resource == "/chat":
		return chatHandler.HandleChatMessage(ctx, request, userID)
	case request.HTTPMethod == "GET" && request.Resource == "/conversations":
		return chatHandler.HandleListConversations(ctx, request, userID)
	case request.HTTPMethod == "GET" && request.Resource == "/conversations/{id}":
		return chatHandler.HandleGetConversation(ctx, request, userID)
	case request.HTTPMethod == "DELETE" && request.Resource == "/conversations/{id}":
		return chatHandler.HandleDeleteConversation(ctx, request, userID)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       `{"error":"Not found"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}
}

func main() {
	lambda.Start(Handler)
}