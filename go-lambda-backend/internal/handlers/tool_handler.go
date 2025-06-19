package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/bobbyrathore/go-lambda-backend/pkg/utils"
)

// ToolHandler handles tool-related API endpoints
type ToolHandler struct {
	toolService     *services.ToolExecutionService
	registryService *services.ToolRegistryService
}

// NewToolHandler creates a new tool handler
func NewToolHandler(toolService *services.ToolExecutionService, registryService *services.ToolRegistryService) *ToolHandler {
	return &ToolHandler{
		toolService:     toolService,
		registryService: registryService,
	}
}

// HandleToolRequest routes tool requests to appropriate handlers
func (h *ToolHandler) HandleToolRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract user ID from API Gateway request context (Cognito claims)
	userID, err := utils.ExtractUserIDFromRequest(request)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusUnauthorized), nil
	}

	// Route based on HTTP method and path
	switch request.HTTPMethod {
	case "GET":
		return h.handleGetRequest(ctx, request, userID)
	case "POST":
		return h.handlePostRequest(ctx, request, userID)
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Method not allowed"), http.StatusMethodNotAllowed), nil
	}
}

// handleGetRequest handles GET requests for tool endpoints
func (h *ToolHandler) handleGetRequest(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	path := request.Path
	pathParams := request.PathParameters

	switch {
	case strings.HasSuffix(path, "/tools"):
		return h.listTools(ctx, request)
	case strings.Contains(path, "/tools/") && pathParams["toolId"] != "":
		return h.getTool(ctx, pathParams["toolId"])
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Endpoint not found"), http.StatusNotFound), nil
	}
}

// handlePostRequest handles POST requests for tool endpoints
func (h *ToolHandler) handlePostRequest(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	path := request.Path

	switch {
	case strings.HasSuffix(path, "/tools/execute"):
		return h.executeTool(ctx, request, userID)
	case strings.HasSuffix(path, "/tools/stream"):
		return h.streamTool(ctx, request, userID)
	default:
		return utils.ErrorResponse(utils.ErrorFromString("Endpoint not found"), http.StatusNotFound), nil
	}
}

// List Tools Endpoint: GET /tools
func (h *ToolHandler) listTools(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Check for category filter
	category := request.QueryStringParameters["category"]
	
	var tools []*models.Tool
	var err error
	
	if category != "" {
		tools, err = h.registryService.GetToolsByCategory(category)
	} else {
		tools, err = h.registryService.GetAllTools()
	}
	
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	response := map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	}

	if category != "" {
		response["category"] = category
	}

	return utils.SuccessResponse(response), nil
}

// Get Tool Endpoint: GET /tools/{toolId}
func (h *ToolHandler) getTool(ctx context.Context, toolID string) (events.APIGatewayProxyResponse, error) {
	tool, err := h.registryService.GetTool(toolID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return utils.ErrorResponse(utils.ErrorFromString(fmt.Sprintf("Tool %s not found", toolID)), http.StatusNotFound), nil
		}
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	return utils.SuccessResponse(tool), nil
}

// Execute Tool Endpoint: POST /tools/execute
func (h *ToolHandler) executeTool(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var req models.ToolExecutionRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Set user ID and generate request ID
	req.Context.UserID = userID
	req.Context.RequestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	req.Context.Timestamp = time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Execute the tool
	result, err := h.toolService.ExecuteTool(ctx, req)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Update tool usage statistics
	h.registryService.UpdateToolUsage(req.ToolID)

	response := map[string]interface{}{
		"success":       result.Success,
		"result":        result,
		"executionTime": result.ExecutionTime,
		"toolId":        req.ToolID,
		"capability":    req.Capability,
	}

	return utils.SuccessResponse(response), nil
}

// Stream Tool Endpoint: POST /tools/stream
func (h *ToolHandler) streamTool(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	// For streaming, we'll return an error for now and implement later
	// This requires Server-Sent Events (SSE) support which is more complex in Lambda
	return utils.ErrorResponse(utils.ErrorFromString("Streaming not yet implemented"), http.StatusNotImplemented), nil
}

// Tool Execution with Chat Integration Endpoint: POST /tools/execute-in-chat
type ExecuteInChatRequest struct {
	ConversationID string                 `json:"conversationId"`
	ToolID         string                 `json:"toolId"`
	Capability     string                 `json:"capability"`
	Input          map[string]interface{} `json:"input"`
	BotID          string                 `json:"botId,omitempty"`
}

