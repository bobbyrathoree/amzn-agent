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

// MockBotRepository implements the BotRepository interface for testing
type MockBotRepository struct {
	mock.Mock
}

func (m *MockBotRepository) Create(ctx context.Context, bot *models.Bot) error {
	args := m.Called(ctx, bot)
	return args.Error(0)
}

func (m *MockBotRepository) GetByID(ctx context.Context, botID string) (*models.Bot, error) {
	args := m.Called(ctx, botID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bot), args.Error(1)
}

func (m *MockBotRepository) Update(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error {
	args := m.Called(ctx, botID, updateReq)
	return args.Error(0)
}

func (m *MockBotRepository) Delete(ctx context.Context, botID string) error {
	args := m.Called(ctx, botID)
	return args.Error(0)
}

func (m *MockBotRepository) GetByOwner(ctx context.Context, ownerUserID string) ([]models.Bot, error) {
	args := m.Called(ctx, ownerUserID)
	return args.Get(0).([]models.Bot), args.Error(1)
}

func (m *MockBotRepository) GetPublicBots(ctx context.Context) ([]models.Bot, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Bot), args.Error(1)
}

func (m *MockBotRepository) GetSharedBots(ctx context.Context, userID string, userGroups []string) ([]models.Bot, error) {
	args := m.Called(ctx, userID, userGroups)
	return args.Get(0).([]models.Bot), args.Error(1)
}

func (m *MockBotRepository) GetPinnedBots(ctx context.Context) ([]models.Bot, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.Bot), args.Error(1)
}

func (m *MockBotRepository) UpdateLastUsedTime(ctx context.Context, botID string, lastUsedTime time.Time) error {
	args := m.Called(ctx, botID, lastUsedTime)
	return args.Error(0)
}

func (m *MockBotRepository) UpdateStarred(ctx context.Context, botID string, starred bool) error {
	args := m.Called(ctx, botID, starred)
	return args.Error(0)
}

func (m *MockBotRepository) IncrementUsageCount(ctx context.Context, botID string) error {
	args := m.Called(ctx, botID)
	return args.Error(0)
}

func (m *MockBotRepository) SearchBots(ctx context.Context, query string, scope string, userID string, userGroups []string) ([]models.Bot, error) {
	args := m.Called(ctx, query, scope, userID, userGroups)
	return args.Get(0).([]models.Bot), args.Error(1)
}

func (m *MockBotRepository) GetBotsByScope(ctx context.Context, scope string, userID string, userGroups []string, limit int) ([]models.Bot, error) {
	args := m.Called(ctx, scope, userID, userGroups, limit)
	return args.Get(0).([]models.Bot), args.Error(1)
}

// Bot Alias methods
func (m *MockBotRepository) CreateAlias(ctx context.Context, alias *models.BotAlias) error {
	args := m.Called(ctx, alias)
	return args.Error(0)
}

func (m *MockBotRepository) GetAliasByID(ctx context.Context, aliasID string) (*models.BotAlias, error) {
	args := m.Called(ctx, aliasID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BotAlias), args.Error(1)
}

func (m *MockBotRepository) GetAliasesByUser(ctx context.Context, userID string) ([]models.BotAlias, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.BotAlias), args.Error(1)
}

func (m *MockBotRepository) GetAliasForUserBot(ctx context.Context, userID, originalBotID string) (*models.BotAlias, error) {
	args := m.Called(ctx, userID, originalBotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BotAlias), args.Error(1)
}

func (m *MockBotRepository) UpdateAlias(ctx context.Context, aliasID string, updates map[string]interface{}) error {
	args := m.Called(ctx, aliasID, updates)
	return args.Error(0)
}

func (m *MockBotRepository) DeleteAlias(ctx context.Context, aliasID string) error {
	args := m.Called(ctx, aliasID)
	return args.Error(0)
}

func (m *MockBotRepository) UpdateAliasLastUsedTime(ctx context.Context, aliasID string, lastUsedTime time.Time) error {
	args := m.Called(ctx, aliasID, lastUsedTime)
	return args.Error(0)
}

func (m *MockBotRepository) UpdateAliasStarred(ctx context.Context, aliasID string, starred bool) error {
	args := m.Called(ctx, aliasID, starred)
	return args.Error(0)
}

// MockCDKDeploymentService implements CDKDeploymentServiceInterface for testing
type MockCDKDeploymentService struct {
	mock.Mock
}

func (m *MockCDKDeploymentService) DeployBotKnowledgeBaseStack(ctx context.Context, req services.CDKDeploymentRequest) (*services.CDKDeploymentResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.CDKDeploymentResult), args.Error(1)
}

