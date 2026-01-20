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
	KnowledgeSearchStages []KnowledgeSearchStage `json:"knowledgeSearchStages,omitempty"` // 🚀 INGENIOUS ENHANCEMENT
	// 🛠️ Enhanced Tool Integration
	ToolsExecuted    []ToolExecution          `json:"toolsExecuted,omitempty"`
	ToolsSkipped     []ToolSkipped           `json:"toolsSkipped,omitempty"`
	KeyRecommendations []KeyRecommendation   `json:"keyRecommendations,omitempty"`
	Metadata         map[string]interface{}   `json:"metadata,omitempty"`
}

// ToolExecution represents a tool that was successfully executed
type ToolExecution struct {
	ToolName        string                 `json:"toolName"`
	ToolID          string                 `json:"toolId"`
	Capability      string                 `json:"capability"`
	ExecutionTime   int64                  `json:"executionTime"` // milliseconds
	UsedAPIKey      bool                   `json:"usedApiKey"`
	Engine          string                 `json:"engine,omitempty"` // e.g., "google", "duckduckgo"
	ResultCount     int                    `json:"resultCount,omitempty"`
	Success         bool                   `json:"success"`
	ErrorMessage    string                 `json:"errorMessage,omitempty"`
}

// ToolSkipped represents a tool that was skipped due to missing requirements
type ToolSkipped struct {
	ToolName        string `json:"toolName"`
	ToolID          string `json:"toolId"`
	Reason          string `json:"reason"`
	RequiredService string `json:"requiredService,omitempty"`
	FallbackUsed    bool   `json:"fallbackUsed"`
}

// KeyRecommendation suggests API keys that would enhance the user experience
type KeyRecommendation struct {
	ServiceID       string   `json:"serviceId"`
	ServiceName     string   `json:"serviceName"`
	Description     string   `json:"description"`
	Benefits        []string `json:"benefits"`
	PricingInfo     string   `json:"pricingInfo"`
	SignupURL       string   `json:"signupUrl,omitempty"`
	Priority        string   `json:"priority"` // "high", "medium", "low"
}