func (h *ToolHandler) executeToolInChat(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var req ExecuteInChatRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	// Validate required fields
	if req.ConversationID == "" || req.ToolID == "" || req.Capability == "" {
		return utils.ErrorResponse(utils.ErrorFromString("conversationId, toolId, and capability are required"), http.StatusBadRequest), nil
	}

	// Create tool execution request
	toolReq := models.ToolExecutionRequest{
		ToolID:         req.ToolID,
		Capability:     req.Capability,
		Input:          req.Input,
		ConversationID: req.ConversationID,
		Context: models.ExecutionContext{
			UserID:      userID,
			BotID:       req.BotID,
			Permissions: []string{"internet-access:read", "external-api:read"},
			RequestID:   fmt.Sprintf("chat_tool_%d", time.Now().UnixNano()),
			Timestamp:   time.Now(),
		},
	}

	// Execute the tool (this will automatically save to conversation)
	result, err := h.toolService.ExecuteTool(ctx, toolReq)
	if err != nil {
		return utils.ErrorResponse(err, http.StatusInternalServerError), nil
	}

	// Update tool usage statistics
	h.registryService.UpdateToolUsage(req.ToolID)

	response := map[string]interface{}{
		"success":        result.Success,
		"result":         result,
		"conversationId": req.ConversationID,
		"toolId":         req.ToolID,
		"capability":     req.Capability,
		"executionTime":  result.ExecutionTime,
		"message":        "Tool executed and results saved to conversation",
	}

	return utils.SuccessResponse(response), nil
}

// Tool Suggestions Endpoint: POST /tools/suggest
type SuggestToolsRequest struct {
	Content        string `json:"content"`
	ConversationID string `json:"conversationId,omitempty"`
	BotID          string `json:"botId,omitempty"`
}

func (h *ToolHandler) suggestTools(ctx context.Context, request events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	var req SuggestToolsRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return utils.ErrorResponse(err, http.StatusBadRequest), nil
	}

	if req.Content == "" {
		return utils.ErrorResponse(utils.ErrorFromString("content is required"), http.StatusBadRequest), nil
	}

	// TODO: Implement AI-powered tool suggestion logic
	// For now, return basic suggestions based on keywords
	suggestions := h.generateBasicToolSuggestions(req.Content)

	response := map[string]interface{}{
		"suggestions": suggestions,
		"content":     req.Content,
		"count":       len(suggestions),
	}

	return utils.SuccessResponse(response), nil
}

// generateBasicToolSuggestions provides basic tool suggestions based on content analysis
func (h *ToolHandler) generateBasicToolSuggestions(content string) []models.ToolSuggestion {
	content = strings.ToLower(content)
	var suggestions []models.ToolSuggestion

	// Web search suggestions
	if containsAny(content, []string{"search", "find", "look up", "what is", "who is", "when", "where", "how"}) {
		suggestions = append(suggestions, models.ToolSuggestion{
			ToolID:     "web-search",
			Capability: "search",
			Reason:     "Content suggests need for web search information",
			Confidence: 0.8,
			Priority:   3,
			Input: map[string]interface{}{
				"query":  extractSearchQuery(content),
				"engine": "auto",
				"limit":  5,
			},
		})
	}

	// Research suggestions
	if containsAny(content, []string{"research", "analyze", "study", "investigate", "comprehensive", "detailed analysis"}) {
		suggestions = append(suggestions, models.ToolSuggestion{
			ToolID:     "research-assistant",
			Capability: "comprehensive-research",
			Reason:     "Content indicates need for comprehensive research",
			Confidence: 0.9,
			Priority:   4,
			Input: map[string]interface{}{
				"topic": extractResearchTopic(content),
				"depth": "detailed",
			},
		})
	}

	return suggestions
}

// Helper functions

func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func extractSearchQuery(content string) string {
	// Simple extraction - in production, this would be more sophisticated
	words := strings.Fields(content)
	if len(words) > 10 {
		return strings.Join(words[:10], " ")
	}
	return content
}

func extractResearchTopic(content string) string {
	// Simple extraction - in production, this would use NLP
	words := strings.Fields(content)
	if len(words) > 5 {
		return strings.Join(words[:5], " ")
	}
	return content
}

// Tool Health Check Endpoint: GET /tools/health
func (h *ToolHandler) toolHealthCheck(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	stats := h.registryService.GetToolStats()
	
	health := map[string]interface{}{
		"status":     "healthy",
		"toolCount":  len(stats),
		"timestamp":  time.Now(),
		"statistics": stats,
	}

	return utils.SuccessResponse(health), nil
}