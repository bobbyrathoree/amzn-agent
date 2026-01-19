package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// VaultService handles secure API key storage and retrieval
type VaultService struct {
	dynamoClient *dynamodb.Client
	tableName    string
}

// VaultEntry represents an encrypted API key stored in the vault
type VaultEntry struct {
	UserID       string    `json:"userId" dynamodb:"UserID"`
	ServiceID    string    `json:"serviceId" dynamodb:"ServiceID"`
	KeyName      string    `json:"keyName" dynamodb:"KeyName"`
	EncryptedKey string    `json:"encryptedKey" dynamodb:"EncryptedKey"`
	Nonce        string    `json:"nonce" dynamodb:"Nonce"`
	Salt         string    `json:"salt" dynamodb:"Salt"`
	CreatedAt    time.Time `json:"createdAt" dynamodb:"CreatedAt"`
	UpdatedAt    time.Time `json:"updatedAt" dynamodb:"UpdatedAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty" dynamodb:"ExpiresAt,omitempty"`
	LastUsed     *time.Time `json:"lastUsed,omitempty" dynamodb:"LastUsed,omitempty"`
	UsageCount   int       `json:"usageCount" dynamodb:"UsageCount"`
	IsActive     bool      `json:"isActive" dynamodb:"IsActive"`
}

// UserSalt represents a user's master salt for key derivation
type UserSalt struct {
	UserID     string    `json:"userId" dynamodb:"UserID"`
	RecordType string    `json:"recordType" dynamodb:"ServiceID"` // Uses ServiceID as sort key, value is "MASTER_SALT"
	Salt       string    `json:"salt" dynamodb:"Salt"`
	CreatedAt  time.Time `json:"createdAt" dynamodb:"CreatedAt"`
	UpdatedAt  time.Time `json:"updatedAt" dynamodb:"UpdatedAt"`
	Version    int       `json:"version" dynamodb:"Version"`
	IsActive   bool      `json:"isActive" dynamodb:"IsActive"`
}

// VaultStatus represents the status of a user's vault
type VaultStatus struct {
	UserID        string    `json:"userId"`
	IsUnlocked    bool      `json:"isUnlocked"`
	KeyCount      int       `json:"keyCount"`
	LastAccessed  *time.Time `json:"lastAccessed,omitempty"`
	UnlockedUntil *time.Time `json:"unlockedUntil,omitempty"`
}

// SupportedService represents a service that can have API keys stored in the vault
type SupportedService struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Website     string   `json:"website"`
	Pricing     string   `json:"pricing"`
	Icon        string   `json:"icon"`
	Color       string   `json:"color"`
	IsActive    bool     `json:"isActive"`
	SetupSteps  []string `json:"setupSteps"`
}

// ServiceCategory represents a category of services
type ServiceCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// NewVaultService creates a new vault service
func NewVaultService(dynamoClient *dynamodb.Client, tableName string) *VaultService {
	return &VaultService{
		dynamoClient: dynamoClient,
		tableName:    tableName,
	}
}

// StoreAPIKey stores an encrypted API key in the vault
func (s *VaultService) StoreAPIKey(ctx context.Context, userID, serviceID, keyName, encryptedKey, nonce, salt string) error {
	now := time.Now()
	
	entry := VaultEntry{
		UserID:       userID,
		ServiceID:    serviceID,
		KeyName:      keyName,
		EncryptedKey: encryptedKey,
		Nonce:        nonce,
		Salt:         salt,
		CreatedAt:    now,
		UpdatedAt:    now,
		UsageCount:   0,
		IsActive:     true,
	}

	// Convert to DynamoDB item
	item, err := attributevalue.MarshalMap(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal vault entry: %w", err)
	}

	// Put item in DynamoDB
	_, err = s.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
		// Use condition to prevent overwriting existing keys accidentally
		ConditionExpression: aws.String("attribute_not_exists(UserID) AND attribute_not_exists(ServiceID)"),
	})

	if err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	log.Printf("API key stored for user %s, service %s", userID, serviceID)
	return nil
}

