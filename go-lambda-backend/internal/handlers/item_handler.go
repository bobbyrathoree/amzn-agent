package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bobbyrathore/go-lambda-backend/internal/middleware"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/utils"
)

// ItemHandler handles API Gateway events for item operations
type ItemHandler struct {
	service   *services.ItemService
	validator *middleware.Validator
}

// NewItemHandler creates a new ItemHandler
func NewItemHandler(service *services.ItemService) *ItemHandler {
	return &ItemHandler{
		service:   service,
		validator: middleware.NewValidator(),
	}
}

// GetItem handles requests to get a single item
func (h *ItemHandler) GetItem(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add user context from the request
	ctx = middleware.WithUserContext(ctx, request)

	// Extract the item ID from the path parameters
	id := request.PathParameters["id"]
	if id == "" {
		return utils.ErrorResponse(
			errors.New("missing item ID"),
			http.StatusBadRequest,
		), nil
	}

	// Call the service to get the item
	item, err := h.service.GetItem(ctx, id)
	if err != nil {
		if err.Error() == "item not found" {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return the successful response
	return utils.SuccessResponse(models.ItemResponse{
		Item:      item,
		RequestID: request.RequestContext.RequestID,
	}), nil
}

// ListItems handles requests to list items
func (h *ItemHandler) ListItems(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add user context
	ctx = middleware.WithUserContext(ctx, request)

	// Extract query parameters
	category := request.QueryStringParameters["category"]
	limitStr := request.QueryStringParameters["limit"]
	nextToken := request.QueryStringParameters["nextToken"]

	// Parse limit parameter
	limit := 20 // Default limit
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil {
			limit = parsedLimit
		}
	}

	// Call the service to list items
	response, err := h.service.ListItems(ctx, category, limit, nextToken)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Add request ID to the response
	response.RequestID = request.RequestContext.RequestID

	// Return the successful response
	return utils.SuccessResponse(response), nil
}

// CreateItem handles requests to create a new item
func (h *ItemHandler) CreateItem(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add user context
	ctx = middleware.WithUserContext(ctx, request)

	// Parse and validate the request
	var req models.CreateItemRequest
	if err := h.validator.ValidateRequest(request, &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Call the service to create the item
	item, err := h.service.CreateItem(ctx, req)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return the successful response
	return utils.CreatedResponse(models.ItemResponse{
		Item:      item,
		Message:   "Item created successfully",
		RequestID: request.RequestContext.RequestID,
	}), nil
}

// UpdateItem handles requests to update an existing item
func (h *ItemHandler) UpdateItem(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add user context
	ctx = middleware.WithUserContext(ctx, request)

	// Extract the item ID from the path parameters
	id := request.PathParameters["id"]
	if id == "" {
		return utils.ErrorResponse(
			errors.New("missing item ID"),
			http.StatusBadRequest,
		), nil
	}

	// Parse and validate the request
	var req models.UpdateItemRequest
	if err := h.validator.ValidateRequest(request, &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Call the service to update the item
	item, err := h.service.UpdateItem(ctx, id, req)
	if err != nil {
		if err.Error() == "item not found" {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return the successful response
	return utils.SuccessResponse(models.ItemResponse{
		Item:      item,
		Message:   "Item updated successfully",
		RequestID: request.RequestContext.RequestID,
	}), nil
}

// DeleteItem handles requests to delete an item
func (h *ItemHandler) DeleteItem(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Add user context
	ctx = middleware.WithUserContext(ctx, request)

	// Extract the item ID from the path parameters
	id := request.PathParameters["id"]
	if id == "" {
		return utils.ErrorResponse(
			errors.New("missing item ID"),
			http.StatusBadRequest,
		), nil
	}

	// Call the service to delete the item
	err := h.service.DeleteItem(ctx, id)
	if err != nil {
		if err.Error() == "item not found" {
			return utils.ErrorResponse(err, http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Return a success response with no content
	return utils.NoContentResponse(), nil
}