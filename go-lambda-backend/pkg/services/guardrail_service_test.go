package services

import (
	"testing"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

func TestGuardrailService_BuildGuardrailConfig(t *testing.T) {
	service := NewGuardrailService()

	t.Run("No guardrails configured", func(t *testing.T) {
		bot := &models.Bot{
			ID: "test-bot",
		}

		config, err := service.BuildGuardrailConfig(bot, false)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if config != nil {
			t.Errorf("Expected nil config, got %v", config)
		}
	})

	t.Run("Valid guardrail configuration", func(t *testing.T) {
		arn := "arn:aws:bedrock:us-east-1:123456789012:guardrail/test"
		version := "1"
		bot := &models.Bot{
			ID:               "test-bot",
			GuardrailArn:     &arn,
			GuardrailVersion: &version,
		}

		config, err := service.BuildGuardrailConfig(bot, false)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if config == nil {
			t.Errorf("Expected config, got nil")
			return
		}
		if *config.GuardrailIdentifier != arn {
			t.Errorf("Expected ARN %s, got %s", arn, *config.GuardrailIdentifier)
		}
		if *config.GuardrailVersion != version {
			t.Errorf("Expected version %s, got %s", version, *config.GuardrailVersion)
		}
	})

	t.Run("Invalid ARN format", func(t *testing.T) {
		arn := "invalid-arn"
		version := "1"
		bot := &models.Bot{
			ID:               "test-bot",
			GuardrailArn:     &arn,
			GuardrailVersion: &version,
		}

		config, err := service.BuildGuardrailConfig(bot, false)
		if err == nil {
			t.Errorf("Expected error for invalid ARN, got nil")
		}
		if config != nil {
			t.Errorf("Expected nil config for invalid ARN, got %v", config)
		}
	})
}

func TestGuardrailService_ValidateGuardrailConfiguration(t *testing.T) {
	service := NewGuardrailService()

	t.Run("No guardrails", func(t *testing.T) {
		bot := &models.Bot{ID: "test-bot"}
		err := service.ValidateGuardrailConfiguration(bot)
		if err != nil {
			t.Errorf("Expected no error for bot without guardrails, got %v", err)
		}
	})

	t.Run("Valid configuration", func(t *testing.T) {
		arn := "arn:aws:bedrock:us-east-1:123456789012:guardrail/test"
		version := "DRAFT"
		bot := &models.Bot{
			ID:               "test-bot",
			GuardrailArn:     &arn,
			GuardrailVersion: &version,
		}
		err := service.ValidateGuardrailConfiguration(bot)
		if err != nil {
			t.Errorf("Expected no error for valid configuration, got %v", err)
		}
	})

	t.Run("Numeric version", func(t *testing.T) {
		arn := "arn:aws:bedrock:us-east-1:123456789012:guardrail/test"
		version := "123"
		bot := &models.Bot{
			ID:               "test-bot",
			GuardrailArn:     &arn,
			GuardrailVersion: &version,
		}
		err := service.ValidateGuardrailConfiguration(bot)
		if err != nil {
			t.Errorf("Expected no error for numeric version, got %v", err)
		}
	})

	t.Run("Invalid ARN", func(t *testing.T) {
		arn := "invalid-arn"
		version := "1"
		bot := &models.Bot{
			ID:               "test-bot",
			GuardrailArn:     &arn,
			GuardrailVersion: &version,
		}
		err := service.ValidateGuardrailConfiguration(bot)
		if err == nil {
			t.Errorf("Expected error for invalid ARN, got nil")
		}
	})
}