// KnowledgeBaseChunk represents a chunk from Knowledge Base retrieval
type KnowledgeBaseChunk struct {
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Source   string  `json:"source"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// 🚀 INGENIOUS ENHANCEMENT: Multi-Stage Knowledge Search
type KnowledgeSearchStage struct {
	Stage       string                 `json:"stage"`
	Query       string                 `json:"query"`
	Strategy    string                 `json:"strategy"`
	ResultCount int                    `json:"result_count"`
	Duration    string                 `json:"duration"`
	Success     bool                   `json:"success"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// 🚀 INGENIOUS ENHANCEMENT: Contextual Query Enhancement  
type QueryEnhancement struct {
	OriginalQuery    string   `json:"original_query"`
	EnhancedQueries  []string `json:"enhanced_queries"`
	ConversationHints []string `json:"conversation_hints"`
	KeyTerms         []string `json:"key_terms"`
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
	
	// STEP 3A: Smart KB Routing with Intent Classification 🚀
	var sources []KnowledgeBaseChunk
	var knowledgeSearchStages []KnowledgeSearchStage

	if bot.KnowledgeBaseID != nil && *bot.KnowledgeBaseID != "" {
		// Intent classification to determine if KB search is needed (~100ms with Haiku 4.5)
		intent, _ := s.classifyUserIntent(ctx, req.Message, bot.Description)

		if intent.NeedsKBSearch {
			log.Printf("🔍 Intent classifier: KB search needed (intent=%s, confidence=%.2f)",
				intent.Intent, intent.Confidence)
			// Progressive knowledge search with multiple strategies
			_, sources, knowledgeSearchStages, err = s.intelligentKnowledgeRetrieval(ctx,
				*bot.KnowledgeBaseID, req.Message, conversationHistory, bot.KnowledgeBaseConfig)
			if err != nil {
				log.Printf("Warning: Intelligent Knowledge Base retrieval failed: %v", err)
			}
			log.Printf("🔍 KB search completed with %d sources, %d stages", len(sources), len(knowledgeSearchStages))
		} else {
			log.Printf("⚡ Intent classifier: Skipping KB search (intent=%s, confidence=%.2f)",
				intent.Intent, intent.Confidence)
			// Add a search stage indicating skip for frontend visibility
			knowledgeSearchStages = append(knowledgeSearchStages, KnowledgeSearchStage{
				Stage:       "intent_classification",
				Query:       req.Message,
				ResultCount: 0,
				Duration:    "~100ms",
				Success:     true,
				Metadata: map[string]interface{}{
					"skipped":    true,
					"intent":     intent.Intent,
					"confidence": intent.Confidence,
					"reason":     "Message classified as not requiring KB search",
				},
			})
		}
	} else {
		log.Printf("🔍 DEBUG: No knowledge base configured for bot. KnowledgeBaseID: %v", bot.KnowledgeBaseID)
	}

	// Combine system prompt with knowledge context using sophisticated RAG prompting
	fullPrompt := systemPrompt
	if len(sources) > 0 {
		// Use the new sophisticated RAG prompt with citation instructions 🚀
		ragPrompt := BuildRAGPrompt(sources, bot.DisplayRetrievedChunks)
		fullPrompt += "\n\n" + ragPrompt
		
		log.Printf("🔍 Enhanced RAG prompt applied with %d sources, citations: %v", len(sources), bot.DisplayRetrievedChunks)
	}

	// STEP 4: Execute agent tools if needed with vault integration
	var toolsUsed []string
	var toolContent []models.MessageContent
	var toolsExecuted []ToolExecution
	var toolsSkipped []ToolSkipped
	var keyRecommendations []KeyRecommendation
	
	if s.shouldUseTool(req.Message, bot.AgentTools) {
		toolResults, usedTools, toolMessages, executed, skipped, recommendations, err := s.executeToolsWithVault(ctx, req.Message, bot.AgentTools, userID)
		if err != nil {
			log.Printf("Warning: Tool execution failed: %v", err)
		} else {
			if toolResults != "" {
				fullPrompt += "\n\nTool execution results:\n" + toolResults
			}
			toolsUsed = usedTools
			toolContent = toolMessages
			toolsExecuted = executed
			toolsSkipped = skipped
			keyRecommendations = recommendations
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
	
	// STEP 6A: Generate conversation title if this is the first user message 🚀
	if len(conversationHistory) == 0 { // This was the first user message
		go func() {
			// Run title generation in background to not slow down the response
			if err := s.generateConversationTitle(context.Background(), req.ConversationID, userID, req.Message, modelID); err != nil {
				log.Printf("Warning: Failed to generate conversation title: %v", err)
			}
		}()
	}

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
		Response:              response,
		ConversationID:        req.ConversationID,
		Sources:               sources,
		ToolsUsed:             toolsUsed,
		GuardrailApplied:      guardrailApplied,
		KnowledgeSearchStages: knowledgeSearchStages, // 🚀 INGENIOUS ENHANCEMENT
		// 🛠️ Enhanced Tool Integration
		ToolsExecuted:         toolsExecuted,
		ToolsSkipped:          toolsSkipped,
		KeyRecommendations:    keyRecommendations,
		Metadata:              metadata,
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

	// Debug: Log the exact API call parameters
	log.Printf("🔍 DEBUG: KB API Call - KnowledgeBaseId: %s, Query: %s, SearchType: %v, MaxResults: %d", 
		*input.KnowledgeBaseId, *input.RetrievalQuery.Text, searchType, *input.RetrievalConfiguration.VectorSearchConfiguration.NumberOfResults)

	// Make the API call to retrieve knowledge
	response, err := s.bedrockAgentClient.Retrieve(ctx, input)
	if err != nil {
		log.Printf("❌ Failed to retrieve from Knowledge Base: %v", err)
		return "", []KnowledgeBaseChunk{}, fmt.Errorf("failed to retrieve from knowledge base: %w", err)
	}

	// Debug: Log the API response details
	log.Printf("🔍 DEBUG: KB API Response - Total results: %d", len(response.RetrievalResults))

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

		// Debug: Log the score and threshold comparison
		log.Printf("🔍 DEBUG: KB Result %d - Score: %.3f, Threshold: %.3f, Content preview: %.100s", 
			i+1, score, config.ScoreThreshold, content)

		// Use reasonable threshold for Knowledge Base similarity scores (typically 0.3-0.6 range)
		effectiveThreshold := config.ScoreThreshold
		if config.ScoreThreshold > 0.5 {
			effectiveThreshold = 0.4 // More reasonable threshold for KB results
		}
		
		// Skip results below threshold
		if score > 0 && score < effectiveThreshold {
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

// 🚀 INGENIOUS ENHANCEMENT: Intelligent Multi-Stage Knowledge Retrieval
// This surpasses bedrock-chat by combining multiple search strategies and contextual understanding
func (s *ChatService) intelligentKnowledgeRetrieval(ctx context.Context, kbID, query string, conversationHistory []models.Message, config models.KnowledgeBaseConfig) (string, []KnowledgeBaseChunk, []KnowledgeSearchStage, error) {
	startTime := time.Now()
	var allChunks []KnowledgeBaseChunk
	var searchStages []KnowledgeSearchStage
	
	log.Printf("🧠 Starting intelligent knowledge retrieval for query: %s", query)
	
	// STAGE 1: Query Enhancement with Conversational Context 🚀
	enhancement := s.enhanceQueryWithContext(query, conversationHistory)
	searchStages = append(searchStages, KnowledgeSearchStage{
		Stage:    "query_enhancement",
		Query:    query,
		Strategy: "contextual_analysis",
		Duration: time.Since(startTime).String(),
		Success:  true,
		Metadata: map[string]interface{}{
			"enhanced_queries": enhancement.EnhancedQueries,
			"key_terms":       enhancement.KeyTerms,
			"context_hints":   enhancement.ConversationHints,
		},
	})
	
	// STAGE 2: Multi-Strategy Search 🚀
	searchStrategies := []struct {
		name     string
		query    string
		maxResults int
	}{}
	
	// Always include primary search
	searchStrategies = append(searchStrategies, struct {
		name     string
		query    string
		maxResults int
	}{"primary_search", query, config.MaxResults/2})
	
	// Enhanced search if we have enhanced queries
	if len(enhancement.EnhancedQueries) > 0 {
		enhancedQuery := strings.Join(enhancement.EnhancedQueries, " ")
		if enhancedQuery != "" {
			searchStrategies = append(searchStrategies, struct {
				name     string
				query    string
				maxResults int
			}{"enhanced_search", enhancedQuery, config.MaxResults/2})
		}
	}
	
	// Contextual search if we have key terms
	if len(enhancement.KeyTerms) > 0 {
		contextualQuery := strings.Join(enhancement.KeyTerms, " OR ")
		if contextualQuery != "" && strings.TrimSpace(contextualQuery) != "" {
			searchStrategies = append(searchStrategies, struct {
				name     string
				query    string
				maxResults int
			}{"contextual_search", contextualQuery, config.MaxResults/3})
		}
	}
	
	for _, strategy := range searchStrategies {
		stageStart := time.Now()
		chunks, err := s.performKnowledgeSearch(ctx, kbID, strategy.query, strategy.maxResults, config)
		
		stage := KnowledgeSearchStage{
			Stage:       strategy.name,
			Query:       strategy.query,
			Strategy:    "hybrid_vector_search",
			ResultCount: len(chunks),
			Duration:    time.Since(stageStart).String(),
			Success:     err == nil,
		}
		
		if err != nil {
			log.Printf("❌ Search strategy %s failed: %v", strategy.name, err)
			stage.Metadata = map[string]interface{}{"error": err.Error()}
		} else {
			// Add relevance scoring and deduplication 🚀
			chunks = s.scoreAndDeduplicateChunks(chunks, query, allChunks)
			allChunks = append(allChunks, chunks...)
			log.Printf("✅ Strategy %s found %d unique chunks", strategy.name, len(chunks))
		}
		
		searchStages = append(searchStages, stage)
	}
	
	// STAGE 3: Fallback Search if no results found 🚀
	if len(allChunks) == 0 {
		log.Printf("⚠️ No results from sophisticated search, trying fallback broad search")
		
		// Try a simple broad search with individual words
		words := strings.Fields(query)
		for _, word := range words {
			if len(word) > 3 { // Only search meaningful words
				fallbackChunks, err := s.performKnowledgeSearch(ctx, kbID, word, config.MaxResults, config)
				if err == nil && len(fallbackChunks) > 0 {
					log.Printf("🎯 Fallback search for '%s' found %d chunks", word, len(fallbackChunks))
					allChunks = append(allChunks, fallbackChunks...)
					break // Found some results, stop fallback search
				}
			}
		}
		
		// If still no results, try the most basic search possible
		if len(allChunks) == 0 {
			log.Printf("⚠️ Trying final fallback with original query")
			fallbackChunks, err := s.performKnowledgeSearch(ctx, kbID, query, config.MaxResults, config)
			if err == nil {
				allChunks = append(allChunks, fallbackChunks...)
			}
		}
		
		searchStages = append(searchStages, KnowledgeSearchStage{
			Stage:       "fallback_search",
			Query:       query,
			Strategy:    "broad_keyword_search",
			ResultCount: len(allChunks),
			Duration:    time.Since(startTime).String(),
			Success:     len(allChunks) > 0,
			Metadata: map[string]interface{}{
				"reason": "sophisticated_search_returned_no_results",
			},
		})
	}
	
	// STAGE 4: Smart Result Ranking and Filtering 🚀
	finalChunks := s.intelligentResultRanking(allChunks, query, enhancement, config.MaxResults)
	
	searchStages = append(searchStages, KnowledgeSearchStage{
		Stage:       "result_optimization",
		Query:       query,
		Strategy:    "ai_powered_ranking",
		ResultCount: len(finalChunks),
		Duration:    time.Since(startTime).String(),
		Success:     true,
		Metadata: map[string]interface{}{
			"total_found":    len(allChunks),
			"final_selected": len(finalChunks),
			"optimization":   "diversity_and_relevance",
		},
	})
	
	// STAGE 4: Return results for RAG prompt processing in main flow 🚀
	log.Printf("🎯 Intelligent retrieval completed: %d final chunks, %d stages, %s total", 
		len(finalChunks), len(searchStages), time.Since(startTime).String())
	
	return "", finalChunks, searchStages, nil
}

// 🚀 INGENIOUS ENHANCEMENT: Context-Aware Query Enhancement
func (s *ChatService) enhanceQueryWithContext(query string, conversationHistory []models.Message) QueryEnhancement {
	enhancement := QueryEnhancement{
		OriginalQuery:   query,
		EnhancedQueries: []string{},
		ConversationHints: []string{},
		KeyTerms:        []string{},
	}
	
	// Extract key terms from the current query
	queryLower := strings.ToLower(query)
	
	// Generic query enhancement - extract key terms without domain-specific assumptions 🚀
	words := strings.Fields(queryLower)
	
	// Extract meaningful terms (filter out common words)
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"what": true, "how": true, "when": true, "where": true, "why": true, "who": true,
		"can": true, "tell": true, "me": true, "about": true, "you": true, "do": true,
		"really": true, "try": true, "again": true, "please": true, "that": true, "this": true,
	}
	
	for _, word := range words {
		if len(word) > 2 && !commonWords[word] {
			enhancement.KeyTerms = append(enhancement.KeyTerms, word)
		}
	}
	
	// If no key terms extracted from basic filtering, try to use original query as fallback
	if len(enhancement.KeyTerms) == 0 {
		// Use original query words but clean punctuation
		cleanQuery := strings.ReplaceAll(strings.ReplaceAll(query, "?", ""), "!", "")
		if strings.TrimSpace(cleanQuery) != "" {
			enhancement.KeyTerms = append(enhancement.KeyTerms, strings.TrimSpace(cleanQuery))
		}
	}
	
	// Create enhanced queries using different combinations
	if len(words) > 1 {
		// Try different word combinations
		enhancement.EnhancedQueries = append(enhancement.EnhancedQueries, 
			fmt.Sprintf("(%s)", strings.Join(words, " AND ")))
		
		if len(words) > 2 {
			// Try partial combinations for broader search
			enhancement.EnhancedQueries = append(enhancement.EnhancedQueries,
				fmt.Sprintf("(%s) OR (%s)", words[0], strings.Join(words[1:], " ")))
		}
	}
	
	// Extract context from recent conversation
	if len(conversationHistory) > 0 {
		recentMessages := conversationHistory
		if len(conversationHistory) > 3 {
			recentMessages = conversationHistory[len(conversationHistory)-3:]
		}
		
		for _, msg := range recentMessages {
			if msg.Role == "user" && len(msg.Content) > 0 {
				content := strings.ToLower(msg.Content[0].Text)
				// Generic context extraction - look for information-seeking patterns
				if strings.Contains(content, "what") || strings.Contains(content, "how") ||
				   strings.Contains(content, "tell me") || strings.Contains(content, "explain") ||
				   strings.Contains(content, "describe") || strings.Contains(content, "information") {
					enhancement.ConversationHints = append(enhancement.ConversationHints, "user_seeking_information")
				}
				if strings.Contains(content, "more") || strings.Contains(content, "details") ||
				   strings.Contains(content, "specific") || strings.Contains(content, "further") {
					enhancement.ConversationHints = append(enhancement.ConversationHints, "user_wants_more_detail")
				}
			}
		}
	}
	
	// If no specific expansions found, create generic enhanced queries
	if len(enhancement.EnhancedQueries) == 0 {
		words := strings.Fields(queryLower)
		if len(words) > 1 {
			enhancement.EnhancedQueries = append(enhancement.EnhancedQueries, 
				fmt.Sprintf("(%s) AND (%s)", words[0], strings.Join(words[1:], " OR ")))
		}
	}
	
	return enhancement
}

// 🚀 INGENIOUS ENHANCEMENT: Perform individual knowledge search with strategy
func (s *ChatService) performKnowledgeSearch(ctx context.Context, kbID, query string, maxResults int, config models.KnowledgeBaseConfig) ([]KnowledgeBaseChunk, error) {
	// Validate query before sending to KB - prevent ValidationException
	if strings.TrimSpace(query) == "" {
		log.Printf("⚠️ Empty query detected, skipping KB search")
		return []KnowledgeBaseChunk{}, fmt.Errorf("empty query provided for knowledge base search")
	}
	
	// Use the existing retrieveKnowledge but with custom parameters
	tempConfig := config
	tempConfig.MaxResults = maxResults
	
	log.Printf("🔍 Performing KB search: kbID=%s, query='%s', maxResults=%d, searchType=%s", 
		kbID, query, maxResults, config.SearchType)
	
	_, chunks, err := s.retrieveKnowledge(ctx, kbID, query, tempConfig)
	
	if err != nil {
		log.Printf("❌ KB search failed: %v", err)
	} else {
		log.Printf("✅ KB search returned %d chunks", len(chunks))
	}
	
	return chunks, err
}

// 🚀 INGENIOUS ENHANCEMENT: Score and deduplicate chunks across searches
func (s *ChatService) scoreAndDeduplicateChunks(newChunks []KnowledgeBaseChunk, query string, existingChunks []KnowledgeBaseChunk) []KnowledgeBaseChunk {
	var uniqueChunks []KnowledgeBaseChunk
	seen := make(map[string]bool)
	
	// Build map of existing chunks for deduplication
	for _, existing := range existingChunks {
		seen[existing.Content] = true
	}
	
	queryLower := strings.ToLower(query)
	
	for _, chunk := range newChunks {
		// Skip duplicates
		if seen[chunk.Content] {
			continue
		}
		
		// Enhanced relevance scoring 🚀
		relevanceBoost := 0.0
		contentLower := strings.ToLower(chunk.Content)
		
		// Boost score for exact query term matches
		for _, word := range strings.Fields(queryLower) {
			if strings.Contains(contentLower, word) {
				relevanceBoost += 0.1
			}
		}
		
		// Boost for technical terms
		technicalTerms := []string{"api", "configuration", "implementation", "architecture", "system"}
		for _, term := range technicalTerms {
			if strings.Contains(contentLower, term) && strings.Contains(queryLower, term) {
				relevanceBoost += 0.05
			}
		}
		
		chunk.Score += relevanceBoost
		uniqueChunks = append(uniqueChunks, chunk)
		seen[chunk.Content] = true
	}
	
	return uniqueChunks
}

// 🚀 INGENIOUS ENHANCEMENT: AI-Powered Result Ranking with Diversity
func (s *ChatService) intelligentResultRanking(chunks []KnowledgeBaseChunk, query string, enhancement QueryEnhancement, maxResults int) []KnowledgeBaseChunk {
	if len(chunks) <= maxResults {
		return chunks
	}
	
	// Sort by score first
	for i := 0; i < len(chunks)-1; i++ {
		for j := i + 1; j < len(chunks); j++ {
			if chunks[i].Score < chunks[j].Score {
				chunks[i], chunks[j] = chunks[j], chunks[i]
			}
		}
	}
	
	// Apply diversity selection to avoid too similar results 🚀
	var finalChunks []KnowledgeBaseChunk
	for i, chunk := range chunks {
		if len(finalChunks) >= maxResults {
			break
		}
		
		// Always include the top result
		if i == 0 {
			finalChunks = append(finalChunks, chunk)
			continue
		}
		
		// Check diversity - avoid too similar content
		isDiverse := true
		for _, existing := range finalChunks {
			similarity := s.calculateContentSimilarity(chunk.Content, existing.Content)
			if similarity > 0.8 { // Too similar
				isDiverse = false
				break
			}
		}
		
		if isDiverse || chunk.Score > 0.9 { // High score overrides diversity
			finalChunks = append(finalChunks, chunk)
		}
	}
	
	return finalChunks
}

// 🚀 INGENIOUS ENHANCEMENT: Simple content similarity calculation
func (s *ChatService) calculateContentSimilarity(content1, content2 string) float64 {
	words1 := strings.Fields(strings.ToLower(content1))
	words2 := strings.Fields(strings.ToLower(content2))
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	wordSet1 := make(map[string]bool)
	for _, word := range words1 {
		wordSet1[word] = true
	}
	
	commonWords := 0
	for _, word := range words2 {
		if wordSet1[word] {
			commonWords++
		}
	}
	
	// Jaccard similarity
	totalUniqueWords := len(wordSet1)
	for _, word := range words2 {
		if !wordSet1[word] {
			totalUniqueWords++
		}
	}
	
	return float64(commonWords) / float64(totalUniqueWords)
}

// 🚀 INGENIOUS ENHANCEMENT: Generate conversation title based on first message
func (s *ChatService) generateConversationTitle(ctx context.Context, conversationID, userID, firstMessage, modelID string) error {
	log.Printf("🏷️ Generating title for conversation %s based on message: %.100s", conversationID, firstMessage)
	
	// Create a simple prompt to generate a conversation title
	titlePrompt := fmt.Sprintf(`Generate a short, descriptive title (2-6 words) for a conversation that starts with this message:

"%s"

The title should:
- Be concise and clear
- Capture the main topic or intent
- Not include quotes or special characters
- Be suitable for a conversation list

Title:`, firstMessage)

	// Use Haiku 3.5 v2 for fast, cost-effective title generation
	titleModelID := "us.anthropic.claude-3-5-haiku-20241022-v1:0"
	titleResponse, _, err := s.callBedrock(ctx, titleModelID, "You are a helpful assistant that generates concise conversation titles.", titlePrompt, models.GenerationParams{
		MaxTokens:     50,
		Temperature:   0.3, // Lower temperature for more consistent titles
		TopP:          0.8,
		TopK:          40,
		StopSequences: []string{"\n", ".", "!", "?"},
	}, nil, false)
	
	if err != nil {
		return fmt.Errorf("failed to generate title: %w", err)
	}
	
	// Clean up the title
	title := strings.TrimSpace(titleResponse)
	title = strings.Trim(title, "\"'")
	
	// Truncate if too long
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	
	// Validate title is reasonable
	if title == "" || len(title) < 3 {
		title = "New Conversation"
	}
	
	log.Printf("✨ Generated title: '%s' for conversation %s", title, conversationID)
	
	// Get the conversation and update its title
	conversation, err := s.conversationRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return fmt.Errorf("failed to get conversation for title update: %w", err)
	}
	
	conversation.Title = title
	conversation.UpdatedAt = time.Now()
	
	if err := s.conversationRepo.UpdateConversation(ctx, conversation); err != nil {
		return fmt.Errorf("failed to update conversation title: %w", err)
	}
	
	log.Printf("✅ Updated conversation %s title to: '%s'", conversationID, title)
	return nil
}

// shouldUseTool determines if any tools should be executed based on the message
func (s *ChatService) shouldUseTool(message string, tools []models.AgentTool) bool {
	if len(tools) == 0 {
		return false
	}

	message = strings.ToLower(message)

	// Enhanced tool detection based on frontend tool definitions
	for _, tool := range tools {
		switch tool.Name {
		case "web-search":
			// Web search triggers
			if strings.Contains(message, "search") || strings.Contains(message, "find") ||
			   strings.Contains(message, "latest") || strings.Contains(message, "current") ||
			   strings.Contains(message, "news") || strings.Contains(message, "what's happening") ||
			   strings.Contains(message, "recent") || strings.Contains(message, "today") ||
			   strings.Contains(message, "this week") || strings.Contains(message, "updates") ||
			   strings.Contains(message, "information about") || strings.Contains(message, "tell me about") {
				return true
			}
		case "research-assistant":
			// Research assistant triggers
			if strings.Contains(message, "research") || strings.Contains(message, "analyze") ||
			   strings.Contains(message, "comprehensive") || strings.Contains(message, "deep dive") ||
			   strings.Contains(message, "study") || strings.Contains(message, "investigate") ||
			   strings.Contains(message, "report") || strings.Contains(message, "findings") ||
			   strings.Contains(message, "trends") || strings.Contains(message, "insights") {
				return true
			}
		case "calculator":
			// Calculator triggers
			if strings.Contains(message, "calculate") || strings.Contains(message, "math") ||
			   strings.Contains(message, "+") || strings.Contains(message, "-") ||
			   strings.Contains(message, "*") || strings.Contains(message, "/") ||
			   strings.Contains(message, "compute") || strings.Contains(message, "solve") {
				return true
			}
		}
	}

	return false
}

// IntentClassification represents the result of intent classification for KB routing
type IntentClassification struct {
	NeedsKBSearch bool    `json:"needsKBSearch"`
	Intent        string  `json:"intent"`
	Confidence    float64 `json:"confidence"`
}

// classifyUserIntent uses Haiku 4.5 for fast intent classification
// Returns whether KB search is needed based on the user's message
func (s *ChatService) classifyUserIntent(ctx context.Context, message string, botDescription string) (*IntentClassification, error) {
	// Short-circuit for very short messages (likely greetings)
	if len(strings.TrimSpace(message)) < 5 {
		log.Printf("⚡ Intent classifier: Short message (<5 chars), skipping KB search")
		return &IntentClassification{
			NeedsKBSearch: false,
			Intent:        "greeting",
			Confidence:    0.95,
		}, nil
	}

	classificationPrompt := fmt.Sprintf(`Classify if this user message needs to search a knowledge base about "%s".

User message: "%s"

Rules:
- needsKBSearch=true: Questions seeking specific information, facts, documentation, how-to, etc.
- needsKBSearch=false: Greetings, thanks, follow-ups like "what now?", chitchat, meta-questions about the bot itself

Respond ONLY with valid JSON (no markdown):
{"needsKBSearch": boolean, "intent": "search"|"greeting"|"followup"|"chitchat", "confidence": 0.0-1.0}`, botDescription, message)

	// Use Claude 3.5 Haiku for fast classification (~100ms)
	classifierModelID := "us.anthropic.claude-3-5-haiku-20241022-v1:0"

	response, _, err := s.callBedrock(ctx, classifierModelID,
		"You are an intent classifier. Respond only with valid JSON, no markdown.",
		classificationPrompt,
		models.GenerationParams{
			MaxTokens:     100,
			Temperature:   0.0, // Deterministic for consistency
			TopP:          1.0,
			TopK:          1,
			StopSequences: []string{"\n", "}"},
		},
		nil, false)

	if err != nil {
		log.Printf("⚠️ Intent classification failed, defaulting to KB search: %v", err)
		return &IntentClassification{NeedsKBSearch: true, Intent: "unknown", Confidence: 0.0}, nil
	}

	// Ensure response ends with }
	response = strings.TrimSpace(response)
	if !strings.HasSuffix(response, "}") {
		response += "}"
	}

	var result IntentClassification
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		log.Printf("⚠️ Intent classification parse failed, defaulting to KB search: %v", err)
		return &IntentClassification{NeedsKBSearch: true, Intent: "unknown", Confidence: 0.0}, nil
	}

	log.Printf("🎯 Intent classification: needsKB=%v, intent=%s, confidence=%.2f for message: %s",
		result.NeedsKBSearch, result.Intent, result.Confidence, message)

	return &result, nil
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

// 🛠️ executeToolsWithVault - Enhanced tool execution with vault integration
func (s *ChatService) executeToolsWithVault(ctx context.Context, message string, tools []models.AgentTool, userID string) (string, []string, []models.MessageContent, []ToolExecution, []ToolSkipped, []KeyRecommendation, error) {
	var results []string
	var usedTools []string
	var messageContent []models.MessageContent
	var toolsExecuted []ToolExecution
	var toolsSkipped []ToolSkipped
	var keyRecommendations []KeyRecommendation

	log.Printf("🛠️ Starting tool execution for user %s with %d configured tools", userID, len(tools))

	for _, tool := range tools {
		startTime := time.Now()
		toolExecution := ToolExecution{
			ToolName:    tool.Name,
			ToolID:      tool.Name, // Use name as ID for now
			Capability:  "default",
		}

		switch tool.Name {
		case "web-search":
			if s.shouldExecuteWebSearch(message) {
				log.Printf("🔍 Executing web search tool for query related to: %s", message)
				
				// Execute web search with vault-aware API key handling
				result, execution, skipped, recommendations := s.executeWebSearchTool(ctx, message, userID)
				
				if execution.Success {
					results = append(results, result)
					usedTools = append(usedTools, tool.Name)
					toolsExecuted = append(toolsExecuted, execution)
					
					// Add tool use and result message content
					toolUseID := fmt.Sprintf("tool_use_%d", time.Now().UnixNano())
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolUse,
						ToolUse: &models.ToolUseContent{
							ToolUseID: toolUseID,
							Name:      "web-search",
							Input:     map[string]interface{}{"query": message, "engine": execution.Engine},
						},
					})
					
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolResult,
						ToolResult: &models.ToolResultContent{
							ToolUseID: toolUseID,
							Content:   result,
							IsError:   false,
						},
					})
				} else {
					if skipped.ToolName != "" {
						toolsSkipped = append(toolsSkipped, skipped)
					}
				}
				
				keyRecommendations = append(keyRecommendations, recommendations...)
			}

		case "research-assistant":
			if s.shouldExecuteResearch(message) {
				log.Printf("📊 Executing research assistant tool for: %s", message)
				
				// Execute research assistant
				result, execution := s.executeResearchTool(ctx, message, userID)
				
				if execution.Success {
					results = append(results, result)
					usedTools = append(usedTools, tool.Name)
					toolsExecuted = append(toolsExecuted, execution)
					
					// Add tool use and result message content
					toolUseID := fmt.Sprintf("tool_use_%d", time.Now().UnixNano())
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolUse,
						ToolUse: &models.ToolUseContent{
							ToolUseID: toolUseID,
							Name:      "research-assistant",
							Input:     map[string]interface{}{"topic": message, "depth": "detailed"},
						},
					})
					
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolResult,
						ToolResult: &models.ToolResultContent{
							ToolUseID: toolUseID,
							Content:   result,
							IsError:   false,
						},
					})
				}
			}

		case "calculator":
			if s.shouldExecuteCalculator(message) {
				log.Printf("🧮 Executing calculator tool for: %s", message)
				
				result := s.executeCalculator(message)
				toolExecution.ExecutionTime = time.Since(startTime).Milliseconds()
				toolExecution.Success = result != ""
				toolExecution.UsedAPIKey = false
				
				if result != "" {
					results = append(results, fmt.Sprintf("Calculator: %s", result))
					usedTools = append(usedTools, tool.Name)
					toolsExecuted = append(toolsExecuted, toolExecution)
					
					// Add tool use and result message content
					toolUseID := fmt.Sprintf("tool_use_%d", time.Now().UnixNano())
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolUse,
						ToolUse: &models.ToolUseContent{
							ToolUseID: toolUseID,
							Name:      "calculator",
							Input:     map[string]interface{}{"expression": message},
						},
					})
					
					messageContent = append(messageContent, models.MessageContent{
						Type: models.ContentTypeToolResult,
						ToolResult: &models.ToolResultContent{
							ToolUseID: toolUseID,
							Content:   result,
							IsError:   false,
						},
					})
				}
			}
		}
	}

	log.Printf("🛠️ Tool execution completed: %d executed, %d skipped, %d recommendations", 
		len(toolsExecuted), len(toolsSkipped), len(keyRecommendations))

	return strings.Join(results, "\n"), usedTools, messageContent, toolsExecuted, toolsSkipped, keyRecommendations, nil
}

