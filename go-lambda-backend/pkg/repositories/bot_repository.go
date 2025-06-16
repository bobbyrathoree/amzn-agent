package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	// AWS SDK v1 for workaround
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodbv1 "github.com/aws/aws-sdk-go/service/dynamodb"
	awsv1 "github.com/aws/aws-sdk-go/aws"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// BotRepository provides methods to interact with full-featured bot storage
type BotRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, bot *models.Bot) error
	GetByID(ctx context.Context, botID string) (*models.Bot, error)
	Update(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error
	Delete(ctx context.Context, botID string) error
	
	// Ownership and listing
	GetByOwner(ctx context.Context, ownerUserID string) ([]models.Bot, error)
	
	// Sharing and discovery (3-tier system)
	GetPublicBots(ctx context.Context) ([]models.Bot, error)
	GetSharedBots(ctx context.Context, userID string, userGroups []string) ([]models.Bot, error)
	GetPinnedBots(ctx context.Context) ([]models.Bot, error)
	
	// Bot interaction tracking
	UpdateLastUsedTime(ctx context.Context, botID string, lastUsedTime time.Time) error
	UpdateStarred(ctx context.Context, botID string, starred bool) error
	IncrementUsageCount(ctx context.Context, botID string) error
	
	// Search and filtering
	SearchBots(ctx context.Context, query string, scope string, userID string, userGroups []string) ([]models.Bot, error)
	GetBotsByScope(ctx context.Context, scope string, userID string, userGroups []string, limit int) ([]models.Bot, error)
	
	// Bot Aliases operations
	CreateAlias(ctx context.Context, alias *models.BotAlias) error
	GetAliasByID(ctx context.Context, aliasID string) (*models.BotAlias, error)
	GetAliasesByUser(ctx context.Context, userID string) ([]models.BotAlias, error)
	GetAliasForUserBot(ctx context.Context, userID, originalBotID string) (*models.BotAlias, error)
	UpdateAlias(ctx context.Context, aliasID string, updates map[string]interface{}) error
	DeleteAlias(ctx context.Context, aliasID string) error
	UpdateAliasLastUsedTime(ctx context.Context, aliasID string, lastUsedTime time.Time) error
	UpdateAliasStarred(ctx context.Context, aliasID string, starred bool) error
}

// DynamoBotRepository handles data persistence for full-featured bots using DynamoDB
type DynamoBotRepository struct {
	client     *dynamodb.Client  // Keep for compatibility
	clientV1   *dynamodbv1.DynamoDB // SDK v1 for actual operations
	tableName  string
	useV1Fallback bool
}

// NewBotRepository creates a new full-featured BotRepository with SDK v1 workaround
func NewBotRepository(client *dynamodb.Client, tableName string) BotRepository {
	// Create SDK v1 client as workaround for ResolveEndpointV2 issues
	var clientV1 *dynamodbv1.DynamoDB
	useV1Fallback := false
	
	sess, err := session.NewSession()
	if err != nil {
		log.Printf("⚠️ Failed to create AWS SDK v1 session: %v", err)
	} else {
		clientV1 = dynamodbv1.New(sess)
		// Test SDK v1 connectivity
		_, testErr := clientV1.ListTables(&dynamodbv1.ListTablesInput{})
		if testErr != nil {
			log.Printf("⚠️ AWS SDK v1 test failed: %v", testErr)
		} else {
			log.Printf("✅ AWS SDK v1 DynamoDB working - using as workaround")
			useV1Fallback = true
		}
	}
	
	return &DynamoBotRepository{
		client:        client,
		clientV1:      clientV1,
		tableName:     tableName,
		useV1Fallback: useV1Fallback,
	}
}

