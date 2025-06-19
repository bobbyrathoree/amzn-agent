package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

// ToolExecutionService handles unified tool execution
type ToolExecutionService struct {
	registry                *ToolRegistryService
	vaultService           *VaultService
	chatService            *ChatService
	conversationRepository repositories.ConversationRepository
}

// NewToolExecutionService creates a new tool execution service
func NewToolExecutionService(
	registry *ToolRegistryService,
	vaultService *VaultService,
	chatService *ChatService,
	conversationRepository repositories.ConversationRepository,
) *ToolExecutionService {
	return &ToolExecutionService{
		registry:                registry,
		vaultService:           vaultService,
		chatService:            chatService,
		conversationRepository: conversationRepository,
	}
}

// ExecuteTool executes a tool capability with the given input
func (s *ToolExecutionService) ExecuteTool(ctx context.Context, req models.ToolExecutionRequest) (*models.ToolResult, error) {
	startTime := time.Now()
	
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get tool from registry
	tool, err := s.registry.GetTool(req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("tool not found: %w", err)
	}

	// Get capability
	capability := tool.GetCapability(req.Capability)
	if capability == nil {
		return nil, fmt.Errorf("capability %s not found for tool %s", req.Capability, req.ToolID)
	}

	// Check API key requirements
	apiKeysUsed, err := s.checkAndRetrieveAPIKeys(ctx, tool, req.Context.UserID)
	if err != nil {
		return nil, fmt.Errorf("API key validation failed: %w", err)
	}

	// Create execution context
	execCtx := &ToolExecutionContext{
		Tool:       tool,
		Capability: capability,
		Request:    req,
		APIKeys:    apiKeysUsed,
		StartTime:  startTime,
		Context:    ctx,
	}

	// Execute the tool
	result := s.executeToolCapability(execCtx)
	
	// Set execution metadata
	result.ExecutionTime = time.Since(startTime).Milliseconds()
	result.ExecutedAt = time.Now()
	result.ToolVersion = tool.Version
	result.APIKeysUsed = getServiceIDs(apiKeysUsed)

	// Save execution to conversation if specified
	if req.ConversationID != "" {
		if err := s.saveExecutionToConversation(ctx, req, result); err != nil {
			log.Printf("Failed to save tool execution to conversation: %v", err)
			// Don't fail the execution for this error
		}
	}

	return result, nil
}

// ToolExecutionContext holds context for tool execution
type ToolExecutionContext struct {
	Tool       *models.Tool
	Capability *models.ToolCapability
	Request    models.ToolExecutionRequest
	APIKeys    map[string]string // serviceID -> API key
	StartTime  time.Time
	Context    context.Context
}

// executeToolCapability performs the actual tool execution
func (s *ToolExecutionService) executeToolCapability(ctx *ToolExecutionContext) *models.ToolResult {
	switch ctx.Tool.ID {
	case "web-search":
		return s.executeWebSearch(ctx)
	case "research-assistant":
		return s.executeResearchAssistant(ctx)
	default:
		return &models.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("Tool %s not implemented", ctx.Tool.ID),
			ErrorCode: "TOOL_NOT_IMPLEMENTED",
		}
	}
}

// executeWebSearch executes web search tool
func (s *ToolExecutionService) executeWebSearch(ctx *ToolExecutionContext) *models.ToolResult {
	input := ctx.Request.Input
	
	// Extract parameters
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return &models.ToolResult{
			Success:   false,
			Error:     "query parameter is required",
			ErrorCode: "INVALID_INPUT",
		}
	}

	engine, _ := input["engine"].(string)
	if engine == "" {
		engine = "duckduckgo"
	}

	limit, _ := input["limit"].(float64)
	if limit == 0 {
		limit = 10
	}

	// Execute search based on available API keys and engine preference
	var results interface{}
	var err error
	var engineUsed string

	// Try premium engines first if API keys are available
	if engine == "google" || engine == "auto" {
		if googleKey, hasGoogle := ctx.APIKeys["google-search"]; hasGoogle {
			results, err = s.executeGoogleSearch(query, int(limit), googleKey)
			engineUsed = "google"
		}
	}

	if results == nil && (engine == "tavily" || engine == "auto") {
		if tavilyKey, hasTavily := ctx.APIKeys["tavily"]; hasTavily {
			results, err = s.executeTavilySearch(query, int(limit), tavilyKey)
			engineUsed = "tavily"
		}
	}

	if results == nil && (engine == "serp-api" || engine == "auto") {
		if serpKey, hasSerp := ctx.APIKeys["serp-api"]; hasSerp {
			results, err = s.executeSerpAPISearch(query, int(limit), serpKey)
			engineUsed = "serp-api"
		}
	}

	// Fallback to DuckDuckGo (free)
	if results == nil {
		results, err = s.executeDuckDuckGoSearch(query, int(limit))
		engineUsed = "duckduckgo"
	}

	if err != nil {
		return &models.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("Search failed: %v", err),
			ErrorCode: "SEARCH_FAILED",
		}
	}

	return &models.ToolResult{
		Success: true,
		Data:    results,
		Metadata: map[string]interface{}{
			"engine":      engineUsed,
			"query":       query,
			"limit":       int(limit),
			"resultCount": getResultCount(results),
		},
	}
}

