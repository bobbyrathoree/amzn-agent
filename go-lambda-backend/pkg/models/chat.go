package models

import "time"

// Message represents a chat message
type Message struct {
	ID             string    `json:"id" dynamodbav:"id"`
	ConversationID string    `json:"conversationId" dynamodbav:"conversationId"`
	Role           string    `json:"role" dynamodbav:"role"` // user, assistant, system
	Content        string    `json:"content" dynamodbav:"content"`
	CreatedAt      time.Time `json:"createdAt" dynamodbav:"createdAt"`
	TokenCount     int       `json:"tokenCount,omitempty" dynamodbav:"tokenCount,omitempty"`
}

// Conversation represents a chat conversation
type Conversation struct {
	ID           string    `json:"id" dynamodbav:"id"`
	UserID       string    `json:"userId" dynamodbav:"userId"`
	BotID        string    `json:"botId" dynamodbav:"botId"`
	Title        string    `json:"title" dynamodbav:"title"`
	CreatedAt    time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	MessageCount int       `json:"messageCount,omitempty" dynamodbav:"messageCount,omitempty"`
}

// ConversationWithMessages represents a conversation with its messages
type ConversationWithMessages struct {
	Conversation Conversation `json:"conversation"`
	Messages     []Message    `json:"messages"`
}

// ChatMessageInput represents a single message in the chat input
type ChatMessageInput struct {
	Role    string `json:"role" validate:"required,oneof=user assistant system"`
	Content string `json:"content" validate:"required"`
}

// ChatInput represents the input for generating a chat response
type ChatInput struct {
	ConversationID string             `json:"conversationId,omitempty"`
	BotID          string             `json:"botId" validate:"required"`
	Messages       []ChatMessageInput `json:"messages" validate:"required,min=1"`
	SystemPrompt   string             `json:"systemPrompt,omitempty"`
	ModelID        string             `json:"modelId,omitempty"`
	MaxTokens      int                `json:"maxTokens,omitempty"`
	Temperature    float64            `json:"temperature,omitempty"`
	Stream         bool               `json:"stream,omitempty"`
}

// ChatResponse represents the response from a chat request
type ChatResponse struct {
	Content        string `json:"content,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
	MessageID      string `json:"messageId,omitempty"`
	FinishReason   string `json:"finishReason,omitempty"`
	TokenCount     int    `json:"tokenCount,omitempty"`
}

// StreamChunk represents a chunk of streaming data
type StreamChunk struct {
	Type    string `json:"type"` // "content", "start", "end", "error"
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ConversationsResponse represents a paginated list of conversations
type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
	Count         int            `json:"count"`
	Total         int            `json:"total"`
	NextToken     string         `json:"nextToken,omitempty"`
	RequestID     string         `json:"requestId,omitempty"`
}