// GetAPIKey retrieves an encrypted API key from the vault
func (s *VaultService) GetAPIKey(ctx context.Context, userID, serviceID string) (string, error) {
	// Get item from DynamoDB
	result, err := s.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: serviceID},
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to get API key: %w", err)
	}

	if result.Item == nil {
		return "", fmt.Errorf("API key not found for service %s", serviceID)
	}

	var entry VaultEntry
	err = attributevalue.UnmarshalMap(result.Item, &entry)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal vault entry: %w", err)
	}

	if !entry.IsActive {
		return "", fmt.Errorf("API key for service %s is disabled", serviceID)
	}

	// Update usage statistics
	go s.updateUsageStats(ctx, userID, serviceID)

	// Return the encrypted key (frontend will decrypt it)
	return entry.EncryptedKey, nil
}

// GetVaultEntry retrieves the complete vault entry for a service
func (s *VaultService) GetVaultEntry(ctx context.Context, userID, serviceID string) (*VaultEntry, error) {
	result, err := s.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: serviceID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get vault entry: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("vault entry not found for service %s", serviceID)
	}

	var entry VaultEntry
	err = attributevalue.UnmarshalMap(result.Item, &entry)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal vault entry: %w", err)
	}

	return &entry, nil
}

// UpdateAPIKey updates an existing API key in the vault
func (s *VaultService) UpdateAPIKey(ctx context.Context, userID, serviceID, encryptedKey, nonce string) error {
	now := time.Now()

	// Update the item
	_, err := s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: serviceID},
		},
		UpdateExpression: aws.String("SET EncryptedKey = :key, Nonce = :nonce, UpdatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":key":     &types.AttributeValueMemberS{Value: encryptedKey},
			":nonce":   &types.AttributeValueMemberS{Value: nonce},
			":updated": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(UserID) AND attribute_exists(ServiceID)"),
	})

	if err != nil {
		// Check if the error is due to the key not existing
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {
			return fmt.Errorf("API key not found for service %s", serviceID)
		}
		return fmt.Errorf("failed to update API key: %w", err)
	}

	return nil
}

// DeleteAPIKey removes an API key from the vault
func (s *VaultService) DeleteAPIKey(ctx context.Context, userID, serviceID string) error {
	_, err := s.dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: serviceID},
		},
		ConditionExpression: aws.String("attribute_exists(UserID) AND attribute_exists(ServiceID)"),
	})

	if err != nil {
		// Check if the error is due to the key not existing
		var ccfe *types.ConditionalCheckFailedException
		if errors.As(err, &ccfe) {
			return fmt.Errorf("API key not found for service %s", serviceID)
		}
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	return nil
}