// Create creates a new bot in DynamoDB with full-featured support
func (r *DynamoBotRepository) Create(ctx context.Context, bot *models.Bot) error {
	log.Printf("🤖 BotRepository.Create: Starting bot creation for ID: %s", bot.ID)
	
	// Set update time
	now := time.Now()
	bot.UpdateTime = now
	if bot.CreateTime.IsZero() {
		bot.CreateTime = now
	}
	if bot.LastUsedTime.IsZero() {
		bot.LastUsedTime = now
	}
	
	if r.useV1Fallback && r.clientV1 != nil {
		log.Printf("🔧 BotRepository.Create: Using AWS SDK v1 workaround...")
		return r.createWithV1(ctx, bot)
	}
	
	log.Printf("🔧 BotRepository.Create: Using AWS SDK v2 (might fail)...")
	log.Printf("🔧 BotRepository.Create: Marshaling bot data...")
	item, err := attributevalue.MarshalMap(bot)
	if err != nil {
		log.Printf("❌ BotRepository.Create: Failed to marshal bot: %v", err)
		return fmt.Errorf("failed to marshal bot: %w", err)
	}

	log.Printf("📊 BotRepository.Create: Calling DynamoDB PutItem on table: %s", r.tableName)
	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepository.Create: DynamoDB PutItem failed: %v", err)
		log.Printf("🔍 BotRepository.Create: Error type: %T", err)
		return fmt.Errorf("failed to create bot: %w", err)
	}

	log.Printf("✅ BotRepository.Create: Bot created successfully in DynamoDB")
	return nil
}

// createWithV1 creates a bot using AWS SDK v1 as a workaround
func (r *DynamoBotRepository) createWithV1(ctx context.Context, bot *models.Bot) error {
	log.Printf("🔧 BotRepository.createWithV1: Marshaling bot data for SDK v1...")
	log.Printf("🔍 BotRepository.createWithV1: Bot ID before conversion: %s", bot.ID)
	
	// Convert bot to JSON then to SDK v1 attribute map
	botJSON, err := json.Marshal(bot)
	if err != nil {
		log.Printf("❌ BotRepository.createWithV1: Failed to marshal bot to JSON: %v", err)
		return fmt.Errorf("failed to marshal bot to JSON: %w", err)
	}
	
	log.Printf("🔍 BotRepository.createWithV1: Bot JSON length: %d bytes", len(botJSON))
	// Don't log full JSON as it might be large, but check if ID is there
	if strings.Contains(string(botJSON), `"id":`) {
		log.Printf("✅ BotRepository.createWithV1: ID field found in JSON")
	} else {
		log.Printf("❌ BotRepository.createWithV1: ID field NOT found in JSON!")
		log.Printf("🔍 BotRepository.createWithV1: JSON snippet: %.200s", string(botJSON))
	}
	
	var botMap map[string]interface{}
	if err := json.Unmarshal(botJSON, &botMap); err != nil {
		log.Printf("❌ BotRepository.createWithV1: Failed to unmarshal bot JSON: %v", err)
		return fmt.Errorf("failed to unmarshal bot JSON: %w", err)
	}
	
	// Check if ID exists in the map
	if idValue, exists := botMap["id"]; exists {
		log.Printf("✅ BotRepository.createWithV1: ID field found in map: %v", idValue)
	} else {
		log.Printf("❌ BotRepository.createWithV1: ID field NOT found in map!")
		log.Printf("🔍 BotRepository.createWithV1: Available keys: %v", getKeys(botMap))
	}
	
	// Convert to SDK v1 attribute values
	item, err := r.convertMapToV1AttributeValues(botMap)
	if err != nil {
		log.Printf("❌ BotRepository.createWithV1: Failed to convert to v1 attributes: %v", err)
		return fmt.Errorf("failed to convert to v1 attributes: %w", err)
	}
	
	// Fix the primary key name: DynamoDB table expects "ID" but our model uses "id"
	if idAttr, exists := item["id"]; exists {
		item["ID"] = idAttr  // Copy to uppercase key name
		delete(item, "id")   // Remove lowercase key
		log.Printf("🔧 BotRepository.createWithV1: Fixed primary key: id -> ID")
	}
	
	// Check if ID exists in the final item (now uppercase)
	if idAttr, exists := item["ID"]; exists {
		log.Printf("✅ BotRepository.createWithV1: ID attribute found in final item: %v", idAttr)
	} else {
		log.Printf("❌ BotRepository.createWithV1: ID attribute NOT found in final item!")
		log.Printf("🔍 BotRepository.createWithV1: Available attribute keys: %v", getAttributeKeys(item))
	}
	
	log.Printf("📊 BotRepository.createWithV1: Calling SDK v1 PutItem on table: %s", r.tableName)
	_, err = r.clientV1.PutItem(&dynamodbv1.PutItemInput{
		TableName:           awsv1.String(r.tableName),
		Item:                item,
		ConditionExpression: awsv1.String("attribute_not_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepository.createWithV1: SDK v1 PutItem failed: %v", err)
		return fmt.Errorf("failed to create bot with SDK v1: %w", err)
	}
	
	log.Printf("✅ BotRepository.createWithV1: Bot created successfully with SDK v1")
	return nil
}

// convertMapToV1AttributeValues converts a generic map to SDK v1 DynamoDB attribute values
func (r *DynamoBotRepository) convertMapToV1AttributeValues(data map[string]interface{}) (map[string]*dynamodbv1.AttributeValue, error) {
	result := make(map[string]*dynamodbv1.AttributeValue)
	
	for key, value := range data {
		av, err := r.convertValueToV1AttributeValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}
		if av != nil { // Skip nil values
			result[key] = av
		}
	}
	
	return result, nil
}

