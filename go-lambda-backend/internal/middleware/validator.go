package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
)

// Validator is a middleware for validating request payloads
type Validator struct {
	validate *validator.Validate
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	v := validator.New()
	
	// Register custom validation functions if needed
	// Example: v.RegisterValidation("custom_rule", customValidationFunc)
	
	return &Validator{validate: v}
}

// ValidateRequest validates a request body against a model
func (v *Validator) ValidateRequest(request events.APIGatewayProxyRequest, model interface{}) error {
	// Check if the model is a pointer
	if reflect.ValueOf(model).Kind() != reflect.Ptr {
		return errors.New("model must be a pointer")
	}

	// Parse the request body
	if err := json.Unmarshal([]byte(request.Body), model); err != nil {
		return fmt.Errorf("invalid request format: %s", err)
	}

	// Validate the model
	if err := v.validate.Struct(model); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorMessages := make([]string, len(validationErrors))
			for i, e := range validationErrors {
				errorMessages[i] = fmt.Sprintf("field '%s' failed validation: %s", e.Field(), e.Tag())
			}
			return fmt.Errorf("validation error: %s", strings.Join(errorMessages, ", "))
		}
		return err
	}

	return nil
}

// Validate validates a struct
func (v *Validator) Validate(model interface{}) error {
	if err := v.validate.Struct(model); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorMessages := make([]string, len(validationErrors))
			for i, e := range validationErrors {
				errorMessages[i] = fmt.Sprintf("field '%s' failed validation: %s", e.Field(), e.Tag())
			}
			return fmt.Errorf("validation error: %s", strings.Join(errorMessages, ", "))
		}
		return err
	}
	return nil
}

// WithUserContext adds user information to the context
func WithUserContext(ctx context.Context, request events.APIGatewayProxyRequest) context.Context {
	// Extract user ID from the request (e.g., from JWT claims or request headers)
	// This is a simplified example - in a real application, you'd verify JWT tokens, etc.
	userID := "anonymous"
	
	if authHeader, ok := request.Headers["Authorization"]; ok {
		// Parse Authorization header (simplified)
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			// In a real app, you'd verify and decode the JWT token
			// For this example, let's just use a placeholder
			userID = "user-from-token"
		}
	}
	
	// Add the user ID to the context
	return context.WithValue(ctx, "userID", userID)
}