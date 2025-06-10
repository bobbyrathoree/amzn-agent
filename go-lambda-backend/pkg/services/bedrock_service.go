package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// BedrockService provides methods to interact with AWS Bedrock
type BedrockService struct {
	client *bedrockruntime.Client
}

// NewBedrockService creates a new BedrockService instance
func NewBedrockService(cfg aws.Config) *BedrockService {
	return &BedrockService{
		client: bedrockruntime.NewFromConfig(cfg),
	}
}

// GenerateResponse generates a response from Bedrock model
func (s *BedrockService) GenerateResponse(ctx context.Context, input *models.ChatInput) (*models.ChatResponse, error) {
	// Prepare request payload based on the model
	payload, err := s.prepareRequestPayload(input)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare request payload: %w", err)
	}

	modelID := input.ModelID
	if modelID == "" {
		modelID = "anthropic.claude-3-sonnet-20240229-v1:0" // Default model
	}

	// For now, always use non-streaming to avoid complexity
	return s.invokeModel(ctx, modelID, payload)
}

// invokeModel calls Bedrock model without streaming
func (s *BedrockService) invokeModel(ctx context.Context, modelID string, payload []byte) (*models.ChatResponse, error) {
	resp, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke model: %w", err)
	}

	return s.parseResponse(resp.Body, modelID)
}

// prepareRequestPayload prepares the request payload for the model
func (s *BedrockService) prepareRequestPayload(input *models.ChatInput) ([]byte, error) {
	if strings.Contains(input.ModelID, "anthropic.claude") {
		return s.prepareAnthropicPayload(input)
	}
	
	// Default to Anthropic format
	return s.prepareAnthropicPayload(input)
}

// prepareAnthropicPayload prepares payload for Anthropic Claude models
func (s *BedrockService) prepareAnthropicPayload(input *models.ChatInput) ([]byte, error) {
	type AnthropicMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	payload := struct {
		System      string             `json:"system,omitempty"`
		Messages    []AnthropicMessage `json:"messages"`
		MaxTokens   int                `json:"max_tokens"`
		Temperature float64            `json:"temperature,omitempty"`
	}{
		System:      input.SystemPrompt,
		MaxTokens:   input.MaxTokens,
		Temperature: input.Temperature,
		Messages:    []AnthropicMessage{},
	}

	// Set defaults
	if payload.MaxTokens == 0 {
		payload.MaxTokens = 1000
	}
	if payload.Temperature == 0 {
		payload.Temperature = 0.7
	}

	// Convert messages to Anthropic format
	for _, msg := range input.Messages {
		payload.Messages = append(payload.Messages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return json.Marshal(payload)
}

// parseResponse parses the response from Bedrock
func (s *BedrockService) parseResponse(body []byte, modelID string) (*models.ChatResponse, error) {
	if strings.Contains(modelID, "anthropic.claude") {
		return s.parseAnthropicResponse(body)
	}
	
	// Default to Anthropic format
	return s.parseAnthropicResponse(body)
}

// parseAnthropicResponse parses response from Anthropic Claude models
func (s *BedrockService) parseAnthropicResponse(body []byte) (*models.ChatResponse, error) {
	type AnthropicResponse struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	var resp AnthropicResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	content := ""
	if len(resp.Content) > 0 {
		content = resp.Content[0].Text
	}

	return &models.ChatResponse{
		Content:      content,
		FinishReason: resp.StopReason,
		TokenCount:   resp.Usage.OutputTokens,
	}, nil
}