// convertValueToV1AttributeValue converts a Go value to SDK v1 DynamoDB attribute value
func (r *DynamoBotRepository) convertValueToV1AttributeValue(value interface{}) (*dynamodbv1.AttributeValue, error) {
	if value == nil {
		return nil, nil // Skip nil values
	}
	
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, nil // Skip empty strings
		}
		return &dynamodbv1.AttributeValue{S: awsv1.String(v)}, nil
		
	case bool:
		return &dynamodbv1.AttributeValue{BOOL: awsv1.Bool(v)}, nil
		
	case float64:
		return &dynamodbv1.AttributeValue{N: awsv1.String(strconv.FormatFloat(v, 'f', -1, 64))}, nil
		
	case int:
		return &dynamodbv1.AttributeValue{N: awsv1.String(strconv.Itoa(v))}, nil
		
	case int64:
		return &dynamodbv1.AttributeValue{N: awsv1.String(strconv.FormatInt(v, 10))}, nil
		
	case []interface{}:
		if len(v) == 0 {
			return nil, nil // Skip empty arrays
		}
		
		// Check if it's a string array
		var stringList []*string
		allStrings := true
		for _, item := range v {
			if str, ok := item.(string); ok && str != "" {
				stringList = append(stringList, awsv1.String(str))
			} else {
				allStrings = false
				break
			}
		}
		
		if allStrings && len(stringList) > 0 {
			return &dynamodbv1.AttributeValue{SS: stringList}, nil
		}
		
		// Otherwise, treat as a list of mixed values
		var list []*dynamodbv1.AttributeValue
		for _, item := range v {
			av, err := r.convertValueToV1AttributeValue(item)
			if err != nil {
				return nil, err
			}
			if av != nil {
				list = append(list, av)
			}
		}
		
		if len(list) > 0 {
			return &dynamodbv1.AttributeValue{L: list}, nil
		}
		return nil, nil
		
	case map[string]interface{}:
		if len(v) == 0 {
			return nil, nil // Skip empty maps
		}
		
		subMap, err := r.convertMapToV1AttributeValues(v)
		if err != nil {
			return nil, err
		}
		
		if len(subMap) > 0 {
			return &dynamodbv1.AttributeValue{M: subMap}, nil
		}
		return nil, nil
		
	default:
		// For complex types, try to marshal to JSON and store as string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("unsupported type %T: %w", v, err)
		}
		jsonStr := string(jsonBytes)
		if jsonStr == "null" || jsonStr == "" {
			return nil, nil
		}
		return &dynamodbv1.AttributeValue{S: awsv1.String(jsonStr)}, nil
	}
}