// executeResearchAssistant executes research assistant tool
func (s *ToolExecutionService) executeResearchAssistant(ctx *ToolExecutionContext) *models.ToolResult {
	input := ctx.Request.Input
	
	topic, ok := input["topic"].(string)
	if !ok || topic == "" {
		return &models.ToolResult{
			Success:   false,
			Error:     "topic parameter is required",
			ErrorCode: "INVALID_INPUT",
		}
	}

	depth, _ := input["depth"].(string)
	if depth == "" {
		depth = "detailed"
	}

	// Perform research using available services
	research, err := s.performResearch(ctx, topic, depth)
	if err != nil {
		return &models.ToolResult{
			Success:   false,
			Error:     fmt.Sprintf("Research failed: %v", err),
			ErrorCode: "RESEARCH_FAILED",
		}
	}

	return &models.ToolResult{
		Success: true,
		Data:    research,
		Metadata: map[string]interface{}{
			"topic":         topic,
			"depth":         depth,
			"sourcesUsed":   len(research.Sources),
			"confidence":    research.Confidence,
		},
	}
}

// checkAndRetrieveAPIKeys validates and retrieves required API keys
func (s *ToolExecutionService) checkAndRetrieveAPIKeys(ctx context.Context, tool *models.Tool, userID string) (map[string]string, error) {
	apiKeys := make(map[string]string)
	
	if !tool.HasAPIRequirements() {
		return apiKeys, nil
	}

	for serviceID, requirement := range tool.APIRequirements {
		key, err := s.vaultService.GetAPIKey(ctx, userID, serviceID)
		if err != nil {
			if requirement.Required {
				return nil, fmt.Errorf("required API key for %s not found: %w", serviceID, err)
			}
			// Optional API key not available - continue
			continue
		}
		apiKeys[serviceID] = key
	}

	return apiKeys, nil
}

// saveExecutionToConversation saves tool execution as a message
func (s *ToolExecutionService) saveExecutionToConversation(ctx context.Context, req models.ToolExecutionRequest, result *models.ToolResult) error {

	// Convert to regular message format for storage
	toolUseId := fmt.Sprintf("tool_%d", time.Now().UnixNano())
	messageContent := []models.MessageContent{
		{
			Type: models.ContentTypeToolUse,
			ToolUse: &models.ToolUseContent{
				ToolUseID: toolUseId,
				Name:      req.ToolID,
				Input:     req.Input,
			},
		},
	}

	if result.Success {
		// Convert result data to string format for storage
		var contentStr string
		if data, ok := result.Data.(string); ok {
			contentStr = data
		} else if dataBytes, ok := result.Data.([]byte); ok {
			contentStr = string(dataBytes)
		} else {
			// Try to marshal to JSON
			if jsonData, err := json.Marshal(result.Data); err == nil {
				contentStr = string(jsonData)
			} else {
				contentStr = fmt.Sprintf("%v", result.Data)
			}
		}

		messageContent = append(messageContent, models.MessageContent{
			Type: models.ContentTypeToolResult,
			ToolResult: &models.ToolResultContent{
				ToolUseID: toolUseId,
				Content:   contentStr,
				IsError:   false,
			},
		})
	} else {
		messageContent = append(messageContent, models.MessageContent{
			Type: models.ContentTypeToolResult,
			ToolResult: &models.ToolResultContent{
				ToolUseID: toolUseId,
				Content:   result.Error,
				IsError:   true,
			},
		})
	}

	// Save as assistant message to conversation
	if req.ConversationID != "" {
		message := &models.Message{
			ID:             fmt.Sprintf("msg_%d", time.Now().UnixNano()),
			ConversationID: req.ConversationID,
			Role:           "assistant",
			Content:        messageContent,
			CreatedAt:      time.Now(),
			TokenCount:     0, // Tool executions don't use tokens
		}

		err := s.conversationRepository.AddMessage(ctx, req.ConversationID, req.Context.UserID, message)
		if err != nil {
			log.Printf("❌ Failed to save tool execution message: %v", err)
			return err
		}
		
		log.Printf("✅ Tool execution saved to conversation: %s", req.ConversationID)
	}

	return nil
}

