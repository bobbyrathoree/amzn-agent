package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	// AWS SDK v1 only
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodbv1 "github.com/aws/aws-sdk-go/service/dynamodb"
	awsv1 "github.com/aws/aws-sdk-go/aws"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// BotRepositoryV1 handles data persistence for bots using AWS SDK v1 only
type BotRepositoryV1 struct {
	client    *dynamodbv1.DynamoDB
	tableName string
}

// NewBotRepositoryV1 creates a new BotRepository using AWS SDK v1 only
func NewBotRepositoryV1(tableName string) (BotRepository, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}
	
	client := dynamodbv1.New(sess)
	
	// Test connectivity
	_, err = client.ListTables(&dynamodbv1.ListTablesInput{})
	if err != nil {
		log.Printf("⚠️ AWS SDK v1 test failed: %v", err)
		return nil, fmt.Errorf("DynamoDB connectivity test failed: %w", err)
	}
	
	log.Printf("✅ AWS SDK v1 DynamoDB repository initialized successfully")
	
	return &BotRepositoryV1{
		client:    client,
		tableName: tableName,
	}, nil
}

// Create creates a new bot in DynamoDB with proper schema mapping
func (r *BotRepositoryV1) Create(ctx context.Context, bot *models.Bot) error {
	log.Printf("🤖 BotRepositoryV1.Create: Starting bot creation for ID: %s", bot.ID)
	
	// Set timestamps
	now := time.Now()
	bot.UpdateTime = now
	if bot.CreateTime.IsZero() {
		bot.CreateTime = now
	}
	if bot.LastUsedTime.IsZero() {
		bot.LastUsedTime = now
	}
	
	// Convert bot to DynamoDB item with proper field mapping
	item, err := r.botToDynamoItem(bot)
	if err != nil {
		log.Printf("❌ BotRepositoryV1.Create: Failed to convert bot: %v", err)
		return fmt.Errorf("failed to convert bot: %w", err)
	}
	
	log.Printf("📊 BotRepositoryV1.Create: Calling PutItem on table: %s", r.tableName)
	_, err = r.client.PutItem(&dynamodbv1.PutItemInput{
		TableName:           awsv1.String(r.tableName),
		Item:                item,
		ConditionExpression: awsv1.String("attribute_not_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.Create: PutItem failed: %v", err)
		return fmt.Errorf("failed to create bot: %w", err)
	}
	
	log.Printf("✅ BotRepositoryV1.Create: Bot created successfully")
	return nil
}

// GetByID retrieves a bot by its ID
func (r *BotRepositoryV1) GetByID(ctx context.Context, botID string) (*models.Bot, error) {
	log.Printf("🔍 BotRepositoryV1.GetByID: Getting bot: %s", botID)
	
	result, err := r.client.GetItem(&dynamodbv1.GetItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(botID)}, // Schema uses uppercase ID
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetByID: GetItem failed: %v", err)
		return nil, fmt.Errorf("failed to get bot %s: %w", botID, err)
	}
	
	if result.Item == nil {
		log.Printf("ℹ️ BotRepositoryV1.GetByID: Bot not found: %s", botID)
		return nil, nil
	}
	
	bot, err := r.dynamoItemToBot(result.Item)
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetByID: Failed to convert item: %v", err)
		return nil, fmt.Errorf("failed to convert bot: %w", err)
	}
	
	log.Printf("✅ BotRepositoryV1.GetByID: Bot retrieved successfully")
	return bot, nil
}