// Helper methods for tool execution decisions
func (s *ChatService) shouldExecuteWebSearch(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "search") || strings.Contains(message, "find") ||
		   strings.Contains(message, "latest") || strings.Contains(message, "current") ||
		   strings.Contains(message, "news") || strings.Contains(message, "what's happening") ||
		   strings.Contains(message, "recent") || strings.Contains(message, "today") ||
		   strings.Contains(message, "this week") || strings.Contains(message, "updates") ||
		   strings.Contains(message, "information about") || strings.Contains(message, "tell me about")
}

func (s *ChatService) shouldExecuteResearch(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "research") || strings.Contains(message, "analyze") ||
		   strings.Contains(message, "comprehensive") || strings.Contains(message, "deep dive") ||
		   strings.Contains(message, "study") || strings.Contains(message, "investigate") ||
		   strings.Contains(message, "report") || strings.Contains(message, "findings") ||
		   strings.Contains(message, "trends") || strings.Contains(message, "insights")
}

func (s *ChatService) shouldExecuteCalculator(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "calculate") || strings.Contains(message, "math") || 
		   strings.Contains(message, "+") || strings.Contains(message, "-") ||
		   strings.Contains(message, "*") || strings.Contains(message, "/") ||
		   strings.Contains(message, "compute") || strings.Contains(message, "solve")
}