// ListAPIKeys lists all API keys for a user
func (s *VaultService) ListAPIKeys(ctx context.Context, userID string) ([]VaultEntry, error) {
	result, err := s.dynamoClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("UserID = :userId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberS{Value: userID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	var entries []VaultEntry
	for _, item := range result.Items {
		var entry VaultEntry
		err := attributevalue.UnmarshalMap(item, &entry)
		if err != nil {
			log.Printf("Failed to unmarshal vault entry: %v", err)
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// GetVaultStatus returns the status of a user's vault
func (s *VaultService) GetVaultStatus(ctx context.Context, userID string) (*VaultStatus, error) {
	entries, err := s.ListAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault status: %w", err)
	}

	status := &VaultStatus{
		UserID:     userID,
		IsUnlocked: false, // This would be managed by session/cache in production
		KeyCount:   len(entries),
	}

	// Find the most recent access
	for _, entry := range entries {
		if entry.LastUsed != nil {
			if status.LastAccessed == nil || entry.LastUsed.After(*status.LastAccessed) {
				status.LastAccessed = entry.LastUsed
			}
		}
	}

	return status, nil
}

// TestAPIKey tests if an API key is valid for a service
func (s *VaultService) TestAPIKey(ctx context.Context, userID, serviceID string) (bool, error) {
	entry, err := s.GetVaultEntry(ctx, userID, serviceID)
	if err != nil {
		return false, err
	}

	if !entry.IsActive {
		return false, fmt.Errorf("API key is disabled for service %s", serviceID)
	}

	// For security, we don't decrypt the API key in the backend
	// Instead, we return true if the key exists and is active
	// The actual testing should be done on the frontend with the decrypted key
	log.Printf("API key test requested for service %s (key exists and is active)", serviceID)
	
	// Update usage stats
	s.updateUsageStats(ctx, userID, serviceID)
	
	return true, nil
}

// HasAPIKey checks if a user has an API key for a service
func (s *VaultService) HasAPIKey(ctx context.Context, userID, serviceID string) (bool, error) {
	_, err := s.GetVaultEntry(ctx, userID, serviceID)
	if err != nil {
		if fmt.Sprintf("%v", err) == fmt.Sprintf("vault entry not found for service %s", serviceID) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// updateUsageStats updates the usage statistics for an API key
func (s *VaultService) updateUsageStats(ctx context.Context, userID, serviceID string) {
	now := time.Now()
	
	_, err := s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"userId":    &types.AttributeValueMemberS{Value: userID},
			"serviceId": &types.AttributeValueMemberS{Value: serviceID},
		},
		UpdateExpression: aws.String("SET lastUsed = :lastUsed, usageCount = usageCount + :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":lastUsed": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":inc":      &types.AttributeValueMemberN{Value: "1"},
		},
	})

	if err != nil {
		log.Printf("Failed to update usage stats for %s/%s: %v", userID, serviceID, err)
	}
}

// GetAPIKeyMetadata returns metadata about an API key without the encrypted key
func (s *VaultService) GetAPIKeyMetadata(ctx context.Context, userID, serviceID string) (map[string]interface{}, error) {
	entry, err := s.GetVaultEntry(ctx, userID, serviceID)
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"serviceId":   entry.ServiceID,
		"keyName":     entry.KeyName,
		"createdAt":   entry.CreatedAt,
		"updatedAt":   entry.UpdatedAt,
		"lastUsed":    entry.LastUsed,
		"usageCount":  entry.UsageCount,
		"isActive":    entry.IsActive,
	}

	if entry.ExpiresAt != nil {
		metadata["expiresAt"] = entry.ExpiresAt
	}

	return metadata, nil
}

// EnableAPIKey enables a disabled API key
func (s *VaultService) EnableAPIKey(ctx context.Context, userID, serviceID string) error {
	_, err := s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"userId":    &types.AttributeValueMemberS{Value: userID},
			"serviceId": &types.AttributeValueMemberS{Value: serviceID},
		},
		UpdateExpression: aws.String("SET isActive = :active, updatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":active":  &types.AttributeValueMemberBOOL{Value: true},
			":updated": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(userId) AND attribute_exists(serviceId)"),
	})

	return err
}

// DisableAPIKey disables an API key without deleting it
func (s *VaultService) DisableAPIKey(ctx context.Context, userID, serviceID string) error {
	_, err := s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"userId":    &types.AttributeValueMemberS{Value: userID},
			"serviceId": &types.AttributeValueMemberS{Value: serviceID},
		},
		UpdateExpression: aws.String("SET isActive = :active, updatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":active":  &types.AttributeValueMemberBOOL{Value: false},
			":updated": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(userId) AND attribute_exists(serviceId)"),
	})

	return err
}

