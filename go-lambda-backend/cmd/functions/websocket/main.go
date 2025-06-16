package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"

	appConfig "github.com/bobbyrathore/go-lambda-backend/pkg/config"
)

var (
	cfg *appConfig.Config
)

func init() {
	// Load configuration
	cfg = appConfig.NewConfig()

	// Initialize AWS SDK (Lambda provides AWS_REGION by default)
	_, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
}

// Handler is the Lambda function handler for WebSocket events
func Handler(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Handle WebSocket events
	switch request.RequestContext.EventType {
	case "CONNECT":
		return handleConnect(ctx, request)
	case "DISCONNECT":
		return handleDisconnect(ctx, request)
	case "MESSAGE":
		return handleMessage(ctx, request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"message": "Unsupported event type"}`,
		}, nil
	}
}

func handleConnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket connection established: %s", request.RequestContext.ConnectionID)
	
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"message": "Connected"}`,
	}, nil
}

func handleDisconnect(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket connection closed: %s", request.RequestContext.ConnectionID)
	
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       `{"message": "Disconnected"}`,
	}, nil
}

func handleMessage(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("WebSocket message received: %s", request.Body)
	
	// Parse the message
	var message map[string]interface{}
	if err := json.Unmarshal([]byte(request.Body), &message); err != nil {
		log.Printf("Failed to parse message: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       `{"error": "Invalid message format"}`,
		}, nil
	}

	// For demo purposes, just echo back the message
	response := map[string]interface{}{
		"type":    "echo",
		"message": message,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       `{"error": "Failed to create response"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func main() {
	lambda.Start(Handler)
}