// GetByOwner retrieves all bots owned by a specific user using GSI
func (r *BotRepositoryV1) GetByOwner(ctx context.Context, ownerUserID string) ([]models.Bot, error) {
	log.Printf("🔍 BotRepositoryV1.GetByOwner: Getting bots for owner: %s", ownerUserID)
	
	// Use GSI OwnerIDIndex to efficiently query by owner
	result, err := r.client.Query(&dynamodbv1.QueryInput{
		TableName: awsv1.String(r.tableName),
		IndexName: awsv1.String("OwnerIDIndex"), // GSI name from schema
		KeyConditionExpression: awsv1.String("OwnerID = :ownerID"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":ownerID": {S: awsv1.String(ownerUserID)},
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetByOwner: Query failed: %v", err)
		return nil, fmt.Errorf("failed to get bots for owner %s: %w", ownerUserID, err)
	}
	
	var bots []models.Bot
	for _, item := range result.Items {
		bot, err := r.dynamoItemToBot(item)
		if err != nil {
			log.Printf("⚠️ BotRepositoryV1.GetByOwner: Failed to convert item, skipping: %v", err)
			continue
		}
		bots = append(bots, *bot)
	}
	
	log.Printf("✅ BotRepositoryV1.GetByOwner: Retrieved %d bots", len(bots))
	return bots, nil
}

// botToDynamoItem converts a Bot struct to DynamoDB item with proper field mapping
func (r *BotRepositoryV1) botToDynamoItem(bot *models.Bot) (map[string]*dynamodbv1.AttributeValue, error) {
	// Convert to JSON first for easy handling of complex types
	botJSON, err := json.Marshal(bot)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bot: %w", err)
	}
	
	var botData map[string]interface{}
	if err := json.Unmarshal(botJSON, &botData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bot: %w", err)
	}
	
	// Convert to DynamoDB attributes
	item, err := r.convertMapToV1AttributeValues(botData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to attributes: %w", err)
	}
	
	// Apply field name mappings to match DynamoDB schema
	r.applyBotSchemaMapping(item)
	
	return item, nil
}

// applyBotSchemaMapping fixes field names to match DynamoDB schema
func (r *BotRepositoryV1) applyBotSchemaMapping(item map[string]*dynamodbv1.AttributeValue) {
	// Fix primary key: id -> ID
	if val, exists := item["id"]; exists {
		item["ID"] = val
		delete(item, "id")
	}
	
	// Fix GSI key: ownerUserId -> OwnerID  
	if val, exists := item["ownerUserId"]; exists {
		item["OwnerID"] = val
		delete(item, "ownerUserId")
	}
	
	log.Printf("🔧 BotRepositoryV1.applyBotSchemaMapping: Applied schema field mappings")
}

// dynamoItemToBot converts DynamoDB item back to Bot struct
func (r *BotRepositoryV1) dynamoItemToBot(item map[string]*dynamodbv1.AttributeValue) (*models.Bot, error) {
	// Reverse the schema mapping first
	r.reverseBotSchemaMapping(item)
	
	// Convert DynamoDB item to generic map
	data, err := r.convertV1AttributeValuesToMap(item)
	if err != nil {
		return nil, fmt.Errorf("failed to convert attributes: %w", err)
	}
	
	// Convert to JSON and then to Bot struct
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	
	var bot models.Bot
	if err := json.Unmarshal(jsonData, &bot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bot: %w", err)
	}
	
	return &bot, nil
}

// reverseBotSchemaMapping reverses field name mappings from DynamoDB schema to Go struct
func (r *BotRepositoryV1) reverseBotSchemaMapping(item map[string]*dynamodbv1.AttributeValue) {
	// Reverse: ID -> id
	if val, exists := item["ID"]; exists {
		item["id"] = val
		delete(item, "ID")
	}
	
	// Reverse: OwnerID -> ownerUserId
	if val, exists := item["OwnerID"]; exists {
		item["ownerUserId"] = val  
		delete(item, "OwnerID")
	}
}

// Helper methods for attribute conversion
func (r *BotRepositoryV1) convertMapToV1AttributeValues(data map[string]interface{}) (map[string]*dynamodbv1.AttributeValue, error) {
	result := make(map[string]*dynamodbv1.AttributeValue)
	
	for key, value := range data {
		av, err := r.convertValueToV1AttributeValue(value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}
		if av != nil {
			result[key] = av
		}
	}
	
	return result, nil
}

func (r *BotRepositoryV1) convertValueToV1AttributeValue(value interface{}) (*dynamodbv1.AttributeValue, error) {
	if value == nil {
		return nil, nil
	}
	
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, nil
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
			return nil, nil
		}
		
		// Handle string slices
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
		
		// Handle mixed lists
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
			return nil, nil
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
		// For complex types, serialize to JSON string
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

func (r *BotRepositoryV1) convertV1AttributeValuesToMap(item map[string]*dynamodbv1.AttributeValue) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for key, av := range item {
		value, err := r.convertV1AttributeValueToValue(av)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %w", key, err)
		}
		if value != nil {
			result[key] = value
		}
	}
	
	return result, nil
}

