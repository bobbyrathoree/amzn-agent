package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

// ChatHandler handles chat-related requests with full conversation management
type ChatHandler struct {
	chatService          *services.ChatService
	conversationRepo     repositories.ConversationRepository
	botService          *services.BotService
}

// NewChatHandler creates a new ChatHandler with conversation management
func NewChatHandler(
	chatService *services.ChatService,
	conversationRepo repositories.ConversationRepository,
	botService *services.BotService,
) *ChatHandler {
	return &ChatHandler{
		chatService:      chatService,
		conversationRepo: conversationRepo,
		botService:       botService,
	}
}

// extractUserID extracts user ID from API Gateway request context (Cognito claims)
func (h *ChatHandler) extractUserID(request events.APIGatewayProxyRequest) string {
	// Extract from Cognito authorizer context
	if id, ok := request.RequestContext.Authorizer["sub"].(string); ok && id != "" {
		return id
	}
	if id, ok := request.RequestContext.Authorizer["userId"].(string); ok && id != "" {
		return id
	}

	// Check for claims in different format
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, exists := claims["sub"].(string); exists && sub != "" {
			return sub
		}
	}

	// Legacy fallback for headers (development/testing)
	if id := request.Headers["X-User-ID"]; id != "" {
		return id
	}
	if id := request.Headers["x-user-id"]; id != "" {
		return id
	}

	return ""
}

