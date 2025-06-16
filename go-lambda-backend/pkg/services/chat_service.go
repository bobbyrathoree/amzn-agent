package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"net/url"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/google/uuid"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

// ChatService handles bot-specific chat interactions
type ChatService struct {
	bedrockClient      *bedrockruntime.Client
	bedrockAgentClient *bedrockagentruntime.Client
	botService         *BotService
	kbService          *KnowledgeBaseService
	guardrailService   *GuardrailService
	conversationRepo   repositories.ConversationRepository
	region             string
}

// NewChatService creates a new ChatService
func NewChatService(awsConfig aws.Config, botService *BotService, kbService *KnowledgeBaseService, conversationRepo repositories.ConversationRepository, region string) *ChatService {
	return &ChatService{
		bedrockClient:      bedrockruntime.NewFromConfig(awsConfig),
		bedrockAgentClient: bedrockagentruntime.NewFromConfig(awsConfig),
		botService:         botService,
		kbService:          kbService,
		guardrailService:   NewGuardrailService(),
		conversationRepo:   conversationRepo,
		region:             region,
	}
}

// ChatRequest represents a chat message request
type ChatRequest struct {
	Message        string            `json:"message"`
	ConversationID string            `json:"conversationId,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
	Context        map[string]string `json:"context,omitempty"`
	SessionModelID string            `json:"sessionModelId,omitempty"` // Override model for this session
}

// ChatResponse represents a chat message response
type ChatResponse struct {
	Response         string                   `json:"response"`
	ConversationID   string                   `json:"conversationId"`
	Sources          []KnowledgeBaseChunk     `json:"sources,omitempty"`
	ToolsUsed        []string                 `json:"toolsUsed,omitempty"`
	GuardrailApplied bool                     `json:"guardrailApplied,omitempty"`
	Metadata         map[string]interface{}   `json:"metadata,omitempty"`
}

// KnowledgeBaseChunk represents a chunk from Knowledge Base retrieval
type KnowledgeBaseChunk struct {
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Source   string  `json:"source"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChatWithBot handles a chat interaction with a specific bot with full conversation persistence
func (s *ChatService) ChatWithBot(ctx context.Context, botID, userID string, userGroups []string, isAdmin bool, req *ChatRequest) (*ChatResponse, error) {
	log.Printf("💬 Processing chat request for bot: %s, conversation: %s", botID, req.ConversationID)

	// Get bot configuration with access control
	bot, err := s.botService.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot: %w", err)
	}

	// Update bot usage
	if err := s.botService.IncrementBotUsage(ctx, botID); err != nil {
		log.Printf("Warning: failed to increment bot usage: %v", err)
	}

	log.Printf("🤖 Using bot: %s with model: %v", bot.Title, bot.ActiveModels)

	// STEP 1: Load conversation context for AI processing
	var conversation *models.Conversation
	var conversationHistory []models.Message
	
	if req.ConversationID != "" {
		// Load existing conversation
		conversation, err = s.conversationRepo.GetConversation(ctx, req.ConversationID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation %s: %w", req.ConversationID, err)
		}
		
		// Extract messages for AI context (sorted by creation time)
		for _, msg := range conversation.MessageMap {
			conversationHistory = append(conversationHistory, *msg)
		}
		
		// Sort messages by creation time for proper context
		for i := 0; i < len(conversationHistory)-1; i++ {
			for j := i + 1; j < len(conversationHistory); j++ {
				if conversationHistory[i].CreatedAt.After(conversationHistory[j].CreatedAt) {
					conversationHistory[i], conversationHistory[j] = conversationHistory[j], conversationHistory[i]
				}
			}
		}
		
		log.Printf("📚 Loaded conversation with %d messages for context", len(conversationHistory))
	}

	// STEP 2: Create and save user message to conversation
	userMessageID := uuid.New().String()
	userMessage := &models.Message{
		ID:             userMessageID,
		ConversationID: req.ConversationID,
		Role:           "user",
		Content: []models.MessageContent{
			{
				Type: models.ContentTypeText,
				Text: req.Message,
			},
		},
		CreatedAt:  time.Now(),
		TokenCount: len(req.Message) / 4, // Rough token estimate
	}

	// Add user message to conversation
	if err := s.conversationRepo.AddMessage(ctx, req.ConversationID, userID, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}
	log.Printf("💾 Saved user message to conversation")

	// STEP 3: Build AI context with conversation history
	systemPrompt := s.buildSystemPrompt(bot)
	
	// Build conversation context for AI
	conversationContext := s.buildConversationContext(conversationHistory)
	
	// Retrieve relevant knowledge if bot has Knowledge Base
	var knowledgeContext string
	var sources []KnowledgeBaseChunk
	if bot.KnowledgeBaseID != nil && *bot.KnowledgeBaseID != "" {
		knowledgeContext, sources, err = s.retrieveKnowledge(ctx, *bot.KnowledgeBaseID, req.Message, bot.KnowledgeBaseConfig)
		if err != nil {
			log.Printf("Warning: Knowledge Base retrieval failed: %v", err)
		}
	}

	// Combine system prompt with knowledge context
	fullPrompt := systemPrompt
	if knowledgeContext != "" {
		fullPrompt += "\n\nRelevant information from knowledge base:\n" + knowledgeContext
		fullPrompt += "\n\nPlease use the above information to help answer the user's question. If the information doesn't directly relate to the question, you may rely on your general knowledge."
	}

	// STEP 4: Execute agent tools if needed
	var toolsUsed []string
	var toolContent []models.MessageContent
	
	if s.shouldUseTool(req.Message, bot.AgentTools) {
		toolResults, usedTools, toolMessages, err := s.executeToolsWithMessages(ctx, req.Message, bot.AgentTools)
		if err != nil {
			log.Printf("Warning: Tool execution failed: %v", err)
		} else {
			if toolResults != "" {
				fullPrompt += "\n\nTool execution results:\n" + toolResults
			}
			toolsUsed = usedTools
			toolContent = toolMessages
		}
	}

	// STEP 5: Select model and call Bedrock with full context
	// Priority: 1) Session model from request 2) Session model from conversation 3) Bot's active models
	var sessionModel *string
	if req.SessionModelID != "" {
		sessionModel = &req.SessionModelID
	} else if conversation != nil && conversation.SessionModelID != nil {
		sessionModel = conversation.SessionModelID
	}
	
	modelID := s.selectModel(bot.ActiveModels, sessionModel)

	// Build the complete prompt with conversation history
	finalPrompt := fullPrompt
	if conversationContext != "" {
		finalPrompt += "\n\nConversation history:\n" + conversationContext
	}

	response, guardrailApplied, err := s.callBedrock(ctx, modelID, finalPrompt, req.Message, bot.GenerationParams, bot, req.Stream)
	if err != nil {
		return nil, fmt.Errorf("failed to call Bedrock: %w", err)
	}

	// STEP 6: Create and save AI response message
	assistantMessageID := uuid.New().String()
	assistantContent := []models.MessageContent{
		{
			Type: models.ContentTypeText,
			Text: response,
		},
	}
	
	// Add tool usage content if tools were used
	assistantContent = append(assistantContent, toolContent...)
	
	assistantMessage := &models.Message{
		ID:             assistantMessageID,  
		ConversationID: req.ConversationID,
		Role:           "assistant",
		Content:        assistantContent,
		Model:          modelID,
		ParentID:       &userMessageID,
		CreatedAt:      time.Now(),
		TokenCount:     len(response) / 4, // Rough token estimate
		Feedback:       &models.MessageFeedback{}, // Initialize empty feedback
	}

	// Save AI response to conversation
	if err := s.conversationRepo.AddMessage(ctx, req.ConversationID, userID, assistantMessage); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}
	log.Printf("💾 Saved AI response to conversation")

	// Build response metadata
	metadata := map[string]interface{}{
		"model":               modelID,
		"botId":               botID,
		"hasKB":               bot.KnowledgeBaseID != nil,
		"toolsCount":          len(toolsUsed),
		"guardrailApplied":    guardrailApplied,
		"userMessageId":       userMessageID,
		"assistantMessageId":  assistantMessageID,
	}
	
	chatResponse := &ChatResponse{
		Response:         response,
		ConversationID:   req.ConversationID,
		Sources:          sources,
		ToolsUsed:        toolsUsed,
		GuardrailApplied: guardrailApplied,
		Metadata:         metadata,
	}

	log.Printf("✅ Chat response generated and saved for bot %s, conversation %s", botID, req.ConversationID)
	return chatResponse, nil
}