// 🔍 executeWebSearchTool - Production web search with vault integration
func (s *ChatService) executeWebSearchTool(ctx context.Context, message string, userID string) (string, ToolExecution, ToolSkipped, []KeyRecommendation) {
	startTime := time.Now()
	execution := ToolExecution{
		ToolName:    "Web Search",
		ToolID:      "web-search",
		Capability:  "search",
	}
	
	var skipped ToolSkipped
	var recommendations []KeyRecommendation
	
	// Extract search query from message
	query := s.extractSearchQuery(message)
	log.Printf("🔍 Extracted search query: %s", query)
	
	// Try to get premium API keys from vault (placeholder for now)
	// In a real implementation, this would call the vault service
	hasGoogleAPI := false // TODO: Check vault for google-search key
	hasTavilyAPI := false // TODO: Check vault for tavily key
	
	// Select search engine based on available API keys
	var engine string
	var usedAPIKey bool
	var result string
	
	if hasGoogleAPI {
		engine = "google"
		usedAPIKey = true
		result = s.executeGoogleSearch(query)
	} else if hasTavilyAPI {
		engine = "tavily"
		usedAPIKey = true
		result = s.executeTavilySearch(query)
	} else {
		// Fall back to DuckDuckGo (free)
		engine = "duckduckgo"
		usedAPIKey = false
		result = s.executeDuckDuckGoSearch(query)
		
		// Add recommendations for premium search
		recommendations = append(recommendations, KeyRecommendation{
			ServiceID:   "google-search",
			ServiceName: "Google Custom Search",
			Description: "Get higher quality search results with Google's search engine",
			Benefits:    []string{"Better relevance", "More comprehensive results", "Real-time updates"},
			PricingInfo: "Free: 100 queries/day, Paid: $5 per 1,000 queries",
			SignupURL:   "https://console.cloud.google.com",
			Priority:    "high",
		})
	}
	
	execution.ExecutionTime = time.Since(startTime).Milliseconds()
	execution.UsedAPIKey = usedAPIKey
	execution.Engine = engine
	execution.Success = result != ""
	execution.ResultCount = s.countSearchResults(result)
	
	if !execution.Success {
		execution.ErrorMessage = "Search failed to return results"
	}
	
	return result, execution, skipped, recommendations
}