// GetSupportedServices returns the list of supported services for API key storage
func (s *VaultService) GetSupportedServices() []SupportedService {
	return []SupportedService{
		{
			ID:          "google-search",
			Name:        "Google Custom Search",
			Category:    "search",
			Description: "High-quality web search results from Google",
			Website:     "https://developers.google.com/custom-search",
			Pricing:     "Free: 100/day, Paid: $5/1000 queries",
			Icon:        "🔍",
			Color:       "#4285f4",
			IsActive:    true,
			SetupSteps: []string{
				"1. Go to Google Cloud Console",
				"2. Enable Custom Search API",
				"3. Create credentials (API Key)",
				"4. Copy the API key and paste it here",
			},
		},
		{
			ID:          "openai",
			Name:        "OpenAI API",
			Category:    "ai",
			Description: "GPT models for text generation and analysis",
			Website:     "https://platform.openai.com",
			Pricing:     "Pay per token usage",
			Icon:        "🤖",
			Color:       "#10a37f",
			IsActive:    true,
			SetupSteps: []string{
				"1. Sign up at OpenAI Platform",
				"2. Go to API Keys section",
				"3. Create a new API key",
				"4. Copy the key and paste it here",
			},
		},
		{
			ID:          "anthropic",
			Name:        "Anthropic Claude",
			Category:    "ai",
			Description: "Claude models for advanced reasoning and analysis",
			Website:     "https://console.anthropic.com",
			Pricing:     "Pay per token usage",
			Icon:        "🧠",
			Color:       "#d97706",
			IsActive:    true,
			SetupSteps: []string{
				"1. Sign up at Anthropic Console",
				"2. Navigate to API Keys",
				"3. Generate a new API key",
				"4. Copy the key and paste it here",
			},
		},
		{
			ID:          "tavily",
			Name:        "Tavily Search",
			Category:    "search",
			Description: "AI-optimized search engine for agents",
			Website:     "https://tavily.com",
			Pricing:     "Free: 1000/month, Pro: $20/month",
			Icon:        "🔎",
			Color:       "#059669",
			IsActive:    true,
			SetupSteps: []string{
				"1. Sign up at Tavily.com",
				"2. Go to your dashboard",
				"3. Copy your API key",
				"4. Paste the key here",
			},
		},
		{
			ID:          "serp-api",
			Name:        "SerpAPI",
			Category:    "search",
			Description: "Google/Bing search results API",
			Website:     "https://serpapi.com",
			Pricing:     "Free: 100/month, Paid: $50/month",
			Icon:        "🐍",
			Color:       "#7c3aed",
			IsActive:    true,
			SetupSteps: []string{
				"1. Sign up at SerpAPI.com",
				"2. Go to your dashboard",
				"3. Find your API key",
				"4. Copy and paste it here",
			},
		},
		{
			ID:          "weather-api",
			Name:        "Weather API",
			Category:    "data",
			Description: "Current weather and forecasts worldwide",
			Website:     "https://weatherapi.com",
			Pricing:     "Free: 1M calls/month, Paid: $4/month",
			Icon:        "🌤️",
			Color:       "#0ea5e9",
			IsActive:    true,
			SetupSteps: []string{
				"1. Sign up at WeatherAPI.com",
				"2. Go to your dashboard",
				"3. Copy your API key",
				"4. Paste it here",
			},
		},
		{
			ID:          "newsapi",
			Name:        "News API",
			Category:    "data",
			Description: "Latest news from thousands of sources",
			Website:     "https://newsapi.org",
			Pricing:     "Free: 1000/day, Pro: $449/month",
			Icon:        "📰",
			Color:       "#ef4444",
			IsActive:    true,
			SetupSteps: []string{
				"1. Sign up at NewsAPI.org",
				"2. Get your API key from dashboard",
				"3. Copy the key",
				"4. Paste it here",
			},
		},
		{
			ID:          "github",
			Name:        "GitHub API",
			Category:    "development",
			Description: "Access GitHub repositories and data",
			Website:     "https://github.com/settings/tokens",
			Pricing:     "Free for public repos, paid for private",
			Icon:        "🐙",
			Color:       "#333333",
			IsActive:    true,
			SetupSteps: []string{
				"1. Go to GitHub Settings > Developer settings",
				"2. Generate a new personal access token",
				"3. Select appropriate scopes",
				"4. Copy the token and paste it here",
			},
		},
		{
			ID:          "slack",
			Name:        "Slack API",
			Category:    "communication",
			Description: "Send messages and interact with Slack",
			Website:     "https://api.slack.com",
			Pricing:     "Free for basic usage",
			Icon:        "💬",
			Color:       "#4a154b",
			IsActive:    true,
			SetupSteps: []string{
				"1. Go to Slack API dashboard",
				"2. Create a new app",
				"3. Generate a bot token",
				"4. Copy the token and paste it here",
			},
		},
		{
			ID:          "hubspot",
			Name:        "HubSpot API",
			Category:    "business",
			Description: "CRM and marketing automation",
			Website:     "https://developers.hubspot.com",
			Pricing:     "Free tier available",
			Icon:        "🏢",
			Color:       "#ff7a59",
			IsActive:    true,
			SetupSteps: []string{
				"1. Go to HubSpot Developer portal",
				"2. Create a private app",
				"3. Generate an access token",
				"4. Copy the token and paste it here",
			},
		},
	}
}

