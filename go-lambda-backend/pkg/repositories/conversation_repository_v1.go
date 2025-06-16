package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	// AWS SDK v1 only
	"github.com/aws/aws-sdk-go/aws/session"
	dynamodbv1 "github.com/aws/aws-sdk-go/service/dynamodb"
	s3v1 "github.com/aws/aws-sdk-go/service/s3"
	awsv1 "github.com/aws/aws-sdk-go/aws"
	"github.com/google/uuid"

	"github.com/bobbyrathore/go-lambda-backend/pkg/models"
)

// ConversationRepository provides methods to interact with conversation storage
type ConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *models.Conversation) error
	CreateConversationWithDetails(ctx context.Context, userID string, botID *string, title string, sessionModelID *string) (*models.Conversation, error)
	GetConversation(ctx context.Context, conversationID string, userID string) (*models.Conversation, error)
	GetConversationsByBotAndUser(ctx context.Context, botID string, userID string) ([]*models.Conversation, error)
	GetConversationMeta(ctx context.Context, conversationID string, userID string) (*models.ConversationMeta, error)
	ListConversations(ctx context.Context, userID string, limit int, nextToken string) ([]*models.ConversationMeta, string, error)
	UpdateConversation(ctx context.Context, conversation *models.Conversation) error
	DeleteConversation(ctx context.Context, conversationID string, userID string) error
	AddMessage(ctx context.Context, conversationID string, userID string, message *models.Message) error
	StoreConversationInS3(ctx context.Context, conversation *models.Conversation) error
	LoadConversationFromS3(ctx context.Context, conversation *models.Conversation) error
}

// ConversationRepositoryV1 implements ConversationRepository using AWS SDK v1 only
type ConversationRepositoryV1 struct {
	dynamoClient             *dynamodbv1.DynamoDB
	s3Client                 *s3v1.S3
	conversationsTableName   string
	messagesTableName        string
	s3Bucket                 string
	conversationSizeThreshold int // Size threshold in bytes for S3 storage
}

// NewConversationRepositoryV1 creates a new ConversationRepositoryV1 using AWS SDK v1 only
func NewConversationRepositoryV1(conversationsTableName, messagesTableName, s3Bucket string) (ConversationRepository, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}
	
	dynamoClient := dynamodbv1.New(sess)
	s3Client := s3v1.New(sess)
	
	// Test connectivity
	_, err = dynamoClient.ListTables(&dynamodbv1.ListTablesInput{})
	if err != nil {
		log.Printf("⚠️ AWS SDK v1 test failed: %v", err)
		return nil, fmt.Errorf("DynamoDB connectivity test failed: %w", err)
	}
	
	log.Printf("✅ AWS SDK v1 ConversationRepository initialized successfully")
	
	return &ConversationRepositoryV1{
		dynamoClient:              dynamoClient,
		s3Client:                  s3Client,
		conversationsTableName:    conversationsTableName,
		messagesTableName:         messagesTableName,
		s3Bucket:                  s3Bucket,
		conversationSizeThreshold: 300 * 1024, // 300KB like bedrock-chat
	}, nil
}

// CreateConversation creates a new conversation in DynamoDB with proper schema mapping
func (r *ConversationRepositoryV1) CreateConversation(ctx context.Context, conversation *models.Conversation) error {
	log.Printf("💬 ConversationRepositoryV1.CreateConversation: Creating conversation: %s", conversation.ID)
	
	// Initialize conversation if needed
	if conversation.ID == "" {
		conversation.ID = uuid.New().String()
	}
	if conversation.MessageMap == nil {
		conversation.MessageMap = make(map[string]*models.Message)
	}
	if conversation.CreatedAt.IsZero() {
		conversation.CreatedAt = time.Now()
	}
	conversation.UpdatedAt = time.Now()

	// Create system message as root
	systemMsg := &models.Message{
		ID:             "system",
		ConversationID: conversation.ID,
		Role:           "system",
		Content:        []models.MessageContent{},
		CreatedAt:      conversation.CreatedAt,
	}
	conversation.AddMessage(systemMsg)

	// Add instruction message if bot is specified
	if conversation.BotID != nil {
		instructionMsg := &models.Message{
			ID:             "instruction",
			ConversationID: conversation.ID,
			Role:           "instruction",
			Content:        []models.MessageContent{},
			ParentID:       awsv1.String("system"),
			CreatedAt:      conversation.CreatedAt,
		}
		conversation.AddMessage(instructionMsg)
	}

	return r.UpdateConversation(ctx, conversation)
}

