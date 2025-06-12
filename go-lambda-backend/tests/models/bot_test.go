package models

import (
	"testing"
	"time"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/stretchr/testify/assert"
)

// TestBotModel follows bedrock-chat testing patterns for bot model validation
func TestBotModel(t *testing.T) {
	t.Run("IsAccessibleByUser", func(t *testing.T) {
		// Test cases following bedrock-chat permission patterns
		testCases := []struct {
			name        string
			bot         *models.Bot
			userID      string
			userGroups  []string
			isAdmin     bool
			expected    bool
			description string
		}{
			{
				name: "Owner Access",
				bot: &models.Bot{
					ID:          "test-bot-1",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePrivate,
				},
				userID:      "owner-123",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    true,
				description: "Owner should always have access to their own bot",
			},
			{
				name: "Admin Access",
				bot: &models.Bot{
					ID:          "test-bot-2",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePrivate,
				},
				userID:      "admin-456",
				userGroups:  []string{},
				isAdmin:     true,
				expected:    true,
				description: "Admin should have access to any bot",
			},
			{
				name: "Private Bot - No Access",
				bot: &models.Bot{
					ID:          "test-bot-3",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePrivate,
				},
				userID:      "user-789",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    false,
				description: "Non-owner should not have access to private bot",
			},
			{
				name: "Public Bot - Universal Access",
				bot: &models.Bot{
					ID:          "test-bot-4",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePublic,
				},
				userID:      "user-789",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    true,
				description: "Any user should have access to public bot",
			},
			{
				name: "Partial Bot - User in AllowedUsers",
				bot: &models.Bot{
					ID:           "test-bot-5",
					OwnerUserID:  "owner-123",
					SharedScope:  models.SharedScopePartial,
					AllowedUsers: []string{"user-789", "user-456"},
				},
				userID:      "user-789",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    true,
				description: "User in AllowedUsers should have access to partial bot",
			},
			{
				name: "Partial Bot - User in AllowedGroups",
				bot: &models.Bot{
					ID:            "test-bot-6",
					OwnerUserID:   "owner-123",
					SharedScope:   models.SharedScopePartial,
					AllowedUsers:  []string{},
					AllowedGroups: []string{"engineering", "product"},
				},
				userID:      "user-789",
				userGroups:  []string{"engineering", "design"},
				isAdmin:     false,
				expected:    true,
				description: "User in AllowedGroups should have access to partial bot",
			},
			{
				name: "Partial Bot - No Access",
				bot: &models.Bot{
					ID:            "test-bot-7",
					OwnerUserID:   "owner-123",
					SharedScope:   models.SharedScopePartial,
					AllowedUsers:  []string{"user-456"},
					AllowedGroups: []string{"product"},
				},
				userID:      "user-789",
				userGroups:  []string{"design", "marketing"},
				isAdmin:     false,
				expected:    false,
				description: "User not in AllowedUsers or AllowedGroups should not have access",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.bot.IsAccessibleByUser(tc.userID, tc.userGroups, tc.isAdmin)
				assert.Equal(t, tc.expected, result, tc.description)
			})
		}
	})

	t.Run("IsEditableByUser", func(t *testing.T) {
		// Test edit permissions (more restrictive than access permissions)
		testCases := []struct {
			name        string
			bot         *models.Bot
			userID      string
			userGroups  []string
			isAdmin     bool
			expected    bool
			description string
		}{
			{
				name: "Owner Edit Access",
				bot: &models.Bot{
					ID:          "test-bot-1",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePublic,
				},
				userID:      "owner-123",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    true,
				description: "Owner should be able to edit their own bot",
			},
			{
				name: "Admin Edit Access",
				bot: &models.Bot{
					ID:          "test-bot-2",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePublic,
				},
				userID:      "admin-456",
				userGroups:  []string{},
				isAdmin:     true,
				expected:    true,
				description: "Admin should be able to edit any bot",
			},
			{
				name: "Public Bot - No Edit Access for Non-Owner",
				bot: &models.Bot{
					ID:          "test-bot-3",
					OwnerUserID: "owner-123",
					SharedScope: models.SharedScopePublic,
				},
				userID:      "user-789",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    false,
				description: "Non-owner should not be able to edit public bot even if they can access it",
			},
			{
				name: "Partial Bot - No Edit Access for Allowed User",
				bot: &models.Bot{
					ID:           "test-bot-4",
					OwnerUserID:  "owner-123",
					SharedScope:  models.SharedScopePartial,
					AllowedUsers: []string{"user-789"},
				},
				userID:      "user-789",
				userGroups:  []string{},
				isAdmin:     false,
				expected:    false,
				description: "User with access to partial bot should not be able to edit it",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.bot.IsEditableByUser(tc.userID, tc.userGroups, tc.isAdmin)
				assert.Equal(t, tc.expected, result, tc.description)
			})
		}
	})

	t.Run("IsOwnedByUser", func(t *testing.T) {
		bot := &models.Bot{
			ID:          "test-bot",
			OwnerUserID: "owner-123",
		}

		assert.True(t, bot.IsOwnedByUser("owner-123"), "Owner should be recognized")
		assert.False(t, bot.IsOwnedByUser("other-user"), "Non-owner should not be recognized")
	})

	t.Run("Pinning Functions", func(t *testing.T) {
		t.Run("IsPinned", func(t *testing.T) {
			testCases := []struct {
				name         string
				sharedScope  string
				sharedStatus string
				expected     bool
			}{
				{
					name:         "Pinned Public Bot",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned@001",
					expected:     true,
				},
				{
					name:         "Pinned Public Bot Higher Order",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned@042",
					expected:     true,
				},
				{
					name:         "Shared Public Bot (Not Pinned)",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: models.SharedStatusShared,
					expected:     false,
				},
				{
					name:         "Private Bot with Pinned Status (Invalid)",
					sharedScope:  models.SharedScopePrivate,
					sharedStatus: "pinned@001",
					expected:     false,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					bot := &models.Bot{
						SharedScope:  tc.sharedScope,
						SharedStatus: tc.sharedStatus,
					}
					result := bot.IsPinned()
					assert.Equal(t, tc.expected, result)
				})
			}
		})

		t.Run("GetPinOrder", func(t *testing.T) {
			testCases := []struct {
				name         string
				sharedScope  string
				sharedStatus string
				expected     int
			}{
				{
					name:         "Pin Order 1",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned@001",
					expected:     1,
				},
				{
					name:         "Pin Order 42",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned@042",
					expected:     42,
				},
				{
					name:         "Pin Order 999",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned@999",
					expected:     999,
				},
				{
					name:         "Not Pinned",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: models.SharedStatusShared,
					expected:     0,
				},
				{
					name:         "Invalid Pin Format",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned",
					expected:     0,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					bot := &models.Bot{
						SharedScope:  tc.sharedScope,
						SharedStatus: tc.sharedStatus,
					}
					result := bot.GetPinOrder()
					assert.Equal(t, tc.expected, result)
				})
			}
		})

		t.Run("SetPinnedStatus", func(t *testing.T) {
			bot := &models.Bot{}
			
			bot.SetPinnedStatus(5)
			assert.Equal(t, "pinned@005", bot.SharedStatus)
			
			bot.SetPinnedStatus(123)
			assert.Equal(t, "pinned@123", bot.SharedStatus)
		})

		t.Run("CanBePinned", func(t *testing.T) {
			testCases := []struct {
				name         string
				sharedScope  string
				sharedStatus string
				expected     bool
			}{
				{
					name:         "Public Shared Bot",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: models.SharedStatusShared,
					expected:     true,
				},
				{
					name:         "Public Pinned Bot",
					sharedScope:  models.SharedScopePublic,
					sharedStatus: "pinned@001",
					expected:     true,
				},
				{
					name:         "Private Bot",
					sharedScope:  models.SharedScopePrivate,
					sharedStatus: models.SharedStatusShared,
					expected:     false,
				},
				{
					name:         "Partial Bot",
					sharedScope:  models.SharedScopePartial,
					sharedStatus: models.SharedStatusShared,
					expected:     false,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					bot := &models.Bot{
						SharedScope:  tc.sharedScope,
						SharedStatus: tc.sharedStatus,
					}
					result := bot.CanBePinned()
					assert.Equal(t, tc.expected, result)
				})
			}
		})
	})

	t.Run("CloudFormation Stack Methods", func(t *testing.T) {
		t.Run("HasDynamicKnowledgeBase", func(t *testing.T) {
			stackName := "BrChatKbStacktest123"
			
			botWithStack := &models.Bot{
				CloudFormationStackName: &stackName,
			}
			assert.True(t, botWithStack.HasDynamicKnowledgeBase())
			
			botWithoutStack := &models.Bot{
				CloudFormationStackName: nil,
			}
			assert.False(t, botWithoutStack.HasDynamicKnowledgeBase())
			
			emptyStackName := ""
			botWithEmptyStack := &models.Bot{
				CloudFormationStackName: &emptyStackName,
			}
			assert.False(t, botWithEmptyStack.HasDynamicKnowledgeBase())
		})

		t.Run("IsStackComplete", func(t *testing.T) {
			completeStatus := models.StackStatusCreateComplete
			progressStatus := models.StackStatusCreateInProgress
			
			botComplete := &models.Bot{
				StackStatus: &completeStatus,
			}
			assert.True(t, botComplete.IsStackComplete())
			
			botInProgress := &models.Bot{
				StackStatus: &progressStatus,
			}
			assert.False(t, botInProgress.IsStackComplete())
			
			botNoStatus := &models.Bot{
				StackStatus: nil,
			}
			assert.False(t, botNoStatus.IsStackComplete())
		})
	})

	t.Run("Usage Analytics", func(t *testing.T) {
		t.Run("GetPopularityScore", func(t *testing.T) {
			now := time.Now()
			oneMonthAgo := now.AddDate(0, -1, 0)
			
			// Recent bot with high usage
			recentBot := &models.Bot{
				UsageCount: 100,
				CreateTime: now.AddDate(0, 0, -1), // 1 day old
			}
			recentScore := recentBot.GetPopularityScore()
			
			// Older bot with same usage
			oldBot := &models.Bot{
				UsageCount: 100,
				CreateTime: oneMonthAgo,
			}
			oldScore := oldBot.GetPopularityScore()
			
			// Recent bot should have higher popularity score due to time decay
			assert.Greater(t, recentScore, oldScore, "Recent bot should have higher popularity score")
			assert.Greater(t, recentScore, 50.0, "Recent bot should have substantial score")
			assert.Less(t, oldScore, recentScore, "Old bot should have lower score due to decay")
		})

		t.Run("IsPopular", func(t *testing.T) {
			popularBot := &models.Bot{UsageCount: 15}
			assert.True(t, popularBot.IsPopular())
			
			unpopularBot := &models.Bot{UsageCount: 5}
			assert.False(t, unpopularBot.IsPopular())
			
			thresholdBot := &models.Bot{UsageCount: 10}
			assert.True(t, thresholdBot.IsPopular())
		})
	})

	t.Run("Sync Status Methods", func(t *testing.T) {
		testCases := []struct {
			status         string
			inProgress     bool
			complete       bool
			failed         bool
		}{
			{models.SyncStatusQueued, true, false, false},
			{models.SyncStatusRunning, true, false, false},
			{models.SyncStatusSucceeded, false, true, false},
			{models.SyncStatusFailed, false, false, true},
		}

		for _, tc := range testCases {
			t.Run(tc.status, func(t *testing.T) {
				bot := &models.Bot{SyncStatus: tc.status}
				
				assert.Equal(t, tc.inProgress, bot.IsSyncInProgress())
				assert.Equal(t, tc.complete, bot.IsSyncComplete())
				assert.Equal(t, tc.failed, bot.IsSyncFailed())
			})
		}
	})

	t.Run("ToSummary", func(t *testing.T) {
		kbID := "test-kb-123"
		bot := &models.Bot{
			ID:          "test-bot",
			Title:       "Test Bot",
			Description: "Test Description",
			IsStarred:   true,
			OwnerUserID: "owner-123",
			CreateTime:  time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC),
			LastUsedTime: time.Date(2025, 6, 11, 13, 0, 0, 0, time.UTC),
			KnowledgeBaseID: &kbID,
			ConversationStarters: []models.ConversationStarter{
				{Title: "Hello", Example: "Say hello to the bot"},
				{Title: "Help", Example: "Ask for help"},
			},
			SharedScope:  models.SharedScopePublic,
			SharedStatus: models.SharedStatusShared,
		}

		summary := bot.ToSummary()

		assert.Equal(t, "test-bot", summary.ID)
		assert.Equal(t, "Test Bot", summary.Title)
		assert.Equal(t, "Test Description", summary.Description)
		assert.True(t, summary.IsStarred)
		assert.Equal(t, "owner-123", summary.OwnerUserID)
		assert.Equal(t, bot.CreateTime, summary.CreateTime)
		assert.Equal(t, bot.LastUsedTime, summary.LastUsedTime)
		assert.True(t, summary.HasKnowledgeBase)
		assert.Len(t, summary.ConversationStarters, 2)
		assert.Equal(t, models.SharedScopePublic, summary.SharedScope)
		assert.Equal(t, models.SharedStatusShared, summary.SharedStatus)
	})
}

