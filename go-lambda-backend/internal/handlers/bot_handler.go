package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/oklog/ulid/v2"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/utils"
)

// BotHandler handles HTTP requests for bot operations
type BotHandler struct {
	botService *services.BotService
	validator  *validator.Validate
}

// NewBotHandler creates a new BotHandler
func NewBotHandler(botService *services.BotService) *BotHandler {
	return &BotHandler{
		botService: botService,
		validator:  validator.New(),
	}
}

// HandleBotRequest routes bot-related HTTP requests
func (h *BotHandler) HandleBotRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	method := request.HTTPMethod
	pathSegments := strings.Split(strings.Trim(request.Path, "/"), "/")
	
	// Extract user ID from headers or JWT token (simplified for demo)
	userID := h.extractUserID(request)
	if userID == "" {
		return utils.ErrorResponse(http.StatusUnauthorized, "Unauthorized", ""), nil
	}

	switch method {
	case "GET":
		if len(pathSegments) == 1 { // GET /bots
			return h.listBots(ctx, request, userID)
		} else if len(pathSegments) == 2 { // GET /bots/{id}
			return h.getBot(ctx, pathSegments[1], userID)
		}
	case "POST":
		if len(pathSegments) == 1 { // POST /bots
			return h.createBot(ctx, request, userID)
		}
	case "PUT", "PATCH":
		if len(pathSegments) == 2 { // PUT/PATCH /bots/{id}
			return h.updateBot(ctx, request, pathSegments[1], userID)
		}
	case "DELETE":
		if len(pathSegments) == 2 { // DELETE /bots/{id}
			return h.deleteBot(ctx, pathSegments[1], userID)
		}
	}

	return utils.ErrorResponse(http.StatusNotFound, "Endpoint not found", ""), nil
}

// listBots handles GET /bots
func (h *BotHandler) listBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters
	includePublic := request.QueryStringParameters["includePublic"] == "true"
	starred := request.QueryStringParameters["starred"] == "true"
	
	bots, err := h.botService.ListBots(ctx, userID, includePublic, starred)
	if err != nil {
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to list bots", err.Error()), nil
	}

	response := models.BotsResponse{
		Bots:      bots,
		Count:     len(bots),
		Total:     len(bots),
		RequestID: h.generateRequestID(),
	}

	return utils.JSONResponse(http.StatusOK, response), nil
}

// getBot handles GET /bots/{id}
func (h *BotHandler) getBot(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	bot, err := h.botService.GetBot(ctx, botID, userID)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(http.StatusNotFound, "Bot not found", ""), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(http.StatusForbidden, "Access denied", ""), nil
		}
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to get bot", err.Error()), nil
	}

	// Update last used time
	_ = h.botService.UpdateLastUsedTime(ctx, botID)

	response := models.BotResponse{
		Bot:       bot,
		RequestID: h.generateRequestID(),
	}

	return utils.JSONResponse(http.StatusOK, response), nil
}

