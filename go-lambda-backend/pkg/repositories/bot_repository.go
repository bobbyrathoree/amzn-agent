package repositories

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// BotRepository provides methods to interact with bot storage
type BotRepository interface {
	Create(ctx context.Context, bot *models.Bot) error
	GetByID(ctx context.Context, botID string) (*models.Bot, error)
	GetByOwner(ctx context.Context, ownerUserID string) ([]models.Bot, error)
	GetPublicBots(ctx context.Context) ([]models.Bot, error)
	Update(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error
	Delete(ctx context.Context, botID string) error
	UpdateLastUsedTime(ctx context.Context, botID string, lastUsedTime time.Time) error
	UpdateStarred(ctx context.Context, botID string, starred bool) error
}

// DynamoBotRepository handles data persistence for bots using DynamoDB
type DynamoBotRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewBotRepository creates a new BotRepository
func NewBotRepository(client *dynamodb.Client, tableName string) BotRepository {
	return &DynamoBotRepository{
		client:    client,
		tableName: tableName,
	}
}

// Create creates a new bot in DynamoDB
func (r *DynamoBotRepository) Create(ctx context.Context, bot *models.Bot) error {
	item, err := attributevalue.MarshalMap(bot)
	if err != nil {
		return err
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})

	return err
}

// GetByID retrieves a bot by its ID
func (r *DynamoBotRepository) GetByID(ctx context.Context, botID string) (*models.Bot, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, nil
	}

	var bot models.Bot
	err = attributevalue.UnmarshalMap(result.Item, &bot)
	if err != nil {
		return nil, err
	}

	return &bot, nil
}

// GetByOwner retrieves all bots owned by a specific user
func (r *DynamoBotRepository) GetByOwner(ctx context.Context, ownerUserID string) ([]models.Bot, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("ownerUserId = :ownerUserId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ownerUserId": &types.AttributeValueMemberS{Value: ownerUserID},
		},
	})
	if err != nil {
		return nil, err
	}

	var bots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &bots)
	if err != nil {
		return nil, err
	}

	return bots, nil
}

// GetPublicBots retrieves all public bots
func (r *DynamoBotRepository) GetPublicBots(ctx context.Context) ([]models.Bot, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("isPublic = :isPublic"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":isPublic": &types.AttributeValueMemberBOOL{Value: true},
		},
	})
	if err != nil {
		return nil, err
	}

	var bots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &bots)
	if err != nil {
		return nil, err
	}

	return bots, nil
}

// Update updates an existing bot
func (r *DynamoBotRepository) Update(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error {
	updateExpression := "SET "
	expressionAttributeValues := make(map[string]types.AttributeValue)
	expressionAttributeNames := make(map[string]string)
	updates := []string{}

	if updateReq.Title != nil {
		updates = append(updates, "#title = :title")
		expressionAttributeNames["#title"] = "title"
		expressionAttributeValues[":title"] = &types.AttributeValueMemberS{Value: *updateReq.Title}
	}

	if updateReq.Description != nil {
		updates = append(updates, "#description = :description")
		expressionAttributeNames["#description"] = "description"
		expressionAttributeValues[":description"] = &types.AttributeValueMemberS{Value: *updateReq.Description}
	}

	if updateReq.Instruction != nil {
		updates = append(updates, "instruction = :instruction")
		expressionAttributeValues[":instruction"] = &types.AttributeValueMemberS{Value: *updateReq.Instruction}
	}

	if updateReq.IsPublic != nil {
		updates = append(updates, "isPublic = :isPublic")
		expressionAttributeValues[":isPublic"] = &types.AttributeValueMemberBOOL{Value: *updateReq.IsPublic}
	}

	if updateReq.IsStarred != nil {
		updates = append(updates, "isStarred = :isStarred")
		expressionAttributeValues[":isStarred"] = &types.AttributeValueMemberBOOL{Value: *updateReq.IsStarred}
	}

	if updateReq.KnowledgeBaseID != nil {
		updates = append(updates, "knowledgeBaseId = :knowledgeBaseId")
		expressionAttributeValues[":knowledgeBaseId"] = &types.AttributeValueMemberS{Value: *updateReq.KnowledgeBaseID}
	}

	if updateReq.DisplayRetrievedChunks != nil {
		updates = append(updates, "displayRetrievedChunks = :displayRetrievedChunks")
		expressionAttributeValues[":displayRetrievedChunks"] = &types.AttributeValueMemberBOOL{Value: *updateReq.DisplayRetrievedChunks}
	}

	if updateReq.GenerationParams != nil {
		genParams, err := attributevalue.Marshal(updateReq.GenerationParams)
		if err != nil {
			return err
		}
		updates = append(updates, "generationParams = :generationParams")
		expressionAttributeValues[":generationParams"] = genParams
	}

	if updateReq.KnowledgeBaseConfig != nil {
		kbConfig, err := attributevalue.Marshal(updateReq.KnowledgeBaseConfig)
		if err != nil {
			return err
		}
		updates = append(updates, "knowledgeBaseConfig = :knowledgeBaseConfig")
		expressionAttributeValues[":knowledgeBaseConfig"] = kbConfig
	}

	if len(updateReq.ConversationStarters) > 0 {
		starters, err := attributevalue.Marshal(updateReq.ConversationStarters)
		if err != nil {
			return err
		}
		updates = append(updates, "conversationStarters = :conversationStarters")
		expressionAttributeValues[":conversationStarters"] = starters
	}

	if len(updateReq.ActiveModels) > 0 {
		models, err := attributevalue.Marshal(updateReq.ActiveModels)
		if err != nil {
			return err
		}
		updates = append(updates, "activeModels = :activeModels")
		expressionAttributeValues[":activeModels"] = models
	}

	if len(updateReq.AgentTools) > 0 {
		tools, err := attributevalue.Marshal(updateReq.AgentTools)
		if err != nil {
			return err
		}
		updates = append(updates, "agentTools = :agentTools")
		expressionAttributeValues[":agentTools"] = tools
	}

	if len(updates) == 0 {
		return nil // No updates to perform
	}

	updateExpression += joinStrings(updates, ", ")

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression:       aws.String("attribute_exists(id)"),
	}

	if len(expressionAttributeNames) > 0 {
		input.ExpressionAttributeNames = expressionAttributeNames
	}

	_, err := r.client.UpdateItem(ctx, input)
	return err
}

// Delete deletes a bot by its ID
func (r *DynamoBotRepository) Delete(ctx context.Context, botID string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	return err
}

// UpdateLastUsedTime updates the last used timestamp for a bot
func (r *DynamoBotRepository) UpdateLastUsedTime(ctx context.Context, botID string, lastUsedTime time.Time) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		UpdateExpression: aws.String("SET lastUsedTime = :lastUsedTime"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":lastUsedTime": &types.AttributeValueMemberS{Value: lastUsedTime.Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	return err
}

// UpdateStarred updates the starred status of a bot
func (r *DynamoBotRepository) UpdateStarred(ctx context.Context, botID string, starred bool) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		UpdateExpression: aws.String("SET isStarred = :isStarred"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":isStarred": &types.AttributeValueMemberBOOL{Value: starred},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	return err
}

// Helper function to join strings
func joinStrings(strings []string, separator string) string {
	if len(strings) == 0 {
		return ""
	}
	if len(strings) == 1 {
		return strings[0]
	}

	result := strings[0]
	for i := 1; i < len(strings); i++ {
		result += separator + strings[i]
	}
	return result
}