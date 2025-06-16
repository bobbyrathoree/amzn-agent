package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
)

// KnowledgeBaseHandler handles Knowledge Base related requests
type KnowledgeBaseHandler struct {
	kbService *services.KnowledgeBaseService
}

// NewKnowledgeBaseHandler creates a new KnowledgeBaseHandler
func NewKnowledgeBaseHandler(kbService *services.KnowledgeBaseService) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		kbService: kbService,
	}
}

// ListKnowledgeBasesResponse represents the response for listing Knowledge Bases
type ListKnowledgeBasesResponse struct {
	KnowledgeBases []services.KnowledgeBaseInfo `json:"knowledgeBases"`
	Count          int                          `json:"count"`
	Message        string                       `json:"message,omitempty"`
}

// HandleListKnowledgeBases handles GET /knowledge-bases requests
func (h *KnowledgeBaseHandler) HandleListKnowledgeBases(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("📚 Processing Knowledge Bases list request")

	// Parse query parameters
	maxResults := int32(50) // Default value
	if maxResultsStr := request.QueryStringParameters["maxResults"]; maxResultsStr != "" {
		if parsed, err := strconv.ParseInt(maxResultsStr, 10, 32); err == nil && parsed > 0 && parsed <= 100 {
			maxResults = int32(parsed)
		}
	}

	status := request.QueryStringParameters["status"]
	activeOnly := request.QueryStringParameters["activeOnly"] == "true"

	// List Knowledge Bases based on filters
	var knowledgeBases []services.KnowledgeBaseInfo
	var err error

	if activeOnly {
		knowledgeBases, err = h.kbService.GetActiveKnowledgeBases(ctx, maxResults)
	} else if status != "" {
		// Validate status parameter
		switch status {
		case "CREATING", "ACTIVE", "DELETING", "UPDATING", "FAILED":
			// These are valid Bedrock Knowledge Base statuses
			knowledgeBases, err = h.kbService.GetKnowledgeBasesByStatus(ctx, types.KnowledgeBaseStatus(status), maxResults)
		default:
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusBadRequest,
				Headers:    corsHeaders(),
				Body:       `{"error": "Invalid status parameter. Must be one of: CREATING, ACTIVE, DELETING, UPDATING, FAILED"}`,
			}, nil
		}
	} else {
		knowledgeBases, err = h.kbService.ListKnowledgeBases(ctx, maxResults)
	}

	if err != nil {
		log.Printf("❌ Failed to list Knowledge Bases: %v", err)
		
		// Check if it's a permissions/access error
		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"error": "Access denied. Please ensure your AWS credentials have permission to list Bedrock Knowledge Bases"}`,
			}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to list Knowledge Bases: %s"}`, err.Error()),
		}, nil
	}

	// Prepare response
	response := ListKnowledgeBasesResponse{
		KnowledgeBases: knowledgeBases,
		Count:          len(knowledgeBases),
	}

	// Add informative message
	if len(knowledgeBases) == 0 {
		response.Message = "No Knowledge Bases found. You may need to create one in the AWS Bedrock console first."
	} else {
		response.Message = fmt.Sprintf("Found %d Knowledge Base(s)", len(knowledgeBases))
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("❌ Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Successfully listed %d Knowledge Bases", len(knowledgeBases))

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// GetKnowledgeBaseResponse represents the response for getting a specific Knowledge Base
type GetKnowledgeBaseResponse struct {
	KnowledgeBase *services.KnowledgeBaseInfo `json:"knowledgeBase"`
	Message       string                      `json:"message,omitempty"`
}

// HandleGetKnowledgeBase handles GET /knowledge-bases/{id} requests
func (h *KnowledgeBaseHandler) HandleGetKnowledgeBase(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract Knowledge Base ID from path parameters
	kbID := request.PathParameters["id"]
	if kbID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Knowledge Base ID is required"}`,
		}, nil
	}

	log.Printf("📚 Processing Knowledge Base details request for: %s", kbID)

	// Get Knowledge Base details
	kb, err := h.kbService.GetKnowledgeBase(ctx, kbID)
	if err != nil {
		log.Printf("❌ Failed to get Knowledge Base %s: %v", kbID, err)

		// Handle different error types
		if isNotFoundError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Headers:    corsHeaders(),
				Body:       fmt.Sprintf(`{"error": "Knowledge Base '%s' not found"}`, kbID),
			}, nil
		}

		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"error": "Access denied. You don't have permission to access this Knowledge Base"}`,
			}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to get Knowledge Base: %s"}`, err.Error()),
		}, nil
	}

	// Prepare response
	response := GetKnowledgeBaseResponse{
		KnowledgeBase: kb,
		Message:       fmt.Sprintf("Knowledge Base '%s' retrieved successfully", kb.Name),
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("❌ Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Successfully retrieved Knowledge Base: %s", kb.Name)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleKnowledgeBaseValidation handles POST /knowledge-bases/validate requests
func (h *KnowledgeBaseHandler) HandleKnowledgeBaseValidation(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse request body
	var validationRequest struct {
		KnowledgeBaseID string `json:"knowledgeBaseId"`
	}

	if err := json.Unmarshal([]byte(request.Body), &validationRequest); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Invalid request body"}`,
		}, nil
	}

	if validationRequest.KnowledgeBaseID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Knowledge Base ID is required"}`,
		}, nil
	}

	log.Printf("📚 Validating Knowledge Base access: %s", validationRequest.KnowledgeBaseID)

	// Validate access
	err := h.kbService.ValidateKnowledgeBaseAccess(ctx, validationRequest.KnowledgeBaseID)
	if err != nil {
		log.Printf("❌ Knowledge Base validation failed: %v", err)

		if isNotFoundError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Headers:    corsHeaders(),
				Body:       fmt.Sprintf(`{"valid": false, "error": "Knowledge Base '%s' not found"}`, validationRequest.KnowledgeBaseID),
			}, nil
		}

		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"valid": false, "error": "Access denied to Knowledge Base"}`,
			}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"valid": false, "error": "Knowledge Base is not accessible: %s"}`, err.Error()),
		}, nil
	}

	log.Printf("✅ Knowledge Base validation successful: %s", validationRequest.KnowledgeBaseID)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       fmt.Sprintf(`{"valid": true, "message": "Knowledge Base '%s' is accessible"}`, validationRequest.KnowledgeBaseID),
	}, nil
}

// Helper functions for error handling
func isAccessDeniedError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "AccessDenied") ||
		contains(errStr, "UnauthorizedOperation") ||
		contains(errStr, "Forbidden") ||
		contains(errStr, "access denied")
}

func isNotFoundError(err error) bool {
	errStr := err.Error()
	return contains(errStr, "ResourceNotFound") ||
		contains(errStr, "NotFound") ||
		contains(errStr, "does not exist")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)+1 && s[len(s)-len(substr)-1:len(s)-1] == substr)))
}

// corsHeaders returns CORS headers for API responses
func corsHeaders() map[string]string {
	return map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}
}