// Helper functions for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getAttributeKeys(m map[string]*dynamodbv1.AttributeValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
		return nil, fmt.Errorf("failed to get bot %s: %w", botID, err)
	}

	if result.Item == nil {
		return nil, nil // Bot not found
	}

	var bot models.Bot
	err = attributevalue.UnmarshalMap(result.Item, &bot)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bot: %w", err)
	}

	return &bot, nil
}

// GetByOwner retrieves all bots owned by a specific user
func (r *DynamoBotRepository) GetByOwner(ctx context.Context, ownerUserID string) ([]models.Bot, error) {
	// Use Query instead of Scan for better performance (requires GSI on ownerUserId)
	// For now, using Scan but in production you'd want a GSI
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("ownerUserId = :ownerUserId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":ownerUserId": &types.AttributeValueMemberS{Value: ownerUserID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bots for owner %s: %w", ownerUserID, err)
	}

	var bots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &bots)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bots: %w", err)
	}

	return bots, nil
}

// GetPublicBots retrieves all public bots (sharedScope = "public")
func (r *DynamoBotRepository) GetPublicBots(ctx context.Context) ([]models.Bot, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("sharedScope = :sharedScope"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sharedScope": &types.AttributeValueMemberS{Value: models.SharedScopePublic},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get public bots: %w", err)
	}

	var bots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &bots)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public bots: %w", err)
	}

	return bots, nil
}

// GetSharedBots retrieves bots shared with specific user/groups (sharedScope = "partial")
func (r *DynamoBotRepository) GetSharedBots(ctx context.Context, userID string, userGroups []string) ([]models.Bot, error) {
	// Get all bots with partial sharing
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("sharedScope = :sharedScope"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sharedScope": &types.AttributeValueMemberS{Value: models.SharedScopePartial},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get shared bots: %w", err)
	}

	var allSharedBots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &allSharedBots)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal shared bots: %w", err)
	}

	// Filter bots that the user has access to
	var accessibleBots []models.Bot
	for _, bot := range allSharedBots {
		if r.userHasAccessToPartialBot(bot, userID, userGroups) {
			accessibleBots = append(accessibleBots, bot)
		}
	}

	return accessibleBots, nil
}

// GetPinnedBots retrieves pinned public bots (sharedStatus starts with "pinned@")
func (r *DynamoBotRepository) GetPinnedBots(ctx context.Context) ([]models.Bot, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("sharedScope = :sharedScope AND begins_with(sharedStatus, :pinnedPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sharedScope":   &types.AttributeValueMemberS{Value: models.SharedScopePublic},
			":pinnedPrefix":  &types.AttributeValueMemberS{Value: "pinned@"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pinned bots: %w", err)
	}

	var bots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &bots)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal pinned bots: %w", err)
	}

	return bots, nil
}

