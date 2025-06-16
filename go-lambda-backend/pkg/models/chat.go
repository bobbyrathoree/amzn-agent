package models

import (
	"time"
	"encoding/json"
)

// ContentType represents different types of message content
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
	ContentTypeReasoning  ContentType = "reasoning"
)

// MessageContent represents different types of content in a message
type MessageContent struct {
	Type     ContentType            `json:"type" dynamodbav:"type"`
	Text     string                 `json:"text,omitempty" dynamodbav:"text,omitempty"`
	ImageURL string                 `json:"imageUrl,omitempty" dynamodbav:"imageUrl,omitempty"`
	ToolUse  *ToolUseContent        `json:"toolUse,omitempty" dynamodbav:"toolUse,omitempty"`
	ToolResult *ToolResultContent   `json:"toolResult,omitempty" dynamodbav:"toolResult,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty" dynamodbav:"metadata,omitempty"`
}

// ToolUseContent represents tool usage in messages
type ToolUseContent struct {
	ToolUseID string                 `json:"toolUseId" dynamodbav:"toolUseId"`
	Name      string                 `json:"name" dynamodbav:"name"`
	Input     map[string]interface{} `json:"input" dynamodbav:"input"`
}

// ToolResultContent represents tool execution results
type ToolResultContent struct {
	ToolUseID string `json:"toolUseId" dynamodbav:"toolUseId"`
	Content   string `json:"content" dynamodbav:"content"`
	IsError   bool   `json:"isError,omitempty" dynamodbav:"isError,omitempty"`
}

// Message represents a chat message with conversation tree support
type Message struct {
	ID             string            `json:"id" dynamodbav:"id"`
	ConversationID string            `json:"conversationId" dynamodbav:"conversationId"`
	Role           string            `json:"role" dynamodbav:"role"` // user, assistant, system, instruction
	Content        []MessageContent  `json:"content" dynamodbav:"content"`
	Model          string            `json:"model,omitempty" dynamodbav:"model,omitempty"`
	ParentID       *string           `json:"parentId,omitempty" dynamodbav:"parentId,omitempty"`
	Children       []string          `json:"children,omitempty" dynamodbav:"children,omitempty"`
	CreatedAt      time.Time         `json:"createdAt" dynamodbav:"createdAt"`
	TokenCount     int               `json:"tokenCount,omitempty" dynamodbav:"tokenCount,omitempty"`
	Feedback       *MessageFeedback  `json:"feedback,omitempty" dynamodbav:"feedback,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty" dynamodbav:"metadata,omitempty"`
}

// MessageFeedback represents user feedback on messages
type MessageFeedback struct {
	ThumbsUp   *bool  `json:"thumbsUp,omitempty" dynamodbav:"thumbsUp,omitempty"`
	ThumbsDown *bool  `json:"thumbsDown,omitempty" dynamodbav:"thumbsDown,omitempty"`
	Comment    string `json:"comment,omitempty" dynamodbav:"comment,omitempty"`
	CreatedAt  time.Time `json:"createdAt,omitempty" dynamodbav:"createdAt,omitempty"`
}

// Conversation represents a chat conversation with enhanced features
type Conversation struct {
	ID               string                 `json:"id" dynamodbav:"id"`
	UserID           string                 `json:"userId" dynamodbav:"userId"`
	BotID            *string                `json:"botId,omitempty" dynamodbav:"botId,omitempty"`
	Title            string                 `json:"title" dynamodbav:"title"`
	SessionModelID   *string                `json:"sessionModelId,omitempty" dynamodbav:"sessionModelId,omitempty"` // Override model for this conversation
	MessageMap       map[string]*Message    `json:"messageMap" dynamodbav:"messageMap"`
	LastMessageID    string                 `json:"lastMessageId" dynamodbav:"lastMessageId"`
	ShouldContinue   bool                   `json:"shouldContinue,omitempty" dynamodbav:"shouldContinue,omitempty"`
	CreatedAt        time.Time              `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt" dynamodbav:"updatedAt"`
	TotalTokens      int                    `json:"totalTokens,omitempty" dynamodbav:"totalTokens,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty" dynamodbav:"metadata,omitempty"`
	
	// Storage optimization flags
	IsLargeConversation bool   `json:"isLargeConversation,omitempty" dynamodbav:"isLargeConversation,omitempty"`
	S3Location         string `json:"s3Location,omitempty" dynamodbav:"s3Location,omitempty"`
}