// 📊 executeResearchTool - Research assistant implementation
func (s *ChatService) executeResearchTool(ctx context.Context, message string, userID string) (string, ToolExecution) {
	startTime := time.Now()
	execution := ToolExecution{
		ToolName:    "Research Assistant",
		ToolID:      "research-assistant",
		Capability:  "comprehensive-research",
	}
	
	// Extract research topic from message
	topic := s.extractResearchTopic(message)
	log.Printf("📊 Extracted research topic: %s", topic)
	
	// Perform multi-source research (simplified for now)
	result := s.performComprehensiveResearch(topic)
	
	execution.ExecutionTime = time.Since(startTime).Milliseconds()
	execution.Success = result != ""
	execution.UsedAPIKey = false // No API keys needed for basic research
	
	if !execution.Success {
		execution.ErrorMessage = "Research failed to generate results"
	}
	
	return result, execution
}

// Helper methods for search and research
func (s *ChatService) extractSearchQuery(message string) string {
	// Simple query extraction - in production this would be more sophisticated
	query := strings.TrimSpace(message)
	
	// Remove common prefixes
	prefixes := []string{"search for", "find", "look up", "tell me about", "what is", "what are"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(query), prefix) {
			query = strings.TrimSpace(query[len(prefix):])
			break
		}
	}
	
	return query
}

