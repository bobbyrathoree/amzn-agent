package utils

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// APIResponse standardizes API responses
type APIResponse struct {
	StatusCode int         `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       interface{} `json:"body"`
}

// NewAPIResponse creates a new APIResponse
func NewAPIResponse(statusCode int, body interface{}) events.APIGatewayProxyResponse {
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*", // For CORS support
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,x-user-id",
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"error":"Failed to marshal response"}`,
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       string(bodyBytes),
	}
}

// ErrorResponse standardizes error responses
func ErrorResponse(err error, statusCode int) events.APIGatewayProxyResponse {
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	errorBody := map[string]string{
		"error": err.Error(),
	}

	return NewAPIResponse(statusCode, errorBody)
}

// SuccessResponse returns a successful API response
func SuccessResponse(data interface{}) events.APIGatewayProxyResponse {
	return NewAPIResponse(http.StatusOK, data)
}

// CreatedResponse returns a 201 Created response
func CreatedResponse(data interface{}) events.APIGatewayProxyResponse {
	return NewAPIResponse(http.StatusCreated, data)
}

// NoContentResponse returns a 204 No Content response
func NoContentResponse() events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusNoContent,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
			"Access-Control-Allow-Headers": "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token,x-user-id",
		},
	}
}

// ExtractUserIDFromRequest extracts user ID from API Gateway request context (Cognito JWT)
func ExtractUserIDFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	userID := ""
	
	// Extract user ID from the authorizer context (Cognito User Pool)
	if request.RequestContext.Authorizer != nil {
		// Try direct sub claim
		if id, ok := request.RequestContext.Authorizer["sub"].(string); ok {
			userID = id
		}
		// Try alternative userId format
		if id, ok := request.RequestContext.Authorizer["userId"].(string); ok && userID == "" {
			userID = id
		}
		
		// Check for claims in nested format
		if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
			if sub, exists := claims["sub"].(string); exists && userID == "" {
				userID = sub
			}
		}
	}
	
	// Fallback for development/testing
	if userID == "" {
		if id := request.Headers["X-User-ID"]; id != "" {
			userID = id
		}
		if id := request.Headers["x-user-id"]; id != "" && userID == "" {
			userID = id
		}
	}
	
	if userID == "" {
		return "", errors.New("user ID not found in request context or headers")
	}
	
	return userID, nil
}

// ErrorFromString creates an error from a string message
func ErrorFromString(message string) error {
	return errors.New(message)
}