// GetServiceCategories returns the list of service categories
func (s *VaultService) GetServiceCategories() []ServiceCategory {
	return []ServiceCategory{
		{
			ID:          "search",
			Name:        "Search & Discovery",
			Description: "Web search and information discovery services",
			Icon:        "🔍",
		},
		{
			ID:          "ai",
			Name:        "AI & Language Models",
			Description: "AI services for text generation and analysis",
			Icon:        "🤖",
		},
		{
			ID:          "data",
			Name:        "Data & Analytics",
			Description: "Data retrieval and analytics services",
			Icon:        "📊",
		},
		{
			ID:          "development",
			Name:        "Development Tools",
			Description: "Developer tools and code management",
			Icon:        "⚙️",
		},
		{
			ID:          "communication",
			Name:        "Communication",
			Description: "Messaging and collaboration platforms",
			Icon:        "💬",
		},
		{
			ID:          "business",
			Name:        "Business & CRM",
			Description: "Business automation and customer management",
			Icon:        "🏢",
		},
	}
}

// GetSupportedServiceByID returns a specific supported service by ID
func (s *VaultService) GetSupportedServiceByID(serviceID string) *SupportedService {
	services := s.GetSupportedServices()
	for _, service := range services {
		if service.ID == serviceID {
			return &service
		}
	}
	return nil
}

// GetSupportedServicesByCategory returns services filtered by category
func (s *VaultService) GetSupportedServicesByCategory(category string) []SupportedService {
	services := s.GetSupportedServices()
	var filtered []SupportedService
	
	for _, service := range services {
		if service.Category == category {
			filtered = append(filtered, service)
		}
	}
	
	return filtered
}

// 🧂 SALT MANAGEMENT METHODS

// GetUserSalt retrieves the master salt for a user
func (s *VaultService) GetUserSalt(ctx context.Context, userID string) (*UserSalt, error) {
	log.Printf("🧂 Retrieving master salt for user: %s", userID)
	
	result, err := s.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: "MASTER_SALT"},
		},
	})
	
	if err != nil {
		log.Printf("❌ Error retrieving salt for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to retrieve user salt: %w", err)
	}
	
	if result.Item == nil {
		log.Printf("ℹ️ No salt found for user %s", userID)
		return nil, nil
	}
	
	var userSalt UserSalt
	if err := attributevalue.UnmarshalMap(result.Item, &userSalt); err != nil {
		log.Printf("❌ Error unmarshaling salt for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to unmarshal user salt: %w", err)
	}
	
	if !userSalt.IsActive {
		log.Printf("⚠️ Salt for user %s is inactive", userID)
		return nil, fmt.Errorf("user salt is inactive")
	}
	
	log.Printf("✅ Successfully retrieved salt for user %s (version %d)", userID, userSalt.Version)
	return &userSalt, nil
}

