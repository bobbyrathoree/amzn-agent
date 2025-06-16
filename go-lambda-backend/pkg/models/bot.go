package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Common errors
var (
	ErrNotAuthorized = errors.New("not authorized")
)

// Bot represents an AI chat bot configuration with full-featured support
type Bot struct {
	// Core properties
	ID                    string                 `json:"id" dynamodbav:"id"`
	Title                 string                 `json:"title" dynamodbav:"title" validate:"required,max=100"`
	Description           string                 `json:"description" dynamodbav:"description" validate:"max=500"`
	Instruction           string                 `json:"instruction" dynamodbav:"instruction" validate:"required,max=4000"`
	OwnerUserID           string                 `json:"ownerUserId" dynamodbav:"ownerUserId" validate:"required"`
	
	// Timestamps
	CreateTime            time.Time              `json:"createTime" dynamodbav:"createTime"`
	LastUsedTime          time.Time              `json:"lastUsedTime" dynamodbav:"lastUsedTime"`
	UpdateTime            time.Time              `json:"updateTime" dynamodbav:"updateTime"`
	
	// Knowledge Base configuration (conditional)
	KnowledgeBaseID       *string                `json:"knowledgeBaseId,omitempty" dynamodbav:"knowledgeBaseId,omitempty"`
	KnowledgeBaseConfig   KnowledgeBaseConfig    `json:"knowledgeBaseConfig" dynamodbav:"knowledgeBaseConfig"`
	
	// CloudFormation stack tracking (for dynamically created KBs)
	CloudFormationStackName *string              `json:"cloudFormationStackName,omitempty" dynamodbav:"cloudFormationStackName,omitempty"`
	StackStatus           *string                `json:"stackStatus,omitempty" dynamodbav:"stackStatus,omitempty"` // CREATE_IN_PROGRESS, CREATE_COMPLETE, etc.
	DocumentBucketName    *string                `json:"documentBucketName,omitempty" dynamodbav:"documentBucketName,omitempty"`
	GuardrailArn          *string                `json:"guardrailArn,omitempty" dynamodbav:"guardrailArn,omitempty"`
	GuardrailVersion      *string                `json:"guardrailVersion,omitempty" dynamodbav:"guardrailVersion,omitempty"`
	
	// Async processing status tracking
	SyncStatus            string                 `json:"syncStatus" dynamodbav:"syncStatus" validate:"oneof=QUEUED RUNNING SUCCEEDED FAILED"`
	SyncStatusReason      string                 `json:"syncStatusReason,omitempty" dynamodbav:"syncStatusReason,omitempty"`
	SyncLastExecID        string                 `json:"syncLastExecId,omitempty" dynamodbav:"syncLastExecId,omitempty"`
	
	// Sharing configuration (3-tier system)
	SharedScope           string                 `json:"sharedScope" dynamodbav:"sharedScope" validate:"oneof=private partial public"`
	SharedStatus          string                 `json:"sharedStatus" dynamodbav:"sharedStatus" validate:"oneof=unshared shared pinned"`
	AllowedUsers          []string               `json:"allowedUsers,omitempty" dynamodbav:"allowedUsers,omitempty"`
	AllowedGroups         []string               `json:"allowedGroups,omitempty" dynamodbav:"allowedGroups,omitempty"`
	
	// User preferences
	IsStarred             bool                   `json:"isStarred" dynamodbav:"isStarred"`
	
	// Usage analytics
	UsageCount            int                    `json:"usageCount" dynamodbav:"usageCount"`
	
	// Generation and behavior configuration
	GenerationParams      GenerationParams       `json:"generationParams" dynamodbav:"generationParams"`
	ConversationStarters  []ConversationStarter  `json:"conversationStarters" dynamodbav:"conversationStarters"`
	ActiveModels          []string               `json:"activeModels" dynamodbav:"activeModels"`
	AgentTools            []AgentTool            `json:"agentTools" dynamodbav:"agentTools"`
	DisplayRetrievedChunks bool                  `json:"displayRetrievedChunks" dynamodbav:"displayRetrievedChunks"`
}

// SharedScope constants
const (
	SharedScopePrivate = "private"
	SharedScopePartial = "partial"
	SharedScopePublic  = "public"
)

