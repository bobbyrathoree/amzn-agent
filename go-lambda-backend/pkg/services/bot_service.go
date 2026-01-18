package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

var (
	ErrBotNotFound        = errors.New("bot not found")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrStackDeployment   = errors.New("stack deployment failed")
	ErrKnowledgeBaseRequired = errors.New("either existing knowledge base ID or creation config is required")
)

// BotService handles business logic for full-featured bot operations
type BotService struct {
	botRepo           repositories.BotRepository
	cfnService        CloudFormationServiceInterface
	stackOutputService StackOutputServiceInterface
	region            string
	envPrefix         string
}

// NewBotService creates a new BotService with full-featured capabilities
func NewBotService(
	botRepo repositories.BotRepository,
	cfnService CloudFormationServiceInterface,
	stackOutputService StackOutputServiceInterface,
	region, envPrefix string,
) *BotService {
	return &BotService{
		botRepo:           botRepo,
		cfnService:        cfnService,
		stackOutputService: stackOutputService,
		region:            region,
		envPrefix:         envPrefix,
	}
}

// CreateBotResult contains the result of bot creation
type CreateBotResult struct {
	Bot                 *models.Bot `json:"bot"`
	StackDeploymentStarted bool       `json:"stackDeploymentStarted"`
	Message             string      `json:"message"`
}