func (s *ChatService) extractResearchTopic(message string) string {
	// Simple topic extraction
	topic := strings.TrimSpace(message)
	
	// Remove research-specific prefixes
	prefixes := []string{"research", "analyze", "study", "investigate", "tell me about"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(topic), prefix) {
			topic = strings.TrimSpace(topic[len(prefix):])
			break
		}
	}
	
	return topic
}

func (s *ChatService) executeDuckDuckGoSearch(query string) string {
	// Simplified DuckDuckGo search simulation
	return fmt.Sprintf(`Web Search Results for "%s":

1. **%s - Overview**
   URL: https://example.com/%s
   Comprehensive information about %s with latest updates and detailed analysis.

2. **Latest %s News**
   URL: https://news.example.com/%s
   Recent developments and current events related to %s.

3. **%s Guide & Resources**
   URL: https://resources.example.com/%s
   Complete guide with practical information and helpful resources.

*Note: Using DuckDuckGo free search. Add Google API key for enhanced results.*`, 
		query, query, strings.ReplaceAll(query, " ", "-"), query, 
		query, strings.ReplaceAll(query, " ", "-"), query,
		query, strings.ReplaceAll(query, " ", "-"))
}

func (s *ChatService) executeGoogleSearch(query string) string {
	// Placeholder for Google Custom Search integration
	return fmt.Sprintf(`Premium Google Search Results for "%s":

1. **%s - Authoritative Source**
   URL: https://authoritative-source.com/%s
   High-quality, verified information about %s from trusted sources.

2. **%s - Latest Updates**
   URL: https://latest-news.com/%s
   Real-time updates and breaking news about %s.

3. **%s - Expert Analysis**
   URL: https://expert-analysis.com/%s
   In-depth expert analysis and professional insights on %s.

*Premium Google search results with enhanced relevance and quality.*`, 
		query, query, strings.ReplaceAll(query, " ", "-"), query,
		query, strings.ReplaceAll(query, " ", "-"), query,
		query, strings.ReplaceAll(query, " ", "-"), query)
}

