package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

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
	
	// Extract user ID and groups from JWT token
	userID, userGroups := h.extractUserInfoFromJWT(request)
	if userID == "" {
		return utils.ErrorResponse(errors.New("unauthorized"), http.StatusUnauthorized), nil
	}

	switch method {
	case "GET":
		if len(pathSegments) == 1 { // GET /bots
			return h.listBots(ctx, request, userID, userGroups)
		} else if len(pathSegments) == 2 {
			if pathSegments[1] == "public" { // GET /bots/public
				return h.listPublicBots(ctx, request)
			} else if pathSegments[1] == "pinned" { // GET /bots/pinned
				return h.listPinnedBots(ctx, request)
			} else if pathSegments[1] == "shared" { // GET /bots/shared
				return h.listSharedBots(ctx, request, userID, userGroups)
			} else if pathSegments[1] == "search" { // GET /bots/search
				return h.searchBots(ctx, request, userID, userGroups)
			} else if pathSegments[1] == "popular" { // GET /bots/popular
				return h.getPopularBots(ctx, request, userID, userGroups)
			} else if pathSegments[1] == "discovery" { // GET /bots/discovery
				return h.getDiscoveryBots(ctx, request, userID, userGroups)
			} else { // GET /bots/{id}
				return h.getBot(ctx, pathSegments[1], userID, userGroups)
			}
		} else if len(pathSegments) == 3 {
			if pathSegments[2] == "visibility" { // GET /bots/{id}/visibility
				return h.getBotVisibility(ctx, pathSegments[1], userID)
			} else if pathSegments[2] == "stack-status" { // GET /bots/{id}/stack-status
				return h.getBotStackStatus(ctx, pathSegments[1], userID)
			} else if pathSegments[2] == "summary" { // GET /bots/{id}/summary
				return h.getBotSummary(ctx, pathSegments[1], userID, userGroups)
			}
		}
	case "POST":
		if len(pathSegments) == 1 { // POST /bots
			return h.createBot(ctx, request, userID)
		} else if len(pathSegments) == 3 && pathSegments[2] == "access" { // POST /bots/{id}/access
			return h.addBotAccess(ctx, pathSegments[1], userID, userGroups)
		}
	case "PUT", "PATCH":
		if len(pathSegments) == 2 { // PUT/PATCH /bots/{id}
			return h.updateBot(ctx, request, pathSegments[1], userID)
		} else if len(pathSegments) == 3 && pathSegments[2] == "visibility" { // PATCH /bots/{id}/visibility
			return h.updateBotVisibility(ctx, request, pathSegments[1], userID)
		} else if len(pathSegments) == 3 && pathSegments[2] == "starred" { // PATCH /bots/{id}/starred
			return h.updateBotStarred(ctx, request, pathSegments[1], userID, userGroups)
		}
	case "DELETE":
		if len(pathSegments) == 2 { // DELETE /bots/{id}
			return h.deleteBot(ctx, pathSegments[1], userID)
		}
	}

	return utils.ErrorResponse(errors.New("Endpoint not found"), http.StatusNotFound), nil
}

// addBotAccess handles POST /bots/{id}/access - creates bot alias for user to access shared bot
func (h *BotHandler) addBotAccess(ctx context.Context, botID, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	// Get or create bot alias for the shared bot
	alias, err := h.botService.GetOrCreateBotAlias(ctx, botID, userID, userGroups)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied: bot is not accessible to you"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to add bot access"), http.StatusInternalServerError), nil
	}

	// If alias is nil, user already owns the bot
	if alias == nil {
		return utils.ErrorResponse(errors.New("You already own this bot"), http.StatusBadRequest), nil
	}

	response := map[string]interface{}{
		"message":    "Bot added to your collection successfully",
		"alias":      alias.ToSummary(),
		"requestId":  h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusCreated, response), nil
}

