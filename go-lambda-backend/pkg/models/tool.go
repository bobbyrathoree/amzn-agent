package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Tool represents a unified tool definition
type Tool struct {
	ID              string                    `json:"id" dynamodb:"id"`
	Name            string                    `json:"name" dynamodb:"name"`
	Description     string                    `json:"description" dynamodb:"description"`
	Category        string                    `json:"category" dynamodb:"category"`
	Version         string                    `json:"version" dynamodb:"version"`
	Author          string                    `json:"author,omitempty" dynamodb:"author,omitempty"`
	Icon            string                    `json:"icon,omitempty" dynamodb:"icon,omitempty"`
	Featured        bool                      `json:"featured" dynamodb:"featured"`
	Tags            []string                  `json:"tags" dynamodb:"tags"`
	Keywords        []string                  `json:"keywords" dynamodb:"keywords"`
	Capabilities    []ToolCapability          `json:"capabilities" dynamodb:"capabilities"`
	APIRequirements map[string]APIRequirement `json:"apiRequirements,omitempty" dynamodb:"apiRequirements,omitempty"`
	Examples        []ToolExample             `json:"examples,omitempty" dynamodb:"examples,omitempty"`
	CreatedAt       time.Time                 `json:"createdAt" dynamodb:"createdAt"`
	UpdatedAt       time.Time                 `json:"updatedAt" dynamodb:"updatedAt"`
}

// ToolCapability represents a specific capability of a tool
type ToolCapability struct {
	Name         string            `json:"name" dynamodb:"name"`
	Description  string            `json:"description" dynamodb:"description"`
	Parameters   []ParameterDef    `json:"parameters" dynamodb:"parameters"`
	Examples     []string          `json:"examples,omitempty" dynamodb:"examples,omitempty"`
	TimeEstimate *int              `json:"timeEstimate,omitempty" dynamodb:"timeEstimate,omitempty"` // milliseconds
	CostEstimate *CostEstimate     `json:"costEstimate,omitempty" dynamodb:"costEstimate,omitempty"`
	InputSchema  string            `json:"inputSchema,omitempty" dynamodb:"inputSchema,omitempty"`   // JSON Schema
	OutputSchema string            `json:"outputSchema,omitempty" dynamodb:"outputSchema,omitempty"` // JSON Schema
}

// ParameterDef defines a parameter for a tool capability
type ParameterDef struct {
	Name        string      `json:"name" dynamodb:"name"`
	Type        string      `json:"type" dynamodb:"type"` // string, number, boolean, array, object
	Required    bool        `json:"required" dynamodb:"required"`
	Description string      `json:"description,omitempty" dynamodb:"description,omitempty"`
	Default     interface{} `json:"default,omitempty" dynamodb:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty" dynamodb:"enum,omitempty"`
	MinLength   *int        `json:"minLength,omitempty" dynamodb:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty" dynamodb:"maxLength,omitempty"`
}

// APIRequirement defines API key requirements for a tool
type APIRequirement struct {
	Required         bool     `json:"required" dynamodb:"required"`
	KeyName          string   `json:"keyName" dynamodb:"keyName"`
	Description      string   `json:"description" dynamodb:"description"`
	SignupURL        string   `json:"signupUrl,omitempty" dynamodb:"signupUrl,omitempty"`
	TestEndpoint     string   `json:"testEndpoint,omitempty" dynamodb:"testEndpoint,omitempty"`
	PricingInfo      string   `json:"pricingInfo,omitempty" dynamodb:"pricingInfo,omitempty"`
	SetupInstructions []string `json:"setupInstructions,omitempty" dynamodb:"setupInstructions,omitempty"`
}

// CostEstimate represents the estimated cost of executing a tool
type CostEstimate struct {
	Credits     int    `json:"credits" dynamodb:"credits"`
	Description string `json:"description,omitempty" dynamodb:"description,omitempty"`
}

// ToolExample provides usage examples for a tool
type ToolExample struct {
	Input       string `json:"input" dynamodb:"input"`
	Output      string `json:"output" dynamodb:"output"`
	Description string `json:"description,omitempty" dynamodb:"description,omitempty"`
}

// ToolExecutionRequest represents a request to execute a tool
type ToolExecutionRequest struct {
	ToolID         string                 `json:"toolId"`
	Capability     string                 `json:"capability"`
	Input          map[string]interface{} `json:"input"`
	ConversationID string                 `json:"conversationId,omitempty"`
	Context        ExecutionContext       `json:"context"`
	Stream         bool                   `json:"stream,omitempty"`
}