// TestBotAlias tests the BotAlias model following bedrock-chat patterns
func TestBotAlias(t *testing.T) {
	t.Run("IsEditableByUser", func(t *testing.T) {
		alias := &models.BotAlias{
			ID:     "alias-123",
			UserID: "user-789",
		}

		// Owner can edit
		assert.True(t, alias.IsEditableByUser("user-789", []string{}, false))
		
		// Admin can edit
		assert.True(t, alias.IsEditableByUser("admin-456", []string{}, true))
		
		// Other user cannot edit
		assert.False(t, alias.IsEditableByUser("other-user", []string{}, false))
	})

	t.Run("IsAccessibleByUser", func(t *testing.T) {
		alias := &models.BotAlias{
			ID:     "alias-123",
			UserID: "user-789",
		}

		// Same logic as edit permissions for aliases
		assert.True(t, alias.IsAccessibleByUser("user-789", []string{}, false))
		assert.True(t, alias.IsAccessibleByUser("admin-456", []string{}, true))
		assert.False(t, alias.IsAccessibleByUser("other-user", []string{}, false))
	})

	t.Run("ToSummary", func(t *testing.T) {
		alias := &models.BotAlias{
			ID:          "alias-123",
			Title:       "Shared Test Bot",
			Description: "A shared bot",
			IsStarred:   true,
			OwnerUserID: "original-owner-123",
			CreateTime:  time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC),
			LastUsedTime: time.Date(2025, 6, 11, 13, 0, 0, 0, time.UTC),
			HasKnowledgeBase: true,
			ConversationStarters: []models.ConversationStarter{
				{Title: "Start", Example: "Get started with this bot"},
			},
			SharedScope:  models.SharedScopePublic,
			SharedStatus: models.SharedStatusShared,
		}

		summary := alias.ToSummary()

		assert.Equal(t, "alias-123", summary.ID)
		assert.Equal(t, "Shared Test Bot", summary.Title)
		assert.Equal(t, "A shared bot", summary.Description)
		assert.True(t, summary.IsStarred)
		assert.Equal(t, "original-owner-123", summary.OwnerUserID) // Original owner, not alias owner
		assert.Equal(t, alias.CreateTime, summary.CreateTime)
		assert.Equal(t, alias.LastUsedTime, summary.LastUsedTime)
		assert.True(t, summary.HasKnowledgeBase)
		assert.Len(t, summary.ConversationStarters, 1)
		assert.Equal(t, models.SharedScopePublic, summary.SharedScope)
		assert.Equal(t, models.SharedStatusShared, summary.SharedStatus)
	})

	t.Run("GetBotID", func(t *testing.T) {
		alias := &models.BotAlias{
			ID:            "alias-123",
			OriginalBotID: "original-bot-456",
		}

		assert.Equal(t, "original-bot-456", alias.GetBotID())
	})

	t.Run("IsAlias", func(t *testing.T) {
		alias := &models.BotAlias{}
		assert.True(t, alias.IsAlias())
	})
}

