package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// ChatRepository provides methods to interact with chat storage
type ChatRepository interface {
	CreateConversation(ctx context.Context, conversation *models.Conversation) error
	GetConversation(ctx context.Context, conversationID string, userID string) (*models.Conversation, error)
	ListConversations(ctx context.Context, userID string, limit int, nextToken string) ([]models.Conversation, string, error)
	UpdateConversation(ctx context.Context, conversationID string, userID string) error
	DeleteConversation(ctx context.Context, conversationID string, userID string) error
	CreateMessage(ctx context.Context, message *models.Message) error
	GetMessages(ctx context.Context, conversationID string) ([]models.Message, error)
	DeleteMessages(ctx context.Context, conversationID string) error
}

// DynamoChatRepository implements ChatRepository using DynamoDB
type DynamoChatRepository struct {
	client                *dynamodb.Client
	conversationsTable    string
	messagesTable         string
}

// NewChatRepository creates a new DynamoChatRepository
func NewChatRepository(client *dynamodb.Client, conversationsTable, messagesTable string) ChatRepository {
	return &DynamoChatRepository{
		client:                client,
		conversationsTable:    conversationsTable,
		messagesTable:         messagesTable,
	}
}

// CreateConversation creates a new conversation in DynamoDB
func (r *DynamoChatRepository) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	item, err := attributevalue.MarshalMap(conversation)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.conversationsTable),
		Item:      item,
	})

	return err
}

// GetConversation gets a conversation by ID
func (r *DynamoChatRepository) GetConversation(ctx context.Context, conversationID string, userID string) (*models.Conversation, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.conversationsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: conversationID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	var conversation models.Conversation
	if err := attributevalue.UnmarshalMap(result.Item, &conversation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}

	// Check if user owns this conversation
	if conversation.UserID != userID {
		return nil, models.ErrNotAuthorized
	}

	return &conversation, nil
}

// ListConversations lists all conversations for a user
func (r *DynamoChatRepository) ListConversations(ctx context.Context, userID string, limit int, nextToken string) ([]models.Conversation, string, error) {
	queryInput := &dynamodb.ScanInput{
		TableName:        aws.String(r.conversationsTable),
		FilterExpression: aws.String("userId = :userID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userID": &types.AttributeValueMemberS{Value: userID},
		},
		Limit: aws.Int32(int32(limit)),
	}

	// Handle pagination
	if nextToken != "" {
		// In a real implementation, you'd decode the nextToken
		// For simplicity, we'll skip pagination logic here
	}

	result, err := r.client.Scan(ctx, queryInput)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list conversations: %w", err)
	}

	var conversations []models.Conversation
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &conversations); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal conversations: %w", err)
	}

	var newNextToken string
	if result.LastEvaluatedKey != nil {
		newNextToken = "next"
	}

	return conversations, newNextToken, nil
}

// UpdateConversation updates a conversation's timestamp and message count
func (r *DynamoChatRepository) UpdateConversation(ctx context.Context, conversationID string, userID string) error {
	// First, get the current conversation to ensure it exists and user owns it
	conversation, err := r.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	// Update timestamp and increment message count
	conversation.UpdatedAt = time.Now().UTC()
	conversation.MessageCount++

	item, err := attributevalue.MarshalMap(conversation)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.conversationsTable),
		Item:      item,
	})

	return err
}

// DeleteConversation deletes a conversation
func (r *DynamoChatRepository) DeleteConversation(ctx context.Context, conversationID string, userID string) error {
	// First verify the user owns the conversation
	_, err := r.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.conversationsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: conversationID},
		},
	})

	return err
}

// CreateMessage creates a new message in DynamoDB
func (r *DynamoChatRepository) CreateMessage(ctx context.Context, message *models.Message) error {
	item, err := attributevalue.MarshalMap(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.messagesTable),
		Item:      item,
	})

	return err
}

// GetMessages gets all messages for a conversation
func (r *DynamoChatRepository) GetMessages(ctx context.Context, conversationID string) ([]models.Message, error) {
	queryInput := &dynamodb.ScanInput{
		TableName:        aws.String(r.messagesTable),
		FilterExpression: aws.String("conversationId = :conversationID"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":conversationID": &types.AttributeValueMemberS{Value: conversationID},
		},
	}

	result, err := r.client.Scan(ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var messages []models.Message
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

// DeleteMessages deletes all messages for a conversation
func (r *DynamoChatRepository) DeleteMessages(ctx context.Context, conversationID string) error {
	// First, get all messages for the conversation
	messages, err := r.GetMessages(ctx, conversationID)
	if err != nil {
		return err
	}

	// Delete each message
	for _, message := range messages {
		_, err = r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(r.messagesTable),
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{Value: message.ID},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to delete message %s: %w", message.ID, err)
		}
	}

	return nil
}