// Update updates an existing bot with full-featured field support
func (r *DynamoBotRepository) Update(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error {
	updateExpression := "SET updateTime = :updateTime"
	expressionAttributeValues := map[string]types.AttributeValue{
		":updateTime": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}
	expressionAttributeNames := make(map[string]string)
	updates := []string{"updateTime = :updateTime"}

	// Core fields
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

	// Knowledge Base fields
	if updateReq.KnowledgeBaseID != nil {
		updates = append(updates, "knowledgeBaseId = :knowledgeBaseId")
		expressionAttributeValues[":knowledgeBaseId"] = &types.AttributeValueMemberS{Value: *updateReq.KnowledgeBaseID}
	}

	// CloudFormation stack fields
	if updateReq.CloudFormationStackName != nil {
		updates = append(updates, "cloudFormationStackName = :cloudFormationStackName")
		expressionAttributeValues[":cloudFormationStackName"] = &types.AttributeValueMemberS{Value: *updateReq.CloudFormationStackName}
	}

	if updateReq.StackStatus != nil {
		updates = append(updates, "stackStatus = :stackStatus")
		expressionAttributeValues[":stackStatus"] = &types.AttributeValueMemberS{Value: *updateReq.StackStatus}
	}

	if updateReq.DocumentBucketName != nil {
		updates = append(updates, "documentBucketName = :documentBucketName")
		expressionAttributeValues[":documentBucketName"] = &types.AttributeValueMemberS{Value: *updateReq.DocumentBucketName}
	}

	if updateReq.GuardrailArn != nil {
		updates = append(updates, "guardrailArn = :guardrailArn")
		expressionAttributeValues[":guardrailArn"] = &types.AttributeValueMemberS{Value: *updateReq.GuardrailArn}
	}

	if updateReq.GuardrailVersion != nil {
		updates = append(updates, "guardrailVersion = :guardrailVersion")
		expressionAttributeValues[":guardrailVersion"] = &types.AttributeValueMemberS{Value: *updateReq.GuardrailVersion}
	}

	// Sharing fields
	if updateReq.SharedScope != nil {
		updates = append(updates, "sharedScope = :sharedScope")
		expressionAttributeValues[":sharedScope"] = &types.AttributeValueMemberS{Value: *updateReq.SharedScope}
	}

	if updateReq.SharedStatus != nil {
		updates = append(updates, "sharedStatus = :sharedStatus")
		expressionAttributeValues[":sharedStatus"] = &types.AttributeValueMemberS{Value: *updateReq.SharedStatus}
	}

	if updateReq.AllowedUsers != nil {
		allowedUsers, err := attributevalue.Marshal(updateReq.AllowedUsers)
		if err != nil {
			return fmt.Errorf("failed to marshal allowed users: %w", err)
		}
		updates = append(updates, "allowedUsers = :allowedUsers")
		expressionAttributeValues[":allowedUsers"] = allowedUsers
	}

	if updateReq.AllowedGroups != nil {
		allowedGroups, err := attributevalue.Marshal(updateReq.AllowedGroups)
		if err != nil {
			return fmt.Errorf("failed to marshal allowed groups: %w", err)
		}
		updates = append(updates, "allowedGroups = :allowedGroups")
		expressionAttributeValues[":allowedGroups"] = allowedGroups
	}

	// User preferences
	if updateReq.IsStarred != nil {
		updates = append(updates, "isStarred = :isStarred")
		expressionAttributeValues[":isStarred"] = &types.AttributeValueMemberBOOL{Value: *updateReq.IsStarred}
	}

	// Usage analytics
	if updateReq.UsageCount != nil {
		updates = append(updates, "usageCount = :usageCount")
		expressionAttributeValues[":usageCount"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", *updateReq.UsageCount)}
	}

	// Sync status fields
	if updateReq.SyncStatus != nil {
		updates = append(updates, "syncStatus = :syncStatus")
		expressionAttributeValues[":syncStatus"] = &types.AttributeValueMemberS{Value: *updateReq.SyncStatus}
	}

	if updateReq.SyncStatusReason != nil {
		updates = append(updates, "syncStatusReason = :syncStatusReason")
		expressionAttributeValues[":syncStatusReason"] = &types.AttributeValueMemberS{Value: *updateReq.SyncStatusReason}
	}

	if updateReq.SyncLastExecID != nil {
		updates = append(updates, "syncLastExecId = :syncLastExecId")
		expressionAttributeValues[":syncLastExecId"] = &types.AttributeValueMemberS{Value: *updateReq.SyncLastExecID}
	}

	// Configuration fields
	if updateReq.DisplayRetrievedChunks != nil {
		updates = append(updates, "displayRetrievedChunks = :displayRetrievedChunks")
		expressionAttributeValues[":displayRetrievedChunks"] = &types.AttributeValueMemberBOOL{Value: *updateReq.DisplayRetrievedChunks}
	}

	if updateReq.GenerationParams != nil {
		genParams, err := attributevalue.Marshal(updateReq.GenerationParams)
		if err != nil {
			return fmt.Errorf("failed to marshal generation params: %w", err)
		}
		updates = append(updates, "generationParams = :generationParams")
		expressionAttributeValues[":generationParams"] = genParams
	}

	if updateReq.KnowledgeBaseConfig != nil {
		kbConfig, err := attributevalue.Marshal(updateReq.KnowledgeBaseConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal knowledge base config: %w", err)
		}
		updates = append(updates, "knowledgeBaseConfig = :knowledgeBaseConfig")
		expressionAttributeValues[":knowledgeBaseConfig"] = kbConfig
	}

	if updateReq.ConversationStarters != nil {
		starters, err := attributevalue.Marshal(updateReq.ConversationStarters)
		if err != nil {
			return fmt.Errorf("failed to marshal conversation starters: %w", err)
		}
		updates = append(updates, "conversationStarters = :conversationStarters")
		expressionAttributeValues[":conversationStarters"] = starters
	}

	if updateReq.ActiveModels != nil {
		activeModels, err := attributevalue.Marshal(updateReq.ActiveModels)
		if err != nil {
			return fmt.Errorf("failed to marshal active models: %w", err)
		}
		updates = append(updates, "activeModels = :activeModels")
		expressionAttributeValues[":activeModels"] = activeModels
	}

	if updateReq.AgentTools != nil {
		tools, err := attributevalue.Marshal(updateReq.AgentTools)
		if err != nil {
			return fmt.Errorf("failed to marshal agent tools: %w", err)
		}
		updates = append(updates, "agentTools = :agentTools")
		expressionAttributeValues[":agentTools"] = tools
	}

	updateExpression = "SET " + strings.Join(updates, ", ")

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
	if err != nil {
		return fmt.Errorf("failed to update bot %s: %w", botID, err)
	}

	return nil
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

	if err != nil {
		return fmt.Errorf("failed to delete bot %s: %w", botID, err)
	}

	return nil
}

// UpdateLastUsedTime updates the last used timestamp for a bot
func (r *DynamoBotRepository) UpdateLastUsedTime(ctx context.Context, botID string, lastUsedTime time.Time) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		UpdateExpression: aws.String("SET lastUsedTime = :lastUsedTime, updateTime = :updateTime"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":lastUsedTime": &types.AttributeValueMemberS{Value: lastUsedTime.Format(time.RFC3339)},
			":updateTime":   &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return fmt.Errorf("failed to update last used time for bot %s: %w", botID, err)
	}

	return nil
}

// UpdateStarred updates the starred status of a bot
func (r *DynamoBotRepository) UpdateStarred(ctx context.Context, botID string, starred bool) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		UpdateExpression: aws.String("SET isStarred = :isStarred, updateTime = :updateTime"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":isStarred":  &types.AttributeValueMemberBOOL{Value: starred},
			":updateTime": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})

	if err != nil {
		return fmt.Errorf("failed to update starred status for bot %s: %w", botID, err)
	}

	return nil
}

