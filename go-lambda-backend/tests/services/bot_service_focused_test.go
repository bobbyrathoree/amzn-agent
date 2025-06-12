package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestBotServiceCore focuses on the most critical business logic patterns
func TestBotServiceCore(t *testing.T) {
	ctx := context.Background()

	// Helper to create service with mocks for focused testing
	createFocusedService := func() (*services.BotService, *MockBotRepository) {
		mockRepo := &MockBotRepository{}
		mockCDK := &MockCDKDeploymentService{}
		mockStack := &MockStackOutputService{}
		
		// Cast to interfaces
		var cdkInterface services.CDKDeploymentServiceInterface = mockCDK
		var stackInterface services.StackOutputServiceInterface = mockStack
		service := services.NewBotService(mockRepo, cdkInterface, stackInterface, "us-east-1", "test")
		return service, mockRepo
	}

	t.Run("CreateBot_WithExistingKB_Success", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		existingKBID := "existing-kb-123"
		req := &models.CreateBotRequest{
			Title:                   "Test Bot",
			Description:            "Test Description",
			Instruction:            "Test Instruction",
			ExistingKnowledgeBaseID: &existingKBID,
			SharedScope:            models.SharedScopePrivate,
			GenerationParams:       models.DefaultGenerationParams(),
			KnowledgeBaseConfig:    models.DefaultKnowledgeBaseConfig(),
		}

		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Bot")).Return(nil)

		result, err := service.CreateBot(ctx, req, "owner-123", []string{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Bot)
		assert.Equal(t, &existingKBID, result.Bot.KnowledgeBaseID)
		assert.False(t, result.StackDeploymentStarted)
		assert.Contains(t, result.Message, "existing Knowledge Base")
		assert.Equal(t, models.SyncStatusSucceeded, result.Bot.SyncStatus)
		
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBot_OwnerAccess_Success", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		expectedBot := &models.Bot{
			ID:          "test-bot-123",
			Title:       "Test Bot",
			OwnerUserID: "owner-123",
			SharedScope: models.SharedScopePrivate,
		}

		mockRepo.On("GetByID", ctx, "test-bot-123").Return(expectedBot, nil)

		bot, err := service.GetBot(ctx, "test-bot-123", "owner-123", []string{}, false)

		assert.NoError(t, err)
		assert.Equal(t, expectedBot, bot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBot_UnauthorizedAccess_Denied", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		privateBot := &models.Bot{
			ID:          "private-bot-123",
			Title:       "Private Bot",
			OwnerUserID: "owner-123",
			SharedScope: models.SharedScopePrivate,
		}

		mockRepo.On("GetByID", ctx, "private-bot-123").Return(privateBot, nil)

		bot, err := service.GetBot(ctx, "private-bot-123", "other-user", []string{}, false)

		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)
		assert.Nil(t, bot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBot_PublicBot_AccessGranted", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		publicBot := &models.Bot{
			ID:          "public-bot-123",
			Title:       "Public Bot", 
			OwnerUserID: "owner-123",
			SharedScope: models.SharedScopePublic,
		}

		mockRepo.On("GetByID", ctx, "public-bot-123").Return(publicBot, nil)

		bot, err := service.GetBot(ctx, "public-bot-123", "any-user", []string{}, false)

		assert.NoError(t, err)
		assert.Equal(t, publicBot, bot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBot_PartialBot_GroupAccess", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		partialBot := &models.Bot{
			ID:            "partial-bot-123",
			Title:         "Partial Bot",
			OwnerUserID:   "owner-123",
			SharedScope:   models.SharedScopePartial,
			AllowedGroups: []string{"engineering", "product"},
		}

		mockRepo.On("GetByID", ctx, "partial-bot-123").Return(partialBot, nil)

		bot, err := service.GetBot(ctx, "partial-bot-123", "user-789", []string{"engineering", "design"}, false)

		assert.NoError(t, err)
		assert.Equal(t, partialBot, bot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBot_NotFound", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		mockRepo.On("GetByID", ctx, "nonexistent-bot").Return(nil, errors.New("not found"))

		bot, err := service.GetBot(ctx, "nonexistent-bot", "user-123", []string{}, false)

		assert.Error(t, err)
		assert.Equal(t, services.ErrBotNotFound, err)
		assert.Nil(t, bot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ListBots_PrivateScope", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		now := time.Now()
		expectedBots := []models.Bot{
			{ID: "bot1", OwnerUserID: "user-123", SharedScope: models.SharedScopePrivate, LastUsedTime: now.Add(-time.Hour)}, // Older
			{ID: "bot2", OwnerUserID: "user-123", SharedScope: models.SharedScopePublic, LastUsedTime: now}, // Newer, will be first
		}

		mockRepo.On("GetByOwner", ctx, "user-123").Return(expectedBots, nil)

		bots, err := service.ListBots(ctx, "user-123", []string{}, false, "private", false, 100)

		assert.NoError(t, err)
		assert.Len(t, bots, 2)
		// Bots are sorted by LastUsedTime (most recent first)
		assert.Equal(t, "bot2", bots[0].ID, "Most recently used bot should be first")
		assert.Equal(t, "bot1", bots[1].ID, "Less recently used bot should be second")
		mockRepo.AssertExpectations(t)
	})

	t.Run("ListBots_PublicScope", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		expectedBots := []models.Bot{
			{ID: "public-bot1", OwnerUserID: "owner1", SharedScope: models.SharedScopePublic, LastUsedTime: time.Now()},
			{ID: "public-bot2", OwnerUserID: "owner2", SharedScope: models.SharedScopePublic, LastUsedTime: time.Now()},
		}

		mockRepo.On("GetPublicBots", ctx).Return(expectedBots, nil)

		bots, err := service.ListBots(ctx, "user-123", []string{}, false, "public", false, 100)

		assert.NoError(t, err)
		assert.Len(t, bots, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBotSummaries", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		expectedBots := []models.Bot{
			{
				ID:           "bot1",
				Title:        "Bot 1",
				Description:  "Description 1",
				OwnerUserID:  "user-123",
				SharedScope:  models.SharedScopePrivate,
				CreateTime:   time.Now(),
				LastUsedTime: time.Now(),
				ConversationStarters: []models.ConversationStarter{
					{Title: "Hello", Example: "Say hello"},
				},
			},
		}

		mockRepo.On("GetByOwner", ctx, "user-123").Return(expectedBots, nil)

		summaries, err := service.GetBotSummaries(ctx, "user-123", []string{}, false, "private", false, 100)

		assert.NoError(t, err)
		assert.Len(t, summaries, 1)
		assert.Equal(t, "bot1", summaries[0].ID)
		assert.Equal(t, "Bot 1", summaries[0].Title)
		assert.Len(t, summaries[0].ConversationStarters, 1)
		assert.Equal(t, "Hello", summaries[0].ConversationStarters[0].Title)
		mockRepo.AssertExpectations(t)
	})

	t.Run("IncrementBotUsage", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		mockRepo.On("IncrementUsageCount", ctx, "bot-123").Return(nil)
		mockRepo.On("UpdateLastUsedTime", ctx, "bot-123", mock.AnythingOfType("time.Time")).Return(nil)

		err := service.IncrementBotUsage(ctx, "bot-123")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ToggleStar", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		mockRepo.On("UpdateStarred", ctx, "bot-123", true).Return(nil)

		err := service.ToggleStar(ctx, "bot-123", "user-123", true)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateBotSharing_Success", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		bot := &models.Bot{
			ID:          "test-bot-123",
			OwnerUserID: "owner-123",
			SharedScope: models.SharedScopePrivate,
		}

		mockRepo.On("GetByID", ctx, "test-bot-123").Return(bot, nil)
		mockRepo.On("Update", ctx, "test-bot-123", mock.AnythingOfType("*models.UpdateBotRequest")).Return(nil)

		err := service.UpdateBotSharing(ctx, "test-bot-123", "owner-123", models.SharedScopePublic, models.SharedStatusShared, []string{}, []string{})

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateBotSharing_NotOwner", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		bot := &models.Bot{
			ID:          "test-bot-123",
			OwnerUserID: "owner-123",
			SharedScope: models.SharedScopePrivate,
		}

		mockRepo.On("GetByID", ctx, "test-bot-123").Return(bot, nil)

		err := service.UpdateBotSharing(ctx, "test-bot-123", "other-user", models.SharedScopePublic, models.SharedStatusShared, []string{}, []string{})

		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestBotAliasServiceCore focuses on alias functionality