// Helper functions for specific search implementations

func (s *ToolExecutionService) executeGoogleSearch(query string, limit int, apiKey string) (interface{}, error) {
	// TODO: Implement Google Custom Search API
	return nil, fmt.Errorf("Google search not implemented")
}

func (s *ToolExecutionService) executeTavilySearch(query string, limit int, apiKey string) (interface{}, error) {
	// TODO: Implement Tavily Search API
	return nil, fmt.Errorf("Tavily search not implemented")
}

func (s *ToolExecutionService) executeSerpAPISearch(query string, limit int, apiKey string) (interface{}, error) {
	// TODO: Implement SerpAPI
	return nil, fmt.Errorf("SerpAPI search not implemented")
}

func (s *ToolExecutionService) executeDuckDuckGoSearch(query string, limit int) (interface{}, error) {
	// TODO: Implement DuckDuckGo search (free, no API key required)
	// This would use a library or HTTP client to search DuckDuckGo
	return map[string]interface{}{
		"results": []map[string]interface{}{
			{
				"title":   "DuckDuckGo Search Result",
				"url":     "https://example.com",
				"snippet": "This is a placeholder result for: " + query,
			},
		},
		"query": query,
		"engine": "duckduckgo",
	}, nil
}

// ResearchResult represents the result of research assistant
type ResearchResult struct {
	Summary      string                   `json:"summary"`
	KeyFindings  []string                 `json:"keyFindings"`
	Sources      []ResearchSource         `json:"sources"`
	Analysis     string                   `json:"analysis"`
	Confidence   float64                  `json:"confidence"`
	Trends       []string                 `json:"trends,omitempty"`
	Gaps         []string                 `json:"gaps,omitempty"`
	Recommendations []string             `json:"recommendations,omitempty"`
}

type ResearchSource struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Type      string `json:"type"` // "web", "academic", "news"
	Relevance float64 `json:"relevance"`
	Date      string `json:"date,omitempty"`
}

func (s *ToolExecutionService) performResearch(ctx *ToolExecutionContext, topic string, depth string) (*ResearchResult, error) {
	// TODO: Implement comprehensive research functionality
	// This would involve multiple search queries, analysis, and synthesis
	return &ResearchResult{
		Summary: fmt.Sprintf("Research summary for: %s", topic),
		KeyFindings: []string{
			"Key finding 1",
			"Key finding 2",
		},
		Sources: []ResearchSource{
			{
				Title:     "Example Source",
				URL:       "https://example.com",
				Type:      "web",
				Relevance: 0.9,
			},
		},
		Analysis:    fmt.Sprintf("Detailed analysis of %s", topic),
		Confidence:  0.8,
		Trends:      []string{"Trend 1", "Trend 2"},
		Gaps:        []string{"Research gap identified"},
		Recommendations: []string{"Recommendation 1"},
	}, nil
}

// Utility functions

func getServiceIDs(apiKeys map[string]string) []string {
	var serviceIDs []string
	for serviceID := range apiKeys {
		serviceIDs = append(serviceIDs, serviceID)
	}
	return serviceIDs
}

func getResultCount(results interface{}) int {
	if resultsMap, ok := results.(map[string]interface{}); ok {
		if resultsList, ok := resultsMap["results"].([]interface{}); ok {
			return len(resultsList)
		}
	}
	return 0
}