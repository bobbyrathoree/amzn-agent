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
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	vaultHandler *handlers.VaultHandler
	cfg          *appConfig.Config
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

	// Initialize vault service
	vaultService := services.NewVaultService(dynamoClient, os.Getenv("VAULT_TABLE"))

	// Initialize handler
	vaultHandler = handlers.NewVaultHandler(vaultService)
}

// Handler is the Lambda function handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("🔐 Vault Lambda handling request: %s %s", request.HTTPMethod, request.Path)
	return vaultHandler.HandleVaultRequest(ctx, request)
}

func main() {
	lambda.Start(Handler)
}