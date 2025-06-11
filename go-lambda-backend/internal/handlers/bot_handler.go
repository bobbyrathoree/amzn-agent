package handlers

import (
	"context"
	"encoding/json"
	"errors"
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
		return utils.ErrorResponse(errors.New("unauthorized"), http.StatusUnauthorized), nil
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

	return utils.ErrorResponse(errors.New("Endpoint not found"), http.StatusNotFound), nil
}

// listBots handles GET /bots
func (h *BotHandler) listBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters
	includePublic := request.QueryStringParameters["includePublic"] == "true"
	starred := request.QueryStringParameters["starred"] == "true"
	
	bots, err := h.botService.ListBots(ctx, userID, includePublic, starred)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to list bots"), http.StatusInternalServerError), nil
	}

	response := models.BotsResponse{
		Bots:      bots,
		Count:     len(bots),
		Total:     len(bots),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getBot handles GET /bots/{id}
func (h *BotHandler) getBot(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	bot, err := h.botService.GetBot(ctx, botID, userID)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Update last used time
	_ = h.botService.UpdateLastUsedTime(ctx, botID)

	response := models.BotResponse{
		Bot:       bot,
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// createBot handles POST /bots
func (h *BotHandler) createBot(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var createReq models.CreateBotRequest
	if err := json.Unmarshal([]byte(request.Body), &createReq); err != nil {
		return utils.ErrorResponse(errors.New("Invalid JSON"), http.StatusBadRequest), nil
	}

	// Validate request
	if err := h.validator.Struct(&createReq); err != nil {
		return utils.ErrorResponse(errors.New("Validation failed"), http.StatusBadRequest), nil
	}

	// Set defaults if not provided
	if len(createReq.ActiveModels) == 0 {
		createReq.ActiveModels = []string{"anthropic.claude-3-5-sonnet-20241022-v2:0"}
	}
	if createReq.GenerationParams.MaxTokens == 0 {
		createReq.GenerationParams = models.DefaultGenerationParams()
	}
	if createReq.KnowledgeBaseConfig.MaxResults == 0 {
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
		return utils.ErrorResponse(errors.New("Failed to create bot"), http.StatusInternalServerError), nil
	}

	response := models.BotResponse{
		Bot:       bot,
		Message:   "Bot created successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusCreated, response), nil
}

// updateBot handles PUT/PATCH /bots/{id}
func (h *BotHandler) updateBot(ctx context.Context, request events.APIGatewayProxyRequest, botID, userID string) (events.APIGatewayProxyResponse, error) {
	var updateReq models.UpdateBotRequest
	if err := json.Unmarshal([]byte(request.Body), &updateReq); err != nil {
		return utils.ErrorResponse(errors.New("Invalid JSON"), http.StatusBadRequest), nil
	}

	// Validate request
	if err := h.validator.Struct(&updateReq); err != nil {
		return utils.ErrorResponse(errors.New("Validation failed"), http.StatusBadRequest), nil
	}

	// Check if bot exists and user has permission
	existingBot, err := h.botService.GetBot(ctx, botID, userID)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Only owner can modify
	if existingBot.OwnerUserID != userID {
		return utils.ErrorResponse(errors.New("Only the owner can modify this bot"), http.StatusForbidden), nil
	}

	if err := h.botService.UpdateBot(ctx, botID, &updateReq); err != nil {
		return utils.ErrorResponse(errors.New("Failed to update bot"), http.StatusInternalServerError), nil
	}

	// Get updated bot
	updatedBot, _ := h.botService.GetBot(ctx, botID, userID)

	response := models.BotResponse{
		Bot:       updatedBot,
		Message:   "Bot updated successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// deleteBot handles DELETE /bots/{id}
func (h *BotHandler) deleteBot(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	// Check if bot exists and user has permission
	existingBot, err := h.botService.GetBot(ctx, botID, userID)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Only owner can delete
	if existingBot.OwnerUserID != userID {
		return utils.ErrorResponse(errors.New("Only the owner can delete this bot"), http.StatusForbidden), nil
	}

	if err := h.botService.DeleteBot(ctx, botID); err != nil {
		return utils.ErrorResponse(errors.New("Failed to delete bot"), http.StatusInternalServerError), nil
	}

	response := models.BotResponse{
		Message:   "Bot deleted successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// extractUserID extracts user ID from request (from Cognito JWT claims)
func (h *BotHandler) extractUserID(request events.APIGatewayProxyRequest) string {
	// Debug: Log the entire authorizer context
	fmt.Printf("DEBUG: Full RequestContext.Authorizer: %+v\n", request.RequestContext.Authorizer)
	
	// Extract user ID from the authorizer context (Cognito User Pool)
	if userID, ok := request.RequestContext.Authorizer["sub"].(string); ok {
		fmt.Printf("DEBUG: Found sub in authorizer context: %s\n", userID)
		return userID
	}
	if userID, ok := request.RequestContext.Authorizer["userId"].(string); ok {
		fmt.Printf("DEBUG: Found userId in authorizer context: %s\n", userID)
		return userID
	}
	
	// Check for claims in different format
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		fmt.Printf("DEBUG: Found claims in authorizer context: %+v\n", claims)
		if sub, exists := claims["sub"].(string); exists {
			fmt.Printf("DEBUG: Found sub in claims: %s\n", sub)
			return sub
		}
	}
	
	// Legacy fallback for development
	if userID := request.Headers["X-User-ID"]; userID != "" {
		fmt.Printf("DEBUG: Using X-User-ID header: %s\n", userID)
		return userID
	}
	
	fmt.Printf("DEBUG: No user ID found anywhere\n")
	return ""
}

// generateRequestID generates a unique request ID
func (h *BotHandler) generateRequestID() string {
	return fmt.Sprintf("req_%s", ulid.Make().String())
}