func (r *BotRepositoryV1) convertV1AttributeValueToValue(av *dynamodbv1.AttributeValue) (interface{}, error) {
	if av == nil {
		return nil, nil
	}
	
	if av.S != nil {
		return *av.S, nil
	}
	if av.N != nil {
		// Try int first, then float
		if intVal, err := strconv.Atoi(*av.N); err == nil {
			return intVal, nil
		}
		if floatVal, err := strconv.ParseFloat(*av.N, 64); err == nil {
			return floatVal, nil
		}
		return *av.N, nil // Return as string if parsing fails
	}
	if av.BOOL != nil {
		return *av.BOOL, nil
	}
	if av.SS != nil {
		result := make([]string, len(av.SS))
		for i, s := range av.SS {
			result[i] = *s
		}
		return result, nil
	}
	if av.L != nil {
		result := make([]interface{}, len(av.L))
		for i, item := range av.L {
			val, err := r.convertV1AttributeValueToValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = val
		}
		return result, nil
	}
	if av.M != nil {
		return r.convertV1AttributeValuesToMap(av.M)
	}
	
	return nil, nil
}

// Helper methods for BotAlias conversion
func (r *BotRepositoryV1) aliasToDynamoItem(alias *models.BotAlias) (map[string]*dynamodbv1.AttributeValue, error) {
	// Convert to JSON first for easy handling
	aliasJSON, err := json.Marshal(alias)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alias: %w", err)
	}
	
	var aliasData map[string]interface{}
	if err := json.Unmarshal(aliasJSON, &aliasData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alias: %w", err)
	}
	
	// Convert to DynamoDB attributes
	item, err := r.convertMapToV1AttributeValues(aliasData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to attributes: %w", err)
	}
	
	// Apply field name mappings for aliases (similar to bots but different fields)
	r.applyAliasSchemaMapping(item)
	
	return item, nil
}

func (r *BotRepositoryV1) applyAliasSchemaMapping(item map[string]*dynamodbv1.AttributeValue) {
	// Fix primary key: id -> ID
	if val, exists := item["id"]; exists {
		item["ID"] = val
		delete(item, "id")
	}
	
	// Fix user field: userId -> UserID (different from bot's ownerUserId -> OwnerID)
	if val, exists := item["userId"]; exists {
		item["UserID"] = val
		delete(item, "userId")
	}
	
	// Fix owner field: ownerUserId -> OwnerUserID (keep the original field name for aliases)
	if val, exists := item["ownerUserId"]; exists {
		item["OwnerUserID"] = val
		delete(item, "ownerUserId")
	}
	
	log.Printf("🔧 BotRepositoryV1.applyAliasSchemaMapping: Applied alias schema field mappings")
}

func (r *BotRepositoryV1) dynamoItemToAlias(item map[string]*dynamodbv1.AttributeValue) (*models.BotAlias, error) {
	// Reverse the schema mapping first
	r.reverseAliasSchemaMapping(item)
	
	// Convert DynamoDB item to generic map
	data, err := r.convertV1AttributeValuesToMap(item)
	if err != nil {
		return nil, fmt.Errorf("failed to convert attributes: %w", err)
	}
	
	// Convert to JSON and then to BotAlias struct
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	
	var alias models.BotAlias
	if err := json.Unmarshal(jsonData, &alias); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alias: %w", err)
	}
	
	return &alias, nil
}

func (r *BotRepositoryV1) reverseAliasSchemaMapping(item map[string]*dynamodbv1.AttributeValue) {
	// Reverse: ID -> id
	if val, exists := item["ID"]; exists {
		item["id"] = val
		delete(item, "ID")
	}
	
	// Reverse: UserID -> userId
	if val, exists := item["UserID"]; exists {
		item["userId"] = val
		delete(item, "UserID")
	}
	
	// Reverse: OwnerUserID -> ownerUserId
	if val, exists := item["OwnerUserID"]; exists {
		item["ownerUserId"] = val
		delete(item, "OwnerUserID")
	}
}