// CreateConversationWithDetails creates a new conversation with user ID, bot ID, title, and optional session model
func (r *ConversationRepositoryV1) CreateConversationWithDetails(ctx context.Context, userID string, botID *string, title string, sessionModelID *string) (*models.Conversation, error) {
	log.Printf("💬 ConversationRepositoryV1.CreateConversationWithDetails: Creating conversation for user: %s, bot: %v, sessionModel: %v", userID, botID, sessionModelID)
	
	// Create conversation object
	conversation := &models.Conversation{
		ID:             uuid.New().String(),
		UserID:         userID,
		BotID:          botID,
		Title:          title,
		SessionModelID: sessionModelID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		MessageMap:     make(map[string]*models.Message),
	}
	
	// Use existing CreateConversation method to handle the creation
	if err := r.CreateConversation(ctx, conversation); err != nil {
		return nil, err
	}
	
	return conversation, nil
}

// GetConversation gets a conversation by ID with proper schema mapping
func (r *ConversationRepositoryV1) GetConversation(ctx context.Context, conversationID string, userID string) (*models.Conversation, error) {
	log.Printf("🔍 ConversationRepositoryV1.GetConversation: Getting conversation: %s for user: %s", conversationID, userID)
	
	// Use correct schema: UserID (partition), ID (sort)
	result, err := r.dynamoClient.GetItem(&dynamodbv1.GetItemInput{
		TableName: awsv1.String(r.conversationsTableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"UserID": {S: awsv1.String(userID)},   // Schema uses uppercase UserID
			"ID":     {S: awsv1.String(conversationID)}, // Schema uses uppercase ID
		},
	})

	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.GetConversation: GetItem failed: %v", err)
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	if result.Item == nil {
		log.Printf("ℹ️ ConversationRepositoryV1.GetConversation: Conversation not found")
		return nil, fmt.Errorf("conversation not found")
	}

	conversation, err := r.dynamoItemToConversation(result.Item)
	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.GetConversation: Failed to convert item: %v", err)
		return nil, fmt.Errorf("failed to convert conversation: %w", err)
	}

	// If conversation is stored in S3, load the full data
	if conversation.IsLargeConversation && conversation.S3Location != "" {
		if err := r.LoadConversationFromS3(ctx, conversation); err != nil {
			log.Printf("⚠️ Warning: failed to load conversation from S3: %v", err)
			// Continue with DynamoDB data (metadata only)
		}
	}

	log.Printf("✅ ConversationRepositoryV1.GetConversation: Conversation retrieved successfully")
	return conversation, nil
}

// GetConversationMeta gets lightweight conversation metadata
func (r *ConversationRepositoryV1) GetConversationMeta(ctx context.Context, conversationID string, userID string) (*models.ConversationMeta, error) {
	conv, err := r.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	meta := conv.ToMeta()
	return &meta, nil
}

// GetConversationsByBotAndUser gets all conversations for a specific bot and user
func (r *ConversationRepositoryV1) GetConversationsByBotAndUser(ctx context.Context, botID string, userID string) ([]*models.Conversation, error) {
	log.Printf("🤖 ConversationRepositoryV1.GetConversationsByBotAndUser: Getting conversations for bot: %s, user: %s", botID, userID)
	
	// Query conversations for user and filter by botID
	queryInput := &dynamodbv1.QueryInput{
		TableName:              awsv1.String(r.conversationsTableName),
		KeyConditionExpression: awsv1.String("UserID = :userID"),
		FilterExpression:       awsv1.String("BotID = :botID"),
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":userID": {S: awsv1.String(userID)},
			":botID":  {S: awsv1.String(botID)},
		},
		ScanIndexForward: awsv1.Bool(false), // Most recent first
	}

	result, err := r.dynamoClient.Query(queryInput)
	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.GetConversationsByBotAndUser: Query failed: %v", err)
		return nil, fmt.Errorf("failed to get conversations for bot %s: %w", botID, err)
	}

	conversations := make([]*models.Conversation, 0, len(result.Items))
	for _, item := range result.Items {
		conv, err := r.dynamoItemToConversation(item)
		if err != nil {
			log.Printf("Warning: failed to unmarshal conversation: %v", err)
			continue
		}
		conversations = append(conversations, conv)
	}

	log.Printf("✅ ConversationRepositoryV1.GetConversationsByBotAndUser: Found %d conversations", len(conversations))
	return conversations, nil
}