// CreateBot creates a new full-featured bot with conditional Knowledge Base provisioning
func (s *BotService) CreateBot(ctx context.Context, req *models.CreateBotRequest, ownerUserID string, userGroups []string) (*CreateBotResult, error) {
	log.Printf("🤖 Creating full-featured bot for user: %s", ownerUserID)
	
	// Validate request
	if err := s.validateCreateBotRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, err.Error())
	}
	
	// Generate unique bot ID
	botID := s.generateBotID()
	
	// Determine sync status and execution ID
	syncStatus := req.DetermineSyncStatus()
	var syncExecID string
	if syncStatus == models.SyncStatusQueued {
		syncExecID = uuid.New().String() // Generate unique execution ID for tracking
	}
	
	// Create initial bot record
	bot := &models.Bot{
		ID:                    botID,
		Title:                 req.Title,
		Description:           req.Description,
		Instruction:           req.Instruction,
		OwnerUserID:           ownerUserID,
		CreateTime:            time.Now(),
		UpdateTime:            time.Now(),
		LastUsedTime:          time.Now(),
		
		// Sharing configuration
		SharedScope:           req.SharedScope,
		SharedStatus:          models.SharedStatusUnshared, // Start as unshared
		AllowedUsers:          req.AllowedUsers,
		AllowedGroups:         req.AllowedGroups,
		
		// Async processing status
		SyncStatus:            syncStatus,
		SyncLastExecID:        syncExecID,
		
		// Configuration
		GenerationParams:      req.GenerationParams,
		KnowledgeBaseConfig:   req.KnowledgeBaseConfig,
		ConversationStarters:  req.ConversationStarters,
		ActiveModels:          req.ActiveModels,
		AgentTools:            req.AgentTools,
		DisplayRetrievedChunks: req.DisplayRetrievedChunks,
		IsStarred:             false,
	}
	
	// Set default generation params if not provided
	if bot.GenerationParams.MaxTokens == 0 {
		bot.GenerationParams = models.DefaultGenerationParams()
	}
	
	// Set default knowledge base config if not provided
	if bot.KnowledgeBaseConfig.SearchType == "" {
		bot.KnowledgeBaseConfig = models.DefaultKnowledgeBaseConfig()
	}
	
	// CONDITIONAL KNOWLEDGE BASE LOGIC
	if req.ExistingKnowledgeBaseID != nil {
		// PATH 1: Use existing Knowledge Base
		log.Printf("📋 Using existing Knowledge Base: %s", *req.ExistingKnowledgeBaseID)
		
		bot.KnowledgeBaseID = req.ExistingKnowledgeBaseID
		// No stack deployment needed
		
		// Save bot to database
		log.Printf("💾 BotService.CreateBot: Saving bot to database...")
		if err := s.botRepo.Create(ctx, bot); err != nil {
			log.Printf("❌ BotService.CreateBot: Failed to save bot to database: %v", err)
			return nil, fmt.Errorf("failed to create bot with existing KB: %w", err)
		}
		log.Printf("✅ BotService.CreateBot: Bot saved to database successfully")
		
		return &CreateBotResult{
			Bot:                   bot,
			StackDeploymentStarted: false,
			Message:               fmt.Sprintf("Bot created successfully using existing Knowledge Base: %s", *req.ExistingKnowledgeBaseID),
		}, nil
		
	} else if req.KnowledgeBaseCreation != nil {
		// PATH 2: Create new Knowledge Base infrastructure
		log.Printf("🏗️ Creating new Knowledge Base infrastructure for bot: %s", botID)

		// Set up stack tracking
		stackName := fmt.Sprintf("BrChatKbStack%s", botID)
		bot.CloudFormationStackName = &stackName
		initialStatus := models.StackStatusCreateInProgress
		bot.StackStatus = &initialStatus

		// Save bot to database with stack in progress
		log.Printf("💾 BotService.CreateBot: Saving bot to database with stack in progress...")
		if err := s.botRepo.Create(ctx, bot); err != nil {
			log.Printf("❌ BotService.CreateBot: Failed to save bot to database: %v", err)
			return nil, fmt.Errorf("failed to create bot record: %w", err)
		}
		log.Printf("✅ BotService.CreateBot: Bot saved to database successfully")

		// Start asynchronous stack deployment
		go s.deployBotKnowledgeBaseStackAsync(context.Background(), botID, ownerUserID, req)

		return &CreateBotResult{
			Bot:                    bot,
			StackDeploymentStarted: true,
			Message:                fmt.Sprintf("Bot created successfully. Knowledge Base infrastructure deployment started (stack: %s). This may take 5-10 minutes.", stackName),
		}, nil

	} else {
		// PATH 3: Simple bot without Knowledge Base (no RAG, just LLM chat)
		log.Printf("💬 Creating simple chatbot without Knowledge Base for user: %s", ownerUserID)

		// Save bot to database
		log.Printf("💾 BotService.CreateBot: Saving simple bot to database...")
		if err := s.botRepo.Create(ctx, bot); err != nil {
			log.Printf("❌ BotService.CreateBot: Failed to save bot to database: %v", err)
			return nil, fmt.Errorf("failed to create simple bot: %w", err)
		}
		log.Printf("✅ BotService.CreateBot: Simple bot saved to database successfully")

		return &CreateBotResult{
			Bot:                    bot,
			StackDeploymentStarted: false,
			Message:                "Simple chatbot created successfully (no Knowledge Base - LLM only mode)",
		}, nil
	}
}