// buildSystemPrompt creates the system prompt based on bot configuration
func (s *ChatService) buildSystemPrompt(bot *models.Bot) string {
	prompt := fmt.Sprintf("You are %s. %s\n\n", bot.Title, bot.Description)
	prompt += bot.Instruction

	if bot.DisplayRetrievedChunks && bot.KnowledgeBaseID != nil {
		prompt += "\n\nWhen using information from the knowledge base, please cite your sources and show which documents you referenced."
	}

	return prompt
}

// retrieveKnowledge queries the Knowledge Base for relevant information
func (s *ChatService) retrieveKnowledge(ctx context.Context, kbID, query string, config models.KnowledgeBaseConfig) (string, []KnowledgeBaseChunk, error) {
	log.Printf("🔍 Retrieving knowledge from KB: %s with query: %s", kbID, query)

	// Determine search type based on config
	searchType := types.SearchTypeSemantic
	if config.SearchType == "HYBRID" {
		searchType = types.SearchTypeHybrid
	}

	// Build the retrieval request
	input := &bedrockagentruntime.RetrieveInput{
		KnowledgeBaseId: aws.String(kbID),
		RetrievalQuery: &types.KnowledgeBaseQuery{
			Text: aws.String(query),
		},
		RetrievalConfiguration: &types.KnowledgeBaseRetrievalConfiguration{
			VectorSearchConfiguration: &types.KnowledgeBaseVectorSearchConfiguration{
				NumberOfResults:    aws.Int32(int32(config.MaxResults)),
				OverrideSearchType: searchType,
			},
		},
	}

	// Make the API call to retrieve knowledge
	response, err := s.bedrockAgentClient.Retrieve(ctx, input)
	if err != nil {
		log.Printf("❌ Failed to retrieve from Knowledge Base: %v", err)
		return "", []KnowledgeBaseChunk{}, fmt.Errorf("failed to retrieve from knowledge base: %w", err)
	}

	// Process the results
	var chunks []KnowledgeBaseChunk
	var contextParts []string

	for i, result := range response.RetrievalResults {
		// Extract content
		content := ""
		if result.Content != nil && result.Content.Text != nil {
			content = aws.ToString(result.Content.Text)
		}

		// Extract source information
		sourceName, sourceLink := s.extractSourceFromLocation(result.Location)
		
		// Extract score if available
		score := 0.0
		if result.Score != nil {
			score = float64(*result.Score)
		}

		// Skip results below threshold
		if score > 0 && score < config.ScoreThreshold {
			continue
		}

		// Create chunk
		chunk := KnowledgeBaseChunk{
			Content: content,
			Score:   score,
			Source:  sourceName,
			Metadata: map[string]interface{}{
				"rank":        i + 1,
				"sourceLink":  sourceLink,
				"location":    result.Location,
			},
		}

		chunks = append(chunks, chunk)
		contextParts = append(contextParts, fmt.Sprintf("Source: %s\\nContent: %s", sourceName, content))
	}

	knowledgeContext := strings.Join(contextParts, "\\n\\n")
	
	log.Printf("✅ Retrieved %d knowledge chunks from KB %s", len(chunks), kbID)
	return knowledgeContext, chunks, nil
}