// ListConversations lists conversation metadata for a user using proper schema
func (r *ConversationRepositoryV1) ListConversations(ctx context.Context, userID string, limit int, nextToken string) ([]*models.ConversationMeta, string, error) {
	log.Printf("📋 ConversationRepositoryV1.ListConversations: Listing conversations for user: %s", userID)
	
	queryInput := &dynamodbv1.QueryInput{
		TableName:              awsv1.String(r.conversationsTableName),
		KeyConditionExpression: awsv1.String("UserID = :userID"), // Schema uses uppercase UserID
		ExpressionAttributeValues: map[string]*dynamodbv1.AttributeValue{
			":userID": {S: awsv1.String(userID)},
		},
		Limit:            awsv1.Int64(int64(limit)),
		ScanIndexForward: awsv1.Bool(false), // Most recent first
	}

	// Handle pagination (simplified)
	if nextToken != "" {
		// In production, decode nextToken properly
		log.Printf("ℹ️ Pagination token provided: %s", nextToken)
	}

	result, err := r.dynamoClient.Query(queryInput)
	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.ListConversations: Query failed: %v", err)
		return nil, "", fmt.Errorf("failed to list conversations: %w", err)
	}

	var metas []*models.ConversationMeta
	for _, item := range result.Items {
		conversation, err := r.dynamoItemToConversation(item)
		if err != nil {
			log.Printf("⚠️ ConversationRepositoryV1.ListConversations: Failed to convert item, skipping: %v", err)
			continue
		}
		meta := conversation.ToMeta()
		metas = append(metas, &meta)
	}

	var newNextToken string
	if result.LastEvaluatedKey != nil {
		newNextToken = "next" // Proper implementation would encode the key
	}

	log.Printf("✅ ConversationRepositoryV1.ListConversations: Retrieved %d conversations", len(metas))
	return metas, newNextToken, nil
}

// UpdateConversation updates or creates a conversation with S3 optimization and proper schema mapping
func (r *ConversationRepositoryV1) UpdateConversation(ctx context.Context, conversation *models.Conversation) error {
	log.Printf("📝 ConversationRepositoryV1.UpdateConversation: Updating conversation: %s", conversation.ID)
	conversation.UpdatedAt = time.Now()

	// Check if conversation should be stored in S3
	if conversation.ShouldStoreInS3() {
		if err := r.StoreConversationInS3(ctx, conversation); err != nil {
			return fmt.Errorf("failed to store conversation in S3: %w", err)
		}
		
		// Store metadata in DynamoDB
		metadataConv := *conversation
		metadataConv.MessageMap = nil // Clear message map for metadata storage
		metadataConv.IsLargeConversation = true
		
		return r.storeConversationMetadata(ctx, &metadataConv)
	} else {
		// Store entire conversation in DynamoDB
		conversation.IsLargeConversation = false
		conversation.S3Location = ""
		return r.storeConversationMetadata(ctx, conversation)
	}
}

