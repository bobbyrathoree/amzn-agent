package models

import (
	"testing"
	"time"
)

func TestBot_IsAccessibleByUser(t *testing.T) {
	tests := []struct {
		name       string
		bot        *Bot
		userID     string
		userGroups []string
		isAdmin    bool
		expected   bool
	}{
		{
			name: "Owner access",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-123",
				SharedScope: "private",
			},
			userID:   "user-123",
			isAdmin:  false,
			expected: true,
		},
		{
			name: "Admin access",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-456",
				SharedScope: "private",
			},
			userID:   "user-123",
			isAdmin:  true,
			expected: true,
		},
		{
			name: "Private bot - no access",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-456",
				SharedScope: "private",
			},
			userID:   "user-123",
			isAdmin:  false,
			expected: false,
		},
		{
			name: "Public bot - universal access",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-456",
				SharedScope: "public",
			},
			userID:   "user-123",
			isAdmin:  false,
			expected: true,
		},
		{
			name: "Partial bot - user in allowed list",
			bot: &Bot{
				ID:           "bot-1",
				OwnerUserID:  "user-456",
				SharedScope:  "partial",
				AllowedUsers: []string{"user-123", "user-789"},
			},
			userID:   "user-123",
			isAdmin:  false,
			expected: true,
		},
		{
			name: "Partial bot - user in allowed group",
			bot: &Bot{
				ID:            "bot-1",
				OwnerUserID:   "user-456",
				SharedScope:   "partial",
				AllowedGroups: []string{"developers", "admins"},
			},
			userID:     "user-123",
			userGroups: []string{"developers"},
			isAdmin:    false,
			expected:   true,
		},
		{
			name: "Partial bot - no access",
			bot: &Bot{
				ID:            "bot-1",
				OwnerUserID:   "user-456",
				SharedScope:   "partial",
				AllowedUsers:  []string{"user-789"},
				AllowedGroups: []string{"admins"},
			},
			userID:     "user-123",
			userGroups: []string{"users"},
			isAdmin:    false,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bot.IsAccessibleByUser(tt.userID, tt.userGroups, tt.isAdmin)
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestBot_IsEditableByUser(t *testing.T) {
	tests := []struct {
		name     string
		bot      *Bot
		userID   string
		isAdmin  bool
		expected bool
	}{
		{
			name: "Owner can edit",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-123",
			},
			userID:   "user-123",
			isAdmin:  false,
			expected: true,
		},
		{
			name: "Admin can edit",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-456",
			},
			userID:   "user-123",
			isAdmin:  true,
			expected: true,
		},
		{
			name: "Non-owner cannot edit",
			bot: &Bot{
				ID:          "bot-1",
				OwnerUserID: "user-456",
			},
			userID:   "user-123",
			isAdmin:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bot.IsEditableByUser(tt.userID, []string{}, tt.isAdmin)
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestBot_IsPinned(t *testing.T) {
	tests := []struct {
		name     string
		bot      *Bot
		expected bool
	}{
		{
			name: "Pinned bot",
			bot: &Bot{
				SharedScope:  "public",
				SharedStatus: "pinned@001",
			},
			expected: true,
		},
		{
			name: "Shared but not pinned",
			bot: &Bot{
				SharedScope:  "public",
				SharedStatus: "shared",
			},
			expected: false,
		},
		{
			name: "Private bot",
			bot: &Bot{
				SharedScope:  "private",
				SharedStatus: "unshared",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bot.IsPinned()
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestBot_GetPinOrder(t *testing.T) {
	tests := []struct {
		name     string
		bot      *Bot
		expected int
	}{
		{
			name: "Pinned with order",
			bot: &Bot{
				SharedScope:  "public",
				SharedStatus: "pinned@005",
			},
			expected: 5,
		},
		{
			name: "Not pinned",
			bot: &Bot{
				SharedScope:  "public",
				SharedStatus: "shared",
			},
			expected: 0,
		},
		{
			name: "Invalid pin format",
			bot: &Bot{
				SharedScope:  "public",
				SharedStatus: "pinned@abc",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bot.GetPinOrder()
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestBot_IsStackComplete(t *testing.T) {
	completeStatus := StackStatusCreateComplete
	inProgressStatus := StackStatusCreateInProgress

	tests := []struct {
		name     string
		bot      *Bot
		expected bool
	}{
		{
			name: "Stack complete",
			bot: &Bot{
				StackStatus: &completeStatus,
			},
			expected: true,
		},
		{
			name: "Stack in progress",
			bot: &Bot{
				StackStatus: &inProgressStatus,
			},
			expected: false,
		},
		{
			name: "No stack status",
			bot:      &Bot{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.bot.IsStackComplete()
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestBot_IncrementUsage(t *testing.T) {
	bot := &Bot{
		ID:         "test-bot",
		UsageCount: 5,
	}

	bot.IncrementUsage()

	if bot.UsageCount != 6 {
		t.Errorf("Expected usage count to be 6, got %d", bot.UsageCount)
	}
}

func TestBot_IsPopular(t *testing.T) {
	tests := []struct {
		name        string
		usageCount  int
		expected    bool
	}{
		{
			name:       "Popular bot",
			usageCount: 15,
			expected:   true,
		},
		{
			name:       "Not popular bot",
			usageCount: 5,
			expected:   false,
		},
		{
			name:       "Threshold bot",
			usageCount: 10,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bot := &Bot{UsageCount: tt.usageCount}
			result := bot.IsPopular()
			if result != tt.expected {
				t.Errorf("Expected: %v, Got: %v", tt.expected, result)
			}
		})
	}
}

func TestBot_GetPopularityScore(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name       string
		bot        *Bot
		expectGreaterThan float64
		expectLessThan    float64
	}{
		{
			name: "New popular bot",
			bot: &Bot{
				UsageCount: 20,
				CreateTime: now,
			},
			expectGreaterThan: 15.0,
			expectLessThan:    25.0,
		},
		{
			name: "Old popular bot",
			bot: &Bot{
				UsageCount: 20,
				CreateTime: now.AddDate(0, 0, -60), // 60 days old
			},
			expectGreaterThan: 5.0,
			expectLessThan:    15.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := tt.bot.GetPopularityScore()
			if score <= tt.expectGreaterThan || score >= tt.expectLessThan {
				t.Errorf("Expected score between %f and %f, got %f", 
					tt.expectGreaterThan, tt.expectLessThan, score)
			}
		})
	}
}

func TestDefaultGenerationParams(t *testing.T) {
	params := DefaultGenerationParams()
	
	if params.MaxTokens <= 0 {
		t.Error("Expected MaxTokens to be positive")
	}
	
	if params.Temperature < 0 || params.Temperature > 1 {
		t.Error("Expected Temperature to be between 0 and 1")
	}
	
	if params.TopP < 0 || params.TopP > 1 {
		t.Error("Expected TopP to be between 0 and 1")
	}
}

func TestDefaultKnowledgeBaseConfig(t *testing.T) {
	config := DefaultKnowledgeBaseConfig()
	
	if config.SearchType != "HYBRID" && config.SearchType != "SEMANTIC" {
		t.Error("Expected SearchType to be HYBRID or SEMANTIC")
	}
	
	if config.MaxResults <= 0 {
		t.Error("Expected MaxResults to be positive")
	}
	
	if config.ScoreThreshold < 0 || config.ScoreThreshold > 1 {
		t.Error("Expected ScoreThreshold to be between 0 and 1")
	}
}