// SharedStatus constants
const (
	SharedStatusUnshared = "unshared"
	SharedStatusShared   = "shared" 
	SharedStatusPinned   = "pinned"   // This is the base for pinned status
	// Actual pinned status format: "pinned@001", "pinned@002", etc.
)

// StackStatus constants
const (
	StackStatusCreateInProgress = "CREATE_IN_PROGRESS"
	StackStatusCreateComplete   = "CREATE_COMPLETE"
	StackStatusCreateFailed     = "CREATE_FAILED"
	StackStatusDeleteInProgress = "DELETE_IN_PROGRESS"
	StackStatusDeleteComplete   = "DELETE_COMPLETE"
	StackStatusDeleteFailed     = "DELETE_FAILED"
	StackStatusUpdateInProgress = "UPDATE_IN_PROGRESS"
	StackStatusUpdateComplete   = "UPDATE_COMPLETE"
	StackStatusUpdateFailed     = "UPDATE_FAILED"
)

// SyncStatus constants for async processing tracking
const (
	SyncStatusQueued    = "QUEUED"    // Processing queued but not started
	SyncStatusRunning   = "RUNNING"   // Currently processing
	SyncStatusSucceeded = "SUCCEEDED" // Processing completed successfully
	SyncStatusFailed    = "FAILED"    // Processing failed
)

// GenerationParams contains LLM generation parameters
type GenerationParams struct {
	MaxTokens       int      `json:"maxTokens" dynamodbav:"maxTokens" validate:"min=1,max=4096"`
	Temperature     float64  `json:"temperature" dynamodbav:"temperature" validate:"min=0,max=1"`
	TopP            float64  `json:"topP" dynamodbav:"topP" validate:"min=0,max=1"`
	TopK            int      `json:"topK" dynamodbav:"topK" validate:"min=0,max=500"`
	StopSequences   []string `json:"stopSequences" dynamodbav:"stopSequences"`
}

// KnowledgeBaseConfig contains knowledge base configuration
type KnowledgeBaseConfig struct {
	SearchType      string  `json:"searchType" dynamodbav:"searchType" validate:"oneof=SEMANTIC HYBRID"`
	MaxResults      int     `json:"maxResults" dynamodbav:"maxResults" validate:"min=1,max=100"`
	ScoreThreshold  float64 `json:"scoreThreshold" dynamodbav:"scoreThreshold" validate:"min=0,max=1"`
}

// ConversationStarter represents a rich conversation starter object (bedrock-chat compatibility)
type ConversationStarter struct {
	Title   string `json:"title" dynamodbav:"title" validate:"required,max=100"`
	Example string `json:"example" dynamodbav:"example" validate:"required,max=500"`
}

// AgentTool represents a tool that can be used by the AI agent
// This follows bedrock-chat's polymorphic tool system
type AgentTool struct {
	Type        string                 `json:"type" dynamodbav:"type" validate:"required,oneof=plain internet bedrock_agent"`
	Name        string                 `json:"name" dynamodbav:"name" validate:"required"`
	Description string                 `json:"description" dynamodbav:"description" validate:"required"`
	Config      map[string]interface{} `json:"config,omitempty" dynamodbav:"config,omitempty"`
}

// PlainTool represents a basic tool without external dependencies
type PlainTool struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
}

// InternetTool represents a tool that performs internet searches
type InternetTool struct {
	Name         string `json:"name" validate:"required"`
	Description  string `json:"description" validate:"required"`
	SearchEngine string `json:"searchEngine" validate:"oneof=tavily duckduckgo google"`
	APIKey       string `json:"apiKey,omitempty"` // For services that require API keys
	MaxResults   int    `json:"maxResults" validate:"min=1,max=20"`
}

// BedrockAgentTool represents integration with AWS Bedrock Agents
type BedrockAgentTool struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description" validate:"required"`
	AgentID     string `json:"agentId" validate:"required"`
	AgentAlias  string `json:"agentAlias" validate:"required"`
	Region      string `json:"region" validate:"required"`
}

// Tool configuration helpers
func NewPlainTool(name, description string) AgentTool {
	return AgentTool{
		Type:        "plain",
		Name:        name,
		Description: description,
		Config: map[string]interface{}{
			"toolType": "plain",
		},
	}
}