// deployBotKnowledgeBaseStackAsync deploys the Knowledge Base stack asynchronously
func (s *BotService) deployBotKnowledgeBaseStackAsync(ctx context.Context, botID, ownerUserID string, req *models.CreateBotRequest) {
	log.Printf("🚀 Starting async deployment for bot: %s", botID)
	
	// Add panic recovery to prevent silent failures
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 PANIC in async deployment for bot %s: %v", botID, r)
			// Update status to failed
			failureReason := fmt.Sprintf("Deployment panic: %v", r)
			if err := s.UpdateSyncStatusToFailed(ctx, botID, "", failureReason); err != nil {
				log.Printf("❌ Failed to update panic status for bot %s: %v", botID, err)
			}
		}
	}()
	
	// Update sync status to RUNNING
	execID := uuid.New().String()
	err := s.UpdateSyncStatusToRunning(ctx, botID, execID)
	if err != nil {
		log.Printf("⚠️ Failed to update sync status to running for bot %s: %v", botID, err)
		// Don't return here - try to continue with deployment
	}
	
	// Prepare deployment request
	deployReq := KnowledgeBaseStackRequest{
		BotID:                   botID,
		OwnerUserID:             ownerUserID,
		Instruction:             req.Instruction,
		KnowledgeBaseCreation:   req.KnowledgeBaseCreation,
	}
	
	// Deploy the stack using CloudFormation
	result, err := s.cfnService.CreateKnowledgeBaseStack(ctx, deployReq)
	if err != nil {
		log.Printf("❌ Stack deployment failed for bot %s: %v", botID, err)
		
		// Update sync status to FAILED
		reason := fmt.Sprintf("Stack deployment failed: %v", err)
		if updateErr := s.UpdateSyncStatusToFailed(ctx, botID, execID, reason); updateErr != nil {
			log.Printf("⚠️ Failed to update sync status to failed for bot %s: %v", botID, updateErr)
		}
		
		// Also update stack status
		failedStatus := models.StackStatusCreateFailed
		updateReq := &models.UpdateBotRequest{
			StackStatus: &failedStatus,
		}
		s.botRepo.Update(ctx, botID, updateReq)
		return
	}
	
	log.Printf("✅ Stack deployment completed for bot %s", botID)
	
	// Update sync status to SUCCEEDED
	if updateErr := s.UpdateSyncStatusToSucceeded(ctx, botID, execID); updateErr != nil {
		log.Printf("⚠️ Failed to update sync status to succeeded for bot %s: %v", botID, updateErr)
	}
	
	// Update bot with deployment results
	updateReq := &models.UpdateBotRequest{
		KnowledgeBaseID:     &result.KnowledgeBaseId,
		StackStatus:         &result.Status,
		DocumentBucketName:  &result.S3BucketName,
		// GuardrailArn and GuardrailVersion not available in our simple CloudFormation template
	}
	
	if err := s.botRepo.Update(ctx, botID, updateReq); err != nil {
		log.Printf("❌ Failed to update bot with deployment results: %v", err)
	} else {
		log.Printf("✅ Bot %s updated with Knowledge Base ID: %s", botID, result.KnowledgeBaseId)
	}
}

// GetBot retrieves a bot by ID with full access control
func (s *BotService) GetBot(ctx context.Context, botID, userID string, userGroups []string, isAdmin bool) (*models.Bot, error) {
	bot, err := s.botRepo.GetByID(ctx, botID)
	if err != nil {
		return nil, ErrBotNotFound
	}

	// Check access permissions using the new 3-tier system
	if !bot.IsAccessibleByUser(userID, userGroups, isAdmin) {
		return nil, ErrUnauthorized
	}

	return bot, nil
}

// ListBots retrieves bots based on sharing scope and user permissions
func (s *BotService) ListBots(ctx context.Context, userID string, userGroups []string, isAdmin bool, scope string, starred bool, limit int) ([]models.Bot, error) {
	var allBots []models.Bot
	
	switch scope {
	case "private":
		// Only user's own bots
		userBots, err := s.botRepo.GetByOwner(ctx, userID)
		if err != nil {
			return nil, err
		}
		allBots = userBots
		
	case "public":
		// Only public bots
		publicBots, err := s.botRepo.GetPublicBots(ctx)
		if err != nil {
			return nil, err
		}
		allBots = publicBots
		
	case "shared":
		// Only shared bots (bots shared with this user)
		sharedBots, err := s.botRepo.GetSharedBots(ctx, userID, userGroups)
		if err != nil {
			return nil, err
		}
		allBots = sharedBots
		
	case "accessible":
		// All bots user can access (own + shared + public)
		userBots, err := s.botRepo.GetByOwner(ctx, userID)
		if err != nil {
			return nil, err
		}
		allBots = append(allBots, userBots...)
		
		// Add shared bots (partial scope where user has access)
		sharedBots, err := s.botRepo.GetSharedBots(ctx, userID, userGroups)
		if err != nil {
			return nil, err
		}
		allBots = append(allBots, sharedBots...)
		
		// Add public bots
		publicBots, err := s.botRepo.GetPublicBots(ctx)
		if err != nil {
			return nil, err
		}
		allBots = append(allBots, publicBots...)
		
	default:
		return nil, fmt.Errorf("invalid scope: %s. Must be private, public, shared, or accessible", scope)
	}

	// Filter by starred status if requested
	if starred {
		var starredBots []models.Bot
		for _, bot := range allBots {
			if bot.IsStarred {
				starredBots = append(starredBots, bot)
			}
		}
		allBots = starredBots
	}

	// Remove duplicates and apply access control
	uniqueBots := s.filterAccessibleBots(allBots, userID, userGroups, isAdmin)
	
	// Sort by last used time (most recent first)
	sort.Slice(uniqueBots, func(i, j int) bool {
		return uniqueBots[i].LastUsedTime.After(uniqueBots[j].LastUsedTime)
	})
	
	// Apply limit if specified
	if limit > 0 && len(uniqueBots) > limit {
		uniqueBots = uniqueBots[:limit]
	}

	return uniqueBots, nil
}