// listBots handles GET /bots
func (h *BotHandler) listBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters (handle nil case)
	var scope, limitStr string
	var starred bool
	if request.QueryStringParameters != nil {
		scope = request.QueryStringParameters["scope"]
		starred = request.QueryStringParameters["starred"] == "true"
		limitStr = request.QueryStringParameters["limit"]
	}
	if scope == "" {
		scope = "private" // Default to private bots (user's own bots)
	}

	// Validate scope parameter
	validScopes := map[string]bool{"private": true, "public": true, "shared": true, "accessible": true}
	if !validScopes[scope] {
		return utils.ErrorResponse(fmt.Errorf("invalid scope: %s. Must be one of: private, public, shared, accessible", scope), http.StatusBadRequest), nil
	}

	limit := 100 // default
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// Get bot summaries based on scope
	summaries, err := h.botService.GetBotSummaries(ctx, userID, userGroups, false, scope, starred, limit)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to list bots"), http.StatusInternalServerError), nil
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getBot handles GET /bots/{id}
func (h *BotHandler) getBot(ctx context.Context, botID, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	bot, err := h.botService.GetBot(ctx, botID, userID, userGroups, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Update usage analytics (last used time + usage count)
	_ = h.botService.IncrementBotUsage(ctx, botID)

	response := models.BotResponse{
		Bot:       bot,
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// createBot handles POST /bots
func (h *BotHandler) createBot(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	log.Printf("🤖 Creating bot for user: %s", userID)
	log.Printf("📝 Request body length: %d bytes", len(request.Body))
	
	var createReq models.CreateBotRequest
	if err := json.Unmarshal([]byte(request.Body), &createReq); err != nil {
		log.Printf("❌ JSON unmarshal error: %v", err)
		return utils.ErrorResponse(errors.New("Invalid JSON format"), http.StatusBadRequest), nil
	}

	// Log key request details for debugging
	log.Printf("📋 Bot details - Title: %s, KB Option: %v, Tools: %d", 
		createReq.Title, 
		createReq.ExistingKnowledgeBaseID != nil || createReq.KnowledgeBaseCreation != nil,
		len(createReq.AgentTools))

	// Validate request
	if err := h.validator.Struct(&createReq); err != nil {
		log.Printf("❌ Validation error: %v", err)
		return utils.ErrorResponse(fmt.Errorf("Validation failed: %v", err), http.StatusBadRequest), nil
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
	if createReq.SharedScope == "" {
		createReq.SharedScope = models.SharedScopePrivate
	}

	// Create bot using the service
	log.Printf("🚀 Calling BotService.CreateBot...")
	result, err := h.botService.CreateBot(ctx, &createReq, userID, []string{})
	if err != nil {
		log.Printf("❌ Bot creation failed: %v", err)
		return utils.ErrorResponse(fmt.Errorf("Failed to create bot: %v", err), http.StatusInternalServerError), nil
	}
	
	log.Printf("✅ Bot created successfully - ID: %s, Stack deployment: %v", 
		result.Bot.ID, result.StackDeploymentStarted)

	response := models.BotResponse{
		Bot:       result.Bot,
		Message:   result.Message,
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
	existingBot, err := h.botService.GetBot(ctx, botID, userID, []string{}, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Check edit permissions
	if !existingBot.IsEditableByUser(userID, []string{}, false) {
		return utils.ErrorResponse(errors.New("Access denied: you can only edit bots you own"), http.StatusForbidden), nil
	}

	if err := h.botService.UpdateBot(ctx, botID, &updateReq); err != nil {
		return utils.ErrorResponse(errors.New("Failed to update bot"), http.StatusInternalServerError), nil
	}

	// Get updated bot
	updatedBot, _ := h.botService.GetBot(ctx, botID, userID, []string{}, false)

	response := models.BotResponse{
		Bot:       updatedBot,
		Message:   "Bot updated successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// deleteBot handles DELETE /bots/{id}
func (h *BotHandler) deleteBot(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	// Delete bot with permission check
	if err := h.botService.DeleteBot(ctx, botID, userID, []string{}, false); err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to delete bot"), http.StatusInternalServerError), nil
	}

	response := models.BotResponse{
		Message:   "Bot deleted successfully",
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// extractUserInfoFromJWT extracts user ID and groups from JWT claims
func (h *BotHandler) extractUserInfoFromJWT(request events.APIGatewayProxyRequest) (string, []string) {
	userID := ""
	var userGroups []string
	
	// Debug: Log the entire authorizer context
	fmt.Printf("DEBUG: Full RequestContext.Authorizer: %+v\n", request.RequestContext.Authorizer)
	
	// Extract user ID from the authorizer context (Cognito User Pool)
	if id, ok := request.RequestContext.Authorizer["sub"].(string); ok {
		userID = id
	}
	if id, ok := request.RequestContext.Authorizer["userId"].(string); ok && userID == "" {
		userID = id
	}
	
	// Check for claims in different format
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, exists := claims["sub"].(string); exists && userID == "" {
			userID = sub
		}
		
		// Extract groups from cognito:groups claim
		if groups, exists := claims["cognito:groups"].(string); exists {
			// Groups are comma-separated in the JWT claim
			userGroups = strings.Split(groups, ",")
		}
	}
	
	// Legacy fallback for development
	if userID == "" {
		if id := request.Headers["X-User-ID"]; id != "" {
			userID = id
		}
	}
	
	// Parse X-User-Groups header for development
	if groupsHeader := request.Headers["X-User-Groups"]; groupsHeader != "" {
		userGroups = strings.Split(groupsHeader, ",")
	}
	
	fmt.Printf("DEBUG: Extracted userID: %s, groups: %v\n", userID, userGroups)
	return userID, userGroups
}

// listPublicBots handles GET /bots/public
func (h *BotHandler) listPublicBots(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	summaries, err := h.botService.GetBotSummaries(ctx, "", []string{}, false, "public", false, 100)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to get public bots"), http.StatusInternalServerError), nil
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// listPinnedBots handles GET /bots/pinned
func (h *BotHandler) listPinnedBots(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get pinned bots (public bots with pinned status)
	bots, err := h.botService.ListBots(ctx, "", []string{}, false, "public", false, 100)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to get pinned bots"), http.StatusInternalServerError), nil
	}

	// Filter for pinned bots
	var pinnedBots []models.Bot
	for _, bot := range bots {
		if bot.IsPinned() {
			pinnedBots = append(pinnedBots, bot)
		}
	}

	// Convert to summaries
	summaries := make([]models.BotSummary, len(pinnedBots))
	for i, bot := range pinnedBots {
		summaries[i] = bot.ToSummary()
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// listSharedBots handles GET /bots/shared
func (h *BotHandler) listSharedBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	summaries, err := h.botService.GetBotSummaries(ctx, userID, userGroups, false, "shared", false, 100)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to get shared bots"), http.StatusInternalServerError), nil
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// searchBots handles GET /bots/search
func (h *BotHandler) searchBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	query := request.QueryStringParameters["query"]
	scope := request.QueryStringParameters["scope"] // public, private, shared, accessible

	if query == "" {
		return utils.ErrorResponse(errors.New("Query parameter is required"), http.StatusBadRequest), nil
	}

	// Validate scope parameter if provided
	if scope != "" {
		validScopes := map[string]bool{"private": true, "public": true, "shared": true, "accessible": true}
		if !validScopes[scope] {
			return utils.ErrorResponse(fmt.Errorf("invalid scope: %s. Must be one of: private, public, shared, accessible", scope), http.StatusBadRequest), nil
		}
	}

	// For now, do a simple search using ListBots and filter by title/description
	bots, err := h.botService.ListBots(ctx, userID, userGroups, false, scope, false, 100)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to search bots"), http.StatusInternalServerError), nil
	}

	// Simple text search filter
	var filteredBots []models.Bot
	for _, bot := range bots {
		if strings.Contains(strings.ToLower(bot.Title), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(bot.Description), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(bot.Instruction), strings.ToLower(query)) {
			filteredBots = append(filteredBots, bot)
		}
	}

	// Convert to summaries
	summaries := make([]models.BotSummary, len(filteredBots))
	for i, bot := range filteredBots {
		summaries[i] = bot.ToSummary()
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getBotVisibility handles GET /bots/{id}/visibility
func (h *BotHandler) getBotVisibility(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	bot, err := h.botService.GetBot(ctx, botID, userID, []string{}, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Check edit permissions for visibility settings
	if !bot.IsEditableByUser(userID, []string{}, false) {
		return utils.ErrorResponse(errors.New("Access denied: you can only view visibility settings for bots you own"), http.StatusForbidden), nil
	}

	visibilityResponse := map[string]interface{}{
		"sharedScope":  bot.SharedScope,
		"sharedStatus": bot.SharedStatus,
		"allowedUsers": bot.AllowedUsers,
		"allowedGroups": bot.AllowedGroups,
		"requestId":    h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, visibilityResponse), nil
}

// updateBotVisibility handles PATCH /bots/{id}/visibility
func (h *BotHandler) updateBotVisibility(ctx context.Context, request events.APIGatewayProxyRequest, botID, userID string) (events.APIGatewayProxyResponse, error) {
	var visibilityReq struct {
		TargetSharedScope string   `json:"targetSharedScope" validate:"required,oneof=private partial public"`
		AllowedUsers      []string `json:"allowedUsers,omitempty"`
		AllowedGroups     []string `json:"allowedGroups,omitempty"`
	}

	if err := json.Unmarshal([]byte(request.Body), &visibilityReq); err != nil {
		return utils.ErrorResponse(errors.New("Invalid JSON"), http.StatusBadRequest), nil
	}

	if err := h.validator.Struct(&visibilityReq); err != nil {
		return utils.ErrorResponse(errors.New("Validation failed"), http.StatusBadRequest), nil
	}

	// Update bot sharing configuration
	err := h.botService.UpdateBotSharing(ctx, botID, userID, visibilityReq.TargetSharedScope, "shared", visibilityReq.AllowedUsers, visibilityReq.AllowedGroups)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to update bot visibility"), http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"message":   "Bot visibility updated successfully",
		"requestId": h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getBotStackStatus handles GET /bots/{id}/stack-status
func (h *BotHandler) getBotStackStatus(ctx context.Context, botID, userID string) (events.APIGatewayProxyResponse, error) {
	status, err := h.botService.GetBotStackStatus(ctx, botID, userID, []string{}, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get stack status"), http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"stackStatus": status,
		"requestId":   h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getBotSummary handles GET /bots/{id}/summary (bedrock-chat compatibility)
func (h *BotHandler) getBotSummary(ctx context.Context, botID, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	// Get bot with alias handling (core bedrock-chat pattern)
	bot, alias, err := h.botService.GetBotWithAlias(ctx, botID, userID, userGroups, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot summary"), http.StatusInternalServerError), nil
	}

	// Update usage analytics for the bot
	_ = h.botService.IncrementBotUsage(ctx, bot.ID)

	// If there's an alias, also update its usage
	if alias != nil {
		_ = h.botService.UpdateAliasUsage(ctx, alias.ID)
	}

	// Create summary response (use alias summary if available, else original bot)
	var summary models.BotSummary
	if alias != nil {
		summary = alias.ToSummary()
	} else {
		summary = bot.ToSummary()
	}

	response := map[string]interface{}{
		"summary":   summary,
		"isAlias":   alias != nil,
		"requestId": h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// updateBotStarred handles PATCH /bots/{id}/starred
func (h *BotHandler) updateBotStarred(ctx context.Context, request events.APIGatewayProxyRequest, botID, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	var starReq struct {
		Starred bool `json:"starred"`
	}

	if err := json.Unmarshal([]byte(request.Body), &starReq); err != nil {
		return utils.ErrorResponse(errors.New("Invalid JSON"), http.StatusBadRequest), nil
	}

	// Check if this is an alias or original bot
	bot, alias, err := h.botService.GetBotWithAlias(ctx, botID, userID, userGroups, false)
	if err != nil {
		if err == services.ErrBotNotFound {
			return utils.ErrorResponse(errors.New("Bot not found"), http.StatusNotFound), nil
		}
		if err == services.ErrUnauthorized {
			return utils.ErrorResponse(errors.New("Access denied"), http.StatusForbidden), nil
		}
		return utils.ErrorResponse(errors.New("Failed to get bot"), http.StatusInternalServerError), nil
	}

	// Update starred status
	if alias != nil {
		// Update alias starred status
		err = h.botService.ToggleAliasStarred(ctx, alias.ID, starReq.Starred)
	} else {
		// Update original bot starred status  
		err = h.botService.ToggleStar(ctx, bot.ID, userID, starReq.Starred)
	}

	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to update starred status"), http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"message":   fmt.Sprintf("Bot %s successfully", map[bool]string{true: "starred", false: "unstarred"}[starReq.Starred]),
		"starred":   starReq.Starred,
		"requestId": h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getPopularBots handles GET /bots/popular (bedrock-chat compatibility)
func (h *BotHandler) getPopularBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	limitStr := request.QueryStringParameters["limit"]
	limit := 20 // default
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// Get popular bots using usage analytics
	summaries, err := h.botService.GetPopularBots(ctx, userID, userGroups, limit)
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to get popular bots"), http.StatusInternalServerError), nil
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// getDiscoveryBots handles GET /bots/discovery (random discovery feature)
func (h *BotHandler) getDiscoveryBots(ctx context.Context, request events.APIGatewayProxyRequest, userID string, userGroups []string) (events.APIGatewayProxyResponse, error) {
	limitStr := request.QueryStringParameters["limit"]
	limit := 10 // default for discovery
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// Get random public bots for discovery
	summaries, err := h.botService.GetBotSummaries(ctx, userID, userGroups, false, "public", false, limit*3) // Get more than needed
	if err != nil {
		return utils.ErrorResponse(errors.New("Failed to get discovery bots"), http.StatusInternalServerError), nil
	}

	// Shuffle and limit for random discovery
	if len(summaries) > limit {
		// Simple shuffle logic (not cryptographically secure, but fine for discovery)
		for i := len(summaries) - 1; i > 0; i-- {
			j := i % (len(summaries))
			summaries[i], summaries[j] = summaries[j], summaries[i]
		}
		summaries = summaries[:limit]
	}

	response := models.BotsResponse{
		Bots:      summaries,
		Count:     len(summaries),
		Total:     len(summaries),
		RequestID: h.generateRequestID(),
	}

	return utils.NewAPIResponse(http.StatusOK, response), nil
}

// generateRequestID generates a unique request ID
func (h *BotHandler) generateRequestID() string {
	return fmt.Sprintf("req_%s", ulid.Make().String())
}