func NewInternetTool(name, description, searchEngine string, maxResults int) AgentTool {
	config := map[string]interface{}{
		"toolType":     "internet",
		"searchEngine": searchEngine,
		"maxResults":   maxResults,
	}
	
	return AgentTool{
		Type:        "internet",
		Name:        name,
		Description: description,
		Config:      config,
	}
}

func NewBedrockAgentTool(name, description, agentID, agentAlias, region string) AgentTool {
	config := map[string]interface{}{
		"toolType":   "bedrock_agent",
		"agentId":    agentID,
		"agentAlias": agentAlias,
		"region":     region,
	}
	
	return AgentTool{
		Type:        "bedrock_agent",
		Name:        name,
		Description: description,
		Config:      config,
	}
}

// CreateBotRequest represents the input to create a new bot with full-featured support
type CreateBotRequest struct {
	// Core properties
	Title                 string                 `json:"title" validate:"required,max=100"`
	Description           string                 `json:"description" validate:"max=500"`
	Instruction           string                 `json:"instruction" validate:"required,max=4000"`
	
	// Conditional Knowledge Base options - either provide existing KB ID OR creation parameters
	ExistingKnowledgeBaseID *string              `json:"existingKnowledgeBaseId,omitempty"`
	
	// New KB creation parameters (used only if ExistingKnowledgeBaseID is not provided)
	KnowledgeBaseCreation *KnowledgeBaseCreationConfig `json:"knowledgeBaseCreation,omitempty"`
	
	// Sharing configuration
	SharedScope           string                 `json:"sharedScope" validate:"required,oneof=private partial public"`
	AllowedUsers          []string               `json:"allowedUsers,omitempty"`
	AllowedGroups         []string               `json:"allowedGroups,omitempty"`
	
	// Configuration
	GenerationParams      GenerationParams       `json:"generationParams"`
	KnowledgeBaseConfig   KnowledgeBaseConfig    `json:"knowledgeBaseConfig"`
	ConversationStarters  []ConversationStarter  `json:"conversationStarters"`
	ActiveModels          []string               `json:"activeModels"`
	AgentTools            []AgentTool            `json:"agentTools"`
	DisplayRetrievedChunks bool                  `json:"displayRetrievedChunks"`
}

// KnowledgeBaseCreationConfig contains parameters for creating a new Knowledge Base
type KnowledgeBaseCreationConfig struct {
	EmbeddingsModel       string                 `json:"embeddingsModel,omitempty" validate:"omitempty,oneof=amazon.titan-embed-text-v1 amazon.titan-embed-text-v2:0 cohere.embed-english-v3 cohere.embed-multilingual-v3"`
	ChunkingStrategy      string                 `json:"chunkingStrategy,omitempty" validate:"omitempty,oneof=FIXED_SIZE NONE HIERARCHICAL SEMANTIC"`
	MaxTokens             int                    `json:"maxTokens,omitempty" validate:"omitempty,min=20,max=8192"`
	OverlapPercentage     int                    `json:"overlapPercentage,omitempty" validate:"omitempty,min=1,max=99"`
	
	// Data sources for new KB
	ExistingS3Urls        []string               `json:"existingS3Urls,omitempty"`    // S3 URLs to index
	SourceUrls            []string               `json:"sourceUrls,omitempty"`        // Web URLs to crawl
	
	// Guardrail configuration
	GuardrailConfig       *GuardrailConfig       `json:"guardrailConfig,omitempty"`
	
	// Infrastructure options
	EnableRagReplicas     bool                   `json:"enableRagReplicas,omitempty"`
}

// GuardrailConfig contains content filtering configuration
type GuardrailConfig struct {
	IsEnabled             bool                   `json:"isEnabled"`
	HateThreshold         *float64               `json:"hateThreshold,omitempty" validate:"omitempty,min=0,max=1"`
	InsultsThreshold      *float64               `json:"insultsThreshold,omitempty" validate:"omitempty,min=0,max=1"`
	SexualThreshold       *float64               `json:"sexualThreshold,omitempty" validate:"omitempty,min=0,max=1"`
	ViolenceThreshold     *float64               `json:"violenceThreshold,omitempty" validate:"omitempty,min=0,max=1"`
	MisconductThreshold   *float64               `json:"misconductThreshold,omitempty" validate:"omitempty,min=0,max=1"`
	GroundingThreshold    *float64               `json:"groundingThreshold,omitempty" validate:"omitempty,min=0,max=1"`
	RelevanceThreshold    *float64               `json:"relevanceThreshold,omitempty" validate:"omitempty,min=0,max=1"`
}

