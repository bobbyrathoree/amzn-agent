package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// GuardrailService handles AWS Bedrock Guardrails functionality
// Following the bedrock-chat reference implementation patterns
type GuardrailService struct{}

// NewGuardrailService creates a new GuardrailService
func NewGuardrailService() *GuardrailService {
	return &GuardrailService{}
}

// BuildGuardrailConfig builds guardrail configuration for Bedrock API calls
// This follows the bedrock-chat pattern from compose_args_for_converse_api line 471-481
func (s *GuardrailService) BuildGuardrailConfig(bot *models.Bot, stream bool) (*types.GuardrailConfiguration, error) {
	// Check if bot is nil (e.g., when called from intent classification)
	if bot == nil {
		return nil, nil
	}

	// Check if guardrails are configured - matching bedrock-chat's check pattern
	if bot.GuardrailArn == nil || bot.GuardrailVersion == nil {
		return nil, nil
	}
	
	if *bot.GuardrailArn == "" || *bot.GuardrailVersion == "" {
		return nil, nil
	}

	// Validate configuration first
	if err := s.ValidateGuardrailConfiguration(bot); err != nil {
		return nil, fmt.Errorf("invalid guardrail configuration: %w", err)
	}

	// Build config following bedrock-chat pattern exactly
	config := &types.GuardrailConfiguration{
		GuardrailIdentifier: aws.String(*bot.GuardrailArn),
		GuardrailVersion:    aws.String(*bot.GuardrailVersion),
		Trace:               types.GuardrailTraceEnabled,
	}

	// Note: StreamProcessingMode is not available in GuardrailConfiguration
	// It's handled at the model invocation level

	log.Printf("🛡️ Built guardrail config for bot %s: arn=%s, version=%s, stream=%v", 
		bot.ID, *bot.GuardrailArn, *bot.GuardrailVersion, stream)
	
	return config, nil
}

// ValidateGuardrailConfiguration validates the bot's guardrail configuration
func (s *GuardrailService) ValidateGuardrailConfiguration(bot *models.Bot) error {
	if bot == nil || bot.GuardrailArn == nil || bot.GuardrailVersion == nil {
		return nil // No guardrails configured, which is valid
	}

	// Validate ARN format
	if !strings.HasPrefix(*bot.GuardrailArn, "arn:aws:bedrock:") {
		return fmt.Errorf("invalid guardrail ARN format: %s", *bot.GuardrailArn)
	}

	// Validate version format (should be "DRAFT" or a number)
	version := *bot.GuardrailVersion
	if version != "DRAFT" && !isNumericVersion(version) {
		return fmt.Errorf("invalid guardrail version format: %s", version)
	}

	return nil
}

// isNumericVersion checks if a version string is numeric
func isNumericVersion(version string) bool {
	for _, char := range version {
		if char < '0' || char > '9' {
			return false
		}
	}
	return len(version) > 0
}