// GetBotSummaries returns lightweight bot information for listing
func (s *BotService) GetBotSummaries(ctx context.Context, userID string, userGroups []string, isAdmin bool, scope string, starred bool, limit int) ([]models.BotSummary, error) {
	bots, err := s.ListBots(ctx, userID, userGroups, isAdmin, scope, starred, limit)
	if err != nil {
		return nil, err
	}

	summaries := make([]models.BotSummary, len(bots))
	for i, bot := range bots {
		summaries[i] = bot.ToSummary()
	}

	return summaries, nil
}

// UpdateBotSharing updates the sharing configuration of a bot
func (s *BotService) UpdateBotSharing(ctx context.Context, botID, userID string, sharedScope, sharedStatus string, allowedUsers, allowedGroups []string) error {
	// Get bot and verify edit permissions
	bot, err := s.botRepo.GetByID(ctx, botID)
	if err != nil {
		return ErrBotNotFound
	}
	
	if !bot.IsEditableByUser(userID, []string{}, false) {
		return ErrUnauthorized
	}
	
	// Validate sharing scope
	if !s.isValidSharedScope(sharedScope) {
		return fmt.Errorf("invalid shared scope: %s", sharedScope)
	}
	
	// Update sharing configuration
	updateReq := &models.UpdateBotRequest{
		SharedScope:   &sharedScope,
		SharedStatus:  &sharedStatus,
		AllowedUsers:  allowedUsers,
		AllowedGroups: allowedGroups,
	}
	
	return s.botRepo.Update(ctx, botID, updateReq)
}

// GetBotStackStatus returns the CloudFormation stack status for a bot
func (s *BotService) GetBotStackStatus(ctx context.Context, botID, userID string, userGroups []string, isAdmin bool) (*StackOutputResult, error) {
	// Get bot and verify access
	bot, err := s.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		return nil, err
	}
	
	// Only return stack info if bot has dynamic Knowledge Base
	if !bot.HasDynamicKnowledgeBase() {
		return nil, fmt.Errorf("bot does not have a dynamic knowledge base stack")
	}
	
	// Fetch stack outputs
	return s.stackOutputService.FetchStackOutputs(ctx, botID, bot.OwnerUserID)
}

// DeleteBot deletes a bot and its associated Knowledge Base stack if applicable
func (s *BotService) DeleteBot(ctx context.Context, botID, userID string, userGroups []string, isAdmin bool) error {
	// Get bot and verify edit permissions
	bot, err := s.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		return err
	}
	
	// Only users with edit permissions can delete
	if !bot.IsEditableByUser(userID, userGroups, isAdmin) {
		return ErrUnauthorized
	}
	
	// If bot has dynamic Knowledge Base, destroy the stack
	if bot.HasDynamicKnowledgeBase() {
		log.Printf("🗑️ Destroying Knowledge Base stack for bot: %s", botID)
		
		stackName := fmt.Sprintf("BrChatKbStack%s", botID)
		if err := s.cfnService.DeleteStack(ctx, stackName); err != nil {
			log.Printf("❌ Failed to destroy stack for bot %s: %v", botID, err)
			// Continue with bot deletion even if stack destruction fails
		}
	}
	
	// Delete bot from database
	return s.botRepo.Delete(ctx, botID)
}

// Helper methods