// UpdateBotRequest represents the input to update an existing bot with full-featured support
type UpdateBotRequest struct {
	// Core fields
	Title                 *string                `json:"title" validate:"omitempty,max=100"`
	Description           *string                `json:"description" validate:"omitempty,max=500"`
	Instruction           *string                `json:"instruction" validate:"omitempty,max=4000"`
	
	// Knowledge Base fields
	KnowledgeBaseID       *string                `json:"knowledgeBaseId"`
	KnowledgeBaseConfig   *KnowledgeBaseConfig   `json:"knowledgeBaseConfig"`
	
	// CloudFormation stack fields
	CloudFormationStackName *string              `json:"cloudFormationStackName"`
	StackStatus           *string                `json:"stackStatus"`
	DocumentBucketName    *string                `json:"documentBucketName"`
	GuardrailArn          *string                `json:"guardrailArn"`
	GuardrailVersion      *string                `json:"guardrailVersion"`
	
	// Sharing fields
	SharedScope           *string                `json:"sharedScope" validate:"omitempty,oneof=private partial public"`
	SharedStatus          *string                `json:"sharedStatus" validate:"omitempty,oneof=unshared shared pinned"`
	AllowedUsers          []string               `json:"allowedUsers"`
	AllowedGroups         []string               `json:"allowedGroups"`
	
	// User preferences
	IsStarred             *bool                  `json:"isStarred"`
	
	// Usage analytics
	UsageCount            *int                   `json:"usageCount"`
	
	// Sync status fields
	SyncStatus            *string                `json:"syncStatus" validate:"omitempty,oneof=QUEUED RUNNING SUCCEEDED FAILED"`
	SyncStatusReason      *string                `json:"syncStatusReason"`
	SyncLastExecID        *string                `json:"syncLastExecId"`
	
	// Configuration fields
	GenerationParams      *GenerationParams      `json:"generationParams"`
	ConversationStarters  []ConversationStarter  `json:"conversationStarters"`
	ActiveModels          []string               `json:"activeModels"`
	AgentTools            []AgentTool            `json:"agentTools"`
	DisplayRetrievedChunks *bool                 `json:"displayRetrievedChunks"`
}

// BotResponse represents the API response structure
type BotResponse struct {
	Bot       *Bot   `json:"bot,omitempty"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// BotsResponse represents a paginated list of bots
type BotsResponse struct {
	Bots      interface{} `json:"bots"` // Can be []Bot or []BotSummary
	Count     int         `json:"count"`
	Total     int         `json:"total"`
	NextToken string      `json:"nextToken,omitempty"`
	RequestID string      `json:"requestId,omitempty"`
}

// BotSummary represents a minimal bot for listing purposes
type BotSummary struct {
	ID                   string                `json:"id"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	IsStarred            bool                  `json:"isStarred"`
	OwnerUserID          string                `json:"ownerUserId"`
	CreateTime           time.Time             `json:"createTime"`
	LastUsedTime         time.Time             `json:"lastUsedTime"`
	HasKnowledgeBase     bool                  `json:"hasKnowledgeBase"`
	ConversationStarters []ConversationStarter `json:"conversationStarters"`
	SharedScope          string                `json:"sharedScope"`
	SharedStatus         string                `json:"sharedStatus"`
	ActiveModels         []string              `json:"activeModels"`
}

// Available models for bot configuration
var AvailableModels = []string{
	"anthropic.claude-3-5-sonnet-20241022-v2:0",
	"anthropic.claude-3-5-sonnet-20240620-v1:0",
	"anthropic.claude-3-haiku-20240307-v1:0",
	"anthropic.claude-3-opus-20240229-v1:0",
	"amazon.titan-text-premier-v1:0",
	"meta.llama3-2-90b-instruct-v1:0",
	"meta.llama3-2-11b-instruct-v1:0",
	"meta.llama3-2-3b-instruct-v1:0",
	"meta.llama3-2-1b-instruct-v1:0",
}

