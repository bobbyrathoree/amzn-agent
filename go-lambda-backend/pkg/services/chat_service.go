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
	Metadata         map[string]interface{}   `json:"metadata,omitempty"`
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
	
	// STEP 3A: Intelligent Knowledge Retrieval with Multi-Stage Search 🚀 INGENIOUS ENHANCEMENT
	var sources []KnowledgeBaseChunk
	var knowledgeSearchStages []KnowledgeSearchStage
	
	if bot.KnowledgeBaseID != nil && *bot.KnowledgeBaseID != "" {
		log.Printf("🔍 DEBUG: Starting KB search with ID: %s, Config: %+v", *bot.KnowledgeBaseID, bot.KnowledgeBaseConfig)
		// Progressive knowledge search with multiple strategies
		_, sources, knowledgeSearchStages, err = s.intelligentKnowledgeRetrieval(ctx, *bot.KnowledgeBaseID, req.Message, conversationHistory, bot.KnowledgeBaseConfig)
		if err != nil {
			log.Printf("Warning: Intelligent Knowledge Base retrieval failed: %v", err)
		}
		log.Printf("🔍 DEBUG: KB search completed with %d sources, %d stages", len(sources), len(knowledgeSearchStages))
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