func (m *MockCDKDeploymentService) DestroyBotKnowledgeBaseStack(ctx context.Context, botID string) error {
	args := m.Called(ctx, botID)
	return args.Error(0)
}

// MockStackOutputService implements StackOutputServiceInterface for testing
type MockStackOutputService struct {
	mock.Mock
}

// Ensure mocks implement interfaces
var _ services.CDKDeploymentServiceInterface = (*MockCDKDeploymentService)(nil)
var _ services.StackOutputServiceInterface = (*MockStackOutputService)(nil)

func (m *MockStackOutputService) FetchStackOutputs(ctx context.Context, botID, ownerUserID string) (*services.StackOutputResult, error) {
	args := m.Called(ctx, botID, ownerUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.StackOutputResult), args.Error(1)
}

// TestBotService tests the BotService following bedrock-chat patterns
func TestBotService(t *testing.T) {
	ctx := context.Background()
	
	// Helper to create service with mocks
	createService := func() (*services.BotService, *MockBotRepository, *MockCDKDeploymentService, *MockStackOutputService) {
		mockRepo := &MockBotRepository{}
		mockCDK := &MockCDKDeploymentService{}
		mockStack := &MockStackOutputService{}
		
		// Cast to interfaces for NewBotService
		var cdkInterface services.CDKDeploymentServiceInterface = mockCDK
		var stackInterface services.StackOutputServiceInterface = mockStack
		service := services.NewBotService(mockRepo, cdkInterface, stackInterface, "us-east-1", "test")
		return service, mockRepo, mockCDK, mockStack
	}

	t.Run("CreateBot_ExistingKB", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

		// Mock expectations
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Bot")).Return(nil)

		result, err := service.CreateBot(ctx, req, "owner-123", []string{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Bot)
		assert.Equal(t, &existingKBID, result.Bot.KnowledgeBaseID)
		assert.False(t, result.StackDeploymentStarted)
		assert.Contains(t, result.Message, "existing Knowledge Base")
		
		mockRepo.AssertExpectations(t)
	})

	t.Run("CreateBot_NewKB", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		req := &models.CreateBotRequest{
			Title:       "Test Bot",
			Description: "Test Description", 
			Instruction: "Test Instruction",
			KnowledgeBaseCreation: &models.KnowledgeBaseCreationConfig{
				EmbeddingsModel:  "amazon.titan-embed-text-v1",
				ChunkingStrategy: "FIXED_SIZE",
			},
			SharedScope:         models.SharedScopePrivate,
			GenerationParams:    models.DefaultGenerationParams(),
			KnowledgeBaseConfig: models.DefaultKnowledgeBaseConfig(),
		}

		// Mock expectations
		mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Bot")).Return(nil)
		// Mock async update calls that happen in the background goroutine
		mockRepo.On("Update", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("*models.UpdateBotRequest")).Return(nil).Maybe()

		result, err := service.CreateBot(ctx, req, "owner-123", []string{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotNil(t, result.Bot)
		assert.Nil(t, result.Bot.KnowledgeBaseID) // Not set until async deployment completes
		assert.True(t, result.StackDeploymentStarted)
		assert.Contains(t, result.Message, "Knowledge Base infrastructure deployment started")
		assert.Equal(t, models.SyncStatusQueued, result.Bot.SyncStatus)
		
		mockRepo.AssertExpectations(t)
	})

	t.Run("CreateBot_ValidationError", func(t *testing.T) {
		service, _, _, _ := createService()
		
		// Request with missing required fields
		req := &models.CreateBotRequest{
			// Missing title and instruction
			SharedScope: models.SharedScopePrivate,
		}

		result, err := service.CreateBot(ctx, req, "owner-123", []string{})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("GetBot_Success", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

	t.Run("GetBot_NotFound", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		mockRepo.On("GetByID", ctx, "nonexistent-bot").Return(nil, errors.New("not found"))

		bot, err := service.GetBot(ctx, "nonexistent-bot", "user-123", []string{}, false)

		assert.Error(t, err)
		assert.Equal(t, services.ErrBotNotFound, err)
		assert.Nil(t, bot)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBot_Unauthorized", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

	t.Run("ListBots_Private", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		expectedBots := []models.Bot{
			{ID: "bot1", OwnerUserID: "user-123", SharedScope: models.SharedScopePrivate},
			{ID: "bot2", OwnerUserID: "user-123", SharedScope: models.SharedScopePublic},
		}

		mockRepo.On("GetByOwner", ctx, "user-123").Return(expectedBots, nil)

		bots, err := service.ListBots(ctx, "user-123", []string{}, false, "private", false, 100)

		assert.NoError(t, err)
		assert.Len(t, bots, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ListBots_Public", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		expectedBots := []models.Bot{
			{ID: "public-bot1", OwnerUserID: "owner1", SharedScope: models.SharedScopePublic},
			{ID: "public-bot2", OwnerUserID: "owner2", SharedScope: models.SharedScopePublic},
		}

		mockRepo.On("GetPublicBots", ctx).Return(expectedBots, nil)

		bots, err := service.ListBots(ctx, "user-123", []string{}, false, "public", false, 100)

		assert.NoError(t, err)
		assert.Len(t, bots, 2)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBotSummaries", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		expectedBots := []models.Bot{
			{
				ID:          "bot1",
				Title:       "Bot 1",
				Description: "Description 1",
				OwnerUserID: "user-123",
				SharedScope: models.SharedScopePrivate,
				CreateTime:  time.Now(),
				LastUsedTime: time.Now(),
			},
		}

		mockRepo.On("GetByOwner", ctx, "user-123").Return(expectedBots, nil)

		summaries, err := service.GetBotSummaries(ctx, "user-123", []string{}, false, "private", false, 100)

		assert.NoError(t, err)
		assert.Len(t, summaries, 1)
		assert.Equal(t, "bot1", summaries[0].ID)
		assert.Equal(t, "Bot 1", summaries[0].Title)
		mockRepo.AssertExpectations(t)
	})

	t.Run("DeleteBot_Success", func(t *testing.T) {
		service, mockRepo, mockCDK, _ := createService()
		
		stackName := "BrChatKbStacktest123"
		botWithStack := &models.Bot{
			ID:                      "test-bot-123",
			OwnerUserID:            "owner-123",
			SharedScope:            models.SharedScopePrivate,
			CloudFormationStackName: &stackName,
		}

		mockRepo.On("GetByID", ctx, "test-bot-123").Return(botWithStack, nil)
		mockCDK.On("DestroyBotKnowledgeBaseStack", ctx, "test-bot-123").Return(nil)
		mockRepo.On("Delete", ctx, "test-bot-123").Return(nil)

		err := service.DeleteBot(ctx, "test-bot-123", "owner-123", []string{}, false)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		mockCDK.AssertExpectations(t)
	})

	t.Run("DeleteBot_NotOwner", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		bot := &models.Bot{
			ID:          "test-bot-123",
			OwnerUserID: "owner-123",
			SharedScope: models.SharedScopePrivate,
		}

		mockRepo.On("GetByID", ctx, "test-bot-123").Return(bot, nil)

		err := service.DeleteBot(ctx, "test-bot-123", "other-user", []string{}, false)

		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("UpdateBotSharing", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

	t.Run("IncrementBotUsage", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		mockRepo.On("IncrementUsageCount", ctx, "test-bot-123").Return(nil)
		mockRepo.On("UpdateLastUsedTime", ctx, "test-bot-123", mock.AnythingOfType("time.Time")).Return(nil)

		err := service.IncrementBotUsage(ctx, "test-bot-123")

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetPopularBots", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		publicBots := []models.Bot{
			{ID: "bot1", OwnerUserID: "owner1", SharedScope: models.SharedScopePublic, UsageCount: 100, CreateTime: time.Now().AddDate(0, 0, -1)},
			{ID: "bot2", OwnerUserID: "owner2", SharedScope: models.SharedScopePublic, UsageCount: 50, CreateTime: time.Now().AddDate(0, 0, -30)},
		}
		sharedBots := []models.Bot{}

		mockRepo.On("GetPublicBots", ctx).Return(publicBots, nil)
		mockRepo.On("GetSharedBots", ctx, "user-123", []string{"group1"}).Return(sharedBots, nil)

		summaries, err := service.GetPopularBots(ctx, "user-123", []string{"group1"}, 10)

		assert.NoError(t, err)
		assert.Len(t, summaries, 2)
		// First bot should be more popular due to recent usage
		assert.Equal(t, "bot1", summaries[0].ID)
		mockRepo.AssertExpectations(t)
	})
}

// TestBotAliasService tests the alias functionality
func TestBotAliasService(t *testing.T) {
	ctx := context.Background()
	
	createService := func() (*services.BotService, *MockBotRepository, *MockCDKDeploymentService, *MockStackOutputService) {
		mockRepo := &MockBotRepository{}
		mockCDK := &MockCDKDeploymentService{}
		mockStack := &MockStackOutputService{}
		
		// Cast to interfaces for NewBotService
		var cdkInterface services.CDKDeploymentServiceInterface = mockCDK
		var stackInterface services.StackOutputServiceInterface = mockStack
		service := services.NewBotService(mockRepo, cdkInterface, stackInterface, "us-east-1", "test")
		return service, mockRepo, mockCDK, mockStack
	}

	t.Run("GetOrCreateBotAlias_OwnerBot", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

	t.Run("GetOrCreateBotAlias_SharedBot_New", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

	t.Run("GetOrCreateBotAlias_SharedBot_Existing", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		sharedBot := &models.Bot{
			ID:          "bot-123",
			OwnerUserID: "owner-456",
			SharedScope: models.SharedScopePublic,
		}

		existingAlias := &models.BotAlias{
			ID:                  "alias-789",
			OriginalBotID:       "bot-123",
			UserID:              "user-123",
			IsOriginAccessible:  true,
		}

		mockRepo.On("GetByID", ctx, "bot-123").Return(sharedBot, nil)
		mockRepo.On("GetAliasForUserBot", ctx, "user-123", "bot-123").Return(existingAlias, nil)

		alias, err := service.GetOrCreateBotAlias(ctx, "bot-123", "user-123", []string{})

		assert.NoError(t, err)
		assert.Equal(t, existingAlias, alias)
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetBotWithAlias_OwnerBot", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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

	t.Run("DeleteBotAlias_Success", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
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
		service, mockRepo, _, _ := createService()
		
		alias := &models.BotAlias{
			ID:     "alias-123",
			UserID: "owner-456",
		}

		mockRepo.On("GetAliasByID", ctx, "alias-123").Return(alias, nil)

		err := service.DeleteBotAlias(ctx, "alias-123", "user-123")

		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("ToggleAliasStarred", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		mockRepo.On("UpdateAliasStarred", ctx, "alias-123", true).Return(nil)

		err := service.ToggleAliasStarred(ctx, "alias-123", true)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})
}

// TestBotPinningService tests the pinning functionality
func TestBotPinningService(t *testing.T) {
	ctx := context.Background()
	
	createService := func() (*services.BotService, *MockBotRepository, *MockCDKDeploymentService, *MockStackOutputService) {
		mockRepo := &MockBotRepository{}
		mockCDK := &MockCDKDeploymentService{}
		mockStack := &MockStackOutputService{}
		
		// Cast to interfaces for NewBotService
		var cdkInterface services.CDKDeploymentServiceInterface = mockCDK
		var stackInterface services.StackOutputServiceInterface = mockStack
		service := services.NewBotService(mockRepo, cdkInterface, stackInterface, "us-east-1", "test")
		return service, mockRepo, mockCDK, mockStack
	}

	t.Run("PinBot_Success", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		publicBot := &models.Bot{
			ID:           "bot-123",
			SharedScope:  models.SharedScopePublic,
			SharedStatus: models.SharedStatusShared,
		}

		mockRepo.On("GetByID", ctx, "bot-123").Return(publicBot, nil)
		mockRepo.On("GetPinnedBots", ctx).Return([]models.Bot{}, nil) // No conflicts
		mockRepo.On("Update", ctx, "bot-123", mock.AnythingOfType("*models.UpdateBotRequest")).Return(nil)

		err := service.PinBot(ctx, "bot-123", "admin-123", true, 5)

		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("PinBot_NotAdmin", func(t *testing.T) {
		service, _, _, _ := createService()

		err := service.PinBot(ctx, "bot-123", "user-123", false, 5)

		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)
	})

	t.Run("PinBot_NotPublic", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		privateBot := &models.Bot{
			ID:           "bot-123",
			SharedScope:  models.SharedScopePrivate,
			SharedStatus: models.SharedStatusUnshared,
		}

		mockRepo.On("GetByID", ctx, "bot-123").Return(privateBot, nil)

		err := service.PinBot(ctx, "bot-123", "admin-123", true, 5)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be pinned")
		mockRepo.AssertExpectations(t)
	})

	t.Run("GetNextAvailablePinOrder", func(t *testing.T) {
		service, mockRepo, _, _ := createService()
		
		pinnedBots := []models.Bot{
			{SharedStatus: "pinned@001"},
			{SharedStatus: "pinned@003"},
			{SharedStatus: "pinned@005"},
		}

		mockRepo.On("GetPinnedBots", ctx).Return(pinnedBots, nil)

		order, err := service.GetNextAvailablePinOrder(ctx)

		assert.NoError(t, err)
		assert.Equal(t, 2, order) // First available order
		mockRepo.AssertExpectations(t)
	})
}