// Available agent tools - predefined tools that can be used
var AvailableAgentTools = []AgentTool{
	NewPlainTool("calculator", "Perform mathematical calculations and solve equations"),
	NewPlainTool("code_interpreter", "Execute and analyze code in various programming languages"),
	NewPlainTool("file_analysis", "Analyze and process file contents and metadata"),
	NewInternetTool("web_search", "Search the internet for current information", "duckduckgo", 10),
}

// GetAvailableToolNames returns just the names of available tools for backward compatibility
func GetAvailableToolNames() []string {
	names := make([]string, len(AvailableAgentTools))
	for i, tool := range AvailableAgentTools {
		names[i] = tool.Name
	}
	return names
}

// Default generation parameters
func DefaultGenerationParams() GenerationParams {
	return GenerationParams{
		MaxTokens:     2048,
		Temperature:   0.7,
		TopP:          0.9,
		TopK:          250,
		StopSequences: []string{},
	}
}

// Default knowledge base configuration
func DefaultKnowledgeBaseConfig() KnowledgeBaseConfig {
	return KnowledgeBaseConfig{
		SearchType:     "HYBRID",
		MaxResults:     20,
		ScoreThreshold: 0.7,
	}
}

// Access control methods for bot sharing

// IsAccessibleByUser checks if a user has access to this bot based on sharing configuration
func (b *Bot) IsAccessibleByUser(userID string, userGroups []string, isAdmin bool) bool {
	// Admin and owner always have access
	if isAdmin || b.OwnerUserID == userID {
		return true
	}
	
	// Private bots - no access for others
	if b.SharedScope == SharedScopePrivate {
		return false
	}
	
	// Public bots - universal access
	if b.SharedScope == SharedScopePublic {
		return true
	}
	
	// Partial sharing - check explicit permissions
	if b.SharedScope == SharedScopePartial {
		// Check if user is in allowed users list
		for _, allowedUser := range b.AllowedUsers {
			if allowedUser == userID {
				return true
			}
		}
		
		// Check if user belongs to any allowed groups
		for _, allowedGroup := range b.AllowedGroups {
			for _, userGroup := range userGroups {
				if allowedGroup == userGroup {
					return true
				}
			}
		}
	}
	
	return false
}

// IsOwnedByUser checks if the bot is owned by the specified user
func (b *Bot) IsOwnedByUser(userID string) bool {
	return b.OwnerUserID == userID
}

// IsEditableByUser checks if a user can edit/modify this bot (owner only)
func (b *Bot) IsEditableByUser(userID string, userGroups []string, isAdmin bool) bool {
	// Only admins and owners can edit bots
	return isAdmin || b.OwnerUserID == userID
}

// IsPubliclyDiscoverable checks if the bot should appear in public bot discovery
func (b *Bot) IsPubliclyDiscoverable() bool {
	return b.SharedScope == SharedScopePublic && b.SharedStatus == SharedStatusShared
}

// IsPinned checks if the bot is pinned (featured in public discovery)
func (b *Bot) IsPinned() bool {
	return b.SharedScope == SharedScopePublic && strings.HasPrefix(b.SharedStatus, "pinned@")
}

// GetPinOrder returns the pin order number (001, 002, etc.) or 0 if not pinned
func (b *Bot) GetPinOrder() int {
	if !b.IsPinned() {
		return 0
	}
	
	// Extract order from "pinned@XXX" format
	parts := strings.Split(b.SharedStatus, "@")
	if len(parts) != 2 {
		return 0
	}
	
	order := 0
	fmt.Sscanf(parts[1], "%d", &order)
	return order
}

// SetPinnedStatus sets the bot as pinned with the specified order
func (b *Bot) SetPinnedStatus(order int) {
	b.SharedStatus = fmt.Sprintf("pinned@%03d", order)
}

// UnpinBot removes the pinned status, setting it back to shared
func (b *Bot) UnpinBot() {
	if b.IsPinned() {
		b.SharedStatus = SharedStatusShared
	}
}