// GetPublicBots retrieves all publicly shared bots
func (r *BotRepositoryV1) GetPublicBots(ctx context.Context) ([]models.Bot, error) {
	log.Printf("🌍 BotRepositoryV1.GetPublicBots: Getting public bots")
	
	// Scan for bots with SharedScope = "public" and SharedStatus = "shared" or "pinned@*"
	result, err := r.client.Scan(&dynamodbv1.ScanInput{
		TableName: awsv1.String(r.tableName),
		FilterExpression: awsv1.String("SharedScope = :scope AND (SharedStatus = :shared OR begins_with(SharedStatus, :pinned))"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":scope": {S: awsv1.String("public")},
			":shared": {S: awsv1.String("shared")},
			":pinned": {S: awsv1.String("pinned@")},
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetPublicBots: Scan failed: %v", err)
		return nil, fmt.Errorf("failed to get public bots: %w", err)
	}
	
	var bots []models.Bot
	for _, item := range result.Items {
		bot, err := r.dynamoItemToBot(item)
		if err != nil {
			log.Printf("⚠️ BotRepositoryV1.GetPublicBots: Failed to convert item, skipping: %v", err)
			continue
		}
		bots = append(bots, *bot)
	}
	
	log.Printf("✅ BotRepositoryV1.GetPublicBots: Retrieved %d public bots", len(bots))
	return bots, nil
}

func (r *BotRepositoryV1) GetSharedBots(ctx context.Context, userID string, userGroups []string) ([]models.Bot, error) {
	log.Printf("🤝 BotRepositoryV1.GetSharedBots: Getting shared bots for user: %s", userID)
	
	// Scan for bots with SharedScope = "partial" where user is in allowed lists
	result, err := r.client.Scan(&dynamodbv1.ScanInput{
		TableName: awsv1.String(r.tableName),
		FilterExpression: awsv1.String("SharedScope = :scope AND (contains(AllowedUsers, :userID) OR #owner <> :userID)"),
		ExpressionAttributeNames: map[string]*string{
			"#owner": awsv1.String("OwnerID"),
		},
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":scope": {S: awsv1.String("partial")},
			":userID": {S: awsv1.String(userID)},
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetSharedBots: Scan failed: %v", err)
		return nil, fmt.Errorf("failed to get shared bots: %w", err)
	}
	
	var bots []models.Bot
	for _, item := range result.Items {
		bot, err := r.dynamoItemToBot(item)
		if err != nil {
			log.Printf("⚠️ BotRepositoryV1.GetSharedBots: Failed to convert item, skipping: %v", err)
			continue
		}
		
		// Additional access check in application logic
		if bot.IsAccessibleByUser(userID, userGroups, false) {
			bots = append(bots, *bot)
		}
	}
	
	log.Printf("✅ BotRepositoryV1.GetSharedBots: Retrieved %d shared bots", len(bots))
	return bots, nil
}

func (r *BotRepositoryV1) GetPinnedBots(ctx context.Context) ([]models.Bot, error) {
	log.Printf("📌 BotRepositoryV1.GetPinnedBots: Getting pinned bots")
	
	// Scan for bots with SharedScope = "public" and SharedStatus starts with "pinned@"
	result, err := r.client.Scan(&dynamodbv1.ScanInput{
		TableName: awsv1.String(r.tableName),
		FilterExpression: awsv1.String("SharedScope = :scope AND begins_with(SharedStatus, :pinned)"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":scope": {S: awsv1.String("public")},
			":pinned": {S: awsv1.String("pinned@")},
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetPinnedBots: Scan failed: %v", err)
		return nil, fmt.Errorf("failed to get pinned bots: %w", err)
	}
	
	var bots []models.Bot
	for _, item := range result.Items {
		bot, err := r.dynamoItemToBot(item)
		if err != nil {
			log.Printf("⚠️ BotRepositoryV1.GetPinnedBots: Failed to convert item, skipping: %v", err)
			continue
		}
		bots = append(bots, *bot)
	}
	
	log.Printf("✅ BotRepositoryV1.GetPinnedBots: Retrieved %d pinned bots", len(bots))
	return bots, nil
}

func (r *BotRepositoryV1) Update(ctx context.Context, botID string, updateReq *models.UpdateBotRequest) error {
	log.Printf("🔄 BotRepositoryV1.Update: Updating bot: %s", botID)
	
	// Build update expression dynamically based on provided fields
	updateExpression := "SET #updateTime = :updateTime"
	expressionAttributeNames := map[string]*string{
		"#updateTime": awsv1.String("UpdateTime"),
	}
	expressionAttributeValues := map[string]*dynamodbv1.AttributeValue{
		":updateTime": {S: awsv1.String(time.Now().Format(time.RFC3339))},
	}
	
	// Add fields to update expression if they are provided
	// Note: Field names must match JSON keys (lowercase) used during bot creation
	if updateReq.Title != nil {
		updateExpression += ", title = :title"
		expressionAttributeValues[":title"] = &dynamodbv1.AttributeValue{S: updateReq.Title}
	}
	if updateReq.Description != nil {
		updateExpression += ", description = :description"
		expressionAttributeValues[":description"] = &dynamodbv1.AttributeValue{S: updateReq.Description}
	}
	if updateReq.Instruction != nil {
		updateExpression += ", instruction = :instruction"
		expressionAttributeValues[":instruction"] = &dynamodbv1.AttributeValue{S: updateReq.Instruction}
	}
	if updateReq.IsStarred != nil {
		updateExpression += ", isStarred = :starred"
		expressionAttributeValues[":starred"] = &dynamodbv1.AttributeValue{BOOL: updateReq.IsStarred}
	}
	if updateReq.SharedScope != nil {
		updateExpression += ", sharedScope = :scope"
		expressionAttributeValues[":scope"] = &dynamodbv1.AttributeValue{S: updateReq.SharedScope}
	}
	if updateReq.SharedStatus != nil {
		updateExpression += ", sharedStatus = :status"
		expressionAttributeValues[":status"] = &dynamodbv1.AttributeValue{S: updateReq.SharedStatus}
	}
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(botID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String(updateExpression),
		ExpressionAttributeNames: expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression: awsv1.String("attribute_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.Update: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to update bot %s: %w", botID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.Update: Bot updated successfully")
	return nil
}

func (r *BotRepositoryV1) Delete(ctx context.Context, botID string) error {
	log.Printf("🗑️ BotRepositoryV1.Delete: Deleting bot: %s", botID)
	
	_, err := r.client.DeleteItem(&dynamodbv1.DeleteItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(botID)}, // Schema uses uppercase ID
		},
		ConditionExpression: awsv1.String("attribute_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.Delete: DeleteItem failed: %v", err)
		return fmt.Errorf("failed to delete bot %s: %w", botID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.Delete: Bot deleted successfully")
	return nil
}

func (r *BotRepositoryV1) UpdateLastUsedTime(ctx context.Context, botID string, lastUsedTime time.Time) error {
	log.Printf("⏰ BotRepositoryV1.UpdateLastUsedTime: Updating last used time for bot: %s", botID)
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(botID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String("SET LastUsedTime = :lastUsed, #updateTime = :updateTime"),
		ExpressionAttributeNames: map[string]*string{
			"#updateTime": awsv1.String("UpdateTime"),
		},
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":lastUsed": {S: awsv1.String(lastUsedTime.Format(time.RFC3339))},
			":updateTime": {S: awsv1.String(time.Now().Format(time.RFC3339))},
		},
		ConditionExpression: awsv1.String("attribute_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.UpdateLastUsedTime: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to update last used time for bot %s: %w", botID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.UpdateLastUsedTime: Last used time updated successfully")
	return nil
}

func (r *BotRepositoryV1) UpdateStarred(ctx context.Context, botID string, starred bool) error {
	log.Printf("⭐ BotRepositoryV1.UpdateStarred: Updating starred status for bot: %s to %t", botID, starred)
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(botID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String("SET IsStarred = :starred, #updateTime = :updateTime"),
		ExpressionAttributeNames: map[string]*string{
			"#updateTime": awsv1.String("UpdateTime"),
		},
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":starred": {BOOL: awsv1.Bool(starred)},
			":updateTime": {S: awsv1.String(time.Now().Format(time.RFC3339))},
		},
		ConditionExpression: awsv1.String("attribute_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.UpdateStarred: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to update starred status for bot %s: %w", botID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.UpdateStarred: Starred status updated successfully")
	return nil
}

func (r *BotRepositoryV1) IncrementUsageCount(ctx context.Context, botID string) error {
	log.Printf("📊 BotRepositoryV1.IncrementUsageCount: Incrementing usage count for bot: %s", botID)
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(botID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String("ADD UsageCount :increment SET #updateTime = :updateTime"),
		ExpressionAttributeNames: map[string]*string{
			"#updateTime": awsv1.String("UpdateTime"),
		},
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":increment": {N: awsv1.String("1")},
			":updateTime": {S: awsv1.String(time.Now().Format(time.RFC3339))},
		},
		ConditionExpression: awsv1.String("attribute_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.IncrementUsageCount: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to increment usage count for bot %s: %w", botID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.IncrementUsageCount: Usage count incremented successfully")
	return nil
}

func (r *BotRepositoryV1) SearchBots(ctx context.Context, query string, scope string, userID string, userGroups []string) ([]models.Bot, error) {
	log.Printf("🔍 BotRepositoryV1.SearchBots: Searching bots with query: %s, scope: %s", query, scope)
	
	// Basic text search in title and description using scan with filter
	filterExpression := "contains(Title, :query) OR contains(Description, :query)"
	expressionAttributeValues := map[string]*dynamodbv1.AttributeValue{
		":query": {S: awsv1.String(query)},
	}
	
	// Add scope-based filtering
	switch scope {
	case "private":
		filterExpression += " AND OwnerID = :ownerID"
		expressionAttributeValues[":ownerID"] = &dynamodbv1.AttributeValue{S: awsv1.String(userID)}
	case "public":
		filterExpression += " AND SharedScope = :scope"
		expressionAttributeValues[":scope"] = &dynamodbv1.AttributeValue{S: awsv1.String("public")}
	case "shared":
		filterExpression += " AND (SharedScope = :publicScope OR (SharedScope = :partialScope AND contains(AllowedUsers, :userID)))"
		expressionAttributeValues[":publicScope"] = &dynamodbv1.AttributeValue{S: awsv1.String("public")}
		expressionAttributeValues[":partialScope"] = &dynamodbv1.AttributeValue{S: awsv1.String("partial")}
		expressionAttributeValues[":userID"] = &dynamodbv1.AttributeValue{S: awsv1.String(userID)}
	}
	
	result, err := r.client.Scan(&dynamodbv1.ScanInput{
		TableName: awsv1.String(r.tableName),
		FilterExpression: awsv1.String(filterExpression),
		ExpressionAttributeValues: expressionAttributeValues,
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.SearchBots: Scan failed: %v", err)
		return nil, fmt.Errorf("failed to search bots: %w", err)
	}
	
	var bots []models.Bot
	for _, item := range result.Items {
		bot, err := r.dynamoItemToBot(item)
		if err != nil {
			log.Printf("⚠️ BotRepositoryV1.SearchBots: Failed to convert item, skipping: %v", err)
			continue
		}
		
		// Additional access check
		if bot.IsAccessibleByUser(userID, userGroups, false) {
			bots = append(bots, *bot)
		}
	}
	
	log.Printf("✅ BotRepositoryV1.SearchBots: Found %d matching bots", len(bots))
	return bots, nil
}

func (r *BotRepositoryV1) GetBotsByScope(ctx context.Context, scope string, userID string, userGroups []string, limit int) ([]models.Bot, error) {
	log.Printf("🎯 BotRepositoryV1.GetBotsByScope: Getting bots by scope: %s (limit: %d)", scope, limit)
	
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
	
	// Apply limit if specified
	if limit > 0 && len(bots) > limit {
		bots = bots[:limit]
	}
	
	log.Printf("✅ BotRepositoryV1.GetBotsByScope: Retrieved %d bots for scope %s", len(bots), scope)
	return bots, nil
}

// Bot Aliases methods - implemented with proper schema mapping
func (r *BotRepositoryV1) CreateAlias(ctx context.Context, alias *models.BotAlias) error {
	log.Printf("🔗 BotRepositoryV1.CreateAlias: Creating alias for bot: %s", alias.OriginalBotID)
	
	// Set timestamps
	now := time.Now()
	if alias.CreateTime.IsZero() {
		alias.CreateTime = now
	}
	alias.LastUsedTime = now
	
	// Convert alias to DynamoDB item (aliases stored in same table with different ID pattern)
	item, err := r.aliasToDynamoItem(alias)
	if err != nil {
		return fmt.Errorf("failed to convert alias: %w", err)
	}
	
	_, err = r.client.PutItem(&dynamodbv1.PutItemInput{
		TableName: awsv1.String(r.tableName),
		Item: item,
		ConditionExpression: awsv1.String("attribute_not_exists(ID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.CreateAlias: PutItem failed: %v", err)
		return fmt.Errorf("failed to create alias: %w", err)
	}
	
	log.Printf("✅ BotRepositoryV1.CreateAlias: Alias created successfully")
	return nil
}

func (r *BotRepositoryV1) GetAliasByID(ctx context.Context, aliasID string) (*models.BotAlias, error) {
	log.Printf("🔍 BotRepositoryV1.GetAliasByID: Getting alias: %s", aliasID)
	
	result, err := r.client.GetItem(&dynamodbv1.GetItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(aliasID)}, // Schema uses uppercase ID
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetAliasByID: GetItem failed: %v", err)
		return nil, fmt.Errorf("failed to get alias %s: %w", aliasID, err)
	}
	
	if result.Item == nil {
		return nil, nil
	}
	
	// Check if this is actually an alias (has OriginalBotID field)
	if _, hasOriginalBot := result.Item["OriginalBotID"]; !hasOriginalBot {
		return nil, nil // This is a regular bot, not an alias
	}
	
	alias, err := r.dynamoItemToAlias(result.Item)
	if err != nil {
		return nil, fmt.Errorf("failed to convert alias: %w", err)
	}
	
	log.Printf("✅ BotRepositoryV1.GetAliasByID: Alias retrieved successfully")
	return alias, nil
}

func (r *BotRepositoryV1) GetAliasesByUser(ctx context.Context, userID string) ([]models.BotAlias, error) {
	log.Printf("📁 BotRepositoryV1.GetAliasesByUser: Getting aliases for user: %s", userID)
	
	// Scan for items with UserID = userID and OriginalBotID exists (aliases only)
	result, err := r.client.Scan(&dynamodbv1.ScanInput{
		TableName: awsv1.String(r.tableName),
		FilterExpression: awsv1.String("UserID = :userID AND attribute_exists(OriginalBotID)"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":userID": {S: awsv1.String(userID)},
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetAliasesByUser: Scan failed: %v", err)
		return nil, fmt.Errorf("failed to get aliases for user %s: %w", userID, err)
	}
	
	var aliases []models.BotAlias
	for _, item := range result.Items {
		alias, err := r.dynamoItemToAlias(item)
		if err != nil {
			log.Printf("⚠️ BotRepositoryV1.GetAliasesByUser: Failed to convert item, skipping: %v", err)
			continue
		}
		aliases = append(aliases, *alias)
	}
	
	log.Printf("✅ BotRepositoryV1.GetAliasesByUser: Retrieved %d aliases", len(aliases))
	return aliases, nil
}

func (r *BotRepositoryV1) GetAliasForUserBot(ctx context.Context, userID, originalBotID string) (*models.BotAlias, error) {
	log.Printf("🔎 BotRepositoryV1.GetAliasForUserBot: Getting alias for user: %s, bot: %s", userID, originalBotID)
	
	// Scan for alias with specific user and original bot
	result, err := r.client.Scan(&dynamodbv1.ScanInput{
		TableName: awsv1.String(r.tableName),
		FilterExpression: awsv1.String("UserID = :userID AND OriginalBotID = :originalBotID"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":userID": {S: awsv1.String(userID)},
			":originalBotID": {S: awsv1.String(originalBotID)},
		},
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.GetAliasForUserBot: Scan failed: %v", err)
		return nil, fmt.Errorf("failed to get alias: %w", err)
	}
	
	if len(result.Items) == 0 {
		return nil, nil
	}
	
	// Return first match (should be unique)
	alias, err := r.dynamoItemToAlias(result.Items[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert alias: %w", err)
	}
	
	log.Printf("✅ BotRepositoryV1.GetAliasForUserBot: Alias found")
	return alias, nil
}

func (r *BotRepositoryV1) UpdateAlias(ctx context.Context, aliasID string, updates map[string]interface{}) error {
	log.Printf("🔄 BotRepositoryV1.UpdateAlias: Updating alias: %s", aliasID)
	
	// Build update expression from provided updates
	updateExpression := "SET #updateTime = :updateTime"
	expressionAttributeNames := map[string]*string{
		"#updateTime": awsv1.String("UpdateTime"),
	}
	expressionAttributeValues := map[string]*dynamodbv1.AttributeValue{
		":updateTime": {S: awsv1.String(time.Now().Format(time.RFC3339))},
	}
	
	// Process updates dynamically
	for key, value := range updates {
		av, err := r.convertValueToV1AttributeValue(value)
		if err != nil {
			return fmt.Errorf("failed to convert update value for %s: %w", key, err)
		}
		if av != nil {
			updateExpression += fmt.Sprintf(", %s = :%s", key, key)
			expressionAttributeValues[":"+key] = av
		}
	}
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(aliasID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String(updateExpression),
		ExpressionAttributeNames: expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression: awsv1.String("attribute_exists(ID) AND attribute_exists(OriginalBotID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.UpdateAlias: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to update alias %s: %w", aliasID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.UpdateAlias: Alias updated successfully")
	return nil
}

func (r *BotRepositoryV1) DeleteAlias(ctx context.Context, aliasID string) error {
	log.Printf("🗑️ BotRepositoryV1.DeleteAlias: Deleting alias: %s", aliasID)
	
	_, err := r.client.DeleteItem(&dynamodbv1.DeleteItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(aliasID)}, // Schema uses uppercase ID
		},
		ConditionExpression: awsv1.String("attribute_exists(ID) AND attribute_exists(OriginalBotID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.DeleteAlias: DeleteItem failed: %v", err)
		return fmt.Errorf("failed to delete alias %s: %w", aliasID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.DeleteAlias: Alias deleted successfully")
	return nil
}

func (r *BotRepositoryV1) UpdateAliasLastUsedTime(ctx context.Context, aliasID string, lastUsedTime time.Time) error {
	log.Printf("⏰ BotRepositoryV1.UpdateAliasLastUsedTime: Updating last used time for alias: %s", aliasID)
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(aliasID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String("SET LastUsedTime = :lastUsed"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":lastUsed": {S: awsv1.String(lastUsedTime.Format(time.RFC3339))},
		},
		ConditionExpression: awsv1.String("attribute_exists(ID) AND attribute_exists(OriginalBotID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.UpdateAliasLastUsedTime: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to update alias last used time %s: %w", aliasID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.UpdateAliasLastUsedTime: Alias last used time updated successfully")
	return nil
}

func (r *BotRepositoryV1) UpdateAliasStarred(ctx context.Context, aliasID string, starred bool) error {
	log.Printf("⭐ BotRepositoryV1.UpdateAliasStarred: Updating starred status for alias: %s to %t", aliasID, starred)
	
	_, err := r.client.UpdateItem(&dynamodbv1.UpdateItemInput{
		TableName: awsv1.String(r.tableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"ID": {S: awsv1.String(aliasID)}, // Schema uses uppercase ID
		},
		UpdateExpression: awsv1.String("SET IsStarred = :starred"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":starred": {BOOL: awsv1.Bool(starred)},
		},
		ConditionExpression: awsv1.String("attribute_exists(ID) AND attribute_exists(OriginalBotID)"),
	})
	
	if err != nil {
		log.Printf("❌ BotRepositoryV1.UpdateAliasStarred: UpdateItem failed: %v", err)
		return fmt.Errorf("failed to update alias starred status %s: %w", aliasID, err)
	}
	
	log.Printf("✅ BotRepositoryV1.UpdateAliasStarred: Alias starred status updated successfully")
	return nil
}