// TestCreateBotRequest tests the request validation logic
func TestCreateBotRequest(t *testing.T) {
	t.Run("RequiresAsync", func(t *testing.T) {
		// Existing KB - no async needed
		existingKBID := "existing-kb-123"
		reqExisting := &models.CreateBotRequest{
			ExistingKnowledgeBaseID: &existingKBID,
		}
		assert.False(t, reqExisting.RequiresAsync())

		// New KB creation - async needed
		reqNew := &models.CreateBotRequest{
			KnowledgeBaseCreation: &models.KnowledgeBaseCreationConfig{
				EmbeddingsModel: "amazon.titan-embed-text-v1",
			},
		}
		assert.True(t, reqNew.RequiresAsync())

		// Guardrails enabled - async needed
		reqGuardrails := &models.CreateBotRequest{
			KnowledgeBaseCreation: &models.KnowledgeBaseCreationConfig{
				GuardrailConfig: &models.GuardrailConfig{
					IsEnabled: true,
				},
			},
		}
		assert.True(t, reqGuardrails.RequiresAsync())

		// Neither - should be false
		reqNeither := &models.CreateBotRequest{}
		assert.False(t, reqNeither.RequiresAsync())
	})

	t.Run("DetermineSyncStatus", func(t *testing.T) {
		// Async required
		reqAsync := &models.CreateBotRequest{
			KnowledgeBaseCreation: &models.KnowledgeBaseCreationConfig{},
		}
		assert.Equal(t, models.SyncStatusQueued, reqAsync.DetermineSyncStatus())

		// No async required
		existingKBID := "existing-kb-123"
		reqSync := &models.CreateBotRequest{
			ExistingKnowledgeBaseID: &existingKBID,
		}
		assert.Equal(t, models.SyncStatusSucceeded, reqSync.DetermineSyncStatus())
	})
}

// TestDefaultConfigurations tests the default value functions
func TestDefaultConfigurations(t *testing.T) {
	t.Run("DefaultGenerationParams", func(t *testing.T) {
		params := models.DefaultGenerationParams()
		
		assert.Equal(t, 2048, params.MaxTokens)
		assert.Equal(t, 0.7, params.Temperature)
		assert.Equal(t, 0.9, params.TopP)
		assert.Equal(t, 250, params.TopK)
		assert.Equal(t, []string{}, params.StopSequences)
	})

	t.Run("DefaultKnowledgeBaseConfig", func(t *testing.T) {
		config := models.DefaultKnowledgeBaseConfig()
		
		assert.Equal(t, "HYBRID", config.SearchType)
		assert.Equal(t, 20, config.MaxResults)
		assert.Equal(t, 0.7, config.ScoreThreshold)
	})
}