// CanBePinned checks if a bot can be pinned (must be public and shared)
func (b *Bot) CanBePinned() bool {
	return b.SharedScope == SharedScopePublic && 
		   (b.SharedStatus == SharedStatusShared || b.IsPinned())
}

// HasDynamicKnowledgeBase checks if this bot uses a dynamically created Knowledge Base
func (b *Bot) HasDynamicKnowledgeBase() bool {
	return b.CloudFormationStackName != nil && *b.CloudFormationStackName != ""
}

// IsStackComplete checks if the CloudFormation stack is fully deployed
func (b *Bot) IsStackComplete() bool {
	return b.StackStatus != nil && *b.StackStatus == StackStatusCreateComplete
}

// IsStackInProgress checks if the CloudFormation stack is currently being created/updated
func (b *Bot) IsStackInProgress() bool {
	if b.StackStatus == nil {
		return false
	}
	return *b.StackStatus == StackStatusCreateInProgress || 
		   *b.StackStatus == StackStatusUpdateInProgress
}

// IsStackFailed checks if the CloudFormation stack deployment failed
func (b *Bot) IsStackFailed() bool {
	if b.StackStatus == nil {
		return false
	}
	return *b.StackStatus == StackStatusCreateFailed || 
		   *b.StackStatus == StackStatusUpdateFailed
}

// GetStackName returns the CloudFormation stack name for this bot
func (b *Bot) GetStackName() string {
	return fmt.Sprintf("BrChatKbStack%s", b.ID)
}

// ToSummary converts the bot to a summary for listing purposes
func (b *Bot) ToSummary() BotSummary {
	return BotSummary{
		ID:                   b.ID,
		Title:                b.Title,
		Description:          b.Description,
		IsStarred:            b.IsStarred,
		OwnerUserID:          b.OwnerUserID,
		CreateTime:           b.CreateTime,
		LastUsedTime:         b.LastUsedTime,
		HasKnowledgeBase:     b.KnowledgeBaseID != nil && *b.KnowledgeBaseID != "",
		ConversationStarters: b.ConversationStarters,
		SharedScope:          b.SharedScope,
		SharedStatus:         b.SharedStatus,
		ActiveModels:         b.ActiveModels,
	}
}

// BotAlias represents a user's personal reference to a shared bot
type BotAlias struct {
	// Identity
	ID                    string                 `json:"id" dynamodbav:"id"`                         // Unique alias ID
	OriginalBotID         string                 `json:"originalBotId" dynamodbav:"originalBotId"`   // Reference to original bot
	UserID                string                 `json:"userId" dynamodbav:"userId"`                 // User who created this alias
	
	// Cached metadata for performance (from original bot)
	Title                 string                 `json:"title" dynamodbav:"title"`
	Description           string                 `json:"description" dynamodbav:"description"`
	OwnerUserID           string                 `json:"ownerUserId" dynamodbav:"ownerUserId"`       // Original bot owner
	
	// User-specific preferences
	IsStarred             bool                   `json:"isStarred" dynamodbav:"isStarred"`
	LastUsedTime          time.Time              `json:"lastUsedTime" dynamodbav:"lastUsedTime"`
	CreateTime            time.Time              `json:"createTime" dynamodbav:"createTime"`         // When alias was created
	
	// Access tracking
	IsOriginAccessible    bool                   `json:"isOriginAccessible" dynamodbav:"isOriginAccessible"` // Can still access original?
	
	// Cached bot properties for quick access
	SharedScope           string                 `json:"sharedScope" dynamodbav:"sharedScope"`
	SharedStatus          string                 `json:"sharedStatus" dynamodbav:"sharedStatus"`
	HasKnowledgeBase      bool                   `json:"hasKnowledgeBase" dynamodbav:"hasKnowledgeBase"`
	ConversationStarters  []ConversationStarter  `json:"conversationStarters" dynamodbav:"conversationStarters"`
}

// AliasCreateRequest represents a request to create a bot alias
type AliasCreateRequest struct {
	OriginalBotID string `json:"originalBotId" validate:"required"`
}

// BotAliasResponse represents the API response for bot alias operations
type BotAliasResponse struct {
	Alias     *BotAlias `json:"alias,omitempty"`
	Message   string    `json:"message,omitempty"`
	RequestID string    `json:"requestId,omitempty"`
}