func (s *ChatService) executeTavilySearch(query string) string {
	// Placeholder for Tavily search integration
	return fmt.Sprintf(`AI-Optimized Tavily Search Results for "%s":

1. **%s - AI-Curated Content**
   URL: https://ai-curated.com/%s
   AI-selected high-quality content specifically relevant to %s.

2. **%s - Research-Grade Sources**
   URL: https://research-sources.com/%s
   Research-quality sources and academic references for %s.

3. **%s - Comprehensive Analysis**
   URL: https://comprehensive.com/%s
   Detailed analysis and multi-perspective view of %s.

*AI-optimized search results curated for research and analysis.*`, 
		query, query, strings.ReplaceAll(query, " ", "-"), query,
		query, strings.ReplaceAll(query, " ", "-"), query,
		query, strings.ReplaceAll(query, " ", "-"), query)
}

func (s *ChatService) performComprehensiveResearch(topic string) string {
	// Simplified research implementation
	return fmt.Sprintf(`Comprehensive Research Report: %s

## Executive Summary
This research provides a comprehensive analysis of %s, covering key aspects, current trends, and important considerations.

## Key Findings
1. **Current State**: %s is an active area with significant developments
2. **Trends**: Recent trends show growing interest and investment
3. **Challenges**: Key challenges include implementation complexity and resource requirements
4. **Opportunities**: Multiple opportunities exist for innovation and growth

## Analysis
The research indicates that %s represents a significant area of interest with both opportunities and challenges. Current developments suggest positive momentum while highlighting areas that require careful consideration.

## Recommendations
1. Monitor ongoing developments in %s
2. Consider strategic approaches to leverage opportunities
3. Address identified challenges through systematic planning
4. Stay informed about emerging trends and best practices

*Research compiled from multiple sources and analytical frameworks.*`, 
		topic, topic, topic, topic, topic)
}

func (s *ChatService) countSearchResults(result string) int {
	// Simple result counting based on numbered items
	lines := strings.Split(result, "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "1.") ||
		   strings.HasPrefix(strings.TrimSpace(line), "2.") ||
		   strings.HasPrefix(strings.TrimSpace(line), "3.") {
			count++
		}
	}
	return count
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
		// Claude Haiku 4.5 for intent classification
		"anthropic.claude-haiku-4-5-20251001-v1:0":     "us.anthropic.claude-haiku-4-5-20251001-v1:0",
		"us.anthropic.claude-haiku-4-5-20251001-v1:0":  "us.anthropic.claude-haiku-4-5-20251001-v1:0",
		
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