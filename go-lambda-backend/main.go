package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response is a generic API response
type Response struct {
	Message string `json:"message"`
}

// Handler is the Lambda handler
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log the request
	log.Printf("Request: %+v", request)

	// Get the Lambda function name from environment variable
	functionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	// Create response based on the function
	var message string
	switch functionName {
	case "chat":
		message = "Chat Lambda function invoked successfully"
	case "bots":
		message = "Bots Lambda function invoked successfully"
	case "knowledge":
		message = "Knowledge Lambda function invoked successfully"
	case "websocket":
		message = "WebSocket Lambda function invoked successfully"
	default:
		message = "Lambda function invoked successfully"
	}

	// Create response
	response := Response{
		Message: message,
	}

	// Convert response to JSON
	body, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "%s"}`, err.Error()),
		}, nil
	}

	// Return response
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(body),
	}, nil
}

func main() {
	lambda.Start(Handler)
}