func (s *BotService) validateCreateBotRequest(req *models.CreateBotRequest) error {
	if req.Title == "" {
		return errors.New("title is required")
	}
	
	if req.Instruction == "" {
		return errors.New("instruction is required")
	}
	
	if req.SharedScope == "" {
		return errors.New("shared scope is required")
	}
	
	if !s.isValidSharedScope(req.SharedScope) {
		return fmt.Errorf("invalid shared scope: %s", req.SharedScope)
	}
	
	// Validate partial sharing configuration
	if req.SharedScope == models.SharedScopePartial {
		if len(req.AllowedUsers) == 0 && len(req.AllowedGroups) == 0 {
			return errors.New("partial sharing requires at least one allowed user or group")
		}
	}
	
	// Knowledge base is optional - bots can work as simple chatbots without RAG
	// If neither is provided, bot will function without document retrieval

	// Can't provide both KB options if one is specified
	if req.ExistingKnowledgeBaseID != nil && req.KnowledgeBaseCreation != nil {
		return errors.New("cannot provide both existing knowledge base ID and creation config")
	}
	
	return nil
}

func (s *BotService) isValidSharedScope(scope string) bool {
	return scope == models.SharedScopePrivate || 
		   scope == models.SharedScopePartial || 
		   scope == models.SharedScopePublic
}

func (s *BotService) generateBotID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
}

func (s *BotService) filterAccessibleBots(bots []models.Bot, userID string, userGroups []string, isAdmin bool) []models.Bot {
	seen := make(map[string]bool)
	var accessible []models.Bot
	
	for _, bot := range bots {
		// Skip duplicates
		if seen[bot.ID] {
			continue
		}
		
		// Check access permissions
		if bot.IsAccessibleByUser(userID, userGroups, isAdmin) {
			seen[bot.ID] = true
			accessible = append(accessible, bot)
		}
	}
	
	return accessible
}

// Legacy methods for backward compatibility (simplified versions)