// SearchBots searches for bots by title, description, or instruction
func (r *DynamoBotRepository) SearchBots(ctx context.Context, query string, scope string, userID string, userGroups []string) ([]models.Bot, error) {
	// Simple implementation using Scan with filter
	// In production, you'd want to use ElasticSearch or CloudSearch for better text search
	
	var filterExpression string
	expressionAttributeValues := make(map[string]types.AttributeValue)
	
	// Build search filter
	searchFilter := "contains(#title, :query) OR contains(#description, :query) OR contains(instruction, :query)"
	expressionAttributeValues[":query"] = &types.AttributeValueMemberS{Value: strings.ToLower(query)}
	
	// Add scope filter
	switch scope {
	case "public":
		filterExpression = fmt.Sprintf("(%s) AND sharedScope = :sharedScope", searchFilter)
		expressionAttributeValues[":sharedScope"] = &types.AttributeValueMemberS{Value: models.SharedScopePublic}
	case "private":
		filterExpression = fmt.Sprintf("(%s) AND ownerUserId = :ownerUserId", searchFilter)
		expressionAttributeValues[":ownerUserId"] = &types.AttributeValueMemberS{Value: userID}
	default:
		filterExpression = searchFilter
	}

	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String(filterExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ExpressionAttributeNames: map[string]string{
			"#title":       "title",
			"#description": "description",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search bots: %w", err)
	}

	var bots []models.Bot
	err = attributevalue.UnmarshalListOfMaps(result.Items, &bots)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal search results: %w", err)
	}

	// Apply access control filtering
	var accessibleBots []models.Bot
	for _, bot := range bots {
		if bot.IsAccessibleByUser(userID, userGroups, false) {
			accessibleBots = append(accessibleBots, bot)
		}
	}

	return accessibleBots, nil
}

