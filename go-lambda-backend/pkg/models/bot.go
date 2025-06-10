package models

import (
	"errors"
	"time"
)

// Common errors
var (
	ErrNotAuthorized = errors.New("not authorized")
)

// Bot represents an AI chat bot configuration
type Bot struct {
	ID                    string                 `json:"id" dynamodbav:"id"`
	Title                 string                 `json:"title" dynamodbav:"title" validate:"required,max=100"`
	Description           string                 `json:"description" dynamodbav:"description" validate:"max=500"`
	Instruction           string                 `json:"instruction" dynamodbav:"instruction" validate:"required,max=4000"`
	OwnerUserID           string                 `json:"ownerUserId" dynamodbav:"ownerUserId" validate:"required"`
	CreateTime            time.Time              `json:"createTime" dynamodbav:"createTime"`
	LastUsedTime          time.Time              `json:"lastUsedTime" dynamodbav:"lastUsedTime"`
	IsPublic              bool                   `json:"isPublic" dynamodbav:"isPublic"`
	IsStarred             bool                   `json:"isStarred" dynamodbav:"isStarred"`
	GenerationParams      GenerationParams       `json:"generationParams" dynamodbav:"generationParams"`
	KnowledgeBaseID       string                 `json:"knowledgeBaseId" dynamodbav:"knowledgeBaseId" validate:"required"`
	KnowledgeBaseConfig   KnowledgeBaseConfig    `json:"knowledgeBaseConfig" dynamodbav:"knowledgeBaseConfig"`
	ConversationStarters  []string               `json:"conversationStarters" dynamodbav:"conversationStarters"`
	ActiveModels          []string               `json:"activeModels" dynamodbav:"activeModels"`
	AgentTools            []string               `json:"agentTools" dynamodbav:"agentTools"`
	DisplayRetrievedChunks bool                  `json:"displayRetrievedChunks" dynamodbav:"displayRetrievedChunks"`
}

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

// CreateBotRequest represents the input to create a new bot
type CreateBotRequest struct {
	Title                 string                 `json:"title" validate:"required,max=100"`
	Description           string                 `json:"description" validate:"max=500"`
	Instruction           string                 `json:"instruction" validate:"required,max=4000"`
	IsPublic              bool                   `json:"isPublic"`
	GenerationParams      GenerationParams       `json:"generationParams"`
	KnowledgeBaseID       string                 `json:"knowledgeBaseId" validate:"required"`
	KnowledgeBaseConfig   KnowledgeBaseConfig    `json:"knowledgeBaseConfig"`
	ConversationStarters  []string               `json:"conversationStarters"`
	ActiveModels          []string               `json:"activeModels"`
	AgentTools            []string               `json:"agentTools"`
	DisplayRetrievedChunks bool                  `json:"displayRetrievedChunks"`
}

// UpdateBotRequest represents the input to update an existing bot
type UpdateBotRequest struct {
	Title                 *string                `json:"title" validate:"omitempty,max=100"`
	Description           *string                `json:"description" validate:"omitempty,max=500"`
	Instruction           *string                `json:"instruction" validate:"omitempty,max=4000"`
	IsPublic              *bool                  `json:"isPublic"`
	IsStarred             *bool                  `json:"isStarred"`
	GenerationParams      *GenerationParams      `json:"generationParams"`
	KnowledgeBaseID       *string                `json:"knowledgeBaseId"`
	KnowledgeBaseConfig   *KnowledgeBaseConfig   `json:"knowledgeBaseConfig"`
	ConversationStarters  []string               `json:"conversationStarters"`
	ActiveModels          []string               `json:"activeModels"`
	AgentTools            []string               `json:"agentTools"`
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
	Bots      []Bot  `json:"bots"`
	Count     int    `json:"count"`
	Total     int    `json:"total"`
	NextToken string `json:"nextToken,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// BotSummary represents a minimal bot for listing purposes
type BotSummary struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	IsPublic             bool      `json:"isPublic"`
	IsStarred            bool      `json:"isStarred"`
	OwnerUserID          string    `json:"ownerUserId"`
	CreateTime           time.Time `json:"createTime"`
	LastUsedTime         time.Time `json:"lastUsedTime"`
	HasKnowledgeBase     bool      `json:"hasKnowledgeBase"`
	ConversationStarters []string  `json:"conversationStarters"`
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

// Available agent tools
var AvailableAgentTools = []string{
	"web_search",
	"calculator",
	"code_interpreter",
	"file_analysis",
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