func (s *BotService) UpdateBot(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error {
	return s.botRepo.Update(ctx, botID, updateReq)
}

func (s *BotService) UpdateLastUsedTime(ctx context.Context, botID string) error {
	return s.botRepo.UpdateLastUsedTime(ctx, botID, time.Now())
}

func (s *BotService) ToggleStar(ctx context.Context, botID, userID string, starred bool) error {
	return s.botRepo.UpdateStarred(ctx, botID, starred)
}

// === Bot Aliases Service Methods ===

// GetOrCreateBotAlias gets or creates an alias for a shared bot
func (s *BotService) GetOrCreateBotAlias(ctx context.Context, originalBotID, userID string, userGroups []string) (*models.BotAlias, error) {
	// First, get the original bot to verify access and get metadata
	originalBot, err := s.GetBot(ctx, originalBotID, userID, userGroups, false)
	if err != nil {
		return nil, err
	}

	// If user owns the bot, they don't need an alias
	if originalBot.OwnerUserID == userID {
		return nil, nil // No alias needed for owned bots
	}

	// Check if alias already exists
	existingAlias, err := s.botRepo.GetAliasForUserBot(ctx, userID, originalBotID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing alias: %w", err)
	}

	if existingAlias != nil {
		// Update the accessibility status of existing alias
		isAccessible := originalBot.IsAccessibleByUser(userID, userGroups, false)
		if existingAlias.IsOriginAccessible != isAccessible {
			err = s.botRepo.UpdateAlias(ctx, existingAlias.ID, map[string]interface{}{
				"isOriginAccessible": isAccessible,
			})
			if err != nil {
				log.Printf("Warning: failed to update alias accessibility: %v", err)
			}
			existingAlias.IsOriginAccessible = isAccessible
		}
		return existingAlias, nil
	}

	// Create new alias
	aliasID := s.generateBotID() // Generate unique alias ID
	newAlias := &models.BotAlias{
		ID:                   aliasID,
		OriginalBotID:        originalBotID,
		UserID:               userID,
		Title:                originalBot.Title,
		Description:          originalBot.Description,
		OwnerUserID:          originalBot.OwnerUserID,
		IsStarred:            false,
		IsOriginAccessible:   true,
		SharedScope:          originalBot.SharedScope,
		SharedStatus:         originalBot.SharedStatus,
		HasKnowledgeBase:     originalBot.KnowledgeBaseID != nil && *originalBot.KnowledgeBaseID != "",
		ConversationStarters: originalBot.ConversationStarters,
	}

	err = s.botRepo.CreateAlias(ctx, newAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot alias: %w", err)
	}

	return newAlias, nil
}

// GetBotWithAlias retrieves a bot and creates/updates alias if it's a shared bot
func (s *BotService) GetBotWithAlias(ctx context.Context, botID, userID string, userGroups []string, isAdmin bool) (*models.Bot, *models.BotAlias, error) {
	// Get the original bot
	bot, err := s.GetBot(ctx, botID, userID, userGroups, isAdmin)
	if err != nil {
		return nil, nil, err
	}

	// If user owns the bot, no alias needed
	if bot.OwnerUserID == userID {
		return bot, nil, nil
	}

	// Get or create alias for shared bot
	alias, err := s.GetOrCreateBotAlias(ctx, botID, userID, userGroups)
	if err != nil {
		return nil, nil, err
	}

	return bot, alias, nil
}

// UpdateAliasUsage updates the last used time for a bot alias
func (s *BotService) UpdateAliasUsage(ctx context.Context, aliasID string) error {
	return s.botRepo.UpdateAliasLastUsedTime(ctx, aliasID, time.Now())
}

// ToggleAliasStarred toggles the starred status of a bot alias
func (s *BotService) ToggleAliasStarred(ctx context.Context, aliasID string, starred bool) error {
	return s.botRepo.UpdateAliasStarred(ctx, aliasID, starred)
}

// GetUserBotAliases gets all bot aliases for a user
func (s *BotService) GetUserBotAliases(ctx context.Context, userID string) ([]models.BotAlias, error) {
	return s.botRepo.GetAliasesByUser(ctx, userID)
}

// DeleteBotAlias deletes a bot alias (removes shared bot from user's list)
func (s *BotService) DeleteBotAlias(ctx context.Context, aliasID, userID string) error {
	// Verify the alias and check edit permissions
	alias, err := s.botRepo.GetAliasByID(ctx, aliasID)
	if err != nil {
		return err
	}
	if alias == nil {
		return ErrBotNotFound
	}
	if !alias.IsEditableByUser(userID, []string{}, false) {
		return ErrUnauthorized
	}

	return s.botRepo.DeleteAlias(ctx, aliasID)
}

// GetMixedBotList gets a mixed list of owned bots and aliases (for user's bot list)
func (s *BotService) GetMixedBotList(ctx context.Context, userID string, userGroups []string, starred bool, limit int) ([]models.BotSummary, error) {
	var allSummaries []models.BotSummary

	// Get owned bots
	ownedBots, err := s.botRepo.GetByOwner(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get owned bots: %w", err)
	}

	// Filter owned bots if starred is requested
	for _, bot := range ownedBots {
		if !starred || bot.IsStarred {
			allSummaries = append(allSummaries, bot.ToSummary())
		}
	}

	// Get bot aliases
	aliases, err := s.botRepo.GetAliasesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot aliases: %w", err)
	}

	// Filter aliases if starred is requested
	for _, alias := range aliases {
		if alias.IsOriginAccessible && (!starred || alias.IsStarred) {
			allSummaries = append(allSummaries, alias.ToSummary())
		}
	}

	// Sort by last used time (most recent first)
	sort.Slice(allSummaries, func(i, j int) bool {
		return allSummaries[i].LastUsedTime.After(allSummaries[j].LastUsedTime)
	})

	// Apply limit
	if limit > 0 && len(allSummaries) > limit {
		allSummaries = allSummaries[:limit]
	}

	return allSummaries, nil
}

// === Sync Status Management Methods ===