// createBot handles POST /bots
func (h *BotHandler) createBot(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var createReq models.CreateBotRequest
	if err := json.Unmarshal([]byte(request.Body), &createReq); err != nil {
		return utils.ErrorResponse(http.StatusBadRequest, "Invalid JSON", err.Error()), nil
	}

	// Validate request
	if err := h.validator.Struct(&createReq); err != nil {
		return utils.ErrorResponse(http.StatusBadRequest, "Validation failed", err.Error()), nil
	}

	// Set defaults if not provided
	if len(createReq.ActiveModels) == 0 {
		createReq.ActiveModels = []string{"anthropic.claude-3-5-sonnet-20241022-v2:0"}
	}
	if createReq.GenerationParams == (models.GenerationParams{}) {
		createReq.GenerationParams = models.DefaultGenerationParams()
	}
	if createReq.KnowledgeBaseConfig == (models.KnowledgeBaseConfig{}) {
		createReq.KnowledgeBaseConfig = models.DefaultKnowledgeBaseConfig()
	}

	// Create bot
	bot := &models.Bot{
		ID:                    ulid.Make().String(),
		Title:                 createReq.Title,
		Description:           createReq.Description,
		Instruction:           createReq.Instruction,
		OwnerUserID:           userID,
		CreateTime:            time.Now(),
		LastUsedTime:          time.Now(),
		IsPublic:              createReq.IsPublic,
		IsStarred:             false,
		GenerationParams:      createReq.GenerationParams,
		KnowledgeBaseID:       createReq.KnowledgeBaseID,
		KnowledgeBaseConfig:   createReq.KnowledgeBaseConfig,
		ConversationStarters:  createReq.ConversationStarters,
		ActiveModels:          createReq.ActiveModels,
		AgentTools:            createReq.AgentTools,
		DisplayRetrievedChunks: createReq.DisplayRetrievedChunks,
	}

	if err := h.botService.CreateBot(ctx, bot); err != nil {
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to create bot", err.Error()), nil
	}

	response := models.BotResponse{
		Bot:       bot,
		Message:   "Bot created successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.JSONResponse(http.StatusCreated, response), nil
}

// updateBot handles PUT/PATCH /bots/{id}
func (h *BotHandler) updateBot(ctx context.Context, request events.APIGatewayProxyRequest, botID, userID string) (events.APIGatewayProxyResponse, error) {
	var updateReq models.UpdateBotRequest
	if err := json.Unmarshal([]byte(request.Body), &updateReq); err != nil {
		return utils.ErrorResponse(http.StatusBadRequest, "Invalid JSON", err.Error()), nil
	}

	// Validate request
	if err := h.validator.Struct(&updateReq); err != nil {
		return utils.ErrorResponse(http.StatusBadRequest, "Validation failed", err.Error()), nil
	}

	// Check if bot exists and user has permission
	existingBot, err := h.botService.GetBot(ctx, botID, userID)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(http.StatusNotFound, "Bot not found", ""), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(http.StatusForbidden, "Access denied", ""), nil
		}
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to get bot", err.Error()), nil
	}

	// Only owner can modify
	if existingBot.OwnerUserID != userID {
		return utils.ErrorResponse(http.StatusForbidden, "Only the owner can modify this bot", ""), nil
	}

	if err := h.botService.UpdateBot(ctx, botID, &updateReq); err != nil {
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to update bot", err.Error()), nil
	}

	// Get updated bot
	updatedBot, _ := h.botService.GetBot(ctx, botID, userID)

	response := models.BotResponse{
		Bot:       updatedBot,
		Message:   "Bot updated successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.JSONResponse(http.StatusOK, response), nil
}

// deleteBot handles DELETE /bots/{id}
func (h *BotHandler) deleteBot(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	// Check if bot exists and user has permission
	existingBot, err := h.botService.GetBot(ctx, botID, userID)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(http.StatusNotFound, "Bot not found", ""), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(http.StatusForbidden, "Access denied", ""), nil
		}
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to get bot", err.Error()), nil
	}

	// Only owner can delete
	if existingBot.OwnerUserID != userID {
		return utils.ErrorResponse(http.StatusForbidden, "Only the owner can delete this bot", ""), nil
	}

	if err := h.botService.DeleteBot(ctx, botID); err != nil {
		return utils.ErrorResponse(http.StatusInternalServerError, "Failed to delete bot", err.Error()), nil
	}

	response := models.BotResponse{
		Message:   "Bot deleted successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.JSONResponse(http.StatusOK, response), nil
}

// extractUserID extracts user ID from request (simplified implementation)
func (h *BotHandler) extractUserID(request events.APIGatewayProxyRequest) string {
	// In a real implementation, extract from JWT token
	// For demo purposes, use a header or return hardcoded value
	if userID := request.Headers["X-User-ID"]; userID != "" {
		return userID
	}
	// For demo/development, return the hardcoded user
	return "bobrt-user-id"
}

// generateRequestID generates a unique request ID
func (h *BotHandler) generateRequestID() string {
	return fmt.Sprintf("req_%s", ulid.Make().String())
}