// storeConversationMetadata stores conversation in DynamoDB with proper schema mapping
func (r *ConversationRepositoryV1) storeConversationMetadata(ctx context.Context, conversation *models.Conversation) error {
	// Convert conversation to DynamoDB item with proper field mapping
	item, err := r.conversationToDynamoItem(conversation)
	if err != nil {
		return fmt.Errorf("failed to convert conversation: %w", err)
	}

	_, err = r.dynamoClient.PutItem(&dynamodbv1.PutItemInput{
		TableName: awsv1.String(r.conversationsTableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.storeConversationMetadata: PutItem failed: %v", err)
		return fmt.Errorf("failed to store conversation: %w", err)
	}

	log.Printf("✅ ConversationRepositoryV1.storeConversationMetadata: Conversation stored successfully")
	return nil
}

// DeleteConversation deletes a conversation with proper schema mapping
func (r *ConversationRepositoryV1) DeleteConversation(ctx context.Context, conversationID string, userID string) error {
	log.Printf("🗑️ ConversationRepositoryV1.DeleteConversation: Deleting conversation: %s", conversationID)
	
	// First get the conversation to check S3 location
	conversation, err := r.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	// Delete from S3 if stored there
	if conversation.IsLargeConversation && conversation.S3Location != "" {
		_, err = r.s3Client.DeleteObject(&s3v1.DeleteObjectInput{
			Bucket: awsv1.String(r.s3Bucket),
			Key:    awsv1.String(conversation.S3Location),
		})
		if err != nil {
			log.Printf("⚠️ Warning: failed to delete conversation from S3: %v", err)
		}
	}

	// Delete from DynamoDB using correct schema
	_, err = r.dynamoClient.DeleteItem(&dynamodbv1.DeleteItemInput{
		TableName: awsv1.String(r.conversationsTableName),
		Key: map[string]*dynamodbv1.AttributeValue{
			"UserID": {S: awsv1.String(userID)},     // Schema uses uppercase UserID
			"ID":     {S: awsv1.String(conversationID)}, // Schema uses uppercase ID
		},
	})

	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.DeleteConversation: DeleteItem failed: %v", err)
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	log.Printf("✅ ConversationRepositoryV1.DeleteConversation: Conversation deleted successfully")
	return nil
}

// AddMessage adds a message to an existing conversation
func (r *ConversationRepositoryV1) AddMessage(ctx context.Context, conversationID string, userID string, message *models.Message) error {
	log.Printf("➕ ConversationRepositoryV1.AddMessage: Adding message to conversation: %s", conversationID)
	
	// Get the conversation
	conversation, err := r.GetConversation(ctx, conversationID, userID)
	if err != nil {
		return err
	}

	// Initialize message
	if message.ID == "" {
		message.ID = uuid.New().String()
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now()
	}
	message.ConversationID = conversationID

	// Add message to conversation
	conversation.AddMessage(message)

	// Update conversation
	return r.UpdateConversation(ctx, conversation)
}

// StoreConversationInS3 stores large conversations in S3
func (r *ConversationRepositoryV1) StoreConversationInS3(ctx context.Context, conversation *models.Conversation) error {
	log.Printf("☁️ ConversationRepositoryV1.StoreConversationInS3: Storing conversation in S3: %s", conversation.ID)
	
	// Create S3 key
	s3Key := fmt.Sprintf("%s/%s/message_map.json", conversation.UserID, conversation.ID)
	
	// Marshal conversation data
	data, err := json.Marshal(conversation)
	if err != nil {
		return fmt.Errorf("failed to marshal conversation for S3: %w", err)
	}

	// Upload to S3
	_, err = r.s3Client.PutObject(&s3v1.PutObjectInput{
		Bucket:      awsv1.String(r.s3Bucket),
		Key:         awsv1.String(s3Key),
		Body:        bytes.NewReader(data),
		ContentType: awsv1.String("application/json"),
	})
	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.StoreConversationInS3: PutObject failed: %v", err)
		return fmt.Errorf("failed to upload conversation to S3: %w", err)
	}

	conversation.S3Location = s3Key
	log.Printf("✅ ConversationRepositoryV1.StoreConversationInS3: Conversation stored in S3 successfully")
	return nil
}

// LoadConversationFromS3 loads large conversations from S3
func (r *ConversationRepositoryV1) LoadConversationFromS3(ctx context.Context, conversation *models.Conversation) error {
	log.Printf("📥 ConversationRepositoryV1.LoadConversationFromS3: Loading conversation from S3: %s", conversation.S3Location)
	
	if conversation.S3Location == "" {
		return fmt.Errorf("no S3 location specified")
	}

	// Download from S3
	result, err := r.s3Client.GetObject(&s3v1.GetObjectInput{
		Bucket: awsv1.String(r.s3Bucket),
		Key:    awsv1.String(conversation.S3Location),
	})
	if err != nil {
		log.Printf("❌ ConversationRepositoryV1.LoadConversationFromS3: GetObject failed: %v", err)
		return fmt.Errorf("failed to download conversation from S3: %w", err)
	}
	defer result.Body.Close()

	// Unmarshal data
	var fullConversation models.Conversation
	if err := json.NewDecoder(result.Body).Decode(&fullConversation); err != nil {
		return fmt.Errorf("failed to unmarshal conversation from S3: %w", err)
	}

	// Merge S3 data with metadata
	conversation.MessageMap = fullConversation.MessageMap
	log.Printf("✅ ConversationRepositoryV1.LoadConversationFromS3: Conversation loaded from S3 successfully")
	return nil
}

// Helper methods for conversation conversion with proper schema mapping
func (r *ConversationRepositoryV1) conversationToDynamoItem(conversation *models.Conversation) (map[string]*dynamodbv1.AttributeValue, error) {
	// Convert to JSON first for easy handling of complex types
	conversationJSON, err := json.Marshal(conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conversation: %w", err)
	}
	
	var conversationData map[string]interface{}
	if err := json.Unmarshal(conversationJSON, &conversationData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}
	
	// Convert to DynamoDB attributes
	item, err := r.convertMapToV1AttributeValues(conversationData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to attributes: %w", err)
	}
	
	// Apply field name mappings to match DynamoDB schema
	r.applyConversationSchemaMapping(item)
	
	return item, nil
}

// applyConversationSchemaMapping fixes field names to match DynamoDB schema
func (r *ConversationRepositoryV1) applyConversationSchemaMapping(item map[string]*dynamodbv1.AttributeValue) {
	// Fix primary key: id -> ID
	if val, exists := item["id"]; exists {
		item["ID"] = val
		delete(item, "id")
	}
	
	// Fix partition key: userId -> UserID  
	if val, exists := item["userId"]; exists {
		item["UserID"] = val
		delete(item, "userId")
	}
	
	// Fix GSI key: botId -> BotID
	if val, exists := item["botId"]; exists {
		item["BotID"] = val
		delete(item, "botId")
	}
	
	// Fix GSI sort key: updatedAt -> UpdatedAt
	if val, exists := item["updatedAt"]; exists {
		item["UpdatedAt"] = val
		delete(item, "updatedAt")
	}
	
	log.Printf("🔧 ConversationRepositoryV1.applyConversationSchemaMapping: Applied schema field mappings")
}

// dynamoItemToConversation converts DynamoDB item back to Conversation struct
func (r *ConversationRepositoryV1) dynamoItemToConversation(item map[string]*dynamodbv1.AttributeValue) (*models.Conversation, error) {
	// Reverse the schema mapping first
	r.reverseConversationSchemaMapping(item)
	
	// Convert DynamoDB item to generic map
	data, err := r.convertV1AttributeValuesToMap(item)
	if err != nil {
		return nil, fmt.Errorf("failed to convert attributes: %w", err)
	}
	
	// Convert to JSON and then to Conversation struct
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}
	
	var conversation models.Conversation
	if err := json.Unmarshal(jsonData, &conversation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation: %w", err)
	}
	
	return &conversation, nil
}

// reverseConversationSchemaMapping reverses field name mappings from DynamoDB schema to Go struct
func (r *ConversationRepositoryV1) reverseConversationSchemaMapping(item map[string]*dynamodbv1.AttributeValue) {
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
	
	// Reverse: BotID -> botId
	if val, exists := item["BotID"]; exists {
		item["botId"] = val
		delete(item, "BotID")
	}
	
	// Reverse: UpdatedAt -> updatedAt
	if val, exists := item["UpdatedAt"]; exists {
		item["updatedAt"] = val
		delete(item, "UpdatedAt")
	}
}

// Reuse attribute conversion helpers from BotRepositoryV1
func (r *ConversationRepositoryV1) convertMapToV1AttributeValues(data map[string]interface{}) (map[string]*dynamodbv1.AttributeValue, error) {
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

func (r *ConversationRepositoryV1) convertValueToV1AttributeValue(value interface{}) (*dynamodbv1.AttributeValue, error) {
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

func (r *ConversationRepositoryV1) convertV1AttributeValuesToMap(item map[string]*dynamodbv1.AttributeValue) (map[string]interface{}, error) {
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

func (r *ConversationRepositoryV1) convertV1AttributeValueToValue(av *dynamodbv1.AttributeValue) (interface{}, error) {
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