func TestBotAliasServiceCore(t *testing.T) {
	ctx := context.Background()

	createFocusedService := func() (*services.BotService, *MockBotRepository) {
		mockRepo := &MockBotRepository{}
		mockCDK := &MockCDKDeploymentService{}
		mockStack := &MockStackOutputService{}
		
		var cdkInterface services.CDKDeploymentServiceInterface = mockCDK
		var stackInterface services.StackOutputServiceInterface = mockStack
		service := services.NewBotService(mockRepo, cdkInterface, stackInterface, "us-east-1", "test")
		return service, mockRepo
	}

	t.Run("GetOrCreateBotAlias_OwnerBot_NoAlias", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		ownedBot := &models.Bot{
			ID:          "bot-123",
			OwnerUserID: "user-123",
			SharedScope: models.SharedScopePublic,
		}

		mockRepo.On("GetByID", ctx, "bot-123").Return(ownedBot, nil)

		alias, err := service.GetOrCreateBotAlias(ctx, "bot-123", "user-123", []string{})

		assert.NoError(t, err)
		assert.Nil(t, alias) // No alias needed for owned bot
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetOrCreateBotAlias_SharedBot_CreateNew", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		sharedBot := &models.Bot{
			ID:          "bot-123",
			Title:       "Shared Bot",
			Description: "A shared bot",
			OwnerUserID: "owner-456",
			SharedScope: models.SharedScopePublic,
			ConversationStarters: []models.ConversationStarter{
				{Title: "Hello", Example: "Say hello"},
			},
		}

		mockRepo.On("GetByID", ctx, "bot-123").Return(sharedBot, nil)
		mockRepo.On("GetAliasForUserBot", ctx, "user-123", "bot-123").Return(nil, nil)
		mockRepo.On("CreateAlias", ctx, mock.AnythingOfType("*models.BotAlias")).Return(nil)

		alias, err := service.GetOrCreateBotAlias(ctx, "bot-123", "user-123", []string{})

		assert.NoError(t, err)
		assert.NotNil(t, alias)
		assert.Equal(t, "bot-123", alias.OriginalBotID)
		assert.Equal(t, "user-123", alias.UserID)
		assert.Equal(t, "Shared Bot", alias.Title)
		assert.True(t, alias.IsOriginAccessible)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBotWithAlias_OwnerBot", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		ownedBot := &models.Bot{
			ID:          "bot-123",
			OwnerUserID: "user-123",
			SharedScope: models.SharedScopePrivate,
		}

		mockRepo.On("GetByID", ctx, "bot-123").Return(ownedBot, nil)

		bot, alias, err := service.GetBotWithAlias(ctx, "bot-123", "user-123", []string{}, false)

		assert.NoError(t, err)
		assert.Equal(t, ownedBot, bot)
		assert.Nil(t, alias)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ToggleAliasStarred", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		mockRepo.On("UpdateAliasStarred", ctx, "alias-123", true).Return(nil)

		err := service.ToggleAliasStarred(ctx, "alias-123", true)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DeleteBotAlias_Success", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		alias := &models.BotAlias{
			ID:     "alias-123",
			UserID: "user-123",
		}

		mockRepo.On("GetAliasByID", ctx, "alias-123").Return(alias, nil)
		mockRepo.On("DeleteAlias", ctx, "alias-123").Return(nil)

		err := service.DeleteBotAlias(ctx, "alias-123", "user-123")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DeleteBotAlias_Unauthorized", func(t *testing.T) {
		service, mockRepo := createFocusedService()
		
		alias := &models.BotAlias{
			ID:     "alias-123",
			UserID: "owner-456", // Different user
		}

		mockRepo.On("GetAliasByID", ctx, "alias-123").Return(alias, nil)

		err := service.DeleteBotAlias(ctx, "alias-123", "user-123")

		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)
		mockRepo.AssertExpectations(t)
	})
}