// UpdateSyncStatus updates the sync status of a bot during async processing
func (s *BotService) UpdateSyncStatus(ctx context.Context, botID, execID, status, reason string) error {
	updateReq := &models.UpdateBotRequest{
		SyncStatus:       &status,
		SyncLastExecID:   &execID,
		SyncStatusReason: &reason,
	}
	
	return s.botRepo.Update(ctx, botID, updateReq)
}

// UpdateSyncStatusToRunning marks a bot's sync status as running
func (s *BotService) UpdateSyncStatusToRunning(ctx context.Context, botID, execID string) error {
	return s.UpdateSyncStatus(ctx, botID, execID, models.SyncStatusRunning, "Knowledge Base creation in progress")
}

// UpdateSyncStatusToSucceeded marks a bot's sync status as succeeded
func (s *BotService) UpdateSyncStatusToSucceeded(ctx context.Context, botID, execID string) error {
	return s.UpdateSyncStatus(ctx, botID, execID, models.SyncStatusSucceeded, "Knowledge Base created successfully")
}

// UpdateSyncStatusToFailed marks a bot's sync status as failed
func (s *BotService) UpdateSyncStatusToFailed(ctx context.Context, botID, execID, reason string) error {
	return s.UpdateSyncStatus(ctx, botID, execID, models.SyncStatusFailed, reason)
}

// GetBotSyncStatus retrieves the current sync status of a bot
func (s *BotService) GetBotSyncStatus(ctx context.Context, botID string) (string, string, error) {
	bot, err := s.botRepo.GetByID(ctx, botID)
	if err != nil {
		return "", "", err
	}
	if bot == nil {
		return "", "", ErrBotNotFound
	}
	
	return bot.SyncStatus, bot.SyncStatusReason, nil
}

// === Bot Pinning Management Methods ===

// PinBot pins a public bot with the specified order
func (s *BotService) PinBot(ctx context.Context, botID string, userID string, isAdmin bool, order int) error {
	// Only admins can pin bots
	if !isAdmin {
		return ErrUnauthorized
	}

	// Get the bot to verify it can be pinned
	bot, err := s.botRepo.GetByID(ctx, botID)
	if err != nil {
		return err
	}
	if bot == nil {
		return ErrBotNotFound
	}

	// Check if bot can be pinned
	if !bot.CanBePinned() {
		return fmt.Errorf("bot cannot be pinned: must be public and shared")
	}

	// Check if the order is already taken
	existingBot, err := s.GetBotByPinOrder(ctx, order)
	if err != nil && err != ErrBotNotFound {
		return fmt.Errorf("failed to check pin order: %w", err)
	}
	if existingBot != nil && existingBot.ID != botID {
		return fmt.Errorf("pin order %d is already taken by bot: %s", order, existingBot.ID)
	}

	// Set the pinned status
	pinnedStatus := fmt.Sprintf("pinned@%03d", order)
	updateReq := &models.UpdateBotRequest{
		SharedStatus: &pinnedStatus,
	}

	return s.botRepo.Update(ctx, botID, updateReq)
}

// UnpinBot removes the pinned status from a bot
func (s *BotService) UnpinBot(ctx context.Context, botID string, userID string, isAdmin bool) error {
	// Only admins can unpin bots
	if !isAdmin {
		return ErrUnauthorized
	}

	// Get the bot to verify it's pinned
	bot, err := s.botRepo.GetByID(ctx, botID)
	if err != nil {
		return err
	}
	if bot == nil {
		return ErrBotNotFound
	}

	// Check if bot is actually pinned
	if !bot.IsPinned() {
		return fmt.Errorf("bot is not pinned")
	}

	// Remove pinned status
	sharedStatus := models.SharedStatusShared
	updateReq := &models.UpdateBotRequest{
		SharedStatus: &sharedStatus,
	}

	return s.botRepo.Update(ctx, botID, updateReq)
}

// GetBotByPinOrder finds a bot by its pin order
func (s *BotService) GetBotByPinOrder(ctx context.Context, order int) (*models.Bot, error) {
	// Get all pinned bots
	pinnedBots, err := s.botRepo.GetPinnedBots(ctx)
	if err != nil {
		return nil, err
	}

	// Find bot with matching order
	for _, bot := range pinnedBots {
		if bot.GetPinOrder() == order {
			return &bot, nil
		}
	}

	return nil, ErrBotNotFound
}

