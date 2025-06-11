package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
	"github.com/bobbyrathore/go-lambda-backend/pkg/repositories"
)

// ChatService provides methods for chat functionality
type ChatService struct {
	chatRepo    repositories.ChatRepository
	botRepo     repositories.BotRepository
	bedrockSvc  *BedrockService
}

// NewChatService creates a new ChatService
func NewChatService(
	chatRepo repositories.ChatRepository,
	botRepo repositories.BotRepository,
	bedrockSvc *BedrockService,
) *ChatService {
	return &ChatService{
		chatRepo:    chatRepo,
		botRepo:     botRepo,
		bedrockSvc:  bedrockSvc,
	}
}

// CreateChatMessage creates a new chat message and gets a response
func (s *ChatService) CreateChatMessage(ctx context.Context, input models.ChatInput, userID string) (*models.ChatResponse, error) {
	// Get the bot information
	bot, err := s.botRepo.GetByID(ctx, input.BotID)
	if err != nil {
		return nil, err
	}

	// Add system prompt from bot if not provided
	if input.SystemPrompt == "" && bot.Instruction != "" {
		input.SystemPrompt = bot.Instruction
	}

	// Set default model from bot if not provided
	if input.ModelID == "" && len(bot.ActiveModels) > 0 {
		input.ModelID = bot.ActiveModels[0]
	}

	// Set temperature and max tokens from bot if not provided
	if input.Temperature == 0 && bot.GenerationParams.Temperature > 0 {
		input.Temperature = bot.GenerationParams.Temperature
	}

	if input.MaxTokens == 0 && bot.GenerationParams.MaxTokens > 0 {
		input.MaxTokens = bot.GenerationParams.MaxTokens
	}

	// Generate response from Bedrock
	response, err := s.bedrockSvc.GenerateResponse(ctx, &input)
	if err != nil {
		return nil, err
	}

	// Create conversation if it doesn't exist
	var conversationID string
	if input.ConversationID == "" {
		// Create new conversation
		conversation := models.Conversation{
			ID:        uuid.New().String(),
			UserID:    userID,
			BotID:     input.BotID,
			Title:     s.generateConversationTitle(input.Messages),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		if err := s.chatRepo.CreateConversation(ctx, &conversation); err != nil {
			return nil, err
		}

		conversationID = conversation.ID
	} else {
		conversationID = input.ConversationID
	}

	// Save user message
	userMessage := models.Message{
		ID:             uuid.New().String(),
		ConversationID: conversationID,
		Role:           "user",
		Content:        input.Messages[len(input.Messages)-1].Content,
		CreatedAt:      time.Now().UTC(),
	}

	if err := s.chatRepo.CreateMessage(ctx, &userMessage); err != nil {
		return nil, err
	}

	// Save assistant response
	assistantMessageID := uuid.New().String()
	assistantMessage := models.Message{
		ID:             assistantMessageID,
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        response.Content,
		CreatedAt:      time.Now().UTC(),
		TokenCount:     response.TokenCount,
	}

	if err := s.chatRepo.CreateMessage(ctx, &assistantMessage); err != nil {
		return nil, err
	}

	// Update conversation timestamp and message count
	if err := s.chatRepo.UpdateConversation(ctx, conversationID, userID); err != nil {
		return nil, err
	}

	// Set conversation ID and message ID in response
	response.ConversationID = conversationID
	response.MessageID = assistantMessageID

	return response, nil
}

// GetConversation gets a conversation by ID
func (s *ChatService) GetConversation(ctx context.Context, conversationID string, userID string) (*models.ConversationWithMessages, error) {
	// Get conversation
	conversation, err := s.chatRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}

	// Get messages
	messages, err := s.chatRepo.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	return &models.ConversationWithMessages{
		Conversation: *conversation,
		Messages:     messages,
	}, nil
}

// ListConversations lists all conversations for a user
func (s *ChatService) ListConversations(ctx context.Context, userID string, limit int, nextToken string) (*models.ConversationsResponse, error) {
	conversations, newNextToken, err := s.chatRepo.ListConversations(ctx, userID, limit, nextToken)
	if err != nil {
		return nil, err
	}

	return &models.ConversationsResponse{
		Conversations: conversations,
		Count:         len(conversations),
		NextToken:     newNextToken,
	}, nil
}

// DeleteConversation deletes a conversation
func (s *ChatService) DeleteConversation(ctx context.Context, conversationID string, userID string) error {
	// Check if conversation exists and belongs to user
	_, err := s.chatRepo.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	// Delete all messages first
	if err := s.chatRepo.DeleteMessages(ctx, conversationID); err != nil {
		return err
	}

	// Delete conversation
	return s.chatRepo.DeleteConversation(ctx, conversationID, userID)
}

// generateConversationTitle generates a title for a new conversation
func (s *ChatService) generateConversationTitle(messages []models.ChatMessageInput) string {
	if len(messages) > 0 {
		content := messages[0].Content
		if len(content) > 50 {
			return content[:47] + "..."
		}
		return content
	}
	return "New Conversation"
}