// extractSourceFromLocation extracts source name and link from retrieval result location
func (s *ChatService) extractSourceFromLocation(location *types.RetrievalResultLocation) (string, string) {
	if location == nil {
		return "Unknown Source", ""
	}

	switch location.Type {
	case types.RetrievalResultLocationTypeWeb:
		if location.WebLocation != nil && location.WebLocation.Url != nil {
			urlStr := aws.ToString(location.WebLocation.Url)
			return urlStr, urlStr
		}
	case types.RetrievalResultLocationTypeS3:
		if location.S3Location != nil && location.S3Location.Uri != nil {
			uriStr := aws.ToString(location.S3Location.Uri)
			if parsedURL, err := url.Parse(uriStr); err == nil {
				sourceName := path.Base(parsedURL.Path)
				if sourceName == "" || sourceName == "/" {
					sourceName = "S3 Document"
				}
				return sourceName, uriStr
			}
			return "S3 Document", uriStr
		}
	case types.RetrievalResultLocationTypeConfluence:
		if location.ConfluenceLocation != nil && location.ConfluenceLocation.Url != nil {
			urlStr := aws.ToString(location.ConfluenceLocation.Url)
			return "Confluence Document", urlStr
		}
	case types.RetrievalResultLocationTypeSalesforce:
		if location.SalesforceLocation != nil && location.SalesforceLocation.Url != nil {
			urlStr := aws.ToString(location.SalesforceLocation.Url)
			return "Salesforce Document", urlStr
		}
	case types.RetrievalResultLocationTypeSharepoint:
		if location.SharePointLocation != nil && location.SharePointLocation.Url != nil {
			urlStr := aws.ToString(location.SharePointLocation.Url)
			return "SharePoint Document", urlStr
		}
	}

	return "Unknown Source", ""
}

