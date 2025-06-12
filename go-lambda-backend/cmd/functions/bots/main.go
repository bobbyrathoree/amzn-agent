package main

import (
	"context"
	"encoding/json"
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
	botHandler *handlers.BotHandler
	cfg        *appConfig.Config
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

	// Initialize repository
	botRepo := repositories.NewBotRepository(dynamoClient, os.Getenv("BOTS_TABLE"))

	// Initialize CDK deployment service
	cdkService := services.NewCDKDeploymentService("cdk", "../../../infrastructure", "dev", os.Getenv("REGION"))
	
	// Initialize stack output service
	stackOutputService := services.NewStackOutputService(awsConfig, os.Getenv("REGION"))

	// Initialize service
	botService := services.NewBotService(botRepo, cdkService, stackOutputService, "dev", os.Getenv("REGION"))

	// Initialize handler
	botHandler = handlers.NewBotHandler(botService)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// DEBUG: Log everything about the request
	log.Printf("=== LAMBDA REQUEST DEBUG ===")
	log.Printf("HTTP Method: %s", request.HTTPMethod)
	log.Printf("Path: %s", request.Path)
	log.Printf("Headers: %+v", request.Headers)
	log.Printf("RequestContext.RequestId: %s", request.RequestContext.RequestID)
	log.Printf("RequestContext.Authorizer: %+v", request.RequestContext.Authorizer)
	
	// If this is a GET to /bots, return debug info instead of calling handler
	if request.HTTPMethod == "GET" && request.Path == "/bots" {
		response := map[string]interface{}{
			"message": "Lambda reached successfully!",
			"method": request.HTTPMethod,
			"path": request.Path,
			"authorizer": request.RequestContext.Authorizer,
			"headers": request.Headers,
		}
		
		responseBody, _ := json.Marshal(response)
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Access-Control-Allow-Origin": "*",
			},
			Body: string(responseBody),
		}, nil
	}
	
	// Use the bot handler's routing logic for other requests
	return botHandler.HandleBotRequest(ctx, request)
}

func main() {
	lambda.Start(Handler)
}