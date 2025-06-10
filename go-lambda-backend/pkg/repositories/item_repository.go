package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// ItemRepository defines methods for interacting with items in the database
type ItemRepository interface {
	GetItem(ctx context.Context, id string) (*models.Item, error)
	ListItems(ctx context.Context, category string, limit int, nextToken string) ([]models.Item, string, error)
	CreateItem(ctx context.Context, req models.CreateItemRequest) (*models.Item, error)
	UpdateItem(ctx context.Context, id string, req models.UpdateItemRequest) (*models.Item, error)
	DeleteItem(ctx context.Context, id string) error
}

// DynamoDBItemRepository implements ItemRepository using DynamoDB
type DynamoDBItemRepository struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoDBItemRepository creates a new DynamoDBItemRepository
func NewDynamoDBItemRepository(client *dynamodb.Client, tableName string) *DynamoDBItemRepository {
	return &DynamoDBItemRepository{
		client:    client,
		tableName: tableName,
	}
}

// GetItem fetches an item by ID from DynamoDB
func (r *DynamoDBItemRepository) GetItem(ctx context.Context, id string) (*models.Item, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, err
	}

	if result.Item == nil {
		return nil, errors.New("item not found")
	}

	var item models.Item
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// ListItems retrieves a paginated list of items
func (r *DynamoDBItemRepository) ListItems(ctx context.Context, category string, limit int, nextToken string) ([]models.Item, string, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String("category-index"),
		KeyConditionExpression: aws.String("category = :category"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":category": &types.AttributeValueMemberS{Value: category},
		},
		Limit: aws.Int32(int32(limit)),
	}

	// Handle pagination
	if nextToken != "" {
		input.ExclusiveStartKey = map[string]types.AttributeValue{
			"id":       &types.AttributeValueMemberS{Value: nextToken},
			"category": &types.AttributeValueMemberS{Value: category},
		}
	}

	result, err := r.client.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}

	var items []models.Item
	err = attributevalue.UnmarshalListOfMaps(result.Items, &items)
	if err != nil {
		return nil, "", err
	}

	// Handle pagination token for next page
	var newNextToken string
	if result.LastEvaluatedKey != nil {
		if idAttr, ok := result.LastEvaluatedKey["id"]; ok {
			if id, ok := idAttr.(*types.AttributeValueMemberS); ok {
				newNextToken = id.Value
			}
		}
	}

	return items, newNextToken, nil
}

// CreateItem creates a new item in DynamoDB
func (r *DynamoDBItemRepository) CreateItem(ctx context.Context, req models.CreateItemRequest) (*models.Item, error) {
	now := time.Now()
	id := uuid.New().String()

	item := models.Item{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Status:      "active",
		Metadata: models.Metadata{
			Tags:       req.Tags,
			Attributes: req.Attributes,
			UserID:     getUserIDFromContext(ctx),
			VersionInfo: models.VersionInfo{
				Version:   1,
				ChangedBy: getUserIDFromContext(ctx),
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return nil, err
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// UpdateItem updates an existing item in DynamoDB
func (r *DynamoDBItemRepository) UpdateItem(ctx context.Context, id string, req models.UpdateItemRequest) (*models.Item, error) {
	// First, get the current item
	item, err := r.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Category != nil {
		item.Category = *req.Category
	}
	if req.Status != nil {
		item.Status = *req.Status
	}
	if req.Tags != nil {
		item.Metadata.Tags = req.Tags
	}
	if req.Attributes != nil {
		item.Metadata.Attributes = req.Attributes
	}

	// Update metadata
	item.UpdatedAt = time.Now()
	item.Metadata.VersionInfo.Version++
	item.Metadata.VersionInfo.ChangedBy = getUserIDFromContext(ctx)
	item.Metadata.VersionInfo.PublishedAt = time.Now()

	// Save the updated item
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return nil, err
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      av,
	})
	if err != nil {
		return nil, err
	}

	return item, nil
}

// DeleteItem removes an item from DynamoDB
func (r *DynamoDBItemRepository) DeleteItem(ctx context.Context, id string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	return err
}

// getUserIDFromContext extracts the user ID from the context
func getUserIDFromContext(ctx context.Context) string {
	// In a real application, you would get the authenticated user's ID from the context
	// This is just a placeholder implementation
	if userID, ok := ctx.Value("userID").(string); ok {
		return userID
	}
	return "anonymous"
}