package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/bobbyrathore/go-lambda-backend/internal/handlers"
	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	knowledgeHandler *handlers.KnowledgeHandler
	cfg              *appConfig.Config
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

	// Initialize handler
	knowledgeHandler = handlers.NewKnowledgeHandler(awsConfig)
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
	switch request.HTTPMethod {
	case "GET":
		return knowledgeHandler.ListKnowledgeBases(ctx, request, userID)
	default:
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