// ExecutionContext provides context for tool execution
type ExecutionContext struct {
	UserID      string            `json:"userId"`
	BotID       string            `json:"botId,omitempty"`
	Permissions []string          `json:"permissions"`
	RequestID   string            `json:"requestId"`
	Timestamp   time.Time         `json:"timestamp"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success        bool                   `json:"success"`
	Data           interface{}            `json:"data,omitempty"`
	Error          string                 `json:"error,omitempty"`
	ErrorCode      string                 `json:"errorCode,omitempty"`
	ExecutionTime  int64                  `json:"executionTime"` // milliseconds
	TokensUsed     int                    `json:"tokensUsed,omitempty"`
	CostIncurred   float64                `json:"costIncurred,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	ToolVersion    string                 `json:"toolVersion,omitempty"`
	ExecutedAt     time.Time              `json:"executedAt"`
	APIKeysUsed    []string               `json:"apiKeysUsed,omitempty"` // Service IDs of API keys used
}

// ToolProgress represents progress updates for streaming tool execution
type ToolProgress struct {
	Stage       string                 `json:"stage"`
	Progress    int                    `json:"progress"` // 0-100
	Message     string                 `json:"message"`
	Data        interface{}            `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Complete    bool                   `json:"complete"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ToolSuggestion represents an AI-suggested tool for a conversation
type ToolSuggestion struct {
	ToolID      string                 `json:"toolId"`
	Capability  string                 `json:"capability"`
	Reason      string                 `json:"reason"`
	Confidence  float64                `json:"confidence"` // 0.0-1.0
	Input       map[string]interface{} `json:"input,omitempty"`
	Priority    int                    `json:"priority"` // 1-5, higher is more important
}

// ToolExecutionMessage represents a tool execution stored as a message
type ToolExecutionMessage struct {
	MessageID      string        `json:"messageId"`
	ConversationID string        `json:"conversationId"`
	UserID         string        `json:"userId"`
	ToolID         string        `json:"toolId"`
	Capability     string        `json:"capability"`
	Input          interface{}   `json:"input"`
	Result         ToolResult    `json:"result"`
	ExecutedAt     time.Time     `json:"executedAt"`
	Duration       int64         `json:"duration"` // milliseconds
}

// ToolRegistry manages available tools
type ToolRegistryEntry struct {
	Tool       Tool      `json:"tool"`
	Enabled    bool      `json:"enabled"`
	LastUsed   time.Time `json:"lastUsed,omitempty"`
	UsageCount int64     `json:"usageCount"`
}

// Validation methods

func (t *Tool) Validate() error {
	if t.ID == "" {
		return NewValidationError("tool ID is required")
	}
	if t.Name == "" {
		return NewValidationError("tool name is required")
	}
	if len(t.Capabilities) == 0 {
		return NewValidationError("tool must have at least one capability")
	}
	
	// Validate capabilities
	for i, cap := range t.Capabilities {
		if cap.Name == "" {
			return NewValidationError("capability name is required at index %d", i)
		}
		if cap.Description == "" {
			return NewValidationError("capability description is required for %s", cap.Name)
		}
	}
	
	return nil
}

func (r *ToolExecutionRequest) Validate() error {
	if r.ToolID == "" {
		return NewValidationError("toolId is required")
	}
	if r.Capability == "" {
		return NewValidationError("capability is required")
	}
	if r.Context.UserID == "" {
		return NewValidationError("context.userId is required")
	}
	if r.Context.RequestID == "" {
		return NewValidationError("context.requestId is required")
	}
	
	return nil
}

// Helper functions

func (t *Tool) GetCapability(name string) *ToolCapability {
	for _, cap := range t.Capabilities {
		if cap.Name == name {
			return &cap
		}
	}
	return nil
}

func (t *Tool) HasAPIRequirements() bool {
	return len(t.APIRequirements) > 0
}

func (t *Tool) RequiredServices() []string {
	var services []string
	for serviceID, req := range t.APIRequirements {
		if req.Required {
			services = append(services, serviceID)
		}
	}
	return services
}

func (t *Tool) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

func (r *ToolResult) IsSuccess() bool {
	return r.Success && r.Error == ""
}

func (r *ToolResult) HasError() bool {
	return !r.Success || r.Error != ""
}

// Constants for tool categories
const (
	ToolCategoryInformation   = "information"
	ToolCategoryCreative      = "creative"
	ToolCategoryBusiness      = "business"
	ToolCategoryDevelopment   = "development"
	ToolCategoryResearch      = "research"
	ToolCategoryAutomation    = "automation"
	ToolCategoryCommunication = "communication"
)

// Constants for execution stages
const (
	StageInitializing = "initializing"
	StageValidating   = "validating"
	StageExecuting    = "executing"
	StageProcessing   = "processing"
	StageCompleting   = "completing"
	StageComplete     = "complete"
	StageError        = "error"
)

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error with formatted message
func NewValidationError(format string, args ...interface{}) error {
	return ValidationError{
		Message: fmt.Sprintf(format, args...),
	}
}