// ToSummary converts the bot alias to a summary that looks like a regular bot summary
func (a *BotAlias) ToSummary() BotSummary {
	return BotSummary{
		ID:                   a.ID,
		Title:                a.Title,
		Description:          a.Description,
		IsStarred:            a.IsStarred,
		OwnerUserID:          a.OwnerUserID,          // Original owner
		CreateTime:           a.CreateTime,           // Alias creation time
		LastUsedTime:         a.LastUsedTime,
		HasKnowledgeBase:     a.HasKnowledgeBase,
		ConversationStarters: a.ConversationStarters,
		SharedScope:          a.SharedScope,
		SharedStatus:         a.SharedStatus,
		ActiveModels:         []string{}, // TODO: Cache active models from original bot or fetch dynamically
	}
}

// GetBotID returns the effective bot ID to use for accessing the actual bot
func (a *BotAlias) GetBotID() string {
	return a.OriginalBotID
}

// IsAlias indicates this is an alias (for type checking)
func (a *BotAlias) IsAlias() bool {
	return true
}

// IsEditableByUser checks if a user can edit this alias (only the alias owner)
func (a *BotAlias) IsEditableByUser(userID string, userGroups []string, isAdmin bool) bool {
	// Only admins and the alias owner can edit aliases
	return isAdmin || a.UserID == userID
}

// IsAccessibleByUser checks if a user can access this alias
func (a *BotAlias) IsAccessibleByUser(userID string, userGroups []string, isAdmin bool) bool {
	// Same as edit permissions for aliases - only owner/admin can see their aliases
	return a.IsEditableByUser(userID, userGroups, isAdmin)
}

// === Sync Status Helper Methods ===

// IsSyncInProgress checks if Knowledge Base processing is currently in progress
func (b *Bot) IsSyncInProgress() bool {
	return b.SyncStatus == SyncStatusQueued || b.SyncStatus == SyncStatusRunning
}

// IsSyncComplete checks if Knowledge Base processing completed successfully
func (b *Bot) IsSyncComplete() bool {
	return b.SyncStatus == SyncStatusSucceeded
}

// IsSyncFailed checks if Knowledge Base processing failed
func (b *Bot) IsSyncFailed() bool {
	return b.SyncStatus == SyncStatusFailed
}

// RequiresAsync determines if a bot creation request requires async processing
func (req *CreateBotRequest) RequiresAsync() bool {
	// If using existing KB, no async processing needed
	if req.ExistingKnowledgeBaseID != nil && *req.ExistingKnowledgeBaseID != "" {
		return false
	}
	
	// If creating new KB, async processing is required
	if req.KnowledgeBaseCreation != nil {
		return true
	}
	
	// If guardrails are specified, may require async setup
	if req.KnowledgeBaseCreation != nil && req.KnowledgeBaseCreation.GuardrailConfig != nil && req.KnowledgeBaseCreation.GuardrailConfig.IsEnabled {
		return true
	}
	
	return false
}

// DetermineSyncStatus determines the initial sync status for a bot creation request
func (req *CreateBotRequest) DetermineSyncStatus() string {
	if req.RequiresAsync() {
		return SyncStatusQueued
	}
	return SyncStatusSucceeded
}

// === Usage Analytics Helper Methods ===

// IncrementUsage increments the usage count by 1
func (b *Bot) IncrementUsage() {
	b.UsageCount++
}

// GetUsageCount returns the current usage count
func (b *Bot) GetUsageCount() int {
	return b.UsageCount
}

// IsPopular checks if the bot has high usage (for discovery algorithms)
func (b *Bot) IsPopular() bool {
	return b.UsageCount >= 10 // Configurable threshold
}

// GetPopularityScore returns a score for ranking popular bots
func (b *Bot) GetPopularityScore() float64 {
	// Simple scoring: usage count with decay for age
	daysSinceCreation := time.Since(b.CreateTime).Hours() / 24
	if daysSinceCreation == 0 {
		daysSinceCreation = 1 // Avoid division by zero
	}
	
	// Score decreases over time to promote newer popular content
	return float64(b.UsageCount) / (1 + daysSinceCreation/30) // 30-day decay factor
}