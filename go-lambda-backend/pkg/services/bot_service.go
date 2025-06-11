package services

import (
	"context"
	"errors"
	"time"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

var (
	ErrBotNotFound  = errors.New("bot not found")
	ErrUnauthorized = errors.New("unauthorized access")
)

// BotService handles business logic for bot operations
type BotService struct {
	botRepo repositories.BotRepository
}

// NewBotService creates a new BotService
func NewBotService(botRepo repositories.BotRepository) *BotService {
	return &BotService{
		botRepo: botRepo,
	}
}

// CreateBot creates a new bot
func (s *BotService) CreateBot(ctx context.Context, bot *models.Bot) error {
	return s.botRepo.Create(ctx, bot)
}

// GetBot retrieves a bot by ID with access control
func (s *BotService) GetBot(ctx context.Context, botID, userID string) (*models.Bot, error) {
	bot, err := s.botRepo.GetByID(ctx, botID)
	if err != nil {
		return nil, ErrBotNotFound
	}

	// Check access permissions
	if !s.canUserAccessBot(bot, userID) {
		return nil, ErrUnauthorized
	}

	return bot, nil
}

// ListBots retrieves bots for a user
func (s *BotService) ListBots(ctx context.Context, userID string, includePublic, starred bool) ([]models.Bot, error) {
	// Get user's own bots
	userBots, err := s.botRepo.GetByOwner(ctx, userID)
	if err != nil {
		return nil, err
	}

	var allBots []models.Bot = userBots

	// Add public bots if requested
	if includePublic {
		publicBots, err := s.botRepo.GetPublicBots(ctx)
		if err != nil {
			return nil, err
		}
		allBots = append(allBots, publicBots...)
	}

	// Filter by starred if requested
	if starred {
		var starredBots []models.Bot
		for _, bot := range allBots {
			if bot.IsStarred {
				starredBots = append(starredBots, bot)
			}
		}
		allBots = starredBots
	}

	// Remove duplicates and sort by last used time
	uniqueBots := s.removeDuplicateBots(allBots)
	s.sortBotsByLastUsed(uniqueBots)

	return uniqueBots, nil
}

// UpdateBot updates an existing bot
func (s *BotService) UpdateBot(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error {
	return s.botRepo.Update(ctx, botID, updateReq)
}

// DeleteBot deletes a bot
func (s *BotService) DeleteBot(ctx context.Context, botID string) error {
	return s.botRepo.Delete(ctx, botID)
}

// UpdateLastUsedTime updates the last used timestamp for a bot
func (s *BotService) UpdateLastUsedTime(ctx context.Context, botID string) error {
	return s.botRepo.UpdateLastUsedTime(ctx, botID, time.Now())
}

// ToggleStar toggles the starred status of a bot for a user
func (s *BotService) ToggleStar(ctx context.Context, botID, userID string, starred bool) error {
	// Check if user can access the bot
	bot, err := s.GetBot(ctx, botID, userID)
	if err != nil {
		return err
	}

	// Only allow starring for owner or public bots
	if bot.OwnerUserID != userID && !bot.IsPublic {
		return ErrUnauthorized
	}

	return s.botRepo.UpdateStarred(ctx, botID, starred)
}

// GetBotSummaries returns lightweight bot information for listing
func (s *BotService) GetBotSummaries(ctx context.Context, userID string, includePublic bool) ([]models.BotSummary, error) {
	bots, err := s.ListBots(ctx, userID, includePublic, false)
	if err != nil {
		return nil, err
	}

	summaries := make([]models.BotSummary, len(bots))
	for i, bot := range bots {
		summaries[i] = models.BotSummary{
			ID:                   bot.ID,
			Title:                bot.Title,
			Description:          bot.Description,
			IsPublic:             bot.IsPublic,
			IsStarred:            bot.IsStarred,
			OwnerUserID:          bot.OwnerUserID,
			CreateTime:           bot.CreateTime,
			LastUsedTime:         bot.LastUsedTime,
			HasKnowledgeBase:     bot.KnowledgeBaseID != "",
			ConversationStarters: bot.ConversationStarters,
		}
	}

	return summaries, nil
}

// ValidateKnowledgeBaseAccess validates that a user can access a knowledge base
func (s *BotService) ValidateKnowledgeBaseAccess(ctx context.Context, knowledgeBaseID, userID string) error {
	// This would typically validate against AWS Bedrock Knowledge Base permissions
	// For now, return nil (allow all access)
	// In production, implement proper permission checking
	return nil
}

// canUserAccessBot checks if a user can access a specific bot
func (s *BotService) canUserAccessBot(bot *models.Bot, userID string) bool {
	// Owner can always access
	if bot.OwnerUserID == userID {
		return true
	}

	// Public bots can be accessed by anyone
	if bot.IsPublic {
		return true
	}

	// Private bots can only be accessed by owner
	return false
}

// removeDuplicateBots removes duplicate bots from a slice
func (s *BotService) removeDuplicateBots(bots []models.Bot) []models.Bot {
	seen := make(map[string]bool)
	var unique []models.Bot

	for _, bot := range bots {
		if !seen[bot.ID] {
			seen[bot.ID] = true
			unique = append(unique, bot)
		}
	}

	return unique
}

// sortBotsByLastUsed sorts bots by last used time (most recent first)
func (s *BotService) sortBotsByLastUsed(bots []models.Bot) {
	for i := 0; i < len(bots)-1; i++ {
		for j := i + 1; j < len(bots); j++ {
			if bots[i].LastUsedTime.Before(bots[j].LastUsedTime) {
				bots[i], bots[j] = bots[j], bots[i]
			}
		}
	}
}