// GetNextAvailablePinOrder finds the next available pin order number
func (s *BotService) GetNextAvailablePinOrder(ctx context.Context) (int, error) {
	// Get all pinned bots
	pinnedBots, err := s.botRepo.GetPinnedBots(ctx)
	if err != nil {
		return 0, err
	}

	// Extract all existing orders
	usedOrders := make(map[int]bool)
	for _, bot := range pinnedBots {
		order := bot.GetPinOrder()
		if order > 0 {
			usedOrders[order] = true
		}
	}

	// Find the first available order starting from 1
	for order := 1; order <= 999; order++ {
		if !usedOrders[order] {
			return order, nil
		}
	}

	return 0, fmt.Errorf("no available pin orders (max 999 pinned bots)")
}

// GetPinnedBotsOrdered returns pinned bots sorted by pin order
func (s *BotService) GetPinnedBotsOrdered(ctx context.Context) ([]models.Bot, error) {
	pinnedBots, err := s.botRepo.GetPinnedBots(ctx)
	if err != nil {
		return nil, err
	}

	// Sort by pin order
	sort.Slice(pinnedBots, func(i, j int) bool {
		return pinnedBots[i].GetPinOrder() < pinnedBots[j].GetPinOrder()
	})

	return pinnedBots, nil
}

// === Usage Analytics Methods ===

// IncrementBotUsage increments usage count and updates last used time
func (s *BotService) IncrementBotUsage(ctx context.Context, botID string) error {
	// Increment usage count atomically
	if err := s.botRepo.IncrementUsageCount(ctx, botID); err != nil {
		return fmt.Errorf("failed to increment usage count: %w", err)
	}

	// Update last used time
	if err := s.botRepo.UpdateLastUsedTime(ctx, botID, time.Now()); err != nil {
		log.Printf("Warning: failed to update last used time for bot %s: %v", botID, err)
		// Don't fail the request if last used time update fails
	}

	return nil
}

// GetPopularBots returns bots sorted by popularity (usage count with time decay)
func (s *BotService) GetPopularBots(ctx context.Context, userID string, userGroups []string, limit int) ([]models.BotSummary, error) {
	// Get all accessible bots
	var allBots []models.Bot
	
	// Get public bots
	publicBots, err := s.botRepo.GetPublicBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get public bots: %w", err)
	}
	allBots = append(allBots, publicBots...)
	
	// Get shared bots accessible to user
	sharedBots, err := s.botRepo.GetSharedBots(ctx, userID, userGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared bots: %w", err)
	}
	allBots = append(allBots, sharedBots...)
	
	// Filter out bots user can't access and apply access control
	filteredBots := s.filterAccessibleBots(allBots, userID, userGroups, false)
	
	// Sort by popularity score (highest first)
	sort.Slice(filteredBots, func(i, j int) bool {
		return filteredBots[i].GetPopularityScore() > filteredBots[j].GetPopularityScore()
	})
	
	// Apply limit
	if limit > 0 && len(filteredBots) > limit {
		filteredBots = filteredBots[:limit]
	}
	
	// Convert to summaries
	summaries := make([]models.BotSummary, len(filteredBots))
	for i, bot := range filteredBots {
		summaries[i] = bot.ToSummary()
	}
	
	return summaries, nil
}

// GetBotsByUsageCount returns bots sorted by raw usage count (for admin analytics)
func (s *BotService) GetBotsByUsageCount(ctx context.Context, limit int) ([]models.Bot, error) {
	// Get all public bots (admin feature - could expand to all bots)
	bots, err := s.botRepo.GetPublicBots(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bots: %w", err)
	}
	
	// Sort by usage count (highest first)
	sort.Slice(bots, func(i, j int) bool {
		return bots[i].UsageCount > bots[j].UsageCount
	})
	
	// Apply limit
	if limit > 0 && len(bots) > limit {
		bots = bots[:limit]
	}
	
	return bots, nil
}

