package main

import (
	"context"
	"log"

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
	// Initialize these at package level for reuse across invocations
	itemHandler *handlers.ItemHandler
	cfg         *appConfig.Config
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Initialize AWS SDK
	awsConfig, err := config.LoadDefaultConfig(context.Background(), 
		config.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	// Initialize repository
	itemRepo := repositories.NewDynamoDBItemRepository(dynamoClient, cfg.DynamoDBTable)

	// Initialize service
	itemService := services.NewItemService(itemRepo)

	// Initialize handler
	itemHandler = handlers.NewItemHandler(itemService)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Route based on the HTTP method and path
	switch request.HTTPMethod {
	case "GET":
		if id, ok := request.PathParameters["id"]; ok && id != "" {
			return itemHandler.GetItem(ctx, request)
		}
		return itemHandler.ListItems(ctx, request)
	case "POST":
		return itemHandler.CreateItem(ctx, request)
	case "PUT", "PATCH":
		return itemHandler.UpdateItem(ctx, request)
	case "DELETE":
		return itemHandler.DeleteItem(ctx, request)
	default:
		// Return 405 Method Not Allowed for unsupported methods
		return events.APIGatewayProxyResponse{
			StatusCode: 405,
			Body:       `{"error":"Method not allowed"}`,
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}, nil
	}
}

func main() {
	lambda.Start(Handler)
}