// GetBotsByScope retrieves bots by scope with limit
func (r *DynamoBotRepository) GetBotsByScope(ctx context.Context, scope string, userID string, userGroups []string, limit int) ([]models.Bot, error) {
	var bots []models.Bot
	var err error
	
	switch scope {
	case "private":
		bots, err = r.GetByOwner(ctx, userID)
	case "public":
		bots, err = r.GetPublicBots(ctx)
	case "shared":
		bots, err = r.GetSharedBots(ctx, userID, userGroups)
	case "pinned":
		bots, err = r.GetPinnedBots(ctx)
	default:
		return nil, fmt.Errorf("invalid scope: %s", scope)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Apply limit
	if limit > 0 && len(bots) > limit {
		bots = bots[:limit]
	}
	
	return bots, nil
}

// Helper method to check if user has access to a partially shared bot
func (r *DynamoBotRepository) userHasAccessToPartialBot(bot models.Bot, userID string, userGroups []string) bool {
	// Check if user is in allowed users list
	for _, allowedUser := range bot.AllowedUsers {
		if allowedUser == userID {
			return true
		}
	}
	
	// Check if user belongs to any allowed groups
	for _, allowedGroup := range bot.AllowedGroups {
		for _, userGroup := range userGroups {
			if allowedGroup == userGroup {
				return true
			}
		}
	}
	
	return false
}

// === Bot Aliases Implementation ===

// CreateAlias creates a new bot alias for a user
func (r *DynamoBotRepository) CreateAlias(ctx context.Context, alias *models.BotAlias) error {
	// Set timestamps
	now := time.Now()
	alias.CreateTime = now
	alias.LastUsedTime = now
	
	item, err := attributevalue.MarshalMap(alias)
	if err != nil {
		return fmt.Errorf("failed to marshal alias: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	
	if err != nil {
		return fmt.Errorf("failed to create alias: %w", err)
	}

	return nil
}

// GetAliasByID retrieves a bot alias by its ID
func (r *DynamoBotRepository) GetAliasByID(ctx context.Context, aliasID string) (*models.BotAlias, error) {
	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: aliasID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get alias %s: %w", aliasID, err)
	}

	if result.Item == nil {
		return nil, nil // Alias not found
	}

	var alias models.BotAlias
	err = attributevalue.UnmarshalMap(result.Item, &alias)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal alias: %w", err)
	}

	return &alias, nil
}

// GetAliasesByUser retrieves all aliases owned by a specific user
func (r *DynamoBotRepository) GetAliasesByUser(ctx context.Context, userID string) ([]models.BotAlias, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("userId = :userId AND attribute_exists(originalBotId)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get aliases for user %s: %w", userID, err)
	}

	var aliases []models.BotAlias
	err = attributevalue.UnmarshalListOfMaps(result.Items, &aliases)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal aliases: %w", err)
	}

	return aliases, nil
}