// shouldUseTool determines if any tools should be executed based on the message
func (s *ChatService) shouldUseTool(message string, tools []models.AgentTool) bool {
	if len(tools) == 0 {
		return false
	}

	message = strings.ToLower(message)
	
	// Simple heuristics for tool usage
	for _, tool := range tools {
		switch tool.Name {
		case "calculator":
			if strings.Contains(message, "calculate") || strings.Contains(message, "math") || 
			   strings.Contains(message, "+") || strings.Contains(message, "-") ||
			   strings.Contains(message, "*") || strings.Contains(message, "/") {
				return true
			}
		case "web_search":
			if strings.Contains(message, "search") || strings.Contains(message, "find") ||
			   strings.Contains(message, "latest") || strings.Contains(message, "current") {
				return true
			}
		}
	}
	
	return false
}

// executeTools executes relevant agent tools
func (s *ChatService) executeTools(ctx context.Context, message string, tools []models.AgentTool) (string, []string, error) {
	var results []string
	var usedTools []string

	// Basic tool execution - would be expanded in a full implementation
	for _, tool := range tools {
		switch tool.Name {
		case "calculator":
			if strings.Contains(strings.ToLower(message), "calculate") || 
			   strings.Contains(message, "+") || strings.Contains(message, "-") {
				result := s.executeCalculator(message)
				if result != "" {
					results = append(results, fmt.Sprintf("Calculator: %s", result))
					usedTools = append(usedTools, "calculator")
				}
			}
		case "web_search":
			// Placeholder for web search - would integrate with search API
			if strings.Contains(strings.ToLower(message), "search") {
				results = append(results, "Web search: [Feature not fully implemented]")
				usedTools = append(usedTools, "web_search")
			}
		}
	}

	return strings.Join(results, "\n"), usedTools, nil
}

// executeCalculator performs basic mathematical calculations
func (s *ChatService) executeCalculator(message string) string {
	// Very basic calculator - would be replaced with proper math parser
	if strings.Contains(message, "2+2") || strings.Contains(message, "2 + 2") {
		return "2 + 2 = 4"
	}
	if strings.Contains(message, "5*5") || strings.Contains(message, "5 * 5") {
		return "5 * 5 = 25"
	}
	// TODO: Implement proper mathematical expression parser
	return "Basic calculation detected (full calculator not implemented)"
}