// HandleBotChat handles POST /bots/{id}/chat requests
func (h *ChatHandler) HandleBotChat(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract bot ID from path parameters
	botID := request.PathParameters["id"]
	if botID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Bot ID is required"}`,
		}, nil
	}

	log.Printf("💬 Processing chat request for bot: %s", botID)

	// Parse request body
	var chatReq services.ChatRequest
	if err := json.Unmarshal([]byte(request.Body), &chatReq); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Invalid request body"}`,
		}, nil
	}

	if chatReq.Message == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Message is required"}`,
		}, nil
	}

	// Extract user information from request context
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}
	
	// TODO: Extract user groups from JWT token or user service
	userGroups := []string{} // For now, extract from headers if available
	isAdmin := false // For now, could be determined from user roles

	// Process chat with the bot
	response, err := h.chatService.ChatWithBot(ctx, botID, userID, userGroups, isAdmin, &chatReq)
	if err != nil {
		log.Printf("❌ Chat processing failed: %v", err)

		// Handle different error types
		if isNotFoundError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Headers:    corsHeaders(),
				Body:       fmt.Sprintf(`{"error": "Bot not found: %s"}`, botID),
			}, nil
		}

		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"error": "Access denied to bot"}`,
			}, nil
		}

		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Chat processing failed: %s"}`, err.Error()),
		}, nil
	}

	// Marshal response
	responseBody, err := json.Marshal(response)
	if err != nil {
		log.Printf("❌ Failed to marshal chat response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Chat response sent for bot %s", botID)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleGetBotConversations handles GET /bots/{id}/conversations
func (h *ChatHandler) HandleGetBotConversations(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	botID := request.PathParameters["id"]
	if botID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Bot ID is required"}`,
		}, nil
	}

	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("🗂️ Getting conversations for bot: %s, user: %s", botID, userID)

	// Verify user has access to this bot
	userGroups := []string{} // TODO: Extract from authentication context
	isAdmin := false
	_, err := h.botService.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		if isNotFoundError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Headers:    corsHeaders(),
				Body:       fmt.Sprintf(`{"error": "Bot not found: %s"}`, botID),
			}, nil
		}
		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"error": "Access denied to bot"}`,
			}, nil
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to verify bot access: %s"}`, err.Error()),
		}, nil
	}

	// Get conversation metadata for this bot-user combination
	conversations, err := h.conversationRepo.GetConversationsByBotAndUser(ctx, botID, userID)
	if err != nil {
		log.Printf("❌ Failed to get conversations: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to load conversations: %s"}`, err.Error()),
		}, nil
	}

	// Convert to metadata format for efficient listing
	conversationMetas := make([]models.ConversationMeta, len(conversations))
	for i, conv := range conversations {
		// Get last message preview
		lastMessage := ""
		if len(conv.MessageMap) > 0 && conv.LastMessageID != "" {
			if msg, exists := conv.MessageMap[conv.LastMessageID]; exists && len(msg.Content) > 0 {
				if msg.Content[0].Text != "" {
					lastMessage = msg.Content[0].Text
					if len(lastMessage) > 100 {
						lastMessage = lastMessage[:100] + "..."
					}
				}
			}
		}

		conversationMetas[i] = models.ConversationMeta{
			ID:             conv.ID,
			UserID:         conv.UserID,
			BotID:          conv.BotID,
			Title:          conv.Title,
			SessionModelID: conv.SessionModelID,
			CreatedAt:      conv.CreatedAt,
			UpdatedAt:      conv.UpdatedAt,
			MessageCount:   len(conv.MessageMap),
			LastMessage:    lastMessage,
		}
	}

	response := map[string]interface{}{
		"conversations": conversationMetas,
		"count":         len(conversationMetas),
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Returned %d conversations for bot %s", len(conversationMetas), botID)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleStartBotConversation handles POST /bots/{id}/conversations
func (h *ChatHandler) HandleStartBotConversation(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	botID := request.PathParameters["id"]
	if botID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Bot ID is required"}`,
		}, nil
	}

	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("🆕 Starting new conversation for bot: %s, user: %s", botID, userID)

	// Parse request body for conversation options
	type CreateConversationRequest struct {
		Title          string  `json:"title,omitempty"`
		SessionModelID *string `json:"sessionModelId,omitempty"`
	}
	
	var createReq CreateConversationRequest
	if request.Body != "" {
		if err := json.Unmarshal([]byte(request.Body), &createReq); err != nil {
			// Non-fatal error, just use default title
			log.Printf("Warning: failed to parse create conversation request: %v", err)
		}
	}

	// Verify user has access to this bot
	userGroups := []string{} // TODO: Extract from authentication context
	isAdmin := false
	bot, err := h.botService.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		if isNotFoundError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Headers:    corsHeaders(),
				Body:       fmt.Sprintf(`{"error": "Bot not found: %s"}`, botID),
			}, nil
		}
		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"error": "Access denied to bot"}`,
			}, nil
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to verify bot access: %s"}`, err.Error()),
		}, nil
	}

	// Generate conversation title if not provided
	title := createReq.Title
	if title == "" {
		title = fmt.Sprintf("New conversation with %s", bot.Title)
	}

	// Create conversation using repository
	conversation, err := h.conversationRepo.CreateConversationWithDetails(ctx, userID, &botID, title, createReq.SessionModelID)
	if err != nil {
		log.Printf("❌ Failed to create conversation: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to create conversation: %s"}`, err.Error()),
		}, nil
	}

	response := map[string]interface{}{
		"conversationId": conversation.ID,
		"botId":          botID,
		"title":          conversation.Title,
		"createdAt":      conversation.CreatedAt,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleGetBotConversation handles GET /bots/{id}/conversations/{conversationId}
func (h *ChatHandler) HandleGetBotConversation(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	botID := request.PathParameters["id"]
	conversationID := request.PathParameters["conversationId"]
	
	if botID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Bot ID is required"}`,
		}, nil
	}
	
	if conversationID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation ID is required"}`,
		}, nil
	}

	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("📖 Getting conversation: %s for bot: %s, user: %s", conversationID, botID, userID)

	// Verify user has access to this bot
	userGroups := []string{} // TODO: Extract from authentication context
	isAdmin := false
	_, err := h.botService.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		if isNotFoundError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusNotFound,
				Headers:    corsHeaders(),
				Body:       fmt.Sprintf(`{"error": "Bot not found: %s"}`, botID),
			}, nil
		}
		if isAccessDeniedError(err) {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusForbidden,
				Headers:    corsHeaders(),
				Body:       `{"error": "Access denied to bot"}`,
			}, nil
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to verify bot access: %s"}`, err.Error()),
		}, nil
	}

	// Get the conversation with full message history
	conversation, err := h.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		log.Printf("❌ Failed to get conversation: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Conversation not found: %s"}`, conversationID),
		}, nil
	}

	// Verify conversation belongs to this user and bot
	if conversation.UserID != userID {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusForbidden,
			Headers:    corsHeaders(),
			Body:       `{"error": "Access denied to conversation"}`,
		}, nil
	}

	if conversation.BotID == nil || *conversation.BotID != botID {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation does not belong to this bot"}`,
		}, nil
	}

	// Convert message map to ordered slice
	messages := make([]*models.Message, 0, len(conversation.MessageMap))
	for _, msg := range conversation.MessageMap {
		messages = append(messages, msg)
	}

	// Sort messages by creation time
	for i := 0; i < len(messages)-1; i++ {
		for j := i + 1; j < len(messages); j++ {
			if messages[i].CreatedAt.After(messages[j].CreatedAt) {
				messages[i], messages[j] = messages[j], messages[i]
			}
		}
	}

	response := map[string]interface{}{
		"conversation": conversation,
		"messages":     messages,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Returned conversation %s with %d messages", conversationID, len(messages))

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleDeleteBotConversation handles DELETE /bots/{id}/conversations/{conversationId}
func (h *ChatHandler) HandleDeleteBotConversation(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	botID := request.PathParameters["id"]
	conversationID := request.PathParameters["conversationId"]
	
	if botID == "" || conversationID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Bot ID and Conversation ID are required"}`,
		}, nil
	}

	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("🗑️ Deleting conversation %s for bot %s by user %s", conversationID, botID, userID)

	// First verify the conversation exists and belongs to this user/bot
	conversation, err := h.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		log.Printf("❌ Failed to get conversation for deletion: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation not found"}`,
		}, nil
	}

	// Verify conversation belongs to this user and bot
	if conversation.UserID != userID {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusForbidden,
			Headers:    corsHeaders(),
			Body:       `{"error": "Access denied to conversation"}`,
		}, nil
	}

	if conversation.BotID == nil || *conversation.BotID != botID {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation does not belong to this bot"}`,
		}, nil
	}

	// Delete the conversation
	err = h.conversationRepo.DeleteConversation(ctx, conversationID, userID)
	if err != nil {
		log.Printf("❌ Failed to delete conversation: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to delete conversation: %s"}`, err.Error()),
		}, nil
	}

	log.Printf("✅ Successfully deleted conversation %s", conversationID)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       `{"message": "Conversation deleted successfully"}`,
	}, nil
}

// HandleListAllConversations handles GET /conversations - list all conversations for user
func (h *ChatHandler) HandleListAllConversations(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("🗂️ Listing all conversations for user: %s", userID)

	// Parse query parameters
	limitStr := request.QueryStringParameters["limit"]
	nextToken := request.QueryStringParameters["nextToken"]
	limit := 50 // default
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	// Get all conversations for user
	conversations, newNextToken, err := h.conversationRepo.ListConversations(ctx, userID, limit, nextToken)
	if err != nil {
		log.Printf("❌ Failed to list conversations: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to list conversations: %s"}`, err.Error()),
		}, nil
	}

	response := map[string]interface{}{
		"conversations": conversations,
		"count":         len(conversations),
	}
	if newNextToken != "" {
		response["nextToken"] = newNextToken
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Returned %d conversations for user %s", len(conversations), userID)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleGetConversationById handles GET /conversations/{id}
func (h *ChatHandler) HandleGetConversationById(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	conversationID := request.PathParameters["id"]
	if conversationID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation ID is required"}`,
		}, nil
	}

	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("📖 Getting conversation: %s for user: %s", conversationID, userID)

	// Get the conversation with full message history
	conversation, err := h.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		log.Printf("❌ Failed to get conversation: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Conversation not found: %s"}`, conversationID),
		}, nil
	}

	// Verify conversation belongs to this user
	if conversation.UserID != userID {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusForbidden,
			Headers:    corsHeaders(),
			Body:       `{"error": "Access denied to conversation"}`,
		}, nil
	}

	// Convert message map to ordered slice
	messages := make([]*models.Message, 0, len(conversation.MessageMap))
	for _, msg := range conversation.MessageMap {
		messages = append(messages, msg)
	}

	// Sort messages by creation time
	for i := 0; i < len(messages)-1; i++ {
		for j := i + 1; j < len(messages); j++ {
			if messages[i].CreatedAt.After(messages[j].CreatedAt) {
				messages[i], messages[j] = messages[j], messages[i]
			}
		}
	}

	response := map[string]interface{}{
		"conversation": conversation,
		"messages":     messages,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       `{"error": "Failed to prepare response"}`,
		}, nil
	}

	log.Printf("✅ Returned conversation %s with %d messages", conversationID, len(messages))

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       string(responseBody),
	}, nil
}

// HandleDeleteConversationById handles DELETE /conversations/{id}
func (h *ChatHandler) HandleDeleteConversationById(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	conversationID := request.PathParameters["id"]
	if conversationID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation ID is required"}`,
		}, nil
	}

	// Extract user information
	userID := h.extractUserID(request)
	if userID == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    corsHeaders(),
			Body:       `{"error": "User ID required"}`,
		}, nil
	}

	log.Printf("🗑️ Deleting conversation %s for user %s", conversationID, userID)

	// First verify the conversation exists and belongs to this user
	conversation, err := h.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		log.Printf("❌ Failed to get conversation for deletion: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers:    corsHeaders(),
			Body:       `{"error": "Conversation not found"}`,
		}, nil
	}

	// Verify conversation belongs to this user
	if conversation.UserID != userID {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusForbidden,
			Headers:    corsHeaders(),
			Body:       `{"error": "Access denied to conversation"}`,
		}, nil
	}

	// Delete the conversation
	err = h.conversationRepo.DeleteConversation(ctx, conversationID, userID)
	if err != nil {
		log.Printf("❌ Failed to delete conversation: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Failed to delete conversation: %s"}`, err.Error()),
		}, nil
	}

	log.Printf("✅ Successfully deleted conversation %s", conversationID)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    corsHeaders(),
		Body:       `{"message": "Conversation deleted successfully"}`,
	}, nil
}

// HandleChatRequest routes different chat-related requests
func (h *ChatHandler) HandleChatRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("🔀 Routing chat request: %s %s", request.HTTPMethod, request.Path)

	switch {
	// Top-level /conversations routes (without bot context)
	case request.HTTPMethod == "GET" && request.Path == "/conversations":
		return h.HandleListAllConversations(ctx, request)

	case request.HTTPMethod == "GET" && matches(request.Path, "/conversations/{id}"):
		return h.HandleGetConversationById(ctx, request)

	case request.HTTPMethod == "DELETE" && matches(request.Path, "/conversations/{id}"):
		return h.HandleDeleteConversationById(ctx, request)

	// Bot-scoped /bots/{id}/conversations routes
	case request.HTTPMethod == "POST" && matches(request.Path, "/bots/{id}/chat"):
		return h.HandleBotChat(ctx, request)

	case request.HTTPMethod == "GET" && matches(request.Path, "/bots/{id}/conversations/{conversationId}"):
		return h.HandleGetBotConversation(ctx, request)

	case request.HTTPMethod == "GET" && matches(request.Path, "/bots/{id}/conversations"):
		return h.HandleGetBotConversations(ctx, request)

	case request.HTTPMethod == "POST" && matches(request.Path, "/bots/{id}/conversations"):
		return h.HandleStartBotConversation(ctx, request)

	case request.HTTPMethod == "DELETE" && matches(request.Path, "/bots/{id}/conversations/{conversationId}"):
		return h.HandleDeleteBotConversation(ctx, request)

	case request.HTTPMethod == "OPTIONS":
		// Handle CORS preflight
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, DELETE, OPTIONS",
				"Access-Control-Allow-Headers": "Content-Type, Authorization",
			},
		}, nil

	default:
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNotFound,
			Headers:    corsHeaders(),
			Body:       fmt.Sprintf(`{"error": "Chat endpoint not found: %s %s"}`, request.HTTPMethod, request.Path),
		}, nil
	}
}

// matches checks if a path matches a pattern (simple implementation)
func matches(path, pattern string) bool {
	// This is a simple implementation - would use a proper router in production
	if pattern == "/conversations/{id}" {
		// Path should be: /conversations/{id}
		return len(path) > 15 && path[:15] == "/conversations/" && !strings.Contains(path[15:], "/")
	}
	if pattern == "/bots/{id}/chat" {
		return len(path) > 6 && path[:6] == "/bots/" &&
			len(path) > 10 && path[len(path)-5:] == "/chat"
	}
	if pattern == "/bots/{id}/conversations/{conversationId}" {
		// Path should be: /bots/{botId}/conversations/{conversationId}
		if len(path) < 20 || path[:6] != "/bots/" {
			return false
		}
		// Find the "/conversations/" part
		conversationsIndex := strings.Index(path, "/conversations/")
		if conversationsIndex == -1 {
			return false
		}
		// Check if there's something after "/conversations/"
		return len(path) > conversationsIndex+15
	}
	if pattern == "/bots/{id}/conversations" {
		return len(path) > 6 && path[:6] == "/bots/" &&
			len(path) > 14 && path[len(path)-14:] == "/conversations"
	}
	return false
}