// GetAliasForUserBot retrieves a user's alias for a specific original bot (if exists)
func (r *DynamoBotRepository) GetAliasForUserBot(ctx context.Context, userID, originalBotID string) (*models.BotAlias, error) {
	result, err := r.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("userId = :userId AND originalBotId = :originalBotId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userId":        &types.AttributeValueMemberS{Value: userID},
			":originalBotId": &types.AttributeValueMemberS{Value: originalBotID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get alias for user %s and bot %s: %w", userID, originalBotID, err)
	}

	if len(result.Items) == 0 {
		return nil, nil // No alias found
	}

	var alias models.BotAlias
	err = attributevalue.UnmarshalMap(result.Items[0], &alias)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal alias: %w", err)
	}

	return &alias, nil
}

// UpdateAlias updates an alias with the provided fields
func (r *DynamoBotRepository) UpdateAlias(ctx context.Context, aliasID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	updateExpression := "SET updateTime = :updateTime"
	expressionAttributeValues := map[string]types.AttributeValue{
		":updateTime": &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
	}
	updateList := []string{"updateTime = :updateTime"}

	for field, value := range updates {
		switch field {
		case "isStarred":
			if starred, ok := value.(bool); ok {
				updateList = append(updateList, "isStarred = :isStarred")
				expressionAttributeValues[":isStarred"] = &types.AttributeValueMemberBOOL{Value: starred}
			}
		case "lastUsedTime":
			if timestamp, ok := value.(time.Time); ok {
				updateList = append(updateList, "lastUsedTime = :lastUsedTime")
				expressionAttributeValues[":lastUsedTime"] = &types.AttributeValueMemberS{Value: timestamp.Format(time.RFC3339)}
			}
		case "isOriginAccessible":
			if accessible, ok := value.(bool); ok {
				updateList = append(updateList, "isOriginAccessible = :isOriginAccessible")
				expressionAttributeValues[":isOriginAccessible"] = &types.AttributeValueMemberBOOL{Value: accessible}
			}
		}
	}

	updateExpression += ", " + strings.Join(updateList[1:], ", ")

	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: aliasID},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	})

	if err != nil {
		return fmt.Errorf("failed to update alias %s: %w", aliasID, err)
	}

	return nil
}

// DeleteAlias deletes a bot alias
func (r *DynamoBotRepository) DeleteAlias(ctx context.Context, aliasID string) error {
	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: aliasID},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete alias %s: %w", aliasID, err)
	}

	return nil
}

// UpdateAliasLastUsedTime updates the last used time for an alias
func (r *DynamoBotRepository) UpdateAliasLastUsedTime(ctx context.Context, aliasID string, lastUsedTime time.Time) error {
	return r.UpdateAlias(ctx, aliasID, map[string]interface{}{
		"lastUsedTime": lastUsedTime,
	})
}

// UpdateAliasStarred updates the starred status for an alias
func (r *DynamoBotRepository) UpdateAliasStarred(ctx context.Context, aliasID string, starred bool) error {
	return r.UpdateAlias(ctx, aliasID, map[string]interface{}{
		"isStarred": starred,
	})
}

// IncrementUsageCount atomically increments the usage count for a bot
func (r *DynamoBotRepository) IncrementUsageCount(ctx context.Context, botID string) error {
	_, err := r.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: botID},
		},
		UpdateExpression: aws.String("ADD usageCount :increment SET updateTime = :updateTime"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":increment":   &types.AttributeValueMemberN{Value: "1"},
			":updateTime":  &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to increment usage count for bot %s: %w", botID, err)
	}

	return nil
}