// CreateUserSalt creates a new master salt for a user
func (s *VaultService) CreateUserSalt(ctx context.Context, userID, salt string) (*UserSalt, error) {
	log.Printf("🧂 Creating new master salt for user: %s", userID)
	
	// Check if salt already exists
	existingSalt, err := s.GetUserSalt(ctx, userID)
	if err != nil && !strings.Contains(err.Error(), "salt is inactive") {
		return nil, fmt.Errorf("failed to check existing salt: %w", err)
	}
	
	if existingSalt != nil {
		log.Printf("⚠️ Salt already exists for user %s", userID)
		return existingSalt, nil
	}
	
	now := time.Now()
	userSalt := UserSalt{
		UserID:     userID,
		RecordType: "MASTER_SALT",
		Salt:       salt,
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
		IsActive:   true,
	}
	
	// Marshal the salt data
	item, err := attributevalue.MarshalMap(userSalt)
	if err != nil {
		log.Printf("❌ Error marshaling salt for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to marshal user salt: %w", err)
	}
	
	// Store in DynamoDB
	_, err = s.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
		ConditionExpression: aws.String("attribute_not_exists(UserID) AND attribute_not_exists(ServiceID)"),
	})
	
	if err != nil {
		log.Printf("❌ Error storing salt for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to store user salt: %w", err)
	}
	
	log.Printf("✅ Successfully created salt for user %s", userID)
	return &userSalt, nil
}

// UpdateUserSalt updates an existing master salt (for rotation)
func (s *VaultService) UpdateUserSalt(ctx context.Context, userID, newSalt string) (*UserSalt, error) {
	log.Printf("🔄 Rotating salt for user: %s", userID)
	
	// Get current salt
	currentSalt, err := s.GetUserSalt(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current salt: %w", err)
	}
	
	if currentSalt == nil {
		return nil, fmt.Errorf("no salt found for user %s", userID)
	}
	
	now := time.Now()
	newVersion := currentSalt.Version + 1
	
	// Update salt with new version
	_, err = s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: "MASTER_SALT"},
		},
		UpdateExpression: aws.String("SET Salt = :salt, UpdatedAt = :updated, Version = :version"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":salt":    &types.AttributeValueMemberS{Value: newSalt},
			":updated": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
			":version": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", newVersion)},
		},
		ConditionExpression: aws.String("attribute_exists(UserID) AND attribute_exists(ServiceID)"),
	})
	
	if err != nil {
		log.Printf("❌ Error updating salt for user %s: %v", userID, err)
		return nil, fmt.Errorf("failed to update user salt: %w", err)
	}
	
	// Return updated salt
	updatedSalt := &UserSalt{
		UserID:     userID,
		RecordType: "MASTER_SALT",
		Salt:       newSalt,
		CreatedAt:  currentSalt.CreatedAt,
		UpdatedAt:  now,
		Version:    newVersion,
		IsActive:   true,
	}
	
	log.Printf("✅ Successfully rotated salt for user %s (version %d)", userID, newVersion)
	return updatedSalt, nil
}

// DeactivateUserSalt deactivates a user's salt (for security)
func (s *VaultService) DeactivateUserSalt(ctx context.Context, userID string) error {
	log.Printf("🔒 Deactivating salt for user: %s", userID)
	
	now := time.Now()
	
	_, err := s.dynamoClient.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"UserID":    &types.AttributeValueMemberS{Value: userID},
			"ServiceID": &types.AttributeValueMemberS{Value: "MASTER_SALT"},
		},
		UpdateExpression: aws.String("SET IsActive = :active, UpdatedAt = :updated"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":active":  &types.AttributeValueMemberBOOL{Value: false},
			":updated": &types.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(UserID) AND attribute_exists(ServiceID)"),
	})
	
	if err != nil {
		log.Printf("❌ Error deactivating salt for user %s: %v", userID, err)
		return fmt.Errorf("failed to deactivate user salt: %w", err)
	}
	
	log.Printf("✅ Successfully deactivated salt for user %s", userID)
	return nil
}