// buildConversationContext builds a string representation of conversation history for AI context
func (s *ChatService) buildConversationContext(messages []models.Message) string {
	if len(messages) == 0 {
		return ""
	}

	var context []string
	for _, msg := range messages {
		// Skip system and instruction messages for context
		if msg.Role == "system" || msg.Role == "instruction" {
			continue
		}

		// Extract text content from message
		var messageText string
		for _, content := range msg.Content {
			if content.Type == models.ContentTypeText {
				messageText = content.Text
				break
			}
		}

		if messageText != "" {
			roleLabel := msg.Role
			if msg.Role == "assistant" {
				roleLabel = "AI"
			} else if msg.Role == "user" {
				roleLabel = "Human"
			}
			context = append(context, fmt.Sprintf("%s: %s", roleLabel, messageText))
		}
	}

	return strings.Join(context, "\n")
}

// executeToolsWithMessages executes tools and returns both results and message content
func (s *ChatService) executeToolsWithMessages(ctx context.Context, message string, tools []models.AgentTool) (string, []string, []models.MessageContent, error) {
	var results []string
	var usedTools []string
	var messageContent []models.MessageContent

	// Basic tool execution with message content generation
	for _, tool := range tools {
		switch tool.Name {
		case "calculator":
			if strings.Contains(strings.ToLower(message), "calculate") || 
			   strings.Contains(message, "+") || strings.Contains(message, "-") {
				result := s.executeCalculator(message)
				if result != "" {
					results = append(results, fmt.Sprintf("Calculator: %s", result))
					usedTools = append(usedTools, "calculator")
					
					// Add tool use content
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolUse,
						ToolUse: &models.ToolUseContent{
							ToolUseID: uuid.New().String(),
							Name:      "calculator",
							Input:     map[string]interface{}{"expression": message},
						},
					})
					
					// Add tool result content
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolResult,
						ToolResult: &models.ToolResultContent{
							ToolUseID: messageContent[len(messageContent)-1].ToolUse.ToolUseID,
							Content:   result,
							IsError:   false,
						},
					})
				}
			}
		case "web_search":
			// Placeholder for web search - would integrate with search API
			if strings.Contains(strings.ToLower(message), "search") {
				searchResult := "Web search: [Feature not fully implemented]"
				results = append(results, searchResult)
				usedTools = append(usedTools, "web_search")
				
				// Add tool use content
				messageContent = append(messageContent, models.MessageContent{
					Type: models.ContentTypeToolUse,
					ToolUse: &models.ToolUseContent{
						ToolUseID: uuid.New().String(),
						Name:      "web_search",
						Input:     map[string]interface{}{"query": message},
					},
				})
				
				// Add tool result content
				messageContent = append(messageContent, models.MessageContent{
					Type: models.ContentTypeToolResult,
					ToolResult: &models.ToolResultContent{
						ToolUseID: messageContent[len(messageContent)-1].ToolUse.ToolUseID,
						Content:   searchResult,
						IsError:   false,
					},
				})
			}
		}
	}

	return strings.Join(results, "\n"), usedTools, messageContent, nil
}

// selectModel chooses which model to use from the bot's active models
func (s *ChatService) selectModel(activeModels []string, sessionModel *string) string {
	// Priority: 1) Session model override 2) Bot's active models 3) Default fallback
	if sessionModel != nil && *sessionModel != "" {
		log.Printf("🎯 Using session model override: %s", *sessionModel)
		return s.convertToInferenceProfile(*sessionModel)
	}
	
	if len(activeModels) == 0 {
		return "us.anthropic.claude-3-7-sonnet-20250219-v1:0" // Default fallback with inference profile
	}
	
	// Convert old model IDs to inference profiles for backwards compatibility
	selectedModel := activeModels[0]
	return s.convertToInferenceProfile(selectedModel)
}