// ConversationMeta represents lightweight conversation metadata for listings
type ConversationMeta struct {
	ID        string    `json:"id" dynamodbav:"id"`
	UserID    string    `json:"userId" dynamodbav:"userId"`
	BotID     *string   `json:"botId,omitempty" dynamodbav:"botId,omitempty"`
	Title     string    `json:"title" dynamodbav:"title"`
	SessionModelID *string `json:"sessionModelId,omitempty" dynamodbav:"sessionModelId,omitempty"` // Override model for this conversation
	CreatedAt time.Time `json:"createdAt" dynamodbav:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" dynamodbav:"updatedAt"`
	MessageCount int    `json:"messageCount" dynamodbav:"messageCount"`
	LastMessage  string `json:"lastMessage,omitempty" dynamodbav:"lastMessage,omitempty"`
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
	Conversations []ConversationMeta `json:"conversations"`
	Count         int                `json:"count"`
	Total         int                `json:"total"`
	NextToken     string             `json:"nextToken,omitempty"`
	RequestID     string             `json:"requestId,omitempty"`
}

// Helper methods for Message

// AddChild adds a child message ID to this message
func (m *Message) AddChild(childID string) {
	if m.Children == nil {
		m.Children = make([]string, 0)
	}
	for _, existing := range m.Children {
		if existing == childID {
			return // Already exists
		}
	}
	m.Children = append(m.Children, childID)
}

// GetTextContent returns the first text content from message
func (m *Message) GetTextContent() string {
	for _, content := range m.Content {
		if content.Type == ContentTypeText {
			return content.Text
		}
	}
	return ""
}

// SetTextContent sets the message content to a single text block
func (m *Message) SetTextContent(text string) {
	m.Content = []MessageContent{
		{
			Type: ContentTypeText,
			Text: text,
		},
	}
}

// Helper methods for Conversation

// GetMessage retrieves a message from the conversation by ID
func (c *Conversation) GetMessage(messageID string) *Message {
	if c.MessageMap == nil {
		return nil
	}
	return c.MessageMap[messageID]
}

// AddMessage adds a message to the conversation
func (c *Conversation) AddMessage(message *Message) {
	if c.MessageMap == nil {
		c.MessageMap = make(map[string]*Message)
	}
	c.MessageMap[message.ID] = message
	c.LastMessageID = message.ID
	c.UpdatedAt = time.Now()
	
	// Update parent's children list
	if message.ParentID != nil {
		if parent := c.GetMessage(*message.ParentID); parent != nil {
			parent.AddChild(message.ID)
		}
	}
}

// TraceToRoot traces from a message back to the conversation root
func (c *Conversation) TraceToRoot(messageID string) []*Message {
	var path []*Message
	current := c.GetMessage(messageID)
	
	for current != nil {
		path = append([]*Message{current}, path...) // Prepend to maintain order
		if current.ParentID == nil {
			break
		}
		current = c.GetMessage(*current.ParentID)
	}
	
	return path
}

// GetLinearHistory gets messages in linear conversation order (root to last)
func (c *Conversation) GetLinearHistory() []*Message {
	return c.TraceToRoot(c.LastMessageID)
}

// EstimateSize estimates the conversation size for storage optimization
func (c *Conversation) EstimateSize() int {
	data, _ := json.Marshal(c)
	return len(data)
}

// ShouldStoreInS3 determines if conversation should be stored in S3
func (c *Conversation) ShouldStoreInS3() bool {
	return c.EstimateSize() > 300*1024 // 300KB threshold like bedrock-chat
}

// ToMeta converts conversation to lightweight metadata
func (c *Conversation) ToMeta() ConversationMeta {
	lastMessage := ""
	if lastMsg := c.GetMessage(c.LastMessageID); lastMsg != nil {
		lastMessage = lastMsg.GetTextContent()
		if len(lastMessage) > 100 {
			lastMessage = lastMessage[:100] + "..."
		}
	}
	
	return ConversationMeta{
		ID:             c.ID,
		UserID:         c.UserID,
		BotID:          c.BotID,
		Title:          c.Title,
		SessionModelID: c.SessionModelID,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
		MessageCount:   len(c.MessageMap),
		LastMessage:    lastMessage,
	}
}