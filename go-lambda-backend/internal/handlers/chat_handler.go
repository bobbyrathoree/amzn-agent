package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bobbyrathore/go-lambda-backend/internal/middleware"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/utils"
)

// ChatHandler handles API Gateway events for chat operations
type ChatHandler struct {
	service   *services.ChatService
	validator *middleware.Validator
}

// NewChatHandler creates a new ChatHandler
func NewChatHandler(service *services.ChatService) *ChatHandler {
	return &ChatHandler{
		service:   service,
		validator: middleware.NewValidator(),
	}
}

// HandleChatMessage handles requests to send a chat message
func (h *ChatHandler) HandleChatMessage(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// Parse and validate the request
	var req models.ChatInput
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(
			errors.New("invalid JSON in request body"),
			http.StatusBadRequest,
		), nil
	}

	if err := h.validator.Validate(&req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Call the service to create the chat message
	response, err := h.service.CreateChatMessage(ctx, req, userID)
	if err != nil {
		if err == models.ErrNotAuthorized {
			return utils.ErrorResponse(err, http.StatusForbidden), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return the successful response
	return utils.SuccessResponse(response), nil
}

// HandleListConversations handles requests to list conversations
func (h *ChatHandler) HandleListConversations(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// Extract query parameters
	limitStr := request.QueryStringParameters["limit"]
	nextToken := request.QueryStringParameters["nextToken"]

	// Parse limit parameter
	limit := 20 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Call the service to list conversations
	response, err := h.service.ListConversations(ctx, userID, limit, nextToken)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Add request ID to the response
	response.RequestID = request.RequestContext.RequestID

	// Return the successful response
	return utils.SuccessResponse(response), nil
}

// HandleGetConversation handles requests to get a specific conversation
func (h *ChatHandler) HandleGetConversation(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// Extract the conversation ID from the path parameters
	id := request.PathParameters["id"]
	if id == "" {
		return utils.ErrorResponse(
			errors.New("missing conversation ID"),
			http.StatusBadRequest,
		), nil
	}

	// Call the service to get the conversation
	conversation, err := h.service.GetConversation(ctx, id, userID)
	if err != nil {
		if err == models.ErrNotAuthorized {
			return utils.ErrorResponse(err, http.StatusForbidden), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return the successful response
	return utils.SuccessResponse(conversation), nil
}

// HandleDeleteConversation handles requests to delete a conversation
func (h *ChatHandler) HandleDeleteConversation(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// Extract the conversation ID from the path parameters
	id := request.PathParameters["id"]
	if id == "" {
		return utils.ErrorResponse(
			errors.New("missing conversation ID"),
			http.StatusBadRequest,
		), nil
	}

	// Call the service to delete the conversation
	err := h.service.DeleteConversation(ctx, id, userID)
	if err != nil {
		if err == models.ErrNotAuthorized {
			return utils.ErrorResponse(err, http.StatusForbidden), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return a success response with no content
	return utils.NoContentResponse(), nil
}