// convertToInferenceProfile converts old model IDs to inference profile format
func (s *ChatService) convertToInferenceProfile(modelID string) string {
	// If already an inference profile, return as-is
	if strings.Contains(modelID, "us.") || strings.Contains(modelID, "eu.") || strings.Contains(modelID, "apac.") {
		return modelID
	}
	
	// Map old model IDs to inference profiles (using US region for best availability)
	modelMappings := map[string]string{
		// Claude models
		"anthropic.claude-opus-4-20250514-v1:0":        "us.anthropic.claude-opus-4-20250514-v1:0",
		"anthropic.claude-sonnet-4-20250514-v1:0":      "us.anthropic.claude-sonnet-4-20250514-v1:0",
		"anthropic.claude-3-7-sonnet-20250219-v1:0":    "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
		"anthropic.claude-3-5-sonnet-20241022-v2:0":    "us.anthropic.claude-3-5-sonnet-20241022-v2:0",
		"anthropic.claude-3-5-sonnet-20240620-v1:0":    "us.anthropic.claude-3-5-sonnet-20240620-v1:0",
		"anthropic.claude-3-5-haiku-20241022-v1:0":     "us.anthropic.claude-3-5-haiku-20241022-v1:0",
		"anthropic.claude-3-haiku-20240307-v1:0":       "us.anthropic.claude-3-haiku-20240307-v1:0",
		"anthropic.claude-3-opus-20240229-v1:0":        "us.anthropic.claude-3-opus-20240229-v1:0",
		
		// Amazon models  
		"amazon.nova-pro-v1:0":                         "us.amazon.nova-pro-v1:0",
		"amazon.nova-lite-v1:0":                        "us.amazon.nova-lite-v1:0",
		"amazon.nova-micro-v1:0":                       "us.amazon.nova-micro-v1:0",
		"amazon.titan-text-premier-v1:0":               "us.amazon.titan-text-premier-v1:0",
		
		// Mistral models
		"mistral.mistral-large-2407-v1:0":              "us.mistral.mistral-large-2407-v1:0",
		"mistral.mistral-large-2402-v1:0":              "us.mistral.mistral-large-2402-v1:0",
		"mistral.mixtral-8x7b-instruct-v0:1":           "us.mistral.mixtral-8x7b-instruct-v0:1",
		"mistral.mistral-7b-instruct-v0:2":             "us.mistral.mistral-7b-instruct-v0:2",
		
		// DeepSeek models
		"deepseek.r1-v1:0":                             "us.deepseek.r1-v1:0",
		
		// Meta Llama models
		"meta.llama3-3-70b-instruct-v1:0":              "us.meta.llama3-3-70b-instruct-v1:0",
		"meta.llama3-2-90b-instruct-v1:0":              "us.meta.llama3-2-90b-instruct-v1:0",
		"meta.llama3-2-11b-instruct-v1:0":              "us.meta.llama3-2-11b-instruct-v1:0",
		"meta.llama3-2-3b-instruct-v1:0":               "us.meta.llama3-2-3b-instruct-v1:0",
		"meta.llama3-2-1b-instruct-v1:0":               "us.meta.llama3-2-1b-instruct-v1:0",
	}
	
	if inferenceProfile, exists := modelMappings[modelID]; exists {
		log.Printf("🔄 Converting old model ID %s to inference profile %s", modelID, inferenceProfile)
		return inferenceProfile
	}
	
	// If not in mapping, try to auto-convert by adding us. prefix
	if strings.Contains(modelID, "anthropic.") || strings.Contains(modelID, "amazon.") || 
	   strings.Contains(modelID, "mistral.") || strings.Contains(modelID, "deepseek.") || 
	   strings.Contains(modelID, "meta.") {
		converted := "us." + modelID
		log.Printf("🔄 Auto-converting model ID %s to inference profile %s", modelID, converted)
		return converted
	}
	
	// Fallback to original ID if we can't convert
	log.Printf("⚠️ Could not convert model ID %s to inference profile, using as-is", modelID)
	return modelID
}

// Helper functions to detect model families (handles both regular and inference profile formats)
func isAnthropicModel(modelID string) bool {
	return strings.Contains(modelID, "anthropic.claude") || strings.Contains(modelID, ".anthropic.claude")
}

func isMetaLlamaModel(modelID string) bool {
	return strings.Contains(modelID, "meta.llama") || strings.Contains(modelID, ".meta.llama")
}

func isAmazonModel(modelID string) bool {
	return strings.Contains(modelID, "amazon.") || strings.Contains(modelID, ".amazon.")
}

func isMistralModel(modelID string) bool {
	return strings.Contains(modelID, "mistral.") || strings.Contains(modelID, ".mistral.")
}

func isDeepSeekModel(modelID string) bool {
	return strings.Contains(modelID, "deepseek.") || strings.Contains(modelID, ".deepseek.")
}

// callBedrock makes the actual API call to Bedrock with optional guardrail integration
// Following the bedrock-chat pattern of inline guardrails
func (s *ChatService) callBedrock(ctx context.Context, modelID, systemPrompt, userMessage string, params models.GenerationParams, bot *models.Bot, stream bool) (string, bool, error) {
	log.Printf("🚀 Calling Bedrock model: %s", modelID)

	// Build the message for Claude
	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": userMessage,
		},
	}

	// Create the request body based on the model type
	var requestBody map[string]interface{}
	
	if isAnthropicModel(modelID) {
		requestBody = map[string]interface{}{
			"anthropic_version": "bedrock-2023-05-31",
			"system":            systemPrompt,
			"messages":          messages,
			"max_tokens":        params.MaxTokens,
			"temperature":       params.Temperature,
			"top_p":            params.TopP,
			"top_k":            params.TopK,
		}
		
		if len(params.StopSequences) > 0 {
			requestBody["stop_sequences"] = params.StopSequences
		}
	} else if isMetaLlamaModel(modelID) {
		// Llama models use a different format
		requestBody = map[string]interface{}{
			"prompt":            fmt.Sprintf("%s\n\n%s", systemPrompt, userMessage),
			"max_gen_len":       params.MaxTokens,
			"temperature":       params.Temperature,
			"top_p":            params.TopP,
		}
	} else if isAmazonModel(modelID) && strings.Contains(modelID, "titan") {
		// Titan models format
		requestBody = map[string]interface{}{
			"inputText": fmt.Sprintf("%s\n\nHuman: %s\n\nAssistant:", systemPrompt, userMessage),
			"textGenerationConfig": map[string]interface{}{
				"maxTokenCount": params.MaxTokens,
				"temperature":   params.Temperature,
				"topP":          params.TopP,
			},
		}
	} else if isAmazonModel(modelID) && strings.Contains(modelID, "nova") {
		// Amazon Nova models format
		requestBody = map[string]interface{}{
			"messages": messages,
			"inferenceConfig": map[string]interface{}{
				"max_new_tokens": params.MaxTokens,
				"temperature":    params.Temperature,
				"top_p":         params.TopP,
				"top_k":         params.TopK,
			},
		}
		if systemPrompt != "" {
			requestBody["system"] = []map[string]interface{}{
				{"text": systemPrompt},
			}
		}
	} else if isMistralModel(modelID) {
		// Mistral models format
		requestBody = map[string]interface{}{
			"prompt":      fmt.Sprintf("%s\n\nHuman: %s\n\nAssistant:", systemPrompt, userMessage),
			"max_tokens":  params.MaxTokens,
			"temperature": params.Temperature,
			"top_p":       params.TopP,
			"top_k":       params.TopK,
		}
	} else if isDeepSeekModel(modelID) {
		// DeepSeek models format (similar to OpenAI)
		requestBody = map[string]interface{}{
			"messages": append([]map[string]interface{}{
				{"role": "system", "content": systemPrompt},
			}, messages...),
			"max_tokens":  params.MaxTokens,
			"temperature": params.Temperature,
			"top_p":       params.TopP,
		}
	} else {
		// Generic format for other models
		requestBody = map[string]interface{}{
			"prompt":            fmt.Sprintf("%s\n\nHuman: %s\n\nAssistant:", systemPrompt, userMessage),
			"max_tokens_to_sample": params.MaxTokens,
			"temperature":       params.Temperature,
			"top_p":            params.TopP,
			"top_k":            params.TopK,
		}
	}

	// Convert to JSON
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return "", false, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Build the Bedrock API call with optional guardrail configuration
	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		Body:        requestBodyBytes,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}
	
	// Build and add guardrail configuration using GuardrailService (following bedrock-chat pattern)
	guardrailApplied := false
	guardrailConfig, err := s.guardrailService.BuildGuardrailConfig(bot, stream)
	if err != nil {
		log.Printf("Warning: Failed to build guardrail config: %v", err)
	} else if guardrailConfig != nil {
		input.GuardrailIdentifier = guardrailConfig.GuardrailIdentifier
		input.GuardrailVersion = guardrailConfig.GuardrailVersion
		guardrailApplied = true
		log.Printf("🛡️ Using inline guardrails for model call: %s", *guardrailConfig.GuardrailIdentifier)
	}

	// Make the actual API call
	response, err := s.bedrockClient.InvokeModel(ctx, input)
	if err != nil {
		log.Printf("❌ Bedrock API call failed: %v", err)
		return "", false, fmt.Errorf("failed to invoke Bedrock model: %w", err)
	}

	// Parse the response based on model type
	var result string
	var responseData map[string]interface{}
	if err := json.Unmarshal(response.Body, &responseData); err != nil {
			return "", false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract the generated text based on model type
	if isAnthropicModel(modelID) {
		// Claude response format
		if content, ok := responseData["content"].([]interface{}); ok && len(content) > 0 {
			if textBlock, ok := content[0].(map[string]interface{}); ok {
				if text, ok := textBlock["text"].(string); ok {
					result = text
				}
			}
		}
	} else if isMetaLlamaModel(modelID) {
		// Llama response format
		if generation, ok := responseData["generation"].(string); ok {
			result = generation
		}
	} else if isAmazonModel(modelID) && strings.Contains(modelID, "titan") {
		// Titan response format
		if results, ok := responseData["results"].([]interface{}); ok && len(results) > 0 {
			if resultBlock, ok := results[0].(map[string]interface{}); ok {
				if outputText, ok := resultBlock["outputText"].(string); ok {
					result = outputText
				}
			}
		}
	} else if isAmazonModel(modelID) && strings.Contains(modelID, "nova") {
		// Amazon Nova response format
		if output, ok := responseData["output"].(map[string]interface{}); ok {
			if message, ok := output["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].([]interface{}); ok && len(content) > 0 {
					if textBlock, ok := content[0].(map[string]interface{}); ok {
						if text, ok := textBlock["text"].(string); ok {
							result = text
						}
					}
				}
			}
		}
	} else if isMistralModel(modelID) {
		// Mistral response format
		if outputs, ok := responseData["outputs"].([]interface{}); ok && len(outputs) > 0 {
			if output, ok := outputs[0].(map[string]interface{}); ok {
				if text, ok := output["text"].(string); ok {
					result = text
				}
			}
		}
	} else if isDeepSeekModel(modelID) {
		// DeepSeek response format
		if choices, ok := responseData["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if content, ok := message["content"].(string); ok {
						result = content
					}
				}
			}
		}
	} else {
		// Try common response formats
		if completion, ok := responseData["completion"].(string); ok {
			result = completion
		} else if completions, ok := responseData["completions"].([]interface{}); ok && len(completions) > 0 {
			if comp, ok := completions[0].(map[string]interface{}); ok {
				if data, ok := comp["data"].(map[string]interface{}); ok {
					if text, ok := data["text"].(string); ok {
						result = text
					}
				}
			}
		}
	}

	if result == "" {
			return "", false, fmt.Errorf("unable to extract response text from model output")
	}

	log.Printf("✅ Bedrock API call successful, response length: %d characters", len(result))
	return result, guardrailApplied, nil
}